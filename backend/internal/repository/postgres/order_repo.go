package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/order"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

type OrderRepo struct {
	db *pgxpool.Pool
}

func NewOrderRepo(db *pgxpool.Pool) *OrderRepo { return &OrderRepo{db: db} }

func (r *OrderRepo) NextOrderNumber(ctx context.Context, tenantID string) (int64, error) {
	var n int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO order_counters (tenant_id, counter) VALUES ($1, 1)
		ON CONFLICT (tenant_id) DO UPDATE SET counter = order_counters.counter + 1
		RETURNING counter`, tenantID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("failed to get next order number: %w", err)
	}
	return n, nil
}

// Create inserts the order with its items and modifiers in one transaction.
func (r *OrderRepo) Create(ctx context.Context, tenantID string, o *order.Order) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO orders (tenant_id, order_number, order_type, table_number, customer_id,
			cashier_user_id, status, subtotal, discount_total, tax_total, total, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, kitchen_status, created_at, updated_at`,
		tenantID, o.OrderNumber, o.OrderType, o.TableNumber, o.CustomerID,
		o.CashierUserID, o.Status, o.Subtotal, o.DiscountTotal, o.TaxTotal, o.Total, o.Notes,
	).Scan(&o.ID, &o.KitchenStatus, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	for i := range o.Items {
		item := &o.Items[i]
		item.OrderID = o.ID
		err = tx.QueryRow(ctx, `
			INSERT INTO order_items (tenant_id, order_id, product_id, variant_id, name, variant_name,
				unit_price, qty, discount_amount, tax_amount, line_total, notes, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			RETURNING id`,
			tenantID, o.ID, item.ProductID, item.VariantID, item.Name, item.VariantName,
			item.UnitPrice, item.Qty, item.DiscountAmount, item.TaxAmount, item.LineTotal,
			item.Notes, item.Status,
		).Scan(&item.ID)
		if err != nil {
			return fmt.Errorf("failed to insert order item: %w", err)
		}
		for j := range item.Modifiers {
			mod := &item.Modifiers[j]
			mod.OrderItemID = item.ID
			err = tx.QueryRow(ctx, `
				INSERT INTO order_item_modifiers (tenant_id, order_item_id, modifier_id, group_name, name, price_delta)
				VALUES ($1, $2, $3, $4, $5, $6)
				RETURNING id`,
				tenantID, item.ID, mod.ModifierID, mod.GroupName, mod.Name, mod.PriceDelta,
			).Scan(&mod.ID)
			if err != nil {
				return fmt.Errorf("failed to insert item modifier: %w", err)
			}
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO order_status_history (tenant_id, order_id, field, from_value, to_value, changed_by)
		VALUES ($1, $2, 'status', '', $3, $4)`,
		tenantID, o.ID, o.Status, o.CashierUserID); err != nil {
		return fmt.Errorf("failed to insert status history: %w", err)
	}

	return tx.Commit(ctx)
}

const orderColumns = `o.id, o.tenant_id, o.order_number, o.order_type, o.table_number, o.customer_id,
	o.cashier_user_id, o.status, o.kitchen_status, o.priority, o.subtotal, o.discount_total,
	o.tax_total, o.total, o.tendered, o.change, o.notes, o.discount_id, o.coupon_id,
	o.completed_at, o.voided_by, o.void_reason, o.created_at, o.updated_at`

func scanOrder(row pgx.Row) (*order.Order, error) {
	var o order.Order
	err := row.Scan(&o.ID, &o.TenantID, &o.OrderNumber, &o.OrderType, &o.TableNumber, &o.CustomerID,
		&o.CashierUserID, &o.Status, &o.KitchenStatus, &o.Priority, &o.Subtotal, &o.DiscountTotal,
		&o.TaxTotal, &o.Total, &o.Tendered, &o.Change, &o.Notes, &o.DiscountID, &o.CouponID,
		&o.CompletedAt, &o.VoidedBy, &o.VoidReason, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("order")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan order: %w", err)
	}
	return &o, nil
}

func (r *OrderRepo) GetByID(ctx context.Context, tenantID, id string) (*order.Order, error) {
	o, err := scanOrder(r.db.QueryRow(ctx, `
		SELECT `+orderColumns+` FROM orders o
		WHERE o.tenant_id = $1 AND o.id = $2 AND o.deleted_at IS NULL`, tenantID, id))
	if err != nil {
		return nil, err
	}
	if err := r.attachItems(ctx, tenantID, o); err != nil {
		return nil, err
	}
	payments, err := r.ListPayments(ctx, tenantID, o.ID)
	if err != nil {
		return nil, err
	}
	o.Payments = payments

	splits, err := r.ListSplits(ctx, tenantID, o.ID)
	if err != nil {
		return nil, err
	}
	o.Splits = splits

	refunds, err := r.ListRefunds(ctx, tenantID, o.ID)
	if err != nil {
		return nil, err
	}
	o.Refunds = refunds

	// Cashier display name for receipts.
	_ = r.db.QueryRow(ctx,
		`SELECT full_name FROM users WHERE id = $1`, o.CashierUserID).Scan(&o.CashierName)
	return o, nil
}

func (r *OrderRepo) attachItems(ctx context.Context, tenantID string, o *order.Order) error {
	rows, err := r.db.Query(ctx, `
		SELECT id, order_id, product_id, variant_id, name, variant_name, unit_price, qty,
		       discount_amount, tax_amount, line_total, notes, status
		FROM order_items
		WHERE tenant_id = $1 AND order_id = $2 AND deleted_at IS NULL
		ORDER BY created_at`, tenantID, o.ID)
	if err != nil {
		return fmt.Errorf("failed to list order items: %w", err)
	}
	defer rows.Close()

	itemIndex := map[string]int{}
	for rows.Next() {
		var it order.Item
		if err := rows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.VariantID, &it.Name, &it.VariantName,
			&it.UnitPrice, &it.Qty, &it.DiscountAmount, &it.TaxAmount, &it.LineTotal, &it.Notes, &it.Status); err != nil {
			return fmt.Errorf("failed to scan order item: %w", err)
		}
		o.Items = append(o.Items, it)
		itemIndex[it.ID] = len(o.Items) - 1
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(o.Items) == 0 {
		return nil
	}

	ids := make([]string, 0, len(itemIndex))
	for id := range itemIndex {
		ids = append(ids, id)
	}
	mrows, err := r.db.Query(ctx, `
		SELECT id, order_item_id, modifier_id, group_name, name, price_delta
		FROM order_item_modifiers
		WHERE tenant_id = $1 AND order_item_id = ANY($2)
		ORDER BY created_at`, tenantID, ids)
	if err != nil {
		return fmt.Errorf("failed to list item modifiers: %w", err)
	}
	defer mrows.Close()

	for mrows.Next() {
		var m order.ItemModifier
		if err := mrows.Scan(&m.ID, &m.OrderItemID, &m.ModifierID, &m.GroupName, &m.Name, &m.PriceDelta); err != nil {
			return fmt.Errorf("failed to scan item modifier: %w", err)
		}
		idx := itemIndex[m.OrderItemID]
		o.Items[idx].Modifiers = append(o.Items[idx].Modifiers, m)
	}
	return mrows.Err()
}

func (r *OrderRepo) List(ctx context.Context, tenantID string, f order.Filter) ([]order.Order, int64, error) {
	where := `o.tenant_id = $1 AND o.deleted_at IS NULL`
	args := []any{tenantID}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(` AND o.status = $%d`, len(args))
	}
	if f.Search != "" {
		args = append(args, f.Search)
		where += fmt.Sprintf(` AND o.order_number::text = $%d`, len(args))
	}
	if f.CustomerID != "" {
		args = append(args, f.CustomerID)
		where += fmt.Sprintf(` AND o.customer_id = $%d`, len(args))
	}

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM orders o WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	args = append(args, limit, f.Offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT %s FROM orders o WHERE %s
		ORDER BY o.created_at DESC
		LIMIT $%d OFFSET $%d`, orderColumns, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	var orders []order.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, *o)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Attach items so lists can show line summaries.
	for i := range orders {
		if err := r.attachItems(ctx, tenantID, &orders[i]); err != nil {
			return nil, 0, err
		}
	}
	return orders, total, nil
}

func (r *OrderRepo) UpdateStatus(ctx context.Context, tenantID, id, status string, completedAt *time.Time) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE orders SET status = $3, completed_at = COALESCE($4, completed_at), updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, status, completedAt)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("order")
	}
	return nil
}

func (r *OrderRepo) UpdatePaymentTotals(ctx context.Context, tenantID, id string, tendered, change int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE orders SET tendered = $3, change = $4, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, tendered, change)
	if err != nil {
		return fmt.Errorf("failed to update payment totals: %w", err)
	}
	return nil
}

func (r *OrderRepo) AddStatusHistory(ctx context.Context, tenantID, orderID, field, from, to, changedBy string) error {
	var changedByArg any
	if changedBy != "" {
		changedByArg = changedBy
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO order_status_history (tenant_id, order_id, field, from_value, to_value, changed_by)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		tenantID, orderID, field, from, to, changedByArg)
	if err != nil {
		return fmt.Errorf("failed to add status history: %w", err)
	}
	return nil
}

func (r *OrderRepo) AddPayment(ctx context.Context, tenantID string, p *order.Payment) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO payments (tenant_id, order_id, split_id, method, amount, reference_no, received_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, status, created_at`,
		tenantID, p.OrderID, p.SplitID, p.Method, p.Amount, p.ReferenceNo, p.ReceivedBy,
	).Scan(&p.ID, &p.Status, &p.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to add payment: %w", err)
	}
	return nil
}

func (r *OrderRepo) ListPayments(ctx context.Context, tenantID, orderID string) ([]order.Payment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, order_id, split_id, method, amount, reference_no, status, received_by, created_at
		FROM payments
		WHERE tenant_id = $1 AND order_id = $2 AND deleted_at IS NULL
		ORDER BY created_at`, tenantID, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}
	defer rows.Close()

	var payments []order.Payment
	for rows.Next() {
		var p order.Payment
		if err := rows.Scan(&p.ID, &p.OrderID, &p.SplitID, &p.Method, &p.Amount, &p.ReferenceNo, &p.Status, &p.ReceivedBy, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}
