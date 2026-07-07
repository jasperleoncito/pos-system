package server

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	goredis "github.com/redis/go-redis/v9"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"

	"github.com/jasperleoncito/pos-system/backend/internal/config"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/rbac"
	v1 "github.com/jasperleoncito/pos-system/backend/internal/handler/v1"
	"github.com/jasperleoncito/pos-system/backend/internal/middleware"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/mailer"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/token"
	"github.com/jasperleoncito/pos-system/backend/internal/repository/miniostore"
	"github.com/jasperleoncito/pos-system/backend/internal/repository/postgres"
	redisrepo "github.com/jasperleoncito/pos-system/backend/internal/repository/redis"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
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

	// ---- shared packages ----
	tokens := token.NewManager(deps.Config.JWT.Secret, deps.Config.JWT.AccessTokenTTL, deps.Config.JWT.RefreshTokenTTL)
	mail := mailer.New(deps.Config.SMTP.Host, deps.Config.SMTP.Port, deps.Config.SMTP.User,
		deps.Config.SMTP.Password, deps.Config.SMTP.From)

	// ---- repositories ----
	userRepo := postgres.NewUserRepo(deps.DB)
	sessionRepo := postgres.NewSessionRepo(deps.DB)
	tenantRepo := postgres.NewTenantRepo(deps.DB)
	settingsRepo := postgres.NewTenantSettingsRepo(deps.DB)
	membershipRepo := postgres.NewMembershipRepo(deps.DB)
	auditRepo := postgres.NewAuditRepo(deps.DB)
	otpStore := redisrepo.NewTokenStore(deps.Redis)
	objectStore := miniostore.New(deps.MinIO, deps.Config.MinIO.Bucket, deps.Config.MinIO.PublicBaseURL)

	// ---- services ----
	auditSvc := service.NewAuditService(auditRepo, deps.Logger)
	authSvc := service.NewAuthService(service.AuthServiceDeps{
		Users: userRepo, Sessions: sessionRepo, Tenants: tenantRepo, Settings: settingsRepo,
		Memberships: membershipRepo, Tokens: tokens, OTP: otpStore, Mailer: mail,
		Auditor: auditSvc, Logger: deps.Logger, AppBaseURL: deps.Config.HTTP.CORSOrigins,
	})
	tenantSvc := service.NewTenantService(tenantRepo, settingsRepo, objectStore, auditSvc, deps.Logger)

	// ---- handlers ----
	healthHandler := v1.NewHealthHandler(deps.DB, deps.Redis, deps.MinIO, deps.Config.MinIO.Bucket)
	authHandler := v1.NewAuthHandler(authSvc)
	tenantHandler := v1.NewTenantHandler(tenantSvc)

	api := r.Group("/api/v1")
	api.GET("/health", healthHandler.Health)

	if !deps.Config.App.IsProduction() {
		api.GET("/docs/*any", ginswagger.WrapHandler(swaggerfiles.Handler))
	}

	// ---- auth routes ----
	authLimiter := middleware.RateLimit(deps.Redis, "auth", 20, time.Minute)
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/register", authLimiter, authHandler.Register)
		authGroup.POST("/login", authLimiter, authHandler.Login)
		authGroup.POST("/refresh", authLimiter, authHandler.Refresh)
		authGroup.POST("/logout", authHandler.Logout)
		authGroup.POST("/forgot-password", authLimiter, authHandler.ForgotPassword)
		authGroup.POST("/reset-password", authLimiter, authHandler.ResetPassword)
		authGroup.POST("/verify-email", authLimiter, authHandler.VerifyEmail)

		authed := authGroup.Group("", middleware.Auth(tokens))
		{
			authed.POST("/logout-all", authHandler.LogoutAll)
			authed.GET("/sessions", authHandler.Sessions)
			authed.DELETE("/sessions/:id", authHandler.RevokeSession)
			authed.POST("/resend-verification", authHandler.ResendVerification)
			authed.POST("/switch-tenant", authHandler.SwitchTenant)
		}
	}

	// ---- tenant branding routes ----
	tenantGroup := api.Group("/tenant", middleware.Auth(tokens), middleware.RequireTenant())
	{
		// Readable by every member — branding must theme all roles' UI.
		tenantGroup.GET("/settings", tenantHandler.GetSettings)
		tenantGroup.PUT("/settings",
			middleware.RequirePermission(rbac.PermTenantSettingsWrite), tenantHandler.UpdateSettings)
		tenantGroup.POST("/logo",
			middleware.RequirePermission(rbac.PermTenantSettingsWrite), tenantHandler.UploadLogo)
	}

	// ---- super-admin routes ----
	adminGroup := api.Group("/admin", middleware.Auth(tokens), middleware.RequireSuperAdmin())
	{
		adminGroup.GET("/tenants", tenantHandler.AdminListTenants)
		adminGroup.PATCH("/tenants/:id/status", tenantHandler.AdminSetTenantStatus)
	}

	return r
}
