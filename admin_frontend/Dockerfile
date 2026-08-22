# Сборка
FROM node:latest AS builder
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
# Переменные Vite вшиваются на этапе сборки, не читаются в рантайме контейнера.
ARG VITE_API_BASE_URL=http://localhost:8080
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
RUN npm run build

# Прод (docker-compose.yml): статику отдаёт основной caddy сам (см. корневой
# Caddyfile, handle "/"), не отдельный сервер — этот контейнер только
# копирует dist/ в общий volume и завершается (Exited(0) в docker ps — так и
# задумано, restart: "no" в compose). Alpine, а не сам builder-образ — node
# тянуть в рантайм незачем.
FROM alpine:latest AS dist-export
COPY --from=builder /app/dist /dist
CMD ["sh", "-c", "cp -r /dist/. /output/"]

# Debug (docker-compose.debug.yml): там нет основного caddy — фронт и бэк на
# разных портах, без домена/TLS — так что раздаём статику сами тем же Caddy,
# просто отдельным маленьким инстансом со своим Caddyfile.
FROM caddy:2-alpine AS standalone
COPY --from=builder /app/dist /srv
COPY Caddyfile /etc/caddy/Caddyfile
EXPOSE 80
