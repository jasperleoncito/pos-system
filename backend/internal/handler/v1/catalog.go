package v1

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/catalog"
	"github.com/jasperleoncito/pos-system/backend/internal/dto"
	"github.com/jasperleoncito/pos-system/backend/internal/middleware"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/imageproc"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

// CatalogHandler exposes menu management endpoints.
type CatalogHandler struct {
	catalog *service.CatalogService
}

func NewCatalogHandler(catalogSvc *service.CatalogService) *CatalogHandler {
	return &CatalogHandler{catalog: catalogSvc}
}

func tenantUser(c *gin.Context) (tenantID, userID string) {
	return c.GetString(middleware.CtxTenantID), c.GetString(middleware.CtxUserID)
}

func boolOrDefault(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

// ---- categories ----

// ListCategories godoc
//
//	@Summary	List menu categories
//	@Tags		catalog
//	@Security	BearerAuth
//	@Produce	json
//	@Param		active	query		bool	false	"Only active categories"
//	@Success	200		{object}	response.Envelope
//	@Router		/categories [get]
func (h *CatalogHandler) ListCategories(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	activeOnly := c.Query("active") == "true"
	categories, err := h.catalog.ListCategories(c.Request.Context(), tenantID, activeOnly)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", categories)
}

// CreateCategory godoc
//
//	@Summary	Create a menu category
//	@Tags		catalog
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.CategoryRequest	true	"Category"
//	@Success	201		{object}	response.Envelope
//	@Router		/categories [post]
func (h *CatalogHandler) CreateCategory(c *gin.Context) {
	var req dto.CategoryRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	category := &catalog.Category{
		Name: req.Name, Description: req.Description,
		SortOrder: req.SortOrder, IsActive: boolOrDefault(req.IsActive, true),
	}
	if err := h.catalog.CreateCategory(c.Request.Context(), tenantID, userID, category); err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "category created", category)
}

// UpdateCategory godoc
//
//	@Summary	Update a menu category
//	@Tags		catalog
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Category ID"
//	@Param		payload	body		dto.CategoryRequest	true	"Category"
//	@Success	200		{object}	response.Envelope
//	@Router		/categories/{id} [put]
func (h *CatalogHandler) UpdateCategory(c *gin.Context) {
	var req dto.CategoryRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	category := &catalog.Category{
		ID: c.Param("id"), Name: req.Name, Description: req.Description,
		SortOrder: req.SortOrder, IsActive: boolOrDefault(req.IsActive, true),
	}
	if err := h.catalog.UpdateCategory(c.Request.Context(), tenantID, userID, category); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "category updated", category)
}

// DeleteCategory godoc
//
//	@Summary	Delete a menu category (must be empty)
//	@Tags		catalog
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Category ID"
//	@Success	200	{object}	response.Envelope
//	@Failure	409	{object}	response.ErrorEnvelope
//	@Router		/categories/{id} [delete]
func (h *CatalogHandler) DeleteCategory(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	if err := h.catalog.DeleteCategory(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "category deleted", nil)
}

// ---- products ----

// ListProducts godoc
//
//	@Summary	List products with search, category filter, and pagination
//	@Tags		catalog
//	@Security	BearerAuth
//	@Produce	json
//	@Param		category_id	query		string	false	"Filter by category"
//	@Param		search		query		string	false	"Search name or SKU"
//	@Param		active		query		bool	false	"Only active products"
//	@Param		page		query		int		false	"Page (1-based)"
//	@Param		limit		query		int		false	"Page size"
//	@Success	200			{object}	response.Envelope
//	@Router		/products [get]
func (h *CatalogHandler) ListProducts(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}

	products, total, err := h.catalog.ListProducts(c.Request.Context(), tenantID, catalog.ProductFilter{
		CategoryID: c.Query("category_id"),
		Search:     c.Query("search"),
		ActiveOnly: c.Query("active") == "true",
		Limit:      limit,
		Offset:     (page - 1) * limit,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Paginated(c, "", products, response.Meta{Total: total, Page: page, Limit: limit})
}

// GetProduct godoc
//
//	@Summary	Get one product with variants and modifier groups
//	@Tags		catalog
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Product ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/products/{id} [get]
func (h *CatalogHandler) GetProduct(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	product, err := h.catalog.GetProduct(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", product)
}

func productInputFromRequest(req dto.ProductRequest) service.ProductInput {
	variants := make([]catalog.Variant, len(req.Variants))
	for i, v := range req.Variants {
		variants[i] = catalog.Variant{Name: v.Name, PriceDelta: v.PriceDelta, SKU: v.SKU}
	}
	return service.ProductInput{
		CategoryID: req.CategoryID, TaxID: req.TaxID, Name: req.Name,
		Description: req.Description, SKU: req.SKU, BasePrice: req.BasePrice,
		CostPrice: req.CostPrice, IsActive: boolOrDefault(req.IsActive, true),
		TrackInventory: req.TrackInventory, SortOrder: req.SortOrder,
		Variants: variants, ModifierGroups: req.ModifierGroups,
	}
}

// CreateProduct godoc
//
//	@Summary	Create a product with variants and modifier assignments
//	@Tags		catalog
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.ProductRequest	true	"Product"
//	@Success	201		{object}	response.Envelope
//	@Router		/products [post]
func (h *CatalogHandler) CreateProduct(c *gin.Context) {
	var req dto.ProductRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	product, err := h.catalog.CreateProduct(c.Request.Context(), tenantID, userID, productInputFromRequest(req))
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "product created", product)
}

// UpdateProduct godoc
//
//	@Summary	Update a product
//	@Tags		catalog
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Product ID"
//	@Param		payload	body		dto.ProductRequest	true	"Product"
//	@Success	200		{object}	response.Envelope
//	@Router		/products/{id} [put]
func (h *CatalogHandler) UpdateProduct(c *gin.Context) {
	var req dto.ProductRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	product, err := h.catalog.UpdateProduct(c.Request.Context(), tenantID, userID, c.Param("id"), productInputFromRequest(req))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "product updated", product)
}

// DeleteProduct godoc
//
//	@Summary	Soft-delete a product
//	@Tags		catalog
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Product ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/products/{id} [delete]
func (h *CatalogHandler) DeleteProduct(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	if err := h.catalog.DeleteProduct(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "product deleted", nil)
}

// UploadProductImage godoc
//
//	@Summary	Upload a product photo (optimized to WebP automatically)
//	@Tags		catalog
//	@Security	BearerAuth
//	@Accept		multipart/form-data
//	@Produce	json
//	@Param		id		path		string	true	"Product ID"
//	@Param		image	formData	file	true	"PNG/JPG/WEBP, max 10MB"
//	@Success	200		{object}	response.Envelope
//	@Router		/products/{id}/image [post]
func (h *CatalogHandler) UploadProductImage(c *gin.Context) {
	file, _, err := c.Request.FormFile("image")
	if err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "attach an image file in the 'image' field")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, imageproc.MaxUploadBytes+1))
	if err != nil {
		respondError(c, err)
		return
	}

	tenantID, userID := tenantUser(c)
	product, err := h.catalog.UploadProductImage(c.Request.Context(), tenantID, userID, c.Param("id"), data)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "product image updated", product)
}

// ---- modifier groups ----

// ListModifierGroups godoc
//
//	@Summary	List modifier groups with options
//	@Tags		catalog
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/modifier-groups [get]
func (h *CatalogHandler) ListModifierGroups(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	groups, err := h.catalog.ListModifierGroups(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", groups)
}

func modifierGroupFromRequest(req dto.ModifierGroupRequest) (*catalog.ModifierGroup, []catalog.Modifier) {
	group := &catalog.ModifierGroup{
		Name: req.Name, MinSelect: req.MinSelect, MaxSelect: req.MaxSelect,
		IsRequired: req.IsRequired, SortOrder: req.SortOrder,
	}
	options := make([]catalog.Modifier, len(req.Modifiers))
	for i, m := range req.Modifiers {
		options[i] = catalog.Modifier{Name: m.Name, PriceDelta: m.PriceDelta, IsActive: boolOrDefault(m.IsActive, true)}
	}
	return group, options
}

// CreateModifierGroup godoc
//
//	@Summary	Create a modifier group with options
//	@Tags		catalog
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.ModifierGroupRequest	true	"Modifier group"
//	@Success	201		{object}	response.Envelope
//	@Router		/modifier-groups [post]
func (h *CatalogHandler) CreateModifierGroup(c *gin.Context) {
	var req dto.ModifierGroupRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	group, options := modifierGroupFromRequest(req)
	created, err := h.catalog.CreateModifierGroup(c.Request.Context(), tenantID, userID, group, options)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "modifier group created", created)
}

// UpdateModifierGroup godoc
//
//	@Summary	Update a modifier group and replace its options
//	@Tags		catalog
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"Group ID"
//	@Param		payload	body		dto.ModifierGroupRequest	true	"Modifier group"
//	@Success	200		{object}	response.Envelope
//	@Router		/modifier-groups/{id} [put]
func (h *CatalogHandler) UpdateModifierGroup(c *gin.Context) {
	var req dto.ModifierGroupRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	group, options := modifierGroupFromRequest(req)
	group.ID = c.Param("id")
	updated, err := h.catalog.UpdateModifierGroup(c.Request.Context(), tenantID, userID, group, options)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "modifier group updated", updated)
}

// DeleteModifierGroup godoc
//
//	@Summary	Delete a modifier group
//	@Tags		catalog
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Group ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/modifier-groups/{id} [delete]
func (h *CatalogHandler) DeleteModifierGroup(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	if err := h.catalog.DeleteModifierGroup(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "modifier group deleted", nil)
}

// ---- taxes ----

// ListTaxes godoc
//
//	@Summary	List taxes
//	@Tags		catalog
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/taxes [get]
func (h *CatalogHandler) ListTaxes(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	taxes, err := h.catalog.ListTaxes(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", taxes)
}

// CreateTax godoc
//
//	@Summary	Create a tax
//	@Tags		catalog
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.TaxRequest	true	"Tax"
//	@Success	201		{object}	response.Envelope
//	@Router		/taxes [post]
func (h *CatalogHandler) CreateTax(c *gin.Context) {
	var req dto.TaxRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	tax := &catalog.Tax{
		Name: req.Name, RatePercent: req.RatePercent, IsInclusive: req.IsInclusive,
		IsDefault: req.IsDefault, IsActive: boolOrDefault(req.IsActive, true),
	}
	if err := h.catalog.CreateTax(c.Request.Context(), tenantID, userID, tax); err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "tax created", tax)
}

// UpdateTax godoc
//
//	@Summary	Update a tax
//	@Tags		catalog
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string			true	"Tax ID"
//	@Param		payload	body		dto.TaxRequest	true	"Tax"
//	@Success	200		{object}	response.Envelope
//	@Router		/taxes/{id} [put]
func (h *CatalogHandler) UpdateTax(c *gin.Context) {
	var req dto.TaxRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	tax := &catalog.Tax{
		ID: c.Param("id"), Name: req.Name, RatePercent: req.RatePercent,
		IsInclusive: req.IsInclusive, IsDefault: req.IsDefault, IsActive: boolOrDefault(req.IsActive, true),
	}
	if err := h.catalog.UpdateTax(c.Request.Context(), tenantID, userID, tax); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "tax updated", tax)
}

// DeleteTax godoc
//
//	@Summary	Delete a tax
//	@Tags		catalog
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Tax ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/taxes/{id} [delete]
func (h *CatalogHandler) DeleteTax(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	if err := h.catalog.DeleteTax(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "tax deleted", nil)
}
