// Package inventory defines stock items, recipes (BOM), and the
// append-only movement ledger. Quantities are decimal (kg, L, pcs);
// costs are centavos.
package inventory

import (
	"context"
	"time"
)

const (
	TypeIngredient   = "ingredient"
	TypeFinishedGood = "finished_good"
)

// Movement types.
const (
	MoveStockIn      = "stock_in"
	MoveStockOut     = "stock_out"
	MoveAdjustment   = "adjustment"
	MoveSale         = "sale"
	MovePOReceive    = "po_receive"
	MoveRefundReturn = "refund_return"
	MoveWaste        = "waste"
)

type Unit struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
}

type Item struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	UnitID       string    `json:"unit_id"`
	UnitAbbr     string    `json:"unit_abbr,omitempty"`
	CurrentStock float64   `json:"current_stock"`
	ReorderLevel float64   `json:"reorder_level"`
	CostPerUnit  int64     `json:"cost_per_unit"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RecipeItem struct {
	ID              string  `json:"id"`
	ProductID       string  `json:"product_id"`
	InventoryItemID string  `json:"inventory_item_id"`
	ItemName        string  `json:"item_name,omitempty"`
	UnitAbbr        string  `json:"unit_abbr,omitempty"`
	Qty             float64 `json:"qty"`
}

type Movement struct {
	ID            string    `json:"id"`
	ItemID        string    `json:"item_id"`
	ItemName      string    `json:"item_name,omitempty"`
	MovementType  string    `json:"movement_type"`
	QtyDelta      float64   `json:"qty_delta"`
	QtyBefore     float64   `json:"qty_before"`
	QtyAfter      float64   `json:"qty_after"`
	UnitCost      int64     `json:"unit_cost"`
	ReferenceType string    `json:"reference_type"`
	ReferenceID   string    `json:"reference_id"`
	Notes         string    `json:"notes"`
	PerformedBy   string    `json:"performed_by"`
	CreatedAt     time.Time `json:"created_at"`
}

// ApplyInput is one atomic stock change.
type ApplyInput struct {
	ItemID        string
	MovementType  string
	QtyDelta      float64 // signed
	UnitCost      int64
	ReferenceType string
	ReferenceID   string
	Notes         string
	PerformedBy   string
}

type Repository interface {
	CreateUnit(ctx context.Context, tenantID string, u *Unit) error
	ListUnits(ctx context.Context, tenantID string) ([]Unit, error)

	CreateItem(ctx context.Context, tenantID string, i *Item) error
	GetItem(ctx context.Context, tenantID, id string) (*Item, error)
	GetItemByName(ctx context.Context, tenantID, name string) (*Item, error)
	ListItems(ctx context.Context, tenantID string, search string) ([]Item, error)
	UpdateItem(ctx context.Context, tenantID string, i *Item) error
	SoftDeleteItem(ctx context.Context, tenantID, id string) error

	// Apply locks the item row, writes the ledger entry with before/after,
	// and updates current_stock — all in one transaction.
	Apply(ctx context.Context, tenantID string, in ApplyInput) (*Movement, error)
	ListMovements(ctx context.Context, tenantID, itemID string, limit, offset int) ([]Movement, int64, error)
	// HasMovements reports whether a reference already produced movements
	// of the given type (idempotency guard).
	HasMovements(ctx context.Context, tenantID, referenceType, referenceID, movementType string) (bool, error)

	GetRecipe(ctx context.Context, tenantID, productID string) ([]RecipeItem, error)
	ReplaceRecipe(ctx context.Context, tenantID, productID string, items []RecipeItem) error
}
