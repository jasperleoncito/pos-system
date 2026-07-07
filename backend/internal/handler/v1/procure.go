package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/procure"
	"github.com/jasperleoncito/pos-system/backend/internal/dto"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

// ProcureHandler exposes suppliers, purchase orders, and stock alerts.
type ProcureHandler struct {
	procure *service.ProcureService
}

func NewProcureHandler(p *service.ProcureService) *ProcureHandler { return &ProcureHandler{procure: p} }

// ListSuppliers godoc
//
//	@Summary	List suppliers
//	@Tags		procurement
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/suppliers [get]
func (h *ProcureHandler) ListSuppliers(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	suppliers, err := h.procure.ListSuppliers(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", suppliers)
}

func supplierFromRequest(req dto.SupplierRequest) *procure.Supplier {
	return &procure.Supplier{
		Name: req.Name, ContactPerson: req.ContactPerson, Phone: req.Phone,
		Email: req.Email, Address: req.Address, Notes: req.Notes,
		IsActive: boolOrDefault(req.IsActive, true),
	}
}

// CreateSupplier godoc
//
//	@Summary	Create a supplier
//	@Tags		procurement
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.SupplierRequest	true	"Supplier"
//	@Success	201		{object}	response.Envelope
//	@Router		/suppliers [post]
func (h *ProcureHandler) CreateSupplier(c *gin.Context) {
	var req dto.SupplierRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	supplier := supplierFromRequest(req)
	if err := h.procure.CreateSupplier(c.Request.Context(), tenantID, userID, supplier); err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "supplier created", supplier)
}

// UpdateSupplier godoc
//
//	@Summary	Update a supplier
//	@Tags		procurement
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Supplier ID"
//	@Param		payload	body		dto.SupplierRequest	true	"Supplier"
//	@Success	200		{object}	response.Envelope
//	@Router		/suppliers/{id} [put]
func (h *ProcureHandler) UpdateSupplier(c *gin.Context) {
	var req dto.SupplierRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	supplier := supplierFromRequest(req)
	supplier.ID = c.Param("id")
	if err := h.procure.UpdateSupplier(c.Request.Context(), tenantID, userID, supplier); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "supplier updated", supplier)
}

// DeleteSupplier godoc
//
//	@Summary	Delete a supplier
//	@Tags		procurement
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Supplier ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/suppliers/{id} [delete]
func (h *ProcureHandler) DeleteSupplier(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	if err := h.procure.DeleteSupplier(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "supplier deleted", nil)
}

// ListPOs godoc
//
//	@Summary	List purchase orders
//	@Tags		procurement
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/purchase-orders [get]
func (h *ProcureHandler) ListPOs(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	pos, err := h.procure.ListPOs(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", pos)
}

// GetPO godoc
//
//	@Summary	Get one purchase order with lines
//	@Tags		procurement
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"PO ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/purchase-orders/{id} [get]
func (h *ProcureHandler) GetPO(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	po, err := h.procure.GetPO(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", po)
}

// CreatePO godoc
//
//	@Summary	Create a draft purchase order
//	@Tags		procurement
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.CreatePORequest	true	"Purchase order"
//	@Success	201		{object}	response.Envelope
//	@Router		/purchase-orders [post]
func (h *ProcureHandler) CreatePO(c *gin.Context) {
	var req dto.CreatePORequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	po := &procure.PurchaseOrder{SupplierID: req.SupplierID, Notes: req.Notes}
	for _, it := range req.Items {
		po.Items = append(po.Items, procure.POItem{ItemID: it.ItemID, QtyOrdered: it.Qty, UnitCost: it.UnitCost})
	}
	created, err := h.procure.CreatePO(c.Request.Context(), tenantID, userID, po)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "purchase order created", created)
}

// OrderPO godoc
//
//	@Summary	Mark a draft PO as ordered
//	@Tags		procurement
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"PO ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/purchase-orders/{id}/order [post]
func (h *ProcureHandler) OrderPO(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	po, err := h.procure.MarkOrdered(c.Request.Context(), tenantID, userID, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "purchase order marked ordered", po)
}

// CancelPO godoc
//
//	@Summary	Cancel a purchase order
//	@Tags		procurement
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"PO ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/purchase-orders/{id}/cancel [post]
func (h *ProcureHandler) CancelPO(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	po, err := h.procure.CancelPO(c.Request.Context(), tenantID, userID, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "purchase order cancelled", po)
}

// ReceivePO godoc
//
//	@Summary	Receive quantities against a PO (stock in + cost update)
//	@Tags		procurement
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"PO ID"
//	@Param		payload	body		dto.ReceivePORequest	true	"Received lines"
//	@Success	200		{object}	response.Envelope
//	@Router		/purchase-orders/{id}/receive [post]
func (h *ProcureHandler) ReceivePO(c *gin.Context) {
	var req dto.ReceivePORequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	lines := make([]service.ReceiveLine, len(req.Lines))
	for i, l := range req.Lines {
		lines[i] = service.ReceiveLine{POItemID: l.POItemID, Qty: l.Qty}
	}
	po, err := h.procure.Receive(c.Request.Context(), tenantID, userID, c.Param("id"), lines)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "stock received", po)
}

// ListAlerts godoc
//
//	@Summary	List stock alerts
//	@Tags		procurement
//	@Security	BearerAuth
//	@Produce	json
//	@Param		open	query		bool	false	"Only unacknowledged"
//	@Success	200		{object}	response.Envelope
//	@Router		/inventory/alerts [get]
func (h *ProcureHandler) ListAlerts(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	alerts, err := h.procure.ListAlerts(c.Request.Context(), tenantID, c.DefaultQuery("open", "true") == "true")
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", alerts)
}

// AckAlert godoc
//
//	@Summary	Acknowledge a stock alert
//	@Tags		procurement
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Alert ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/inventory/alerts/{id}/ack [post]
func (h *ProcureHandler) AckAlert(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	if err := h.procure.AckAlert(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "alert acknowledged", nil)
}
