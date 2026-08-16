package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trottling/Telegram-Store/backend/dto"
	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// ListReplenishments — межпользовательский список пополнений, ?user_id/?merchant опциональны.
func (h *Handlers) ListReplenishments(c *gin.Context) {
	offset, limit := parsePagination(c)
	filter := models.ReplenishmentAdminFilter{
		UserID:   parseOptionalIDQuery(c, "user_id"),
		Merchant: parseOptionalMerchantQuery(c, "merchant"),
	}

	items, err := h.replenishmentService.ListAllAdmin(c.Request.Context(), filter, offset, limit)
	if err != nil {
		h.writeError(c, err)
		return
	}
	total, err := h.replenishmentService.CountAllAdmin(c.Request.Context(), filter)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Paginated[models.ReplenishmentAdminItem]{Items: items, Total: total, Offset: offset, Limit: limit})
}
