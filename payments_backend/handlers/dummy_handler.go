// Тестовый (dummy) провайдер пополнения — не настоящий мерчант, эмулирует
// сам себя через internal/service/payment.DummyProvider. Отдельный файл от
// webhook_handler.go: те три хендлера про реальных мерчантов и проверяют
// подпись, этому проверять нечего — вызывающая сторона тут наш же процесс,
// не внешний мерчант.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type dummyWebhookRequest struct {
	InvoiceID string `json:"invoice_id" binding:"required"`
}

// DummyWebhook — эмуляция подтверждения мерчанта. Вызывается самим
// DummyProvider через dummyConfirmDelay после CreateInvoice (см.
// internal/service/payment/dummy.go), а не внешним HTTP-клиентом.
func (h *Handlers) DummyWebhook(c *gin.Context) {
	var req dummyWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	err := h.replenishmentService.Confirm(c.Request.Context(), models.MerchantDummy, req.InvoiceID, 0)
	if !h.replenishmentOK(models.MerchantDummy, req.InvoiceID, err) {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
}

// DummyInvoicePage — то, на что ведёт кнопка "Оплатить" у тестового
// провайдера: формы оплаты нет вообще, счёт подтвердится сам через
// dummyConfirmDelay без каких-либо действий пользователя.
func (h *Handlers) DummyInvoicePage(c *gin.Context) {
	c.String(http.StatusOK, "Тестовый счёт %s.\nЭто не настоящий платёж — подтвердится автоматически через несколько секунд.", c.Param("invoice_id"))
}
