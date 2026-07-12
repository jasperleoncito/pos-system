// Package billing defines subscription entities and persistence
// contracts for the Xendit-backed plan billing.
package billing

import (
	"context"
	"time"
)

// Plan cycle values. Prices live in PlatformSettings, not here.
const (
	PlanMonthly = "monthly"
	PlanYearly  = "yearly"
)

// Subscription status values.
const (
	StatusPending  = "pending"  // registered, first invoice unpaid
	StatusActive   = "active"   // paid through current_period_end
	StatusInactive = "inactive" // past due — tenant locked out
)

// Payment status values.
const (
	PaymentPending = "pending"
	PaymentPaid    = "paid"
	PaymentExpired = "expired"
)

// ValidPlan reports whether the string is a billable plan cycle.
func ValidPlan(plan string) bool {
	return plan == PlanMonthly || plan == PlanYearly
}

// PlatformSettings is the singleton price sheet (centavos).
type PlatformSettings struct {
	MonthlyPrice int64     `json:"monthly_price"`
	YearlyPrice  int64     `json:"yearly_price"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PriceFor returns the centavo price for a plan cycle.
func (s PlatformSettings) PriceFor(plan string) int64 {
	if plan == PlanYearly {
		return s.YearlyPrice
	}
	return s.MonthlyPrice
}

type Subscription struct {
	ID                 string     `json:"id"`
	TenantID           string     `json:"tenant_id"`
	Plan               string     `json:"plan"`
	Status             string     `json:"status"`
	CurrentPeriodStart time.Time  `json:"current_period_start"`
	CurrentPeriodEnd   time.Time  `json:"current_period_end"`
	DueNoticeSentAt    *time.Time `json:"due_notice_sent_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type Payment struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenant_id"`
	SubscriptionID   string     `json:"subscription_id"`
	Plan             string     `json:"plan"`
	Amount           int64      `json:"amount"` // centavos
	Status           string     `json:"status"`
	Method           string     `json:"method"` // xendit | manual
	ExternalID       string     `json:"external_id"`
	XenditInvoiceID  string     `json:"xendit_invoice_id"`
	XenditInvoiceURL string     `json:"xendit_invoice_url"`
	PaymentChannel   string     `json:"payment_channel"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
	RecordedBy       *string    `json:"recorded_by,omitempty"`
	Note             string     `json:"note"`
	VoucherID        *string    `json:"voucher_id,omitempty"`
	DiscountCentavos int64      `json:"discount_centavos"`
	CreatedAt        time.Time  `json:"created_at"`
}

// SubscriptionWithOwner joins a subscription with tenant + owner details
// for the super-admin console.
type SubscriptionWithOwner struct {
	Subscription
	TenantName   string     `json:"tenant_name"`
	TenantSlug   string     `json:"tenant_slug"`
	TenantStatus string     `json:"tenant_status"`
	OwnerName    string     `json:"owner_name"`
	OwnerEmail   string     `json:"owner_email"`
	LastPaidAt   *time.Time `json:"last_paid_at,omitempty"`
	LastPaidAmt  *int64     `json:"last_paid_amount,omitempty"`
}

// OwnerRow lists a platform owner and their businesses.
type OwnerRow struct {
	UserID     string    `json:"user_id"`
	FullName   string    `json:"full_name"`
	Email      string    `json:"email"`
	UserStatus string    `json:"user_status"`
	CreatedAt  time.Time `json:"created_at"`
	Businesses []OwnedBusiness `json:"businesses"`
}

type OwnedBusiness struct {
	TenantID   string    `json:"tenant_id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	Plan       string    `json:"plan"`
	SubStatus  string    `json:"sub_status"`
	PeriodEnd  time.Time `json:"period_end"`
}

// DueSubscription is a sweep row: a subscription approaching or past its
// due date, joined with what the notifier needs.
type DueSubscription struct {
	SubscriptionID string
	TenantID       string
	TenantName     string
	Timezone       string
	Plan           string
	PeriodEnd      time.Time
	OwnerUserID    string
	OwnerName      string
	OwnerEmail     string
}

type Repository interface {
	CreateSubscription(ctx context.Context, s *Subscription) error
	GetByTenant(ctx context.Context, tenantID string) (*Subscription, error)
	// SetStatus flips only the status (admin override / sweep).
	SetStatus(ctx context.Context, tenantID, status string) error
	// Extend atomically activates and pushes the period end out by one
	// plan interval from GREATEST(now, current period end), resetting
	// due_notice_sent_at. All date math happens in SQL.
	Extend(ctx context.Context, tenantID, plan string) (*Subscription, error)
	// ExtendMonths grants N calendar months (super-admin comp) and activates.
	ExtendMonths(ctx context.Context, tenantID string, months int) (*Subscription, error)

	CreatePayment(ctx context.Context, p *Payment) error
	// MarkPaymentPaidIfPending flips a pending payment to paid and
	// returns it; returns (nil, nil) when it was not pending (idempotent
	// webhook replays).
	MarkPaymentPaidIfPending(ctx context.Context, externalID, channel string, paidAt time.Time) (*Payment, error)
	MarkPaymentExpiredIfPending(ctx context.Context, externalID string) error
	// FindReusablePendingPayment returns a recent (<24h) pending xendit
	// payment for the tenant+plan, if any, so checkouts don't pile up.
	FindReusablePendingPayment(ctx context.Context, tenantID, plan string) (*Payment, error)
	// LatestPendingXenditPayment returns the tenant's most recent pending
	// xendit payment (any plan) for return-page reconciliation.
	LatestPendingXenditPayment(ctx context.Context, tenantID string) (*Payment, error)
	ListPaymentsByTenant(ctx context.Context, tenantID string, limit, offset int) ([]Payment, int64, error)

	// Sweep queries.
	ListDueForNotice(ctx context.Context, within time.Duration) ([]DueSubscription, error)
	SetNoticeSent(ctx context.Context, subscriptionID string, at time.Time) error
	// DeactivateOverdue marks active past-due subscriptions inactive and
	// returns the affected rows for notification.
	DeactivateOverdue(ctx context.Context) ([]DueSubscription, error)

	// Admin console.
	ListSubscriptionsWithOwner(ctx context.Context, status string, limit, offset int) ([]SubscriptionWithOwner, int64, error)
	ListOwners(ctx context.Context, limit, offset int) ([]OwnerRow, int64, error)
	BillingStats(ctx context.Context) (map[string]any, error)

	GetPlatformSettings(ctx context.Context) (*PlatformSettings, error)
	UpdatePlatformSettings(ctx context.Context, monthly, yearly int64) (*PlatformSettings, error)

	// Vouchers (platform-level subscription discounts).
	CreateVoucher(ctx context.Context, v *Voucher) error
	ListVouchers(ctx context.Context, limit, offset int) ([]Voucher, int64, error)
	// GetActiveVoucherByCode returns a live (active, not deleted) voucher by
	// case-insensitive code, or (nil, nil) if none.
	GetActiveVoucherByCode(ctx context.Context, code string) (*Voucher, error)
	SetVoucherActive(ctx context.Context, id string, active bool) (*Voucher, error)
	SoftDeleteVoucher(ctx context.Context, id string) error
	// IncrementVoucherUse bumps used_count; called once on payment success.
	IncrementVoucherUse(ctx context.Context, id string) error
}
