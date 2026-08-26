package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	adminmetrics "github.com/trottling/Telegram-Store/internal/metrics/admin"
)

// Metrics считает HTTP-запросы и их длительность. Регистрируется до
// gin.Recovery() (снаружи неё) — иначе defer здесь отработал бы раньше, чем
// Recovery успеет проставить 500 после паники, и метрика ушла бы со старым
// статусом. route — c.FullPath(), шаблон маршрута ("/api/products/:id"), не
// сырой путь — иначе кардинальность росла бы с каждым уникальным ID.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		defer func() {
			route := c.FullPath()
			if route == "" {
				route = "unmatched"
			}
			method := c.Request.Method
			status := strconv.Itoa(c.Writer.Status())

			adminmetrics.HTTPRequestsTotal.WithLabelValues(route, method, status).Inc()
			adminmetrics.HTTPRequestDurationSeconds.WithLabelValues(route, method).Observe(time.Since(start).Seconds())
		}()

		c.Next()
	}
}
