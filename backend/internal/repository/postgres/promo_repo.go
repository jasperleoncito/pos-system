package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/promo"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

// ---- discounts ----

type DiscountRepo struct {
	db *pgxpool.Pool
}

func NewDiscountRepo(db *pgxpool.Pool) *DiscountRepo { return &DiscountRepo{db: db} }

const discountColumns = `id, tenant_id, name, type, percent_value, amount_value, requires_approval, is_active, created_at, updated_at`

func scanDiscount(row pgx.Row) (*promo.Discount, error) {
	var d promo.Discount
	err := row.Scan(&d.ID, &d.TenantID, &d.Name, &d.Type, &d.PercentValue, &d.AmountValue,
		&d.RequiresApproval, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("discount")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan discount: %w", err)
	}
	return &d, nil
}

func (r *DiscountRepo) Create(ctx context.Context, tenantID string, d *promo.Discount) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO discounts (tenant_id, name, type, percent_value, amount_value, requires_approval, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, tenant_id, created_at, updated_at`,
		tenantID, d.Name, d.Type, d.PercentValue, d.AmountValue, d.RequiresApproval, d.IsActive,
	).Scan(&d.ID, &d.TenantID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create discount: %w", err)
	}
	return nil
}

func (r *DiscountRepo) GetByID(ctx context.Context, tenantID, id string) (*promo.Discount, error) {
	return scanDiscount(r.db.QueryRow(ctx,
		`SELECT `+discountColumns+` FROM discounts WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id))
}

func (r *DiscountRepo) List(ctx context.Context, tenantID string) ([]promo.Discount, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+discountColumns+` FROM discounts WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY name`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list discounts: %w", err)
	}
	defer rows.Close()

	var discounts []promo.Discount
	for rows.Next() {
		d, err := scanDiscount(rows)
		if err != nil {
			return nil, err
		}
		discounts = append(discounts, *d)
	}
	return discounts, rows.Err()
}

func (r *DiscountRepo) Update(ctx context.Context, tenantID string, d *promo.Discount) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE discounts SET name = $3, type = $4, percent_value = $5, amount_value = $6,
			requires_approval = $7, is_active = $8, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, d.ID, d.Name, d.Type, d.PercentValue, d.AmountValue, d.RequiresApproval, d.IsActive)
	if err != nil {
		return fmt.Errorf("failed to update discount: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("discount")
	}
	return nil
}

func (r *DiscountRepo) SoftDelete(ctx context.Context, tenantID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE discounts SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to delete discount: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("discount")
	}
	return nil
}

// ---- coupons ----

type CouponRepo struct {
	db *pgxpool.Pool
}

func NewCouponRepo(db *pgxpool.Pool) *CouponRepo { return &CouponRepo{db: db} }

const couponColumns = `id, tenant_id, code, discount_type, percent_value, amount_value,
	min_order_amount, max_uses, uses_count, valid_from, valid_to, is_active, created_at, updated_at`

func scanCoupon(row pgx.Row) (*promo.Coupon, error) {
	var c promo.Coupon
	err := row.Scan(&c.ID, &c.TenantID, &c.Code, &c.DiscountType, &c.PercentValue, &c.AmountValue,
		&c.MinOrderAmount, &c.MaxUses, &c.UsesCount, &c.ValidFrom, &c.ValidTo, &c.IsActive,
		&c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("coupon")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan coupon: %w", err)
	}
	return &c, nil
}

func (r *CouponRepo) Create(ctx context.Context, tenantID string, c *promo.Coupon) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO coupons (tenant_id, code, discount_type, percent_value, amount_value,
			min_order_amount, max_uses, valid_from, valid_to, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, tenant_id, uses_count, created_at, updated_at`,
		tenantID, c.Code, c.DiscountType, c.PercentValue, c.AmountValue,
		c.MinOrderAmount, c.MaxUses, c.ValidFrom, c.ValidTo, c.IsActive,
	).Scan(&c.ID, &c.TenantID, &c.UsesCount, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("a coupon with that code already exists")
		}
		return fmt.Errorf("failed to create coupon: %w", err)
	}
	return nil
}

func (r *CouponRepo) GetByID(ctx context.Context, tenantID, id string) (*promo.Coupon, error) {
	return scanCoupon(r.db.QueryRow(ctx,
		`SELECT `+couponColumns+` FROM coupons WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id))
}

func (r *CouponRepo) GetByCode(ctx context.Context, tenantID, code string) (*promo.Coupon, error) {
	return scanCoupon(r.db.QueryRow(ctx,
		`SELECT `+couponColumns+` FROM coupons WHERE tenant_id = $1 AND upper(code) = upper($2) AND deleted_at IS NULL`,
		tenantID, code))
}

func (r *CouponRepo) List(ctx context.Context, tenantID string) ([]promo.Coupon, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+couponColumns+` FROM coupons WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list coupons: %w", err)
	}
	defer rows.Close()

	var coupons []promo.Coupon
	for rows.Next() {
		c, err := scanCoupon(rows)
		if err != nil {
			return nil, err
		}
		coupons = append(coupons, *c)
	}
	return coupons, rows.Err()
}

func (r *CouponRepo) Update(ctx context.Context, tenantID string, c *promo.Coupon) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE coupons SET code = $3, discount_type = $4, percent_value = $5, amount_value = $6,
			min_order_amount = $7, max_uses = $8, valid_from = $9, valid_to = $10, is_active = $11,
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, c.ID, c.Code, c.DiscountType, c.PercentValue, c.AmountValue,
		c.MinOrderAmount, c.MaxUses, c.ValidFrom, c.ValidTo, c.IsActive)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("a coupon with that code already exists")
		}
		return fmt.Errorf("failed to update coupon: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("coupon")
	}
	return nil
}

func (r *CouponRepo) SoftDelete(ctx context.Context, tenantID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE coupons SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to delete coupon: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("coupon")
	}
	return nil
}

// Redeem enforces max_uses atomically: the UPDATE only matches while
// uses remain, so concurrent redemptions cannot oversell.
func (r *CouponRepo) Redeem(ctx context.Context, tenantID, couponID, orderID string) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE coupons SET uses_count = uses_count + 1, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
		  AND (max_uses = 0 OR uses_count < max_uses)`,
		tenantID, couponID)
	if err != nil {
		return false, fmt.Errorf("failed to redeem coupon: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO coupon_redemptions (tenant_id, coupon_id, order_id)
		VALUES ($1, $2, $3)`, tenantID, couponID, orderID); err != nil {
		return false, fmt.Errorf("failed to record redemption: %w", err)
	}
	return true, tx.Commit(ctx)
}

func (r *CouponRepo) Release(ctx context.Context, tenantID, couponID, orderID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE coupon_redemptions SET released_at = now()
		WHERE tenant_id = $1 AND coupon_id = $2 AND order_id = $3 AND released_at IS NULL`,
		tenantID, couponID, orderID)
	if err != nil {
		return fmt.Errorf("failed to release redemption: %w", err)
	}
	if tag.RowsAffected() > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE coupons SET uses_count = greatest(uses_count - 1, 0), updated_at = now()
			WHERE tenant_id = $1 AND id = $2`, tenantID, couponID); err != nil {
			return fmt.Errorf("failed to decrement coupon uses: %w", err)
		}
	}
	return tx.Commit(ctx)
}
