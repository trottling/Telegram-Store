package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// rejectedOriginLogInterval — как часто максимум пишем про отвергнутый origin.
// Заголовок Origin задаёт клиент, поэтому без ограничения любой желающий
// наполняет лог строками произвольного содержания и топит в них всё остальное.
// Неверно настроенный фронтенд бьётся в CORS постоянно, так что для диагностики
// хватает и одной записи в интервал.
const rejectedOriginLogInterval = 10 * time.Second

// CORS разрешает список origin'ов из ADMIN_PANEL_CORS_ORIGIN (через запятую).
func CORS(allowedOrigins string, log *zap.SugaredLogger) gin.HandlerFunc {
	allowed := make(map[string]struct{})
	for o := range strings.SplitSeq(allowedOrigins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			allowed[o] = struct{}{}
		}
	}
	log.Infow("admin_backend: CORS configured", "allowed_origins", allowedOrigins)

	var (
		mu            sync.Mutex
		lastRejectLog time.Time
		suppressed    int
	)

	return cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			if _, ok := allowed[origin]; ok {
				return true
			}

			// Несовпадающий origin — частая причина проблем с CORS, поэтому
			// Warn, но не чаще интервала; сколько записей пропущено, пишем
			// рядом, чтобы всплеск был виден.
			mu.Lock()
			defer mu.Unlock()
			if time.Since(lastRejectLog) < rejectedOriginLogInterval {
				suppressed++
				return false
			}
			log.Warnw("admin_backend: rejected request from disallowed origin",
				"origin", origin, "allowed_origins", allowedOrigins, "suppressed_since_last", suppressed)
			lastRejectLog = time.Now()
			suppressed = 0
			return false
		},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
}
