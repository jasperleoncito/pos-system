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

type ProductRepo struct {
	db *pgxpool.Pool
}

func NewProductRepo(db *pgxpool.Pool) *ProductRepo { return &ProductRepo{db: db} }

const productColumns = `p.id, p.tenant_id, p.category_id, p.tax_id, p.name, p.description, p.sku,
	p.base_price, p.cost_price, p.image_key, p.thumb_key, p.is_active, p.track_inventory,
	p.sort_order, p.created_at, p.updated_at`

func scanProduct(row pgx.Row) (*catalog.Product, error) {
	var p catalog.Product
	err := row.Scan(&p.ID, &p.TenantID, &p.CategoryID, &p.TaxID, &p.Name, &p.Description, &p.SKU,
		&p.BasePrice, &p.CostPrice, &p.ImageKey, &p.ThumbKey, &p.IsActive, &p.TrackInventory,
		&p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("product")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan product: %w", err)
	}
	return &p, nil
}

func (r *ProductRepo) Create(ctx context.Context, tenantID string, p *catalog.Product) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO products (tenant_id, category_id, tax_id, name, description, sku, base_price,
			cost_price, is_active, track_inventory, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, tenant_id, created_at, updated_at`,
		tenantID, p.CategoryID, p.TaxID, p.Name, p.Description, p.SKU, p.BasePrice,
		p.CostPrice, p.IsActive, p.TrackInventory, p.SortOrder,
	).Scan(&p.ID, &p.TenantID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}
	return nil
}

func (r *ProductRepo) GetByID(ctx context.Context, tenantID, id string) (*catalog.Product, error) {
	p, err := scanProduct(r.db.QueryRow(ctx, `
		SELECT `+productColumns+` FROM products p
		WHERE p.tenant_id = $1 AND p.id = $2 AND p.deleted_at IS NULL`, tenantID, id))
	if err != nil {
		return nil, err
	}
	if err := r.attachChildren(ctx, tenantID, []*catalog.Product{p}); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *ProductRepo) List(ctx context.Context, tenantID string, f catalog.ProductFilter) ([]catalog.Product, int64, error) {
	where := `p.tenant_id = $1 AND p.deleted_at IS NULL`
	args := []any{tenantID}
	if f.CategoryID != "" {
		args = append(args, f.CategoryID)
		where += fmt.Sprintf(` AND p.category_id = $%d`, len(args))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		where += fmt.Sprintf(` AND (p.name ILIKE $%d OR p.sku ILIKE $%d)`, len(args), len(args))
	}
	if f.ActiveOnly {
		where += ` AND p.is_active`
	}

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM products p WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	limit, offset := f.Limit, f.Offset
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT %s, c.name FROM products p
		JOIN categories c ON c.id = p.category_id
		WHERE %s
		ORDER BY p.sort_order, p.name
		LIMIT $%d OFFSET $%d`, productColumns, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	var products []*catalog.Product
	for rows.Next() {
		var p catalog.Product
		if err := rows.Scan(&p.ID, &p.TenantID, &p.CategoryID, &p.TaxID, &p.Name, &p.Description, &p.SKU,
			&p.BasePrice, &p.CostPrice, &p.ImageKey, &p.ThumbKey, &p.IsActive, &p.TrackInventory,
			&p.SortOrder, &p.CreatedAt, &p.UpdatedAt, &p.CategoryName); err != nil {
			return nil, 0, fmt.Errorf("failed to scan product row: %w", err)
		}
		products = append(products, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if err := r.attachChildren(ctx, tenantID, products); err != nil {
		return nil, 0, err
	}

	result := make([]catalog.Product, len(products))
	for i, p := range products {
		result[i] = *p
	}
	return result, total, nil
}

// attachChildren loads variants and modifier groups for a set of products
// in two batched queries (no N+1).
func (r *ProductRepo) attachChildren(ctx context.Context, tenantID string, products []*catalog.Product) error {
	if len(products) == 0 {
		return nil
	}
	ids := make([]string, len(products))
	byID := make(map[string]*catalog.Product, len(products))
	for i, p := range products {
		ids[i] = p.ID
		byID[p.ID] = p
	}

	// Variants
	rows, err := r.db.Query(ctx, `
		SELECT id, product_id, name, price_delta, sku, sort_order
		FROM product_variants
		WHERE tenant_id = $1 AND product_id = ANY($2) AND deleted_at IS NULL
		ORDER BY sort_order, name`, tenantID, ids)
	if err != nil {
		return fmt.Errorf("failed to list variants: %w", err)
	}
	for rows.Next() {
		var v catalog.Variant
		if err := rows.Scan(&v.ID, &v.ProductID, &v.Name, &v.PriceDelta, &v.SKU, &v.SortOrder); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan variant: %w", err)
		}
		byID[v.ProductID].Variants = append(byID[v.ProductID].Variants, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// Modifier groups with their options
	rows, err = r.db.Query(ctx, `
		SELECT pmg.product_id, g.id, g.name, g.min_select, g.max_select, g.is_required, g.sort_order,
		       m.id, m.name, m.price_delta, m.is_active, m.sort_order
		FROM product_modifier_groups pmg
		JOIN modifier_groups g ON g.id = pmg.modifier_group_id AND g.deleted_at IS NULL
		LEFT JOIN modifiers m ON m.group_id = g.id AND m.deleted_at IS NULL
		WHERE pmg.tenant_id = $1 AND pmg.product_id = ANY($2) AND pmg.deleted_at IS NULL
		ORDER BY pmg.sort_order, g.name, m.sort_order, m.name`, tenantID, ids)
	if err != nil {
		return fmt.Errorf("failed to list product modifier groups: %w", err)
	}
	defer rows.Close()

	groupIndex := map[string]map[string]int{} // productID -> groupID -> index
	for rows.Next() {
		var productID string
		var g catalog.ModifierGroup
		var mID, mName *string
		var mDelta *int64
		var mActive *bool
		var mSort *int
		if err := rows.Scan(&productID, &g.ID, &g.Name, &g.MinSelect, &g.MaxSelect, &g.IsRequired, &g.SortOrder,
			&mID, &mName, &mDelta, &mActive, &mSort); err != nil {
			return fmt.Errorf("failed to scan product modifier group: %w", err)
		}
		p := byID[productID]
		if groupIndex[productID] == nil {
			groupIndex[productID] = map[string]int{}
		}
		idx, ok := groupIndex[productID][g.ID]
		if !ok {
			p.ModifierGroups = append(p.ModifierGroups, g)
			idx = len(p.ModifierGroups) - 1
			groupIndex[productID][g.ID] = idx
		}
		if mID != nil {
			p.ModifierGroups[idx].Modifiers = append(p.ModifierGroups[idx].Modifiers, catalog.Modifier{
				ID: *mID, GroupID: g.ID, Name: *mName, PriceDelta: *mDelta, IsActive: *mActive, SortOrder: *mSort,
			})
		}
	}
	return rows.Err()
}

func (r *ProductRepo) Update(ctx context.Context, tenantID string, p *catalog.Product) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE products SET category_id = $3, tax_id = $4, name = $5, description = $6, sku = $7,
			base_price = $8, cost_price = $9, is_active = $10, track_inventory = $11, sort_order = $12,
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, p.ID, p.CategoryID, p.TaxID, p.Name, p.Description, p.SKU,
		p.BasePrice, p.CostPrice, p.IsActive, p.TrackInventory, p.SortOrder)
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("product")
	}
	return nil
}

func (r *ProductRepo) UpdateImage(ctx context.Context, tenantID, id, imageKey, thumbKey string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE products SET image_key = $3, thumb_key = $4, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id, imageKey, thumbKey)
	if err != nil {
		return fmt.Errorf("failed to update product image: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("product")
	}
	return nil
}

func (r *ProductRepo) SoftDelete(ctx context.Context, tenantID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE products SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("product")
	}
	return nil
}

func (r *ProductRepo) ReplaceVariants(ctx context.Context, tenantID, productID string, variants []catalog.Variant) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE product_variants SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND product_id = $2 AND deleted_at IS NULL`, tenantID, productID); err != nil {
		return fmt.Errorf("failed to clear variants: %w", err)
	}
	for i, v := range variants {
		if _, err := tx.Exec(ctx, `
			INSERT INTO product_variants (tenant_id, product_id, name, price_delta, sku, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			tenantID, productID, v.Name, v.PriceDelta, v.SKU, i); err != nil {
			return fmt.Errorf("failed to insert variant: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *ProductRepo) ReplaceModifierGroups(ctx context.Context, tenantID, productID string, groupIDs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE product_modifier_groups SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND product_id = $2 AND deleted_at IS NULL`, tenantID, productID); err != nil {
		return fmt.Errorf("failed to clear modifier group links: %w", err)
	}
	for i, groupID := range groupIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO product_modifier_groups (tenant_id, product_id, modifier_group_id, sort_order)
			VALUES ($1, $2, $3, $4)`,
			tenantID, productID, groupID, i); err != nil {
			return fmt.Errorf("failed to link modifier group: %w", err)
		}
	}
	return tx.Commit(ctx)
}
