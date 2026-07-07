package server

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	goredis "github.com/redis/go-redis/v9"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"

	"github.com/jasperleoncito/pos-system/backend/internal/config"
	v1 "github.com/jasperleoncito/pos-system/backend/internal/handler/v1"
	"github.com/jasperleoncito/pos-system/backend/internal/middleware"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
)

// Dependencies groups the shared infrastructure clients used to wire
// handlers, services, and repositories.
type Dependencies struct {
	Config *config.Config
	Logger *slog.Logger
	DB     *pgxpool.Pool
	Redis  *goredis.Client
	MinIO  *minio.Client
}

// NewRouter builds the Gin engine with all middleware and routes registered.
func NewRouter(deps Dependencies) *gin.Engine {
	if deps.Config.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(middleware.Recovery(deps.Logger))
	r.Use(middleware.CORS(deps.Config.HTTP.CORSOrigins))

	r.NoRoute(func(c *gin.Context) {
		response.Error(c, 404, "route not found")
	})

	api := r.Group("/api/v1")

	healthHandler := v1.NewHealthHandler(deps.DB, deps.Redis, deps.MinIO, deps.Config.MinIO.Bucket)
	api.GET("/health", healthHandler.Health)

	if !deps.Config.App.IsProduction() {
		api.GET("/docs/*any", ginswagger.WrapHandler(swaggerfiles.Handler))
	}

	return r
}
