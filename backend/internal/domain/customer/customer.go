// Package customer defines customer profiles, membership tiers, and the
// loyalty points ledger. Points are integers; money is centavos.
package customer

import (
	"context"
	"time"
)

// Membership tiers in ascending order.
const (
	TierRegular = "regular"
	TierSilver  = "silver"
	TierGold    = "gold"
	TierVIP     = "vip"
)

// Loyalty transaction types.
const (
	TxEarn   = "earn"
	TxRedeem = "redeem"
	TxAdjust = "adjust"
)

type Customer struct {
	ID             string     `json:"id"`
	FullName       string     `json:"full_name"`
	Phone          string     `json:"phone"`
	Email          string     `json:"email"`
	Birthday       *time.Time `json:"birthday"`
	Notes          string     `json:"notes"`
	PointsBalance  int64      `json:"points_balance"`
	LifetimePoints int64      `json:"lifetime_points"`
	Tier           string     `json:"tier"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
}

// Settings is the tenant's loyalty program configuration.
type Settings struct {
	IsEnabled        bool    `json:"is_enabled"`
	EarnRate         int64   `json:"earn_rate"`     // centavos spent per point
	RedeemValue      int64   `json:"redeem_value"`  // centavos of value per point
	SilverThreshold  int64   `json:"silver_threshold"` // lifetime points
	GoldThreshold    int64   `json:"gold_threshold"`
	VIPThreshold     int64   `json:"vip_threshold"`
	SilverMultiplier float64 `json:"silver_multiplier"`
	GoldMultiplier   float64 `json:"gold_multiplier"`
	VIPMultiplier    float64 `json:"vip_multiplier"`
}

// TierFor maps lifetime points onto a tier.
func (s Settings) TierFor(lifetime int64) string {
	switch {
	case lifetime >= s.VIPThreshold:
		return TierVIP
	case lifetime >= s.GoldThreshold:
		return TierGold
	case lifetime >= s.SilverThreshold:
		return TierSilver
	default:
		return TierRegular
	}
}

// MultiplierFor returns the earn multiplier for a tier.
func (s Settings) MultiplierFor(tier string) float64 {
	switch tier {
	case TierVIP:
		return s.VIPMultiplier
	case TierGold:
		return s.GoldMultiplier
	case TierSilver:
		return s.SilverMultiplier
	default:
		return 1
	}
}

type Transaction struct {
	ID           string    `json:"id"`
	CustomerID   string    `json:"customer_id"`
	OrderID      *string   `json:"order_id"`
	OrderNumber  *int64    `json:"order_number,omitempty"`
	Type         string    `json:"type"`
	Points       int64     `json:"points"`
	BalanceAfter int64     `json:"balance_after"`
	Notes        string    `json:"notes"`
	CreatedBy    string    `json:"-"` // empty for system-generated rows
	CreatedAt    time.Time `json:"created_at"`
}

type Repository interface {
	Create(ctx context.Context, tenantID string, c *Customer) error
	GetByID(ctx context.Context, tenantID, id string) (*Customer, error)
	List(ctx context.Context, tenantID, search string) ([]Customer, error)
	Update(ctx context.Context, tenantID string, c *Customer) error
	SoftDelete(ctx context.Context, tenantID, id string) error
	UpdateTier(ctx context.Context, tenantID, id, tier string) error

	GetSettings(ctx context.Context, tenantID string) (*Settings, error)
	SaveSettings(ctx context.Context, tenantID string, s *Settings) error

	// ApplyPoints atomically moves the balance (rejecting overdrafts) and
	// appends the ledger row with balance_after in one transaction.
	ApplyPoints(ctx context.Context, tenantID string, tx *Transaction) error
	ListTransactions(ctx context.Context, tenantID, customerID string, limit int) ([]Transaction, error)
	ListTransactionsByOrder(ctx context.Context, tenantID, orderID string) ([]Transaction, error)
}
