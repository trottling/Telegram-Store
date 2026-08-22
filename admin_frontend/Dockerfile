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

# Рантайм
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
# .template + templates/ — образ сам подставит ADMIN_FRONTEND_LOG_LEVEL через
# envsubst при старте (docker-entrypoint.d/20-envsubst-on-templates.sh) и
# запишет результат в conf.d/default.conf; обычный COPY в conf.d такого не
# делает, там файл отдаётся как есть.
COPY nginx.conf /etc/nginx/templates/default.conf.template
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
