package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CORS разрешает список origin'ов из ADMIN_PANEL_CORS_ORIGIN (через запятую).
func CORS(allowedOrigins string, log *logrus.Logger) gin.HandlerFunc {
	allowed := make(map[string]struct{})
	for o := range strings.SplitSeq(allowedOrigins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			allowed[o] = struct{}{}
		}
	}
	log.WithField("allowed_origins", allowedOrigins).Info("backend: CORS configured")

	return cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			if _, ok := allowed[origin]; ok {
				return true
			}
			// Логируем на Warn — несовпадающий origin частая причина проблем с CORS.
			log.WithFields(logrus.Fields{"origin": origin, "allowed_origins": allowedOrigins}).
				Warn("backend: rejected request from disallowed origin")
			return false
		},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
}
