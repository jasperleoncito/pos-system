package billing

import "time"

// Voucher discount types and plan scopes.
const (
	DiscountFixed      = "fixed"      // DiscountValue is centavos off
	DiscountPercentage = "percentage" // DiscountValue is a percent (1-100)

	VoucherAppliesAll = "all" // applies to any plan
)

// Voucher is a platform-level subscription discount code (super-admin
// managed). It applies to the owner's monthly/yearly subscription payment.
type Voucher struct {
	ID            string     `json:"id"`
	Code          string     `json:"code"`
	DiscountType  string     `json:"discount_type"`
	DiscountValue int64      `json:"discount_value"` // centavos (fixed) | percent (percentage)
	AppliesTo     string     `json:"applies_to"`     // all | monthly | yearly
	MaxUses       *int       `json:"max_uses,omitempty"`
	UsedCount     int        `json:"used_count"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	Active        bool       `json:"active"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ValidDiscountType reports whether the string is a known discount type.
func ValidDiscountType(t string) bool {
	return t == DiscountFixed || t == DiscountPercentage
}

// ValidVoucherScope reports whether the string is a known plan scope.
func ValidVoucherScope(s string) bool {
	return s == VoucherAppliesAll || s == PlanMonthly || s == PlanYearly
}

// DiscountFor returns the centavo discount applied to amount, never more
// than the amount itself.
func (v Voucher) DiscountFor(amount int64) int64 {
	var d int64
	if v.DiscountType == DiscountPercentage {
		d = amount * v.DiscountValue / 100
	} else {
		d = v.DiscountValue
	}
	if d > amount {
		d = amount
	}
	if d < 0 {
		d = 0
	}
	return d
}

// RedeemError returns a human-readable reason the voucher can't be used for
// the given plan at time now, or "" when it's redeemable.
func (v Voucher) RedeemError(plan string, now time.Time) string {
	switch {
	case !v.Active:
		return "this voucher is no longer active"
	case v.ExpiresAt != nil && now.After(*v.ExpiresAt):
		return "this voucher has expired"
	case v.MaxUses != nil && v.UsedCount >= *v.MaxUses:
		return "this voucher has reached its usage limit"
	case v.AppliesTo != VoucherAppliesAll && v.AppliesTo != plan:
		return "this voucher doesn't apply to the " + plan + " plan"
	default:
		return ""
	}
}
