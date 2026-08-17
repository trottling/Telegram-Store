package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/admin_backend/errors"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/service"
)

// RateLimitExchange ограничивает попытки обмена кода входа.
//
// Навешивается только на /api/auth/exchange — единственный роут без
// авторизации, и код там всего 6-значный. Неудачная попытка ничего не
// расходует (GETDEL удаляет ключ только при попадании), так что без счётчика
// код можно подбирать неограниченно, а приз — суточная админ-сессия.
//
// Ключ — IP, без глобального лимита: общий счётчик означал бы, что один
// атакующий закрывает вход всем админам. Полагаться на IP можно только при
// настроенных доверенных прокси, см. SetTrustedProxies в router.go.
func RateLimitExchange(adminAuthService service.AdminAuthService, log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		allowed, err := adminAuthService.AllowExchangeAttempt(c.Request.Context(), ip)
		if err != nil || !allowed {
			// Ошибка Redis — тоже отказ: сам обмен кода всё равно требует
			// Redis, разрешать проход было бы бессмысленно.
			log.WithError(err).WithFields(logrus.Fields{"ip": ip, "allowed": allowed}).
				Warn("middleware: login code exchange rate limited")
			c.JSON(errors.DomainErrorToResponse(domainerrors.ErrTooManyAttempts))
			c.Abort()
			return
		}

		c.Next()
	}
}
