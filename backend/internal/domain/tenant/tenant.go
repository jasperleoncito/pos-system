// Package tenant defines the tenant aggregate and its persistence contracts.
package tenant

import (
	"context"
	"time"
)

type Tenant struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	OwnerUserID string    `json:"owner_user_id"`
	Status      string    `json:"status"`
	Currency    string    `json:"currency"`
	Timezone    string    `json:"timezone"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Settings struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	LogoKey        string            `json:"logo_key"`
	LogoThumbKey   string            `json:"logo_thumb_key"`
	FaviconKeys    map[string]string `json:"favicon_keys"`
	PrimaryColor   string            `json:"primary_color"`
	SecondaryColor string            `json:"secondary_color"`
	AccentColor    string            `json:"accent_color"`
	ReceiptHeader  string            `json:"receipt_header"`
	ReceiptFooter  string            `json:"receipt_footer"`
	ContactNumber  string            `json:"contact_number"`
	Facebook       string            `json:"facebook"`
	Website        string            `json:"website"`
	Address        string            `json:"address"`
	TaxLabel       string            `json:"tax_label"`
	TaxID          string            `json:"tax_id"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// Membership links a user to a tenant with a role.
type Membership struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	// Joined fields for listings.
	TenantName string `json:"tenant_name,omitempty"`
	TenantSlug string `json:"tenant_slug,omitempty"`
}

type Repository interface {
	Create(ctx context.Context, t *Tenant) error
	GetByID(ctx context.Context, id string) (*Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*Tenant, error)
	Update(ctx context.Context, t *Tenant) error
	List(ctx context.Context, limit, offset int) ([]Tenant, int64, error)
}

type SettingsRepository interface {
	Create(ctx context.Context, s *Settings) error
	GetByTenant(ctx context.Context, tenantID string) (*Settings, error)
	Update(ctx context.Context, s *Settings) error
}

type MembershipRepository interface {
	Create(ctx context.Context, m *Membership) error
	Get(ctx context.Context, tenantID, userID string) (*Membership, error)
	ListByUser(ctx context.Context, userID string) ([]Membership, error)
	ListByTenant(ctx context.Context, tenantID string) ([]Membership, error)
	UpdateRole(ctx context.Context, tenantID, userID, role string) error
	Delete(ctx context.Context, tenantID, userID string) error
}
