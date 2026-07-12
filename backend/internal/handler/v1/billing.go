package v1

import (
	"crypto/subtle"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/dto"
	"github.com/jasperleoncito/pos-system/backend/internal/middleware"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/xendit"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

type BillingHandler struct {
	billing      *service.BillingService
	webhookToken string
}

func NewBillingHandler(billing *service.BillingService, webhookToken string) *BillingHandler {
	return &BillingHandler{billing: billing, webhookToken: webhookToken}
}

func pageParams(c *gin.Context) (page, limit int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit
}

// GetPlans godoc
//
//	@Summary	Current subscription prices (public — shown at registration)
//	@Tags		billing
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/billing/plans [get]
func (h *BillingHandler) GetPlans(c *gin.Context) {
	plans, err := h.billing.GetPlans(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", plans)
}

// GetSubscription godoc
//
//	@Summary	The active business's subscription (any member)
//	@Tags		billing
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/billing/subscription [get]
func (h *BillingHandler) GetSubscription(c *gin.Context) {
	sub, err := h.billing.Subscription(c.Request.Context(), c.GetString(middleware.CtxTenantID))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", sub)
}

// Reconcile godoc
//
//	@Summary	Confirm the latest pending payment directly with Xendit
//	@Description Webhook-independent: asks Xendit if the pending invoice is paid and activates the subscription. Polled by the payment-return page.
//	@Tags		billing
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/billing/reconcile [post]
func (h *BillingHandler) Reconcile(c *gin.Context) {
	sub, err := h.billing.ReconcilePending(c.Request.Context(), c.GetString(middleware.CtxTenantID))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "reconciled", sub)
}

// CreateCheckout godoc
//
//	@Summary	Create (or reuse) a Xendit invoice for the next period
//	@Tags		billing
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.CheckoutRequest	true	"Plan"
//	@Success	200		{object}	response.Envelope
//	@Router		/billing/checkout [post]
func (h *BillingHandler) CreateCheckout(c *gin.Context) {
	var req dto.CheckoutRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.billing.CreateCheckout(c.Request.Context(),
		c.GetString(middleware.CtxTenantID), c.GetString(middleware.CtxUserID), req.Plan)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "checkout ready", result)
}

// ListPayments godoc
//
//	@Summary	The active business's payment history (owner)
//	@Tags		billing
//	@Security	BearerAuth
//	@Produce	json
//	@Param		page	query		int	false	"Page (1-based)"
//	@Param		limit	query		int	false	"Page size"
//	@Success	200		{object}	response.Envelope
//	@Router		/billing/payments [get]
func (h *BillingHandler) ListPayments(c *gin.Context) {
	page, limit := pageParams(c)
	payments, total, err := h.billing.ListPayments(c.Request.Context(),
		c.GetString(middleware.CtxTenantID), limit, (page-1)*limit)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Paginated(c, "", payments, response.Meta{Total: total, Page: page, Limit: limit})
}

// Webhook godoc
//
//	@Summary	Xendit invoice callback (verified by x-callback-token)
//	@Tags		billing
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Failure	401	{object}	response.ErrorEnvelope
//	@Router		/webhooks/xendit [post]
func (h *BillingHandler) Webhook(c *gin.Context) {
	token := c.GetHeader("x-callback-token")
	if h.webhookToken == "" ||
		subtle.ConstantTimeCompare([]byte(token), []byte(h.webhookToken)) != 1 {
		response.Error(c, http.StatusUnauthorized, "invalid callback token")
		return
	}

	var cb xendit.InvoiceCallback
	if err := c.ShouldBindJSON(&cb); err != nil {
		// Malformed body: acknowledge so Xendit doesn't retry forever.
		response.OK(c, "ignored", nil)
		return
	}
	if err := h.billing.HandleWebhook(c.Request.Context(), cb); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "processed", nil)
}

// ---- super-admin console ----

// AdminListSubscriptions godoc
//
//	@Summary	All subscriptions with owner details (super admin)
//	@Tags		admin
//	@Security	BearerAuth
//	@Produce	json
//	@Param		status	query		string	false	"Filter: pending|active|inactive"
//	@Param		page	query		int		false	"Page (1-based)"
//	@Param		limit	query		int		false	"Page size"
//	@Success	200		{object}	response.Envelope
//	@Router		/admin/subscriptions [get]
func (h *BillingHandler) AdminListSubscriptions(c *gin.Context) {
	page, limit := pageParams(c)
	subs, total, err := h.billing.ListSubscriptions(c.Request.Context(),
		c.Query("status"), limit, (page-1)*limit)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Paginated(c, "", subs, response.Meta{Total: total, Page: page, Limit: limit})
}

// AdminListOwners godoc
//
//	@Summary	All business owners and their businesses (super admin)
//	@Tags		admin
//	@Security	BearerAuth
//	@Produce	json
//	@Param		page	query		int	false	"Page (1-based)"
//	@Param		limit	query		int	false	"Page size"
//	@Success	200		{object}	response.Envelope
//	@Router		/admin/owners [get]
func (h *BillingHandler) AdminListOwners(c *gin.Context) {
	page, limit := pageParams(c)
	owners, total, err := h.billing.ListOwners(c.Request.Context(), limit, (page-1)*limit)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Paginated(c, "", owners, response.Meta{Total: total, Page: page, Limit: limit})
}

// AdminBillingStats godoc
//
//	@Summary	Platform billing counters (super admin)
//	@Tags		admin
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/admin/billing/stats [get]
func (h *BillingHandler) AdminBillingStats(c *gin.Context) {
	stats, err := h.billing.BillingStats(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", stats)
}

// AdminMarkPaid godoc
//
//	@Summary	Record a manual payment and extend the subscription (super admin)
//	@Tags		admin
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		tenantId	path		string					true	"Tenant ID"
//	@Param		payload		body		dto.MarkPaidManualRequest	true	"Optional note"
//	@Success	200			{object}	response.Envelope
//	@Router		/admin/subscriptions/{tenantId}/mark-paid [post]
func (h *BillingHandler) AdminMarkPaid(c *gin.Context) {
	var req dto.MarkPaidManualRequest
	if !bindJSON(c, &req) {
		return
	}
	sub, err := h.billing.MarkPaidManual(c.Request.Context(),
		c.GetString(middleware.CtxUserID), c.Param("tenantId"), req.Note)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "payment recorded — subscription extended", sub)
}

// AdminSetSubscriptionStatus godoc
//
//	@Summary	Force a subscription active or inactive (super admin)
//	@Tags		admin
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		tenantId	path		string							true	"Tenant ID"
//	@Param		payload		body		dto.SetSubscriptionStatusRequest	true	"New status"
//	@Success	200			{object}	response.Envelope
//	@Router		/admin/subscriptions/{tenantId}/status [patch]
func (h *BillingHandler) AdminSetSubscriptionStatus(c *gin.Context) {
	var req dto.SetSubscriptionStatusRequest
	if !bindJSON(c, &req) {
		return
	}
	sub, err := h.billing.SetSubscriptionStatus(c.Request.Context(),
		c.GetString(middleware.CtxUserID), c.Param("tenantId"), req.Status)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "subscription status updated", sub)
}

// AdminGetPrices godoc
//
//	@Summary	Current platform prices (super admin)
//	@Tags		admin
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/admin/billing/settings [get]
func (h *BillingHandler) AdminGetPrices(c *gin.Context) {
	settings, err := h.billing.GetPlatformSettings(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", settings)
}

// AdminUpdatePrices godoc
//
//	@Summary	Update subscription prices (super admin)
//	@Tags		admin
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.UpdatePlatformPricesRequest	true	"Prices in centavos"
//	@Success	200		{object}	response.Envelope
//	@Router		/admin/billing/settings [put]
func (h *BillingHandler) AdminUpdatePrices(c *gin.Context) {
	var req dto.UpdatePlatformPricesRequest
	if !bindJSON(c, &req) {
		return
	}
	settings, err := h.billing.UpdatePrices(c.Request.Context(),
		c.GetString(middleware.CtxUserID), req.MonthlyPrice, req.YearlyPrice)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "prices updated", settings)
}
