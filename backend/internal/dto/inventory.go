package dto

type UnitRequest struct {
	Name         string `json:"name" binding:"required,min=1,max=40"`
	Abbreviation string `json:"abbreviation" binding:"required,min=1,max=10"`
}

type InventoryItemRequest struct {
	Name         string  `json:"name" binding:"required,min=1,max=120"`
	Type         string  `json:"type" binding:"required,oneof=ingredient finished_good"`
	UnitID       string  `json:"unit_id" binding:"required,uuid"`
	CurrentStock float64 `json:"current_stock" binding:"min=0"`
	ReorderLevel float64 `json:"reorder_level" binding:"min=0"`
	CostPerUnit  int64   `json:"cost_per_unit" binding:"min=0"`
	IsActive     *bool   `json:"is_active"`
}

type MovementRequest struct {
	ItemID       string  `json:"item_id" binding:"required,uuid"`
	MovementType string  `json:"movement_type" binding:"required,oneof=stock_in stock_out adjustment waste"`
	Qty          float64 `json:"qty" binding:"required"`
	UnitCost     int64   `json:"unit_cost" binding:"min=0"`
	Notes        string  `json:"notes" binding:"max=300"`
}

type RecipeItemInput struct {
	InventoryItemID string  `json:"inventory_item_id" binding:"required,uuid"`
	Qty             float64 `json:"qty" binding:"required,gt=0"`
}

type RecipeRequest struct {
	Items []RecipeItemInput `json:"items" binding:"dive"`
}
