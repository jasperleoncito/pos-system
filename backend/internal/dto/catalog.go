package dto

type CategoryRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=120"`
	Description string `json:"description" binding:"max=500"`
	SortOrder   int    `json:"sort_order" binding:"min=0"`
	IsActive    *bool  `json:"is_active"`
}

type VariantInput struct {
	Name       string `json:"name" binding:"required,min=1,max=120"`
	PriceDelta int64  `json:"price_delta"`
	SKU        string `json:"sku" binding:"max=60"`
}

type ProductRequest struct {
	CategoryID     string         `json:"category_id" binding:"required,uuid"`
	TaxID          *string        `json:"tax_id" binding:"omitempty,uuid"`
	Name           string         `json:"name" binding:"required,min=1,max=160"`
	Description    string         `json:"description" binding:"max=1000"`
	SKU            string         `json:"sku" binding:"max=60"`
	BasePrice      int64          `json:"base_price" binding:"min=0"`
	CostPrice      int64          `json:"cost_price" binding:"min=0"`
	IsActive       *bool          `json:"is_active"`
	TrackInventory bool           `json:"track_inventory"`
	SortOrder      int            `json:"sort_order" binding:"min=0"`
	Variants       []VariantInput `json:"variants" binding:"dive"`
	ModifierGroups []string       `json:"modifier_groups" binding:"dive,uuid"`
}

type ModifierInput struct {
	Name       string `json:"name" binding:"required,min=1,max=120"`
	PriceDelta int64  `json:"price_delta"`
	IsActive   *bool  `json:"is_active"`
}

type ModifierGroupRequest struct {
	Name       string          `json:"name" binding:"required,min=1,max=120"`
	MinSelect  int             `json:"min_select" binding:"min=0"`
	MaxSelect  int             `json:"max_select" binding:"min=1"`
	IsRequired bool            `json:"is_required"`
	SortOrder  int             `json:"sort_order" binding:"min=0"`
	Modifiers  []ModifierInput `json:"modifiers" binding:"dive"`
}

type TaxRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=120"`
	RatePercent float64 `json:"rate_percent" binding:"min=0,max=100"`
	IsInclusive bool    `json:"is_inclusive"`
	IsDefault   bool    `json:"is_default"`
	IsActive    *bool   `json:"is_active"`
}
