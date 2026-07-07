package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	goredis "github.com/redis/go-redis/v9"

	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
)

// HealthHandler reports liveness of the API and its dependencies.
type HealthHandler struct {
	db     *pgxpool.Pool
	redis  *goredis.Client
	minio  *minio.Client
	bucket string
}

func NewHealthHandler(db *pgxpool.Pool, rdb *goredis.Client, mc *minio.Client, bucket string) *HealthHandler {
	return &HealthHandler{db: db, redis: rdb, minio: mc, bucket: bucket}
}

type dependencyStatus struct {
	Status string `json:"status"` // ok | down
	Error  string `json:"error,omitempty"`
}

// Health godoc
//
//	@Summary		Health check
//	@Description	Reports API liveness and dependency status (PostgreSQL, Redis, MinIO)
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	response.Envelope
//	@Failure		503	{object}	response.Envelope
//	@Router			/health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	checks := gin.H{
		"database": h.checkDatabase(ctx),
		"redis":    h.checkRedis(ctx),
		"storage":  h.checkStorage(ctx),
	}

	healthy := true
	for _, v := range checks {
		if v.(dependencyStatus).Status != "ok" {
			healthy = false
			break
		}
	}

	data := gin.H{
		"status": map[bool]string{true: "healthy", false: "degraded"}[healthy],
		"time":   time.Now().UTC().Format(time.RFC3339),
		"checks": checks,
	}

	if !healthy {
		c.JSON(http.StatusServiceUnavailable, response.Envelope{
			Success: false, Message: "one or more dependencies are down", Data: data,
		})
		return
	}
	response.OK(c, "healthy", data)
}

func (h *HealthHandler) checkDatabase(ctx context.Context) dependencyStatus {
	if err := h.db.Ping(ctx); err != nil {
		return dependencyStatus{Status: "down", Error: err.Error()}
	}
	return dependencyStatus{Status: "ok"}
}

func (h *HealthHandler) checkRedis(ctx context.Context) dependencyStatus {
	if err := h.redis.Ping(ctx).Err(); err != nil {
		return dependencyStatus{Status: "down", Error: err.Error()}
	}
	return dependencyStatus{Status: "ok"}
}

func (h *HealthHandler) checkStorage(ctx context.Context) dependencyStatus {
	exists, err := h.minio.BucketExists(ctx, h.bucket)
	if err != nil {
		return dependencyStatus{Status: "down", Error: err.Error()}
	}
	if !exists {
		return dependencyStatus{Status: "down", Error: "bucket does not exist"}
	}
	return dependencyStatus{Status: "ok"}
}
