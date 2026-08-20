package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/trottling/Telegram-Store/admin_backend/dto"
	"github.com/trottling/Telegram-Store/admin_backend/middleware"
)

// sessionCookieMaxAge — 24ч, совпадает со сроком жизни самого JWT
// (admintoken.GenerateSessionJWT) — cookie не должна пережить токен, который несёт.
const sessionCookieMaxAge = 24 * 60 * 60

// Exchange меняет одноразовый код от бота на сессионный токен. Единственный
// незащищённый /api-роут — см. router.go.
func (h *Handlers) Exchange(c *gin.Context) {
	var req dto.ExchangeCodeRequest
	if !h.decodeJSON(c, &req) {
		return
	}

	token, admin, err := h.adminAuthService.ExchangeLoginCode(c.Request.Context(), req.Code)
	if err != nil {
		// Отдельный Warn: writeError пишет 4xx в Debug, а прод работает на
		// info — то есть подбор кода не оставлял бы в логах ничего вообще.
		// Это единственный роут без авторизации, неудачный обмен здесь —
		// сигнал, а не рядовая клиентская ошибка.
		h.log.WithError(err).WithField("ip", c.ClientIP()).Warn("handlers: login code exchange failed")
		h.writeError(c, err)
		return
	}

	// Та же сессия — в cookie, а не только в теле ответа: SPA всё ещё берёт
	// токен из тела (Authorization-заголовок), а cookie донесёт её браузер
	// сам, в том числе на stats.$DOMAIN_NAME — там Grafana спрятана за
	// forward_auth, и заголовок Authorization там просто некому проставить.
	h.setSessionCookie(c, token)

	h.log.WithField("admin_id", admin.TelegramID).Info("handlers: admin logged in")
	c.JSON(http.StatusOK, dto.TokenResponse{Token: token})
}

// Me возвращает профиль авторизованного админа — фронтенд по 200/401
// проверяет токен, а Caddy forward_auth перед stats.$DOMAIN_NAME так же
// проверяет саму сессию и копирует X-Admin-Username дальше в Grafana
// (auth.proxy, см. Caddyfile/docker-compose.yml).
func (h *Handlers) Me(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	if admin != nil {
		c.Header("X-Admin-Username", adminIdentity(admin.Username, admin.TelegramID))
	}
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
	h.clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}

func (h *Handlers) setSessionCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.SessionCookieName, token, sessionCookieMaxAge, "/", h.cookieDomain, true, true)
}

func (h *Handlers) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.SessionCookieName, "", -1, "/", h.cookieDomain, true, true)
}

// adminIdentity — X-WEBAUTH-USER для Grafana не должен быть пустым: у
// Telegram username опционален, TelegramID есть всегда.
func adminIdentity(username string, telegramID int64) string {
	if username != "" {
		return username
	}
	return strconv.FormatInt(telegramID, 10)
}
