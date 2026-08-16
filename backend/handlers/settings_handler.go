package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/backend/dto"
	"github.com/trottling/Telegram-Store/backend/middleware"
)

func (h *Handlers) GetSettings(c *gin.Context) {
	settings, err := h.settingsService.Get(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *Handlers) UpdateSettings(c *gin.Context) {
	admin, _ := middleware.AdminFromContext(c)
	var req dto.UpdateSettingsRequest
	if !decodeJSON(c, &req) {
		return
	}
	settings, err := h.adminService.UpdateSettings(c.Request.Context(), admin.TelegramID, req.SupportUsername)
	if err != nil {
		h.writeError(c, err)
		return
	}
	h.log.WithFields(logrus.Fields{"admin_id": admin.TelegramID}).Info("handlers: settings updated")
	c.JSON(http.StatusOK, settings)
}
