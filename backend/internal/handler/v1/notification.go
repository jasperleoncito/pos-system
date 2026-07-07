package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/notification"
	"github.com/jasperleoncito/pos-system/backend/internal/dto"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

// NotificationHandler serves the bell feed and email preferences.
type NotificationHandler struct {
	notifications *service.NotificationService
}

func NewNotificationHandler(n *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifications: n}
}

// Feed godoc
//
//	@Summary	Recent notifications with the unread count
//	@Tags		notifications
//	@Security	BearerAuth
//	@Produce	json
//	@Param		limit	query		int	false	"Max items (default 30)"
//	@Success	200		{object}	response.Envelope
//	@Router		/notifications [get]
func (h *NotificationHandler) Feed(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	feed, err := h.notifications.Feed(c.Request.Context(), tenantID, userID, limit)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", feed)
}

// MarkRead godoc
//
//	@Summary	Mark one notification read
//	@Tags		notifications
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Notification ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/notifications/{id}/read [post]
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	if err := h.notifications.MarkRead(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "notification read", nil)
}

// MarkAllRead godoc
//
//	@Summary	Mark every notification read
//	@Tags		notifications
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/notifications/read-all [post]
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	if err := h.notifications.MarkAllRead(c.Request.Context(), tenantID, userID); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "all notifications read", nil)
}

// GetPrefs godoc
//
//	@Summary	Email notification preferences
//	@Tags		notifications
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/notifications/preferences [get]
func (h *NotificationHandler) GetPrefs(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	prefs, err := h.notifications.GetPrefs(c.Request.Context(), tenantID, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", prefs)
}

// UpdatePrefs godoc
//
//	@Summary	Update email notification preferences
//	@Tags		notifications
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.NotificationPrefsRequest	true	"Preferences"
//	@Success	200		{object}	response.Envelope
//	@Router		/notifications/preferences [put]
func (h *NotificationHandler) UpdatePrefs(c *gin.Context) {
	var req dto.NotificationPrefsRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	prefs, err := h.notifications.SavePrefs(c.Request.Context(), tenantID, userID, &notification.Prefs{
		EmailLowStock:     boolOrDefault(req.EmailLowStock, true),
		EmailAttendance:   boolOrDefault(req.EmailAttendance, true),
		EmailDailySummary: boolOrDefault(req.EmailDailySummary, true),
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "preferences saved", prefs)
}
