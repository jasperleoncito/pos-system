package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/order"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

// Splits, refunds, promo columns, and voids — extensions of OrderRepo.

func (r *OrderRepo) UpdatePromo(ctx context.Context, tenantID, id string, discountID, couponID *string, discountTotal, total int64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE orders SET discount_id = $3, coupon_id = $4, discount_total = $5, total = $6, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, discountID, couponID, discountTotal, total)
	if err != nil {
		return fmt.Errorf("failed to update order promo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("order")
	}
	return nil
}

func (r *OrderRepo) SetVoided(ctx context.Context, tenantID, id, voidedBy, reason string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE orders SET status = 'voided', voided_by = $3, void_reason = $4, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, voidedBy, reason)
	if err != nil {
		return fmt.Errorf("failed to void order: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("order")
	}
	return nil
}

func (r *OrderRepo) CreateSplits(ctx context.Context, tenantID, orderID string, amounts []int64) ([]order.Split, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	splits := make([]order.Split, 0, len(amounts))
	for i, amount := range amounts {
		var s order.Split
		err := tx.QueryRow(ctx, `
			INSERT INTO order_splits (tenant_id, order_id, split_number, amount)
			VALUES ($1, $2, $3, $4)
			RETURNING id, order_id, split_number, amount, status, created_at`,
			tenantID, orderID, i+1, amount,
		).Scan(&s.ID, &s.OrderID, &s.SplitNumber, &s.Amount, &s.Status, &s.CreatedAt)
		if err != nil {
			if isUniqueViolation(err) {
				return nil, apperror.Conflict("this order already has splits")
			}
			return nil, fmt.Errorf("failed to create split: %w", err)
		}
		splits = append(splits, s)
	}
	return splits, tx.Commit(ctx)
}

func (r *OrderRepo) ListSplits(ctx context.Context, tenantID, orderID string) ([]order.Split, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, order_id, split_number, amount, status, created_at
		FROM order_splits
		WHERE tenant_id = $1 AND order_id = $2 AND deleted_at IS NULL
		ORDER BY split_number`, tenantID, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to list splits: %w", err)
	}
	defer rows.Close()

	var splits []order.Split
	for rows.Next() {
		var s order.Split
		if err := rows.Scan(&s.ID, &s.OrderID, &s.SplitNumber, &s.Amount, &s.Status, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan split: %w", err)
		}
		splits = append(splits, s)
	}
	return splits, rows.Err()
}

func (r *OrderRepo) GetSplit(ctx context.Context, tenantID, splitID string) (*order.Split, error) {
	var s order.Split
	err := r.db.QueryRow(ctx, `
		SELECT id, order_id, split_number, amount, status, created_at
		FROM order_splits
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, splitID,
	).Scan(&s.ID, &s.OrderID, &s.SplitNumber, &s.Amount, &s.Status, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("split")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get split: %w", err)
	}
	return &s, nil
}

func (r *OrderRepo) MarkSplitPaid(ctx context.Context, tenantID, splitID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE order_splits SET status = 'paid', updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND status = 'pending' AND deleted_at IS NULL`,
		tenantID, splitID)
	if err != nil {
		return fmt.Errorf("failed to mark split paid: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.Validation("split is already paid")
	}
	return nil
}

// NextRefundNumber issues per-tenant refund numbers. max+1 is adequate
// here: refunds are rare, manager-only actions.
func (r *OrderRepo) NextRefundNumber(ctx context.Context, tenantID string) (int64, error) {
	var n int64
	err := r.db.QueryRow(ctx,
		`SELECT coalesce(max(refund_number), 0) + 1 FROM refunds WHERE tenant_id = $1`, tenantID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("failed to get next refund number: %w", err)
	}
	return n, nil
}

func (r *OrderRepo) CreateRefund(ctx context.Context, tenantID string, refund *order.Refund) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO refunds (tenant_id, order_id, refund_number, reason, amount, refunded_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		tenantID, refund.OrderID, refund.RefundNumber, refund.Reason, refund.Amount, refund.RefundedBy,
	).Scan(&refund.ID, &refund.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create refund: %w", err)
	}

	for i := range refund.Items {
		item := &refund.Items[i]
		item.RefundID = refund.ID
		err := tx.QueryRow(ctx, `
			INSERT INTO refund_items (tenant_id, refund_id, order_item_id, qty, amount)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id`,
			tenantID, refund.ID, item.OrderItemID, item.Qty, item.Amount,
		).Scan(&item.ID)
		if err != nil {
			return fmt.Errorf("failed to create refund item: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *OrderRepo) ListRefunds(ctx context.Context, tenantID, orderID string) ([]order.Refund, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, order_id, refund_number, reason, amount, refunded_by, created_at
		FROM refunds
		WHERE tenant_id = $1 AND order_id = $2
		ORDER BY created_at`, tenantID, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to list refunds: %w", err)
	}
	defer rows.Close()

	var refunds []order.Refund
	for rows.Next() {
		var rf order.Refund
		if err := rows.Scan(&rf.ID, &rf.OrderID, &rf.RefundNumber, &rf.Reason, &rf.Amount, &rf.RefundedBy, &rf.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan refund: %w", err)
		}
		refunds = append(refunds, rf)
	}
	return refunds, rows.Err()
}

func (r *OrderRepo) RefundedTotal(ctx context.Context, tenantID, orderID string) (int64, error) {
	var total int64
	err := r.db.QueryRow(ctx,
		`SELECT coalesce(sum(amount), 0) FROM refunds WHERE tenant_id = $1 AND order_id = $2`,
		tenantID, orderID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to sum refunds: %w", err)
	}
	return total, nil
}
