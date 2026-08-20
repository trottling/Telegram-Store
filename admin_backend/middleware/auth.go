package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/trottling/Telegram-Store/admin_backend/errors"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/service"
)

// ctxKeyAdmin — ключ gin.Context для авторизованного админа.
const ctxKeyAdmin = "admin"

// SessionCookieName — та же сессия, что и Authorization-заголовок SPA,
// продублированная в cookie (см. handlers.Exchange), чтобы её донёс браузер
// на другой поддомен — так Caddy forward_auth перед stats.$DOMAIN_NAME
// (Grafana) может спросить /api/auth/me без своей формы логина.
const SessionCookieName = "session"

// Auth проверяет сессионный токен через ValidateSession и кладёт админа в
// контекст. Навешан на всю группу /api, а не по хендлерам. Источник токена —
// сначала "Authorization: Bearer" (SPA), при отсутствии — cookie
// SessionCookieName (forward_auth/браузерная навигация на stats.$DOMAIN_NAME,
// где заголовок никто не проставляет).
func Auth(adminAuthService service.AdminAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := sessionToken(c)
		if token == "" {
			c.JSON(errors.DomainErrorToResponse(domainerrors.ErrInvalidToken))
			c.Abort()
			return
		}

		admin, err := adminAuthService.ValidateSession(c.Request.Context(), token)
		if err != nil {
			c.JSON(errors.DomainErrorToResponse(domainerrors.ErrInvalidToken))
			c.Abort()
			return
		}

		c.Set(ctxKeyAdmin, admin)
		c.Next()
	}
}

func sessionToken(c *gin.Context) string {
	const prefix = "Bearer "
	if h := c.GetHeader("Authorization"); strings.HasPrefix(h, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(h, prefix))
	}
	if cookie, err := c.Cookie(SessionCookieName); err == nil {
		return cookie
	}
	return ""
}

// AdminFromContext возвращает админа, положенного туда Auth.
func AdminFromContext(c *gin.Context) (*models.User, bool) {
	v, ok := c.Get(ctxKeyAdmin)
	if !ok {
		return nil, false
	}
	admin, ok := v.(*models.User)
	return admin, ok
}
