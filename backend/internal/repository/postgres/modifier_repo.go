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

type ModifierRepo struct {
	db *pgxpool.Pool
}

func NewModifierRepo(db *pgxpool.Pool) *ModifierRepo { return &ModifierRepo{db: db} }

func (r *ModifierRepo) CreateGroup(ctx context.Context, tenantID string, g *catalog.ModifierGroup) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO modifier_groups (tenant_id, name, min_select, max_select, is_required, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id`,
		tenantID, g.Name, g.MinSelect, g.MaxSelect, g.IsRequired, g.SortOrder,
	).Scan(&g.ID, &g.TenantID)
	if err != nil {
		return fmt.Errorf("failed to create modifier group: %w", err)
	}
	return nil
}

func (r *ModifierRepo) GetGroup(ctx context.Context, tenantID, id string) (*catalog.ModifierGroup, error) {
	var g catalog.ModifierGroup
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, min_select, max_select, is_required, sort_order
		FROM modifier_groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id,
	).Scan(&g.ID, &g.TenantID, &g.Name, &g.MinSelect, &g.MaxSelect, &g.IsRequired, &g.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("modifier group")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get modifier group: %w", err)
	}

	modifiers, err := r.listModifiers(ctx, tenantID, []string{g.ID})
	if err != nil {
		return nil, err
	}
	g.Modifiers = modifiers[g.ID]
	return &g, nil
}

func (r *ModifierRepo) ListGroups(ctx context.Context, tenantID string) ([]catalog.ModifierGroup, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, min_select, max_select, is_required, sort_order
		FROM modifier_groups WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY sort_order, name`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list modifier groups: %w", err)
	}
	defer rows.Close()

	var groups []catalog.ModifierGroup
	var ids []string
	for rows.Next() {
		var g catalog.ModifierGroup
		if err := rows.Scan(&g.ID, &g.TenantID, &g.Name, &g.MinSelect, &g.MaxSelect, &g.IsRequired, &g.SortOrder); err != nil {
			return nil, fmt.Errorf("failed to scan modifier group: %w", err)
		}
		groups = append(groups, g)
		ids = append(ids, g.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	modifiersByGroup, err := r.listModifiers(ctx, tenantID, ids)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		groups[i].Modifiers = modifiersByGroup[groups[i].ID]
	}
	return groups, nil
}

func (r *ModifierRepo) listModifiers(ctx context.Context, tenantID string, groupIDs []string) (map[string][]catalog.Modifier, error) {
	result := make(map[string][]catalog.Modifier)
	if len(groupIDs) == 0 {
		return result, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, group_id, name, price_delta, is_active, sort_order
		FROM modifiers WHERE tenant_id = $1 AND group_id = ANY($2) AND deleted_at IS NULL
		ORDER BY sort_order, name`, tenantID, groupIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to list modifiers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var m catalog.Modifier
		if err := rows.Scan(&m.ID, &m.GroupID, &m.Name, &m.PriceDelta, &m.IsActive, &m.SortOrder); err != nil {
			return nil, fmt.Errorf("failed to scan modifier: %w", err)
		}
		result[m.GroupID] = append(result[m.GroupID], m)
	}
	return result, rows.Err()
}

func (r *ModifierRepo) UpdateGroup(ctx context.Context, tenantID string, g *catalog.ModifierGroup) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE modifier_groups SET name = $3, min_select = $4, max_select = $5, is_required = $6, sort_order = $7, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, g.ID, g.Name, g.MinSelect, g.MaxSelect, g.IsRequired, g.SortOrder)
	if err != nil {
		return fmt.Errorf("failed to update modifier group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("modifier group")
	}
	return nil
}

func (r *ModifierRepo) SoftDeleteGroup(ctx context.Context, tenantID, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE modifier_groups SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to delete modifier group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("modifier group")
	}
	if _, err := tx.Exec(ctx, `
		UPDATE modifiers SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND group_id = $2 AND deleted_at IS NULL`, tenantID, id); err != nil {
		return fmt.Errorf("failed to delete group modifiers: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE product_modifier_groups SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND modifier_group_id = $2 AND deleted_at IS NULL`, tenantID, id); err != nil {
		return fmt.Errorf("failed to unlink group from products: %w", err)
	}
	return tx.Commit(ctx)
}

// ReplaceModifiers soft-deletes existing options and inserts the new set.
func (r *ModifierRepo) ReplaceModifiers(ctx context.Context, tenantID, groupID string, modifiers []catalog.Modifier) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE modifiers SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND group_id = $2 AND deleted_at IS NULL`, tenantID, groupID); err != nil {
		return fmt.Errorf("failed to clear modifiers: %w", err)
	}
	for i, m := range modifiers {
		if _, err := tx.Exec(ctx, `
			INSERT INTO modifiers (tenant_id, group_id, name, price_delta, is_active, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			tenantID, groupID, m.Name, m.PriceDelta, m.IsActive, i); err != nil {
			return fmt.Errorf("failed to insert modifier: %w", err)
		}
	}
	return tx.Commit(ctx)
}
