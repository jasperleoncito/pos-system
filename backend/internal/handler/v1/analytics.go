package v1

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/analytics"
	"github.com/jasperleoncito/pos-system/backend/internal/dto"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

// AnalyticsHandler exposes the sales dashboard and expenses.
type AnalyticsHandler struct {
	analytics *service.AnalyticsService
}

func NewAnalyticsHandler(a *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analytics: a}
}

// Overview godoc
//
//	@Summary	Today/WTD/MTD/YTD stat cards with previous-period deltas
//	@Tags		analytics
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/analytics/overview [get]
func (h *AnalyticsHandler) Overview(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	stats, err := h.analytics.Overview(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", stats)
}

// Dashboard godoc
//
//	@Summary	Full dashboard payload for a date range
//	@Tags		analytics
//	@Security	BearerAuth
//	@Produce	json
//	@Param		from	query		string	false	"YYYY-MM-DD (tenant-local, inclusive)"
//	@Param		to		query		string	false	"YYYY-MM-DD (tenant-local, inclusive)"
//	@Success	200		{object}	response.Envelope
//	@Router		/analytics/dashboard [get]
func (h *AnalyticsHandler) Dashboard(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	d, err := h.analytics.GetDashboard(c.Request.Context(), tenantID, c.Query("from"), c.Query("to"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", d)
}

// ---- expenses ----

func expenseFromRequest(req dto.ExpenseRequest) *analytics.Expense {
	e := &analytics.Expense{
		Category: req.Category, Description: req.Description, Amount: req.Amount,
		ExpenseDate: time.Now(),
	}
	if req.ExpenseDate != "" {
		if d, err := time.Parse("2006-01-02", req.ExpenseDate); err == nil {
			e.ExpenseDate = d
		}
	}
	return e
}

// ListExpenses godoc
//
//	@Summary	List expenses in a date range
//	@Tags		analytics
//	@Security	BearerAuth
//	@Produce	json
//	@Param		from	query		string	false	"YYYY-MM-DD"
//	@Param		to		query		string	false	"YYYY-MM-DD"
//	@Success	200		{object}	response.Envelope
//	@Router		/expenses [get]
func (h *AnalyticsHandler) ListExpenses(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	expenses, err := h.analytics.ListExpenses(c.Request.Context(), tenantID, c.Query("from"), c.Query("to"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", expenses)
}

// CreateExpense godoc
//
//	@Summary	Record an expense
//	@Tags		analytics
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.ExpenseRequest	true	"Expense"
//	@Success	201		{object}	response.Envelope
//	@Router		/expenses [post]
func (h *AnalyticsHandler) CreateExpense(c *gin.Context) {
	var req dto.ExpenseRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	created, err := h.analytics.CreateExpense(c.Request.Context(), tenantID, userID, expenseFromRequest(req))
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "expense recorded", created)
}

// UpdateExpense godoc
//
//	@Summary	Update an expense
//	@Tags		analytics
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Expense ID"
//	@Param		payload	body		dto.ExpenseRequest	true	"Expense"
//	@Success	200		{object}	response.Envelope
//	@Router		/expenses/{id} [put]
func (h *AnalyticsHandler) UpdateExpense(c *gin.Context) {
	var req dto.ExpenseRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	e := expenseFromRequest(req)
	e.ID = c.Param("id")
	updated, err := h.analytics.UpdateExpense(c.Request.Context(), tenantID, userID, e)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "expense updated", updated)
}

// DeleteExpense godoc
//
//	@Summary	Delete an expense
//	@Tags		analytics
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Expense ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/expenses/{id} [delete]
func (h *AnalyticsHandler) DeleteExpense(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	if err := h.analytics.DeleteExpense(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "expense deleted", nil)
}
