package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
)

// SubscriptionChecker reports whether a tenant's subscription allows
// using the app (implemented by BillingService with a short Redis cache).
type SubscriptionChecker interface {
	IsActive(ctx context.Context, tenantID string) (bool, error)
}

// RequireActiveSubscription blocks tenant routes with 402 when the
// subscription is pending or inactive. Super admins bypass. Fails OPEN
// on checker errors — billing infrastructure problems must never brick
// every tenant.
func RequireActiveSubscription(checker SubscriptionChecker, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool(CtxIsSuperAdmin) {
			c.Next()
			return
		}
		tenantID := c.GetString(CtxTenantID)
		if tenantID == "" {
			c.Next() // RequireTenant handles missing tenants
			return
		}
		active, err := checker.IsActive(c.Request.Context(), tenantID)
		if err != nil {
			logger.Warn("subscription check failed — allowing request", "tenant", tenantID, "error", err)
			c.Next()
			return
		}
		if !active {
			response.Error(c, http.StatusPaymentRequired, "subscription is not active")
			c.Abort()
			return
		}
		c.Next()
	}
}
