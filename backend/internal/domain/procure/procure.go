// Package procure defines suppliers, purchase orders, and stock alerts.
package procure

import (
	"context"
	"time"
)

// PO statuses.
const (
	POStatusDraft             = "draft"
	POStatusOrdered           = "ordered"
	POStatusPartiallyReceived = "partially_received"
	POStatusReceived          = "received"
	POStatusCancelled         = "cancelled"
)

type Supplier struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	ContactPerson string    `json:"contact_person"`
	Phone         string    `json:"phone"`
	Email         string    `json:"email"`
	Address       string    `json:"address"`
	Notes         string    `json:"notes"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}

type POItem struct {
	ID          string  `json:"id"`
	POID        string  `json:"po_id"`
	ItemID      string  `json:"item_id"`
	ItemName    string  `json:"item_name,omitempty"`
	UnitAbbr    string  `json:"unit_abbr,omitempty"`
	QtyOrdered  float64 `json:"qty_ordered"`
	QtyReceived float64 `json:"qty_received"`
	UnitCost    int64   `json:"unit_cost"`
}

type PurchaseOrder struct {
	ID           string     `json:"id"`
	PONumber     int64      `json:"po_number"`
	SupplierID   string     `json:"supplier_id"`
	SupplierName string     `json:"supplier_name,omitempty"`
	Status       string     `json:"status"`
	Notes        string     `json:"notes"`
	Total        int64      `json:"total"`
	CreatedBy    string     `json:"created_by"`
	ReceivedAt   *time.Time `json:"received_at"`
	CreatedAt    time.Time  `json:"created_at"`
	Items        []POItem   `json:"items,omitempty"`
}

type Alert struct {
	ID           string     `json:"id"`
	ItemID       string     `json:"item_id"`
	ItemName     string     `json:"item_name,omitempty"`
	AlertType    string     `json:"alert_type"`
	StockAtAlert float64    `json:"stock_at_alert"`
	Acknowledged *time.Time `json:"acknowledged_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Repository interface {
	CreateSupplier(ctx context.Context, tenantID string, s *Supplier) error
	ListSuppliers(ctx context.Context, tenantID string) ([]Supplier, error)
	UpdateSupplier(ctx context.Context, tenantID string, s *Supplier) error
	SoftDeleteSupplier(ctx context.Context, tenantID, id string) error

	NextPONumber(ctx context.Context, tenantID string) (int64, error)
	CreatePO(ctx context.Context, tenantID string, po *PurchaseOrder) error
	GetPO(ctx context.Context, tenantID, id string) (*PurchaseOrder, error)
	ListPOs(ctx context.Context, tenantID string) ([]PurchaseOrder, error)
	UpdatePOStatus(ctx context.Context, tenantID, id, status string, receivedAt *time.Time) error
	UpdatePOItemReceived(ctx context.Context, tenantID, poItemID string, qtyReceived float64) error

	// EnsureAlert opens an alert if none is open; created reports whether
	// a new row was inserted (used to trigger notifications exactly once).
	EnsureAlert(ctx context.Context, tenantID, itemID, alertType string, stock float64) (created bool, err error)
	ListAlerts(ctx context.Context, tenantID string, openOnly bool) ([]Alert, error)
	AckAlert(ctx context.Context, tenantID, alertID, userID string) error
}
