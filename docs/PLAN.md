# Build Plan — Multi-Tenant Restaurant POS

Full-PRD build (`first-prompt.md`) in sequential phases. The system must stay runnable after every phase; each phase ends with browser/API verification and one conventional commit.

**Status: ALL PHASES 0–14 DONE ✅ — the full PRD is built. Future work = new features/maintenance.**

## Requirements beyond the PRD (user decisions)

- POS works on any device: phone / tablet / iPad / desktop, touch-first (44px+ targets).
- Never use browser `alert()`/`confirm()` — shadcn Dialog/AlertDialog + sonner toasts only.
- All ports are random 4-digit numbers (web 7642, api 9137 internal, frontend 4519 internal, Mailpit 9284, MinIO console 9673).
- Demo tenant Teresa's Eatery seeded with the real menu from `img-menu/` photos, prices in PHP centavos.
- Design direction via ui-ux-pro-max skill; tenant brand colors re-theme the whole UI via CSS variables.

## Phase log

### ✅ Phase 0 — Infrastructure (commit `15d8dcd`)
Docker Compose stack (postgres 16, redis 7, minio+init, backend air hot-reload, worker, frontend, mailpit, nginx), Go/Gin skeleton with `/api/v1/health`, envelope + apperror packages, golang-migrate auto-run, swaggo; Next.js 15 + Tailwind v4 + shadcn + React Query + next-themes + sonner; Makefile, .env.example.

### ✅ Phase 1 — Auth, tenancy, RBAC (commit `0738683`)
JWT access (15m) + rotating refresh (reuse rejected), device sessions, forgot/reset + email verification (Redis OTP + Mailpit), switch-tenant, RBAC matrix + RequirePermission, per-IP rate limits, audit logs, idempotent seeder (super admin + per-role users). Frontend: auth pages, 401 refresh interceptor, role-aware sidebar, tenant switcher, devices page.

### ✅ Phase 2 — Branding & image pipeline (commit `d62c622`)
Pure-Go WebP pipeline (gen2brain/webp WASM; resize 1600px, q80, thumbs, favicon set, metadata stripped), MinIO per-tenant keys served via nginx `/storage/`, tenant settings CRUD + logo upload (old generations cleaned), live tenant theming via CSS variables (contrast-aware), super-admin `/admin/tenants` suspend/activate. Verified: 8MB PNG → optimized WebP set; 18MB → 413 at nginx.

### ✅ Phase 3 — Menu catalog + Teresa's seed (commit `f6d11f8`)
Catalog schema (categories, taxes, products, variants, modifier_groups/modifiers, product_modifier_groups), CRUD with batched child loading (no N+1), product images via pipeline. Seed: 7 categories, 47 products with exact photo prices, modifier groups (Choice of Side Dish / Drink / Dip), Mismo + RC 1L variants, 12% inclusive VAT. Menu UI: Products/Categories/Modifiers/Taxes tabs.

### ✅ Phase 4 — POS core (commit `33b6868`)
Orders (per-tenant numbers via order_counters upsert), server-side pricing + snapshots, required-modifier enforcement, inclusive/exclusive tax (half-up, unit tested), hold/resume, mixed payments (change from cash only; non-cash ≤ due; cash requires open drawer), drawer sessions (one open per tenant) + signed cash-movement ledger + close variance, receipt endpoint with branding. Touch-first POS terminal: category chips, grid, options dialog, side-panel/bottom-sheet cart, payment dialog (quick cash, Exact, split payments), held orders, 80mm print receipt. Verified E2E incl. ₱184 sale paid ₱100 cash + ₱84 GCash, VAT ₱19.71.

### ✅ Phase 5 — Discounts, coupons, splits, refunds, voids (commit `db9e52c`)
Discounts + coupons (atomic max_uses redemption, released on void), order-level promo at creation, split bills (amounts sum to total, per-split payments, completes when all paid), refunds (manager+, capped at remaining, drawer -amount), voids (manager+, reason, net cash returned). Frontend: /promos page, POS Promo + Split-bill dialogs, /orders history with role-gated Refund/Void. Verified: max_uses=1 reuse rejected; 2-way split cash+GCash; ₱50 partial refund moved drawer; cashier refund/void 403.

### ✅ Phase 6 — Kitchen Display (realtime)
`realtime.Hub`: per-tenant SSE subscribers bridged over Redis pub/sub `kitchen:{tenant}` (multi-replica safe, slow consumers dropped not blocked). `GET /kitchen/stream` authenticates via `?token=` (EventSource can't send headers) + 25s heartbeat. Orders fire events on create/resume/settle-from-hold; kitchen_status + item_status + priority all publish. KDS board: New/Preparing/Ready columns, elapsed-time color badges (5m amber/10m red), per-item done toggle, rush badge, Web Audio chime, 10s polling fallback, SSE auto-reconnect with fresh token. Verified: SSE delivered order_fired instantly on cashier order; transitions published; kitchen role 403 on catalog writes.

---

## Remaining phases

### ✅ Phase 7 — Inventory core
Units/items/recipe_items (single-table BOM keyed by product_id)/inventory_movements. `InventoryRepo.Apply` = SELECT FOR UPDATE row lock + ledger insert with qty_before/after in one tx (no Redis lock needed). `DeductForOrder` idempotent via HasMovements(order, sale) — runs on Pay + PaySplit completion; failures logged, never block the sale. Routes under inventory:read/write; recipes GET/PUT /products/:id/recipe. Seed: kg/pcs/L units, 7 items, recipes for Katsudon/Pork Tapa/C2 Solo. UI: /inventory page (stock badges OK/Low/Out, move dialog stock_in/out/adjustment/waste with required reason, per-item history), RecipeDialog on products panel (ChefHat button). Verified: 2× Katsudon → Rice -0.4/Pork -0.3/Egg -2/Oil -0.1 with exact ledger chain; settle retry 422 no double-deduct; two concurrent payments deducted exactly once each.

### ✅ Phase 8 — Suppliers, purchase orders, low-stock alerts
Suppliers CRUD, PO lifecycle draft→ordered→partially_received→received (per-line receive → po_receive ledger movements + item cost update), cancel. stock_alerts raised by InventoryService.checkAlert after every movement (AlertSink interface → ProcureRepo; one open alert per item via partial unique index; out_of_stock vs low_stock). Routes under inventory:read/write. UI: /inventory/procurement page (PO + Suppliers tabs, receive dialog defaults to remaining), alerts banner with Acknowledge on /inventory. Verified: 10kg Rice PO received 6+4 with status transitions, stock 49→59, cost → ₱62; stock_out below reorder raised low_stock alert; ack cleared it.

### ✅ Phase 9 — Employees, schedules, attendance
employees (optional unique user link per tenant, salary hourly/daily/monthly BIGINT centavos, photo via WebP pipeline)/employee_schedules (weekly template, day_of_week 0–6, TIME columns, grace 0–240m)/attendance_records (schedule snapshot at clock-in, one open shift per employee via partial unique index). Clock math in EmployeeService uses server time in the tenant's timezone (`tenantNow`): late = clock_in − (start+grace), early-out/overtime vs snapshot end, breaks accumulate via break_start; approval only for completed pending records (manager+, attendance:approve). Self-service routes need only a linked user (attendance:clock, every role). Dialog-portal theming fixed: brand CSS vars now mirrored onto `<html>` so Radix portals stay branded. Seed: 5 staff (4 linked to role accounts) with Mon–Sat 09:00–17:00 grace 10. UI: /employees directory (avatars, salary, schedule grid dialog, photo upload), /attendance big-button clock (live time, on-shift/break badges) + filterable review table with Approve. Verified E2E: clock-in Manila TZ snapshot 09:00+08:00, double clock-in 409, break accumulation, early-out 783m, employee role 403 on /employees + /attendance, double-approve 422, browser flow incl. 375px.

### ✅ Phase 10 — Customers & loyalty
customers (points_balance + lifetime_points, tier, unique phone per tenant)/loyalty_settings (earn_rate=centavos per point, redeem_value=centavos per point, tier thresholds+multipliers, defaults served when unsaved)/loyalty_transactions (signed points, balance_after; ApplyPoints = guarded UPDATE rejecting overdrafts + ledger insert in one tx; lifetime grows on earn, shrinks only when an earn is reversed). Points redemption = payment method `points` (requires attached customer; redeemed BEFORE payments book; counts as non-cash so ≤ due). Earn on completion (Pay + PaySplit): floor((total − points_value)/earn_rate × tier multiplier); auto tier upgrade (never downgrades). Void → ReverseForOrder (earn and redeem both reversed via adjust rows). Orders: customer_id at creation, ?customer_id= list filter. Loyalty settings RBAC = catalog:write (manager+); customers under customers:read/write. Seed: 3 demo customers. UI: /customers (tier badges, profile dialog w/ ledger + purchases tabs, loyalty program dialog), POS attach-customer (search/quick-create/detach in cart panel), Points method in payment dialog capped at balance value. Verified: ₱200 → 4pts; partial redemption excluded from earn base; overdraft + no-customer 422; void reversal exact; kitchen 403; cashier settings 403.

### ✅ Phase 11 — Sales analytics dashboard
expenses table (category/amount/expense_date). AnalyticsRepo aggregates over sale statuses (completed/partially_refunded/refunded; refunds subtract via refunds table): summary (gross/net/AOV/refunds/expenses/COGS/profit; COGS = 'sale' inventory movements × unit_cost fallback item cost), top products/categories/employees, hourly (dense 24 buckets, tenant TZ via AT TIME ZONE), day×hour heatmap, payment mix (cash bucket minus change). Endpoints: GET /analytics/overview (today/WTD-Mon-start/MTD/YTD each vs previous period) + /analytics/dashboard?from&to (one bundled payload) + expenses CRUD — all under analytics:read (manager+). Redis JSON cache (redisrepo.Cache) TTL 3min, key prefix analytics:{tenant}:; OrderService.SetAnalytics(SalesCacheInvalidator) busts on completion/split completion/refund/void/expense writes. UI (dataviz rules): stat cards w/ trend deltas, preset+custom range picker, summary strip, Recharts hourly bars (--chart-1) + payment donut (fixed Okabe-Ito hue per method), CSS-grid heatmap (sequential brand-hue opacity), top lists w/ proportional bars, expenses card w/ add/delete. recharts added to frontend. Verified: summary math incl. refund ₱50 + COGS ₱370.75; expense create instantly reflected (cache invalidated); cashier 403; charts render at 1440.

### ✅ Phase 12 — Reporting & exports
pkg/export: generic Document{Title,Subtitle,Columns(kind text|money|number),Rows,Totals,LogoPNG} + Exporter interface with CSV (stdlib), XLSX (excelize, numeric peso cells w/ #,##0.00), PDF (maroto/v2, tenant favicon-180 PNG header via new storage.Get; ₱ rendered as "P" — core fonts lack the glyph). ReportRepo = generic queryRows (pgx FieldDescriptions → map rows; values cast in SQL); 7 reports: sales, inventory (stock value), employees, attendance (worked minutes computed), profit (per-day CTE merge of sales/refunds/COGS/expenses via generate_series), tax (per-day net-of-tax + collected), receipts (methods string_agg). GET /reports + /reports/:type?from&to&format=json|csv|xlsx|pdf under reports:read; totals footer summed server-side. UI: /reports center — type chips, range, preview table with money formatting + TOTAL row, CSV/Excel/PDF download buttons (blob). NOTE: excelize/maroto bumped go.mod to go 1.26.1 → backend Dockerfile now golang:1.26-bookworm (keep in sync!). Verified: all 7 types JSON; CSV/XLSX/PDF bytes; PDF visually correct with totals matching dashboard (profit −₱1.75).

### ✅ Phase 13 — Notifications & background jobs
pkg/queue: asynq client + typed tasks (email:send, notify:low_stock, notify:attendance, notify:daily_summary); queue.Client implements the EmailSender contract so ALL transactional mail (verify/reset) now enqueues — cmd/worker delivers via SMTP. internal/worker.Handlers: fan-out to owner+manager members (in-app rows via batched insert + emails honoring per-user notification_prefs); asynq.Scheduler registers one daily-summary cron per tenant at 21:00 in the TENANT's timezone via "CRON_TZ={tz} 0 21 * * *" specs. Hooks: InventoryService.checkAlert enqueues exactly once per newly opened alert (EnsureAlert now returns created bool); EmployeeService.ClockIn enqueues when late_minutes > 0; services get queue via SetJobs (Jobs interface). notifications + notification_prefs tables (unread partial index; prefs default-true). Routes (any member): GET /notifications (items+unread), POST /:id/read, /read-all, GET/PUT /preferences. UI: topbar bell w/ unread badge + dropdown (30s poll), /notifications page w/ email-preference switches. Worker compose runs `go run ./cmd/worker` (restart to pick up changes). Verified: stock_out → alert → in-app + emails to owner+manager in Mailpit; 86m-late clock-in alert; queued password-reset delivered; read/read-all counts; prefs round-trip; crons registered per tenant.

### ✅ Phase 14 — Hardening & production readiness
Audit viewer: GET /audit-logs (audit:read = owner) w/ joined user names; UI at /settings/audit (paginated, action badges, change summaries) linked from Settings. Super-admin: tenants.plan column (free|standard|premium, migration 000012) + PATCH /admin/tenants/:id/plan + GET /admin/stats (tenants/users/orders-30d/GMV-30d) + stats cards & plan select on /admin/tenants. Security: middleware.SecurityHeaders (nosniff, DENY, referrer-policy, permissions-policy, HSTS in prod) + global per-IP rate limit 300/min (auth keeps its tighter 20/min). Integration suite backend/tests (build tag `integration`, black-box vs running stack; `make test-integration`): refresh rotation + replay rejection, RBAC denials, tenant isolation via freshly registered tenant, full order flow w/ exact Rice 0.2 deduction + settle idempotency, security headers — all pass. docker-compose.prod.yml: prod image targets (distroless api/worker, standalone Next), TLS nginx (nginx.prod.conf, 80→443 redirect, HTTP/2, static caching, certs from deploy/certs — see its README), healthchecks + restart unless-stopped, no dev ports/mailpit. scripts/backup.sh (`make backup`): pg_dump custom format + MinIO volume tar via throwaway alpine (MSYS-safe); restore notes inline; backups/ gitignored. Responsive sweep at 375 on dashboard/reports/audit — tables scroll in-card, no page overflow.

---

## Architecture quick reference

DB schema per module, key decisions, and verification recipes are recorded in the phase log above and in `CLAUDE.md`. The original full design (schema column detail per module) also lives in the PRD `first-prompt.md` §Database plus these decisions:

- Shared-schema tenancy; JWT claims `{sub, tid, role, sid, is_super, typ}`.
- KDS realtime = SSE + Redis pub/sub (not WebSocket); polling fallback.
- Queue = hibiken/asynq, same binary via `cmd/worker`.
- Exports = excelize / csv / maroto v2.
- Money = BIGINT centavos; inclusive tax `amount×rate/(100+rate)` half-up.
