package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/billing"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

// BillingRepo persists subscriptions, the payment ledger, and the
// platform price sheet. All period arithmetic happens in SQL so
// Postgres interval clamping applies (Jan 31 + 1 month = Feb 28).
type BillingRepo struct {
	db *pgxpool.Pool
}

func NewBillingRepo(db *pgxpool.Pool) *BillingRepo { return &BillingRepo{db: db} }

const subscriptionColumns = `id, tenant_id, plan, status, current_period_start, current_period_end, due_notice_sent_at, created_at, updated_at`

func scanSubscription(row pgx.Row) (*billing.Subscription, error) {
	var s billing.Subscription
	err := row.Scan(&s.ID, &s.TenantID, &s.Plan, &s.Status,
		&s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.DueNoticeSentAt, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("subscription")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan subscription: %w", err)
	}
	return &s, nil
}

func (r *BillingRepo) CreateSubscription(ctx context.Context, s *billing.Subscription) error {
	// Idempotent: seeder and backfill may race with registration.
	_, err := r.db.Exec(ctx, `
		INSERT INTO subscriptions (tenant_id, plan, status, current_period_start, current_period_end)
		VALUES ($1, $2, $3, now(), CASE WHEN $3 = 'active' THEN now() + interval '30 days' ELSE now() END)
		ON CONFLICT (tenant_id) DO NOTHING`,
		s.TenantID, s.Plan, s.Status)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}
	created, err := r.GetByTenant(ctx, s.TenantID)
	if err != nil {
		return err
	}
	*s = *created
	return nil
}

func (r *BillingRepo) GetByTenant(ctx context.Context, tenantID string) (*billing.Subscription, error) {
	return scanSubscription(r.db.QueryRow(ctx,
		`SELECT `+subscriptionColumns+` FROM subscriptions WHERE tenant_id = $1`, tenantID))
}

func (r *BillingRepo) SetStatus(ctx context.Context, tenantID, status string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE subscriptions SET status = $2, updated_at = now() WHERE tenant_id = $1`,
		tenantID, status)
	if err != nil {
		return fmt.Errorf("failed to set subscription status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("subscription")
	}
	return nil
}

// Extend activates the subscription and pushes the due date out one plan
// interval from GREATEST(now, current period end) — a single atomic
// statement, so webhook and sweep can never interleave badly.
func (r *BillingRepo) Extend(ctx context.Context, tenantID, plan string) (*billing.Subscription, error) {
	return scanSubscription(r.db.QueryRow(ctx, `
		UPDATE subscriptions SET
			plan = $2,
			status = 'active',
			current_period_start = now(),
			current_period_end = GREATEST(now(), current_period_end) +
				CASE WHEN $2 = 'monthly' THEN interval '1 month' ELSE interval '1 year' END,
			due_notice_sent_at = NULL,
			updated_at = now()
		WHERE tenant_id = $1
		RETURNING `+subscriptionColumns, tenantID, plan))
}

// ExtendMonths activates the subscription and pushes the due date out by
// N calendar months from GREATEST(now, current period end) — a super-admin
// comp/grant. Single atomic statement; all date math in SQL.
func (r *BillingRepo) ExtendMonths(ctx context.Context, tenantID string, months int) (*billing.Subscription, error) {
	return scanSubscription(r.db.QueryRow(ctx, `
		UPDATE subscriptions SET
			status = 'active',
			current_period_start = now(),
			current_period_end = GREATEST(now(), current_period_end) + ($2 * interval '1 month'),
			due_notice_sent_at = NULL,
			updated_at = now()
		WHERE tenant_id = $1
		RETURNING `+subscriptionColumns, tenantID, months))
}

const paymentColumns = `id, tenant_id, subscription_id, plan, amount, status, method, external_id,
	xendit_invoice_id, xendit_invoice_url, payment_channel, paid_at, recorded_by, note,
	voucher_id, discount_centavos, created_at`

func scanPayment(row pgx.Row) (*billing.Payment, error) {
	var p billing.Payment
	err := row.Scan(&p.ID, &p.TenantID, &p.SubscriptionID, &p.Plan, &p.Amount, &p.Status, &p.Method,
		&p.ExternalID, &p.XenditInvoiceID, &p.XenditInvoiceURL, &p.PaymentChannel, &p.PaidAt,
		&p.RecordedBy, &p.Note, &p.VoucherID, &p.DiscountCentavos, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *BillingRepo) CreatePayment(ctx context.Context, p *billing.Payment) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO subscription_payments
			(tenant_id, subscription_id, plan, amount, status, method, external_id,
			 xendit_invoice_id, xendit_invoice_url, payment_channel, paid_at, recorded_by, note,
			 voucher_id, discount_centavos)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at`,
		p.TenantID, p.SubscriptionID, p.Plan, p.Amount, p.Status, p.Method, p.ExternalID,
		p.XenditInvoiceID, p.XenditInvoiceURL, p.PaymentChannel, p.PaidAt, p.RecordedBy, p.Note,
		p.VoucherID, p.DiscountCentavos,
	).Scan(&p.ID, &p.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create subscription payment: %w", err)
	}
	return nil
}

// MarkPaymentPaidIfPending is the webhook idempotency gate: zero rows
// affected means the payment was already processed (duplicate callback).
func (r *BillingRepo) MarkPaymentPaidIfPending(ctx context.Context, externalID, channel string, paidAt time.Time) (*billing.Payment, error) {
	p, err := scanPayment(r.db.QueryRow(ctx, `
		UPDATE subscription_payments
		SET status = 'paid', payment_channel = $2, paid_at = $3, updated_at = now()
		WHERE external_id = $1 AND status = 'pending'
		RETURNING `+paymentColumns, externalID, channel, paidAt))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // already paid/expired or unknown — caller decides
	}
	if err != nil {
		return nil, fmt.Errorf("failed to mark payment paid: %w", err)
	}
	return p, nil
}

func (r *BillingRepo) MarkPaymentExpiredIfPending(ctx context.Context, externalID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE subscription_payments SET status = 'expired', updated_at = now()
		WHERE external_id = $1 AND status = 'pending'`, externalID)
	if err != nil {
		return fmt.Errorf("failed to mark payment expired: %w", err)
	}
	return nil
}

func (r *BillingRepo) FindReusablePendingPayment(ctx context.Context, tenantID, plan string) (*billing.Payment, error) {
	p, err := scanPayment(r.db.QueryRow(ctx, `
		SELECT `+paymentColumns+`
		FROM subscription_payments
		WHERE tenant_id = $1 AND plan = $2 AND status = 'pending' AND method = 'xendit'
		  AND xendit_invoice_url <> '' AND created_at > now() - interval '24 hours'
		ORDER BY created_at DESC
		LIMIT 1`, tenantID, plan))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find pending payment: %w", err)
	}
	return p, nil
}

// LatestPendingXenditPayment returns the tenant's most recent pending
// Xendit payment (any plan) that still has an invoice id, so the return
// page can reconcile it directly against Xendit.
func (r *BillingRepo) LatestPendingXenditPayment(ctx context.Context, tenantID string) (*billing.Payment, error) {
	p, err := scanPayment(r.db.QueryRow(ctx, `
		SELECT `+paymentColumns+`
		FROM subscription_payments
		WHERE tenant_id = $1 AND status = 'pending' AND method = 'xendit' AND xendit_invoice_id <> ''
		ORDER BY created_at DESC
		LIMIT 1`, tenantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find pending xendit payment: %w", err)
	}
	return p, nil
}

func (r *BillingRepo) ListPaymentsByTenant(ctx context.Context, tenantID string, limit, offset int) ([]billing.Payment, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM subscription_payments WHERE tenant_id = $1`, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count payments: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT `+paymentColumns+`
		FROM subscription_payments
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list payments: %w", err)
	}
	defer rows.Close()

	var payments []billing.Payment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan payment: %w", err)
		}
		payments = append(payments, *p)
	}
	return payments, total, rows.Err()
}

const dueSubscriptionSelect = `
	SELECT s.id, s.tenant_id, t.name, t.timezone, s.plan, s.current_period_end,
	       t.owner_user_id, u.full_name, u.email`

func scanDueRows(rows pgx.Rows) ([]billing.DueSubscription, error) {
	defer rows.Close()
	var due []billing.DueSubscription
	for rows.Next() {
		var d billing.DueSubscription
		if err := rows.Scan(&d.SubscriptionID, &d.TenantID, &d.TenantName, &d.Timezone, &d.Plan,
			&d.PeriodEnd, &d.OwnerUserID, &d.OwnerName, &d.OwnerEmail); err != nil {
			return nil, fmt.Errorf("failed to scan due subscription: %w", err)
		}
		due = append(due, d)
	}
	return due, rows.Err()
}

func (r *BillingRepo) ListDueForNotice(ctx context.Context, within time.Duration) ([]billing.DueSubscription, error) {
	rows, err := r.db.Query(ctx, dueSubscriptionSelect+`
		FROM subscriptions s
		JOIN tenants t ON t.id = s.tenant_id AND t.deleted_at IS NULL
		JOIN users u ON u.id = t.owner_user_id AND u.deleted_at IS NULL
		WHERE s.status = 'active'
		  AND s.current_period_end <= now() + ($1 * interval '1 second')
		  AND s.current_period_end > now()
		  AND s.due_notice_sent_at IS NULL`,
		int64(within.Seconds()))
	if err != nil {
		return nil, fmt.Errorf("failed to list due subscriptions: %w", err)
	}
	return scanDueRows(rows)
}

func (r *BillingRepo) SetNoticeSent(ctx context.Context, subscriptionID string, at time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE subscriptions SET due_notice_sent_at = $2, updated_at = now() WHERE id = $1`,
		subscriptionID, at)
	if err != nil {
		return fmt.Errorf("failed to set notice sent: %w", err)
	}
	return nil
}

func (r *BillingRepo) DeactivateOverdue(ctx context.Context) ([]billing.DueSubscription, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE subscriptions s SET status = 'inactive', updated_at = now()
		FROM tenants t
		JOIN users u ON u.id = t.owner_user_id AND u.deleted_at IS NULL
		WHERE s.tenant_id = t.id AND t.deleted_at IS NULL
		  AND s.status = 'active' AND s.current_period_end < now()
		RETURNING s.id, s.tenant_id, t.name, t.timezone, s.plan, s.current_period_end,
		          t.owner_user_id, u.full_name, u.email`)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate overdue subscriptions: %w", err)
	}
	return scanDueRows(rows)
}

func (r *BillingRepo) ListSubscriptionsWithOwner(ctx context.Context, status string, limit, offset int) ([]billing.SubscriptionWithOwner, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM subscriptions s
		JOIN tenants t ON t.id = s.tenant_id AND t.deleted_at IS NULL
		WHERE ($1 = '' OR s.status = $1)`, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count subscriptions: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT s.id, s.tenant_id, s.plan, s.status, s.current_period_start, s.current_period_end,
		       s.due_notice_sent_at, s.created_at, s.updated_at,
		       t.name, t.slug, t.status,
		       COALESCE(u.full_name, ''), COALESCE(u.email, ''),
		       lp.paid_at, lp.amount
		FROM subscriptions s
		JOIN tenants t ON t.id = s.tenant_id AND t.deleted_at IS NULL
		LEFT JOIN users u ON u.id = t.owner_user_id AND u.deleted_at IS NULL
		LEFT JOIN LATERAL (
			SELECT paid_at, amount FROM subscription_payments p
			WHERE p.tenant_id = s.tenant_id AND p.status = 'paid'
			ORDER BY p.paid_at DESC LIMIT 1
		) lp ON true
		WHERE ($1 = '' OR s.status = $1)
		ORDER BY s.current_period_end ASC
		LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []billing.SubscriptionWithOwner
	for rows.Next() {
		var s billing.SubscriptionWithOwner
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Plan, &s.Status, &s.CurrentPeriodStart,
			&s.CurrentPeriodEnd, &s.DueNoticeSentAt, &s.CreatedAt, &s.UpdatedAt,
			&s.TenantName, &s.TenantSlug, &s.TenantStatus,
			&s.OwnerName, &s.OwnerEmail, &s.LastPaidAt, &s.LastPaidAmt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan subscription row: %w", err)
		}
		subs = append(subs, s)
	}
	return subs, total, rows.Err()
}

func (r *BillingRepo) ListOwners(ctx context.Context, limit, offset int) ([]billing.OwnerRow, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT t.owner_user_id) FROM tenants t
		WHERE t.deleted_at IS NULL AND t.owner_user_id IS NOT NULL`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count owners: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		WITH owners AS (
			SELECT u.id, u.full_name, u.email, u.status, u.created_at
			FROM users u
			WHERE u.deleted_at IS NULL
			  AND EXISTS (SELECT 1 FROM tenants t WHERE t.owner_user_id = u.id AND t.deleted_at IS NULL)
			ORDER BY u.full_name
			LIMIT $1 OFFSET $2
		)
		SELECT o.id, o.full_name, o.email, o.status, o.created_at,
		       t.id, t.name, t.slug,
		       COALESCE(s.plan, ''), COALESCE(s.status, ''), COALESCE(s.current_period_end, 'epoch'::timestamptz)
		FROM owners o
		JOIN tenants t ON t.owner_user_id = o.id AND t.deleted_at IS NULL
		LEFT JOIN subscriptions s ON s.tenant_id = t.id
		ORDER BY o.full_name, t.name`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list owners: %w", err)
	}
	defer rows.Close()

	var owners []billing.OwnerRow
	index := map[string]int{}
	for rows.Next() {
		var (
			o billing.OwnerRow
			b billing.OwnedBusiness
		)
		if err := rows.Scan(&o.UserID, &o.FullName, &o.Email, &o.UserStatus, &o.CreatedAt,
			&b.TenantID, &b.Name, &b.Slug, &b.Plan, &b.SubStatus, &b.PeriodEnd); err != nil {
			return nil, 0, fmt.Errorf("failed to scan owner row: %w", err)
		}
		i, seen := index[o.UserID]
		if !seen {
			owners = append(owners, o)
			i = len(owners) - 1
			index[o.UserID] = i
		}
		owners[i].Businesses = append(owners[i].Businesses, b)
	}
	return owners, total, rows.Err()
}

func (r *BillingRepo) BillingStats(ctx context.Context) (map[string]any, error) {
	stats := map[string]any{}
	var active, pending, inactive int64
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE status = 'active'),
		       COUNT(*) FILTER (WHERE status = 'pending'),
		       COUNT(*) FILTER (WHERE status = 'inactive')
		FROM subscriptions`).Scan(&active, &pending, &inactive); err != nil {
		return nil, fmt.Errorf("failed to count subscription statuses: %w", err)
	}
	stats["subs_active"] = active
	stats["subs_pending"] = pending
	stats["subs_inactive"] = inactive

	var collectedMonth, collected30d int64
	if err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount) FILTER (WHERE paid_at >= date_trunc('month', now())), 0),
		       COALESCE(SUM(amount) FILTER (WHERE paid_at >= now() - interval '30 days'), 0)
		FROM subscription_payments WHERE status = 'paid'`).Scan(&collectedMonth, &collected30d); err != nil {
		return nil, fmt.Errorf("failed to sum collections: %w", err)
	}
	stats["collected_this_month"] = collectedMonth
	stats["collected_30d"] = collected30d
	return stats, nil
}

func (r *BillingRepo) GetPlatformSettings(ctx context.Context) (*billing.PlatformSettings, error) {
	var s billing.PlatformSettings
	err := r.db.QueryRow(ctx, `
		SELECT monthly_price_centavos, yearly_price_centavos, updated_at
		FROM platform_settings WHERE id = 1`).Scan(&s.MonthlyPrice, &s.YearlyPrice, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get platform settings: %w", err)
	}
	return &s, nil
}

func (r *BillingRepo) UpdatePlatformSettings(ctx context.Context, monthly, yearly int64) (*billing.PlatformSettings, error) {
	var s billing.PlatformSettings
	err := r.db.QueryRow(ctx, `
		UPDATE platform_settings
		SET monthly_price_centavos = $1, yearly_price_centavos = $2, updated_at = now()
		WHERE id = 1
		RETURNING monthly_price_centavos, yearly_price_centavos, updated_at`,
		monthly, yearly).Scan(&s.MonthlyPrice, &s.YearlyPrice, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update platform settings: %w", err)
	}
	return &s, nil
}

// ---- vouchers ----

const voucherColumns = `id, code, discount_type, discount_value, applies_to,
	max_uses, used_count, expires_at, active, created_at, updated_at`

func scanVoucher(row pgx.Row) (*billing.Voucher, error) {
	var v billing.Voucher
	err := row.Scan(&v.ID, &v.Code, &v.DiscountType, &v.DiscountValue, &v.AppliesTo,
		&v.MaxUses, &v.UsedCount, &v.ExpiresAt, &v.Active, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *BillingRepo) CreateVoucher(ctx context.Context, v *billing.Voucher) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO vouchers (code, discount_type, discount_value, applies_to, max_uses, expires_at, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, used_count, created_at, updated_at`,
		v.Code, v.DiscountType, v.DiscountValue, v.AppliesTo, v.MaxUses, v.ExpiresAt, v.Active,
	).Scan(&v.ID, &v.UsedCount, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("a voucher with that code already exists")
		}
		return fmt.Errorf("failed to create voucher: %w", err)
	}
	return nil
}

func (r *BillingRepo) ListVouchers(ctx context.Context, limit, offset int) ([]billing.Voucher, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM vouchers WHERE deleted_at IS NULL`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count vouchers: %w", err)
	}
	rows, err := r.db.Query(ctx, `
		SELECT `+voucherColumns+`
		FROM vouchers WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list vouchers: %w", err)
	}
	defer rows.Close()
	var out []billing.Voucher
	for rows.Next() {
		v, err := scanVoucher(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *v)
	}
	return out, total, rows.Err()
}

func (r *BillingRepo) GetActiveVoucherByCode(ctx context.Context, code string) (*billing.Voucher, error) {
	v, err := scanVoucher(r.db.QueryRow(ctx, `
		SELECT `+voucherColumns+`
		FROM vouchers
		WHERE upper(code) = upper($1) AND active = TRUE AND deleted_at IS NULL`, code))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get voucher: %w", err)
	}
	return v, nil
}

func (r *BillingRepo) SetVoucherActive(ctx context.Context, id string, active bool) (*billing.Voucher, error) {
	v, err := scanVoucher(r.db.QueryRow(ctx, `
		UPDATE vouchers SET active = $2, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+voucherColumns, id, active))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("voucher")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update voucher: %w", err)
	}
	return v, nil
}

func (r *BillingRepo) SoftDeleteVoucher(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE vouchers SET deleted_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("failed to delete voucher: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("voucher")
	}
	return nil
}

func (r *BillingRepo) IncrementVoucherUse(ctx context.Context, id string) error {
	if _, err := r.db.Exec(ctx,
		`UPDATE vouchers SET used_count = used_count + 1, updated_at = now() WHERE id = $1`, id); err != nil {
		return fmt.Errorf("failed to increment voucher use: %w", err)
	}
	return nil
}
