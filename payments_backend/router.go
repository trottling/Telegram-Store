package paymentsbackend

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trottling/Telegram-Store/payments_backend/handlers"
	"github.com/trottling/Telegram-Store/payments_backend/middleware"
)

// newRouter — три вебхука мерчантов, без CORS (вызывающая сторона — сервер
// мерчанта, не браузер) и без Auth (каждый вебхук сам проверяет подпись).
func newRouter(h *handlers.Handlers) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// Recovery снаружи Detach, чтобы накрывать и панику после отцепления ctx.
	r.Use(gin.Recovery())
	r.Use(middleware.Detach())

	r.POST("/api/webhooks/crystalpay", h.CrystalPayWebhook)
	r.POST("/api/webhooks/yookassa", h.YooKassaWebhook)
	r.POST("/api/webhooks/tinkoff", h.TinkoffWebhook)

	return r
}
