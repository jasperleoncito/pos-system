package dto

type SupplierRequest struct {
	Name          string `json:"name" binding:"required,min=1,max=120"`
	ContactPerson string `json:"contact_person" binding:"max=120"`
	Phone         string `json:"phone" binding:"max=40"`
	Email         string `json:"email" binding:"omitempty,email"`
	Address       string `json:"address" binding:"max=300"`
	Notes         string `json:"notes" binding:"max=500"`
	IsActive      *bool  `json:"is_active"`
}

type POItemInput struct {
	ItemID   string  `json:"item_id" binding:"required,uuid"`
	Qty      float64 `json:"qty" binding:"required,gt=0"`
	UnitCost int64   `json:"unit_cost" binding:"min=0"`
}

type CreatePORequest struct {
	SupplierID string        `json:"supplier_id" binding:"required,uuid"`
	Notes      string        `json:"notes" binding:"max=500"`
	Items      []POItemInput `json:"items" binding:"required,min=1,dive"`
}

type ReceiveLineInput struct {
	POItemID string  `json:"po_item_id" binding:"required,uuid"`
	Qty      float64 `json:"qty" binding:"min=0"`
}

type ReceivePORequest struct {
	Lines []ReceiveLineInput `json:"lines" binding:"required,min=1,dive"`
}
