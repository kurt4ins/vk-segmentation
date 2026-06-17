-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    user_id    BIGSERIAL   PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE segments (
    id                  BIGSERIAL   PRIMARY KEY,
    slug                TEXT        NOT NULL UNIQUE,
    auto_assign_percent SMALLINT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ
);

CREATE TABLE user_segments (
    user_id    BIGINT      NOT NULL REFERENCES users (user_id),
    segment_id BIGINT      NOT NULL REFERENCES segments (id),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, segment_id)
);

-- Secondary index on the main read path: GET /users/{id}/segments.
CREATE INDEX idx_user_segments_user_id ON user_segments (user_id);

-- History stores the slug (not an FK) so the audit trail survives segment deletion.
CREATE TABLE segment_history (
    id         BIGSERIAL   PRIMARY KEY,
    user_id    BIGINT      NOT NULL,
    slug       TEXT        NOT NULL,
    operation  TEXT        NOT NULL CHECK (operation IN ('add', 'remove')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_segment_history_user_created ON segment_history (user_id, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS segment_history;
DROP TABLE IF EXISTS user_segments;
DROP TABLE IF EXISTS segments;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
