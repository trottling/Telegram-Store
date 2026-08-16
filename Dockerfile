# Один Dockerfile, три target'а (bot / backend / migrate) — общий build stage,
# docker-compose выбирает бинарник через `build.target`.

# ---- builder: собирает все три бинарника ----
FROM golang:latest AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends git && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/bot ./cmd/bot
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/backend ./cmd/backend
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# ---- runtime-base: общий образ без прав root ----
FROM alpine:latest AS runtime-base

RUN apk add --no-cache ca-certificates
RUN adduser -D -u 1000 appuser
USER appuser
WORKDIR /app

# ---- bot: телеграм-бот ----
FROM runtime-base AS bot
COPY --from=builder /out/bot .
CMD ["./bot"]

# ---- backend: API админ-панели ----
FROM runtime-base AS backend
COPY --from=builder /out/backend .
CMD ["./backend"]

# ---- migrate: разовая миграция схемы + бутстрап root-admin ----
FROM runtime-base AS migrate
COPY --from=builder /out/migrate .
CMD ["./migrate"]
