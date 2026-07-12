package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/tenant"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

type TenantRepo struct {
	db *pgxpool.Pool
}

func NewTenantRepo(db *pgxpool.Pool) *TenantRepo { return &TenantRepo{db: db} }

const tenantColumns = `id, name, slug, owner_user_id, status, currency, timezone, plan, created_at, updated_at`

func scanTenant(row pgx.Row) (*tenant.Tenant, error) {
	var t tenant.Tenant
	err := row.Scan(&t.ID, &t.Name, &t.Slug, &t.OwnerUserID, &t.Status, &t.Currency, &t.Timezone,
		&t.Plan, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("tenant")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan tenant: %w", err)
	}
	return &t, nil
}

func (r *TenantRepo) Create(ctx context.Context, t *tenant.Tenant) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO tenants (name, slug, owner_user_id, status, currency, timezone)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`,
		t.Name, t.Slug, t.OwnerUserID, t.Status, t.Currency, t.Timezone,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("business URL slug is already taken")
		}
		return fmt.Errorf("failed to create tenant: %w", err)
	}
	return nil
}

func (r *TenantRepo) GetByID(ctx context.Context, id string) (*tenant.Tenant, error) {
	return scanTenant(r.db.QueryRow(ctx,
		`SELECT `+tenantColumns+` FROM tenants WHERE id = $1 AND deleted_at IS NULL`, id))
}

func (r *TenantRepo) GetBySlug(ctx context.Context, slug string) (*tenant.Tenant, error) {
	return scanTenant(r.db.QueryRow(ctx,
		`SELECT `+tenantColumns+` FROM tenants WHERE slug = $1 AND deleted_at IS NULL`, slug))
}

func (r *TenantRepo) Update(ctx context.Context, t *tenant.Tenant) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tenants SET name = $2, status = $3, currency = $4, timezone = $5, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL`,
		t.ID, t.Name, t.Status, t.Currency, t.Timezone)
	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}
	return nil
}

// SetPlan updates the subscription plan (platform admin only).
func (r *TenantRepo) SetPlan(ctx context.Context, id, plan string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE tenants SET plan = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id, plan)
	if err != nil {
		return fmt.Errorf("failed to set plan: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("tenant")
	}
	return nil
}

// PlatformStats aggregates cross-tenant counters for the admin console.
func (r *TenantRepo) PlatformStats(ctx context.Context) (map[string]any, error) {
	stats := map[string]any{}
	var tenantsTotal, tenantsActive, users int64
	if err := r.db.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE status = 'active')
		FROM tenants WHERE deleted_at IS NULL`).Scan(&tenantsTotal, &tenantsActive); err != nil {
		return nil, fmt.Errorf("failed to count tenants: %w", err)
	}
	if err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM users WHERE deleted_at IS NULL`).Scan(&users); err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}
	var orders30, gmv30 int64
	if err := r.db.QueryRow(ctx, `
		SELECT count(*), COALESCE(SUM(total), 0)
		FROM orders
		WHERE status IN ('completed', 'partially_refunded', 'refunded') AND deleted_at IS NULL
		  AND completed_at >= now() - interval '30 days'`).Scan(&orders30, &gmv30); err != nil {
		return nil, fmt.Errorf("failed to sum platform orders: %w", err)
	}
	stats["tenants_total"] = tenantsTotal
	stats["tenants_active"] = tenantsActive
	stats["users_total"] = users
	stats["orders_30d"] = orders30
	stats["gmv_30d"] = gmv30
	return stats, nil
}

// PlatformSales aggregates cross-tenant sales for the super-admin analytics
// view: a zero-filled daily series, the totals, top businesses, and the
// subscription revenue collected over the window.
func (r *TenantRepo) PlatformSales(ctx context.Context, days int) (*tenant.PlatformSales, error) {
	out := &tenant.PlatformSales{Days: days}

	rows, err := r.db.Query(ctx, `
		SELECT to_char(d.day, 'YYYY-MM-DD'),
		       COALESCE(SUM(o.total), 0)::bigint,
		       COUNT(o.id)::bigint
		FROM generate_series(
		        (now() - make_interval(days => $1 - 1))::date,
		        now()::date,
		        interval '1 day') AS d(day)
		LEFT JOIN orders o
		       ON o.completed_at >= d.day
		      AND o.completed_at < d.day + interval '1 day'
		      AND o.status IN `+saleStatuses+` AND o.deleted_at IS NULL
		GROUP BY d.day
		ORDER BY d.day`, days)
	if err != nil {
		return nil, fmt.Errorf("failed to load platform sales series: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p tenant.PlatformSalesPoint
		if err := rows.Scan(&p.Date, &p.Sales, &p.Orders); err != nil {
			return nil, err
		}
		out.GrossSales += p.Sales
		out.Orders += p.Orders
		out.Series = append(out.Series, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	brows, err := r.db.Query(ctx, `
		SELECT t.id, t.name, t.slug,
		       COALESCE(SUM(o.total), 0)::bigint, COUNT(o.id)::bigint
		FROM orders o
		JOIN tenants t ON t.id = o.tenant_id AND t.deleted_at IS NULL
		WHERE o.status IN `+saleStatuses+` AND o.deleted_at IS NULL
		  AND o.completed_at >= now() - make_interval(days => $1)
		GROUP BY t.id, t.name, t.slug
		ORDER BY 4 DESC
		LIMIT 10`, days)
	if err != nil {
		return nil, fmt.Errorf("failed to load top businesses: %w", err)
	}
	defer brows.Close()
	for brows.Next() {
		var b tenant.TopBusiness
		if err := brows.Scan(&b.TenantID, &b.Name, &b.Slug, &b.Sales, &b.Orders); err != nil {
			return nil, err
		}
		out.TopBusinesses = append(out.TopBusinesses, b)
	}
	if err := brows.Err(); err != nil {
		return nil, err
	}

	if err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)::bigint
		FROM subscription_payments
		WHERE status = 'paid' AND paid_at >= now() - make_interval(days => $1)`, days).
		Scan(&out.SubscriptionRevenue); err != nil {
		return nil, fmt.Errorf("failed to sum subscription revenue: %w", err)
	}

	return out, nil
}

func (r *TenantRepo) List(ctx context.Context, limit, offset int) ([]tenant.Tenant, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM tenants WHERE deleted_at IS NULL`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count tenants: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT `+tenantColumns+` FROM tenants WHERE deleted_at IS NULL
		ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []tenant.Tenant
	for rows.Next() {
		t, err := scanTenant(rows)
		if err != nil {
			return nil, 0, err
		}
		tenants = append(tenants, *t)
	}
	return tenants, total, rows.Err()
}
