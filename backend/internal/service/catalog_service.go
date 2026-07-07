package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/catalog"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/storage"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/imageproc"
)

// CatalogService owns the menu: categories, products, variants,
// modifier groups, and taxes.
type CatalogService struct {
	categories catalog.CategoryRepository
	products   catalog.ProductRepository
	modifiers  catalog.ModifierRepository
	taxes      catalog.TaxRepository
	store      storage.ObjectStorage
	auditor    *AuditService
	logger     *slog.Logger
}

func NewCatalogService(
	categories catalog.CategoryRepository,
	products catalog.ProductRepository,
	modifiers catalog.ModifierRepository,
	taxes catalog.TaxRepository,
	store storage.ObjectStorage,
	auditor *AuditService,
	logger *slog.Logger,
) *CatalogService {
	return &CatalogService{
		categories: categories, products: products, modifiers: modifiers,
		taxes: taxes, store: store, auditor: auditor, logger: logger,
	}
}

// ---- categories ----

func (s *CatalogService) CreateCategory(ctx context.Context, tenantID, userID string, c *catalog.Category) error {
	if err := s.categories.Create(ctx, tenantID, c); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "catalog.category_created",
		EntityType: "category", EntityID: c.ID, After: map[string]any{"name": c.Name},
	})
	return nil
}

func (s *CatalogService) ListCategories(ctx context.Context, tenantID string, activeOnly bool) ([]catalog.Category, error) {
	return s.categories.List(ctx, tenantID, activeOnly)
}

func (s *CatalogService) UpdateCategory(ctx context.Context, tenantID, userID string, c *catalog.Category) error {
	if err := s.categories.Update(ctx, tenantID, c); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "catalog.category_updated",
		EntityType: "category", EntityID: c.ID, After: map[string]any{"name": c.Name},
	})
	return nil
}

func (s *CatalogService) DeleteCategory(ctx context.Context, tenantID, userID, id string) error {
	// A category with live products must not disappear from under them.
	products, _, err := s.products.List(ctx, tenantID, catalog.ProductFilter{CategoryID: id, Limit: 1})
	if err != nil {
		return err
	}
	if len(products) > 0 {
		return apperror.Conflict("move or delete this category's products first")
	}
	if err := s.categories.SoftDelete(ctx, tenantID, id); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "catalog.category_deleted",
		EntityType: "category", EntityID: id,
	})
	return nil
}

// ---- products ----

// ProductView resolves image URLs for API responses.
type ProductView struct {
	catalog.Product
	ImageURL string `json:"image_url"`
	ThumbURL string `json:"thumb_url"`
}

func (s *CatalogService) productView(p catalog.Product) ProductView {
	return ProductView{
		Product:  p,
		ImageURL: s.store.PublicURL(p.ImageKey),
		ThumbURL: s.store.PublicURL(p.ThumbKey),
	}
}

// ProductInput carries create/update fields plus child collections.
type ProductInput struct {
	CategoryID     string
	TaxID          *string
	Name           string
	Description    string
	SKU            string
	BasePrice      int64
	CostPrice      int64
	IsActive       bool
	TrackInventory bool
	SortOrder      int
	Variants       []catalog.Variant
	ModifierGroups []string
}

func (s *CatalogService) validateProductInput(ctx context.Context, tenantID string, in ProductInput) error {
	if _, err := s.categories.GetByID(ctx, tenantID, in.CategoryID); err != nil {
		return apperror.Validation("category does not exist")
	}
	if in.TaxID != nil && *in.TaxID != "" {
		if _, err := s.taxes.GetByID(ctx, tenantID, *in.TaxID); err != nil {
			return apperror.Validation("tax does not exist")
		}
	}
	for _, groupID := range in.ModifierGroups {
		if _, err := s.modifiers.GetGroup(ctx, tenantID, groupID); err != nil {
			return apperror.Validation("modifier group does not exist")
		}
	}
	return nil
}

func (s *CatalogService) CreateProduct(ctx context.Context, tenantID, userID string, in ProductInput) (*ProductView, error) {
	if err := s.validateProductInput(ctx, tenantID, in); err != nil {
		return nil, err
	}

	taxID := in.TaxID
	if taxID != nil && *taxID == "" {
		taxID = nil
	}
	p := &catalog.Product{
		CategoryID: in.CategoryID, TaxID: taxID, Name: in.Name, Description: in.Description,
		SKU: in.SKU, BasePrice: in.BasePrice, CostPrice: in.CostPrice,
		IsActive: in.IsActive, TrackInventory: in.TrackInventory, SortOrder: in.SortOrder,
	}
	if err := s.products.Create(ctx, tenantID, p); err != nil {
		return nil, err
	}
	if len(in.Variants) > 0 {
		if err := s.products.ReplaceVariants(ctx, tenantID, p.ID, in.Variants); err != nil {
			return nil, err
		}
	}
	if len(in.ModifierGroups) > 0 {
		if err := s.products.ReplaceModifierGroups(ctx, tenantID, p.ID, in.ModifierGroups); err != nil {
			return nil, err
		}
	}

	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "catalog.product_created",
		EntityType: "product", EntityID: p.ID,
		After: map[string]any{"name": p.Name, "base_price": p.BasePrice},
	})

	created, err := s.products.GetByID(ctx, tenantID, p.ID)
	if err != nil {
		return nil, err
	}
	view := s.productView(*created)
	return &view, nil
}

func (s *CatalogService) GetProduct(ctx context.Context, tenantID, id string) (*ProductView, error) {
	p, err := s.products.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	view := s.productView(*p)
	return &view, nil
}

func (s *CatalogService) ListProducts(ctx context.Context, tenantID string, f catalog.ProductFilter) ([]ProductView, int64, error) {
	products, total, err := s.products.List(ctx, tenantID, f)
	if err != nil {
		return nil, 0, err
	}
	views := make([]ProductView, len(products))
	for i, p := range products {
		views[i] = s.productView(p)
	}
	return views, total, nil
}

func (s *CatalogService) UpdateProduct(ctx context.Context, tenantID, userID, id string, in ProductInput) (*ProductView, error) {
	current, err := s.products.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := s.validateProductInput(ctx, tenantID, in); err != nil {
		return nil, err
	}

	taxID := in.TaxID
	if taxID != nil && *taxID == "" {
		taxID = nil
	}
	updated := *current
	updated.CategoryID = in.CategoryID
	updated.TaxID = taxID
	updated.Name = in.Name
	updated.Description = in.Description
	updated.SKU = in.SKU
	updated.BasePrice = in.BasePrice
	updated.CostPrice = in.CostPrice
	updated.IsActive = in.IsActive
	updated.TrackInventory = in.TrackInventory
	updated.SortOrder = in.SortOrder

	if err := s.products.Update(ctx, tenantID, &updated); err != nil {
		return nil, err
	}
	if err := s.products.ReplaceVariants(ctx, tenantID, id, in.Variants); err != nil {
		return nil, err
	}
	if err := s.products.ReplaceModifierGroups(ctx, tenantID, id, in.ModifierGroups); err != nil {
		return nil, err
	}

	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "catalog.product_updated",
		EntityType: "product", EntityID: id,
		Before: map[string]any{"name": current.Name, "base_price": current.BasePrice},
		After:  map[string]any{"name": updated.Name, "base_price": updated.BasePrice},
	})

	fresh, err := s.products.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	view := s.productView(*fresh)
	return &view, nil
}

func (s *CatalogService) DeleteProduct(ctx context.Context, tenantID, userID, id string) error {
	if err := s.products.SoftDelete(ctx, tenantID, id); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "catalog.product_deleted",
		EntityType: "product", EntityID: id,
	})
	return nil
}

// UploadProductImage optimizes and stores a product photo.
func (s *CatalogService) UploadProductImage(ctx context.Context, tenantID, userID, productID string, data []byte) (*ProductView, error) {
	p, err := s.products.GetByID(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}

	result, err := imageproc.Optimize(data)
	if err != nil {
		return nil, err
	}

	version := time.Now().Unix()
	imageKey := storage.TenantKey(tenantID, storage.FolderProducts, fmt.Sprintf("%s-%d.webp", productID, version))
	thumbKey := storage.TenantKey(tenantID, storage.FolderProducts, fmt.Sprintf("%s-%d-thumb.webp", productID, version))

	if err := s.store.Put(ctx, imageKey, result.WebP, "image/webp"); err != nil {
		return nil, apperror.Internal(err)
	}
	if err := s.store.Put(ctx, thumbKey, result.ThumbWebP, "image/webp"); err != nil {
		return nil, apperror.Internal(err)
	}

	// Best-effort cleanup of the previous generation.
	for _, key := range []string{p.ImageKey, p.ThumbKey} {
		if key != "" {
			if err := s.store.Delete(ctx, key); err != nil {
				s.logger.Warn("failed to delete old product image", "key", key, "error", err)
			}
		}
	}

	if err := s.products.UpdateImage(ctx, tenantID, productID, imageKey, thumbKey); err != nil {
		return nil, err
	}

	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "catalog.product_image_uploaded",
		EntityType: "product", EntityID: productID,
		After: map[string]any{"image_key": imageKey, "bytes": len(result.WebP)},
	})
	return s.GetProduct(ctx, tenantID, productID)
}

// ---- modifier groups ----

func (s *CatalogService) CreateModifierGroup(ctx context.Context, tenantID, userID string, g *catalog.ModifierGroup, options []catalog.Modifier) (*catalog.ModifierGroup, error) {
	if g.MinSelect > g.MaxSelect {
		return nil, apperror.Validation("min select cannot exceed max select")
	}
	if err := s.modifiers.CreateGroup(ctx, tenantID, g); err != nil {
		return nil, err
	}
	if len(options) > 0 {
		if err := s.modifiers.ReplaceModifiers(ctx, tenantID, g.ID, options); err != nil {
			return nil, err
		}
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "catalog.modifier_group_created",
		EntityType: "modifier_group", EntityID: g.ID, After: map[string]any{"name": g.Name},
	})
	return s.modifiers.GetGroup(ctx, tenantID, g.ID)
}

func (s *CatalogService) ListModifierGroups(ctx context.Context, tenantID string) ([]catalog.ModifierGroup, error) {
	return s.modifiers.ListGroups(ctx, tenantID)
}

func (s *CatalogService) UpdateModifierGroup(ctx context.Context, tenantID, userID string, g *catalog.ModifierGroup, options []catalog.Modifier) (*catalog.ModifierGroup, error) {
	if g.MinSelect > g.MaxSelect {
		return nil, apperror.Validation("min select cannot exceed max select")
	}
	if err := s.modifiers.UpdateGroup(ctx, tenantID, g); err != nil {
		return nil, err
	}
	if err := s.modifiers.ReplaceModifiers(ctx, tenantID, g.ID, options); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "catalog.modifier_group_updated",
		EntityType: "modifier_group", EntityID: g.ID, After: map[string]any{"name": g.Name},
	})
	return s.modifiers.GetGroup(ctx, tenantID, g.ID)
}

func (s *CatalogService) DeleteModifierGroup(ctx context.Context, tenantID, userID, id string) error {
	if err := s.modifiers.SoftDeleteGroup(ctx, tenantID, id); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "catalog.modifier_group_deleted",
		EntityType: "modifier_group", EntityID: id,
	})
	return nil
}

// ---- taxes ----

func (s *CatalogService) CreateTax(ctx context.Context, tenantID, userID string, t *catalog.Tax) error {
	if err := s.taxes.Create(ctx, tenantID, t); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "catalog.tax_created",
		EntityType: "tax", EntityID: t.ID, After: map[string]any{"name": t.Name, "rate": t.RatePercent},
	})
	return nil
}

func (s *CatalogService) ListTaxes(ctx context.Context, tenantID string) ([]catalog.Tax, error) {
	return s.taxes.List(ctx, tenantID)
}

func (s *CatalogService) UpdateTax(ctx context.Context, tenantID, userID string, t *catalog.Tax) error {
	if err := s.taxes.Update(ctx, tenantID, t); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "catalog.tax_updated",
		EntityType: "tax", EntityID: t.ID, After: map[string]any{"name": t.Name, "rate": t.RatePercent},
	})
	return nil
}

func (s *CatalogService) DeleteTax(ctx context.Context, tenantID, userID, id string) error {
	if err := s.taxes.SoftDelete(ctx, tenantID, id); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "catalog.tax_deleted",
		EntityType: "tax", EntityID: id,
	})
	return nil
}
