package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/inventory"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/procure"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

// ProcureService owns suppliers, purchase orders, and stock alerts.
type ProcureService struct {
	repo      procure.Repository
	inventory *InventoryService
	auditor   *AuditService
	logger    *slog.Logger
}

func NewProcureService(repo procure.Repository, inv *InventoryService, auditor *AuditService, logger *slog.Logger) *ProcureService {
	return &ProcureService{repo: repo, inventory: inv, auditor: auditor, logger: logger}
}

// ---- suppliers ----

func (s *ProcureService) CreateSupplier(ctx context.Context, tenantID, userID string, sup *procure.Supplier) error {
	if err := s.repo.CreateSupplier(ctx, tenantID, sup); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "procure.supplier_created",
		EntityType: "supplier", EntityID: sup.ID, After: map[string]any{"name": sup.Name}})
	return nil
}

func (s *ProcureService) ListSuppliers(ctx context.Context, tenantID string) ([]procure.Supplier, error) {
	return s.repo.ListSuppliers(ctx, tenantID)
}

func (s *ProcureService) UpdateSupplier(ctx context.Context, tenantID, userID string, sup *procure.Supplier) error {
	if err := s.repo.UpdateSupplier(ctx, tenantID, sup); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "procure.supplier_updated",
		EntityType: "supplier", EntityID: sup.ID, After: map[string]any{"name": sup.Name}})
	return nil
}

func (s *ProcureService) DeleteSupplier(ctx context.Context, tenantID, userID, id string) error {
	if err := s.repo.SoftDeleteSupplier(ctx, tenantID, id); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "procure.supplier_deleted",
		EntityType: "supplier", EntityID: id})
	return nil
}

// ---- purchase orders ----

func (s *ProcureService) CreatePO(ctx context.Context, tenantID, userID string, po *procure.PurchaseOrder) (*procure.PurchaseOrder, error) {
	if len(po.Items) == 0 {
		return nil, apperror.Validation("a purchase order needs at least one line")
	}
	var total int64
	for _, it := range po.Items {
		if it.QtyOrdered <= 0 {
			return nil, apperror.Validation("ordered quantities must be positive")
		}
		total += int64(it.QtyOrdered * float64(it.UnitCost))
	}
	number, err := s.repo.NextPONumber(ctx, tenantID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	po.PONumber = number
	po.Status = procure.POStatusDraft
	po.Total = total
	po.CreatedBy = userID
	if err := s.repo.CreatePO(ctx, tenantID, po); err != nil {
		return nil, apperror.Internal(err)
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "procure.po_created",
		EntityType: "purchase_order", EntityID: po.ID, After: map[string]any{"po_number": po.PONumber, "total": total}})
	return s.repo.GetPO(ctx, tenantID, po.ID)
}

func (s *ProcureService) ListPOs(ctx context.Context, tenantID string) ([]procure.PurchaseOrder, error) {
	return s.repo.ListPOs(ctx, tenantID)
}

func (s *ProcureService) GetPO(ctx context.Context, tenantID, id string) (*procure.PurchaseOrder, error) {
	return s.repo.GetPO(ctx, tenantID, id)
}

// MarkOrdered moves a draft PO to ordered.
func (s *ProcureService) MarkOrdered(ctx context.Context, tenantID, userID, id string) (*procure.PurchaseOrder, error) {
	po, err := s.repo.GetPO(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if po.Status != procure.POStatusDraft {
		return nil, apperror.Validation("only draft purchase orders can be marked ordered")
	}
	if err := s.repo.UpdatePOStatus(ctx, tenantID, id, procure.POStatusOrdered, nil); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "procure.po_ordered",
		EntityType: "purchase_order", EntityID: id})
	return s.repo.GetPO(ctx, tenantID, id)
}

func (s *ProcureService) CancelPO(ctx context.Context, tenantID, userID, id string) (*procure.PurchaseOrder, error) {
	po, err := s.repo.GetPO(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if po.Status == procure.POStatusReceived || po.Status == procure.POStatusCancelled {
		return nil, apperror.Validation("this purchase order can no longer be cancelled")
	}
	if err := s.repo.UpdatePOStatus(ctx, tenantID, id, procure.POStatusCancelled, nil); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "procure.po_cancelled",
		EntityType: "purchase_order", EntityID: id})
	return s.repo.GetPO(ctx, tenantID, id)
}

// ReceiveLine is one received quantity for a PO line.
type ReceiveLine struct {
	POItemID string
	Qty      float64
}

// Receive books received quantities: stock moves in via the ledger,
// item cost updates to the PO's unit cost, and the PO status becomes
// partially_received or received.
func (s *ProcureService) Receive(ctx context.Context, tenantID, userID, poID string, lines []ReceiveLine) (*procure.PurchaseOrder, error) {
	po, err := s.repo.GetPO(ctx, tenantID, poID)
	if err != nil {
		return nil, err
	}
	if po.Status != procure.POStatusOrdered && po.Status != procure.POStatusPartiallyReceived {
		return nil, apperror.Validation("only ordered purchase orders can be received")
	}

	itemByID := map[string]*procure.POItem{}
	for i := range po.Items {
		itemByID[po.Items[i].ID] = &po.Items[i]
	}

	receivedAny := false
	for _, line := range lines {
		poItem, ok := itemByID[line.POItemID]
		if !ok {
			return nil, apperror.Validation("receive line does not belong to this purchase order")
		}
		if line.Qty <= 0 {
			continue
		}
		remaining := poItem.QtyOrdered - poItem.QtyReceived
		if line.Qty > remaining {
			return nil, apperror.Validation("received quantity exceeds what remains on " + poItem.ItemName)
		}

		if err := s.inventory.ReceiveStock(ctx, tenantID, userID, poItem.ItemID, line.Qty, poItem.UnitCost, po.ID, po.PONumber); err != nil {
			return nil, err
		}
		if err := s.repo.UpdatePOItemReceived(ctx, tenantID, line.POItemID, line.Qty); err != nil {
			return nil, apperror.Internal(err)
		}
		poItem.QtyReceived += line.Qty
		receivedAny = true
	}
	if !receivedAny {
		return nil, apperror.Validation("nothing to receive")
	}

	complete := true
	for _, it := range po.Items {
		if it.QtyReceived < it.QtyOrdered {
			complete = false
			break
		}
	}
	status := procure.POStatusPartiallyReceived
	var receivedAt *time.Time
	if complete {
		status = procure.POStatusReceived
		now := time.Now()
		receivedAt = &now
	}
	if err := s.repo.UpdatePOStatus(ctx, tenantID, poID, status, receivedAt); err != nil {
		return nil, apperror.Internal(err)
	}

	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "procure.po_received",
		EntityType: "purchase_order", EntityID: poID, After: map[string]any{"status": status}})
	return s.repo.GetPO(ctx, tenantID, poID)
}

// ---- alerts ----

func (s *ProcureService) ListAlerts(ctx context.Context, tenantID string, openOnly bool) ([]procure.Alert, error) {
	return s.repo.ListAlerts(ctx, tenantID, openOnly)
}

func (s *ProcureService) AckAlert(ctx context.Context, tenantID, userID, alertID string) error {
	return s.repo.AckAlert(ctx, tenantID, alertID, userID)
}

// ---- inventory-side hooks (defined here to keep alert logic together) ----

// AlertSink lets the inventory service raise alerts without a cycle.
type AlertSink interface {
	EnsureAlert(ctx context.Context, tenantID, itemID, alertType string, stock float64) error
}

// ReceiveStock books PO stock into the ledger and refreshes unit cost.
func (s *InventoryService) ReceiveStock(ctx context.Context, tenantID, userID, itemID string, qty float64, unitCost int64, poID string, poNumber int64) error {
	if _, err := s.repo.Apply(ctx, tenantID, inventory.ApplyInput{
		ItemID: itemID, MovementType: inventory.MovePOReceive, QtyDelta: qty,
		UnitCost: unitCost, ReferenceType: "purchase_order", ReferenceID: poID,
		PerformedBy: userID,
	}); err != nil {
		return err
	}
	item, err := s.repo.GetItem(ctx, tenantID, itemID)
	if err != nil {
		return err
	}
	if unitCost > 0 && unitCost != item.CostPerUnit {
		item.CostPerUnit = unitCost
		if err := s.repo.UpdateItem(ctx, tenantID, item); err != nil {
			return err
		}
	}
	return nil
}

// SetAlertSink wires the alert store; checkAlert runs after movements.
func (s *InventoryService) SetAlertSink(sink AlertSink) { s.alerts = sink }

func (s *InventoryService) checkAlert(ctx context.Context, tenantID, itemID string) {
	if s.alerts == nil {
		return
	}
	item, err := s.repo.GetItem(ctx, tenantID, itemID)
	if err != nil {
		return
	}
	if item.CurrentStock <= 0 {
		_ = s.alerts.EnsureAlert(ctx, tenantID, itemID, "out_of_stock", item.CurrentStock)
	} else if item.CurrentStock <= item.ReorderLevel && item.ReorderLevel > 0 {
		_ = s.alerts.EnsureAlert(ctx, tenantID, itemID, "low_stock", item.CurrentStock)
	}
}
