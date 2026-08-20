package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trottling/Telegram-Store/admin_backend/dto"
	"github.com/trottling/Telegram-Store/admin_backend/middleware"
	"github.com/trottling/Telegram-Store/internal/domain/models"
)

func (h *Handlers) ListUsers(c *gin.Context) {
	offset, limit := parsePagination(c)
	users, err := h.userService.ListAdmin(c.Request.Context(), offset, limit)
	if err != nil {
		h.writeError(c, err)
		return
	}
	total, err := h.userService.CountAdmin(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Paginated[models.User]{Items: users, Total: total, Offset: offset, Limit: limit})
}

func (h *Handlers) GetUser(c *gin.Context) {
	id, ok := parseIDParam(c, "telegram_id")
	if !ok {
		return
	}
	user, err := h.userService.GetProfile(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *Handlers) BanUser(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	id, ok := parseIDParam(c, "telegram_id")
	if !ok {
		return
	}
	if err := h.adminService.BanUser(c.Request.Context(), admin.TelegramID, id); err != nil {
		h.writeError(c, err)
		return
	}
	h.log.Infow("handlers: user banned", "admin_id", admin.TelegramID, "target_id", id)
	c.Status(http.StatusNoContent)
}

func (h *Handlers) UnbanUser(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	id, ok := parseIDParam(c, "telegram_id")
	if !ok {
		return
	}
	if err := h.adminService.UnbanUser(c.Request.Context(), admin.TelegramID, id); err != nil {
		h.writeError(c, err)
		return
	}
	h.log.Infow("handlers: user unbanned", "admin_id", admin.TelegramID, "target_id", id)
	c.Status(http.StatusNoContent)
}

func (h *Handlers) AdjustBalance(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	id, ok := parseIDParam(c, "telegram_id")
	if !ok {
		return
	}
	var req dto.UpdateBalanceRequest
	if !h.decodeJSON(c, &req) {
		return
	}
	if err := h.adminService.AddBalance(c.Request.Context(), admin.TelegramID, id, req.Amount); err != nil {
		h.writeError(c, err)
		return
	}
	h.log.Infow("handlers: balance adjusted", "admin_id", admin.TelegramID, "target_id", id, "amount", req.Amount)
	c.Status(http.StatusNoContent)
}

// PromoteUser выдаёт права админа — сам эндпоинт не возвращает credential.
func (h *Handlers) PromoteUser(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	id, ok := parseIDParam(c, "telegram_id")
	if !ok {
		return
	}
	if err := h.adminService.MakeAdmin(c.Request.Context(), admin.TelegramID, id); err != nil {
		h.writeError(c, err)
		return
	}
	h.log.Infow("handlers: user promoted to admin", "admin_id", admin.TelegramID, "target_id", id)
	c.Status(http.StatusNoContent)
}

func (h *Handlers) DemoteUser(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	id, ok := parseIDParam(c, "telegram_id")
	if !ok {
		return
	}
	if err := h.adminService.RevokeAdmin(c.Request.Context(), admin.TelegramID, id); err != nil {
		h.writeError(c, err)
		return
	}
	h.log.Infow("handlers: admin revoked", "admin_id", admin.TelegramID, "target_id", id)
	c.Status(http.StatusNoContent)
}

// ListUserReferrals — кого пригласил telegram_id, постранично.
func (h *Handlers) ListUserReferrals(c *gin.Context) {
	id, ok := parseIDParam(c, "telegram_id")
	if !ok {
		return
	}
	offset, limit := parsePagination(c)

	referrals, err := h.userService.ListReferrals(c.Request.Context(), id, offset, limit)
	if err != nil {
		h.writeError(c, err)
		return
	}
	total, err := h.userService.CountReferrals(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Paginated[models.User]{Items: referrals, Total: total, Offset: offset, Limit: limit})
}

func (h *Handlers) EnableUserReferrals(c *gin.Context) {
	h.setUserReferrals(c, true)
}

func (h *Handlers) DisableUserReferrals(c *gin.Context) {
	h.setUserReferrals(c, false)
}

func (h *Handlers) setUserReferrals(c *gin.Context, enabled bool) {
	admin, _ := middleware.AdminFromContext(c)
	id, ok := parseIDParam(c, "telegram_id")
	if !ok {
		return
	}
	if err := h.adminService.SetReferralsEnabled(c.Request.Context(), admin.TelegramID, id, enabled); err != nil {
		h.writeError(c, err)
		return
	}
	h.log.Infow("handlers: user referrals toggled", "admin_id", admin.TelegramID, "target_id", id, "enabled", enabled)
	c.Status(http.StatusNoContent)
}
