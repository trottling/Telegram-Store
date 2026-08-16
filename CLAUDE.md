# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Telegram shop bot written in Go, using [go-telegram/bot](https://github.com/go-telegram/bot) for the Telegram API, GORM (Postgres driver) for persistence, and Redis for both a read-through cache and per-chat FSM conversation state. Users browse a category tree of arbitrary depth, buy products (a multi-step flow: pick quantity → confirm → charge, decrementing pre-stocked `ProductItem` records), top up a balance (UI is done, provider is a stub), and view purchase history. Admin actions (ban/unban, balance, product/category CRUD, promote/demote) are implemented as domain services but have **no UI in the bot** — the bot's `/admin` command replies with a one-time login code for the web panel instead.

The actual admin UI is a separate web panel: a Go+gin JSON API (`backend`, its own binary/container — `cmd/backend`), plus a React+Ant Design frontend (`frontend/`, its own container) that talks to that API cross-origin. Auth has no persistent credential at all: `/admin` issues a 30-second one-time code, the login page exchanges it for a JWT session token (also backed by a Redis-held revocation record), and a fresh code is issued every time — nothing admin-related is ever stored in Postgres.

The three Go binaries (`cmd/bot`, `cmd/backend`, `cmd/migrate`) run as three independent, concurrently-started containers/processes sharing one Postgres and one Redis — see [Commands](#commands) and `cmd/migrate`'s own doc comment for why schema setup is its own one-shot step rather than folded into either long-running service.

## Commands

```bash
go build ./...                     # build everything
go build -o bot ./cmd/bot          # build the Telegram bot binary
go build -o backend ./cmd/backend  # build the admin API binary
go build -o migrate ./cmd/migrate  # build the one-shot migration binary
go run ./cmd/bot                   # run the bot locally (needs .env populated, see below)
go run ./cmd/backend                # run the admin API locally
go vet ./...                        # static checks
go test ./...                       # run tests (no test files exist yet)
docker compose up --build   # run migrate + bot + backend + Postgres + Redis + admin frontend together (reads .env)
```

There is no Makefile or linter config in the repo — use the `go` toolchain directly.

Frontend (`frontend/`, separate npm project, not part of the Go module):

```bash
cd frontend && npm install
npm run dev      # Vite dev server on :3000, needs VITE_API_BASE_URL pointed at a running backend (defaults to http://localhost:8080)
npm run build    # tsc -b + vite build -> dist/ — what frontend/Dockerfile bundles into nginx
```

### Local configuration

Config is loaded from environment variables via [internal/config/config.go](internal/config/config.go), no config file. Copy `.env.example` to `.env` and fill in `TELEGRAM_BOT_TOKEN`, `TELEGRAM_ROOT_ADMIN_ID`, and `ADMIN_JWT_SECRET` (all required — `config.New()` errors without them; DB user/password/name are also required). Also present: `POSTGRES_HOST/PORT/USER/PASSWORD/NAME/SSLMODE`, `REDIS_ADDR/PASSWORD/DB`, `ADMIN_PANEL_BACKEND_PORT/BACKEND_URL/FRONTEND_URL/CORS_ORIGIN` (the admin API's own port and externally-reachable URL — what the frontend build's `VITE_API_BASE_URL` points at — separately `FRONTEND_URL`, where the React panel itself is served, which is what the bot's `/admin` inline button links to; plus a comma-separated list of exact origins — normally just the frontend's own — the API accepts cross-origin requests from; defaults cover both `localhost` and `127.0.0.1` on port 3000, since browsers treat those as different origins).

**`docker compose` needs a `.env` at the repo root** — read both by Compose's own `${VAR}` substitution (the `db`/`redis` services' `environment:` blocks, and the `backend`/`frontend` services' `${ADMIN_PANEL_BACKEND_PORT}`/`${ADMIN_PANEL_BACKEND_URL}` substitutions) and, via each service's `env_file: .env`, by the containers themselves. `docker-compose.yml` wires Postgres, Redis, a one-shot `migrate`, `bot`, `backend`, and the admin `frontend` container together, on two networks: `backend-network` (internal-only, db+redis+bot+backend+migrate) and `public-network` (bot, for outbound Telegram API calls; backend and frontend, so their published ports actually reach the host — a container whose *only* network is `internal: true` can't have a published port reached even though the mapping is declared, see docker-compose.yml's network comments). There's no admin credential to retrieve from logs anymore — every admin, including the root admin, logs in by sending `/admin` to the bot.

## Architecture

**Ports-and-adapters (hexagonal)**: interfaces live under `internal/domain/`, concrete implementations live in parallel packages. When implementing a repository or service, put the interface in `internal/domain/...` (if not already defined) and the struct that satisfies it in the corresponding non-domain package — do not define new interfaces outside `internal/domain`. Both `bot/` and `backend/` depend only on `internal/domain/service` — never on `internal/domain/repository`, `internal/repository/postgres`, or a concrete `internal/service` implementation directly. `cmd/bot/main.go`, `cmd/backend/main.go`, and `cmd/migrate/main.go` are the three composition roots: each one is the only place in its binary that's allowed to import `internal/repository/postgres` and `internal/service`, wiring concrete repos/services and handing only the domain interfaces down.

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
internal/domain/errors/       sentinel error values, mapped to user-facing text by bot/utils.UserFacingError, and
                               to HTTP status/JSON by backend/errors.DomainErrorToResponse

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
                               polls it — confirmation is webhook-driven (see backend/handlers below)
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

backend/                       the web admin panel's HTTP API (package `backend`, binary entrypoint is
                               cmd/backend) — gin, depends only on internal/domain/service, same rule as bot/.
                               backend.New(...) wires everything; cmd/backend/main.go is the composition root
backend/handlers/              one file per resource; read-only admin listings (all products/categories/purchases/
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
                               happen via webhook_handler.go's three handlers (CrystalPay/YooKassa/Tinkoff), each
                               verifying that merchant's own signature scheme before calling ReplenishmentService.
                               Confirm/Fail; none of them go through AdminService (no admin acted, nothing to audit)
backend/middleware/            auth.go (bearer session token -> AdminAuthService.ValidateSession, attaches
                               *models.User to request context), cors.go (comma-separated allowed-origin list via
                               ADMIN_PANEL_CORS_ORIGIN, no credentials — logs rejected origins at Warn for diagnosis)
backend/dto/                   request bodies + the Paginated[T] list envelope + ErrorResponse; responses mostly
                               reuse internal/domain/models types directly (already have clean json tags)
backend/errors/                domain sentinel error -> HTTP status + JSON body (DomainErrorToResponse), mirrors
                               bot/utils.UserFacingError
backend/router.go, server.go   route table + http.Server lifecycle (Start/Shutdown). /api/webhooks/{crystalpay,
                               yookassa,tinkoff} sit next to /api/auth/exchange outside the /api Auth group — the
                               caller is the merchant's server, not a logged-in admin, so there's no session to check

internal/config/               env-var config loader (Telegram/Postgres/Redis/AdminPanel/Logger sub-configs)
internal/logger/               logrus logger construction from config.LoggerConfig
cmd/migrate/main.go            one-shot: config -> logger -> postgres -> pgdb.AutoMigrate(ctx, db, log,
                               rootAdminID) -> exit — that single call does the DDL, legacy-column cleanups, and
                               root-admin/Settings bootstrap (see internal/repository/postgres above); main.go
                               itself has no migration logic of its own. Its own doc comment explains why this is a
                               separate binary/container from cmd/bot and cmd/backend: two independent,
                               concurrently-started long-running services can't both own "run AutoMigrate on
                               startup" without racing
cmd/bot/main.go                entrypoint: config -> logger -> redis client -> postgres (schema already migrated
                               by cmd/migrate) -> repos -> services -> bot.New -> bot.Start (blocks) -> on
                               ctx.Done(): redis close
cmd/backend/main.go            entrypoint: config -> logger -> redis client -> postgres (schema already migrated
                               by cmd/migrate) -> repos -> services -> backend.New -> web server goroutine -> on
                               ctx.Done(): graceful web server shutdown -> redis close

frontend/                      the web admin panel's UI — React + Vite + Ant Design, its own package.json/Dockerfile,
                               deployed as a separate container (nginx serving the built static bundle) that calls
                               backend's API cross-origin (VITE_API_BASE_URL, baked in at image build time, not read
                               at runtime). Not part of the Go module. Every protected page is React.lazy()-loaded
                               (see App.tsx) — StatsPage alone pulls in @ant-design/plots (~1.4MB), which shouldn't
                               block loading the login screen or any other page
```

All repositories and services are fully implemented (not stubs) and logging (`*logrus.Logger`, threaded in from each `main.go`) is wired through every repo/cache/service constructor.

### Data model

`User` is keyed by `TelegramID` directly — there's no separate internal auto-increment ID; every FK that points at a user (`Purchase.UserID`, `AdminLog.AdminID`/`TargetID`) stores the Telegram ID. `User.Role` is a single, mutually exclusive privilege level (`banned`/`user`/`admin`/`root_admin`, `models.Role`) — there's exactly one `root_admin` at a time (the `TELEGRAM_ROOT_ADMIN_ID` from config, bootstrapped into this column by `cmd/migrate`'s `UserRepository.EnsureRootAdminExists`). `User.IsBanned()`/`IsAdmin()`/`IsRootAdmin()` are the derived-boolean helper methods everything else checks against, not the raw `Role` field. Because `Role` is one field, banning an Admin/RootAdmin overwrites their admin rights rather than sitting next to them, and un-banning always restores plain `user`, never whatever role they held before (see `AdminSrv.BanUser`/`UnbanUser`).

`User.ReferrerID *int64` records who invited this user (nil if none) — set exactly once, at row creation, and never touched again (see `UserSrv.GetOrCreate` and the referral paragraph below). `User.ReferralsEnabled` (`default:true`, relies on GORM substituting the tag's default for an omitted/zero-value bool field on `Create` — the same mechanism `Role`'s `default:'user'` already depends on) gates whether *this user, as a referrer*, still earns credit; it's flipped by `AdminSrv.SetReferralsEnabled`, mirroring `BanUser`/`UnbanUser`'s shape.

`Category` is a self-referencing tree (`ParentID *int64`, unbounded depth). `Product.CategoryID *int64` is nullable (uncategorized products are valid) and belongs to a `Category`. `ProductItem` is one pre-stocked unit (`IsSold` flag); `Purchase` fulfills exactly one `ProductItem` via `ItemID` (unique). Buying `count` units creates `count` `Purchase` rows sharing one `BatchID` (a UUID generated once per `Buy()` call) — purchase history groups and paginates by that, not by raw row. `AdminLog` records admin actions (ban, unban, balance_add, make_admin, revoke_admin, product_*, category_*, settings_update) against a target user.

`Settings` is a singleton row (fixed `ID = models.SettingsID`, bootstrapped by `cmd/migrate`'s `SettingsRepo.EnsureExists`) holding `SupportUsername` plus one embedded sub-struct per merchant (`CrystalPaySettings`/`YooKassaSettings`/`TinkoffSettings` — GORM `embedded;embeddedPrefix:<merchant>_`), each with its own credential fields, `Enabled`, and `MinAmount`/`MaxAmount`. The three merchants' credentials are genuinely different shapes (CrystalPay: Login+Secret+Salt; YooKassa: ShopID+SecretKey; Tinkoff: TerminalKey+Password) so they're three distinct structs, not one generic `Token`/`Secret` pair forced across all of them.

Payments are abstracted behind `payment.PaymentProvider` (`CreateInvoice`/`CheckStatus`, see `internal/service/payment/` above). `Replenishment` is one balance top-up attempt: `UserID`, `Merchant` (`crystalpay`/`yookassa`/`tinkoff`/`referral`), `InvoiceID` (the merchant's own payment/invoice ID — empty for `referral` rows, there's no external invoice), `Amount`, `Status` (`pending`/`paid`/`failed`/`cancelled`). `ReplenishmentSrv.CreateInvoice` calls the matching `PaymentProvider` then inserts a `pending` row; `Confirm`/`Fail` (called only from `backend/handlers`' webhook handlers) transition it via `ReplenishmentRepository.UpdateStatus`, which is a conditional `UPDATE ... WHERE status = 'pending'` — the returned `changed` bool is how `Confirm` stays idempotent against a merchant retrying the same webhook (a no-op past the first successful call, so `UserRepository.UpdateBalance` never double-credits).

**Referral program**: `Settings.Referral` (`Enabled` + integer `Percent`, `Enabled=false` overrides `Percent` and turns the whole thing off) is the only global switch — everything else is per-user (`User.ReferrerID`/`ReferralsEnabled` above). Attribution happens once, in `UserSrv.GetOrCreate`, only on the create branch: `bot.go` registers `/start` with `MatchTypeCommandStartOnly` (not `MatchTypeExact`) so `/start <id>` deep-link payloads (`t.me/<bot>?start=<id>`, parsed by `handlers.parseStartPayload`) still match; `GetOrCreate`'s private `validReferrer` helper drops the payload (referral never recorded) if it's self-referral or the ID doesn't belong to an existing user — an *already-existing* user opening a ref link is never attributed, by construction, since that whole code path only runs when the row doesn't exist yet. The actual credit happens in `PurchaseSrv.Buy`, inside the same DB transaction as the purchase itself (`creditReferral`, called after the buyer's balance is debited): reads `Settings.Referral` (cached), re-checks the buyer's referrer is real and `ReferralsEnabled`, credits `Percent`% of the purchase total via `UserRepository.UpdateBalance`, and inserts a `Replenishment{Merchant: MerchantReferral, Status: ReplenishmentStatusPaid}` row so the credit shows up in the referrer's "Мои пополнения" like any other top-up. `Buy` returns `*models.ReferralCredit` (nil if nothing was credited) alongside the purchases — `BuyConfirmHandler` is what actually messages the referrer (`texts.ReferralCreditMsg`), since `PurchaseSrv` has no Telegram dependency and can't send it itself.

### Bot wiring

`bot/bot.go` builds `Middlewares` and `Handlers` from the domain service interfaces plus a `keyboards.Keyboards` (built from `cfg.AdminPanel.URL`). The middleware chain is `Logging -> AnswerCallback -> BanCheck -> FSM` — there's no `AutoMigrate` middleware; the user row is only ever created in `StartHandler` on `/start`, and `BanCheck` always lets `/start` itself through (otherwise a brand-new user could never pass it to get created) but otherwise fails closed on real errors, prompts `ErrUserNotFound` users to `/start`, and tells banned users they're banned.

Reply-keyboard text handlers: `/admin`, `texts.{Help,Catalog,Profile,StartMenu,Purchases,RefillBalance,ProfileRefresh,Replenishments,Referral}Btn` (`ProfileRefreshBtn` re-sends the profile card via `UserService.RefreshProfile`, which reads Postgres directly instead of the user cache — purchase stats are already always read straight from Postgres), and `/start` — registered with `MatchTypeCommandStartOnly` rather than `MatchTypeExact` specifically so a ref-link's `/start <id>` payload still matches (see the referral paragraph above); everything else here still uses `MatchTypeExact`. Callback-query (inline button) prefixes, via `bot.MatchTypePrefix`: `product_`, `buy_`, `buyqty_`, `purchase_` (purchase-batch UUID, not a raw row ID), `purchasespage_`, `category_`, `refillmerchant_` (bare merchant string, own Build/Parse pair — not numeric, doesn't fit `ParseCallbackQuery`), `replenishmentspage_`; exact-match: `buycancel`, `buyconfirm`, `catalog_root`, `main_menu`, `referral_close`. `bot/utils.ParseCallbackQuery` parses the trailing numeric segment; `ParseBatchCallbackQuery` parses the trailing UUID (works because UUIDs use hyphens, not underscores, so splitting on `_` and taking the last part is still safe).

`Handlers.botUsername` (fetched once via `b.GetMe(ctx)` in `bot.New`, after the `bot.Bot` exists but before handlers are built) is what `ReferralHandler` uses to build `t.me/<botUsername>?start=<telegramID>` links — there's no config env var for it, since the token already implies it and asking the API once at startup avoids the two ever drifting.

The buy flow is stateful, backed by `internal/cache/redis.Cache` (which doubles as `domain/fsm.Store`): tap "buy" -> edit the product card into a quantity prompt (quick-pick 1–5 inline or type a number, both converge on the same confirmation step) -> stock is checked before showing confirmation (fast-fail if you ask for more than exists, rather than only finding out after confirming) -> confirmation screen -> only `BuyConfirmHandler` actually charges anything and marks items sold.

### Web admin panel

CRUDL for categories/products, view+edit for users (ban/unban, balance, promote/demote admin, plus `POST /api/users/:telegram_id/referrals/{enable,disable}` -> `AdminService.SetReferralsEnabled`), view for purchases (cross-user — everything on `PurchaseService`/`PurchaseRepository` elsewhere is scoped to one Telegram user, so `ListAllAdmin`/`CountAllAdmin`/`GetAdminByID` exist purely for this screen), a cross-user replenishments view (same `ListAllAdmin`/`CountAllAdmin` pattern on `ReplenishmentService` — this is also where `Merchant: referral` credits are visible across all users, not just each referrer's own "Мои пополнения"), an admin audit-log view, a Statistics screen backed by `StatsService`/`StatsRepository` (plain SQL aggregates, no Prometheus/Grafana), and settings (Support username, per-merchant credentials/`Enabled`/min-max, and the referral `Enabled`+`Percent` pair, via `GET`/`PUT /api/settings` — **frontend page not built yet**, only the backend API).

Auth is code-then-session, entirely Redis-backed, nothing in Postgres: `Handlers.AdminHandler` (`/admin`) calls `AdminAuthService.IssueLoginCode`, which generates a 6-digit code (`admintoken.GenerateCode`), stores `sha256(code) -> telegramID` in Redis for 30 seconds (`domain/adminsession.Store`, implemented by the same `internal/cache/redis.Cache` struct as the read-through cache and FSM state, separate keyspace), and sends it back in the `/admin` reply. The login page's `POST /api/auth/exchange` (the *one* unauthenticated route — registered directly on the top-level gin engine, outside the `/api` route group, so it never passes through `Auth`) calls `AdminAuthService.ExchangeLoginCode`: consumes the code atomically (Redis `GETDEL`, so it's single-use even under concurrent attempts), re-checks `IsAdmin()` (a demote between issuance and exchange must not slip through), and issues a 24h HS256 JWT session token (`admintoken.GenerateSessionJWT`, keyed by `ADMIN_JWT_SECRET`) whose hash is *also* stored in Redis the same way the login code was.

`backend/middleware.Auth` resolves the incoming `Authorization: Bearer <session token>` header via `AdminAuthService.ValidateSession`, which requires all three of: the JWT signature/expiry to check out (`admintoken.ParseSessionJWT`, rejects a forged or expired token before any I/O), the token's hash to still have a live Redis entry (this is what makes the token revocable at all — a signed JWT can't be un-issued, so `Logout` works by deleting this entry instead), and the resolved user to still satisfy `IsAdmin()` in Postgres. `GET /api/auth/me`'s 200-vs-401 response *is* the login check from the frontend's perspective. `POST /api/auth/logout` deletes the Redis session entry early, which is what actually revokes the JWT before its natural expiry.

`AdminService.MakeAdmin`/`RevokeAdmin`/`BanUser`/`UnbanUser` only ever change `User.Role` — they hand back no credential; a newly promoted admin gets in by sending `/admin` themselves. `MakeAdmin` is root-admin-only (`ErrOnlyRootAdminCanPromote` otherwise, checked against the acting admin's own persisted role) — without that, any admin could promote arbitrary users, who could promote further admins, with nothing containing the spread. Because `Role` is a single field, `BanUser`/`RevokeAdmin` both refuse to target the root admin (`ErrCannotBanRootAdmin`/`ErrCannotRevokeRootAdmin`) or the acting admin's own account (`ErrCannotBanSelf`/`ErrCannotRevokeSelf`) — banning or revoking would otherwise overwrite/strip root status with no one left able to grant it back. `UnbanUser` always restores plain `user`, never whatever role the target held before being banned — a banned former admin needs `MakeAdmin` run again after unban. A demoted admin's still-live session keeps validating JWT-wise until its TTL, but `ValidateSession` re-checks the role against Postgres on every call, so it's rejected on their very next request regardless.

`AdminService.DeleteCategory`/`DeleteProduct` refuse to delete a non-empty category or a product with purchase history (`ErrCategoryNotEmpty`/`ErrProductHasPurchases`) rather than surfacing a raw FK error — reassign/delete children first, or deactivate (`IsActive`) instead of deleting a product that's already sold.

Telegram rejects inline URL buttons pointing at `localhost`/loopback hosts outright (`Wrong HTTP URL`) — a near-certainty in local dev, where `ADMIN_PANEL_FRONTEND_URL` defaults to `http://localhost:3000`. `Handlers.AdminHandler` sends the login code with `kb.AdminKb` (the button) first; if that specifically fails with `bot.ErrorBadRequest`, it retries with a plain-text message carrying the panel URL as a link in the text instead of a button, so the code itself is never blocked by an unclickable link. In production, point `ADMIN_PANEL_FRONTEND_URL` at a real `https://` domain to get the clickable button.

### Caching gotcha worth knowing

`cmd/migrate` (and any direct Postgres edit) has zero awareness of the Redis read-through cache by design — it's a Postgres-only step. In the normal deploy flow this is fine (`migrate` completes before `bot`/`backend` even start, so there's nothing cached yet). But re-running `migrate` (or hand-editing a row) against an *already-running* bot/backend can leave a stale cached `User` (10-minute TTL, key `user:<telegram_id>`) — e.g. promoting someone to `root_admin` in Postgres doesn't retroactively fix a session that already cached their old role. Fix: `redis-cli DEL user:<telegram_id>`, or wait out the TTL.
