package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

// AuditHandler exposes the tenant audit trail (owner only).
type AuditHandler struct {
	audits *service.AuditService
}

func NewAuditHandler(a *service.AuditService) *AuditHandler { return &AuditHandler{audits: a} }

// List godoc
//
//	@Summary	Tenant audit trail
//	@Tags		audit
//	@Security	BearerAuth
//	@Produce	json
//	@Param		page	query		int	false	"Page (1-based)"
//	@Param		limit	query		int	false	"Page size"
//	@Success	200		{object}	response.Envelope
//	@Router		/audit-logs [get]
func (h *AuditHandler) List(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	logs, total, err := h.audits.List(c.Request.Context(), tenantID, limit, (page-1)*limit)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Paginated(c, "", logs, response.Meta{Total: total, Page: page, Limit: limit})
}
