// Package order defines the POS order aggregate, payments, and the
// cash drawer. All money is integer centavos.
package order

import (
	"context"
	"time"
)

// Order lifecycle statuses.
const (
	StatusOpen              = "open"
	StatusHeld              = "held"
	StatusCompleted         = "completed"
	StatusVoided            = "voided"
	StatusRefunded          = "refunded"
	StatusPartiallyRefunded = "partially_refunded"
)

// Payment methods.
const (
	MethodCash         = "cash"
	MethodGCash        = "gcash"
	MethodCard         = "card"
	MethodMaya         = "maya"
	MethodBankTransfer = "bank_transfer"
)

// ValidMethod reports whether the payment method is supported.
func ValidMethod(m string) bool {
	switch m {
	case MethodCash, MethodGCash, MethodCard, MethodMaya, MethodBankTransfer:
		return true
	}
	return false
}

// Order types.
const (
	TypeDineIn   = "dine_in"
	TypeTakeout  = "takeout"
	TypeDelivery = "delivery"
)

func ValidOrderType(t string) bool {
	return t == TypeDineIn || t == TypeTakeout || t == TypeDelivery
}

type Order struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	OrderNumber   int64      `json:"order_number"`
	OrderType     string     `json:"order_type"`
	TableNumber   string     `json:"table_number"`
	CustomerID    *string    `json:"customer_id"`
	CashierUserID string     `json:"cashier_user_id"`
	CashierName   string     `json:"cashier_name,omitempty"`
	Status        string     `json:"status"`
	KitchenStatus string     `json:"kitchen_status"`
	Priority      bool       `json:"priority"`
	Subtotal      int64      `json:"subtotal"`
	DiscountTotal int64      `json:"discount_total"`
	TaxTotal      int64      `json:"tax_total"`
	Total         int64      `json:"total"`
	Tendered      int64      `json:"tendered"`
	Change        int64      `json:"change"`
	Notes         string     `json:"notes"`
	DiscountID    *string    `json:"discount_id,omitempty"`
	CouponID      *string    `json:"coupon_id,omitempty"`
	CompletedAt   *time.Time `json:"completed_at"`
	VoidedBy      *string    `json:"voided_by,omitempty"`
	VoidReason    string     `json:"void_reason,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	Items    []Item    `json:"items,omitempty"`
	Payments []Payment `json:"payments,omitempty"`
	Splits   []Split   `json:"splits,omitempty"`
	Refunds  []Refund  `json:"refunds,omitempty"`
}

type Split struct {
	ID          string    `json:"id"`
	OrderID     string    `json:"order_id"`
	SplitNumber int       `json:"split_number"`
	Amount      int64     `json:"amount"`
	Status      string    `json:"status"` // pending | paid
	CreatedAt   time.Time `json:"created_at"`
}

type Refund struct {
	ID           string       `json:"id"`
	OrderID      string       `json:"order_id"`
	RefundNumber int64        `json:"refund_number"`
	Reason       string       `json:"reason"`
	Amount       int64        `json:"amount"`
	RefundedBy   string       `json:"refunded_by"`
	CreatedAt    time.Time    `json:"created_at"`
	Items        []RefundItem `json:"items,omitempty"`
}

type RefundItem struct {
	ID          string `json:"id"`
	RefundID    string `json:"refund_id"`
	OrderItemID string `json:"order_item_id"`
	Qty         int    `json:"qty"`
	Amount      int64  `json:"amount"`
}

type Item struct {
	ID             string         `json:"id"`
	OrderID        string         `json:"order_id"`
	ProductID      string         `json:"product_id"`
	VariantID      *string        `json:"variant_id"`
	Name           string         `json:"name"`
	VariantName    string         `json:"variant_name"`
	UnitPrice      int64          `json:"unit_price"`
	Qty            int            `json:"qty"`
	DiscountAmount int64          `json:"discount_amount"`
	TaxAmount      int64          `json:"tax_amount"`
	LineTotal      int64          `json:"line_total"`
	Notes          string         `json:"notes"`
	Status         string         `json:"status"`
	Modifiers      []ItemModifier `json:"modifiers,omitempty"`
}

type ItemModifier struct {
	ID          string `json:"id"`
	OrderItemID string `json:"order_item_id"`
	ModifierID  string `json:"modifier_id"`
	GroupName   string `json:"group_name"`
	Name        string `json:"name"`
	PriceDelta  int64  `json:"price_delta"`
}

type Payment struct {
	ID          string    `json:"id"`
	OrderID     string    `json:"order_id"`
	SplitID     *string   `json:"split_id,omitempty"`
	Method      string    `json:"method"`
	Amount      int64     `json:"amount"`
	ReferenceNo string    `json:"reference_no"`
	Status      string    `json:"status"`
	ReceivedBy  string    `json:"received_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type DrawerSession struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	OpenedBy     string     `json:"opened_by"`
	ClosedBy     *string    `json:"closed_by"`
	OpeningFloat int64      `json:"opening_float"`
	ExpectedCash int64      `json:"expected_cash"`
	CountedCash  *int64     `json:"counted_cash"`
	Variance     *int64     `json:"variance"`
	Status       string     `json:"status"`
	OpenedAt     time.Time  `json:"opened_at"`
	ClosedAt     *time.Time `json:"closed_at"`
}

type CashMovement struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Type      string    `json:"type"`
	Amount    int64     `json:"amount"`
	OrderID   *string   `json:"order_id"`
	Reason    string    `json:"reason"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// Filter shapes order listing queries.
type Filter struct {
	Status string
	Search string // order number
	Limit  int
	Offset int
}

type Repository interface {
	// NextOrderNumber atomically increments the tenant's counter.
	NextOrderNumber(ctx context.Context, tenantID string) (int64, error)
	Create(ctx context.Context, tenantID string, o *Order) error
	GetByID(ctx context.Context, tenantID, id string) (*Order, error)
	List(ctx context.Context, tenantID string, f Filter) ([]Order, int64, error)
	UpdateStatus(ctx context.Context, tenantID, id, status string, completedAt *time.Time) error
	UpdatePaymentTotals(ctx context.Context, tenantID, id string, tendered, change int64) error
	AddStatusHistory(ctx context.Context, tenantID, orderID, field, from, to, changedBy string) error

	AddPayment(ctx context.Context, tenantID string, p *Payment) error
	ListPayments(ctx context.Context, tenantID, orderID string) ([]Payment, error)

	// UpdatePromo stores the applied discount/coupon and new totals.
	UpdatePromo(ctx context.Context, tenantID, id string, discountID, couponID *string, discountTotal, total int64) error
	SetVoided(ctx context.Context, tenantID, id, voidedBy, reason string) error

	CreateSplits(ctx context.Context, tenantID, orderID string, amounts []int64) ([]Split, error)
	ListSplits(ctx context.Context, tenantID, orderID string) ([]Split, error)
	GetSplit(ctx context.Context, tenantID, splitID string) (*Split, error)
	MarkSplitPaid(ctx context.Context, tenantID, splitID string) error

	NextRefundNumber(ctx context.Context, tenantID string) (int64, error)
	CreateRefund(ctx context.Context, tenantID string, r *Refund) error
	ListRefunds(ctx context.Context, tenantID, orderID string) ([]Refund, error)
	// RefundedTotal is the amount already refunded on an order.
	RefundedTotal(ctx context.Context, tenantID, orderID string) (int64, error)

	// ListKitchen returns fired, still-active kitchen tickets.
	ListKitchen(ctx context.Context, tenantID string) ([]Order, error)
	UpdateKitchenStatus(ctx context.Context, tenantID, orderID, status string) error
	UpdateItemStatus(ctx context.Context, tenantID, orderID, itemID, status string) error
	SetPriority(ctx context.Context, tenantID, orderID string, priority bool) error
}

// Kitchen statuses in flow order.
const (
	KitchenPending   = "pending"
	KitchenPreparing = "preparing"
	KitchenReady     = "ready"
	KitchenCompleted = "completed"
)

func ValidKitchenStatus(s string) bool {
	switch s {
	case KitchenPending, KitchenPreparing, KitchenReady, KitchenCompleted:
		return true
	}
	return false
}

type DrawerRepository interface {
	Open(ctx context.Context, tenantID string, s *DrawerSession) error
	Current(ctx context.Context, tenantID string) (*DrawerSession, error)
	Close(ctx context.Context, tenantID, sessionID, closedBy string, countedCash int64) (*DrawerSession, error)
	AddMovement(ctx context.Context, tenantID string, m *CashMovement) error
	ListMovements(ctx context.Context, tenantID, sessionID string) ([]CashMovement, error)
}
