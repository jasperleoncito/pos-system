package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/catalog"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

type TaxRepo struct {
	db *pgxpool.Pool
}

func NewTaxRepo(db *pgxpool.Pool) *TaxRepo { return &TaxRepo{db: db} }

const taxColumns = `id, tenant_id, name, rate_percent, is_inclusive, is_default, is_active, created_at, updated_at`

func scanTax(row pgx.Row) (*catalog.Tax, error) {
	var t catalog.Tax
	err := row.Scan(&t.ID, &t.TenantID, &t.Name, &t.RatePercent, &t.IsInclusive, &t.IsDefault,
		&t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("tax")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan tax: %w", err)
	}
	return &t, nil
}

func (r *TaxRepo) Create(ctx context.Context, tenantID string, t *catalog.Tax) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO taxes (tenant_id, name, rate_percent, is_inclusive, is_default, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, created_at, updated_at`,
		tenantID, t.Name, t.RatePercent, t.IsInclusive, t.IsDefault, t.IsActive,
	).Scan(&t.ID, &t.TenantID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create tax: %w", err)
	}
	return nil
}

func (r *TaxRepo) GetByID(ctx context.Context, tenantID, id string) (*catalog.Tax, error) {
	return scanTax(r.db.QueryRow(ctx,
		`SELECT `+taxColumns+` FROM taxes WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id))
}

func (r *TaxRepo) List(ctx context.Context, tenantID string) ([]catalog.Tax, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+taxColumns+` FROM taxes WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY name`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list taxes: %w", err)
	}
	defer rows.Close()

	var taxes []catalog.Tax
	for rows.Next() {
		t, err := scanTax(rows)
		if err != nil {
			return nil, err
		}
		taxes = append(taxes, *t)
	}
	return taxes, rows.Err()
}

func (r *TaxRepo) Update(ctx context.Context, tenantID string, t *catalog.Tax) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE taxes SET name = $3, rate_percent = $4, is_inclusive = $5, is_default = $6, is_active = $7, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, t.ID, t.Name, t.RatePercent, t.IsInclusive, t.IsDefault, t.IsActive)
	if err != nil {
		return fmt.Errorf("failed to update tax: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("tax")
	}
	return nil
}

func (r *TaxRepo) SoftDelete(ctx context.Context, tenantID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE taxes SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to delete tax: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("tax")
	}
	return nil
}
