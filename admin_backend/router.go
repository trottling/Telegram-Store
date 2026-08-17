package adminbackend

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/admin_backend/handlers"
	"github.com/trottling/Telegram-Store/admin_backend/middleware"
	"github.com/trottling/Telegram-Store/internal/domain/service"
)

// newRouter собирает таблицу маршрутов. Без Auth — только /api/auth/exchange
// (там ещё нет сессии). Вебхуки мерчантов сюда не входят — они принимаются
// отдельным бинарником, payments_backend (см. его router.go).
func newRouter(h *handlers.Handlers, adminAuthService service.AdminAuthService, corsOrigin string, log *logrus.Logger) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(corsOrigin, log))

	r.POST("/api/auth/exchange", h.Exchange)

	api := r.Group("/api")
	api.Use(middleware.Auth(adminAuthService))
	{
		api.GET("/auth/me", h.Me)
		api.POST("/auth/logout", h.Logout)

		api.GET("/categories", h.ListCategories)
		api.POST("/categories", h.CreateCategory)
		api.GET("/categories/:id", h.GetCategory)
		api.PUT("/categories/:id", h.UpdateCategory)
		api.DELETE("/categories/:id", h.DeleteCategory)

		api.GET("/products", h.ListProducts)
		api.POST("/products", h.CreateProduct)
		api.GET("/products/:id", h.GetProduct)
		api.PUT("/products/:id", h.UpdateProduct)
		api.DELETE("/products/:id", h.DeleteProduct)
		api.POST("/products/:id/items", h.AddProductItems)

		api.GET("/users", h.ListUsers)
		api.GET("/users/:telegram_id", h.GetUser)
		api.POST("/users/:telegram_id/ban", h.BanUser)
		api.POST("/users/:telegram_id/unban", h.UnbanUser)
		api.POST("/users/:telegram_id/balance", h.AdjustBalance)
		api.POST("/users/:telegram_id/promote", h.PromoteUser)
		api.POST("/users/:telegram_id/demote", h.DemoteUser)
		api.GET("/users/:telegram_id/referrals", h.ListUserReferrals)
		api.POST("/users/:telegram_id/referrals/enable", h.EnableUserReferrals)
		api.POST("/users/:telegram_id/referrals/disable", h.DisableUserReferrals)

		api.GET("/purchases", h.ListPurchases)
		api.GET("/purchases/:id", h.GetPurchase)

		api.GET("/stats/dashboard", h.Dashboard)

		api.GET("/admin-logs", h.ListAdminLogs)

		api.GET("/settings", h.GetSettings)
		api.PUT("/settings", h.UpdateSettings)

		api.GET("/replenishments", h.ListReplenishments)
	}

	return r
}
