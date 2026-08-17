package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trottling/Telegram-Store/admin_backend/dto"
	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// ListAdminLogs — журнал действий админов, можно сузить через ?admin_id=.
func (h *Handlers) ListAdminLogs(c *gin.Context) {
	offset, limit := parsePagination(c)
	adminID := parseOptionalIDQuery(c, "admin_id")

	logs, err := h.adminService.ListLogs(c.Request.Context(), adminID, offset, limit)
	if err != nil {
		h.writeError(c, err)
		return
	}
	total, err := h.adminService.CountLogs(c.Request.Context(), adminID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Paginated[models.AdminLog]{Items: logs, Total: total, Offset: offset, Limit: limit})
}
