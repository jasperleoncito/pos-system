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
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/queue"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/token"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/xendit"
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
	r.Use(middleware.SecurityHeaders(deps.Config.App.IsProduction()))
	r.Use(middleware.CORS(deps.Config.HTTP.CORSOrigins))
	// Global per-IP backstop; sensitive auth routes keep their own
	// tighter limiter below.
	r.Use(middleware.RateLimit(deps.Redis, "global", 300, time.Minute))

	r.NoRoute(func(c *gin.Context) {
		response.Error(c, 404, "route not found")
	})

	// ---- shared packages ----
	tokens := token.NewManager(deps.Config.JWT.Secret, deps.Config.JWT.AccessTokenTTL, deps.Config.JWT.RefreshTokenTTL)
	// Transactional mail goes onto the asynq queue; cmd/worker delivers
	// it over SMTP so the API never blocks on a mail server.
	jobQueue := queue.NewClient(deps.Config.Redis.Addr, deps.Config.Redis.Password)

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
		Memberships: membershipRepo, Tokens: tokens, OTP: otpStore, Mailer: jobQueue,
		Auditor: auditSvc, Logger: deps.Logger, AppBaseURL: deps.Config.HTTP.AppURL, AppName: deps.Config.App.Name,
	})
	tenantSvc := service.NewTenantService(tenantRepo, settingsRepo, objectStore, auditSvc, deps.Logger)
	teamSvc := service.NewTeamService(service.TeamServiceDeps{
		Users: userRepo, Memberships: membershipRepo, Tenants: tenantRepo, Settings: settingsRepo,
		OTP: otpStore, Mailer: jobQueue, Auditor: auditSvc, Logger: deps.Logger,
		AppBaseURL: deps.Config.HTTP.AppURL, AppName: deps.Config.App.Name,
	})
	billingSvc := service.NewBillingService(service.BillingServiceDeps{
		Repo: postgres.NewBillingRepo(deps.DB), Tenants: tenantRepo, Users: userRepo,
		Invoices: xendit.New(deps.Config.Xendit.SecretKey), Cache: redisrepo.NewCache(deps.Redis),
		Auditor: auditSvc, Logger: deps.Logger,
		AppBaseURL: deps.Config.HTTP.AppURL, AppName: deps.Config.App.Name,
	})
	teamSvc.SetBilling(billingSvc)
	authSvc.SetBilling(billingSvc)
	// Tenant routes 402 when the subscription lapses; billing + branding
	// + notifications stay open so the pay-modal/blocked-screen work.
	requireActive := middleware.RequireActiveSubscription(billingSvc, deps.Logger)
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
	teamHandler := v1.NewTeamHandler(teamSvc)
	billingHandler := v1.NewBillingHandler(billingSvc, deps.Config.Xendit.WebhookToken)
	catalogHandler := v1.NewCatalogHandler(catalogSvc)
	orderHandler := v1.NewOrderHandler(orderSvc, objectStore)
	promoHandler := v1.NewPromoHandler(promoSvc)
	kitchenHandler := v1.NewKitchenHandler(orderSvc, deps.Hub, tokens)
	inventorySvc := service.NewInventoryService(postgres.NewInventoryRepo(deps.DB), auditSvc, deps.Logger)
	inventorySvc.SetJobs(jobQueue)
	orderSvc.SetInventory(inventorySvc)
	inventoryHandler := v1.NewInventoryHandler(inventorySvc)
	procureRepo := postgres.NewProcureRepo(deps.DB)
	inventorySvc.SetAlertSink(procureRepo)
	procureSvc := service.NewProcureService(procureRepo, inventorySvc, auditSvc, deps.Logger)
	procureHandler := v1.NewProcureHandler(procureSvc)
	employeeSvc := service.NewEmployeeService(postgres.NewEmployeeRepo(deps.DB),
		userRepo, membershipRepo, tenantRepo, objectStore, auditSvc, deps.Logger)
	employeeSvc.SetJobs(jobQueue)
	employeeHandler := v1.NewEmployeeHandler(employeeSvc)
	loyaltySvc := service.NewLoyaltyService(postgres.NewCustomerRepo(deps.DB), auditSvc, deps.Logger)
	orderSvc.SetLoyalty(loyaltySvc)
	customerHandler := v1.NewCustomerHandler(loyaltySvc)
	analyticsSvc := service.NewAnalyticsService(postgres.NewAnalyticsRepo(deps.DB), tenantRepo,
		redisrepo.NewCache(deps.Redis), auditSvc, deps.Logger)
	orderSvc.SetAnalytics(analyticsSvc)
	analyticsHandler := v1.NewAnalyticsHandler(analyticsSvc)
	reportSvc := service.NewReportService(postgres.NewReportRepo(deps.DB), analyticsSvc,
		tenantRepo, settingsRepo, objectStore, deps.Logger)
	reportHandler := v1.NewReportHandler(reportSvc)
	notificationSvc := service.NewNotificationService(postgres.NewNotificationRepo(deps.DB))
	notificationHandler := v1.NewNotificationHandler(notificationSvc)
	auditHandler := v1.NewAuditHandler(auditSvc)

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
		// Readable by every member â€” branding must theme all roles' UI.
		tenantGroup.GET("/settings", tenantHandler.GetSettings)
		tenantGroup.PUT("/settings", requireActive,
			middleware.RequirePermission(rbac.PermTenantSettingsWrite), tenantHandler.UpdateSettings)
		tenantGroup.POST("/logo", requireActive,
			middleware.RequirePermission(rbac.PermTenantSettingsWrite), tenantHandler.UploadLogo)
	}

	// ---- billing routes ----
	// Public: prices for the register page; the Xendit callback is
	// authenticated by its x-callback-token header, not a JWT.
	api.GET("/billing/plans", billingHandler.GetPlans)
	api.POST("/webhooks/xendit", billingHandler.Webhook)
	billingGroup := api.Group("/billing", middleware.Auth(tokens), middleware.RequireTenant())
	{
		// Every member may read status (staff blocked-screen needs it).
		billingGroup.GET("/subscription", billingHandler.GetSubscription)
		billingManage := middleware.RequirePermission(rbac.PermBillingManage)
		billingGroup.POST("/checkout", billingManage, billingHandler.CreateCheckout)
		billingGroup.POST("/voucher/preview", billingManage, billingHandler.PreviewVoucher)
		// Webhook-independent confirmation: the return page polls this.
		billingGroup.POST("/reconcile", billingManage, billingHandler.Reconcile)
		billingGroup.GET("/payments", billingManage, billingHandler.ListPayments)
	}

	// ---- team management routes (owner via users:manage) ----
	teamGroup := api.Group("/team", middleware.Auth(tokens), middleware.RequireTenant(),
		requireActive, middleware.RequirePermission(rbac.PermUsersManage))
	{
		teamGroup.GET("", teamHandler.ListMembers)
		teamGroup.POST("", teamHandler.InviteMember)
		teamGroup.PATCH("/:userId/role", teamHandler.UpdateMemberRole)
		teamGroup.DELETE("/:userId", teamHandler.RemoveMember)
		teamGroup.POST("/:userId/resend-invite", teamHandler.ResendInvite)
	}

	// ---- catalog routes ----
	catalogRead := middleware.RequirePermission(rbac.PermCatalogRead)
	catalogWrite := middleware.RequirePermission(rbac.PermCatalogWrite)
	catalogGroup := api.Group("", middleware.Auth(tokens), middleware.RequireTenant(), requireActive)
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
	orderGroup := api.Group("", middleware.Auth(tokens), middleware.RequireTenant(), requireActive)
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
	kitchenGroup := api.Group("/kitchen", middleware.Auth(tokens), middleware.RequireTenant(), requireActive)
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
	invGroup := api.Group("", middleware.Auth(tokens), middleware.RequireTenant(), requireActive)
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
	empGroup := api.Group("", middleware.Auth(tokens), middleware.RequireTenant(), requireActive)
	{
		empGroup.GET("/employees", empRead, employeeHandler.ListEmployees)
		empGroup.GET("/employees/:id", empRead, employeeHandler.GetEmployee)
		empGroup.POST("/employees", empWrite, employeeHandler.CreateEmployee)
		empGroup.PUT("/employees/:id", empWrite, employeeHandler.UpdateEmployee)
		empGroup.DELETE("/employees/:id", empWrite, employeeHandler.DeleteEmployee)
		empGroup.POST("/employees/:id/photo", empWrite, employeeHandler.UploadEmployeePhoto)
		empGroup.GET("/employees/:id/schedule", empRead, employeeHandler.GetSchedule)
		empGroup.PUT("/employees/:id/schedule", empWrite, employeeHandler.SaveSchedule)

		// Self-service clock â€” every role has attendance:clock.
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

	// ---- customer & loyalty routes ----
	custRead := middleware.RequirePermission(rbac.PermCustomersRead)
	custWrite := middleware.RequirePermission(rbac.PermCustomersWrite)
	custGroup := api.Group("", middleware.Auth(tokens), middleware.RequireTenant(), requireActive)
	{
		custGroup.GET("/customers", custRead, customerHandler.ListCustomers)
		custGroup.GET("/customers/:id", custRead, customerHandler.GetCustomer)
		custGroup.POST("/customers", custWrite, customerHandler.CreateCustomer)
		custGroup.PUT("/customers/:id", custWrite, customerHandler.UpdateCustomer)
		custGroup.DELETE("/customers/:id", custWrite, customerHandler.DeleteCustomer)
		custGroup.GET("/customers/:id/loyalty", custRead, customerHandler.ListLoyaltyTransactions)

		// Program configuration is manager+ (catalog:write), like promos.
		custGroup.GET("/loyalty/settings", custRead, customerHandler.GetLoyaltySettings)
		custGroup.PUT("/loyalty/settings", catalogWrite, customerHandler.UpdateLoyaltySettings)
	}

	// ---- analytics & expenses routes ----
	analyticsRead := middleware.RequirePermission(rbac.PermAnalyticsRead)
	analyticsGroup := api.Group("", middleware.Auth(tokens), middleware.RequireTenant(), requireActive, analyticsRead)
	{
		analyticsGroup.GET("/analytics/overview", analyticsHandler.Overview)
		analyticsGroup.GET("/analytics/dashboard", analyticsHandler.Dashboard)
		analyticsGroup.GET("/expenses", analyticsHandler.ListExpenses)
		analyticsGroup.POST("/expenses", analyticsHandler.CreateExpense)
		analyticsGroup.PUT("/expenses/:id", analyticsHandler.UpdateExpense)
		analyticsGroup.DELETE("/expenses/:id", analyticsHandler.DeleteExpense)
	}

	// ---- reports routes ----
	reportsGroup := api.Group("/reports", middleware.Auth(tokens), middleware.RequireTenant(),
		requireActive, middleware.RequirePermission(rbac.PermReportsRead))
	{
		reportsGroup.GET("", reportHandler.ListReportTypes)
		reportsGroup.GET("/:type", reportHandler.GetReport)
	}

	// ---- notification routes (every member has a bell) ----
	notifGroup := api.Group("/notifications", middleware.Auth(tokens), middleware.RequireTenant())
	{
		notifGroup.GET("", notificationHandler.Feed)
		notifGroup.POST("/read-all", notificationHandler.MarkAllRead)
		notifGroup.POST("/:id/read", notificationHandler.MarkRead)
		notifGroup.GET("/preferences", notificationHandler.GetPrefs)
		notifGroup.PUT("/preferences", notificationHandler.UpdatePrefs)
	}

	// ---- audit trail (owner only via audit:read) ----
	api.GET("/audit-logs", middleware.Auth(tokens), middleware.RequireTenant(),
		requireActive, middleware.RequirePermission(rbac.PermAuditRead), auditHandler.List)

	// ---- super-admin routes ----
	adminGroup := api.Group("/admin", middleware.Auth(tokens), middleware.RequireSuperAdmin())
	{
		adminGroup.GET("/tenants", tenantHandler.AdminListTenants)
		adminGroup.POST("/tenants", teamHandler.AdminCreateTenant)
		adminGroup.GET("/subscriptions", billingHandler.AdminListSubscriptions)
		adminGroup.POST("/subscriptions/:tenantId/mark-paid", billingHandler.AdminMarkPaid)
		adminGroup.POST("/subscriptions/:tenantId/grant", billingHandler.AdminGrantMonths)
		adminGroup.PATCH("/subscriptions/:tenantId/status", billingHandler.AdminSetSubscriptionStatus)
		adminGroup.GET("/owners", billingHandler.AdminListOwners)
		adminGroup.GET("/billing/stats", billingHandler.AdminBillingStats)
		adminGroup.GET("/billing/settings", billingHandler.AdminGetPrices)
		adminGroup.PUT("/billing/settings", billingHandler.AdminUpdatePrices)
		adminGroup.GET("/vouchers", billingHandler.AdminListVouchers)
		adminGroup.POST("/vouchers", billingHandler.AdminCreateVoucher)
		adminGroup.PATCH("/vouchers/:id/active", billingHandler.AdminSetVoucherActive)
		adminGroup.DELETE("/vouchers/:id", billingHandler.AdminDeleteVoucher)
		adminGroup.PATCH("/tenants/:id/status", tenantHandler.AdminSetTenantStatus)
		adminGroup.PATCH("/tenants/:id/plan", tenantHandler.AdminSetTenantPlan)
		adminGroup.GET("/stats", tenantHandler.AdminStats)
		adminGroup.GET("/analytics/sales", tenantHandler.AdminSales)
	}

	return r
}
