// Вебхуки платёжных мерчантов — единственные роуты payments_backend, все без
// авторизации (вызывающая сторона — сервер мерчанта, а не залогиненный
// админ), каждый вебхук сам проверяет подпись.
package handlers

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nikita-vanyasin/tinkoff"
	"github.com/rvinnie/yookassa-sdk-go/yookassa"
	yoopayment "github.com/rvinnie/yookassa-sdk-go/yookassa/payment"
	yoowebhook "github.com/rvinnie/yookassa-sdk-go/yookassa/webhook"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/service/payment"
)

// replenishmentOK — можно ли отвечать мерчанту успехом. Тело успешного ответа
// у мерчантов разное (Tinkoff ждёт своё), поэтому пишет его вызывающий.
//
// Неизвестный invoice_id — не наша внутренняя ошибка, и повторами он не
// вылечится: мерчанты ретраят 5xx часами, поэтому такой вебхук подтверждаем и
// оставляем след в логе. Всё остальное — настоящий сбой, тут ретрай как раз
// нужен.
func (h *Handlers) replenishmentOK(merchant models.Merchant, invoiceID string, err error) bool {
	if err == nil {
		return true
	}

	if errors.Is(err, domainerrors.ErrReplenishmentNotFound) {
		h.log.Warnw("handlers: webhook for unknown invoice, acknowledged without retry", "merchant", merchant, "invoice_id", invoiceID)
		return true
	}

	h.log.Errorw("handlers: webhook processing failed", "error", err, "merchant", merchant, "invoice_id", invoiceID)
	return false
}

// CrystalPayWebhook — POST callback_url из invoice/create. Подпись:
// sha1(id + ":" + salt), salt — отдельный секрет, не auth_secret.
//
// Подпись покрывает только id и постоянна на весь срок жизни счёта, а поле
// state в том же теле ничем не подписано — то есть валидная подпись из вебхука
// о неудачном платеже годится и для подделки state="payed". Поэтому телу не
// доверяем: статус спрашиваем у мерчанта сами, как в YooKassaWebhook. Проверка
// подписи остаётся первым фильтром, чтобы посторонний не гонял нас за
// статусами чужих счётов.
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
		h.log.Errorw("handlers: crystalpay webhook failed to get settings", "error", err)
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

	status, err := h.crystalPayProvider.CheckStatus(c.Request.Context(), payload.ID)
	if err != nil {
		h.log.Warnw("handlers: crystalpay webhook status re-check failed", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	switch status {
	case payment.StatusPaid:
		// CheckStatus суммы не возвращает — сверять нечего.
		err = h.replenishmentService.Confirm(c.Request.Context(), models.MerchantCrystalPay, payload.ID, models.Money{})
	case payment.StatusFailed:
		err = h.replenishmentService.Fail(c.Request.Context(), models.MerchantCrystalPay, payload.ID)
	}
	if !h.replenishmentOK(models.MerchantCrystalPay, payload.ID, err) {
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
		h.log.Errorw("handlers: tinkoff webhook failed to get settings", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	client := tinkoff.NewClient(settings.Tinkoff.TerminalKey, settings.Tinkoff.Password)
	notification, err := client.ParseNotification(c.Request.Body)
	if err != nil {
		h.log.Warnw("handlers: tinkoff webhook verification failed", "error", err)
		c.Status(http.StatusForbidden)
		return
	}

	invoiceID := strconv.FormatUint(notification.PaymentID, 10)
	switch notification.Status {
	case tinkoff.StatusConfirmed:
		// Tinkoff считает в копейках, наши суммы — в рублях.
		var paidAmount models.Money
		if paidAmount, err = models.MoneyFromCents(int64(notification.Amount)); err != nil {
			break
		}
		err = h.replenishmentService.Confirm(c.Request.Context(), models.MerchantTinkoff, invoiceID, paidAmount)
	case tinkoff.StatusRejected, tinkoff.StatusAuthFail, tinkoff.StatusCanceled, tinkoff.StatusDeadlineExpired, tinkoff.StatusReversed:
		err = h.replenishmentService.Fail(c.Request.Context(), models.MerchantTinkoff, invoiceID)
	}
	if !h.replenishmentOK(models.MerchantTinkoff, invoiceID, err) {
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
		h.log.Errorw("handlers: yookassa webhook failed to get settings", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	handler := yookassa.NewPaymentHandler(yookassa.NewClient(settings.YooKassa.ShopID, settings.YooKassa.SecretKey))
	pay, err := handler.FindPayment(c.Request.Context(), event.Object.ID)
	if err != nil {
		h.log.Warnw("handlers: yookassa webhook re-fetch failed", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	switch pay.Status {
	case yoopayment.Succeeded:
		err = h.replenishmentService.Confirm(c.Request.Context(), models.MerchantYooKassa, pay.ID, yooKassaPaidAmount(pay))
	case yoopayment.Canceled:
		err = h.replenishmentService.Fail(c.Request.Context(), models.MerchantYooKassa, pay.ID)
	}
	if !h.replenishmentOK(models.MerchantYooKassa, pay.ID, err) {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
}

// yooKassaPaidAmount — сумма из ответа ЮKassa в рублях; нулевое значение,
// если её нет, не парсится или валюта не рублёвая (сверять тогда нечего).
func yooKassaPaidAmount(pay *yoopayment.Payment) models.Money {
	if pay.Amount == nil || pay.Amount.Currency != "RUB" {
		return models.Money{}
	}

	amount, err := models.NewMoney(pay.Amount.Value)
	if err != nil {
		return models.Money{}
	}
	return amount
}
