package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"

	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
)

// RateLimit applies a fixed-window per-IP limit backed by Redis.
// Fails open if Redis is unavailable so an outage can't lock users out.
func RateLimit(client *goredis.Client, name string, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("ratelimit:%s:%s:%d", name, c.ClientIP(), time.Now().Unix()/int64(window.Seconds()))

		count, err := client.Incr(c.Request.Context(), key).Result()
		if err != nil {
			c.Next()
			return
		}
		if count == 1 {
			client.Expire(c.Request.Context(), key, window)
		}
		if count > int64(limit) {
			response.Error(c, http.StatusTooManyRequests, "too many requests — please slow down")
			c.Abort()
			return
		}
		c.Next()
	}
}
