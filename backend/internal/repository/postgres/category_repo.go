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

type CategoryRepo struct {
	db *pgxpool.Pool
}

func NewCategoryRepo(db *pgxpool.Pool) *CategoryRepo { return &CategoryRepo{db: db} }

const categoryColumns = `id, tenant_id, name, description, sort_order, image_key, is_active, created_at, updated_at`

func scanCategory(row pgx.Row) (*catalog.Category, error) {
	var c catalog.Category
	err := row.Scan(&c.ID, &c.TenantID, &c.Name, &c.Description, &c.SortOrder, &c.ImageKey,
		&c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("category")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan category: %w", err)
	}
	return &c, nil
}

func (r *CategoryRepo) Create(ctx context.Context, tenantID string, c *catalog.Category) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO categories (tenant_id, name, description, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, tenant_id, created_at, updated_at`,
		tenantID, c.Name, c.Description, c.SortOrder, c.IsActive,
	).Scan(&c.ID, &c.TenantID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("a category with that name already exists")
		}
		return fmt.Errorf("failed to create category: %w", err)
	}
	return nil
}

func (r *CategoryRepo) GetByID(ctx context.Context, tenantID, id string) (*catalog.Category, error) {
	return scanCategory(r.db.QueryRow(ctx,
		`SELECT `+categoryColumns+` FROM categories WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id))
}

func (r *CategoryRepo) List(ctx context.Context, tenantID string, activeOnly bool) ([]catalog.Category, error) {
	query := `SELECT ` + categoryColumns + ` FROM categories WHERE tenant_id = $1 AND deleted_at IS NULL`
	if activeOnly {
		query += ` AND is_active`
	}
	query += ` ORDER BY sort_order, name`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	defer rows.Close()

	var categories []catalog.Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		categories = append(categories, *c)
	}
	return categories, rows.Err()
}

func (r *CategoryRepo) Update(ctx context.Context, tenantID string, c *catalog.Category) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE categories SET name = $3, description = $4, sort_order = $5, is_active = $6, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, c.ID, c.Name, c.Description, c.SortOrder, c.IsActive)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("a category with that name already exists")
		}
		return fmt.Errorf("failed to update category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("category")
	}
	return nil
}

func (r *CategoryRepo) SoftDelete(ctx context.Context, tenantID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE categories SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("category")
	}
	return nil
}
