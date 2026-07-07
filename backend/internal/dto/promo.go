package dto

import "time"

type DiscountRequest struct {
	Name             string  `json:"name" binding:"required,min=1,max=120"`
	Type             string  `json:"type" binding:"required,oneof=percent fixed"`
	PercentValue     float64 `json:"percent_value" binding:"min=0,max=100"`
	AmountValue      int64   `json:"amount_value" binding:"min=0"`
	RequiresApproval bool    `json:"requires_approval"`
	IsActive         *bool   `json:"is_active"`
}

type CouponRequest struct {
	Code           string     `json:"code" binding:"required,min=2,max=40"`
	DiscountType   string     `json:"discount_type" binding:"required,oneof=percent fixed"`
	PercentValue   float64    `json:"percent_value" binding:"min=0,max=100"`
	AmountValue    int64      `json:"amount_value" binding:"min=0"`
	MinOrderAmount int64      `json:"min_order_amount" binding:"min=0"`
	MaxUses        int        `json:"max_uses" binding:"min=0"`
	ValidFrom      *time.Time `json:"valid_from"`
	ValidTo        *time.Time `json:"valid_to"`
	IsActive       *bool      `json:"is_active"`
}

type ValidateCouponRequest struct {
	Code     string `json:"code" binding:"required"`
	Subtotal int64  `json:"subtotal" binding:"required,min=1"`
}

type CreateSplitsRequest struct {
	Amounts []int64 `json:"amounts" binding:"required,min=2,dive,min=1"`
}

type RefundItemRequest struct {
	OrderItemID string `json:"order_item_id" binding:"required,uuid"`
	Qty         int    `json:"qty" binding:"required,min=1"`
}

type RefundRequest struct {
	Reason string              `json:"reason" binding:"required,min=3,max=300"`
	Items  []RefundItemRequest `json:"items" binding:"dive"`
	Amount int64               `json:"amount" binding:"min=0"`
}

type VoidRequest struct {
	Reason string `json:"reason" binding:"required,min=3,max=300"`
}
