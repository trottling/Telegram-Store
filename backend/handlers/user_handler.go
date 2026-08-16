package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/backend/dto"
	"github.com/trottling/Telegram-Store/backend/middleware"
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
	h.log.WithFields(logrus.Fields{"admin_id": admin.TelegramID, "target_id": id}).Info("handlers: user banned")
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
	h.log.WithFields(logrus.Fields{"admin_id": admin.TelegramID, "target_id": id}).Info("handlers: user unbanned")
	c.Status(http.StatusNoContent)
}

func (h *Handlers) AdjustBalance(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	id, ok := parseIDParam(c, "telegram_id")
	if !ok {
		return
	}
	var req dto.UpdateBalanceRequest
	if !decodeJSON(c, &req) {
		return
	}
	if err := h.adminService.AddBalance(c.Request.Context(), admin.TelegramID, id, req.Amount); err != nil {
		h.writeError(c, err)
		return
	}
	h.log.WithFields(logrus.Fields{"admin_id": admin.TelegramID, "target_id": id, "amount": req.Amount}).Info("handlers: balance adjusted")
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
	h.log.WithFields(logrus.Fields{"admin_id": admin.TelegramID, "target_id": id}).Info("handlers: user promoted to admin")
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
	h.log.WithFields(logrus.Fields{"admin_id": admin.TelegramID, "target_id": id}).Info("handlers: admin revoked")
	c.Status(http.StatusNoContent)
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
	h.log.WithFields(logrus.Fields{"admin_id": admin.TelegramID, "target_id": id, "enabled": enabled}).Info("handlers: user referrals toggled")
	c.Status(http.StatusNoContent)
}
