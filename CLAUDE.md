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
go test ./...                                      # run tests (no test files exist yet)
docker compose up --build   # run migrate + bot + admin_backend + payments_backend + Postgres + Redis + admin_frontend together (reads .env)
```

There is no Makefile or linter config in the repo — use the `go` toolchain directly.

Frontend (`admin_frontend/`, separate npm project, not part of the Go module):

```bash
cd admin_frontend && npm install
npm run dev      # Vite dev server on :3000, needs VITE_API_BASE_URL pointed at a running admin_backend (defaults to http://localhost:8080)
npm run build    # tsc -b + vite build -> dist/ — what admin_frontend/Dockerfile bundles into nginx
```

### Local configuration

Config is loaded from environment variables via [internal/config/config.go](internal/config/config.go), no config file. Copy `.env.example` to `.env` and fill in `TELEGRAM_BOT_TOKEN`, `TELEGRAM_ROOT_ADMIN_ID`, and `ADMIN_JWT_SECRET` (all required — `config.New()` errors without them; DB user/password/name are also required). Also present: `POSTGRES_HOST/PORT/USER/PASSWORD/NAME/SSLMODE`, `REDIS_ADDR/PASSWORD/DB`, `ADMIN_PANEL_BACKEND_PORT/BACKEND_URL/FRONTEND_URL/CORS_ORIGIN` (`Port` is `admin_backend`'s own listen port; `BACKEND_URL` is read only by docker-compose as the `admin_frontend` build's `VITE_API_BASE_URL` value — Go's `AdminPanelConfig` no longer has a `URL` field, since nothing in Go read it after the payments split; `FrontendURL` is where the React panel itself is served, which is what the bot's `/admin` inline button links to; `CORSOrigin` is a comma-separated list of exact origins — normally just the frontend's own — the API accepts cross-origin requests from; defaults cover both `localhost` and `127.0.0.1` on port 3000, since browsers treat those as different origins), `PAYMENTS_BACKEND_PORT/URL` (`payments_backend`'s own listen port and its externally-reachable URL — `cmd/bot/providers.go` builds CrystalPay/Tinkoff webhook callback URLs from `URL`; YooKassa doesn't take a per-invoice callback URL, so it's unaffected).

**`docker compose` needs a `.env` at the repo root** — read both by Compose's own `${VAR}` substitution (the `db`/`redis` services' `environment:` blocks, and the `admin_backend`/`payments_backend`/`admin_frontend` services' `${ADMIN_PANEL_BACKEND_PORT}`/`${PAYMENTS_BACKEND_PORT}`/`${ADMIN_PANEL_BACKEND_URL}` substitutions) and, via each service's `env_file: .env`, by the containers themselves. `docker-compose.yml` wires Postgres, Redis, a one-shot `migrate`, `bot`, `admin_backend`, `payments_backend`, and the `admin_frontend` container together, on two networks: `backend-network` (internal-only, db+redis+bot+admin_backend+payments_backend+migrate) and `public-network` (bot, for outbound Telegram API calls; admin_backend, payments_backend, and admin_frontend, so their published ports actually reach the host — a container whose *only* network is `internal: true` can't have a published port reached even though the mapping is declared, see docker-compose.yml's network comments). There's no admin credential to retrieve from logs anymore — every admin, including the root admin, logs in by sending `/admin` to the bot.

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
                               ClearFSMState — spelled out, not bare Get/Set/Clear, since the same Redis-backed
                               struct also implements domain/cache's per-entity interfaces)
internal/domain/adminsession/ Store interface — web-panel one-time login codes + sessions (Redis-backed, nothing
                               in Postgres); a distinct bounded concern from domain/cache and domain/fsm, just
                               implemented by the same Redis-backed struct
internal/domain/errors/       sentinel error values, mapped to user-facing text by bot/texts.UserFacingError, and
                               to HTTP status/JSON by admin_backend/errors.DomainErrorToResponse

internal/repository/postgres/ GORM-backed implementations, using the Generics API (gorm.G[T]) for CRUD — aggregate
                               queries (e.g. grouping purchases into batches, dashboard stats) use the classic
                               *gorm.DB chainable builder (Model/Select/Joins/Where/Group/Scan) instead, since
                               gorm.G[T] assumes one row = one T; the one recursive-CTE query (category tree
                               visibility) is the sole .Raw(...).Scan(...) case, since neither of the above can
                               express recursion. migrate.go's AutoMigrate is cmd/migrate's single entry point: DDL,
                               two unexported one-time schema cleanups left over from earlier iterations
                               (backfillUserRoles, dropLegacyAdminTokenColumn — both no-ops once already run), then
                               bootstrapping the root admin (UserRepo.EnsureRootAdminExists) and the default
                               Settings row (SettingsRepo.EnsureExists) — everything cmd/migrate needs, in one call
internal/service/              implementations of internal/domain/service — cache-aside on every read (check
                               cache, miss -> repo, populate cache), explicit invalidation on every write; admin
                               listing methods (*Admin/*All suffix) deliberately skip the cache and always read
                               through to Postgres, since they're tuned for freshness over throughput, not the
                               customer-facing catalog
internal/service/payment/     PaymentProvider implementations: StubProvider (always errors, unused now that real
                               providers exist), CrystalPayProvider (hand-rolled HTTP client, no official Go SDK),
                               YooKassaProvider (github.com/rvinnie/yookassa-sdk-go), TinkoffProvider
                               (github.com/nikita-vanyasin/tinkoff, PayTypeOneStep). All three read their own
                               Settings sub-struct (credentials + Enabled + Min/MaxAmount) fresh on every call via
                               SettingsService — admin edits apply without restarting the bot — and each does its
                               own enabled/range check before calling out, returning ErrMerchantDisabled/
                               ErrAmountOutOfRange. CheckStatus is implemented on all three but nothing currently
                               polls it — confirmation is webhook-driven (see payments_backend/handlers below)
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
bot/middleware/                middleware.go (Logging: logs every update at Debug, first in the chain so nothing
                               skips it), answer_callback.go (acks every callback_query first, or the tapped
                               button stays "loading" until Telegram times it out), ban_check.go (the only
                               per-update user-exists/banned gate — see below), fsm.go (multi-step flows: buy
                               quantity+confirm, refill amount) — registered in that order: Logging ->
                               AnswerCallback -> BanCheck -> FSM. Refill's merchant pick (handlers/refill.go's
                               RefillMerchantHandler, a plain inline callback) happens before FSM state exists;
                               only the subsequent amount prompt is FSM-tracked (State.Merchant carries the pick
                               through to fsm.go's handleRefillAmount)
bot/keyboards/                 Keyboards struct (static reply/inline keyboards, built once via New(adminPanelURL)
                               — not package-level vars, since AdminKb needs the URL) plus per-request builder funcs
bot/texts/                     all user-facing button/message strings — add new UI copy here, not inline in handlers
bot/utils/                     callback.go (build/parse callback_data — numeric "prefix_<id>" and, for purchase
                               batches, "purchase_<uuid>"; refillmerchant_<merchant> carries a bare string, not a
                               number, so it gets its own Build/Parse pair instead of reusing ParseCallbackQuery),
                               errors.go (domain sentinel error -> user-facing text, UserFacingError), stock.go
                               (traffic-light emoji for remaining stock), merchant.go (Merchant/ReplenishmentStatus
                               -> Russian display text, used by both the merchant picker and replenishments.go)

internal/auth/web/            package admintoken (import resolves by package clause, not directory name) —
                               crypto/rand + sha256 generation/hashing for the web panel's one-time login codes
                               (6-digit, GenerateCode), plus HS256 JWT session tokens (GenerateSessionJWT/
                               ParseSessionJWT, keyed by ADMIN_JWT_SECRET) — a leaf utility package (no domain
                               imports) used only by internal/service/admin_auth_service.go. The JWT is
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
                               ADMIN_PANEL_CORS_ORIGIN, no credentials — logs rejected origins at Warn for diagnosis)
admin_backend/dto/             request bodies + the Paginated[T] list envelope + ErrorResponse; responses mostly
                               reuse internal/domain/models types directly (already have clean json tags)
admin_backend/errors/          domain sentinel error -> HTTP status + JSON body (DomainErrorToResponse), mirrors
                               bot/texts.UserFacingError
admin_backend/router.go,       route table + http.Server lifecycle (Start/Shutdown). /api/auth/exchange is the
  server.go                    only route outside the /api Auth group (no session exists yet to check) — every
                               other route requires a valid admin session

payments_backend/              accepts payment-provider webhooks only (package `paymentsbackend`, directory
                               `payments_backend/`, binary entrypoint is cmd/payments_backend) — gin, depends only
                               on internal/domain/service, same rule as bot/ and admin_backend/. Deliberately tiny:
                               paymentsbackend.New(...) takes just SettingsService + ReplenishmentService, nothing
                               admin-related (no AdminAuthService, no session store, no CORS middleware — the
                               caller is a merchant's server, never a browser)
payments_backend/handlers/     handler.go (the 2-service Handlers struct/constructor) + webhook_handler.go
                               (CrystalPayWebhook/YooKassaWebhook/TinkoffWebhook — moved here verbatim from the old
                               single `backend` binary; each verifies that merchant's own signature scheme before
                               calling ReplenishmentService.Confirm/Fail — see the Data model section below)
payments_backend/router.go,    three POST routes (/api/webhooks/{crystalpay,yookassa,tinkoff}), gin.Recovery()
  server.go                    only, no Auth group, no CORS — http.Server lifecycle (Start/Shutdown)

internal/config/               env-var config loader (Telegram/Postgres/Redis/AdminPanel/Payments/Logger sub-configs)
internal/logger/               logrus logger construction from config.LoggerConfig, plus fx.go's NewFxLogger
                               (routes fx's own PROVIDE/INVOKE/START/STOP event log through the same logrus
                               logger at Debug — quiet in prod, visible with LOG_LEVEL=debug); the only
                               importers of this package are the four cmd/* binaries

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
cmd/bot/main.go                fx.New(...).Run() — Run() itself blocks and listens for SIGINT/SIGTERM (no
                               manual signal.NotifyContext anymore). lifecycle.go's runBot launches
                               bt.Start(ctx) in a goroutine from OnStart (Start blocks on long-polling, so it
                               can't run inline — OnStart hooks are expected to return quickly) and cancels
                               that ctx from OnStop, then closes the redis client
cmd/admin_backend/main.go      fx.New(...).Run(), same signal handling as cmd/bot. lifecycle.go's runServer
                               launches webServer.Start() in a goroutine from OnStart (it blocks until
                               Shutdown) and calls webServer.Shutdown(ctx) (10s timeout) then closes the redis
                               client from OnStop. Wires all repos/services admin_backend/handlers needs: eight
                               repos (User/Product/Purchase/Category/Settings/Replenishment/AdminLog/Stats) plus
                               GormTransactor, seven services (User/Product/Category/Settings/Purchase/Admin/Stats
                               via fx.Annotate) plus provideReplenishmentService/provideAdminAuthService
cmd/payments_backend/main.go   Same fx.New(...).Run()/runServer shape as cmd/admin_backend, but a much narrower
                               graph: three repos (User/Settings/Replenishment, no GormTransactor), one service
                               via fx.Annotate (SettingsSrv) plus provideReplenishmentService — no
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

`Category` is a self-referencing tree (`ParentID *int64`, unbounded depth). `Product.CategoryID *int64` is nullable (uncategorized products are valid) and belongs to a `Category`. `ProductItem` is one pre-stocked unit (`IsSold` flag); `Purchase` fulfills exactly one `ProductItem` via `ItemID` (unique). Buying `count` units creates `count` `Purchase` rows sharing one `BatchID` (a UUID generated once per `Buy()` call) — purchase history groups and paginates by that, not by raw row. `AdminLog` records admin actions (ban, unban, balance_add, make_admin, revoke_admin, product_*, category_*, settings_update) against a target user.

`Settings` is a singleton row (fixed `ID = models.SettingsID`, bootstrapped by `cmd/migrate`'s `SettingsRepo.EnsureExists`) holding `SupportUsername` plus one embedded sub-struct per merchant (`CrystalPaySettings`/`YooKassaSettings`/`TinkoffSettings` — GORM `embedded;embeddedPrefix:<merchant>_`), each with its own credential fields, `Enabled`, and `MinAmount`/`MaxAmount`. The three merchants' credentials are genuinely different shapes (CrystalPay: Login+Secret+Salt; YooKassa: ShopID+SecretKey; Tinkoff: TerminalKey+Password) so they're three distinct structs, not one generic `Token`/`Secret` pair forced across all of them.

Payments are abstracted behind `payment.PaymentProvider` (`CreateInvoice`/`CheckStatus`, see `internal/service/payment/` above). `Replenishment` is one balance top-up attempt: `UserID`, `Merchant` (`crystalpay`/`yookassa`/`tinkoff`/`referral`), `InvoiceID` (the merchant's own payment/invoice ID — empty for `referral` rows, there's no external invoice), `Amount`, `Status` (`pending`/`paid`/`failed`/`cancelled`). `ReplenishmentSrv.CreateInvoice` calls the matching `PaymentProvider` then inserts a `pending` row; `Confirm`/`Fail` (called only from `payments_backend/handlers`' webhook handlers) transition it via `ReplenishmentRepository.UpdateStatus`, which is a conditional `UPDATE ... WHERE status = 'pending'` — the returned `changed` bool is how `Confirm` stays idempotent against a merchant retrying the same webhook (a no-op past the first successful call, so `UserRepository.UpdateBalance` never double-credits).

**Referral program**: `Settings.Referral` (`Enabled` + integer `Percent`, `Enabled=false` overrides `Percent` and turns the whole thing off) is the only global switch — everything else is per-user (`User.ReferrerID`/`ReferralsEnabled` above). Attribution happens once, in `UserSrv.GetOrCreate`, only on the create branch: `bot.go` registers `/start` with `MatchTypeCommandStartOnly` (not `MatchTypeExact`) so `/start <id>` deep-link payloads (`t.me/<bot>?start=<id>`, parsed by `handlers.parseStartPayload`) still match; `GetOrCreate`'s private `validReferrer` helper drops the payload (referral never recorded) if it's self-referral or the ID doesn't belong to an existing user — an *already-existing* user opening a ref link is never attributed, by construction, since that whole code path only runs when the row doesn't exist yet. The actual credit happens in `PurchaseSrv.Buy`, inside the same DB transaction as the purchase itself (`creditReferral`, called after the buyer's balance is debited): reads `Settings.Referral` (cached), re-checks the buyer's referrer is real and `ReferralsEnabled`, credits `Percent`% of the purchase total via `UserRepository.UpdateBalance`, and inserts a `Replenishment{Merchant: MerchantReferral, Status: ReplenishmentStatusPaid}` row so the credit shows up in the referrer's "Мои пополнения" like any other top-up. `Buy` returns `*models.ReferralCredit` (nil if nothing was credited) alongside the purchases — `BuyConfirmHandler` is what actually messages the referrer (`texts.ReferralCreditMsg`), since `PurchaseSrv` has no Telegram dependency and can't send it itself.

### Bot wiring

`bot/bot.go` builds `Middlewares` and `Handlers` from the domain service interfaces plus a `keyboards.Keyboards` (built from `cfg.AdminPanel` — `AdminKb`'s URL comes from `FrontendURL`). The middleware chain is `Logging -> AnswerCallback -> BanCheck -> FSM` — there's no `AutoMigrate` middleware; the user row is only ever created in `StartHandler` on `/start`, and `BanCheck` always lets `/start` itself through (otherwise a brand-new user could never pass it to get created) but otherwise fails closed on real errors, prompts `ErrUserNotFound` users to `/start`, and tells banned users they're banned.

Reply-keyboard text handlers: `/admin`, `texts.{Help,Catalog,Profile,StartMenu,Purchases,RefillBalance,ProfileRefresh,Replenishments,Referral}Btn` (`ProfileRefreshBtn` re-sends the profile card via `UserService.RefreshProfile`, which reads Postgres directly instead of the user cache — purchase stats are already always read straight from Postgres), and `/start` — registered with `MatchTypeCommandStartOnly` rather than `MatchTypeExact` specifically so a ref-link's `/start <id>` payload still matches (see the referral paragraph above); everything else here still uses `MatchTypeExact`. Callback-query (inline button) prefixes, via `bot.MatchTypePrefix`: `product_`, `buy_`, `buyqty_`, `purchase_` (purchase-batch UUID, not a raw row ID), `purchasespage_`, `category_`, `refillmerchant_` (bare merchant string, own Build/Parse pair — not numeric, doesn't fit `ParseCallbackQuery`), `replenishmentspage_`; exact-match: `buycancel`, `buyconfirm`, `catalog_root`, `main_menu`, `referral_close`. `bot/utils.ParseCallbackQuery` parses the trailing numeric segment; `ParseBatchCallbackQuery` parses the trailing UUID (works because UUIDs use hyphens, not underscores, so splitting on `_` and taking the last part is still safe).

`Handlers.botUsername` (fetched once via `b.GetMe(ctx)` in `bot.New`, after the `bot.Bot` exists but before handlers are built) is what `ReferralHandler` uses to build `t.me/<botUsername>?start=<telegramID>` links — there's no config env var for it, since the token already implies it and asking the API once at startup avoids the two ever drifting.

The buy flow is stateful, backed by `internal/cache/redis.Cache` (which doubles as `domain/fsm.Store`): tap "buy" -> edit the product card into a quantity prompt (quick-pick 1–5 inline or type a number, both converge on the same confirmation step) -> stock is checked before showing confirmation (fast-fail if you ask for more than exists, rather than only finding out after confirming) -> confirmation screen -> only `BuyConfirmHandler` actually charges anything and marks items sold.

### Web admin panel

CRUDL for categories/products, view+edit for users (ban/unban, balance, promote/demote admin, plus `POST /api/users/:telegram_id/referrals/{enable,disable}` -> `AdminService.SetReferralsEnabled`), view for purchases (cross-user — everything on `PurchaseService`/`PurchaseRepository` elsewhere is scoped to one Telegram user, so `ListAllAdmin`/`CountAllAdmin`/`GetAdminByID` exist purely for this screen), a cross-user replenishments view (same `ListAllAdmin`/`CountAllAdmin` pattern on `ReplenishmentService` — this is also where `Merchant: referral` credits are visible across all users, not just each referrer's own "Мои пополнения"), an admin audit-log view, a Statistics screen backed by `StatsService`/`StatsRepository` (plain SQL aggregates, no Prometheus/Grafana), and settings (Support username, per-merchant credentials/`Enabled`/min-max, and the referral `Enabled`+`Percent` pair, via `GET`/`PUT /api/settings` — **frontend page not built yet**, only the admin_backend API).

Auth is code-then-session, entirely Redis-backed, nothing in Postgres: `Handlers.AdminHandler` (`/admin`) calls `AdminAuthService.IssueLoginCode`, which generates a 6-digit code (`admintoken.GenerateCode`), stores `sha256(code) -> telegramID` in Redis for 30 seconds (`domain/adminsession.Store`, implemented by the same `internal/cache/redis.Cache` struct as the read-through cache and FSM state, separate keyspace), and sends it back in the `/admin` reply. The login page's `POST /api/auth/exchange` (the *one* unauthenticated route — registered directly on the top-level gin engine, outside the `/api` route group, so it never passes through `Auth`) calls `AdminAuthService.ExchangeLoginCode`: consumes the code atomically (Redis `GETDEL`, so it's single-use even under concurrent attempts), re-checks `IsAdmin()` (a demote between issuance and exchange must not slip through), and issues a 24h HS256 JWT session token (`admintoken.GenerateSessionJWT`, keyed by `ADMIN_JWT_SECRET`) whose hash is *also* stored in Redis the same way the login code was.

`admin_backend/middleware.Auth` resolves the incoming `Authorization: Bearer <session token>` header via `AdminAuthService.ValidateSession`, which requires all three of: the JWT signature/expiry to check out (`admintoken.ParseSessionJWT`, rejects a forged or expired token before any I/O), the token's hash to still have a live Redis entry (this is what makes the token revocable at all — a signed JWT can't be un-issued, so `Logout` works by deleting this entry instead), and the resolved user to still satisfy `IsAdmin()` in Postgres. `GET /api/auth/me`'s 200-vs-401 response *is* the login check from the frontend's perspective. `POST /api/auth/logout` deletes the Redis session entry early, which is what actually revokes the JWT before its natural expiry.

`AdminService.MakeAdmin`/`RevokeAdmin`/`BanUser`/`UnbanUser` only ever change `User.Role` — they hand back no credential; a newly promoted admin gets in by sending `/admin` themselves. `MakeAdmin` is root-admin-only (`ErrOnlyRootAdminCanPromote` otherwise, checked against the acting admin's own persisted role) — without that, any admin could promote arbitrary users, who could promote further admins, with nothing containing the spread. Because `Role` is a single field, `BanUser`/`RevokeAdmin` both refuse to target the root admin (`ErrCannotBanRootAdmin`/`ErrCannotRevokeRootAdmin`) or the acting admin's own account (`ErrCannotBanSelf`/`ErrCannotRevokeSelf`) — banning or revoking would otherwise overwrite/strip root status with no one left able to grant it back. `UnbanUser` always restores plain `user`, never whatever role the target held before being banned — a banned former admin needs `MakeAdmin` run again after unban. A demoted admin's still-live session keeps validating JWT-wise until its TTL, but `ValidateSession` re-checks the role against Postgres on every call, so it's rejected on their very next request regardless.

`AdminService.DeleteCategory`/`DeleteProduct` refuse to delete a non-empty category or a product with purchase history (`ErrCategoryNotEmpty`/`ErrProductHasPurchases`) rather than surfacing a raw FK error — reassign/delete children first, or deactivate (`IsActive`) instead of deleting a product that's already sold.

Telegram rejects inline URL buttons pointing at `localhost`/loopback hosts outright (`Wrong HTTP URL`) — a near-certainty in local dev, where `ADMIN_PANEL_FRONTEND_URL` defaults to `http://localhost:3000`. `Handlers.AdminHandler` sends the login code with `kb.AdminKb` (the button) first; if that specifically fails with `bot.ErrorBadRequest`, it retries with a plain-text message carrying the panel URL as a link in the text instead of a button, so the code itself is never blocked by an unclickable link. In production, point `ADMIN_PANEL_FRONTEND_URL` at a real `https://` domain to get the clickable button.

### Caching gotcha worth knowing

`cmd/migrate` (and any direct Postgres edit) has zero awareness of the Redis read-through cache by design — it's a Postgres-only step. In the normal deploy flow this is fine (`migrate` completes before `bot`/`admin_backend`/`payments_backend` even start, so there's nothing cached yet). But re-running `migrate` (or hand-editing a row) against an *already-running* bot/admin_backend can leave a stale cached `User` (10-minute TTL, key `user:<telegram_id>`) — e.g. promoting someone to `root_admin` in Postgres doesn't retroactively fix a session that already cached their old role. Fix: `redis-cli DEL user:<telegram_id>`, or wait out the TTL.
