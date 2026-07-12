package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/auth"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/billing"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/tenant"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/xendit"
)

const (
	subscriptionCacheTTL = 60 * time.Second
	invoiceDuration      = 24 * 60 * 60 // seconds — matches the pending-payment reuse window
)

// SubscriptionCreator provisions the initial subscription row for a new
// tenant (implemented by BillingService; consumed by auth + team flows).
type SubscriptionCreator interface {
	CreateInitialSubscription(ctx context.Context, tenantID, plan, status string) error
}

// InvoiceCreator is the Xendit surface the service needs (fake-able).
type InvoiceCreator interface {
	Configured() bool
	CreateInvoice(ctx context.Context, in xendit.CreateInvoiceRequest) (*xendit.Invoice, error)
	GetInvoice(ctx context.Context, id string) (*xendit.Invoice, error)
}

// StatusCache is the Redis JSON cache surface used for the per-request
// subscription check.
type StatusCache interface {
	GetJSON(ctx context.Context, key string, target any) bool
	SetJSON(ctx context.Context, key string, value any, ttl time.Duration)
	DeletePrefix(ctx context.Context, prefix string)
}

// BillingService owns subscriptions, Xendit checkout/webhooks, and the
// super-admin billing console.
type BillingService struct {
	repo       billing.Repository
	tenants    tenant.Repository
	users      auth.UserRepository
	invoices   InvoiceCreator
	cache      StatusCache
	auditor    *AuditService
	logger     *slog.Logger
	appBaseURL string
	appName    string
}

type BillingServiceDeps struct {
	Repo       billing.Repository
	Tenants    tenant.Repository
	Users      auth.UserRepository
	Invoices   InvoiceCreator
	Cache      StatusCache
	Auditor    *AuditService
	Logger     *slog.Logger
	AppBaseURL string
	AppName    string
}

func NewBillingService(d BillingServiceDeps) *BillingService {
	return &BillingService{
		repo: d.Repo, tenants: d.Tenants, users: d.Users, invoices: d.Invoices, cache: d.Cache,
		auditor: d.Auditor, logger: d.Logger, appBaseURL: d.AppBaseURL, appName: d.AppName,
	}
}

func subscriptionCacheKey(tenantID string) string { return "billing:sub:" + tenantID }

// GetPlans returns the current price sheet (public — the register page
// shows it before any account exists).
func (s *BillingService) GetPlans(ctx context.Context) (*billing.PlatformSettings, error) {
	return s.repo.GetPlatformSettings(ctx)
}

// Subscription returns the tenant's subscription (any member).
func (s *BillingService) Subscription(ctx context.Context, tenantID string) (*billing.Subscription, error) {
	return s.repo.GetByTenant(ctx, tenantID)
}

// IsActive is the middleware check — cached 60s, busted on any status
// change made through this service.
func (s *BillingService) IsActive(ctx context.Context, tenantID string) (bool, error) {
	key := subscriptionCacheKey(tenantID)
	var cached string
	if s.cache.GetJSON(ctx, key, &cached) {
		return cached == billing.StatusActive, nil
	}
	sub, err := s.repo.GetByTenant(ctx, tenantID)
	if err != nil {
		return false, err
	}
	s.cache.SetJSON(ctx, key, sub.Status, subscriptionCacheTTL)
	return sub.Status == billing.StatusActive, nil
}

func (s *BillingService) bustCache(tenantID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.cache.DeletePrefix(ctx, subscriptionCacheKey(tenantID))
}

// CreateInitialSubscription is called from registration (pending),
// admin business creation, and the seeder (active, 30-day comp).
func (s *BillingService) CreateInitialSubscription(ctx context.Context, tenantID, plan, status string) error {
	if !billing.ValidPlan(plan) {
		return apperror.Validation("plan must be monthly or yearly")
	}
	sub := &billing.Subscription{TenantID: tenantID, Plan: plan, Status: status}
	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return err
	}
	s.bustCache(tenantID)
	return nil
}

// CheckoutResult is returned to the frontend, which redirects the
// browser to InvoiceURL.
type CheckoutResult struct {
	PaymentID  string `json:"payment_id"`
	Plan       string `json:"plan"`
	Amount     int64  `json:"amount"`   // final amount to pay (after any voucher)
	Discount   int64  `json:"discount"` // centavos saved by a voucher
	InvoiceURL string `json:"invoice_url"`
	Free       bool   `json:"free"` // a voucher covered the whole amount — no payment needed
}

// CreateCheckout creates (or reuses) a pending Xendit invoice for the
// tenant's next period on the chosen plan, optionally applying a voucher.
// Prices always come from platform settings — never from the client.
func (s *BillingService) CreateCheckout(ctx context.Context, tenantID, userID, plan, voucherCode string) (*CheckoutResult, error) {
	if !billing.ValidPlan(plan) {
		return nil, apperror.Validation("plan must be monthly or yearly")
	}

	sub, err := s.repo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	settings, err := s.repo.GetPlatformSettings(ctx)
	if err != nil {
		return nil, err
	}
	amount := settings.PriceFor(plan)

	// Optional voucher: re-validated server-side, discount computed here.
	var voucherID *string
	discount := int64(0)
	if voucherCode = strings.TrimSpace(voucherCode); voucherCode != "" {
		voucher, err := s.resolveVoucher(ctx, voucherCode, plan)
		if err != nil {
			return nil, err
		}
		discount = voucher.DiscountFor(amount)
		voucherID = &voucher.ID
	}
	finalAmount := amount - discount

	// A voucher that covers the full price → activate for free, no invoice.
	if finalAmount <= 0 {
		return s.redeemFreeCheckout(ctx, sub, userID, plan, amount, discount, voucherID)
	}

	if !s.invoices.Configured() {
		return nil, apperror.Validation("online payments are not configured on this server")
	}

	// Reuse a fresh pending invoice only if its amount AND voucher still
	// match — otherwise the old invoice is stale (Xendit invoices are
	// immutable), so fall through and mint a new one.
	if existing, err := s.repo.FindReusablePendingPayment(ctx, tenantID, plan); err != nil {
		return nil, err
	} else if existing != nil && existing.Amount == finalAmount && ptrStrEqual(existing.VoucherID, voucherID) {
		return &CheckoutResult{
			PaymentID: existing.ID, Plan: existing.Plan, Amount: existing.Amount,
			Discount: existing.DiscountCentavos, InvoiceURL: existing.XenditInvoiceURL,
		}, nil
	}

	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	payerEmail := ""
	if user, err := s.users.GetByID(ctx, userID); err == nil {
		payerEmail = user.Email
	}

	externalID, err := newExternalID()
	if err != nil {
		return nil, apperror.Internal(err)
	}

	invoice, err := s.invoices.CreateInvoice(ctx, xendit.CreateInvoiceRequest{
		ExternalID:         externalID,
		Amount:             float64(finalAmount) / 100, // centavos → PHP
		PayerEmail:         payerEmail,
		Description:        fmt.Sprintf("%s — %s subscription for %s", s.appName, plan, t.Name),
		SuccessRedirectURL: s.appBaseURL + "/billing/return",
		FailureRedirectURL: s.appBaseURL + "/billing/return?status=failed",
		InvoiceDuration:    invoiceDuration,
	})
	if err != nil {
		s.logger.Error("failed to create xendit invoice", "tenant", tenantID, "error", err)
		return nil, apperror.Wrap(apperror.KindInternal, "payment provider is unavailable, try again shortly", err)
	}

	payment := &billing.Payment{
		TenantID:         tenantID,
		SubscriptionID:   sub.ID,
		Plan:             plan,
		Amount:           finalAmount,
		Status:           billing.PaymentPending,
		Method:           "xendit",
		ExternalID:       externalID,
		XenditInvoiceID:  invoice.ID,
		XenditInvoiceURL: invoice.InvoiceURL,
		VoucherID:        voucherID,
		DiscountCentavos: discount,
	}
	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return nil, err
	}

	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "billing.checkout_created",
		EntityType: "subscription_payment", EntityID: payment.ID,
		After: map[string]any{"plan": plan, "amount": finalAmount, "discount": discount, "xendit_invoice_id": invoice.ID},
	})

	return &CheckoutResult{
		PaymentID: payment.ID, Plan: plan, Amount: finalAmount, Discount: discount, InvoiceURL: invoice.InvoiceURL,
	}, nil
}

// redeemFreeCheckout activates a subscription that a voucher made free:
// records a paid zero-amount payment, extends the period, and burns one
// voucher use — no Xendit round-trip.
func (s *BillingService) redeemFreeCheckout(ctx context.Context, sub *billing.Subscription, userID, plan string, amount, discount int64, voucherID *string) (*CheckoutResult, error) {
	externalID, err := newExternalID()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	now := time.Now()
	payment := &billing.Payment{
		TenantID: sub.TenantID, SubscriptionID: sub.ID, Plan: plan,
		Amount: 0, Status: billing.PaymentPaid, Method: "voucher",
		ExternalID: externalID, PaidAt: &now, Note: "covered by voucher",
		VoucherID: voucherID, DiscountCentavos: discount,
	}
	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return nil, err
	}
	if _, err := s.repo.Extend(ctx, sub.TenantID, plan); err != nil {
		return nil, err
	}
	if voucherID != nil {
		if err := s.repo.IncrementVoucherUse(ctx, *voucherID); err != nil {
			s.logger.Warn("failed to increment voucher use", "voucher", *voucherID, "error", err)
		}
	}
	s.bustCache(sub.TenantID)
	s.auditor.Record(audit.Log{
		TenantID: sub.TenantID, UserID: userID, Action: "billing.voucher_free_activation",
		EntityType: "subscription_payment", EntityID: payment.ID,
		After: map[string]any{"plan": plan, "original_amount": amount, "discount": discount},
	})
	return &CheckoutResult{PaymentID: payment.ID, Plan: plan, Amount: 0, Discount: discount, Free: true, InvoiceURL: s.appBaseURL + "/billing/return"}, nil
}

// ptrStrEqual reports whether two optional string pointers hold the same
// value (both nil counts as equal).
func ptrStrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// ListPayments returns the tenant's own payment history (owner).
func (s *BillingService) ListPayments(ctx context.Context, tenantID string, limit, offset int) ([]billing.Payment, int64, error) {
	return s.repo.ListPaymentsByTenant(ctx, tenantID, limit, offset)
}

// ReconcilePending is the webhook-independent confirmation path: the
// return page polls it, and it asks Xendit directly whether the tenant's
// latest pending invoice is paid — activating the subscription if so.
// Idempotent and safe to call repeatedly; returns the current subscription.
func (s *BillingService) ReconcilePending(ctx context.Context, tenantID string) (*billing.Subscription, error) {
	sub, err := s.repo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	// Already active, or no way to check — nothing to reconcile.
	if sub.Status == billing.StatusActive || !s.invoices.Configured() {
		return sub, nil
	}

	pending, err := s.repo.LatestPendingXenditPayment(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if pending == nil {
		return sub, nil
	}

	inv, err := s.invoices.GetInvoice(ctx, pending.XenditInvoiceID)
	if err != nil {
		// Don't fail the return page on a transient Xendit hiccup; the
		// caller just keeps polling and sees the still-pending status.
		s.logger.Warn("reconcile: fetch invoice failed", "tenant", tenantID,
			"invoice", pending.XenditInvoiceID, "error", err)
		return sub, nil
	}
	if !inv.IsPaid() {
		return sub, nil
	}

	// Paid — run the exact same idempotent path the webhook would.
	if err := s.HandleWebhook(ctx, xendit.InvoiceCallback{
		ID: inv.ID, ExternalID: pending.ExternalID, Status: inv.Status,
		PaidAmount: inv.PaidAmount, PaidAt: inv.PaidAt, PaymentChannel: inv.PaymentChannel,
	}); err != nil {
		return nil, err
	}
	s.logger.Info("reconcile activated subscription", "tenant", tenantID, "invoice", inv.ID)
	return s.repo.GetByTenant(ctx, tenantID)
}

// HandleWebhook processes a Xendit invoice callback. The handler has
// already verified the callback token. Always idempotent: duplicate
// callbacks and unknown external IDs resolve to a no-op.
func (s *BillingService) HandleWebhook(ctx context.Context, cb xendit.InvoiceCallback) error {
	switch {
	case cb.IsPaid():
		paidAt := time.Now()
		if t, err := time.Parse(time.RFC3339, cb.PaidAt); err == nil {
			paidAt = t
		}
		payment, err := s.repo.MarkPaymentPaidIfPending(ctx, cb.ExternalID, cb.PaymentChannel, paidAt)
		if err != nil {
			return err
		}
		if payment == nil {
			s.logger.Info("webhook ignored (already processed or unknown)", "external_id", cb.ExternalID, "status", cb.Status)
			return nil
		}
		// Burn one voucher use now that the discounted payment landed.
		if payment.VoucherID != nil {
			if err := s.repo.IncrementVoucherUse(ctx, *payment.VoucherID); err != nil {
				s.logger.Warn("failed to increment voucher use", "voucher", *payment.VoucherID, "error", err)
			}
		}
		if paid := int64(cb.PaidAmount * 100); paid != payment.Amount {
			s.logger.Warn("webhook amount mismatch", "external_id", cb.ExternalID,
				"expected", payment.Amount, "paid", paid)
		}
		sub, err := s.repo.Extend(ctx, payment.TenantID, payment.Plan)
		if err != nil {
			return err
		}
		s.bustCache(payment.TenantID)
		s.auditor.Record(audit.Log{
			TenantID: payment.TenantID, Action: "billing.payment_received",
			EntityType: "subscription_payment", EntityID: payment.ID,
			After: map[string]any{
				"plan": payment.Plan, "amount": payment.Amount,
				"channel": cb.PaymentChannel, "period_end": sub.CurrentPeriodEnd,
			},
		})
		return nil

	case cb.Status == "EXPIRED":
		return s.repo.MarkPaymentExpiredIfPending(ctx, cb.ExternalID)

	default:
		s.logger.Info("webhook status ignored", "external_id", cb.ExternalID, "status", cb.Status)
		return nil
	}
}

// ---- super-admin console ----

func (s *BillingService) ListSubscriptions(ctx context.Context, status string, limit, offset int) ([]billing.SubscriptionWithOwner, int64, error) {
	if status != "" && status != billing.StatusPending && status != billing.StatusActive && status != billing.StatusInactive {
		return nil, 0, apperror.Validation("status filter must be pending, active, or inactive")
	}
	return s.repo.ListSubscriptionsWithOwner(ctx, status, limit, offset)
}

func (s *BillingService) ListOwners(ctx context.Context, limit, offset int) ([]billing.OwnerRow, int64, error) {
	return s.repo.ListOwners(ctx, limit, offset)
}

func (s *BillingService) BillingStats(ctx context.Context) (map[string]any, error) {
	stats, err := s.repo.BillingStats(ctx)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return stats, nil
}

// MarkPaidManual records an out-of-band payment (e.g. bank transfer)
// and extends the subscription exactly like a webhook would.
func (s *BillingService) MarkPaidManual(ctx context.Context, actorID, tenantID, note string) (*billing.Subscription, error) {
	sub, err := s.repo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	settings, err := s.repo.GetPlatformSettings(ctx)
	if err != nil {
		return nil, err
	}

	externalID, err := newExternalID()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	now := time.Now()
	payment := &billing.Payment{
		TenantID:       tenantID,
		SubscriptionID: sub.ID,
		Plan:           sub.Plan,
		Amount:         settings.PriceFor(sub.Plan),
		Status:         billing.PaymentPaid,
		Method:         "manual",
		ExternalID:     externalID,
		PaidAt:         &now,
		RecordedBy:     &actorID,
		Note:           note,
	}
	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return nil, err
	}

	extended, err := s.repo.Extend(ctx, tenantID, sub.Plan)
	if err != nil {
		return nil, err
	}
	s.bustCache(tenantID)

	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: actorID, Action: "billing.manual_payment",
		EntityType: "subscription_payment", EntityID: payment.ID,
		After: map[string]any{"plan": sub.Plan, "amount": payment.Amount, "note": note, "period_end": extended.CurrentPeriodEnd},
	})
	return extended, nil
}

// GrantMonths comps a subscription by N months (1-6): a zero-amount grant
// recorded in the ledger that extends the period and activates. Super-admin.
func (s *BillingService) GrantMonths(ctx context.Context, actorID, tenantID string, months int) (*billing.Subscription, error) {
	if months < 1 || months > 6 {
		return nil, apperror.Validation("months must be between 1 and 6")
	}
	sub, err := s.repo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	externalID, err := newExternalID()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	unit := "months"
	if months == 1 {
		unit = "month"
	}
	now := time.Now()
	payment := &billing.Payment{
		TenantID:       tenantID,
		SubscriptionID: sub.ID,
		Plan:           sub.Plan,
		Amount:         0,
		Status:         billing.PaymentPaid,
		Method:         "grant",
		ExternalID:     externalID,
		PaidAt:         &now,
		RecordedBy:     &actorID,
		Note:           fmt.Sprintf("admin grant · %d %s", months, unit),
	}
	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return nil, err
	}
	extended, err := s.repo.ExtendMonths(ctx, tenantID, months)
	if err != nil {
		return nil, err
	}
	s.bustCache(tenantID)
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: actorID, Action: "billing.grant_months",
		EntityType: "subscription", EntityID: sub.ID,
		After: map[string]any{"months": months, "period_end": extended.CurrentPeriodEnd},
	})
	return extended, nil
}

// SetSubscriptionStatus is the super-admin force override.
func (s *BillingService) SetSubscriptionStatus(ctx context.Context, actorID, tenantID, status string) (*billing.Subscription, error) {
	if status != billing.StatusActive && status != billing.StatusInactive {
		return nil, apperror.Validation("status must be active or inactive")
	}
	before, err := s.repo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetStatus(ctx, tenantID, status); err != nil {
		return nil, err
	}
	s.bustCache(tenantID)
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: actorID, Action: "billing.status_overridden",
		EntityType: "subscription", EntityID: before.ID,
		Before: map[string]any{"status": before.Status}, After: map[string]any{"status": status},
	})
	return s.repo.GetByTenant(ctx, tenantID)
}

func (s *BillingService) GetPlatformSettings(ctx context.Context) (*billing.PlatformSettings, error) {
	return s.repo.GetPlatformSettings(ctx)
}

func (s *BillingService) UpdatePrices(ctx context.Context, actorID string, monthly, yearly int64) (*billing.PlatformSettings, error) {
	if monthly <= 0 || yearly <= 0 {
		return nil, apperror.Validation("prices must be positive centavo amounts")
	}
	before, err := s.repo.GetPlatformSettings(ctx)
	if err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdatePlatformSettings(ctx, monthly, yearly)
	if err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{
		UserID: actorID, Action: "billing.prices_changed", EntityType: "platform_settings", EntityID: "1",
		Before: map[string]any{"monthly": before.MonthlyPrice, "yearly": before.YearlyPrice},
		After:  map[string]any{"monthly": updated.MonthlyPrice, "yearly": updated.YearlyPrice},
	})
	return updated, nil
}

// ---- vouchers (super-admin) ----

// VoucherInput is the validated create payload for a voucher.
type VoucherInput struct {
	Code          string
	DiscountType  string
	DiscountValue int64
	AppliesTo     string
	MaxUses       *int
	ExpiresAt     *time.Time
}

func (s *BillingService) CreateVoucher(ctx context.Context, actorID string, in VoucherInput) (*billing.Voucher, error) {
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	switch {
	case len(code) < 3 || len(code) > 40:
		return nil, apperror.Validation("code must be 3-40 characters")
	case !billing.ValidDiscountType(in.DiscountType):
		return nil, apperror.Validation("discount type must be fixed or percentage")
	case !billing.ValidVoucherScope(in.AppliesTo):
		return nil, apperror.Validation("applies_to must be all, monthly, or yearly")
	case in.DiscountValue <= 0:
		return nil, apperror.Validation("discount value must be positive")
	case in.DiscountType == billing.DiscountPercentage && in.DiscountValue > 100:
		return nil, apperror.Validation("a percentage discount can't exceed 100")
	case in.MaxUses != nil && *in.MaxUses < 1:
		return nil, apperror.Validation("max uses must be at least 1")
	}
	v := &billing.Voucher{
		Code: code, DiscountType: in.DiscountType, DiscountValue: in.DiscountValue,
		AppliesTo: in.AppliesTo, MaxUses: in.MaxUses, ExpiresAt: in.ExpiresAt, Active: true,
	}
	if err := s.repo.CreateVoucher(ctx, v); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{
		UserID: actorID, Action: "billing.voucher_created", EntityType: "voucher", EntityID: v.ID,
		After: map[string]any{"code": v.Code, "type": v.DiscountType, "value": v.DiscountValue, "applies_to": v.AppliesTo},
	})
	return v, nil
}

func (s *BillingService) ListVouchers(ctx context.Context, limit, offset int) ([]billing.Voucher, int64, error) {
	return s.repo.ListVouchers(ctx, limit, offset)
}

func (s *BillingService) SetVoucherActive(ctx context.Context, actorID, id string, active bool) (*billing.Voucher, error) {
	v, err := s.repo.SetVoucherActive(ctx, id, active)
	if err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{
		UserID: actorID, Action: "billing.voucher_toggled", EntityType: "voucher", EntityID: id,
		After: map[string]any{"active": active},
	})
	return v, nil
}

func (s *BillingService) DeleteVoucher(ctx context.Context, actorID, id string) error {
	if err := s.repo.SoftDeleteVoucher(ctx, id); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		UserID: actorID, Action: "billing.voucher_deleted", EntityType: "voucher", EntityID: id,
	})
	return nil
}

// VoucherPreview is the owner-facing pricing for a code + plan.
type VoucherPreview struct {
	Code           string `json:"code"`
	Plan           string `json:"plan"`
	OriginalAmount int64  `json:"original_amount"`
	Discount       int64  `json:"discount"`
	FinalAmount    int64  `json:"final_amount"`
}

// PreviewVoucher validates a code against a plan and returns the pricing so
// the owner can see the discount before paying.
func (s *BillingService) PreviewVoucher(ctx context.Context, code, plan string) (*VoucherPreview, error) {
	if !billing.ValidPlan(plan) {
		return nil, apperror.Validation("plan must be monthly or yearly")
	}
	v, err := s.resolveVoucher(ctx, code, plan)
	if err != nil {
		return nil, err
	}
	settings, err := s.repo.GetPlatformSettings(ctx)
	if err != nil {
		return nil, err
	}
	amount := settings.PriceFor(plan)
	discount := v.DiscountFor(amount)
	return &VoucherPreview{
		Code: v.Code, Plan: plan, OriginalAmount: amount, Discount: discount, FinalAmount: amount - discount,
	}, nil
}

// resolveVoucher looks up an active code and validates it for a plan.
func (s *BillingService) resolveVoucher(ctx context.Context, code, plan string) (*billing.Voucher, error) {
	v, err := s.repo.GetActiveVoucherByCode(ctx, strings.TrimSpace(code))
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, apperror.Validation("that voucher code is invalid")
	}
	if reason := v.RedeemError(plan, time.Now()); reason != "" {
		return nil, apperror.Validation(reason)
	}
	return v, nil
}

// newExternalID mints the reference we hand to Xendit and key webhook
// lookups off — unguessable, prefixed for log readability.
func newExternalID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to generate external id: %w", err)
	}
	return "sub_" + hex.EncodeToString(raw), nil
}
