package v1

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/pkg/export"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

// ReportHandler exposes the reports center with format negotiation.
type ReportHandler struct {
	reports *service.ReportService
}

func NewReportHandler(r *service.ReportService) *ReportHandler { return &ReportHandler{reports: r} }

// ListReportTypes godoc
//
//	@Summary	Available report types
//	@Tags		reports
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/reports [get]
func (h *ReportHandler) ListReportTypes(c *gin.Context) {
	response.OK(c, "", service.ReportTypes())
}

// GetReport godoc
//
//	@Summary	Run a report as json, csv, xlsx, or pdf
//	@Tags		reports
//	@Security	BearerAuth
//	@Produce	json
//	@Param		type	path		string	true	"sales | inventory | employees | attendance | profit | tax | receipts"
//	@Param		from	query		string	false	"YYYY-MM-DD (tenant-local, inclusive)"
//	@Param		to		query		string	false	"YYYY-MM-DD (tenant-local, inclusive)"
//	@Param		format	query		string	false	"json (default) | csv | xlsx | pdf"
//	@Success	200		{object}	response.Envelope
//	@Router		/reports/{type} [get]
func (h *ReportHandler) GetReport(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	reportType := c.Param("type")
	format := c.DefaultQuery("format", "json")

	doc, err := h.reports.Build(c.Request.Context(), tenantID, reportType,
		c.Query("from"), c.Query("to"), format == "pdf")
	if err != nil {
		respondError(c, err)
		return
	}

	if format == "json" {
		response.OK(c, "", doc)
		return
	}

	exporter, ok := export.ForFormat(format)
	if !ok {
		response.Error(c, http.StatusUnprocessableEntity, "format must be json, csv, xlsx, or pdf")
		return
	}

	var buf bytes.Buffer
	if err := exporter.Write(&buf, doc); err != nil {
		respondError(c, err)
		return
	}
	filename := fmt.Sprintf("%s-%s.%s", reportType, time.Now().Format("20060102"), exporter.FileExt())
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, exporter.ContentType(), buf.Bytes())
}
