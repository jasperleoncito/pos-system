# C5 — Subscription Billing (Xendit) ✅ DONE

Owners pay monthly (₱800 default) or yearly (₱8,000 default) via Xendit invoices. Unpaid → tenant inactive (owner sees pay modal, staff see blocked screen). Super admin tracks everything and can override. Prices editable by super admin. See `docs/PLAN.md` for the phase log; this file tracks the C5 build.

**Status: shipped.** All checklist items done; verified E2E (13-section API script + browser check of register plan cards and owner pay-modal). Real Xendit staging invoice created; webhook idempotency, month-end clamping (Jan 31 → Feb 28), sweep notice + deactivation, and admin overrides all confirmed.

## Design keystones

- Money BIGINT centavos; Xendit `amount` = centavos/100 (PHP).
- **All period math in SQL** (Postgres intervals clamp Jan 31 + 1mo → Feb 28; Go AddDate overflows). Extension = one atomic UPDATE: `current_period_end = GREATEST(now(), current_period_end) + interval '1 month'|'1 year'`, `status='active'`, `due_notice_sent_at=NULL`.
- Webhook idempotency: `UPDATE subscription_payments ... WHERE id=$1 AND status='pending'` — 0 rows = duplicate → 200 no-op. `PAID` and `SETTLED` both count as paid. Unknown external_id → 200 + warn. `x-callback-token` verified (constant time).
- Enforcement: `middleware.RequireActiveSubscription` → **402**, super-admin bypass, fail-open on errors. Status cached in Redis 60s (`billing:sub:{tenant}`), busted from API on changes; worker sweep tolerates the stale window.
- Checkout reuses a <24h pending payment for the same plan; prices always read server-side from `platform_settings`.
- Register creates the **pending** subscription only (no Xendit call inline); frontend follows up with POST /billing/checkout → redirect to invoice_url.
- `tenants.status` (suspend) stays a separate super-admin mechanism; `tenants.plan` column stays dormant.

## Checklist

### Backend
- [x] 1. Config: XenditConfig (SecretKey, WebhookToken), prod-only validate; .env.example entries
- [x] 2. Migration 000013_billing: platform_settings (singleton, 80000/800000), subscriptions (unique tenant_id, plan monthly|yearly, status pending|active|inactive, period start/end, due_notice_sent_at), subscription_payments ledger (external_id unique, method xendit|manual); backfill existing tenants active-monthly +30d
- [x] 3. domain/billing + notification.TypeBilling
- [x] 4. pkg/xendit Client.CreateInvoice (POST /v2/invoices, basic auth, 10s timeout)
- [x] 5. repository/postgres/billing_repo.go (conditional updates, GREATEST extension, due-notice/overdue queries, owner joins)
- [x] 6. service/billing_service.go (checkout, webhook, IsActive w/ Redis cache, CreateInitialSubscription, admin ops, audits)
- [x] 7. middleware/subscription.go (402, bypass, fail-open)
- [x] 8. rbac billing:manage (owner) + frontend mirror
- [x] 9. dto/billing.go + handler/v1/billing.go (+ Swagger)
- [x] 10. router wiring: public plans + webhook; member subscription; owner checkout/payments; admin subscriptions/owners/settings/mark-paid/status; requireActive appended to team/catalog/order/kitchen/inv/emp/cust/analytics/reports groups + audit-logs + tenant settings writes (NOT notifications, NOT GET /tenant/settings)
- [x] 11. Register gains plan (pending sub); AdminCreateBusiness → active monthly +30d; seeder ensures Teresa's sub
- [x] 12. Worker: hourly billing:sweep (3-day notices owner-only in-app+email; overdue → inactive + notify)

### Frontend
- [x] 13. types/billing.ts + hooks/use-billing.ts + rbac.ts
- [x] 14. Register page: plan cards → checkout redirect
- [x] 15. /billing/return polling page
- [x] 16. Dashboard gate: PlanPayModal (owner), BlockedScreen (staff, 30s refetch), DueBanner (≤3 days); Settings → Billing page (plan, due date, history, pay/switch)
- [x] 17. Admin: nav (Tenants | Billing), /admin/billing (stats, subscriptions+owners tables, mark-paid dialog, deactivate/reactivate, prices editor)

### Finalization
- [x] 18. Unit tests (checkout reuse, webhook idempotency, notice dedupe) + swagger regen + tsc/lint
- [x] 19. E2E verify (register→402→webhook→active; sweep notice/deactivate; admin overrides) + PLAN.md/CLAUDE.md updates + commit

## Verification recipes

- Simulate webhook: `POST /api/v1/webhooks/xendit` w/ header `x-callback-token: $XENDIT_WEBHOOK_TOKEN`, body `{"external_id":"<payment-uuid>","status":"PAID","paid_amount":800,...}`; re-POST must not double-extend.
- Sweep test: psql `UPDATE subscriptions SET current_period_end = now() + interval '2 days'` then trigger sweep → notice; set past → inactive → 402 within 60s.
- Local email checks need the Mailpit SMTP swap (local .env points at real Gmail — see memory note).
- Local dev cannot receive real Xendit callbacks; on the VPS set the webhook URL in the Xendit dashboard to `https://<domain>/api/v1/webhooks/xendit` with the verification token.
