# Build Plan — Multi-Tenant Restaurant POS

Full-PRD build (`first-prompt.md`) in sequential phases. The system must stay runnable after every phase; each phase ends with browser/API verification and one conventional commit.

**Status: Phases 0–8 DONE ✅ · Continue from Phase 9 (Employees, schedules, attendance).**

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

### ⬜ Phase 9 — Employees, schedules, attendance
Employees (optional user link, salary type/rate, photo via pipeline), weekly schedules with grace minutes, clock in/out (server time) computing late/early-out/overtime/breaks, manager approval, attendance reports. UI: directory + profile, schedule grid, big-button self-service clock page, review + report table.

### ⬜ Phase 10 — Customers & loyalty
Customers (points_balance, tier, birthday, purchase history), membership tiers (Silver/Gold/VIP) with multipliers, loyalty_transactions ledger (balance_after), loyalty settings (earn rate, redemption value); earn on completion, redeem at POS, auto tier upgrade. UI: customer table/profile, attach-customer at POS, redeem points in payment dialog.

### ⬜ Phase 11 — Sales analytics dashboard
Today/WTD/MTD/YTD sales, revenue/profit (− recipe COGS − expenses)/expenses, AOV, top products/categories/employees, hourly sales, day×hour heatmap, payment mix; Redis cache 2–5min TTL invalidated on completion; expenses CRUD. UI: stat cards with deltas, Recharts line/bar/donut, heatmap, date-range picker (use dataviz + ui-ux-pro-max guidance).

### ⬜ Phase 12 — Reporting & exports
Report endpoints (sales, inventory, employees, attendance, profit, tax, receipts reprint) each with `?format=json|csv|xlsx|pdf` behind one Exporter interface — excelize (XLSX), stdlib CSV, maroto/v2 (PDF with tenant logo header). UI: Reports center with filters + preview + export downloads.

### ⬜ Phase 13 — Notifications & background jobs
Wire asynq in the worker container: email templates (verify, reset, low-stock, daily summary, attendance alerts) moved onto the queue; asynq scheduler crons per tenant timezone; in-app notifications table + unread endpoint + preferences. UI: bell dropdown with unread count, notifications page.

### ⬜ Phase 14 — Hardening & production readiness
Audit coverage sweep on all mutating routes + audit log viewer; super-admin system analytics + subscriptions; security pass (headers, strict CORS, global rate limits, validation audit); integration test suite (auth, tenant isolation, order flow, inventory deduction); `docker-compose.prod.yml` (built images, TLS-ready nginx, healthchecks, restart policies); Postgres/MinIO backup script; responsive + a11y sweep at 375/768/1024/1440.

---

## Architecture quick reference

DB schema per module, key decisions, and verification recipes are recorded in the phase log above and in `CLAUDE.md`. The original full design (schema column detail per module) also lives in the PRD `first-prompt.md` §Database plus these decisions:

- Shared-schema tenancy; JWT claims `{sub, tid, role, sid, is_super, typ}`.
- KDS realtime = SSE + Redis pub/sub (not WebSocket); polling fallback.
- Queue = hibiken/asynq, same binary via `cmd/worker`.
- Exports = excelize / csv / maroto v2.
- Money = BIGINT centavos; inclusive tax `amount×rate/(100+rate)` half-up.
