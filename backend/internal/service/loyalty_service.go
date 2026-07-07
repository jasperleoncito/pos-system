package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/customer"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

// LoyaltyService owns customers, membership tiers, and the points ledger.
type LoyaltyService struct {
	repo    customer.Repository
	auditor *AuditService
	logger  *slog.Logger
}

func NewLoyaltyService(repo customer.Repository, auditor *AuditService, logger *slog.Logger) *LoyaltyService {
	return &LoyaltyService{repo: repo, auditor: auditor, logger: logger}
}

// ---- customers ----

func (s *LoyaltyService) CreateCustomer(ctx context.Context, tenantID, userID string, c *customer.Customer) (*customer.Customer, error) {
	if err := s.repo.Create(ctx, tenantID, c); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "customer.created",
		EntityType: "customer", EntityID: c.ID, After: map[string]any{"full_name": c.FullName}})
	return s.repo.GetByID(ctx, tenantID, c.ID)
}

func (s *LoyaltyService) ListCustomers(ctx context.Context, tenantID, search string) ([]customer.Customer, error) {
	return s.repo.List(ctx, tenantID, search)
}

func (s *LoyaltyService) GetCustomer(ctx context.Context, tenantID, id string) (*customer.Customer, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *LoyaltyService) UpdateCustomer(ctx context.Context, tenantID, userID string, c *customer.Customer) (*customer.Customer, error) {
	if err := s.repo.Update(ctx, tenantID, c); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "customer.updated",
		EntityType: "customer", EntityID: c.ID, After: map[string]any{"full_name": c.FullName}})
	return s.repo.GetByID(ctx, tenantID, c.ID)
}

func (s *LoyaltyService) DeleteCustomer(ctx context.Context, tenantID, userID, id string) error {
	if err := s.repo.SoftDelete(ctx, tenantID, id); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "customer.deleted",
		EntityType: "customer", EntityID: id})
	return nil
}

func (s *LoyaltyService) ListTransactions(ctx context.Context, tenantID, customerID string, limit int) ([]customer.Transaction, error) {
	if _, err := s.repo.GetByID(ctx, tenantID, customerID); err != nil {
		return nil, err
	}
	return s.repo.ListTransactions(ctx, tenantID, customerID, limit)
}

// ---- settings ----

func (s *LoyaltyService) GetSettings(ctx context.Context, tenantID string) (*customer.Settings, error) {
	return s.repo.GetSettings(ctx, tenantID)
}

func (s *LoyaltyService) SaveSettings(ctx context.Context, tenantID, userID string, in *customer.Settings) (*customer.Settings, error) {
	if in.EarnRate <= 0 || in.RedeemValue <= 0 {
		return nil, apperror.Validation("earn rate and redemption value must be positive")
	}
	if in.SilverThreshold < 0 || in.GoldThreshold < in.SilverThreshold || in.VIPThreshold < in.GoldThreshold {
		return nil, apperror.Validation("tier thresholds must ascend: silver ≤ gold ≤ vip")
	}
	if in.SilverMultiplier < 1 || in.GoldMultiplier < in.SilverMultiplier || in.VIPMultiplier < in.GoldMultiplier {
		return nil, apperror.Validation("tier multipliers must ascend and be at least 1")
	}
	if err := s.repo.SaveSettings(ctx, tenantID, in); err != nil {
		return nil, apperror.Internal(err)
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "loyalty.settings_updated",
		EntityType: "loyalty_settings", EntityID: tenantID,
		After: map[string]any{"earn_rate": in.EarnRate, "redeem_value": in.RedeemValue, "enabled": in.IsEnabled}})
	return s.repo.GetSettings(ctx, tenantID)
}

// ---- points math used by the POS ----

// RedeemForPayment converts a payment amount (centavos) into points and
// deducts them. Called inside the pay flow before payments are recorded.
func (s *LoyaltyService) RedeemForPayment(ctx context.Context, tenantID, userID, customerID, orderID string, amount int64) (int64, error) {
	settings, err := s.repo.GetSettings(ctx, tenantID)
	if err != nil {
		return 0, apperror.Internal(err)
	}
	if !settings.IsEnabled {
		return 0, apperror.Validation("the loyalty program is disabled")
	}
	points := int64(math.Ceil(float64(amount) / float64(settings.RedeemValue)))
	if points <= 0 {
		return 0, apperror.Validation("redemption amount is too small")
	}
	err = s.repo.ApplyPoints(ctx, tenantID, &customer.Transaction{
		CustomerID: customerID, OrderID: &orderID, Type: customer.TxRedeem,
		Points: -points, Notes: fmt.Sprintf("redeemed as payment (%d centavos)", amount), CreatedBy: userID,
	})
	if err != nil {
		return 0, err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "loyalty.redeemed",
		EntityType: "customer", EntityID: customerID,
		After: map[string]any{"points": points, "value": amount, "order_id": orderID}})
	return points, nil
}

// AwardForOrder credits earned points after an order completes: the paid
// amount excluding points value, divided by the earn rate, times the
// customer's tier multiplier. Failures log — they never block a sale.
func (s *LoyaltyService) AwardForOrder(ctx context.Context, tenantID, userID, customerID, orderID string, earnBase int64) {
	settings, err := s.repo.GetSettings(ctx, tenantID)
	if err != nil || !settings.IsEnabled || earnBase <= 0 {
		return
	}
	c, err := s.repo.GetByID(ctx, tenantID, customerID)
	if err != nil {
		s.logger.Warn("loyalty award: customer lookup failed", "customer_id", customerID, "error", err)
		return
	}
	points := int64(math.Floor(float64(earnBase) / float64(settings.EarnRate) * settings.MultiplierFor(c.Tier)))
	if points <= 0 {
		return
	}
	if err := s.repo.ApplyPoints(ctx, tenantID, &customer.Transaction{
		CustomerID: customerID, OrderID: &orderID, Type: customer.TxEarn,
		Points: points, Notes: "earned from purchase", CreatedBy: userID,
	}); err != nil {
		s.logger.Error("loyalty award failed", "customer_id", customerID, "order_id", orderID, "error", err)
		return
	}
	s.refreshTier(ctx, tenantID, customerID, settings)
}

// refreshTier upgrades the customer's tier when lifetime points cross a
// threshold. Tiers never downgrade automatically.
func (s *LoyaltyService) refreshTier(ctx context.Context, tenantID, customerID string, settings *customer.Settings) {
	c, err := s.repo.GetByID(ctx, tenantID, customerID)
	if err != nil {
		return
	}
	target := settings.TierFor(c.LifetimePoints)
	if tierRank(target) > tierRank(c.Tier) {
		if err := s.repo.UpdateTier(ctx, tenantID, customerID, target); err != nil {
			s.logger.Warn("tier upgrade failed", "customer_id", customerID, "error", err)
			return
		}
		s.auditor.Record(audit.Log{TenantID: tenantID, Action: "loyalty.tier_upgraded",
			EntityType: "customer", EntityID: customerID,
			Before: map[string]any{"tier": c.Tier}, After: map[string]any{"tier": target}})
	}
}

func tierRank(t string) int {
	switch t {
	case customer.TierSilver:
		return 1
	case customer.TierGold:
		return 2
	case customer.TierVIP:
		return 3
	default:
		return 0
	}
}

// ReverseForOrder undoes an order's loyalty activity on void: earned
// points come back out, redeemed points go back to the customer.
func (s *LoyaltyService) ReverseForOrder(ctx context.Context, tenantID, userID, orderID string) {
	txs, err := s.repo.ListTransactionsByOrder(ctx, tenantID, orderID)
	if err != nil {
		s.logger.Warn("loyalty reversal: listing failed", "order_id", orderID, "error", err)
		return
	}
	var reversed int64
	for _, t := range txs {
		if t.Type == customer.TxAdjust {
			continue // already a reversal
		}
		if err := s.repo.ApplyPoints(ctx, tenantID, &customer.Transaction{
			CustomerID: t.CustomerID, OrderID: &orderID, Type: customer.TxAdjust,
			Points: -t.Points, Notes: "order voided — " + t.Type + " reversed", CreatedBy: userID,
		}); err != nil {
			s.logger.Error("loyalty reversal failed", "order_id", orderID, "tx_id", t.ID, "error", err)
			continue
		}
		reversed += t.Points
	}
	if reversed != 0 {
		s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "loyalty.reversed",
			EntityType: "order", EntityID: orderID, After: map[string]any{"points_reversed": reversed}})
	}
}
