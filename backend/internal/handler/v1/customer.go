package v1

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/customer"
	"github.com/jasperleoncito/pos-system/backend/internal/dto"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

// CustomerHandler exposes customer profiles and the loyalty program.
type CustomerHandler struct {
	loyalty *service.LoyaltyService
}

func NewCustomerHandler(l *service.LoyaltyService) *CustomerHandler {
	return &CustomerHandler{loyalty: l}
}

func customerFromRequest(req dto.CustomerRequest) *customer.Customer {
	c := &customer.Customer{
		FullName: req.FullName, Phone: req.Phone, Email: req.Email,
		Notes: req.Notes, IsActive: boolOrDefault(req.IsActive, true),
	}
	if req.Birthday != "" {
		if d, err := time.Parse("2006-01-02", req.Birthday); err == nil {
			c.Birthday = &d
		}
	}
	return c
}

// ListCustomers godoc
//
//	@Summary	List customers
//	@Tags		customers
//	@Security	BearerAuth
//	@Produce	json
//	@Param		search	query		string	false	"Name, phone, or email filter"
//	@Success	200		{object}	response.Envelope
//	@Router		/customers [get]
func (h *CustomerHandler) ListCustomers(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	customers, err := h.loyalty.ListCustomers(c.Request.Context(), tenantID, c.Query("search"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", customers)
}

// GetCustomer godoc
//
//	@Summary	Get one customer
//	@Tags		customers
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Customer ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/customers/{id} [get]
func (h *CustomerHandler) GetCustomer(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	cust, err := h.loyalty.GetCustomer(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", cust)
}

// CreateCustomer godoc
//
//	@Summary	Create a customer
//	@Tags		customers
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.CustomerRequest	true	"Customer"
//	@Success	201		{object}	response.Envelope
//	@Router		/customers [post]
func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var req dto.CustomerRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	created, err := h.loyalty.CreateCustomer(c.Request.Context(), tenantID, userID, customerFromRequest(req))
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "customer created", created)
}

// UpdateCustomer godoc
//
//	@Summary	Update a customer
//	@Tags		customers
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Customer ID"
//	@Param		payload	body		dto.CustomerRequest	true	"Customer"
//	@Success	200		{object}	response.Envelope
//	@Router		/customers/{id} [put]
func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	var req dto.CustomerRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	cust := customerFromRequest(req)
	cust.ID = c.Param("id")
	updated, err := h.loyalty.UpdateCustomer(c.Request.Context(), tenantID, userID, cust)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "customer updated", updated)
}

// DeleteCustomer godoc
//
//	@Summary	Delete a customer
//	@Tags		customers
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Customer ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/customers/{id} [delete]
func (h *CustomerHandler) DeleteCustomer(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	if err := h.loyalty.DeleteCustomer(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "customer deleted", nil)
}

// ListLoyaltyTransactions godoc
//
//	@Summary	A customer's loyalty points history
//	@Tags		customers
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id		path		string	true	"Customer ID"
//	@Param		limit	query		int		false	"Max rows (default 50)"
//	@Success	200		{object}	response.Envelope
//	@Router		/customers/{id}/loyalty [get]
func (h *CustomerHandler) ListLoyaltyTransactions(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	txs, err := h.loyalty.ListTransactions(c.Request.Context(), tenantID, c.Param("id"), limit)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", txs)
}

// GetLoyaltySettings godoc
//
//	@Summary	Loyalty program settings
//	@Tags		customers
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/loyalty/settings [get]
func (h *CustomerHandler) GetLoyaltySettings(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	settings, err := h.loyalty.GetSettings(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", settings)
}

// UpdateLoyaltySettings godoc
//
//	@Summary	Update loyalty program settings (manager+)
//	@Tags		customers
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.LoyaltySettingsRequest	true	"Settings"
//	@Success	200		{object}	response.Envelope
//	@Router		/loyalty/settings [put]
func (h *CustomerHandler) UpdateLoyaltySettings(c *gin.Context) {
	var req dto.LoyaltySettingsRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	settings, err := h.loyalty.SaveSettings(c.Request.Context(), tenantID, userID, &customer.Settings{
		IsEnabled: boolOrDefault(req.IsEnabled, true),
		EarnRate:  req.EarnRate, RedeemValue: req.RedeemValue,
		SilverThreshold: req.SilverThreshold, GoldThreshold: req.GoldThreshold, VIPThreshold: req.VIPThreshold,
		SilverMultiplier: req.SilverMultiplier, GoldMultiplier: req.GoldMultiplier, VIPMultiplier: req.VIPMultiplier,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "loyalty settings saved", settings)
}
