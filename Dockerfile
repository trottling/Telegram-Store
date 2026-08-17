# Один Dockerfile, четыре target'а (bot / admin_backend / payments_backend /
# migrate) — общий build stage, docker-compose выбирает бинарник через `build.target`.

# builder: собирает все три бинарника
FROM golang:latest AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends git && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/bot ./cmd/bot
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/admin_backend ./cmd/admin_backend
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/payments_backend ./cmd/payments_backend
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# runtime-base: общий образ без прав root
FROM alpine:latest AS runtime-base

RUN apk add --no-cache ca-certificates
RUN adduser -D -u 1000 appuser
USER appuser
WORKDIR /app

# bot: телеграм-бот
FROM runtime-base AS bot
COPY --from=builder /out/bot .
CMD ["./bot"]

# admin_backend: API админ-панели
FROM runtime-base AS admin_backend
COPY --from=builder /out/admin_backend .
CMD ["./admin_backend"]

# payments_backend: приём вебхуков платёжных мерчантов
FROM runtime-base AS payments_backend
COPY --from=builder /out/payments_backend .
CMD ["./payments_backend"]

# migrate: разовая миграция схемы + бутстрап root-admin
FROM runtime-base AS migrate
COPY --from=builder /out/migrate .
CMD ["./migrate"]
