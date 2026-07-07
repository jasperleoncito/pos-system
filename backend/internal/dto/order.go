package dto

type OrderItemRequest struct {
	ProductID   string   `json:"product_id" binding:"required,uuid"`
	VariantID   string   `json:"variant_id" binding:"omitempty,uuid"`
	Qty         int      `json:"qty" binding:"required,min=1,max=999"`
	ModifierIDs []string `json:"modifier_ids" binding:"dive,uuid"`
	Notes       string   `json:"notes" binding:"max=300"`
}

type CreateOrderRequest struct {
	OrderType   string             `json:"order_type" binding:"required,oneof=dine_in takeout delivery"`
	TableNumber string             `json:"table_number" binding:"max=20"`
	Notes       string             `json:"notes" binding:"max=500"`
	Hold        bool               `json:"hold"`
	DiscountID  string             `json:"discount_id" binding:"omitempty,uuid"`
	CouponCode  string             `json:"coupon_code" binding:"max=40"`
	Items       []OrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

type PaymentLineRequest struct {
	Method      string `json:"method" binding:"required,oneof=cash gcash card maya bank_transfer"`
	Amount      int64  `json:"amount" binding:"required,min=1"`
	ReferenceNo string `json:"reference_no" binding:"max=100"`
}

type PayOrderRequest struct {
	Payments []PaymentLineRequest `json:"payments" binding:"required,min=1,dive"`
}

type HoldOrderRequest struct {
	Hold bool `json:"hold"`
}

type OpenDrawerRequest struct {
	OpeningFloat int64 `json:"opening_float" binding:"min=0"`
}

type CloseDrawerRequest struct {
	CountedCash int64 `json:"counted_cash" binding:"min=0"`
}
