// Package promo defines discounts and coupons. Percent values are
// 0–100 with two decimals; fixed values are centavos.
package promo

import (
	"context"
	"time"
)

const (
	TypePercent = "percent"
	TypeFixed   = "fixed"
)

func ValidType(t string) bool { return t == TypePercent || t == TypeFixed }

type Discount struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	PercentValue     float64   `json:"percent_value"`
	AmountValue      int64     `json:"amount_value"`
	RequiresApproval bool      `json:"requires_approval"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Coupon struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	Code           string     `json:"code"`
	DiscountType   string     `json:"discount_type"`
	PercentValue   float64    `json:"percent_value"`
	AmountValue    int64      `json:"amount_value"`
	MinOrderAmount int64      `json:"min_order_amount"`
	MaxUses        int        `json:"max_uses"`
	UsesCount      int        `json:"uses_count"`
	ValidFrom      *time.Time `json:"valid_from"`
	ValidTo        *time.Time `json:"valid_to"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Apply computes the discount for a subtotal, capped at the subtotal.
func Apply(discountType string, percentValue float64, amountValue, subtotal int64) int64 {
	var d int64
	switch discountType {
	case TypePercent:
		d = int64(float64(subtotal)*percentValue/100 + 0.5)
	case TypeFixed:
		d = amountValue
	}
	if d > subtotal {
		d = subtotal
	}
	if d < 0 {
		d = 0
	}
	return d
}

type DiscountRepository interface {
	Create(ctx context.Context, tenantID string, d *Discount) error
	GetByID(ctx context.Context, tenantID, id string) (*Discount, error)
	List(ctx context.Context, tenantID string) ([]Discount, error)
	Update(ctx context.Context, tenantID string, d *Discount) error
	SoftDelete(ctx context.Context, tenantID, id string) error
}

type CouponRepository interface {
	Create(ctx context.Context, tenantID string, c *Coupon) error
	GetByID(ctx context.Context, tenantID, id string) (*Coupon, error)
	GetByCode(ctx context.Context, tenantID, code string) (*Coupon, error)
	List(ctx context.Context, tenantID string) ([]Coupon, error)
	Update(ctx context.Context, tenantID string, c *Coupon) error
	SoftDelete(ctx context.Context, tenantID, id string) error
	// Redeem atomically increments uses_count while enforcing max_uses,
	// and records the redemption. Returns false when the coupon is
	// exhausted.
	Redeem(ctx context.Context, tenantID, couponID, orderID string) (bool, error)
	// Release frees a redemption when its order is voided.
	Release(ctx context.Context, tenantID, couponID, orderID string) error
}
