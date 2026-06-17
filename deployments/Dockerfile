FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server



FROM alpine:3.20

RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app

WORKDIR /app
COPY --from=build /out/server /app/server
RUN mkdir -p /app/reports && chown -R app:app /app

USER app

EXPOSE 8080
ENTRYPOINT ["/app/server"]
