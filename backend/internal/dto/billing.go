package dto

type CheckoutRequest struct {
	Plan    string `json:"plan" binding:"required,oneof=monthly yearly"`
	Voucher string `json:"voucher" binding:"omitempty,max=40"`
}

type PreviewVoucherRequest struct {
	Code string `json:"code" binding:"required,max=40"`
	Plan string `json:"plan" binding:"required,oneof=monthly yearly"`
}

type CreateVoucherRequest struct {
	Code          string  `json:"code" binding:"required,min=3,max=40"`
	DiscountType  string  `json:"discount_type" binding:"required,oneof=fixed percentage"`
	DiscountValue int64   `json:"discount_value" binding:"required,min=1"` // centavos (fixed) | percent (percentage)
	AppliesTo     string  `json:"applies_to" binding:"required,oneof=all monthly yearly"`
	MaxUses       *int    `json:"max_uses" binding:"omitempty,min=1"`
	ExpiresAt     *string `json:"expires_at" binding:"omitempty"` // RFC3339, optional
}

type SetVoucherActiveRequest struct {
	Active bool `json:"active"`
}

type MarkPaidManualRequest struct {
	Note string `json:"note" binding:"max=500"`
}

type SetSubscriptionStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active inactive"`
}

type GrantMonthsRequest struct {
	Months int `json:"months" binding:"required,min=1,max=6"`
}

type UpdatePlatformPricesRequest struct {
	MonthlyPrice int64 `json:"monthly_price" binding:"required,min=1"` // centavos
	YearlyPrice  int64 `json:"yearly_price" binding:"required,min=1"`  // centavos
}
