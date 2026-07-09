package service

import (
	"context"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/auth"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/billing"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/tenant"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/xendit"
)

// ---- fakes ----

type fakeBillingRepo struct {
	subs     map[string]*billing.Subscription // by tenant id
	payments map[string]*billing.Payment      // by external id
	settings billing.PlatformSettings
	nextID   int
	extends  int // how many times Extend ran
}

func newFakeBillingRepo() *fakeBillingRepo {
	return &fakeBillingRepo{
		subs:     map[string]*billing.Subscription{},
		payments: map[string]*billing.Payment{},
		settings: billing.PlatformSettings{MonthlyPrice: 80000, YearlyPrice: 800000},
	}
}

func (r *fakeBillingRepo) id(prefix string) string {
	r.nextID++
	return prefix + "-" + strconv.Itoa(r.nextID)
}

func (r *fakeBillingRepo) CreateSubscription(_ context.Context, s *billing.Subscription) error {
	if existing, ok := r.subs[s.TenantID]; ok {
		*s = *existing
		return nil
	}
	s.ID = r.id("sub")
	now := time.Now()
	s.CurrentPeriodStart = now
	if s.Status == billing.StatusActive {
		s.CurrentPeriodEnd = now.Add(30 * 24 * time.Hour)
	} else {
		s.CurrentPeriodEnd = now
	}
	copied := *s
	r.subs[s.TenantID] = &copied
	return nil
}

func (r *fakeBillingRepo) GetByTenant(_ context.Context, tenantID string) (*billing.Subscription, error) {
	if s, ok := r.subs[tenantID]; ok {
		copied := *s
		return &copied, nil
	}
	return nil, apperror.NotFound("subscription")
}

func (r *fakeBillingRepo) SetStatus(_ context.Context, tenantID, status string) error {
	s, ok := r.subs[tenantID]
	if !ok {
		return apperror.NotFound("subscription")
	}
	s.Status = status
	return nil
}

func (r *fakeBillingRepo) Extend(_ context.Context, tenantID, plan string) (*billing.Subscription, error) {
	s, ok := r.subs[tenantID]
	if !ok {
		return nil, apperror.NotFound("subscription")
	}
	r.extends++
	base := s.CurrentPeriodEnd
	if now := time.Now(); base.Before(now) {
		base = now
	}
	if plan == billing.PlanMonthly {
		s.CurrentPeriodEnd = base.AddDate(0, 1, 0)
	} else {
		s.CurrentPeriodEnd = base.AddDate(1, 0, 0)
	}
	s.Plan = plan
	s.Status = billing.StatusActive
	s.DueNoticeSentAt = nil
	copied := *s
	return &copied, nil
}

func (r *fakeBillingRepo) CreatePayment(_ context.Context, p *billing.Payment) error {
	p.ID = r.id("pay")
	p.CreatedAt = time.Now()
	copied := *p
	r.payments[p.ExternalID] = &copied
	return nil
}

func (r *fakeBillingRepo) MarkPaymentPaidIfPending(_ context.Context, externalID, channel string, paidAt time.Time) (*billing.Payment, error) {
	p, ok := r.payments[externalID]
	if !ok || p.Status != billing.PaymentPending {
		return nil, nil
	}
	p.Status = billing.PaymentPaid
	p.PaymentChannel = channel
	p.PaidAt = &paidAt
	copied := *p
	return &copied, nil
}

func (r *fakeBillingRepo) MarkPaymentExpiredIfPending(_ context.Context, externalID string) error {
	if p, ok := r.payments[externalID]; ok && p.Status == billing.PaymentPending {
		p.Status = billing.PaymentExpired
	}
	return nil
}

func (r *fakeBillingRepo) FindReusablePendingPayment(_ context.Context, tenantID, plan string) (*billing.Payment, error) {
	for _, p := range r.payments {
		if p.TenantID == tenantID && p.Plan == plan && p.Status == billing.PaymentPending &&
			p.Method == "xendit" && time.Since(p.CreatedAt) < 24*time.Hour {
			copied := *p
			return &copied, nil
		}
	}
	return nil, nil
}

func (r *fakeBillingRepo) ListPaymentsByTenant(_ context.Context, tenantID string, _, _ int) ([]billing.Payment, int64, error) {
	var out []billing.Payment
	for _, p := range r.payments {
		if p.TenantID == tenantID {
			out = append(out, *p)
		}
	}
	return out, int64(len(out)), nil
}

func (r *fakeBillingRepo) ListDueForNotice(_ context.Context, _ time.Duration) ([]billing.DueSubscription, error) {
	return nil, nil
}
func (r *fakeBillingRepo) SetNoticeSent(_ context.Context, _ string, _ time.Time) error { return nil }
func (r *fakeBillingRepo) DeactivateOverdue(_ context.Context) ([]billing.DueSubscription, error) {
	return nil, nil
}
func (r *fakeBillingRepo) ListSubscriptionsWithOwner(_ context.Context, _ string, _, _ int) ([]billing.SubscriptionWithOwner, int64, error) {
	return nil, 0, nil
}
func (r *fakeBillingRepo) ListOwners(_ context.Context, _, _ int) ([]billing.OwnerRow, int64, error) {
	return nil, 0, nil
}
func (r *fakeBillingRepo) BillingStats(_ context.Context) (map[string]any, error) { return nil, nil }
func (r *fakeBillingRepo) GetPlatformSettings(_ context.Context) (*billing.PlatformSettings, error) {
	copied := r.settings
	return &copied, nil
}
func (r *fakeBillingRepo) UpdatePlatformSettings(_ context.Context, monthly, yearly int64) (*billing.PlatformSettings, error) {
	r.settings.MonthlyPrice = monthly
	r.settings.YearlyPrice = yearly
	copied := r.settings
	return &copied, nil
}

type fakeInvoices struct {
	created    int
	configured bool
}

func (f *fakeInvoices) Configured() bool { return f.configured }
func (f *fakeInvoices) CreateInvoice(_ context.Context, in xendit.CreateInvoiceRequest) (*xendit.Invoice, error) {
	f.created++
	return &xendit.Invoice{
		ID:         "inv-" + strconv.Itoa(f.created),
		InvoiceURL: "https://checkout.xendit.co/" + in.ExternalID,
		Status:     "PENDING",
	}, nil
}

type fakeCache struct {
	store map[string]any
	busts int
}

func newFakeCache() *fakeCache { return &fakeCache{store: map[string]any{}} }

func (c *fakeCache) GetJSON(_ context.Context, key string, target any) bool {
	v, ok := c.store[key]
	if !ok {
		return false
	}
	if s, ok := target.(*string); ok {
		*s = v.(string)
		return true
	}
	return false
}
func (c *fakeCache) SetJSON(_ context.Context, key string, value any, _ time.Duration) {
	c.store[key] = value
}
func (c *fakeCache) DeletePrefix(_ context.Context, prefix string) {
	c.busts++
	for k := range c.store {
		if strings.HasPrefix(k, prefix) {
			delete(c.store, k)
		}
	}
}

// ---- harness ----

type billingFixture struct {
	svc      *BillingService
	repo     *fakeBillingRepo
	invoices *fakeInvoices
	cache    *fakeCache
}

func newBillingFixture(t *testing.T) *billingFixture {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	users := newFakeUserRepo()
	owner := &auth.User{Email: "owner@biz.ph", FullName: "Owner", Status: "active"}
	if err := users.Create(context.Background(), owner); err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	tenants := newFakeTenantRepo()
	biz := &tenant.Tenant{Name: "Biz", Slug: "biz", OwnerUserID: owner.ID, Status: "active"}
	if err := tenants.Create(context.Background(), biz); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	f := &billingFixture{
		repo:     newFakeBillingRepo(),
		invoices: &fakeInvoices{configured: true},
		cache:    newFakeCache(),
	}
	f.svc = NewBillingService(BillingServiceDeps{
		Repo: f.repo, Tenants: tenants, Users: users, Invoices: f.invoices, Cache: f.cache,
		Auditor: NewAuditService(noopAuditRepo{}, logger), Logger: logger,
		AppBaseURL: "http://localhost:7642", AppName: "POS System",
	})
	// The fixture's tenant is always "tenant-1" / owner "user-1".
	return f
}

func (f *billingFixture) seedSubscription(t *testing.T, status string) *billing.Subscription {
	t.Helper()
	sub := &billing.Subscription{TenantID: "tenant-1", Plan: billing.PlanMonthly, Status: status}
	if err := f.repo.CreateSubscription(context.Background(), sub); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}
	return sub
}

// ---- tests ----

func TestCreateCheckoutCreatesThenReusesPendingInvoice(t *testing.T) {
	f := newBillingFixture(t)
	f.seedSubscription(t, billing.StatusPending)

	first, err := f.svc.CreateCheckout(context.Background(), "tenant-1", "user-1", "monthly")
	if err != nil {
		t.Fatalf("first checkout: %v", err)
	}
	if first.Amount != 80000 {
		t.Errorf("amount = %d, want platform monthly price 80000", first.Amount)
	}
	if first.InvoiceURL == "" {
		t.Error("expected an invoice URL")
	}

	second, err := f.svc.CreateCheckout(context.Background(), "tenant-1", "user-1", "monthly")
	if err != nil {
		t.Fatalf("second checkout: %v", err)
	}
	if f.invoices.created != 1 {
		t.Errorf("xendit invoices created = %d, want 1 (reuse)", f.invoices.created)
	}
	if second.PaymentID != first.PaymentID {
		t.Errorf("expected the same pending payment to be reused")
	}

	// A different plan mints a fresh invoice.
	if _, err := f.svc.CreateCheckout(context.Background(), "tenant-1", "user-1", "yearly"); err != nil {
		t.Fatalf("yearly checkout: %v", err)
	}
	if f.invoices.created != 2 {
		t.Errorf("xendit invoices created = %d, want 2 after plan switch", f.invoices.created)
	}
}

func TestCreateCheckoutRejectsBadPlanAndUnconfiguredGateway(t *testing.T) {
	f := newBillingFixture(t)
	f.seedSubscription(t, billing.StatusPending)

	if _, err := f.svc.CreateCheckout(context.Background(), "tenant-1", "user-1", "weekly"); err == nil {
		t.Error("invalid plan should be rejected")
	}
	f.invoices.configured = false
	if _, err := f.svc.CreateCheckout(context.Background(), "tenant-1", "user-1", "monthly"); err == nil {
		t.Error("unconfigured gateway should fail cleanly")
	}
}

func TestWebhookPaidExtendsOnceAndIsIdempotent(t *testing.T) {
	f := newBillingFixture(t)
	f.seedSubscription(t, billing.StatusPending)
	checkout, err := f.svc.CreateCheckout(context.Background(), "tenant-1", "user-1", "monthly")
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	payment := f.repo.payments[externalIDOf(t, f.repo, checkout.PaymentID)]

	cb := xendit.InvoiceCallback{
		ExternalID: payment.ExternalID, Status: "PAID",
		PaidAmount: 800, PaymentChannel: "GCASH", PaidAt: time.Now().Format(time.RFC3339),
	}
	if err := f.svc.HandleWebhook(context.Background(), cb); err != nil {
		t.Fatalf("webhook: %v", err)
	}

	sub, _ := f.repo.GetByTenant(context.Background(), "tenant-1")
	if sub.Status != billing.StatusActive {
		t.Errorf("status = %s, want active", sub.Status)
	}
	if until := time.Until(sub.CurrentPeriodEnd); until < 27*24*time.Hour || until > 32*24*time.Hour {
		t.Errorf("period end %v not ~1 month out", sub.CurrentPeriodEnd)
	}
	if f.repo.extends != 1 {
		t.Fatalf("extends = %d, want 1", f.repo.extends)
	}
	if f.cache.busts == 0 {
		t.Error("expected the status cache to be busted")
	}

	// Duplicate PAID and follow-up SETTLED must both be no-ops.
	if err := f.svc.HandleWebhook(context.Background(), cb); err != nil {
		t.Fatalf("duplicate webhook: %v", err)
	}
	cb.Status = "SETTLED"
	if err := f.svc.HandleWebhook(context.Background(), cb); err != nil {
		t.Fatalf("settled webhook: %v", err)
	}
	if f.repo.extends != 1 {
		t.Errorf("extends = %d after duplicates, want still 1", f.repo.extends)
	}
}

func TestWebhookUnknownAndExpired(t *testing.T) {
	f := newBillingFixture(t)
	f.seedSubscription(t, billing.StatusPending)

	if err := f.svc.HandleWebhook(context.Background(), xendit.InvoiceCallback{
		ExternalID: "sub_unknown", Status: "PAID",
	}); err != nil {
		t.Errorf("unknown external id must not error (Xendit would retry): %v", err)
	}

	checkout, _ := f.svc.CreateCheckout(context.Background(), "tenant-1", "user-1", "monthly")
	extID := externalIDOf(t, f.repo, checkout.PaymentID)
	if err := f.svc.HandleWebhook(context.Background(), xendit.InvoiceCallback{
		ExternalID: extID, Status: "EXPIRED",
	}); err != nil {
		t.Fatalf("expired webhook: %v", err)
	}
	if f.repo.payments[extID].Status != billing.PaymentExpired {
		t.Errorf("payment status = %s, want expired", f.repo.payments[extID].Status)
	}
	if f.repo.extends != 0 {
		t.Error("EXPIRED must never extend")
	}
}

func TestMarkPaidManualExtendsWithLedgerRow(t *testing.T) {
	f := newBillingFixture(t)
	f.seedSubscription(t, billing.StatusInactive)

	sub, err := f.svc.MarkPaidManual(context.Background(), "admin-1", "tenant-1", "bank transfer")
	if err != nil {
		t.Fatalf("MarkPaidManual: %v", err)
	}
	if sub.Status != billing.StatusActive {
		t.Errorf("status = %s, want active", sub.Status)
	}
	var manual *billing.Payment
	for _, p := range f.repo.payments {
		if p.Method == "manual" {
			manual = p
		}
	}
	if manual == nil {
		t.Fatal("expected a manual ledger row")
	}
	if manual.Status != billing.PaymentPaid || manual.Amount != 80000 || manual.Note != "bank transfer" {
		t.Errorf("manual payment wrong: %+v", manual)
	}
}

func TestSetSubscriptionStatusAndPricesValidation(t *testing.T) {
	f := newBillingFixture(t)
	f.seedSubscription(t, billing.StatusActive)

	if _, err := f.svc.SetSubscriptionStatus(context.Background(), "admin-1", "tenant-1", "pending"); err == nil {
		t.Error("forcing status=pending should be rejected")
	}
	sub, err := f.svc.SetSubscriptionStatus(context.Background(), "admin-1", "tenant-1", "inactive")
	if err != nil {
		t.Fatalf("SetSubscriptionStatus: %v", err)
	}
	if sub.Status != billing.StatusInactive {
		t.Errorf("status = %s, want inactive", sub.Status)
	}

	if _, err := f.svc.UpdatePrices(context.Background(), "admin-1", 0, 800000); err == nil {
		t.Error("zero price should be rejected")
	}
	updated, err := f.svc.UpdatePrices(context.Background(), "admin-1", 90000, 900000)
	if err != nil {
		t.Fatalf("UpdatePrices: %v", err)
	}
	if updated.MonthlyPrice != 90000 || updated.YearlyPrice != 900000 {
		t.Errorf("prices not updated: %+v", updated)
	}
}

func TestIsActiveUsesCache(t *testing.T) {
	f := newBillingFixture(t)
	f.seedSubscription(t, billing.StatusActive)

	active, err := f.svc.IsActive(context.Background(), "tenant-1")
	if err != nil || !active {
		t.Fatalf("IsActive = %v, %v; want true", active, err)
	}
	// Flip the repo directly — the cached value must still say active
	// until it is busted.
	f.repo.subs["tenant-1"].Status = billing.StatusInactive
	if active, _ := f.svc.IsActive(context.Background(), "tenant-1"); !active {
		t.Error("expected cached active status")
	}
	f.cache.DeletePrefix(context.Background(), "billing:sub:tenant-1")
	if active, _ := f.svc.IsActive(context.Background(), "tenant-1"); active {
		t.Error("expected fresh inactive status after bust")
	}
}

// externalIDOf finds a payment's external id by row id.
func externalIDOf(t *testing.T, repo *fakeBillingRepo, paymentID string) string {
	t.Helper()
	for extID, p := range repo.payments {
		if p.ID == paymentID {
			return extID
		}
	}
	t.Fatalf("payment %s not found", paymentID)
	return ""
}
