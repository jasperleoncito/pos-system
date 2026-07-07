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
	"github.com/jasperleoncito/pos-system/backend/internal/realtime"
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
	Hub    *realtime.Hub
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
	productRepo := postgres.NewProductRepo(deps.DB)
	taxRepo := postgres.NewTaxRepo(deps.DB)
	catalogSvc := service.NewCatalogService(
		postgres.NewCategoryRepo(deps.DB), productRepo,
		postgres.NewModifierRepo(deps.DB), taxRepo,
		objectStore, auditSvc, deps.Logger)
	discountRepo := postgres.NewDiscountRepo(deps.DB)
	couponRepo := postgres.NewCouponRepo(deps.DB)
	promoSvc := service.NewPromoService(discountRepo, couponRepo, auditSvc)
	orderSvc := service.NewOrderService(service.OrderServiceDeps{
		Orders: postgres.NewOrderRepo(deps.DB), Drawer: postgres.NewDrawerRepo(deps.DB),
		Products: productRepo, Taxes: taxRepo, Settings: settingsRepo, Tenants: tenantRepo,
		Discounts: discountRepo, Coupons: couponRepo, Hub: deps.Hub,
		Auditor: auditSvc, Logger: deps.Logger,
	})

	// ---- handlers ----
	healthHandler := v1.NewHealthHandler(deps.DB, deps.Redis, deps.MinIO, deps.Config.MinIO.Bucket)
	authHandler := v1.NewAuthHandler(authSvc)
	tenantHandler := v1.NewTenantHandler(tenantSvc)
	catalogHandler := v1.NewCatalogHandler(catalogSvc)
	orderHandler := v1.NewOrderHandler(orderSvc, objectStore)
	promoHandler := v1.NewPromoHandler(promoSvc)
	kitchenHandler := v1.NewKitchenHandler(orderSvc, deps.Hub, tokens)
	inventorySvc := service.NewInventoryService(postgres.NewInventoryRepo(deps.DB), auditSvc, deps.Logger)
	orderSvc.SetInventory(inventorySvc)
	inventoryHandler := v1.NewInventoryHandler(inventorySvc)
	procureRepo := postgres.NewProcureRepo(deps.DB)
	inventorySvc.SetAlertSink(procureRepo)
	procureSvc := service.NewProcureService(procureRepo, inventorySvc, auditSvc, deps.Logger)
	procureHandler := v1.NewProcureHandler(procureSvc)
	employeeSvc := service.NewEmployeeService(postgres.NewEmployeeRepo(deps.DB),
		userRepo, membershipRepo, tenantRepo, objectStore, auditSvc, deps.Logger)
	employeeHandler := v1.NewEmployeeHandler(employeeSvc)

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

	// ---- catalog routes ----
	catalogRead := middleware.RequirePermission(rbac.PermCatalogRead)
	catalogWrite := middleware.RequirePermission(rbac.PermCatalogWrite)
	catalogGroup := api.Group("", middleware.Auth(tokens), middleware.RequireTenant())
	{
		catalogGroup.GET("/categories", catalogRead, catalogHandler.ListCategories)
		catalogGroup.POST("/categories", catalogWrite, catalogHandler.CreateCategory)
		catalogGroup.PUT("/categories/:id", catalogWrite, catalogHandler.UpdateCategory)
		catalogGroup.DELETE("/categories/:id", catalogWrite, catalogHandler.DeleteCategory)

		catalogGroup.GET("/products", catalogRead, catalogHandler.ListProducts)
		catalogGroup.GET("/products/:id", catalogRead, catalogHandler.GetProduct)
		catalogGroup.POST("/products", catalogWrite, catalogHandler.CreateProduct)
		catalogGroup.PUT("/products/:id", catalogWrite, catalogHandler.UpdateProduct)
		catalogGroup.DELETE("/products/:id", catalogWrite, catalogHandler.DeleteProduct)
		catalogGroup.POST("/products/:id/image", catalogWrite, catalogHandler.UploadProductImage)

		catalogGroup.GET("/modifier-groups", catalogRead, catalogHandler.ListModifierGroups)
		catalogGroup.POST("/modifier-groups", catalogWrite, catalogHandler.CreateModifierGroup)
		catalogGroup.PUT("/modifier-groups/:id", catalogWrite, catalogHandler.UpdateModifierGroup)
		catalogGroup.DELETE("/modifier-groups/:id", catalogWrite, catalogHandler.DeleteModifierGroup)

		catalogGroup.GET("/taxes", catalogRead, catalogHandler.ListTaxes)
		catalogGroup.POST("/taxes", catalogWrite, catalogHandler.CreateTax)
		catalogGroup.PUT("/taxes/:id", catalogWrite, catalogHandler.UpdateTax)
		catalogGroup.DELETE("/taxes/:id", catalogWrite, catalogHandler.DeleteTax)
	}

	// ---- order & cash drawer routes ----
	ordersCreate := middleware.RequirePermission(rbac.PermOrdersCreate)
	ordersRead := middleware.RequirePermission(rbac.PermOrdersRead)
	orderGroup := api.Group("", middleware.Auth(tokens), middleware.RequireTenant())
	{
		orderGroup.POST("/orders", ordersCreate, orderHandler.CreateOrder)
		orderGroup.GET("/orders", ordersRead, orderHandler.ListOrders)
		orderGroup.GET("/orders/:id", ordersRead, orderHandler.GetOrder)
		orderGroup.POST("/orders/:id/hold", ordersCreate, orderHandler.SetHold)
		orderGroup.POST("/orders/:id/payments", ordersCreate, orderHandler.Pay)
		orderGroup.GET("/orders/:id/receipt", ordersRead, orderHandler.GetReceipt)

		orderGroup.POST("/cash-drawer/open", ordersCreate, orderHandler.OpenDrawer)
		orderGroup.GET("/cash-drawer/current", ordersCreate, orderHandler.CurrentDrawer)
		orderGroup.POST("/cash-drawer/close", ordersCreate, orderHandler.CloseDrawer)

		// Split bills stay cashier-level; refunds and voids are manager+.
		orderGroup.POST("/orders/:id/splits", ordersCreate, orderHandler.CreateSplits)
		orderGroup.POST("/orders/:id/splits/:splitId/payments", ordersCreate, orderHandler.PaySplit)
		orderGroup.POST("/orders/:id/refunds",
			middleware.RequirePermission(rbac.PermOrdersRefund), orderHandler.Refund)
		orderGroup.POST("/orders/:id/void",
			middleware.RequirePermission(rbac.PermOrdersVoid), orderHandler.Void)

		// Promo management (catalog:write); coupon validation for cashiers.
		orderGroup.GET("/discounts", catalogRead, promoHandler.ListDiscounts)
		orderGroup.POST("/discounts", catalogWrite, promoHandler.CreateDiscount)
		orderGroup.PUT("/discounts/:id", catalogWrite, promoHandler.UpdateDiscount)
		orderGroup.DELETE("/discounts/:id", catalogWrite, promoHandler.DeleteDiscount)
		orderGroup.GET("/coupons", catalogWrite, promoHandler.ListCoupons)
		orderGroup.POST("/coupons", catalogWrite, promoHandler.CreateCoupon)
		orderGroup.PUT("/coupons/:id", catalogWrite, promoHandler.UpdateCoupon)
		orderGroup.DELETE("/coupons/:id", catalogWrite, promoHandler.DeleteCoupon)
		orderGroup.POST("/coupons/validate", ordersCreate, promoHandler.ValidateCoupon)
	}

	// ---- kitchen display routes ----
	kitchenRead := middleware.RequirePermission(rbac.PermKitchenRead)
	kitchenWrite := middleware.RequirePermission(rbac.PermKitchenWrite)
	kitchenGroup := api.Group("/kitchen", middleware.Auth(tokens), middleware.RequireTenant())
	{
		kitchenGroup.GET("/orders", kitchenRead, kitchenHandler.ListOrders)
		kitchenGroup.PATCH("/orders/:id/status", kitchenWrite, kitchenHandler.SetStatus)
		kitchenGroup.PATCH("/orders/:id/items/:itemId/status", kitchenWrite, kitchenHandler.SetItemStatus)
	}
	// SSE stream authenticates via ?token= (EventSource cannot send headers).
	api.GET("/kitchen/stream", kitchenHandler.Stream)
	orderGroup.POST("/orders/:id/priority",
		middleware.RequirePermission(rbac.PermKitchenWrite), kitchenHandler.SetPriority)

	// ---- inventory routes ----
	invRead := middleware.RequirePermission(rbac.PermInventoryRead)
	invWrite := middleware.RequirePermission(rbac.PermInventoryWrite)
	invGroup := api.Group("", middleware.Auth(tokens), middleware.RequireTenant())
	{
		invGroup.GET("/units", invRead, inventoryHandler.ListUnits)
		invGroup.POST("/units", invWrite, inventoryHandler.CreateUnit)
		invGroup.GET("/inventory/items", invRead, inventoryHandler.ListItems)
		invGroup.POST("/inventory/items", invWrite, inventoryHandler.CreateItem)
		invGroup.PUT("/inventory/items/:id", invWrite, inventoryHandler.UpdateItem)
		invGroup.DELETE("/inventory/items/:id", invWrite, inventoryHandler.DeleteItem)
		invGroup.GET("/inventory/movements", invRead, inventoryHandler.ListMovements)
		invGroup.POST("/inventory/movements", invWrite, inventoryHandler.Move)
		invGroup.GET("/products/:id/recipe", invRead, inventoryHandler.GetRecipe)
		invGroup.PUT("/products/:id/recipe", invWrite, inventoryHandler.SaveRecipe)

		invGroup.GET("/suppliers", invRead, procureHandler.ListSuppliers)
		invGroup.POST("/suppliers", invWrite, procureHandler.CreateSupplier)
		invGroup.PUT("/suppliers/:id", invWrite, procureHandler.UpdateSupplier)
		invGroup.DELETE("/suppliers/:id", invWrite, procureHandler.DeleteSupplier)
		invGroup.GET("/purchase-orders", invRead, procureHandler.ListPOs)
		invGroup.GET("/purchase-orders/:id", invRead, procureHandler.GetPO)
		invGroup.POST("/purchase-orders", invWrite, procureHandler.CreatePO)
		invGroup.POST("/purchase-orders/:id/order", invWrite, procureHandler.OrderPO)
		invGroup.POST("/purchase-orders/:id/cancel", invWrite, procureHandler.CancelPO)
		invGroup.POST("/purchase-orders/:id/receive", invWrite, procureHandler.ReceivePO)
		invGroup.GET("/inventory/alerts", invRead, procureHandler.ListAlerts)
		invGroup.POST("/inventory/alerts/:id/ack", invWrite, procureHandler.AckAlert)
	}

	// ---- employee & attendance routes ----
	empRead := middleware.RequirePermission(rbac.PermEmployeesRead)
	empWrite := middleware.RequirePermission(rbac.PermEmployeesWrite)
	empGroup := api.Group("", middleware.Auth(tokens), middleware.RequireTenant())
	{
		empGroup.GET("/employees", empRead, employeeHandler.ListEmployees)
		empGroup.GET("/employees/:id", empRead, employeeHandler.GetEmployee)
		empGroup.POST("/employees", empWrite, employeeHandler.CreateEmployee)
		empGroup.PUT("/employees/:id", empWrite, employeeHandler.UpdateEmployee)
		empGroup.DELETE("/employees/:id", empWrite, employeeHandler.DeleteEmployee)
		empGroup.POST("/employees/:id/photo", empWrite, employeeHandler.UploadEmployeePhoto)
		empGroup.GET("/employees/:id/schedule", empRead, employeeHandler.GetSchedule)
		empGroup.PUT("/employees/:id/schedule", empWrite, employeeHandler.SaveSchedule)

		// Self-service clock — every role has attendance:clock.
		clock := middleware.RequirePermission(rbac.PermAttendanceClock)
		empGroup.GET("/attendance/me", clock, employeeHandler.MyClockStatus)
		empGroup.POST("/attendance/clock-in", clock, employeeHandler.ClockIn)
		empGroup.POST("/attendance/clock-out", clock, employeeHandler.ClockOut)
		empGroup.POST("/attendance/break/start", clock, employeeHandler.StartBreak)
		empGroup.POST("/attendance/break/end", clock, employeeHandler.EndBreak)

		empGroup.GET("/attendance",
			middleware.RequirePermission(rbac.PermAttendanceRead), employeeHandler.ListAttendance)
		empGroup.POST("/attendance/:id/approve",
			middleware.RequirePermission(rbac.PermAttendanceApprove), employeeHandler.ApproveAttendance)
	}

	// ---- super-admin routes ----
	adminGroup := api.Group("/admin", middleware.Auth(tokens), middleware.RequireSuperAdmin())
	{
		adminGroup.GET("/tenants", tenantHandler.AdminListTenants)
		adminGroup.PATCH("/tenants/:id/status", tenantHandler.AdminSetTenantStatus)
	}

	return r
}
