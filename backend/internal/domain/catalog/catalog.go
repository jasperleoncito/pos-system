// Package catalog defines the menu aggregate: categories, products,
// variants, modifier groups, and taxes. Prices are integer centavos.
package catalog

import (
	"context"
	"time"
)

type Category struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	SortOrder   int       `json:"sort_order"`
	ImageKey    string    `json:"image_key"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Tax struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	RatePercent float64   `json:"rate_percent"`
	IsInclusive bool      `json:"is_inclusive"`
	IsDefault   bool      `json:"is_default"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Product struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	CategoryID     string    `json:"category_id"`
	TaxID          *string   `json:"tax_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	SKU            string    `json:"sku"`
	BasePrice      int64     `json:"base_price"`
	CostPrice      int64     `json:"cost_price"`
	ImageKey       string    `json:"image_key"`
	ThumbKey       string    `json:"thumb_key"`
	IsActive       bool      `json:"is_active"`
	TrackInventory bool      `json:"track_inventory"`
	SortOrder      int       `json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Joined children for detail views.
	Variants       []Variant       `json:"variants,omitempty"`
	ModifierGroups []ModifierGroup `json:"modifier_groups,omitempty"`
	CategoryName   string          `json:"category_name,omitempty"`
}

type Variant struct {
	ID         string `json:"id"`
	ProductID  string `json:"product_id"`
	Name       string `json:"name"`
	PriceDelta int64  `json:"price_delta"`
	SKU        string `json:"sku"`
	SortOrder  int    `json:"sort_order"`
}

type ModifierGroup struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Name       string     `json:"name"`
	MinSelect  int        `json:"min_select"`
	MaxSelect  int        `json:"max_select"`
	IsRequired bool       `json:"is_required"`
	SortOrder  int        `json:"sort_order"`
	Modifiers  []Modifier `json:"modifiers,omitempty"`
}

type Modifier struct {
	ID         string `json:"id"`
	GroupID    string `json:"group_id"`
	Name       string `json:"name"`
	PriceDelta int64  `json:"price_delta"`
	IsActive   bool   `json:"is_active"`
	SortOrder  int    `json:"sort_order"`
}

// ProductFilter shapes list queries.
type ProductFilter struct {
	CategoryID string
	Search     string
	ActiveOnly bool
	Limit      int
	Offset     int
}

type CategoryRepository interface {
	Create(ctx context.Context, tenantID string, c *Category) error
	GetByID(ctx context.Context, tenantID, id string) (*Category, error)
	List(ctx context.Context, tenantID string, activeOnly bool) ([]Category, error)
	Update(ctx context.Context, tenantID string, c *Category) error
	SoftDelete(ctx context.Context, tenantID, id string) error
}

type ProductRepository interface {
	Create(ctx context.Context, tenantID string, p *Product) error
	GetByID(ctx context.Context, tenantID, id string) (*Product, error)
	List(ctx context.Context, tenantID string, f ProductFilter) ([]Product, int64, error)
	Update(ctx context.Context, tenantID string, p *Product) error
	UpdateImage(ctx context.Context, tenantID, id, imageKey, thumbKey string) error
	SoftDelete(ctx context.Context, tenantID, id string) error

	ReplaceVariants(ctx context.Context, tenantID, productID string, variants []Variant) error
	ReplaceModifierGroups(ctx context.Context, tenantID, productID string, groupIDs []string) error
}

type ModifierRepository interface {
	CreateGroup(ctx context.Context, tenantID string, g *ModifierGroup) error
	GetGroup(ctx context.Context, tenantID, id string) (*ModifierGroup, error)
	ListGroups(ctx context.Context, tenantID string) ([]ModifierGroup, error)
	UpdateGroup(ctx context.Context, tenantID string, g *ModifierGroup) error
	SoftDeleteGroup(ctx context.Context, tenantID, id string) error
	// ReplaceModifiers swaps a group's options in one transaction.
	ReplaceModifiers(ctx context.Context, tenantID, groupID string, modifiers []Modifier) error
}

type TaxRepository interface {
	Create(ctx context.Context, tenantID string, t *Tax) error
	GetByID(ctx context.Context, tenantID, id string) (*Tax, error)
	List(ctx context.Context, tenantID string) ([]Tax, error)
	Update(ctx context.Context, tenantID string, t *Tax) error
	SoftDelete(ctx context.Context, tenantID, id string) error
}
