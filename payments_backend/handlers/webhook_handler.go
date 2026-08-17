// Вебхуки платёжных мерчантов — единственные роуты payments_backend, все без
// авторизации (вызывающая сторона — сервер мерчанта, а не залогиненный
// админ), каждый вебхук сам проверяет подпись.
package handlers

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nikita-vanyasin/tinkoff"
	"github.com/rvinnie/yookassa-sdk-go/yookassa"
	yoopayment "github.com/rvinnie/yookassa-sdk-go/yookassa/payment"
	yoowebhook "github.com/rvinnie/yookassa-sdk-go/yookassa/webhook"
	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// CrystalPayWebhook — POST callback_url из invoice/create. Подпись:
// sha1(id + ":" + salt), salt — отдельный секрет, не auth_secret.
func (h *Handlers) CrystalPayWebhook(c *gin.Context) {
	var payload struct {
		ID        string `json:"id"`
		State     string `json:"state"`
		Signature string `json:"signature"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	settings, err := h.settingsService.Get(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("handlers: crystalpay webhook failed to get settings")
		c.Status(http.StatusInternalServerError)
		return
	}

	sum := sha1.Sum([]byte(payload.ID + ":" + settings.CrystalPay.Salt))
	expected := hex.EncodeToString(sum[:])
	if !hmac.Equal([]byte(expected), []byte(payload.Signature)) {
		h.log.Warn("handlers: crystalpay webhook signature mismatch")
		c.Status(http.StatusForbidden)
		return
	}

	switch payload.State {
	case "payed":
		err = h.replenishmentService.Confirm(c.Request.Context(), models.MerchantCrystalPay, payload.ID)
	case "failed", "unavailable":
		err = h.replenishmentService.Fail(c.Request.Context(), models.MerchantCrystalPay, payload.ID)
	}
	if err != nil {
		h.log.WithError(err).Error("handlers: crystalpay webhook confirm failed")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
}

// TinkoffWebhook — POST NotificationURL из Init. Подпись (Token) проверяется
// самим SDK (ParseNotification) — пересчитывает token из тела запроса и
// текущего Password и сверяет.
func (h *Handlers) TinkoffWebhook(c *gin.Context) {
	settings, err := h.settingsService.Get(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("handlers: tinkoff webhook failed to get settings")
		c.Status(http.StatusInternalServerError)
		return
	}

	client := tinkoff.NewClient(settings.Tinkoff.TerminalKey, settings.Tinkoff.Password)
	notification, err := client.ParseNotification(c.Request.Body)
	if err != nil {
		h.log.WithError(err).Warn("handlers: tinkoff webhook verification failed")
		c.Status(http.StatusForbidden)
		return
	}

	invoiceID := strconv.FormatUint(notification.PaymentID, 10)
	switch notification.Status {
	case tinkoff.StatusConfirmed:
		err = h.replenishmentService.Confirm(c.Request.Context(), models.MerchantTinkoff, invoiceID)
	case tinkoff.StatusRejected, tinkoff.StatusAuthFail, tinkoff.StatusCanceled, tinkoff.StatusDeadlineExpired, tinkoff.StatusReversed:
		err = h.replenishmentService.Fail(c.Request.Context(), models.MerchantTinkoff, invoiceID)
	}
	if err != nil {
		h.log.WithError(err).Error("handlers: tinkoff webhook confirm failed")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.String(http.StatusOK, client.GetNotificationSuccessResponse())
}

// YooKassaWebhook — ЮKassa не подписывает уведомления, поэтому payload
// используется только как триггер: реальный статус берём авторизованным
// FindPayment, телу запроса не доверяем.
func (h *Handlers) YooKassaWebhook(c *gin.Context) {
	var event yoowebhook.WebhookEvent[yoopayment.Payment]
	if err := c.ShouldBindJSON(&event); err != nil || event.Object.ID == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	settings, err := h.settingsService.Get(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("handlers: yookassa webhook failed to get settings")
		c.Status(http.StatusInternalServerError)
		return
	}

	handler := yookassa.NewPaymentHandler(yookassa.NewClient(settings.YooKassa.ShopID, settings.YooKassa.SecretKey))
	pay, err := handler.FindPayment(c.Request.Context(), event.Object.ID)
	if err != nil {
		h.log.WithError(err).Warn("handlers: yookassa webhook re-fetch failed")
		c.Status(http.StatusInternalServerError)
		return
	}

	switch pay.Status {
	case yoopayment.Succeeded:
		err = h.replenishmentService.Confirm(c.Request.Context(), models.MerchantYooKassa, pay.ID)
	case yoopayment.Canceled:
		err = h.replenishmentService.Fail(c.Request.Context(), models.MerchantYooKassa, pay.ID)
	}
	if err != nil {
		h.log.WithError(err).Error("handlers: yookassa webhook confirm failed")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
}
