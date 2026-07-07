package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/rbac"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/token"
)

// Context keys set by the auth middleware chain.
const (
	CtxUserID       = "auth.user_id"
	CtxTenantID     = "auth.tenant_id"
	CtxRole         = "auth.role"
	CtxSessionID    = "auth.session_id"
	CtxIsSuperAdmin = "auth.is_super_admin"
)

// Auth validates the bearer token and stores identity in the context.
func Auth(tokens *token.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		raw, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || raw == "" {
			response.Error(c, http.StatusUnauthorized, "authentication required")
			c.Abort()
			return
		}

		claims, err := tokens.ParseAccessToken(raw)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxTenantID, claims.TenantID)
		c.Set(CtxRole, claims.Role)
		c.Set(CtxSessionID, claims.SessionID)
		c.Set(CtxIsSuperAdmin, claims.IsSuperAdmin)
		c.Next()
	}
}

// RequireTenant ensures the token is scoped to a tenant. Tenant identity
// comes exclusively from the JWT — never from client-supplied parameters.
func RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString(CtxTenantID) == "" {
			response.Error(c, http.StatusForbidden, "no active business selected")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequirePermission enforces the RBAC matrix for the active role.
// Super admins bypass tenant permission checks.
func RequirePermission(perm rbac.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool(CtxIsSuperAdmin) {
			c.Next()
			return
		}
		role := rbac.Role(c.GetString(CtxRole))
		if !rbac.Can(role, perm) {
			response.Error(c, http.StatusForbidden, "you do not have permission to do that")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireSuperAdmin restricts a route to platform super admins.
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !c.GetBool(CtxIsSuperAdmin) {
			response.Error(c, http.StatusForbidden, "super admin access required")
			c.Abort()
			return
		}
		c.Next()
	}
}
