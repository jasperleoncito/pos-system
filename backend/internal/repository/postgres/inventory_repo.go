package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/inventory"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

type InventoryRepo struct {
	db *pgxpool.Pool
}

func NewInventoryRepo(db *pgxpool.Pool) *InventoryRepo { return &InventoryRepo{db: db} }

// ---- units ----

func (r *InventoryRepo) CreateUnit(ctx context.Context, tenantID string, u *inventory.Unit) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO units (tenant_id, name, abbreviation) VALUES ($1, $2, $3) RETURNING id`,
		tenantID, u.Name, u.Abbreviation).Scan(&u.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("a unit with that name already exists")
		}
		return fmt.Errorf("failed to create unit: %w", err)
	}
	return nil
}

func (r *InventoryRepo) ListUnits(ctx context.Context, tenantID string) ([]inventory.Unit, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, abbreviation FROM units WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY name`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list units: %w", err)
	}
	defer rows.Close()
	var units []inventory.Unit
	for rows.Next() {
		var u inventory.Unit
		if err := rows.Scan(&u.ID, &u.Name, &u.Abbreviation); err != nil {
			return nil, fmt.Errorf("failed to scan unit: %w", err)
		}
		units = append(units, u)
	}
	return units, rows.Err()
}

// ---- items ----

const itemColumns = `i.id, i.tenant_id, i.name, i.type, i.unit_id, u.abbreviation, i.current_stock,
	i.reorder_level, i.cost_per_unit, i.is_active, i.created_at, i.updated_at`

func scanItem(row pgx.Row) (*inventory.Item, error) {
	var it inventory.Item
	err := row.Scan(&it.ID, &it.TenantID, &it.Name, &it.Type, &it.UnitID, &it.UnitAbbr, &it.CurrentStock,
		&it.ReorderLevel, &it.CostPerUnit, &it.IsActive, &it.CreatedAt, &it.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("inventory item")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan inventory item: %w", err)
	}
	return &it, nil
}

func (r *InventoryRepo) CreateItem(ctx context.Context, tenantID string, i *inventory.Item) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO inventory_items (tenant_id, name, type, unit_id, current_stock, reorder_level, cost_per_unit, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, tenant_id, created_at, updated_at`,
		tenantID, i.Name, i.Type, i.UnitID, i.CurrentStock, i.ReorderLevel, i.CostPerUnit, i.IsActive,
	).Scan(&i.ID, &i.TenantID, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("an item with that name already exists")
		}
		return fmt.Errorf("failed to create inventory item: %w", err)
	}
	return nil
}

func (r *InventoryRepo) GetItem(ctx context.Context, tenantID, id string) (*inventory.Item, error) {
	return scanItem(r.db.QueryRow(ctx, `
		SELECT `+itemColumns+` FROM inventory_items i JOIN units u ON u.id = i.unit_id
		WHERE i.tenant_id = $1 AND i.id = $2 AND i.deleted_at IS NULL`, tenantID, id))
}

func (r *InventoryRepo) GetItemByName(ctx context.Context, tenantID, name string) (*inventory.Item, error) {
	return scanItem(r.db.QueryRow(ctx, `
		SELECT `+itemColumns+` FROM inventory_items i JOIN units u ON u.id = i.unit_id
		WHERE i.tenant_id = $1 AND lower(i.name) = lower($2) AND i.deleted_at IS NULL`, tenantID, name))
}

func (r *InventoryRepo) ListItems(ctx context.Context, tenantID, search string) ([]inventory.Item, error) {
	query := `SELECT ` + itemColumns + ` FROM inventory_items i JOIN units u ON u.id = i.unit_id
		WHERE i.tenant_id = $1 AND i.deleted_at IS NULL`
	args := []any{tenantID}
	if search != "" {
		args = append(args, "%"+search+"%")
		query += ` AND i.name ILIKE $2`
	}
	query += ` ORDER BY i.name`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory items: %w", err)
	}
	defer rows.Close()
	var items []inventory.Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *it)
	}
	return items, rows.Err()
}

func (r *InventoryRepo) UpdateItem(ctx context.Context, tenantID string, i *inventory.Item) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE inventory_items SET name = $3, type = $4, unit_id = $5, reorder_level = $6,
			cost_per_unit = $7, is_active = $8, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, i.ID, i.Name, i.Type, i.UnitID, i.ReorderLevel, i.CostPerUnit, i.IsActive)
	if err != nil {
		return fmt.Errorf("failed to update inventory item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("inventory item")
	}
	return nil
}

func (r *InventoryRepo) SoftDeleteItem(ctx context.Context, tenantID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE inventory_items SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to delete inventory item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("inventory item")
	}
	return nil
}

// ---- ledger ----

// Apply locks the item row (FOR UPDATE) so concurrent movements
// serialize; the ledger row records the exact before/after chain.
func (r *InventoryRepo) Apply(ctx context.Context, tenantID string, in inventory.ApplyInput) (*inventory.Movement, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var before float64
	err = tx.QueryRow(ctx, `
		SELECT current_stock FROM inventory_items
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
		FOR UPDATE`, tenantID, in.ItemID).Scan(&before)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("inventory item")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to lock item: %w", err)
	}

	after := before + in.QtyDelta

	if _, err := tx.Exec(ctx, `
		UPDATE inventory_items SET current_stock = $3, updated_at = now()
		WHERE tenant_id = $1 AND id = $2`, tenantID, in.ItemID, after); err != nil {
		return nil, fmt.Errorf("failed to update stock: %w", err)
	}

	m := &inventory.Movement{
		ItemID: in.ItemID, MovementType: in.MovementType, QtyDelta: in.QtyDelta,
		QtyBefore: before, QtyAfter: after, UnitCost: in.UnitCost,
		ReferenceType: in.ReferenceType, ReferenceID: in.ReferenceID,
		Notes: in.Notes, PerformedBy: in.PerformedBy,
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO inventory_movements (tenant_id, item_id, movement_type, qty_delta, qty_before,
			qty_after, unit_cost, reference_type, reference_id, notes, performed_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at`,
		tenantID, in.ItemID, in.MovementType, in.QtyDelta, before, after, in.UnitCost,
		in.ReferenceType, in.ReferenceID, in.Notes, in.PerformedBy,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert movement: %w", err)
	}

	return m, tx.Commit(ctx)
}

func (r *InventoryRepo) ListMovements(ctx context.Context, tenantID, itemID string, limit, offset int) ([]inventory.Movement, int64, error) {
	where := `m.tenant_id = $1`
	args := []any{tenantID}
	if itemID != "" {
		args = append(args, itemID)
		where += ` AND m.item_id = $2`
	}

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM inventory_movements m WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count movements: %w", err)
	}

	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT m.id, m.item_id, i.name, m.movement_type, m.qty_delta, m.qty_before, m.qty_after,
		       m.unit_cost, m.reference_type, m.reference_id, m.notes, m.performed_by, m.created_at
		FROM inventory_movements m JOIN inventory_items i ON i.id = m.item_id
		WHERE %s ORDER BY m.created_at DESC LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list movements: %w", err)
	}
	defer rows.Close()

	var movements []inventory.Movement
	for rows.Next() {
		var m inventory.Movement
		if err := rows.Scan(&m.ID, &m.ItemID, &m.ItemName, &m.MovementType, &m.QtyDelta, &m.QtyBefore,
			&m.QtyAfter, &m.UnitCost, &m.ReferenceType, &m.ReferenceID, &m.Notes, &m.PerformedBy, &m.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan movement: %w", err)
		}
		movements = append(movements, m)
	}
	return movements, total, rows.Err()
}

func (r *InventoryRepo) HasMovements(ctx context.Context, tenantID, referenceType, referenceID, movementType string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM inventory_movements
			WHERE tenant_id = $1 AND reference_type = $2 AND reference_id = $3 AND movement_type = $4)`,
		tenantID, referenceType, referenceID, movementType).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check movements: %w", err)
	}
	return exists, nil
}

// ---- recipes ----

func (r *InventoryRepo) GetRecipe(ctx context.Context, tenantID, productID string) ([]inventory.RecipeItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ri.id, ri.product_id, ri.inventory_item_id, ii.name, u.abbreviation, ri.qty
		FROM recipe_items ri
		JOIN inventory_items ii ON ii.id = ri.inventory_item_id
		JOIN units u ON u.id = ii.unit_id
		WHERE ri.tenant_id = $1 AND ri.product_id = $2 AND ri.deleted_at IS NULL
		ORDER BY ii.name`, tenantID, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe: %w", err)
	}
	defer rows.Close()

	var items []inventory.RecipeItem
	for rows.Next() {
		var ri inventory.RecipeItem
		if err := rows.Scan(&ri.ID, &ri.ProductID, &ri.InventoryItemID, &ri.ItemName, &ri.UnitAbbr, &ri.Qty); err != nil {
			return nil, fmt.Errorf("failed to scan recipe item: %w", err)
		}
		items = append(items, ri)
	}
	return items, rows.Err()
}

func (r *InventoryRepo) ReplaceRecipe(ctx context.Context, tenantID, productID string, items []inventory.RecipeItem) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE recipe_items SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND product_id = $2 AND deleted_at IS NULL`, tenantID, productID); err != nil {
		return fmt.Errorf("failed to clear recipe: %w", err)
	}
	for _, ri := range items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO recipe_items (tenant_id, product_id, inventory_item_id, qty)
			VALUES ($1, $2, $3, $4)`,
			tenantID, productID, ri.InventoryItemID, ri.Qty); err != nil {
			return fmt.Errorf("failed to insert recipe item: %w", err)
		}
	}
	return tx.Commit(ctx)
}
