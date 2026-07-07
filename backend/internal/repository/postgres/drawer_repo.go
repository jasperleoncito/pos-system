package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/order"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

type DrawerRepo struct {
	db *pgxpool.Pool
}

func NewDrawerRepo(db *pgxpool.Pool) *DrawerRepo { return &DrawerRepo{db: db} }

func (r *DrawerRepo) Open(ctx context.Context, tenantID string, s *order.DrawerSession) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO cash_drawer_sessions (tenant_id, opened_by, opening_float, expected_cash)
		VALUES ($1, $2, $3, $3)
		RETURNING id, status, opened_at`,
		tenantID, s.OpenedBy, s.OpeningFloat,
	).Scan(&s.ID, &s.Status, &s.OpenedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("a cash drawer is already open — close it first")
		}
		return fmt.Errorf("failed to open drawer: %w", err)
	}
	return nil
}

func (r *DrawerRepo) Current(ctx context.Context, tenantID string) (*order.DrawerSession, error) {
	var s order.DrawerSession
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, opened_by, closed_by, opening_float, expected_cash, counted_cash,
		       variance, status, opened_at, closed_at
		FROM cash_drawer_sessions
		WHERE tenant_id = $1 AND status = 'open' AND deleted_at IS NULL`, tenantID,
	).Scan(&s.ID, &s.TenantID, &s.OpenedBy, &s.ClosedBy, &s.OpeningFloat, &s.ExpectedCash,
		&s.CountedCash, &s.Variance, &s.Status, &s.OpenedAt, &s.ClosedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("open cash drawer")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get current drawer: %w", err)
	}
	return &s, nil
}

func (r *DrawerRepo) Close(ctx context.Context, tenantID, sessionID, closedBy string, countedCash int64) (*order.DrawerSession, error) {
	var s order.DrawerSession
	err := r.db.QueryRow(ctx, `
		UPDATE cash_drawer_sessions
		SET status = 'closed', closed_by = $3, counted_cash = $4,
		    variance = $4 - expected_cash, closed_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND status = 'open' AND deleted_at IS NULL
		RETURNING id, tenant_id, opened_by, closed_by, opening_float, expected_cash, counted_cash,
		          variance, status, opened_at, closed_at`,
		tenantID, sessionID, closedBy, countedCash,
	).Scan(&s.ID, &s.TenantID, &s.OpenedBy, &s.ClosedBy, &s.OpeningFloat, &s.ExpectedCash,
		&s.CountedCash, &s.Variance, &s.Status, &s.OpenedAt, &s.ClosedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("open cash drawer")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to close drawer: %w", err)
	}
	return &s, nil
}

// AddMovement records the movement and keeps expected_cash in sync.
func (r *DrawerRepo) AddMovement(ctx context.Context, tenantID string, m *order.CashMovement) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO cash_movements (tenant_id, session_id, type, amount, order_id, reason, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`,
		tenantID, m.SessionID, m.Type, m.Amount, m.OrderID, m.Reason, m.CreatedBy,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to add cash movement: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE cash_drawer_sessions SET expected_cash = expected_cash + $3, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND status = 'open'`,
		tenantID, m.SessionID, m.Amount); err != nil {
		return fmt.Errorf("failed to update expected cash: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *DrawerRepo) ListMovements(ctx context.Context, tenantID, sessionID string) ([]order.CashMovement, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, session_id, type, amount, order_id, reason, created_by, created_at
		FROM cash_movements
		WHERE tenant_id = $1 AND session_id = $2
		ORDER BY created_at`, tenantID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cash movements: %w", err)
	}
	defer rows.Close()

	var movements []order.CashMovement
	for rows.Next() {
		var m order.CashMovement
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Type, &m.Amount, &m.OrderID, &m.Reason, &m.CreatedBy, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan cash movement: %w", err)
		}
		movements = append(movements, m)
	}
	return movements, rows.Err()
}
