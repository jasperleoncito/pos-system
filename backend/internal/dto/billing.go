package dto

type CheckoutRequest struct {
	Plan string `json:"plan" binding:"required,oneof=monthly yearly"`
}

type MarkPaidManualRequest struct {
	Note string `json:"note" binding:"max=500"`
}

type SetSubscriptionStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active inactive"`
}

type UpdatePlatformPricesRequest struct {
	MonthlyPrice int64 `json:"monthly_price" binding:"required,min=1"` // centavos
	YearlyPrice  int64 `json:"yearly_price" binding:"required,min=1"`  // centavos
}
