package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/inventory"
	"github.com/jasperleoncito/pos-system/backend/internal/dto"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

// InventoryHandler exposes stock items, movements, and recipes.
type InventoryHandler struct {
	inventory *service.InventoryService
}

func NewInventoryHandler(inv *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{inventory: inv}
}

// ListUnits godoc
//
//	@Summary	List measurement units
//	@Tags		inventory
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/units [get]
func (h *InventoryHandler) ListUnits(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	units, err := h.inventory.ListUnits(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", units)
}

// CreateUnit godoc
//
//	@Summary	Create a measurement unit
//	@Tags		inventory
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.UnitRequest	true	"Unit"
//	@Success	201		{object}	response.Envelope
//	@Router		/units [post]
func (h *InventoryHandler) CreateUnit(c *gin.Context) {
	var req dto.UnitRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, _ := tenantUser(c)
	unit := &inventory.Unit{Name: req.Name, Abbreviation: req.Abbreviation}
	if err := h.inventory.CreateUnit(c.Request.Context(), tenantID, unit); err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "unit created", unit)
}

// ListItems godoc
//
//	@Summary	List inventory items with stock levels
//	@Tags		inventory
//	@Security	BearerAuth
//	@Produce	json
//	@Param		search	query		string	false	"Name filter"
//	@Success	200		{object}	response.Envelope
//	@Router		/inventory/items [get]
func (h *InventoryHandler) ListItems(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	items, err := h.inventory.ListItems(c.Request.Context(), tenantID, c.Query("search"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", items)
}

func itemFromRequest(req dto.InventoryItemRequest) *inventory.Item {
	return &inventory.Item{
		Name: req.Name, Type: req.Type, UnitID: req.UnitID,
		CurrentStock: req.CurrentStock, ReorderLevel: req.ReorderLevel,
		CostPerUnit: req.CostPerUnit, IsActive: boolOrDefault(req.IsActive, true),
	}
}

// CreateItem godoc
//
//	@Summary	Create an inventory item
//	@Tags		inventory
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.InventoryItemRequest	true	"Item"
//	@Success	201		{object}	response.Envelope
//	@Router		/inventory/items [post]
func (h *InventoryHandler) CreateItem(c *gin.Context) {
	var req dto.InventoryItemRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	item := itemFromRequest(req)
	if err := h.inventory.CreateItem(c.Request.Context(), tenantID, userID, item); err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "item created", item)
}

// UpdateItem godoc
//
//	@Summary	Update an inventory item (stock changes go through movements)
//	@Tags		inventory
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"Item ID"
//	@Param		payload	body		dto.InventoryItemRequest	true	"Item"
//	@Success	200		{object}	response.Envelope
//	@Router		/inventory/items/{id} [put]
func (h *InventoryHandler) UpdateItem(c *gin.Context) {
	var req dto.InventoryItemRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	item := itemFromRequest(req)
	item.ID = c.Param("id")
	if err := h.inventory.UpdateItem(c.Request.Context(), tenantID, userID, item); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "item updated", item)
}

// DeleteItem godoc
//
//	@Summary	Soft-delete an inventory item
//	@Tags		inventory
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Item ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/inventory/items/{id} [delete]
func (h *InventoryHandler) DeleteItem(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	if err := h.inventory.DeleteItem(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "item deleted", nil)
}

// Move godoc
//
//	@Summary	Apply a stock movement (stock_in, stock_out, adjustment, waste)
//	@Tags		inventory
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.MovementRequest	true	"Movement"
//	@Success	201		{object}	response.Envelope
//	@Router		/inventory/movements [post]
func (h *InventoryHandler) Move(c *gin.Context) {
	var req dto.MovementRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	movement, err := h.inventory.Move(c.Request.Context(), tenantID, userID, inventory.ApplyInput{
		ItemID: req.ItemID, MovementType: req.MovementType, QtyDelta: req.Qty,
		UnitCost: req.UnitCost, Notes: req.Notes,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "movement recorded", movement)
}

// ListMovements godoc
//
//	@Summary	Movement history (optionally per item)
//	@Tags		inventory
//	@Security	BearerAuth
//	@Produce	json
//	@Param		item_id	query		string	false	"Filter by item"
//	@Param		page	query		int		false	"Page"
//	@Param		limit	query		int		false	"Page size"
//	@Success	200		{object}	response.Envelope
//	@Router		/inventory/movements [get]
func (h *InventoryHandler) ListMovements(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 30
	}
	movements, total, err := h.inventory.ListMovements(c.Request.Context(), tenantID, c.Query("item_id"), limit, (page-1)*limit)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Paginated(c, "", movements, response.Meta{Total: total, Page: page, Limit: limit})
}

// GetRecipe godoc
//
//	@Summary	Get a product's recipe (BOM)
//	@Tags		inventory
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Product ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/products/{id}/recipe [get]
func (h *InventoryHandler) GetRecipe(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	recipe, err := h.inventory.GetRecipe(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", recipe)
}

// SaveRecipe godoc
//
//	@Summary	Replace a product's recipe (BOM)
//	@Tags		inventory
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Product ID"
//	@Param		payload	body		dto.RecipeRequest	true	"Recipe items"
//	@Success	200		{object}	response.Envelope
//	@Router		/products/{id}/recipe [put]
func (h *InventoryHandler) SaveRecipe(c *gin.Context) {
	var req dto.RecipeRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	items := make([]inventory.RecipeItem, len(req.Items))
	for i, ri := range req.Items {
		items[i] = inventory.RecipeItem{InventoryItemID: ri.InventoryItemID, Qty: ri.Qty}
	}
	recipe, err := h.inventory.SaveRecipe(c.Request.Context(), tenantID, userID, c.Param("id"), items)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "recipe saved", recipe)
}
