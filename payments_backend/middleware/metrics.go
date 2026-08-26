package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	paymentsmetrics "github.com/trottling/Telegram-Store/internal/metrics/payments"
)

// Metrics считает HTTP-запросы и их длительность — тот же принцип, что
// admin_backend/middleware.Metrics: defer снаружи gin.Recovery(), чтобы
// наблюдать итоговый статус (в том числе 500 после восстановленной паники),
// route — шаблон маршрута (c.FullPath()), не сырой путь.
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

			paymentsmetrics.HTTPRequestsTotal.WithLabelValues(route, method, status).Inc()
			paymentsmetrics.HTTPRequestDurationSeconds.WithLabelValues(route, method).Observe(time.Since(start).Seconds())
		}()

		c.Next()
	}
}
