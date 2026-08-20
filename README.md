# 🛍️ TG-Store

`Внимание: Платежные системы не тестировались`

## Покупатель

- 📂 **Каталог** — дерево категорий произвольной глубины, навигация на inline-клавиатурах
- 🛒 **Покупка** — количество → подтверждение → списание, сток товара уменьшается атомарно, без гонок
- 💰 **Пополнение баланса** — реальные платёжные системы (CrystalPay / ЮKassa / Tinkoff), подтверждение по вебхуку
- 🧾 **История** — покупки и пополнения, сгруппированные по транзакции, с пагинацией
- 🤝 **Реферальная программа** — своя ссылка-приглашение, процент с покупок рефералов начисляется автоматически

## Админ

- 🌐 Отдельная **веб-панель** (React + Ant Design), общается с бэкендом по REST
- 🔑 **Вход без пароля** — одноразовый код из `/admin` в боте обменивается на короткоживущую сессию (JWT + Redis)
- 🚫 Управление пользователями — бан/разбан, начисление баланса, выдача/снятие прав админа
- 📜 Аудит-лог всех административных действий — кто, что и когда поменял
- 📈 Логи и метрики всего стека в Grafana (`stats.$DOMAIN_NAME`) — вход тем же кодом из `/admin`, отдельного логина нет

---

## 🧱 Стек

| Часть                        | Технологии                                                                                            |
|------------------------------|-------------------------------------------------------------------------------------------------------|
| Бот                          | Go, [go-telegram/bot](https://github.com/go-telegram/bot)                                             |
| Admin API / вебхуки платежей | Go, [gin](https://github.com/gin-gonic/gin) (два независимых бинарника)                               |
| Хранение данных              | PostgreSQL через [GORM](https://gorm.io)                                                              |
| Кэш / состояние              | Redis (read-through кэш, состояние FSM, сессии админов)                                               |
| UI панели                    | React, Vite, TypeScript, [Ant Design](https://ant.design)                                             |
| Логи и метрики               | [Prometheus](https://prometheus.io) + [Loki](https://grafana.com/oss/loki/)/[Promtail](https://grafana.com/docs/loki/latest/send-data/promtail/) + [Grafana](https://grafana.com) |
| TLS / прод-деплой            | [Caddy](https://caddyserver.com) (авто-TLS), ежедневный `pg_dump` + [rclone](https://rclone.org) в S3 |

Архитектура — hexagonal/ports-and-adapters: четыре независимых Go-бинарника (`bot`, `admin_backend`, `payments_backend`, `migrate`) вокруг общих Postgres и Redis

## 🚀 Быстрый старт (Docker Compose)

Нужен Docker и токен телеграм-бота от [@BotFather](https://t.me/BotFather).

`docker-compose.yml` — прод-стек: наружу торчит только `caddy` (80/443, авто-TLS
через Let's Encrypt), остальные сервисы (включая Grafana) портов на хост не
публикуют. Нужен реальный домен с A-записями на `admin./api./pay./stats.`
поддоменами (`DOMAIN_NAME` в `.env`) — без них ACME не сможет выпустить
сертификаты. Для локального запуска без домена используйте
`docker compose -f docker-compose.debug.yml up --build` — там все порты
опубликованы напрямую (`admin_backend` на `:8080`, `admin_frontend` на `:3000` и
т.д.), `caddy`/`backup` и стек логов/метрик (Prometheus/Loki/Promtail/Grafana) в
этом файле нет. Порты БД и кеша так-же открыты наружу.

```bash
cp .env.example .env
# заполните TELEGRAM_BOT_TOKEN, TELEGRAM_ROOT_ADMIN_ID (свой Telegram user ID)
# и сгенерируйте настоящий ADMIN_JWT_SECRET: openssl rand -base64 32
# для прод-стека — ещё DOMAIN_NAME/ACME_EMAIL и https://-варианты
# ADMIN_PANEL_BACKEND_URL/FRONTEND_URL/CORS_ORIGIN (см. комментарии в .env.example)

docker compose up --build
```

Это поднимет по порядку: Postgres, Redis, одноразовый контейнер `migrate` (схема + бутстрап root-admin), затем `bot`, `admin_backend`, `payments_backend`, `admin_frontend`, `caddy` (TLS-терминатор), `backup` (ежедневный `pg_dump`, опционально в S3 — см. `backup/`) и стек наблюдаемости — `prometheus`/`loki`/`promtail`/`grafana` (конфиги — в `monitoring/`).

После запуска:

1. Откройте бота в Telegram, отправьте `/start`.
2. Отправьте `/admin` — бот пришлёт ссылку на панель и 6-значный код (действует 30 секунд).
3. Откройте `https://admin.$DOMAIN_NAME` (или `http://localhost:3000` при запуске через `docker-compose.debug.yml`), вставьте код.

Так входит любой админ, включая root — пароль настраивать не нужно. Тем же кодом открывается и `https://stats.$DOMAIN_NAME` (Grafana с логами и метриками всего стека) — своей формы входа у неё нет, она просто доверяет уже залогиненной сессии панели.

## 🛠️ Локальная разработка

### Go-сервисы

```bash
go build ./...                   # собрать всё
go run ./cmd/migrate             # один раз на чистой БД: схема + бутстрап root-admin
go run ./cmd/bot                 # телеграм-бот
go run ./cmd/admin_backend       # admin API (по умолчанию :8080)
go run ./cmd/payments_backend    # приём вебхуков платежей (по умолчанию :8081)
```

Все четыре читают конфигурацию из `.env` в корне репозитория (см. [Конфигурация](#-конфигурация)) — Postgres и Redis должны быть уже доступны (`docker compose up db redis` — самый простой способ поднять оба, не запуская Go-сервисы в контейнерах).

### Фронтенд

```bash
cd admin_frontend
npm install
npm run dev      # Vite dev-сервер на :3000
```

Задайте `VITE_API_BASE_URL` (например, в `admin_frontend/.env.local`), если admin API не на дефолтном `http://localhost:8080`.

## ⚙️ Конфигурация

Скопируйте [.env.example](.env.example) в `.env` и заполните. Обязательные, без значений по умолчанию:

| Переменная                                              | Назначение                                                                                                   |
|---------------------------------------------------------|--------------------------------------------------------------------------------------------------------------|
| `TELEGRAM_BOT_TOKEN`                                    | от [@BotFather](https://t.me/BotFather)                                                                      |
| `TELEGRAM_ROOT_ADMIN_ID`                                | ваш числовой Telegram user ID — станет `root_admin` при первом запуске `migrate`                             |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_NAME` | доступы к базе данных                                                                                        |
| `ADMIN_JWT_SECRET`                                      | подписывает сессионные токены админов — сгенерировать `openssl rand -base64 32`, плейсхолдер не использовать |

Всё остальное (`POSTGRES_HOST/PORT/SSLMODE`, `REDIS_*`, `ADMIN_PANEL_*`, `PAYMENTS_BACKEND_PORT/URL`, `BOT_METRICS_PORT`, `LOKI_RETENTION_DAYS`, `GRAFANA_ADMIN_USER/PASSWORD`, `LOG_LEVEL`) имеет разумные значения по умолчанию для локальной разработки и Docker — что делает каждая переменная и когда её нужно менять, смотрите в комментариях `.env.example` (например, `ADMIN_PANEL_FRONTEND_URL` в продакшене должен быть настоящим `https://`-доменом, так как Telegram отклоняет `localhost`-ссылки в инлайн-кнопках; `PAYMENTS_BACKEND_URL` — внешний адрес, на который платёжные провайдеры шлют вебхуки; `GRAFANA_ADMIN_USER/PASSWORD` — не основной вход в Grafana, а break-glass доступ изнутри docker-сети, публично вход туда идёт кодом из `/admin`, см. выше).

## 📁 Структура проекта

```
cmd/bot/                точка входа телеграм-бота
cmd/admin_backend/      точка входа admin API
cmd/payments_backend/   точка входа приёма вебхуков платежей
cmd/migrate/            разовая миграция схемы + бутстрап root-admin
bot/                    хендлеры, middleware, клавиатуры бота
admin_backend/          хендлеры, middleware, роутинг admin API
payments_backend/       хендлеры, роутинг вебхуков платёжных мерчантов
internal/               доменные интерфейсы + их реализации (hexagonal/ports-and-adapters)
admin_frontend/         React-панель админа (отдельный npm-проект)
backup/                 контейнер ежедневного pg_dump + выгрузки в S3 (rclone)
monitoring/             конфиги Prometheus/Loki/Promtail/Grafana (логи и метрики)
Caddyfile               конфиг TLS-терминатора (docker-compose.yml, сервис caddy)
```
