package postgres

import (
	"context"
	"fmt"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/order"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

// Kitchen queue queries — extensions of OrderRepo.

// ListKitchen returns fired orders still moving through the kitchen:
// held/voided/refunded orders are excluded, completed kitchen tickets
// drop off the board.
func (r *OrderRepo) ListKitchen(ctx context.Context, tenantID string) ([]order.Order, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+orderColumns+` FROM orders o
		WHERE o.tenant_id = $1 AND o.deleted_at IS NULL
		  AND o.status IN ('open', 'completed')
		  AND o.kitchen_status IN ('pending', 'preparing', 'ready')
		ORDER BY o.priority DESC, o.created_at`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list kitchen orders: %w", err)
	}
	defer rows.Close()

	var orders []order.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range orders {
		if err := r.attachItems(ctx, tenantID, &orders[i]); err != nil {
			return nil, err
		}
	}
	return orders, nil
}

func (r *OrderRepo) UpdateKitchenStatus(ctx context.Context, tenantID, orderID, status string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE orders SET kitchen_status = $3, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, orderID, status)
	if err != nil {
		return fmt.Errorf("failed to update kitchen status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("order")
	}
	return nil
}

func (r *OrderRepo) UpdateItemStatus(ctx context.Context, tenantID, orderID, itemID, status string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE order_items SET status = $4, updated_at = now()
		WHERE tenant_id = $1 AND order_id = $2 AND id = $3 AND deleted_at IS NULL`,
		tenantID, orderID, itemID, status)
	if err != nil {
		return fmt.Errorf("failed to update item status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("order item")
	}
	return nil
}

func (r *OrderRepo) SetPriority(ctx context.Context, tenantID, orderID string, priority bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE orders SET priority = $3, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, orderID, priority)
	if err != nil {
		return fmt.Errorf("failed to set priority: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("order")
	}
	return nil
}
