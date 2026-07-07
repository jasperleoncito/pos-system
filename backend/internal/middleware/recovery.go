package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
)

// Recovery converts panics into a clean 500 envelope and logs the cause.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					"panic", r,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
				)
				response.Error(c, http.StatusInternalServerError, "something went wrong")
				c.Abort()
			}
		}()
		c.Next()
	}
}
