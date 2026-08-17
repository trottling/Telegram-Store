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

// Auth проверяет "Authorization: Bearer <token>" через ValidateSession и
// кладёт админа в контекст. Навешан на всю группу /api, а не по хендлерам.
func Auth(adminAuthService service.AdminAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c)
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

func bearerToken(c *gin.Context) string {
	const prefix = "Bearer "
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, prefix))
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
