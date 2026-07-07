package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/order"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/storage"
	"github.com/jasperleoncito/pos-system/backend/internal/dto"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

// OrderHandler exposes the POS order and cash drawer endpoints.
type OrderHandler struct {
	orders *service.OrderService
	store  storage.ObjectStorage
}

func NewOrderHandler(orders *service.OrderService, store storage.ObjectStorage) *OrderHandler {
	return &OrderHandler{orders: orders, store: store}
}

// CreateOrder godoc
//
//	@Summary	Create an order (priced server-side from the catalog)
//	@Tags		orders
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.CreateOrderRequest	true	"Cart"
//	@Success	201		{object}	response.Envelope
//	@Router		/orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req dto.CreateOrderRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)

	items := make([]service.CreateOrderItemInput, len(req.Items))
	for i, it := range req.Items {
		items[i] = service.CreateOrderItemInput{
			ProductID: it.ProductID, VariantID: it.VariantID, Qty: it.Qty,
			ModifierIDs: it.ModifierIDs, Notes: it.Notes,
		}
	}

	o, err := h.orders.CreateOrder(c.Request.Context(), tenantID, userID, service.CreateOrderInput{
		OrderType: req.OrderType, TableNumber: req.TableNumber,
		Notes: req.Notes, Hold: req.Hold, DiscountID: req.DiscountID,
		CouponCode: req.CouponCode, Items: items,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "order created", o)
}

// ListOrders godoc
//
//	@Summary	List orders
//	@Tags		orders
//	@Security	BearerAuth
//	@Produce	json
//	@Param		status	query		string	false	"Filter by status"
//	@Param		search	query		string	false	"Order number"
//	@Param		page	query		int		false	"Page"
//	@Param		limit	query		int		false	"Page size"
//	@Success	200		{object}	response.Envelope
//	@Router		/orders [get]
func (h *OrderHandler) ListOrders(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 25
	}

	orders, total, err := h.orders.ListOrders(c.Request.Context(), tenantID, order.Filter{
		Status: c.Query("status"),
		Search: c.Query("search"),
		Limit:  limit,
		Offset: (page - 1) * limit,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Paginated(c, "", orders, response.Meta{Total: total, Page: page, Limit: limit})
}

// GetOrder godoc
//
//	@Summary	Get one order with items and payments
//	@Tags		orders
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Order ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/orders/{id} [get]
func (h *OrderHandler) GetOrder(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	o, err := h.orders.GetOrder(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", o)
}

// SetHold godoc
//
//	@Summary	Hold or resume an unpaid order
//	@Tags		orders
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Order ID"
//	@Param		payload	body		dto.HoldOrderRequest	true	"Hold flag"
//	@Success	200		{object}	response.Envelope
//	@Router		/orders/{id}/hold [post]
func (h *OrderHandler) SetHold(c *gin.Context) {
	var req dto.HoldOrderRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	o, err := h.orders.SetHold(c.Request.Context(), tenantID, userID, c.Param("id"), req.Hold)
	if err != nil {
		respondError(c, err)
		return
	}
	message := "order resumed"
	if req.Hold {
		message = "order held"
	}
	response.OK(c, message, o)
}

// Pay godoc
//
//	@Summary	Settle an order with one or more payments (mixed methods)
//	@Tags		orders
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Order ID"
//	@Param		payload	body		dto.PayOrderRequest	true	"Payments"
//	@Success	200		{object}	response.Envelope
//	@Failure	422		{object}	response.ErrorEnvelope
//	@Router		/orders/{id}/payments [post]
func (h *OrderHandler) Pay(c *gin.Context) {
	var req dto.PayOrderRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)

	payments := make([]service.PaymentInput, len(req.Payments))
	for i, p := range req.Payments {
		payments[i] = service.PaymentInput{Method: p.Method, Amount: p.Amount, ReferenceNo: p.ReferenceNo}
	}

	o, err := h.orders.Pay(c.Request.Context(), tenantID, userID, c.Param("id"), payments)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "payment recorded — order completed", o)
}

// GetReceipt godoc
//
//	@Summary	Get printable receipt data (order + branding)
//	@Tags		orders
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Order ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/orders/{id}/receipt [get]
func (h *OrderHandler) GetReceipt(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	receipt, err := h.orders.GetReceipt(c.Request.Context(), tenantID, c.Param("id"), h.store.PublicURL)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", receipt)
}

// OpenDrawer godoc
//
//	@Summary	Open the cash drawer with an opening float
//	@Tags		cash-drawer
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.OpenDrawerRequest	true	"Opening float (centavos)"
//	@Success	201		{object}	response.Envelope
//	@Failure	409		{object}	response.ErrorEnvelope
//	@Router		/cash-drawer/open [post]
func (h *OrderHandler) OpenDrawer(c *gin.Context) {
	var req dto.OpenDrawerRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	session, err := h.orders.OpenDrawer(c.Request.Context(), tenantID, userID, req.OpeningFloat)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "drawer opened", session)
}

// CurrentDrawer godoc
//
//	@Summary	Get the open drawer session with its movements
//	@Tags		cash-drawer
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Failure	404	{object}	response.ErrorEnvelope
//	@Router		/cash-drawer/current [get]
func (h *OrderHandler) CurrentDrawer(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	session, movements, err := h.orders.CurrentDrawer(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", gin.H{"session": session, "movements": movements})
}

// CloseDrawer godoc
//
//	@Summary	Close the drawer with counted cash (variance computed)
//	@Tags		cash-drawer
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.CloseDrawerRequest	true	"Counted cash (centavos)"
//	@Success	200		{object}	response.Envelope
//	@Router		/cash-drawer/close [post]
func (h *OrderHandler) CloseDrawer(c *gin.Context) {
	var req dto.CloseDrawerRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	session, err := h.orders.CloseDrawer(c.Request.Context(), tenantID, userID, req.CountedCash)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "drawer closed", session)
}
