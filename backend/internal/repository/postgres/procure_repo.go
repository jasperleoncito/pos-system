package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/procure"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

type ProcureRepo struct {
	db *pgxpool.Pool
}

func NewProcureRepo(db *pgxpool.Pool) *ProcureRepo { return &ProcureRepo{db: db} }

// ---- suppliers ----

func (r *ProcureRepo) CreateSupplier(ctx context.Context, tenantID string, s *procure.Supplier) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO suppliers (tenant_id, name, contact_person, phone, email, address, notes, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at`,
		tenantID, s.Name, s.ContactPerson, s.Phone, s.Email, s.Address, s.Notes, s.IsActive,
	).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("a supplier with that name already exists")
		}
		return fmt.Errorf("failed to create supplier: %w", err)
	}
	return nil
}

func (r *ProcureRepo) ListSuppliers(ctx context.Context, tenantID string) ([]procure.Supplier, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, contact_person, phone, email, address, notes, is_active, created_at
		FROM suppliers WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY name`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list suppliers: %w", err)
	}
	defer rows.Close()
	var suppliers []procure.Supplier
	for rows.Next() {
		var s procure.Supplier
		if err := rows.Scan(&s.ID, &s.Name, &s.ContactPerson, &s.Phone, &s.Email, &s.Address, &s.Notes, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan supplier: %w", err)
		}
		suppliers = append(suppliers, s)
	}
	return suppliers, rows.Err()
}

func (r *ProcureRepo) UpdateSupplier(ctx context.Context, tenantID string, s *procure.Supplier) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE suppliers SET name=$3, contact_person=$4, phone=$5, email=$6, address=$7, notes=$8, is_active=$9, updated_at=now()
		WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`,
		tenantID, s.ID, s.Name, s.ContactPerson, s.Phone, s.Email, s.Address, s.Notes, s.IsActive)
	if err != nil {
		return fmt.Errorf("failed to update supplier: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("supplier")
	}
	return nil
}

func (r *ProcureRepo) SoftDeleteSupplier(ctx context.Context, tenantID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE suppliers SET deleted_at=now(), updated_at=now() WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to delete supplier: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("supplier")
	}
	return nil
}

// ---- purchase orders ----

func (r *ProcureRepo) NextPONumber(ctx context.Context, tenantID string) (int64, error) {
	var n int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO po_counters (tenant_id, counter) VALUES ($1, 1)
		ON CONFLICT (tenant_id) DO UPDATE SET counter = po_counters.counter + 1
		RETURNING counter`, tenantID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("failed to get next po number: %w", err)
	}
	return n, nil
}

func (r *ProcureRepo) CreatePO(ctx context.Context, tenantID string, po *procure.PurchaseOrder) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO purchase_orders (tenant_id, po_number, supplier_id, status, notes, total, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at`,
		tenantID, po.PONumber, po.SupplierID, po.Status, po.Notes, po.Total, po.CreatedBy,
	).Scan(&po.ID, &po.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create po: %w", err)
	}
	for i := range po.Items {
		it := &po.Items[i]
		it.POID = po.ID
		if err := tx.QueryRow(ctx, `
			INSERT INTO purchase_order_items (tenant_id, po_id, item_id, qty_ordered, unit_cost)
			VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			tenantID, po.ID, it.ItemID, it.QtyOrdered, it.UnitCost).Scan(&it.ID); err != nil {
			return fmt.Errorf("failed to create po item: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *ProcureRepo) GetPO(ctx context.Context, tenantID, id string) (*procure.PurchaseOrder, error) {
	var po procure.PurchaseOrder
	err := r.db.QueryRow(ctx, `
		SELECT p.id, p.po_number, p.supplier_id, s.name, p.status, p.notes, p.total, p.created_by, p.received_at, p.created_at
		FROM purchase_orders p JOIN suppliers s ON s.id = p.supplier_id
		WHERE p.tenant_id=$1 AND p.id=$2 AND p.deleted_at IS NULL`, tenantID, id,
	).Scan(&po.ID, &po.PONumber, &po.SupplierID, &po.SupplierName, &po.Status, &po.Notes, &po.Total, &po.CreatedBy, &po.ReceivedAt, &po.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("purchase order")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get po: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT pi.id, pi.po_id, pi.item_id, ii.name, u.abbreviation, pi.qty_ordered, pi.qty_received, pi.unit_cost
		FROM purchase_order_items pi
		JOIN inventory_items ii ON ii.id = pi.item_id
		JOIN units u ON u.id = ii.unit_id
		WHERE pi.tenant_id=$1 AND pi.po_id=$2 ORDER BY ii.name`, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to list po items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var it procure.POItem
		if err := rows.Scan(&it.ID, &it.POID, &it.ItemID, &it.ItemName, &it.UnitAbbr, &it.QtyOrdered, &it.QtyReceived, &it.UnitCost); err != nil {
			return nil, fmt.Errorf("failed to scan po item: %w", err)
		}
		po.Items = append(po.Items, it)
	}
	return &po, rows.Err()
}

func (r *ProcureRepo) ListPOs(ctx context.Context, tenantID string) ([]procure.PurchaseOrder, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.po_number, p.supplier_id, s.name, p.status, p.notes, p.total, p.created_by, p.received_at, p.created_at
		FROM purchase_orders p JOIN suppliers s ON s.id = p.supplier_id
		WHERE p.tenant_id=$1 AND p.deleted_at IS NULL ORDER BY p.created_at DESC LIMIT 100`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pos: %w", err)
	}
	defer rows.Close()
	var pos []procure.PurchaseOrder
	for rows.Next() {
		var po procure.PurchaseOrder
		if err := rows.Scan(&po.ID, &po.PONumber, &po.SupplierID, &po.SupplierName, &po.Status, &po.Notes, &po.Total, &po.CreatedBy, &po.ReceivedAt, &po.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan po: %w", err)
		}
		pos = append(pos, po)
	}
	return pos, rows.Err()
}

func (r *ProcureRepo) UpdatePOStatus(ctx context.Context, tenantID, id, status string, receivedAt *time.Time) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE purchase_orders SET status=$3, received_at=COALESCE($4, received_at), updated_at=now()
		WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, id, status, receivedAt)
	if err != nil {
		return fmt.Errorf("failed to update po status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("purchase order")
	}
	return nil
}

func (r *ProcureRepo) UpdatePOItemReceived(ctx context.Context, tenantID, poItemID string, qtyReceived float64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE purchase_order_items SET qty_received = qty_received + $3, updated_at=now()
		WHERE tenant_id=$1 AND id=$2`, tenantID, poItemID, qtyReceived)
	if err != nil {
		return fmt.Errorf("failed to update po item received: %w", err)
	}
	return nil
}

// ---- alerts ----

func (r *ProcureRepo) EnsureAlert(ctx context.Context, tenantID, itemID, alertType string, stock float64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO stock_alerts (tenant_id, item_id, alert_type, stock_at_alert)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tenant_id, item_id) WHERE acknowledged_at IS NULL DO NOTHING`,
		tenantID, itemID, alertType, stock)
	if err != nil {
		return fmt.Errorf("failed to ensure alert: %w", err)
	}
	return nil
}

func (r *ProcureRepo) ListAlerts(ctx context.Context, tenantID string, openOnly bool) ([]procure.Alert, error) {
	query := `
		SELECT a.id, a.item_id, i.name, a.alert_type, a.stock_at_alert, a.acknowledged_at, a.created_at
		FROM stock_alerts a JOIN inventory_items i ON i.id = a.item_id
		WHERE a.tenant_id = $1`
	if openOnly {
		query += ` AND a.acknowledged_at IS NULL`
	}
	query += ` ORDER BY a.created_at DESC LIMIT 100`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}
	defer rows.Close()
	var alerts []procure.Alert
	for rows.Next() {
		var a procure.Alert
		if err := rows.Scan(&a.ID, &a.ItemID, &a.ItemName, &a.AlertType, &a.StockAtAlert, &a.Acknowledged, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

func (r *ProcureRepo) AckAlert(ctx context.Context, tenantID, alertID, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE stock_alerts SET acknowledged_at=now(), acknowledged_by=$3
		WHERE tenant_id=$1 AND id=$2 AND acknowledged_at IS NULL`, tenantID, alertID, userID)
	if err != nil {
		return fmt.Errorf("failed to ack alert: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("alert")
	}
	return nil
}
