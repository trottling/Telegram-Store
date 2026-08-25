package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trottling/Telegram-Store/admin_backend/dto"
	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// ListPurchases — межпользовательский список покупок с фильтрами
// ?user_id, ?status, ?from, ?to.
func (h *Handlers) ListPurchases(c *gin.Context) {
	offset, limit := parsePagination(c)
	filter := models.PurchaseAdminFilter{
		UserID: parseOptionalIDQuery(c, "user_id"),
		Status: parseOptionalStatusQuery(c, "status"),
		From:   parseOptionalTimeQuery(c, "from"),
		To:     parseOptionalTimeQuery(c, "to"),
	}

	items, err := h.purchaseService.ListAllAdmin(c.Request.Context(), filter, offset, limit)
	if err != nil {
		h.writeError(c, err)
		return
	}
	total, err := h.purchaseService.CountAllAdmin(c.Request.Context(), filter)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Paginated[models.PurchaseAdminItem]{Items: items, Total: total, Offset: offset, Limit: limit})
}

func (h *Handlers) GetPurchase(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id", models.ParsePurchaseID)
	if !ok {
		return
	}
	purchase, err := h.purchaseService.GetAdminByID(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, purchase)
}
