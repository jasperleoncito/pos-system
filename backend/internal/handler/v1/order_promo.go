package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/dto"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

// CreateSplits godoc
//
//	@Summary	Split an unpaid order into per-person amounts
//	@Tags		orders
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Order ID"
//	@Param		payload	body		dto.CreateSplitsRequest	true	"Split amounts (centavos, must sum to total)"
//	@Success	201		{object}	response.Envelope
//	@Router		/orders/{id}/splits [post]
func (h *OrderHandler) CreateSplits(c *gin.Context) {
	var req dto.CreateSplitsRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	o, err := h.orders.CreateSplits(c.Request.Context(), tenantID, userID, c.Param("id"), req.Amounts)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "order split", o)
}

// PaySplit godoc
//
//	@Summary	Settle one split of a split-billed order
//	@Tags		orders
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Order ID"
//	@Param		splitId	path		string				true	"Split ID"
//	@Param		payload	body		dto.PayOrderRequest	true	"Payments"
//	@Success	200		{object}	response.Envelope
//	@Router		/orders/{id}/splits/{splitId}/payments [post]
func (h *OrderHandler) PaySplit(c *gin.Context) {
	var req dto.PayOrderRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)

	payments := make([]service.PaymentInput, len(req.Payments))
	for i, p := range req.Payments {
		payments[i] = service.PaymentInput{Method: p.Method, Amount: p.Amount, ReferenceNo: p.ReferenceNo}
	}

	o, err := h.orders.PaySplit(c.Request.Context(), tenantID, userID, c.Param("id"), c.Param("splitId"), payments)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "split paid", o)
}

// Refund godoc
//
//	@Summary	Refund a completed order (full, by items, or custom amount)
//	@Tags		orders
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Order ID"
//	@Param		payload	body		dto.RefundRequest	true	"Refund details"
//	@Success	200		{object}	response.Envelope
//	@Router		/orders/{id}/refunds [post]
func (h *OrderHandler) Refund(c *gin.Context) {
	var req dto.RefundRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)

	items := make([]service.RefundItemInput, len(req.Items))
	for i, it := range req.Items {
		items[i] = service.RefundItemInput{OrderItemID: it.OrderItemID, Qty: it.Qty}
	}

	o, err := h.orders.Refund(c.Request.Context(), tenantID, userID, c.Param("id"), req.Reason, items, req.Amount)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "refund issued", o)
}

// Void godoc
//
//	@Summary	Void an order (manager only; cash returned, coupon released)
//	@Tags		orders
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string			true	"Order ID"
//	@Param		payload	body		dto.VoidRequest	true	"Void reason"
//	@Success	200		{object}	response.Envelope
//	@Router		/orders/{id}/void [post]
func (h *OrderHandler) Void(c *gin.Context) {
	var req dto.VoidRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	o, err := h.orders.Void(c.Request.Context(), tenantID, userID, c.Param("id"), req.Reason)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "order voided", o)
}
