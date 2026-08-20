# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Telegram shop bot written in Go, using [go-telegram/bot](https://github.com/go-telegram/bot) for the Telegram API, GORM (Postgres driver) for persistence, and Redis for both a read-through cache and per-chat FSM conversation state. Users browse a category tree of arbitrary depth, buy products (a multi-step flow: pick quantity → confirm → charge, decrementing pre-stocked `ProductItem` records), top up a balance (UI is done, provider is a stub), and view purchase history. Admin actions (ban/unban, balance, product/category CRUD, promote/demote) are implemented as domain services but have **no UI in the bot** — the bot's `/admin` command replies with a one-time login code for the web panel instead.

The actual admin UI is a separate web panel: a Go+gin JSON API (`admin_backend`, its own binary/container — `cmd/admin_backend`), plus a React+Ant Design frontend (`admin_frontend/`, its own container) that talks to that API cross-origin. Auth has no persistent credential at all: `/admin` issues a 30-second one-time code, the login page exchanges it for a JWT session token (also backed by a Redis-held revocation record), and a fresh code is issued every time — nothing admin-related is ever stored in Postgres.

Payment-provider webhooks are handled by a third, wholly separate Go+gin API — `payments_backend`/`cmd/payments_backend` — not by `admin_backend`. The split exists because the two surfaces have opposite trust models: `admin_backend`'s routes are all behind a logged-in admin session (except the one code-exchange endpoint), while `payments_backend`'s three routes are unauthenticated-by-design and must be reachable from the public internet by CrystalPay/YooKassa/Tinkoff's own servers. Keeping them as separate binaries means `payments_backend` never links `AdminAuthService`/the session store/CORS handling at all, and `admin_backend` never exposes a route without a session check.

The four Go binaries (`cmd/bot`, `cmd/admin_backend`, `cmd/payments_backend`, `cmd/migrate`) run as four independent, concurrently-started containers/processes sharing one Postgres and one Redis — see [Commands](#commands) and `cmd/migrate`'s own doc comment for why schema setup is its own one-shot step rather than folded into any long-running service.

## Commands

```bash
go build ./...                                    # build everything
go build -o bot ./cmd/bot                         # build the Telegram bot binary
go build -o admin_backend ./cmd/admin_backend      # build the admin API binary
go build -o payments_backend ./cmd/payments_backend # build the payment-webhooks binary
go build -o migrate ./cmd/migrate                 # build the one-shot migration binary
go run ./cmd/bot                                  # run the bot locally (needs .env populated, see below)
go run ./cmd/admin_backend                        # run the admin API locally
go run ./cmd/payments_backend                     # run the payment-webhooks API locally
go vet ./...                                       # static checks
go test ./...                                      # run tests (see the testing note below)
docker compose up --build   # run migrate + bot + admin_backend + payments_backend + Postgres + Redis + admin_frontend + caddy + backup together (reads .env)
```

There is no Makefile or linter config in the repo — use the `go` toolchain directly.

**Testing**: coverage is deliberately narrow, not aspirational. Tests exist only where a bug is invisible to
review and to manual clicking — concurrency, transaction rollback, and one library behaviour the whole design
leans on. Three files, and each one is a regression test for a specific incident: `internal/service/
replenishment_service_test.go` (a webhook credit that fails mid-way must roll the status back, or the retry
silently succeeds and the money is gone — hand-written fakes over the `domain/repository` interfaces, whose
fake transactor models Postgres rollback), `internal/cache/redis/cache_test.go` (`ConsumeFSMState` hands the
state to exactly one of N concurrent callers — needs a real Redis command surface, hence `miniredis`, the only
test-only dependency), `bot/utils/update_test.go` (`CallbackQuery.Message.Message` really is nil for an
inaccessible message — builds the `Update` from raw JSON on purpose, since that nil comes out of
`MaybeInaccessibleMessage.UnmarshalJSON` and a struct literal would not reproduce it), and
`admin_backend/middleware/ratelimit_test.go` (a spoofed `X-Forwarded-For` must not buy a fresh rate-limit
bucket — this one exists because the control is worthless if `SetTrustedProxies` is ever dropped, and that is
invisible in a diff), and `bot/middleware/track_test.go` (the handler ctx must survive cancellation of the
polling ctx, and the in-flight counter must be released even on panic — otherwise one panic would hold shutdown
for the whole drain timeout). Do not add breadth for its own sake; do add a test when a fix's correctness cannot be
seen by reading the diff. `-race` needs `CGO_ENABLED=1` and a C compiler, which this dev machine does not have.

Frontend (`admin_frontend/`, separate npm project, not part of the Go module):

```bash
cd admin_frontend && npm install
npm run dev      # Vite dev server on :3000, needs VITE_API_BASE_URL pointed at a running admin_backend (defaults to http://localhost:8080)
npm run build    # tsc -b + vite build -> dist/ — what admin_frontend/Dockerfile bundles into nginx
```

### Local configuration

Config is loaded from environment variables via [internal/config/config.go](internal/config/config.go), no config file. **`config.New()` validates rather than guesses**: unparseable numbers, out-of-range ports, a non-positive `TELEGRAM_ROOT_ADMIN_ID` (0 would bootstrap the root admin onto a non-existent user, leaving nobody able to grant rights), `ADMIN_PANEL_BACKEND_PORT` colliding with `PAYMENTS_BACKEND_PORT`, malformed URLs, CORS entries carrying a path or trailing slash (`middleware.CORS` compares the `Origin` header byte-for-byte, so those silently never match), and trusted-proxy entries that are neither an IP nor a CIDR all fail the process at startup instead of degrading quietly later. Copy `.env.example` to `.env` and fill in `TELEGRAM_BOT_TOKEN`, `TELEGRAM_ROOT_ADMIN_ID`, and `ADMIN_JWT_SECRET` (all required — `config.New()` errors without them; DB user/password/name are also required). Also present: `POSTGRES_HOST/PORT/USER/PASSWORD/NAME/SSLMODE`, `REDIS_ADDR/USERNAME/PASSWORD/DB` (`USERNAME` is only for an ACL user, Redis 6+ — e.g. a managed instance with multiple ACL users; empty means auth goes through `PASSWORD` alone, same as before this field existed), `ADMIN_PANEL_BACKEND_PORT/BACKEND_URL/FRONTEND_URL/CORS_ORIGIN` (`Port` is `admin_backend`'s own listen port; `BACKEND_URL` is read only by docker-compose as the `admin_frontend` build's `VITE_API_BASE_URL` value — Go's `AdminPanelConfig` no longer has a `URL` field, since nothing in Go read it after the payments split; `FrontendURL` is where the React panel itself is served, which is what the bot's `/admin` inline button links to; `CORSOrigin` is a comma-separated list of exact origins — normally just the frontend's own — the API accepts cross-origin requests from; defaults cover both `localhost` and `127.0.0.1` on port 3000, since browsers treat those as different origins), `ADMIN_PANEL_TRUSTED_PROXIES` (comma-separated CIDRs fed to gin's `SetTrustedProxies`, default `172.28.0.0/16` = the pinned `public-network` subnet. **This is a security control, not cosmetics**: gin's default is to trust `X-Forwarded-For` from anyone, which makes `c.ClientIP()` attacker-controlled and the per-IP login-code rate limit bypassable with one extra header. Keep it in sync with `public-network`'s subnet in docker-compose.yml), `PAYMENTS_BACKEND_PORT/URL` (`payments_backend`'s own listen port and its externally-reachable URL — validated at startup for structure, and `cmd/bot` logs a Warn if it is a loopback address, since a forgotten value otherwise means invoices are created, paid, and never confirmed, with nothing in the logs — `cmd/bot/providers.go` builds CrystalPay/Tinkoff webhook callback URLs from `URL`; YooKassa doesn't take a per-invoice callback URL, so it's unaffected).

Also present: `DOMAIN_NAME`/`ACME_EMAIL` (the `caddy` service's TLS config — one base domain, `admin.`/`api.`/`pay.`/`stats.` subdomains are built from it inside the `Caddyfile` itself rather than four separate domain variables; `ACME_EMAIL` is the Let's Encrypt account email). `DOMAIN_NAME` is the one exception to "compose/Caddy/backup only, no Go binary reads these" — `admin_backend` also reads it (`config.AdminPanelConfig.CookieDomain`) to set `Domain=` on the admin session cookie, so it reaches `stats.$DOMAIN_NAME` too — see the web admin panel Auth section below. Everything else here is still compose/Caddy/backup-only, not read by any Go binary: `BACKUP_RETENTION_DAYS`, `S3_REMOTE`/`S3_BUCKET`, `RCLONE_CONFIG_<REMOTE>_*` (the `backup` service's daily `pg_dump`, kept for `RETENTION_DAYS` days in the `backup_data` volume and, only if `S3_REMOTE`/`S3_BUCKET` are both set, also pushed to S3 via `rclone` — see `backup/backup.sh`); `LOKI_RETENTION_DAYS`, `GRAFANA_ADMIN_USER`/`PASSWORD`, `BOT_METRICS_PORT` (the observability stack — see below).

**`docker compose` needs a `.env` at the repo root** — read both by Compose's own `${VAR}` substitution (the `db` service's `environment:` block, the `redis` service's `command:` — `--user default off` plus `--user "${REDIS_USERNAME}" on ">${REDIS_PASSWORD}" allcommands allkeys allchannels`, so the bundled container is only reachable with that ACL user, not just a bare password; `config.New()` requires both non-empty and rejects the `.env.example` placeholder the same way it does `ADMIN_JWT_SECRET` — see `.env.example`, the `admin_backend`/`payments_backend`/`admin_frontend` services' `${ADMIN_PANEL_BACKEND_URL}` substitution, and the `caddy`/`backup` services' `env_file`) and, via each service's `env_file: .env`, by the containers themselves. `docker-compose.yml` wires Postgres (`postgres:18-alpine`), Redis (`redis:8-alpine`), a one-shot `migrate`, `bot`, `admin_backend`, `payments_backend`, `admin_frontend`, a TLS-terminating `caddy` (see below), a `backup` container, and the observability stack (`prometheus`, `loki`, `promtail`, `grafana` — see the paragraph after this one) together, on two networks: `backend-network` (internal-only, db+redis+bot+admin_backend+payments_backend+migrate+backup+prometheus+loki+promtail+grafana) and `public-network` (bot, for outbound Telegram API calls; `backup`, for outbound S3 calls; `caddy`, `admin_backend`, `payments_backend`, `admin_frontend`, and `grafana`, so they can reach each other by container name — a container whose *only* network is `internal: true` can't reach or be reached from outside that network, see docker-compose.yml's network comments). `public-network`'s subnet is **pinned** (`172.28.0.0/16`) rather than left to Docker's default pool, because `admin_backend` only trusts `X-Forwarded-For` coming from it — see `ADMIN_PANEL_TRUSTED_PROXIES` above; change one and you must change the other. `caddy` is the *only* service with published host ports (80/443) — `admin_backend`/`payments_backend`/`admin_frontend`/`grafana` are reached exclusively through it now (`Caddyfile` at the repo root, `admin.`/`api.`/`pay.`/`stats.` subdomains of `DOMAIN_NAME`, auto-TLS via Let's Encrypt). `stats.` is Grafana, and it's the odd one out: it goes through `forward_auth` to `admin_backend`'s `/api/auth/me` before `reverse_proxy grafana:3000` — see the web admin panel Auth section for how that reuses the same login as the panel itself, and see below for what Grafana is actually showing (Prometheus + Loki, provisioned automatically). Security response headers live in the `Caddyfile`, not in gin middleware — one place, and it also covers the static bundle nginx serves for the panel. The `common_headers` snippet (HSTS + `nosniff` + drop `Server`) applies to all four hosts; `admin.` and `stats.` additionally get document-level headers (`X-Frame-Options`, `Referrer-Policy`), `admin.` more so — it also gets `Permissions-Policy` and a CSP, `stats.` doesn't yet (see below). **Its CSP is fitted to the actual build, not copied from a template**: `style-src` must keep `'unsafe-inline'` because Ant Design injects styles at runtime, and `connect-src` must list `https://api.$DOMAIN_NAME` because the SPA calls the API on a different subdomain — tighten either one and the panel silently stops working, so re-check the browser console after any frontend build change. `api.`/`pay.` serve only JSON and therefore use `default-src 'none'`; `stats.` deliberately ships with **no CSP yet** — unlike `admin.`'s, which was tuned empirically against the actual React/Ant Design build, Grafana's inline-script/style needs haven't been verified live, so it only gets the document-level headers (`X-Frame-Options`, `Referrer-Policy`) until someone checks the browser console after a real deploy and adds one. `docker-compose.debug.yml` still publishes each service's port directly for local debugging and has no `caddy`/`backup` service (nor the observability stack — see below, deliberately not added there). Every service also gets a shared `x-logging` anchor (`json-file`, 10m/3 files) to keep container logs bounded. There's no admin credential to retrieve from logs anymore — every admin, including the root admin, logs in by sending `/admin` to the bot.

**Observability stack** (`prometheus`, `loki`, `promtail`, `grafana` in docker-compose.yml, configs under `monitoring/`): `prometheus` (`backend-network` only, not reverse-proxied — reachable only via Grafana's datasource or `docker exec`) scrapes itself and, once the bot exposes `/metrics` (`BOT_METRICS_PORT`, default 9100), the `bot` service too — `monitoring/prometheus/prometheus.yml` is static YAML, not templated by Compose's `${VAR}` substitution, so a changed `BOT_METRICS_PORT` has to be edited there by hand as well, the same "two things that must change together" trap as `ADMIN_PANEL_TRUSTED_PROXIES`/`public-network`'s subnet. `loki` (`backend-network` only) stores logs on the filesystem (`loki_data` volume), retention via `LOKI_RETENTION_DAYS` (default 14, `limits_config.retention_period` + `compactor` in `monitoring/loki/loki-config.yml`, substituted by Loki itself via `-config.expand-env=true`, not by Compose). `promtail` needs no app-side changes at all to ship every existing container's logs to Loki: it discovers them via `docker_sd_configs` against a read-only-mounted `/var/run/docker.sock` + `/var/lib/docker/containers` (where the `json-file` driver already writes NDJSON for every service, see `x-logging` above), relabeling `__meta_docker_container_name` into a `container` Loki label. `grafana` is the only one of the four with public access (`stats.$DOMAIN_NAME`, see above) and the only one on both networks; its two datasources (Prometheus, Loki) are auto-provisioned from `monitoring/grafana/provisioning/` — nothing to click through in the UI after a fresh deploy. `GF_SECURITY_ADMIN_USER`/`PASSWORD` (`GRAFANA_ADMIN_USER`/`GRAFANA_ADMIN_PASSWORD` in `.env`) are a break-glass account only, reachable by connecting to `grafana:3000` from inside the docker network directly — the public `stats.` host never shows Grafana's own login form, see the Auth section below for why.

## Architecture

**Ports-and-adapters (hexagonal)**: interfaces live under `internal/domain/`, concrete implementations live in parallel packages. When implementing a repository or service, put the interface in `internal/domain/...` (if not already defined) and the struct that satisfies it in the corresponding non-domain package — do not define new interfaces outside `internal/domain`. `bot/`, `admin_backend/`, and `payments_backend/` all depend only on `internal/domain/service` — never on `internal/domain/repository`, `internal/repository/postgres`, or a concrete `internal/service` implementation directly. `cmd/bot/`, `cmd/admin_backend/`, `cmd/payments_backend/`, and `cmd/migrate/` are the four composition roots: each one is the only package allowed to import `internal/repository/postgres` and `internal/service`, wiring concrete repos/services and handing only the domain interfaces down — each is split across main.go/providers.go/lifecycle.go (see below), not a single file, but still one `package main` per binary, so the rule is unchanged, just no longer file-granular. `payments_backend` deliberately wires a much narrower slice of repos/services than `admin_backend` — see its own paragraph below.

```
internal/domain/models/       GORM entities (User, Category, Product, ProductItem, Purchase, AdminLog) — the
                               entities' GORM struct tags living here is an accepted trade-off, but AutoMigrate()
                               itself lives in internal/repository/postgres, not here: driving a schema migration
                               against a specific database is an adapter concern, not a domain one
internal/domain/repository/   repository interfaces + Transactor (unit-of-work over *gorm.DB.Transaction)
internal/domain/service/      service interfaces (User/Product/Purchase/Category/Admin/AdminAuth/Stats/Settings/
                               Replenishment)
internal/domain/service/payment/  PaymentProvider interface + PaymentStatus enum
internal/domain/cache/        read-through cache ports, one interface per cached entity (UserCache/ProductCache/
                               CategoryCache in their own files) — no umbrella Cache interface; a consumer depends
                               on only the entity interface(s) it actually uses, composing more than one locally
                               (e.g. internal/service's unexported multiCache) only when it genuinely needs to
internal/domain/fsm/          Store interface — per-chat FSM conversation state (GetFSMState/SetFSMState/
                               ClearFSMState/ConsumeFSMState — spelled out, not bare Get/Set/Clear, since the same
                               Redis-backed struct also implements domain/cache's per-entity interfaces).
                               **ConsumeFSMState (GETDEL) is the one to use whenever the state gates something
                               that must happen at most once** — go-telegram/bot handles every update in its own
                               goroutine (ProcessUpdate: `go r(...)`, and WithNotAsyncHandlers is not set), so a
                               separate Get-then-Clear lets two fast taps on the same button both pass the step
                               check. BuyConfirmHandler charges money, so it consumes; BuyCancelHandler
                               deliberately still uses Get+Clear, since losing that race is the harmless
                               direction. A consumed state is gone even if its Step turns out to be wrong —
                               re-setting it would reintroduce exactly the race being closed
internal/domain/adminsession/ Store interface — web-panel one-time login codes + sessions + the login-code
                               brute-force counter (IncrExchangeAttempts), all Redis-backed, nothing in Postgres;
                               a distinct bounded concern from domain/cache and domain/fsm, just implemented by
                               the same Redis-backed struct
internal/domain/errors/       sentinel error values, mapped to user-facing text by bot/texts.UserFacingError, and
                               to HTTP status/JSON by admin_backend/errors.DomainErrorToResponse

internal/repository/postgres/ GORM-backed implementations, using the Generics API (gorm.G[T]) for CRUD — aggregate
                               queries (e.g. grouping purchases into batches, dashboard stats) use the classic
                               *gorm.DB chainable builder (Model/Select/Joins/Where/Group/Scan) instead, since
                               gorm.G[T] assumes one row = one T. There are exactly two .Raw(...).Scan(...) cases,
                               both because neither API above can express the SQL: the recursive CTE for category
                               tree visibility, and ProductRepo.ReserveItem — an `UPDATE ... WHERE id = (SELECT
                               ... FOR UPDATE SKIP LOCKED) RETURNING *` that claims one stock item and marks it
                               sold in a single statement. ReserveItem is the anti-overselling primitive of the
                               whole shop and must only ever be called inside Transactor.WithinTransaction:
                               outside one, the item is consumed even when the purchase then fails. migrate.go's AutoMigrate is cmd/migrate's single entry point: DDL,
                               two unexported one-time schema cleanups left over from earlier iterations
                               (backfillUserRoles, dropLegacyAdminTokenColumn — both no-ops once already run), then
                               bootstrapping the root admin (UserRepo.EnsureRootAdminExists) and the default
                               Settings row (SettingsRepo.EnsureExists) — everything cmd/migrate needs, in one call
internal/service/              implementations of internal/domain/service — cache-aside on every read (check
                               cache, miss -> repo, populate cache), explicit invalidation on every write; admin
                               listing methods (*Admin/*All suffix) deliberately skip the cache and always read
                               through to Postgres, since they're tuned for freshness over throughput, not the
                               customer-facing catalog. The same "read straight through" rule covers anything the
                               cache must not be allowed to stale-serve: UserSrv.IsBanned (a ban has to bite on
                               the next update, not up to userTTL later) and AdminAuthSrv.ValidateSession (a
                               demoted admin loses access on their next request). Cache-aside is for display —
                               balances shown in the profile, catalog listings — never for a decision about
                               money or rights
internal/service/payment/     PaymentProvider implementations: StubProvider (always errors, unused now that real
                               providers exist), CrystalPayProvider (hand-rolled HTTP client, no official Go SDK),
                               YooKassaProvider (github.com/rvinnie/yookassa-sdk-go), TinkoffProvider
                               (github.com/nikita-vanyasin/tinkoff, PayTypeOneStep). All three read their own
                               Settings sub-struct (credentials + Enabled + Min/MaxAmount) fresh on every call via
                               SettingsService — admin edits apply without restarting the bot — and each does its
                               own enabled/range check before calling out, returning ErrMerchantDisabled/
                               ErrAmountOutOfRange. Confirmation is webhook-driven, not polled: nothing runs
                               CheckStatus on a timer. It IS called for CrystalPay, synchronously from that
                               merchant's webhook, because CrystalPay's signature covers only the invoice id and
                               leaves `state` unsigned — see payments_backend/handlers below. Tinkoff's and
                               YooKassa's CheckStatus have no caller today (their webhooks establish the status
                               another way) and exist for interface completeness
internal/cache/redis/         the ONE Redis-backed struct implementing domain/cache's per-entity interfaces,
                               domain/fsm.Store, and domain/adminsession.Store — one client, one struct, three
                               unrelated keyspaces: plain keys for the cache, fsm:<telegramID> for conversation
                               state, admin_login_code:<hash>/admin_session:<hash> for the web panel's login flow

bot/                           Telegram-facing layer (package `bot`, binary entrypoint is cmd/bot)
bot/handlers/                  start.go (/start creates the user row, parses the ref-link payload — see the
                               referral paragraph above — main-menu/help; help pulls SupportUsername from
                               Settings), admin.go (/admin — issues the login code), profile.go, catalog.go
                               (category-tree rendering + product cards, edits messages in place), buy.go
                               (quantity -> confirm -> charge; BuyConfirmHandler is also what messages the
                               referrer after PurchaseSrv.Buy returns a non-nil credit), purchases.go (history,
                               paginated), refill.go (merchant picker -> amount prompt with min/max hint ->
                               ReplenishmentService.CreateInvoice -> pay-link message; enabled merchants come from
                               Settings, read fresh on every RefillBalanceHandler call), replenishments.go ("Мои
                               пополнения" — paginated like purchases.go, but one plain-text line per row instead
                               of a button per row, since there's no per-item content to drill into on tap),
                               referral.go (ReferralHandler: link + live invited-count/total-credited stats,
                               bails out to texts.ReferralUnavailableMsg if Settings.Referral.Enabled is false;
                               ReferralCloseHandler just deletes the message)
bot/middleware/                track.go (Track: registers each in-flight update in a WaitGroup and hands the
                               handler a context.WithoutCancel copy of the polling ctx. Both halves exist for
                               shutdown: Bot.Start's WaitGroup covers only the polling loop, never the per-update
                               goroutines, and those goroutines share the polling ctx — so cancelling it to stop
                               polling used to tear live handlers in half. WaitInFlight, surfaced as
                               TelegramBot.WaitInFlight, is what cmd/bot/lifecycle.go waits on between stopping
                               the poller and closing Redis; outermost in the chain so the detached ctx reaches
                               everything, including BanCheck's lookups), recover.go (Recover: go-telegram/bot
                               runs every update in its own goroutine and has
                               no recover() anywhere, so an unhandled panic kills the whole process — this is the
                               bot's equivalent of gin.Recovery(); registered right after Track because applyMiddlewares wraps
                               in reverse, making m[0] outermost, and the panic sites it has to cover include
                               Logging itself), middleware.go (Logging: logs every update at Debug, first after
                               Recover so nothing skips it), answer_callback.go (acks every callback_query first,
                               or the tapped button stays "loading" until Telegram times it out), ban_check.go (the
                               only per-update user-exists/banned gate — see below), fsm.go (multi-step flows: buy
                               quantity+confirm, refill amount) — registered in that order: Track -> Recover ->
                               Logging -> AnswerCallback -> BanCheck -> FSM. Refill's merchant pick (handlers/refill.go's
                               RefillMerchantHandler, a plain inline callback) happens before FSM state exists;
                               only the subsequent amount prompt is FSM-tracked (State.Merchant carries the pick
                               through to fsm.go's handleRefillAmount)
bot/keyboards/                 Keyboards struct (static reply/inline keyboards, built once via New(adminPanelURL)
                               — not package-level vars, since AdminKb needs the URL) plus per-request builder funcs
bot/texts/                     all user-facing button/message strings — add new UI copy here, not inline in handlers
bot/utils/                     callback.go (build/parse callback_data — numeric "prefix_<id>" and, for purchase
                               batches, "purchase_<uuid>"; refillmerchant_<merchant> carries a bare string, not a
                               number, so it gets its own Build/Parse pair instead of reusing ParseCallbackQuery),
                               update.go (CallbackChatID/CallbackTarget — the ONLY sanctioned way to read chat/
                               message out of a callback_query: CallbackQuery.Message is a MaybeInaccessibleMessage
                               whose .Message is nil for a message the bot can no longer access, so a direct
                               .Message.Message.Chat.ID deref panics and takes the process down; CallbackTarget
                               additionally refuses an inaccessible message, since it can't be edited), markdown.go
                               (MarkdownV2 escaping + amount/date formatting), stock.go (traffic-light emoji for
                               remaining stock). Merchant/status display text and domain-error text live in
                               bot/texts, not here.

internal/auth/web/            package admintoken (import resolves by package clause, not directory name) —
                               crypto/rand generation for the web panel's one-time login codes (6-digit,
                               GenerateCode), plus HS256 JWT session tokens (GenerateSessionJWT/ParseSessionJWT)
                               — a leaf utility package (no domain imports) used only by
                               internal/service/admin_auth_service.go. Everything here is keyed by
                               ADMIN_JWT_SECRET, including Hash: it is HMAC-SHA256, not bare sha256, because the
                               login code is only 6 digits and a rainbow table over a million values is
                               instant — an unkeyed hash would mean read access to Redis yields a working login
                               code. Consequence of changing the key: every stored hash becomes unmatchable, so
                               rotating ADMIN_JWT_SECRET logs all admins out (they just send /admin again). The JWT is
                               tamper-evident and self-describes its own expiry, but is not by itself sufficient
                               to trust a session — see AdminAuthSrv.ValidateSession below for why

admin_backend/                 the web admin panel's HTTP API (package `adminbackend`, directory `admin_backend/`,
                               binary entrypoint is cmd/admin_backend) — gin, depends only on
                               internal/domain/service, same rule as bot/. adminbackend.New(...) wires everything;
                               cmd/admin_backend/main.go is the composition root. Payment-provider webhooks are
                               NOT here — see payments_backend/ below, a wholly separate binary
admin_backend/handlers/        one file per resource; read-only admin listings (all products/categories/purchases/
                               admin-logs, unfiltered by active/stock/other-users'-rows) call dedicated *Admin/*All
                               service methods (UserService.ListAdmin, ProductService.ListAllAdmin,
                               CategoryService.ListAllFlat, PurchaseService.ListAllAdmin, AdminService.ListLogs) —
                               no business logic to wrap. Every write (create/update/delete, ban, balance,
                               promote/demote) goes through AdminService/UserService too, so AdminLog auditing and
                               cache invalidation are never bypassed. GET /api/admin-logs (optional ?admin_id=
                               filter) is the only way to see that audit trail. settings_handler.go's PUT rebuilds
                               a whole models.Settings from the request and always goes through AdminService.
                               UpdateSettings (audit-logged), never SettingsService directly. replenishment_handler.go
                               is read-only (ListAllAdmin, ?user_id filter) — writes to Replenishment only ever
                               happen via payments_backend/handlers' webhook handlers (CrystalPay/YooKassa/Tinkoff,
                               a different binary), each verifying that merchant's own signature scheme before
                               calling ReplenishmentService.Confirm/Fail; none of them go through AdminService (no
                               admin acted, nothing to audit)
admin_backend/middleware/      auth.go (bearer session token -> AdminAuthService.ValidateSession, attaches
                               *models.User to request context), cors.go (comma-separated allowed-origin list via
                               ADMIN_PANEL_CORS_ORIGIN, no credentials — logs rejected origins at Warn for
                               diagnosis), ratelimit.go (RateLimitExchange — per-route, only on
                               /api/auth/exchange; keyed on c.ClientIP(), which is only trustworthy because
                               router.go calls SetTrustedProxies — see ADMIN_PANEL_TRUSTED_PROXIES above.
                               ratelimit_test.go covers the real attack shape: a client-supplied
                               X-Forwarded-For with Caddy's appended real IP to its right must still be counted
                               against the real IP)
admin_backend/dto/             request bodies + the Paginated[T] list envelope + ErrorResponse; responses mostly
                               reuse internal/domain/models types directly (already have clean json tags).
                               Request structs carry `binding:` tags that **mirror, and must not diverge from,
                               the AdminSrv checks** — they buy an early 400 instead of a domain error after a
                               pointless DB round-trip, they do not replace the service validation. Two traps
                               dto_test.go pins down: `required` on a number means "not zero" (which is why
                               UpdateBalanceRequest.Amount uses it and IsActive does NOT — for a bool, false is
                               indistinguishable from absent, and tagging it would make deactivating a product
                               impossible), and a wrong validator name is caught by neither the compiler nor
                               go vet, so it would surface as valid requests being rejected
admin_backend/errors/          domain sentinel error -> HTTP status + JSON body (DomainErrorToResponse), mirrors
                               bot/texts.UserFacingError
admin_backend/router.go,       route table + http.Server lifecycle (Start/Shutdown). /api/auth/exchange is the
  server.go                    only route outside the /api Auth group (no session exists yet to check) — every
                               other route requires a valid admin session

payments_backend/              accepts payment-provider webhooks only (package `paymentsbackend`, directory
                               `payments_backend/`, binary entrypoint is cmd/payments_backend) — gin, depends only
                               on internal/domain/service, same rule as bot/ and admin_backend/. Deliberately tiny:
                               paymentsbackend.New(...) takes just SettingsService + ReplenishmentService + the
                               CrystalPay PaymentProvider, nothing admin-related (no AdminAuthService, no session
                               store, no CORS middleware — the caller is a merchant's server, never a browser)
payments_backend/handlers/     handler.go (the Handlers struct/constructor — two services plus the CrystalPay
                               PaymentProvider) + webhook_handler.go (CrystalPayWebhook/YooKassaWebhook/
                               TinkoffWebhook; each verifies that merchant's own signature scheme before calling
                               ReplenishmentService.Confirm/Fail — see the Data model section below). **Signature
                               valid does not mean status trustworthy**, and the three merchants differ: Tinkoff
                               signs the whole body, so notification.Status can be used directly; YooKassa signs
                               nothing, so the payload is a bare trigger and the status comes from an authorized
                               FindPayment; CrystalPay signs only the invoice id, so its unsigned `state` field is
                               ignored too and the status comes from CrystalPayProvider.CheckStatus. Only
                               CrystalPay's provider is wired here — the other two need no re-fetch through the
                               domain interface
payments_backend/router.go,    three POST routes (/api/webhooks/{crystalpay,yookassa,tinkoff}), gin.Recovery() +
  server.go                    middleware.Detach, no Auth group, no CORS — http.Server lifecycle (Start/Shutdown)
payments_backend/middleware/   detach.go (Detach) — the only middleware here, and it exists for a correctness
                               reason, not tidiness: gin cancels the request ctx the moment the client
                               disconnects, and payment merchants cut the connection on their own few-second
                               timeout. That cancellation used to land between Confirm's commit and its cache
                               invalidation. Detach swaps in context.WithoutCancel + a 30s deadline of its own.
                               Consequence to know: an in-flight webhook can now outlive the 10s graceful-shutdown
                               grace in cmd/payments_backend/lifecycle.go and be killed by process exit instead —
                               which is safe precisely because Confirm is transactional (the tx rolls back, the
                               row stays pending, the merchant's retry still credits)

internal/config/               env-var config loader (Telegram/Postgres/Redis/AdminPanel/Payments/Logger sub-configs)
internal/logger/               logrus logger construction from config.LoggerConfig, plus fx.go's NewFxLogger
                               (routes fx's own PROVIDE/INVOKE/START/STOP event log through the same logrus
                               logger at Debug — quiet in prod, visible with LOG_LEVEL=debug); the only
                               importers of this package are the four cmd/* binaries.
                               **Everything logs through this one logrus instance, and nothing should print
                               ANSI**: these binaries always run with stdout as a pipe (docker json-file), so
                               escape codes end up inside the stored log field and break both `docker logs` and
                               any aggregator. That is why TextFormatter does NOT set ForceColors (logrus then
                               colors only a real TTY), and why postgres.NewClient hands GORM its own
                               gormlogger with Colorful:false whose Writer forwards into logrus — GORM otherwise
                               writes coloured lines straight to stderr via the standard log package, ignoring
                               LOG_LEVEL entirely

Each cmd/* binary wires its dependency graph with go.uber.org/fx, split into main.go (the fx.New(...) call —
the actual list of what's wired, read this first) and lifecycle.go (fx.Lifecycle OnStart/OnStop for the
long-running ones). Every repo/service constructor (pgdb.NewX, service.NewX) returns its own concrete type
and stays completely unaware that fx exists — fx itself needs no annotation on a constructor to use it, any
plain Go function works as an fx.Provide entry as-is. The only gap fx has to bridge is concrete-type-vs-
interface (constructors return e.g. *pgdb.UserRepo, consumers want repository.UserRepository): main.go closes
that with fx.Annotate(ctor, fx.As(new(Iface))) directly in the fx.Provide(...) list — fx's own purpose-built
tool for exactly this, so there's no separate provideUserRepo-style wrapper function per repo/service to
maintain. providers.go is deliberately small: only functions with actual logic fx.Annotate can't express —
*config.Config sub-struct extractors (fx resolves by exact type, so each field needs its own provider),
providePaymentProviders (assembles a map from three constructors, not a type cast), and
provideAdminAuthService/provideReplenishmentService (pull a field out of a config struct, or hand-supply a
literal nil — real argument-shaping, not interface-casting). Repos/services needed by more than one binary are
NOT factored into a shared package — each binary's main.go re-declares its own fx.Annotate entries, same
duplication the manual wiring already had; the point was translating the wiring mechanism, not merging separate
composition roots into one — this is also why payments_backend's graph doesn't just reuse admin_backend's: it
deliberately re-declares only the handful of fx.Annotate entries it needs (see its own paragraph below), rather
than importing a shared "all repos/services" provider set that would drag in everything admin_backend wires.
The one Redis-backed cache struct is wired once per binary via fx.Annotate(rdb.NewRedisCache, fx.As(new(X)),
fx.As(new(Y)), ...) — multiple fx.As() on one constructor registers that single instance under every interface
it implements, but each binary annotates only the subset it actually consumes: bot annotates
domaincache.UserCache/ProductCache/CategoryCache/SettingsCache/domain/fsm.Store/adminsession.Store/
service.MultiCache; admin_backend annotates the same set minus domain/fsm.Store (no FSM scenarios there);
payments_backend annotates only domaincache.UserCache and domaincache.SettingsCache — no adminsession.Store, no
MultiCache, no Product/CategoryCache (service.MultiCache is exported specifically so cmd/*/main.go can name it
in fx.As — see internal/service/cache.go). bot.New/adminbackend.New/paymentsbackend.New themselves also take no
fx.Lifecycle param and know nothing about fx — lifecycle.go's runBot/runServer register the OnStart/OnStop hooks
from outside, in cmd/*, the same "fx stays out of non-composition-root packages" principle applied one level up
(an alternative, equally common fx idiom would have the constructor itself accept fx.Lifecycle and self-register
— deliberately not used here, since that would mean bot/, admin_backend/, and payments_backend/ importing fx
just to be constructed, which is exactly the coupling being avoided).
cmd/migrate/main.go            one-shot, no fx.Lifecycle/Run(): fx.New(...) both builds the graph AND runs
                               fx.Invoke(runMigrate) synchronously (Invoke functions execute during fx.New
                               itself, before any Start) — runMigrate's single pgdb.AutoMigrate(ctx, db, log,
                               rootAdminID) call does the DDL, legacy-column cleanups, and root-admin/Settings
                               bootstrap (see internal/repository/postgres above). app.Err() (build error or
                               runMigrate's returned error) is also printed straight to stderr before os.Exit(1)
                               — LOG_LEVEL shouldn't get to hide a fatal migration failure, and if the graph
                               failed to build at all there may be no working logger yet to report through.
                               Its own doc comment explains why this is a separate binary/container from
                               cmd/bot, cmd/admin_backend, and cmd/payments_backend: independent, concurrently-
                               started long-running services can't both own "run AutoMigrate on startup" without racing
cmd/*/main.go (bot,          every long-running binary checks app.Err() before Run() and prints it to stderr,
  admin_backend,             mirroring cmd/migrate: fx.New executes Invoke functions itself, so a graph or
  payments_backend)          constructor failure (unreachable Postgres, bad config) lands there — while Run()
                             alone would only exit 1 and route the reason to the fx logger at Debug, leaving a
                             container crash-looping with an empty log at the default LOG_LEVEL
cmd/bot/main.go                fx.New(...).Run() — Run() itself blocks and listens for SIGINT/SIGTERM (no
                               manual signal.NotifyContext anymore). lifecycle.go's runBot launches
                               bt.Start(ctx) in a goroutine from OnStart (Start blocks on long-polling, so it
                               can't run inline — OnStart hooks are expected to return quickly). OnStop is
                               ordered on purpose and the order is load-bearing: cancel the polling ctx, then
                               bt.WaitInFlight (bounded by drainTimeout) for updates already being handled, and
                               only then close the redis client. Closing Redis first meant a purchase caught by
                               SIGTERM committed but failed to invalidate the balance cache
cmd/admin_backend/main.go      fx.New(...).Run(), same signal handling as cmd/bot. lifecycle.go's runServer
                               launches webServer.Start() in a goroutine from OnStart (it blocks until
                               Shutdown) and calls webServer.Shutdown(ctx) (10s timeout) then closes the redis
                               client from OnStop. Wires all repos/services admin_backend/handlers needs: eight
                               repos (User/Product/Purchase/Category/Settings/Replenishment/AdminLog/Stats) plus
                               GormTransactor, seven services (User/Product/Category/Settings/Purchase/Admin/Stats
                               via fx.Annotate) plus provideReplenishmentService/provideAdminAuthService
cmd/payments_backend/main.go   Same fx.New(...).Run()/runServer shape as cmd/admin_backend, but a much narrower
                               graph: three repos (User/Settings/Replenishment) plus GormTransactor — the
                               transactor is here purely for ReplenishmentSrv.Confirm, which has to commit the
                               status flip and the balance credit together — one service via fx.Annotate
                               (SettingsSrv) plus provideReplenishmentService — no
                               UserService/AdminAuthService/adminsession.Store/Product/Category/AdminLog/Stats
                               anything. ReplenishmentSrv itself takes a raw repository.UserRepository, not
                               UserService, which is why payments_backend never needs to wire UserSrv at all

admin_frontend/                the web admin panel's UI — React + Vite + Ant Design, its own package.json/Dockerfile,
                               deployed as a separate container (nginx serving the built static bundle) that calls
                               admin_backend's API cross-origin (VITE_API_BASE_URL, baked in at image build time, not
                               read at runtime). Not part of the Go module. Every protected page is React.lazy()-loaded
                               (see App.tsx) — StatsPage alone pulls in @ant-design/plots (~1.4MB), which shouldn't
                               block loading the login screen or any other page
```

All repositories and services are fully implemented (not stubs) and logging (`*logrus.Logger`, threaded in from each `main.go`) is wired through every repo/cache/service constructor.

### Data model

`User` is keyed by `TelegramID` directly — there's no separate internal auto-increment ID; every FK that points at a user (`Purchase.UserID`, `AdminLog.AdminID`/`TargetID`) stores the Telegram ID. `User.Role` is a single, mutually exclusive privilege level (`banned`/`user`/`admin`/`root_admin`, `models.Role`) — there's exactly one `root_admin` at a time (the `TELEGRAM_ROOT_ADMIN_ID` from config, bootstrapped into this column by `cmd/migrate`'s `UserRepository.EnsureRootAdminExists`). `User.IsBanned()`/`IsAdmin()`/`IsRootAdmin()` are the derived-boolean helper methods everything else checks against, not the raw `Role` field. Because `Role` is one field, banning an Admin/RootAdmin overwrites their admin rights rather than sitting next to them, and un-banning always restores plain `user`, never whatever role they held before (see `AdminSrv.BanUser`/`UnbanUser`).

`User.ReferrerID *int64` records who invited this user (nil if none) — set exactly once, at row creation, and never touched again (see `UserSrv.GetOrCreate` and the referral paragraph below). `User.ReferralsEnabled` (`default:true`, relies on GORM substituting the tag's default for an omitted/zero-value bool field on `Create` — the same mechanism `Role`'s `default:'user'` already depends on) gates whether *this user, as a referrer*, still earns credit; it's flipped by `AdminSrv.SetReferralsEnabled`, mirroring `BanUser`/`UnbanUser`'s shape.

`Category` is a self-referencing tree (`ParentID *int64`, unbounded depth). `Product.CategoryID *int64` is nullable (uncategorized products are valid) and belongs to a `Category`. `ProductItem` is one pre-stocked unit (`IsSold` flag); `Purchase` fulfills exactly one `ProductItem` via `ItemID`, and that column carries a **unique index — an integrity guarantee, not a performance tweak**: it makes selling the same stock item twice impossible at the database level even if the application logic above it were wrong, backstopping the whole `SKIP LOCKED` reservation scheme. Do not weaken it. Buying `count` units creates `count` `Purchase` rows sharing one `BatchID` (a UUID generated once per `Buy()` call) — purchase history groups and paginates by that, not by raw row. `AdminLog` records admin actions (ban, unban, balance_add, make_admin, revoke_admin, product_*, category_*, settings_update) against a target user.

**Indexes** come from GORM tags on the models (every FK is covered, plus `created_at` on `purchases`/`replenishments`/`admin_logs` for the cross-user admin listings, which all sort by it) — with one exception that tags cannot express and that lives as raw DDL in `postgres.AutoMigrate`'s `partialIndexes`: `idx_product_items_unsold` on `product_items (product_id) WHERE is_sold = false`. It exists because sold rows accumulate forever while unsold ones stay few, so a plain `product_id` index would make every stock query walk the entire sales history of that product to filter it out. Four hot paths depend on it — `ReserveItem` (the money path, holding locks), the `inStockClause` on catalog listings, the recursive category-visibility CTE, and `CountAvailableItems` on each product card. Any new "find me an unsold item" query should stay compatible with it.

`Settings` is a singleton row (fixed `ID = models.SettingsID`, bootstrapped by `cmd/migrate`'s `SettingsRepo.EnsureExists`) holding `SupportUsername` plus one embedded sub-struct per merchant (`CrystalPaySettings`/`YooKassaSettings`/`TinkoffSettings` — GORM `embedded;embeddedPrefix:<merchant>_`), each with its own credential fields, `Enabled`, and `MinAmount`/`MaxAmount`. The three merchants' credentials are genuinely different shapes (CrystalPay: Login+Secret+Salt; YooKassa: ShopID+SecretKey; Tinkoff: TerminalKey+Password) so they're three distinct structs, not one generic `Token`/`Secret` pair forced across all of them.

Payments are abstracted behind `payment.PaymentProvider` (`CreateInvoice`/`CheckStatus`, see `internal/service/payment/` above). `Replenishment` is one balance top-up attempt: `UserID`, `Merchant` (`crystalpay`/`yookassa`/`tinkoff`/`referral`), `InvoiceID` (the merchant's own payment/invoice ID — empty for `referral` rows, there's no external invoice), `Amount`, `Status` (`pending`/`paid`/`failed`/`cancelled`). `ReplenishmentSrv.CreateInvoice` calls the matching `PaymentProvider` then inserts a `pending` row; `Confirm`/`Fail` (called only from `payments_backend/handlers`' webhook handlers) transition it via `ReplenishmentRepository.UpdateStatus`, which is a conditional `UPDATE ... WHERE status = 'pending'` — the returned `changed` bool is how `Confirm` stays idempotent against a merchant retrying the same webhook (a no-op past the first successful call, so `UserRepository.UpdateBalance` never double-credits). `Confirm` also takes the amount the merchant reported (`0` when it doesn't report one — CrystalPay's `CheckStatus` returns no amount): a mismatch against the recorded amount is logged at Warn but does **not** change what gets credited, which is always the recorded amount, since that is the figure the user actually confirmed. **`Confirm` wraps that `UpdateStatus` and the `UpdateBalance` in one `Transactor.WithinTransaction`, and this is load-bearing, not tidiness**: because idempotency is keyed on the status, committing the status separately means a failed credit can never be retried — the retry sees `changed == false` and returns success, so the customer's money is gone with no trace and no reconciliation path (nothing polls `CheckStatus`). Cache invalidation deliberately happens *after* the commit, never inside it. Regression tests for both properties: `internal/service/replenishment_service_test.go`.

**Referral program**: `Settings.Referral` (`Enabled` + integer `Percent`, `Enabled=false` overrides `Percent` and turns the whole thing off) is the only global switch — everything else is per-user (`User.ReferrerID`/`ReferralsEnabled` above). Attribution happens once, in `UserSrv.GetOrCreate`, only on the create branch: `bot.go` registers `/start` with `MatchTypeCommandStartOnly` (not `MatchTypeExact`) so `/start <id>` deep-link payloads (`t.me/<bot>?start=<id>`, parsed by `handlers.parseStartPayload`) still match; `GetOrCreate`'s private `validReferrer` helper drops the payload (referral never recorded) if it's self-referral or the ID doesn't belong to an existing user — an *already-existing* user opening a ref link is never attributed, by construction, since that whole code path only runs when the row doesn't exist yet. The actual credit happens in `PurchaseSrv.Buy` via `creditReferral`, **after** the purchase transaction has committed, in a transaction of its own: reads `Settings.Referral` (cached), re-checks the buyer's referrer is real and `ReferralsEnabled`, credits `Percent`% of the purchase total via `UserRepository.UpdateBalance`, and inserts a `Replenishment{Merchant: MerchantReferral, Status: ReplenishmentStatusPaid}` row so the credit shows up in the referrer's "Мои пополнения" like any other top-up. It used to run *inside* the purchase transaction, which quietly contradicted its own best-effort contract: in Postgres any failed statement aborts the whole transaction, `COMMIT` degrades to ROLLBACK, and pgx surfaces `ErrTxCommitRollback` — so a failed *optional* bonus threw away a fully paid purchase. Hence the split: `creditReferral` swallowing its errors is only honest outside the purchase's transaction. Its own two writes are still wrapped together, so a referrer can't end up with credited money and no history row. Side benefit: the cached `Settings` read no longer happens while `FOR UPDATE` locks on stock are held. `Buy` returns `*models.ReferralCredit` (nil if nothing was credited) alongside the purchases — `BuyConfirmHandler` is what actually messages the referrer (`texts.ReferralCreditMsg`), since `PurchaseSrv` has no Telegram dependency and can't send it itself.

### Bot wiring

`bot/bot.go` builds `Middlewares` and `Handlers` from the domain service interfaces plus a `keyboards.Keyboards` (built from `cfg.AdminPanel` — `AdminKb`'s URL comes from `FrontendURL`). The middleware chain is `Track -> Recover -> Logging -> AnswerCallback -> BanCheck -> FSM` — there's no `AutoMigrate` middleware; the user row is only ever created in `StartHandler` on `/start`, and `BanCheck` always lets `/start` itself through (otherwise a brand-new user could never pass it to get created) but otherwise fails closed on real errors, prompts `ErrUserNotFound` users to `/start`, and tells banned users they're banned.

Reply-keyboard text handlers: `/admin`, `texts.{Help,Catalog,Profile,StartMenu,Purchases,RefillBalance,ProfileRefresh,Replenishments,Referral}Btn` (`ProfileRefreshBtn` re-sends the profile card via `UserService.RefreshProfile`, which reads Postgres directly instead of the user cache — purchase stats are already always read straight from Postgres), and `/start` — registered with `MatchTypeCommandStartOnly` rather than `MatchTypeExact` specifically so a ref-link's `/start <id>` payload still matches (see the referral paragraph above); everything else here still uses `MatchTypeExact`. Callback-query (inline button) prefixes, via `bot.MatchTypePrefix`: `product_`, `buy_`, `buyqty_`, `purchase_` (purchase-batch UUID, not a raw row ID), `purchasespage_`, `category_`, `refillmerchant_` (bare merchant string, own Build/Parse pair — not numeric, doesn't fit `ParseCallbackQuery`), `replenishmentspage_`; exact-match: `buycancel`, `buyconfirm`, `catalog_root`, `main_menu`, `referral_close`. `bot/utils.ParseCallbackQuery` parses the trailing numeric segment; `ParseBatchCallbackQuery` parses the trailing UUID (works because UUIDs use hyphens, not underscores, so splitting on `_` and taking the last part is still safe).

`Handlers.botUsername` (fetched once via `b.GetMe(ctx)` in `bot.New`, after the `bot.Bot` exists but before handlers are built) is what `ReferralHandler` uses to build `t.me/<botUsername>?start=<telegramID>` links — there's no config env var for it, since the token already implies it and asking the API once at startup avoids the two ever drifting.

The buy flow is stateful, backed by `internal/cache/redis.Cache` (which doubles as `domain/fsm.Store`): tap "buy" -> edit the product card into a quantity prompt (quick-pick 1–5 inline or type a number, both converge on the same confirmation step) -> stock is checked before showing confirmation (fast-fail if you ask for more than exists, rather than only finding out after confirming) -> confirmation screen -> only `BuyConfirmHandler` actually charges anything and marks items sold. Quantity is bounded by `service.MaxBuyQuantity` (the constant lives in `internal/domain/service` precisely because both `PurchaseSrv.Buy` and the bot's FSM need it — the bot rejects on input, the service has the last word). Inside the transaction `PurchaseSrv.Buy` re-reads the product before charging: the `IsActive`/price checks above it are a fast-fail for the user, not the authority, since an admin can deactivate or reprice between the two.

### Web admin panel

CRUDL for categories/products, view+edit for users (ban/unban, balance, promote/demote admin, plus `POST /api/users/:telegram_id/referrals/{enable,disable}` -> `AdminService.SetReferralsEnabled`), view for purchases (cross-user — everything on `PurchaseService`/`PurchaseRepository` elsewhere is scoped to one Telegram user, so `ListAllAdmin`/`CountAllAdmin`/`GetAdminByID` exist purely for this screen), a cross-user replenishments view (same `ListAllAdmin`/`CountAllAdmin` pattern on `ReplenishmentService` — this is also where `Merchant: referral` credits are visible across all users, not just each referrer's own "Мои пополнения"), an admin audit-log view, a Statistics screen backed by `StatsService`/`StatsRepository` (plain SQL aggregates — unrelated to the Prometheus/Grafana stack below, which covers infra/bot metrics and logs, not shop analytics), and settings (Support username, per-merchant credentials/`Enabled`/min-max, and the referral `Enabled`+`Percent` pair, via `GET`/`PUT /api/settings` — **frontend page not built yet**, only the admin_backend API).

Auth is code-then-session, entirely Redis-backed, nothing in Postgres: `Handlers.AdminHandler` (`/admin`) calls `AdminAuthService.IssueLoginCode`, which generates a 6-digit code (`admintoken.GenerateCode`), stores `sha256(code) -> telegramID` in Redis for 30 seconds (`domain/adminsession.Store`, implemented by the same `internal/cache/redis.Cache` struct as the read-through cache and FSM state, separate keyspace), and sends it back in the `/admin` reply. The login page's `POST /api/auth/exchange` (the *one* unauthenticated route — registered directly on the top-level gin engine, outside the `/api` route group, so it never passes through `Auth`, but it does get `middleware.RateLimitExchange`: 10 attempts per minute per client IP via `AdminAuthService.AllowExchangeAttempt` -> `adminsession.Store.IncrExchangeAttempts`, since a 6-digit code with free failed guesses is otherwise brute-forceable and the prize is a 24h admin session. Per-IP and deliberately not global — a global counter would let one attacker lock every admin out. A Redis error fails closed, which costs nothing because the exchange needs Redis anyway) calls `AdminAuthService.ExchangeLoginCode`: consumes the code atomically (Redis `GETDEL`, so it's single-use even under concurrent attempts), re-checks `IsAdmin()` (a demote between issuance and exchange must not slip through), and issues a 24h HS256 JWT session token (`admintoken.GenerateSessionJWT`, keyed by `ADMIN_JWT_SECRET`) whose hash is *also* stored in Redis the same way the login code was.

`admin_backend/middleware.Auth` resolves the session token via `AdminAuthService.ValidateSession`, which requires all three of: the JWT signature/expiry to check out (`admintoken.ParseSessionJWT`, rejects a forged or expired token before any I/O), the token's hash to still have a live Redis entry (this is what makes the token revocable at all — a signed JWT can't be un-issued, so `Logout` works by deleting this entry instead), and the resolved user to still satisfy `IsAdmin()` in Postgres. The token itself comes from `Authorization: Bearer <session token>` (the SPA) or, failing that, a `session` cookie (`middleware.SessionCookieName`) — `handlers.Exchange` sets both on a successful code exchange, same JWT in each. `GET /api/auth/me`'s 200-vs-401 response *is* the login check from the frontend's perspective; it also sets `X-Admin-Username` on success, which is what lets **Grafana reuse this exact login** — Caddy's `forward_auth` in front of `stats.$DOMAIN_NAME` calls this same endpoint with the cookie, copies that header into `X-WEBAUTH-USER`, and Grafana (`GF_AUTH_PROXY_ENABLED`) trusts it instead of showing its own login form; an unauthenticated visit gets redirected to the admin panel's login page. See the docker-compose paragraph above for the rest of the observability stack. `POST /api/auth/logout` deletes the Redis session entry early (revoking both the header-based and cookie-based session at once, since they're the same Redis entry) and clears the cookie.

`AdminService.MakeAdmin`/`RevokeAdmin`/`BanUser`/`UnbanUser` only ever change `User.Role` — they hand back no credential; a newly promoted admin gets in by sending `/admin` themselves. `MakeAdmin` is root-admin-only (`ErrOnlyRootAdminCanPromote` otherwise, checked against the acting admin's own persisted role) — without that, any admin could promote arbitrary users, who could promote further admins, with nothing containing the spread. Because `Role` is a single field, `BanUser`/`RevokeAdmin` both refuse to target the root admin (`ErrCannotBanRootAdmin`/`ErrCannotRevokeRootAdmin`) or the acting admin's own account (`ErrCannotBanSelf`/`ErrCannotRevokeSelf`) — banning or revoking would otherwise overwrite/strip root status with no one left able to grant it back. `UnbanUser` always restores plain `user`, never whatever role the target held before being banned — a banned former admin needs `MakeAdmin` run again after unban. A demoted admin's still-live session keeps validating JWT-wise until its TTL, but `ValidateSession` re-checks the role against Postgres on every call, so it's rejected on their very next request regardless.

`AdminService.DeleteCategory`/`DeleteProduct` refuse to delete a non-empty category or a product with purchase history (`ErrCategoryNotEmpty`/`ErrProductHasPurchases`) rather than surfacing a raw FK error — reassign/delete children first, or deactivate (`IsActive`) instead of deleting a product that's already sold.

Telegram rejects inline URL buttons pointing at `localhost`/loopback hosts outright (`Wrong HTTP URL`) — a near-certainty in local dev, where `ADMIN_PANEL_FRONTEND_URL` defaults to `http://localhost:3000`. `Handlers.AdminHandler` sends the login code with `kb.AdminKb` (the button) first; if that specifically fails with `bot.ErrorBadRequest`, it retries with a plain-text message carrying the panel URL as a link in the text instead of a button, so the code itself is never blocked by an unclickable link. In production, point `ADMIN_PANEL_FRONTEND_URL` at a real `https://` domain to get the clickable button.

### Caching gotcha worth knowing

`cmd/migrate` (and any direct Postgres edit) has zero awareness of the Redis read-through cache by design — it's a Postgres-only step. In the normal deploy flow this is fine (`migrate` completes before `bot`/`admin_backend`/`payments_backend` even start, so there's nothing cached yet). But re-running `migrate` (or hand-editing a row) against an *already-running* bot/admin_backend can leave a stale cached `User` (10-minute TTL, key `user:<telegram_id>`) — e.g. promoting someone to `root_admin` in Postgres doesn't retroactively fix a session that already cached their old role. Fix: `redis-cli DEL user:<telegram_id>`, or wait out the TTL.
