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

// Exchange меняет initData от Telegram на сессионный токен. Единственный
// незащищённый /api-роут — см. router.go.
func (h *Handlers) Exchange(c *gin.Context) {
	var req dto.ExchangeRequest
	if !h.decodeJSON(c, &req) {
		return
	}

	token, admin, err := h.adminAuthService.ExchangeInitData(c.Request.Context(), req.InitData)
	if err != nil {
		// Отдельный Warn: writeError пишет 4xx в Debug, а прод работает на
		// info — то есть подбор кода не оставлял бы в логах ничего вообще.
		// Это единственный роут без авторизации, неудачный обмен здесь —
		// сигнал, а не рядовая клиентская ошибка.
		h.log.Warnw("handlers: login code exchange failed", "error", err, "ip", c.ClientIP())
		h.writeError(c, err)
		return
	}

	// Та же сессия — в cookie, а не только в теле ответа: SPA всё ещё берёт
	// токен из тела (Authorization-заголовок), а cookie донесёт её браузер
	// сам, в том числе на /stats — там Grafana спрятана за forward_auth, и
	// заголовок Authorization там просто некому проставить.
	h.setSessionCookie(c, token)

	h.log.Infow("handlers: admin logged in", "admin_id", admin.TelegramID)
	c.JSON(http.StatusOK, dto.TokenResponse{Token: token})
}

// Me возвращает профиль авторизованного админа — фронтенд по 200/401
// проверяет токен, а Caddy forward_auth перед /stats так же проверяет саму
// сессию и копирует X-Admin-Username дальше в Grafana (auth.proxy, см.
// Caddyfile/docker-compose.yml).
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
			h.log.Warnw("handlers: failed to delete session on logout", "error", err)
		} else {
			h.log.Info("handlers: admin logged out")
		}
	}
	h.clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}

// Domain у cookie сознательно не задаём (""): host-only cookie с Path="/"
// и так покрывает все пути одного хоста — панель, /api, /stats — с тех пор
// как всё это один origin вместо четырёх поддоменов (см. Caddyfile).
func (h *Handlers) setSessionCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.SessionCookieName, token, sessionCookieMaxAge, "/", "", true, true)
}

func (h *Handlers) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.SessionCookieName, "", -1, "/", "", true, true)
}

// adminIdentity — X-WEBAUTH-USER для Grafana не должен быть пустым: у
// Telegram username опционален, TelegramID есть всегда.
func adminIdentity(username string, telegramID int64) string {
	if username != "" {
		return username
	}
	return strconv.FormatInt(telegramID, 10)
}
