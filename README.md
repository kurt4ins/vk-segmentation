# VK Segmentation

REST-сервис для управления сегментами пользователей: CRUD сегментов, атомарное
добавление/удаление членства с журналом аудита, материализованная процентная
раздача, членства с TTL и CSV-отчёты по истории.

## Архитектура

Поток запроса направлен строго внутрь: `handler → service → repository/postgres → PostgreSQL`.

```text
cmd/server            точка входа: config → repos → services → router → workers → graceful shutdown
internal/
  domain/             сущности + доменные ошибки (без импортов фреймворков/БД)
  service/            бизнес-логика, границы транзакций; интерфейсы репозиториев живут здесь
    mocks/            сгенерированные моки репозиториев (make mocks)
  repository/postgres/ SQL на pgx; tx пробрасывается через context (Transactor / querier)
  worker/             rollout_worker (асинхронная процентная раздача), ttl_cleaner
  transport/http/     chi-роутер, хендлеры, DTO, middleware
  pkg/errmap          доменные ошибки → HTTP-конверт (единый источник истины)
  pkg/logger          slog JSON-логгер
api/openapi.yaml      рукописная OpenAPI-спека, отдаётся на /swagger
migrations/           goose-миграции (встроены в бинарь)
test/integration/     testcontainers + реальный Postgres
```

Ключевые инварианты:

- Уникальность членства `(user_id, segment_id)` — первичный ключ.
- Любое изменение членства атомарно с соответствующими строками `segment_history`
  в **одной транзакции**.
- TTL применяется на **чтении** (`expires_at IS NULL OR expires_at > now()`),
  поэтому корректность не зависит от воркера очистки.
- Доменные ошибки транслируются в HTTP-коды в **одном месте** (`pkg/errmap`).

`user_id` — это **UUIDv7** (упорядочен по времени, генерируется приложением).
`segments.id` — `BIGSERIAL` и наружу никогда не отдаётся (сегменты адресуются по `slug`).

## Быстрый старт (Docker)

```bash
cp deployments/.env.example deployments/.env    # задайте POSTGRES_USER / POSTGRES_PASSWORD / POSTGRES_DB
make up                                         # docker compose: app + postgres, миграции прогоняются при старте
curl localhost:8080/healthz                     # {"status":"ok"}
```

Остановить (также удаляет тома): `make down`.

Swagger UI доступен по адресу <http://localhost:8080/swagger>, метрики Prometheus — на `/metrics`.

## Конфигурация (env)

| Переменная           | По умолчанию    | Назначение                                  |
| -------------------- | --------------- | ------------------------------------------- |
| `DB_DSN`             | — (обязательна) | строка подключения к PostgreSQL             |
| `HTTP_PORT`          | `8080`          | порт HTTP-сервера                           |
| `LOG_LEVEL`          | `info`          | `debug` / `info` / `warn` / `error`         |
| `TTL_CLEAN_INTERVAL` | `1m`            | как часто запускается очистка TTL           |
| `ROLLOUT_BATCH_SIZE` | `1000`          | размер батча для вставок процентной раздачи |
| `REPORTS_DIR`        | `./reports`     | куда пишутся CSV-отчёты                     |
| `RUN_MIGRATIONS`     | `true`          | прогонять goose-миграции при старте         |

В docker-compose `DB_DSN` собирается из `POSTGRES_USER`/`POSTGRES_PASSWORD`/`POSTGRES_DB`.

## API

Базовый путь: `/api/v1`. Единый формат ошибок: `{"error":{"code","message"}}`.

| Метод    | Путь                            | Описание                                                               |
| -------- | ------------------------------- | ---------------------------------------------------------------------- |
| `POST`   | `/segments`                     | Создать сегмент `{slug, auto_assign_percent?}` → `201` / `400` / `409` |
| `GET`    | `/segments`                     | Список сегментов                                                       |
| `DELETE` | `/segments/{slug}`              | Soft-delete + каскад → `204` / `404`                                   |
| `POST`   | `/users`                        | Зарегистрировать пользователя → `201`                                  |
| `POST`   | `/users/{id}/segments`          | Изменить сегменты `{add, remove, ttl?}` → `200` / `400` / `404`        |
| `GET`    | `/users/{id}/segments`          | Активные сегменты `[{slug, expires_at}]` (главная ручка чтения)        |
| `GET`    | `/users/{id}/history?from=&to=` | CSV-отчёт → `{"link": "..."}`                                          |

```bash
B=localhost:8080/api/v1
curl -XPOST $B/segments -d '{"slug":"MAIL_GPT"}'
curl -XPOST $B/segments -d '{"slug":"AB_TEST","auto_assign_percent":50}'
ID=$(curl -sXPOST $B/users | jq -r .id)
curl -XPOST $B/users/$ID/segments -d '{"add":["MAIL_GPT"],"ttl":"24h"}'
curl $B/users/$ID/segments
curl "$B/users/$ID/history?from=2026-01-01"
```

- **`ttl`** — строка длительности в формате Go (например, `"24h"`, `"30m"`); задаёт
  `expires_at` у добавляемых членств. Без неё членство бессрочно.
- Операции членства **идемпотентны**: повторное добавление существующего сегмента
  или удаление того, которого нет, — безопасный no-op (и не пишет историю). Один и
  тот же slug одновременно в `add` и `remove` отклоняется с `400`.
- `from`/`to` принимают RFC3339 (`2026-01-01T00:00:00Z`) или `YYYY-MM-DD`; оба необязательны.
- Ссылка из отчёта указывает на `/reports/<file>.csv`, который отдаётся статикой.

## Проектные решения

### Семантика создания пользователя

Создание пользователя **явное** — через `POST /users` (там же бросается «кубик»
раздачи для процентных сегментов). Операции членства с неизвестным `id` пользователя
возвращают `404`, а не создают пользователя автоматически. Это держит популяцию
пользователей чётко определённой для процентной раздачи.

### Процентная раздача (материализованная)

Создание сегмента с `auto_assign_percent = P` ставит задачу в in-process очередь;
фоновый воркер выбирает `round(N·P/100)` случайных пользователей, ещё не входящих в
сегмент, пакетно вставляет членства (и историю `add`), затем переводит статус
сегмента `pending → applied`. При регистрации каждый процентный сегмент бросает
«кубик» `random() < P/100`, чтобы новые пользователи тоже попадали под раздачу.

### Путь масштабирования через детерминированный хэш (не реализован)

Для гипермасштаба процентное членство можно вычислять на лету детерминированным
хэшем — `hash(user_id, slug) % 100 < P` — храня только ручные override'ы add/remove
вместо материализации десятков миллионов строк. Этот сервис поставляет
**материализованный** подход (он проще и буквально соответствует «случайному
распределению»); хэш-подход задокументирован здесь как путь масштабирования. Другие
задокументированные, но не реализованные пути: Redis-кэш для read-heavy
`GET /users/{id}/segments` и партиционирование `user_segments` по `user_id`.

## Разработка

```bash
make test              # unit-тесты (сервисы со сгенерированными моками)
make test-integration  # интеграционные тесты (testcontainers + реальный Postgres; нужен Docker)
make lint              # golangci-lint (v2)
make mocks             # перегенерировать моки (go generate; нужен mockgen)
make migrate DB_DSN=postgres://...   # применить миграции через goose CLI
make build             # собрать бинарь сервера
```

Миграции используют **goose** (встроены в бинарь и применяются при старте, когда
`RUN_MIGRATIONS=true`). Моки генерируются через
[`go.uber.org/mock`](https://github.com/uber-go/mock) (`mockgen`). CI
(`.github/workflows/ci.yml`) прогоняет lint + unit + integration + build.

`plan.md` (на русском) — design-документ и источник истины.
