package v1

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/dto"
	"github.com/jasperleoncito/pos-system/backend/internal/middleware"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/imageproc"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

type TenantHandler struct {
	tenants *service.TenantService
}

func NewTenantHandler(tenants *service.TenantService) *TenantHandler {
	return &TenantHandler{tenants: tenants}
}

// GetSettings godoc
//
//	@Summary	Get the active business's branding settings
//	@Tags		tenant
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/tenant/settings [get]
func (h *TenantHandler) GetSettings(c *gin.Context) {
	settings, err := h.tenants.GetSettings(c.Request.Context(), c.GetString(middleware.CtxTenantID))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", settings)
}

// UpdateSettings godoc
//
//	@Summary	Update branding settings
//	@Tags		tenant
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.UpdateTenantSettingsRequest	true	"Branding fields"
//	@Success	200		{object}	response.Envelope
//	@Router		/tenant/settings [put]
func (h *TenantHandler) UpdateSettings(c *gin.Context) {
	var req dto.UpdateTenantSettingsRequest
	if !bindJSON(c, &req) {
		return
	}
	settings, err := h.tenants.UpdateSettings(c.Request.Context(),
		c.GetString(middleware.CtxTenantID), c.GetString(middleware.CtxUserID),
		service.UpdateSettingsInput{
			PrimaryColor:   req.PrimaryColor,
			SecondaryColor: req.SecondaryColor,
			AccentColor:    req.AccentColor,
			ReceiptHeader:  req.ReceiptHeader,
			ReceiptFooter:  req.ReceiptFooter,
			ContactNumber:  req.ContactNumber,
			Facebook:       req.Facebook,
			Website:        req.Website,
			Address:        req.Address,
			TaxLabel:       req.TaxLabel,
			TaxID:          req.TaxID,
		})
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "branding updated", settings)
}

// UploadLogo godoc
//
//	@Summary	Upload the business logo (optimized to WebP automatically)
//	@Tags		tenant
//	@Security	BearerAuth
//	@Accept		multipart/form-data
//	@Produce	json
//	@Param		logo	formData	file	true	"PNG/JPG/WEBP, max 10MB"
//	@Success	200		{object}	response.Envelope
//	@Failure	422		{object}	response.ErrorEnvelope
//	@Router		/tenant/logo [post]
func (h *TenantHandler) UploadLogo(c *gin.Context) {
	file, _, err := c.Request.FormFile("logo")
	if err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "attach an image file in the 'logo' field")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, imageproc.MaxUploadBytes+1))
	if err != nil {
		respondError(c, err)
		return
	}

	settings, err := h.tenants.UploadLogo(c.Request.Context(),
		c.GetString(middleware.CtxTenantID), c.GetString(middleware.CtxUserID), data)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "logo updated", settings)
}

// AdminListTenants godoc
//
//	@Summary	List all tenants (super admin)
//	@Tags		admin
//	@Security	BearerAuth
//	@Produce	json
//	@Param		page	query		int	false	"Page (1-based)"
//	@Param		limit	query		int	false	"Page size"
//	@Success	200		{object}	response.Envelope
//	@Router		/admin/tenants [get]
func (h *TenantHandler) AdminListTenants(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	tenants, total, err := h.tenants.ListTenants(c.Request.Context(), limit, (page-1)*limit)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Paginated(c, "", tenants, response.Meta{Total: total, Page: page, Limit: limit})
}

// AdminSetTenantStatus godoc
//
//	@Summary	Activate or suspend a tenant (super admin)
//	@Tags		admin
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"Tenant ID"
//	@Param		payload	body		dto.SetTenantStatusRequest	true	"New status"
//	@Success	200		{object}	response.Envelope
//	@Router		/admin/tenants/{id}/status [patch]
func (h *TenantHandler) AdminSetTenantStatus(c *gin.Context) {
	var req dto.SetTenantStatusRequest
	if !bindJSON(c, &req) {
		return
	}
	t, err := h.tenants.SetTenantStatus(c.Request.Context(),
		c.GetString(middleware.CtxUserID), c.Param("id"), req.Status)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "tenant status updated", t)
}

// AdminSetTenantPlan godoc
//
//	@Summary	Change a tenant's subscription plan (super admin)
//	@Tags		admin
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"Tenant ID"
//	@Param		payload	body		dto.SetTenantPlanRequest	true	"New plan"
//	@Success	200		{object}	response.Envelope
//	@Router		/admin/tenants/{id}/plan [patch]
func (h *TenantHandler) AdminSetTenantPlan(c *gin.Context) {
	var req dto.SetTenantPlanRequest
	if !bindJSON(c, &req) {
		return
	}
	t, err := h.tenants.SetTenantPlan(c.Request.Context(),
		c.GetString(middleware.CtxUserID), c.Param("id"), req.Plan)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "tenant plan updated", t)
}

// AdminStats godoc
//
//	@Summary	Platform-wide counters (super admin)
//	@Tags		admin
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/admin/stats [get]
func (h *TenantHandler) AdminStats(c *gin.Context) {
	stats, err := h.tenants.PlatformStats(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", stats)
}

// AdminSales godoc
//
//	@Summary	Platform-wide sales analytics (super admin)
//	@Tags		admin
//	@Security	BearerAuth
//	@Produce	json
//	@Param		days	query		int	false	"Window in days (default 30)"
//	@Success	200		{object}	response.Envelope
//	@Router		/admin/analytics/sales [get]
func (h *TenantHandler) AdminSales(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	sales, err := h.tenants.PlatformSales(c.Request.Context(), days)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", sales)
}
