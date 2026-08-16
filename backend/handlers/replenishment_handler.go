package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trottling/Telegram-Store/backend/dto"
	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// ListReplenishments — межпользовательский список пополнений, ?user_id опционален.
func (h *Handlers) ListReplenishments(c *gin.Context) {
	offset, limit := parsePagination(c)
	userID := parseOptionalIDQuery(c, "user_id")

	items, err := h.replenishmentService.ListAllAdmin(c.Request.Context(), userID, offset, limit)
	if err != nil {
		h.writeError(c, err)
		return
	}
	total, err := h.replenishmentService.CountAllAdmin(c.Request.Context(), userID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Paginated[models.ReplenishmentAdminItem]{Items: items, Total: total, Offset: offset, Limit: limit})
}
