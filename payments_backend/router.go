package paymentsbackend

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trottling/Telegram-Store/payments_backend/handlers"
)

// newRouter — три вебхука мерчантов, без CORS (вызывающая сторона — сервер
// мерчанта, не браузер) и без Auth (каждый вебхук сам проверяет подпись).
func newRouter(h *handlers.Handlers) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.POST("/api/webhooks/crystalpay", h.CrystalPayWebhook)
	r.POST("/api/webhooks/yookassa", h.YooKassaWebhook)
	r.POST("/api/webhooks/tinkoff", h.TinkoffWebhook)

	return r
}
