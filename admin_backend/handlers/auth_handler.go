package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/trottling/Telegram-Store/admin_backend/dto"
	"github.com/trottling/Telegram-Store/admin_backend/middleware"
)

// Exchange меняет одноразовый код от бота на сессионный токен. Единственный
// незащищённый /api-роут — см. router.go.
func (h *Handlers) Exchange(c *gin.Context) {
	var req dto.ExchangeCodeRequest
	if !decodeJSON(c, &req) {
		return
	}

	token, admin, err := h.adminAuthService.ExchangeLoginCode(c.Request.Context(), req.Code)
	if err != nil {
		h.writeError(c, err)
		return
	}

	h.log.WithField("admin_id", admin.TelegramID).Info("handlers: admin logged in")
	c.JSON(http.StatusOK, dto.TokenResponse{Token: token})
}

// Me возвращает профиль авторизованного админа — фронтенд по 200/401 проверяет токен.
func (h *Handlers) Me(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	c.JSON(http.StatusOK, admin)
}

// Logout досрочно отзывает токен сессии.
func (h *Handlers) Logout(c *gin.Context) {
	const prefix = "Bearer "
	if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, prefix) {
		token := strings.TrimSpace(strings.TrimPrefix(auth, prefix))
		if err := h.adminAuthService.Logout(c.Request.Context(), token); err != nil {
			// Не критично — сессия всё равно истечёт по TTL.
			h.log.WithError(err).Warn("handlers: failed to delete session on logout")
		} else {
			h.log.Info("handlers: admin logged out")
		}
	}
	c.Status(http.StatusNoContent)
}
