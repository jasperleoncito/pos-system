# POS System — Project Instructions

Multi-tenant restaurant POS SaaS (PRD in `first-prompt.md`). Built in phases — **see `docs/PLAN.md` for the full phase plan and current progress**. ALL phases 0–14 are DONE — the full PRD is built. Production deploy: `docker-compose.prod.yml` (certs in `deploy/certs/`), backups via `make backup`, black-box tests via `make test-integration` (stack must be up + seeded).

## Golden rules

- **Never use browser `alert()`/`confirm()`/`prompt()`** — always shadcn `Dialog`/`AlertDialog` + `sonner` toasts.
- **All money is BIGINT centavos** end to end. Format with `frontend/src/lib/currency.ts` (`formatCentavos`, `pesosToCentavos`). Currency is PHP.
- **Prices are never trusted from the client.** Orders send product/variant/modifier IDs only; the backend re-prices from the catalog and snapshots names/prices.
- **Tenant isolation:** tenant_id comes ONLY from the JWT claim (`tid`), enforced by middleware; every repo query filters `tenant_id = $1 AND deleted_at IS NULL`.
- **Touch-first, fully responsive** (phone / tablet / desktop): 44px+ targets, bottom-sheet cart on mobile, test at 375 / 768 / 1024 / 1440.
- **All ports are random 4-digit numbers, never defaults** (user requirement): web `7642`, backend `9137` (internal), frontend `4519` (internal), Mailpit UI `9284`, MinIO console `9673`. New services get another random 4-digit port.
- Conventional commits (`feat:`, `fix:`, …), one commit per phase, no AI attribution lines.
- After backend route changes: regenerate Swagger (`make swag` or `go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g cmd/api/main.go -o docs` in `backend/`).

## Stack & architecture

- **Backend** `backend/`: Go 1.25 + Gin, clean architecture — `handler/v1` (validate → delegate) → `service` (business logic) → `repository/postgres` (pgx, hand-written SQL). Domain entities + interfaces in `internal/domain/<module>`. Response envelope `{success, message, data}` / `{success, message, errors}` via `pkg/response`; errors via `pkg/apperror` kinds.
- **Frontend** `frontend/`: Next.js 15 App Router + TS + Tailwind v4 + shadcn/ui (new-york) + React Query + RHF/Zod + `motion`. Tenant pages under `src/app/(dashboard)/[tenant]/…`, guarded client-side in the layout. Axios client with single-flight 401 refresh in `src/lib/api.ts`.
- **Auth:** JWT access (15m) + rotating opaque refresh tokens hashed in `device_sessions`. RBAC matrix in `backend/internal/domain/rbac/rbac.go`, mirrored for nav in `frontend/src/lib/rbac.ts` (API is the enforcement point). Roles: owner, manager, cashier, kitchen, employee (+ platform super admin).
- **Images:** pure-Go WebP pipeline (`gen2brain/webp`, WASM — NOT libvips/CGO, host builds must work on Windows) in `pkg/imageproc`: ≤10MB PNG/JPG/WEBP → 1600px WebP q80 + 300px thumb (+ favicons for logos), stored in MinIO `{tenant}/logos|products|…`, served via nginx `/storage/`. Originals never stored.
- **Tenant theming:** brand colors → CSS variables via `frontend/src/lib/theme.ts`, injected in the tenant layout with contrast-aware foregrounds. GET `/tenant/settings` is readable by every member so all roles get the branded UI.
- **Migrations:** `backend/migrations/` (golang-migrate), auto-run at API startup. Every tenant table: `id UUID, tenant_id, created_at, updated_at, deleted_at` + partial indexes `WHERE deleted_at IS NULL`.
- **Inclusive tax math:** `tax = amount × rate / (100 + rate)`, half-up rounding (`internal/service/order_totals.go`).
- Redis: OTP tokens, rate limiting, (later: cache, queues via asynq, SSE pub/sub). Kitchen SSE route `/api/v1/kitchen/stream` already has `proxy_buffering off` in nginx.

## Dev workflow

- `docker compose up -d --build` — full stack (Docker Desktop must be running). Backend hot-reloads via air; the frontend runs the PRODUCTION build by default (instant navigation). Only while editing UI code, switch to hot reload with `make frontend-dev` (uses `docker-compose.frontend-dev.yml`), then back with `make frontend-fast`. Prod deploy = `docker-compose.prod.yml`: plain-HTTP nginx on WEB_PORT (default 7642) — an external reverse proxy (e.g. Nginx Proxy Manager) terminates TLS in front of it; the bundled nginx stays because it routes /api→Go, /storage→MinIO, /→Next on one origin with SSE buffering off.
- App: http://localhost:7642 · Swagger: http://localhost:7642/api/v1/docs/index.html · Mailpit: http://localhost:9284 · MinIO console: http://localhost:9673
- Seed (idempotent): `docker compose exec backend go run ./cmd/seed`
- Tests: `make test` (runs `go test -race` inside the container — the Windows host has no gcc, so run plain `go test ./...` when testing on the host).
- Frontend checks on host: `npx tsc --noEmit` and `npm run lint` in `frontend/`.
- `.env` is gitignored; `.env.example` documents everything.

## Demo accounts (all password `password123`)

`superadmin@pos.local` (platform admin), and for tenant **Teresa's Eatery** (`teresas-eatery`): `owner@`, `manager@`, `cashier@`, `kitchen@`, `employee@teresas.ph`. Menu is seeded from the photos in `img-menu/` (7 categories, 47 products, modifier groups, 12% inclusive VAT).

## Gotchas

- `shadcn` CLI init once failed to patch `globals.css` and skipped utility deps — after `npx shadcn add …`, verify the component files exist and imports resolve.
- Backend go.mod pins Go 1.26.1 (bumped by excelize/maroto in Phase 12) → Docker base image is `golang:1.26-bookworm`; keep them in sync. The container sets GOTOOLCHAIN=local, so a go.mod toolchain bump breaks air until the base image matches.
- Air watches `.go` and `.sql`; writing a migration triggers a backend restart which auto-applies it.
- Receipt printing: `#receipt-print` is the only element visible during `window.print()` (80mm CSS in `globals.css`).
