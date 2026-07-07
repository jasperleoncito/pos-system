package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/rbac"
	"github.com/jasperleoncito/pos-system/backend/internal/dto"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/token"
	"github.com/jasperleoncito/pos-system/backend/internal/realtime"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

const sseHeartbeatInterval = 25 * time.Second

// KitchenHandler exposes the kitchen display queue and SSE stream.
type KitchenHandler struct {
	orders *service.OrderService
	hub    *realtime.Hub
	tokens *token.Manager
}

func NewKitchenHandler(orders *service.OrderService, hub *realtime.Hub, tokens *token.Manager) *KitchenHandler {
	return &KitchenHandler{orders: orders, hub: hub, tokens: tokens}
}

// ListOrders godoc
//
//	@Summary	List active kitchen tickets
//	@Tags		kitchen
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/kitchen/orders [get]
func (h *KitchenHandler) ListOrders(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	orders, err := h.orders.ListKitchenOrders(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", orders)
}

// SetStatus godoc
//
//	@Summary	Advance a ticket's kitchen status
//	@Tags		kitchen
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"Order ID"
//	@Param		payload	body		dto.KitchenStatusRequest	true	"New status"
//	@Success	200		{object}	response.Envelope
//	@Router		/kitchen/orders/{id}/status [patch]
func (h *KitchenHandler) SetStatus(c *gin.Context) {
	var req dto.KitchenStatusRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	o, err := h.orders.SetKitchenStatus(c.Request.Context(), tenantID, userID, c.Param("id"), req.Status)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "kitchen status updated", o)
}

// SetItemStatus godoc
//
//	@Summary	Set one item's kitchen status
//	@Tags		kitchen
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"Order ID"
//	@Param		itemId	path		string						true	"Order item ID"
//	@Param		payload	body		dto.KitchenStatusRequest	true	"New status"
//	@Success	200		{object}	response.Envelope
//	@Router		/kitchen/orders/{id}/items/{itemId}/status [patch]
func (h *KitchenHandler) SetItemStatus(c *gin.Context) {
	var req dto.KitchenStatusRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	o, err := h.orders.SetItemStatus(c.Request.Context(), tenantID, userID, c.Param("id"), c.Param("itemId"), req.Status)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "item status updated", o)
}

// SetPriority godoc
//
//	@Summary	Flag or unflag a rush order
//	@Tags		kitchen
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Order ID"
//	@Param		payload	body		dto.PriorityRequest	true	"Priority flag"
//	@Success	200		{object}	response.Envelope
//	@Router		/orders/{id}/priority [post]
func (h *KitchenHandler) SetPriority(c *gin.Context) {
	var req dto.PriorityRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	o, err := h.orders.SetPriority(c.Request.Context(), tenantID, userID, c.Param("id"), req.Priority)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "priority updated", o)
}

// Stream godoc
//
//	@Summary		Kitchen event stream (SSE)
//	@Description	EventSource cannot send headers, so the access token travels as ?token=
//	@Tags			kitchen
//	@Produce		text/event-stream
//	@Param			token	query	string	true	"Access token"
//	@Success		200
//	@Failure		401	{object}	response.ErrorEnvelope
//	@Router			/kitchen/stream [get]
func (h *KitchenHandler) Stream(c *gin.Context) {
	claims, err := h.tokens.ParseAccessToken(c.Query("token"))
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "invalid or expired token")
		return
	}
	if claims.TenantID == "" {
		response.Error(c, http.StatusForbidden, "no active business selected")
		return
	}
	if !claims.IsSuperAdmin && !rbac.Can(rbac.Role(claims.Role), rbac.PermKitchenRead) &&
		!rbac.Can(rbac.Role(claims.Role), rbac.PermOrdersCreate) {
		response.Error(c, http.StatusForbidden, "you do not have permission to do that")
		return
	}

	events, unsubscribe := h.hub.Subscribe(claims.TenantID)
	defer unsubscribe()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.Flush()

	// Initial hello so clients know the stream is live.
	fmt.Fprintf(c.Writer, "event: hello\ndata: {}\n\n")
	c.Writer.Flush()

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-heartbeat.C:
			// Comment line keeps proxies from closing the connection.
			fmt.Fprint(c.Writer, ": ping\n\n")
			c.Writer.Flush()
		case event, ok := <-events:
			if !ok {
				return
			}
			payload, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(c.Writer, "event: kitchen\ndata: %s\n\n", payload)
			c.Writer.Flush()
		}
	}
}
