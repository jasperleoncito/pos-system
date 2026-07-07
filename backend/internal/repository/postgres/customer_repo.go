package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/customer"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

type CustomerRepo struct {
	db *pgxpool.Pool
}

func NewCustomerRepo(db *pgxpool.Pool) *CustomerRepo { return &CustomerRepo{db: db} }

// ---- customers ----

const customerColumns = `
	id, full_name, phone, email, birthday, notes, points_balance, lifetime_points, tier, is_active, created_at`

func scanCustomer(row pgx.Row) (*customer.Customer, error) {
	var c customer.Customer
	err := row.Scan(&c.ID, &c.FullName, &c.Phone, &c.Email, &c.Birthday, &c.Notes,
		&c.PointsBalance, &c.LifetimePoints, &c.Tier, &c.IsActive, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("customer")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan customer: %w", err)
	}
	return &c, nil
}

func (r *CustomerRepo) Create(ctx context.Context, tenantID string, c *customer.Customer) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO customers (tenant_id, full_name, phone, email, birthday, notes, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, tier, created_at`,
		tenantID, c.FullName, c.Phone, c.Email, c.Birthday, c.Notes, c.IsActive,
	).Scan(&c.ID, &c.Tier, &c.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("a customer with that phone number already exists")
		}
		return fmt.Errorf("failed to create customer: %w", err)
	}
	return nil
}

func (r *CustomerRepo) GetByID(ctx context.Context, tenantID, id string) (*customer.Customer, error) {
	return scanCustomer(r.db.QueryRow(ctx, `
		SELECT `+customerColumns+` FROM customers
		WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, id))
}

func (r *CustomerRepo) List(ctx context.Context, tenantID, search string) ([]customer.Customer, error) {
	query := `SELECT ` + customerColumns + ` FROM customers WHERE tenant_id=$1 AND deleted_at IS NULL`
	args := []any{tenantID}
	if search != "" {
		query += ` AND (full_name ILIKE $2 OR phone ILIKE $2 OR email ILIKE $2)`
		args = append(args, "%"+search+"%")
	}
	query += ` ORDER BY full_name LIMIT 200`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list customers: %w", err)
	}
	defer rows.Close()
	var customers []customer.Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, err
		}
		customers = append(customers, *c)
	}
	return customers, rows.Err()
}

func (r *CustomerRepo) Update(ctx context.Context, tenantID string, c *customer.Customer) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE customers SET full_name=$3, phone=$4, email=$5, birthday=$6, notes=$7, is_active=$8, updated_at=now()
		WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`,
		tenantID, c.ID, c.FullName, c.Phone, c.Email, c.Birthday, c.Notes, c.IsActive)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("a customer with that phone number already exists")
		}
		return fmt.Errorf("failed to update customer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("customer")
	}
	return nil
}

func (r *CustomerRepo) SoftDelete(ctx context.Context, tenantID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE customers SET deleted_at=now(), updated_at=now()
		WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("customer")
	}
	return nil
}

func (r *CustomerRepo) UpdateTier(ctx context.Context, tenantID, id, tier string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE customers SET tier=$3, updated_at=now()
		WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, id, tier)
	if err != nil {
		return fmt.Errorf("failed to update tier: %w", err)
	}
	return nil
}

// ---- loyalty settings ----

func (r *CustomerRepo) GetSettings(ctx context.Context, tenantID string) (*customer.Settings, error) {
	var s customer.Settings
	err := r.db.QueryRow(ctx, `
		SELECT is_enabled, earn_rate, redeem_value, silver_threshold, gold_threshold, vip_threshold,
			silver_multiplier, gold_multiplier, vip_multiplier
		FROM loyalty_settings WHERE tenant_id=$1`, tenantID,
	).Scan(&s.IsEnabled, &s.EarnRate, &s.RedeemValue, &s.SilverThreshold, &s.GoldThreshold, &s.VIPThreshold,
		&s.SilverMultiplier, &s.GoldMultiplier, &s.VIPMultiplier)
	if errors.Is(err, pgx.ErrNoRows) {
		// Program defaults until the tenant saves their own.
		return &customer.Settings{
			IsEnabled: true, EarnRate: 5000, RedeemValue: 100,
			SilverThreshold: 500, GoldThreshold: 1500, VIPThreshold: 4000,
			SilverMultiplier: 1.25, GoldMultiplier: 1.5, VIPMultiplier: 2,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get loyalty settings: %w", err)
	}
	return &s, nil
}

func (r *CustomerRepo) SaveSettings(ctx context.Context, tenantID string, s *customer.Settings) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO loyalty_settings (tenant_id, is_enabled, earn_rate, redeem_value,
			silver_threshold, gold_threshold, vip_threshold, silver_multiplier, gold_multiplier, vip_multiplier)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (tenant_id) DO UPDATE SET
			is_enabled=$2, earn_rate=$3, redeem_value=$4, silver_threshold=$5, gold_threshold=$6,
			vip_threshold=$7, silver_multiplier=$8, gold_multiplier=$9, vip_multiplier=$10, updated_at=now()`,
		tenantID, s.IsEnabled, s.EarnRate, s.RedeemValue,
		s.SilverThreshold, s.GoldThreshold, s.VIPThreshold,
		s.SilverMultiplier, s.GoldMultiplier, s.VIPMultiplier)
	if err != nil {
		return fmt.Errorf("failed to save loyalty settings: %w", err)
	}
	return nil
}

// ---- points ledger ----

// ApplyPoints moves the balance and appends the ledger row atomically.
// The balance CHECK plus the guarded UPDATE reject overdrafts.
func (r *CustomerRepo) ApplyPoints(ctx context.Context, tenantID string, t *customer.Transaction) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lifetime points (tier progress) grow on earns and shrink when an
	// earn is reversed (negative adjust); redemptions leave them alone.
	var balanceAfter int64
	err = tx.QueryRow(ctx, `
		UPDATE customers
		SET points_balance = points_balance + $3,
			lifetime_points = GREATEST(lifetime_points + CASE
				WHEN $4 = 'earn' THEN $3
				WHEN $4 = 'adjust' AND $3 < 0 THEN $3
				ELSE 0 END, 0),
			updated_at = now()
		WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL AND points_balance + $3 >= 0
		RETURNING points_balance`,
		tenantID, t.CustomerID, t.Points, t.Type).Scan(&balanceAfter)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.Validation("the customer does not have enough points")
	}
	if err != nil {
		return fmt.Errorf("failed to move points balance: %w", err)
	}

	t.BalanceAfter = balanceAfter
	var createdBy any
	if t.CreatedBy != "" {
		createdBy = t.CreatedBy
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO loyalty_transactions (tenant_id, customer_id, order_id, type, points, balance_after, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at`,
		tenantID, t.CustomerID, t.OrderID, t.Type, t.Points, t.BalanceAfter, t.Notes, createdBy).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert loyalty transaction: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *CustomerRepo) ListTransactions(ctx context.Context, tenantID, customerID string, limit int) ([]customer.Transaction, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.customer_id, t.order_id, o.order_number, t.type, t.points, t.balance_after, t.notes, t.created_at
		FROM loyalty_transactions t
		LEFT JOIN orders o ON o.id = t.order_id
		WHERE t.tenant_id=$1 AND t.customer_id=$2
		ORDER BY t.created_at DESC LIMIT $3`, tenantID, customerID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list loyalty transactions: %w", err)
	}
	defer rows.Close()
	return scanTransactions(rows)
}

func (r *CustomerRepo) ListTransactionsByOrder(ctx context.Context, tenantID, orderID string) ([]customer.Transaction, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.customer_id, t.order_id, o.order_number, t.type, t.points, t.balance_after, t.notes, t.created_at
		FROM loyalty_transactions t
		LEFT JOIN orders o ON o.id = t.order_id
		WHERE t.tenant_id=$1 AND t.order_id=$2 ORDER BY t.created_at`, tenantID, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to list order loyalty transactions: %w", err)
	}
	defer rows.Close()
	return scanTransactions(rows)
}

func scanTransactions(rows pgx.Rows) ([]customer.Transaction, error) {
	var txs []customer.Transaction
	for rows.Next() {
		var t customer.Transaction
		if err := rows.Scan(&t.ID, &t.CustomerID, &t.OrderID, &t.OrderNumber, &t.Type, &t.Points, &t.BalanceAfter, &t.Notes, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan loyalty transaction: %w", err)
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}
