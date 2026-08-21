package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	domainpayment "github.com/trottling/Telegram-Store/internal/domain/service/payment"
	"go.uber.org/zap"
)

// dummyConfirmDelay — через сколько тестовый провайдер сам подтверждает
// счёт, имитируя вебхук реального мерчанта.
const dummyConfirmDelay = 10 * time.Second

// DummyProvider — тестовый провайдер без реальной оплаты, для разработки и
// демо (см. Settings.Dummy — включается/выключается в панели, как настоящие
// мерчанты). CreateInvoice работает как у всех: проверяет Enabled/диапазон
// суммы, возвращает "ссылку на оплату" — но вместо реального платёжного шлюза
// сама себе бьёт по /api/webhooks/dummy через dummyConfirmDelay, имитируя
// подтверждение мерчанта. Деньги нигде не двигаются кроме как внутри нашей
// же БД через обычный ReplenishmentSrv.Confirm.
type DummyProvider struct {
	settingsService service.SettingsService
	backendURL      string
	httpClient      *http.Client
	log             *zap.SugaredLogger
}

func NewDummyProvider(settingsService service.SettingsService, backendURL string, log *zap.SugaredLogger) *DummyProvider {
	return &DummyProvider{
		settingsService: settingsService,
		backendURL:      backendURL,
		httpClient:      &http.Client{Timeout: 15 * time.Second},
		log:             log,
	}
}

type dummyWebhookPayload struct {
	InvoiceID string `json:"invoice_id"`
}

func (p *DummyProvider) CreateInvoice(ctx context.Context, _ int64, amount float64, _ string) (string, string, error) {
	settings, err := p.settingsService.Get(ctx)
	if err != nil {
		return "", "", err
	}
	cfg := settings.Dummy
	if !cfg.Enabled {
		return "", "", domainerrors.ErrMerchantDisabled
	}
	if amount < cfg.MinAmount || (cfg.MaxAmount > 0 && amount > cfg.MaxAmount) {
		return "", "", domainerrors.ErrAmountOutOfRange
	}

	invoiceID := uuid.NewString()
	go p.confirmAfterDelay(invoiceID)

	return p.backendURL + "/dummy/" + invoiceID, invoiceID, nil
}

// confirmAfterDelay — отдельная горутина, не привязанная к ctx запроса: тот
// умрёт вместе с ответом бота задолго до срабатывания таймера. Ошибки только
// логируются — это лучшее, что можно сделать из detached-горутины, вызывать
// оттуда больше некого.
func (p *DummyProvider) confirmAfterDelay(invoiceID string) {
	time.Sleep(dummyConfirmDelay)

	raw, err := json.Marshal(dummyWebhookPayload{InvoiceID: invoiceID})
	if err != nil {
		p.log.Errorw("dummy_provider: failed to marshal self-webhook payload", "error", err, "invoice_id", invoiceID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.httpClient.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.backendURL+"/api/webhooks/dummy", bytes.NewReader(raw))
	if err != nil {
		p.log.Errorw("dummy_provider: failed to build self-webhook request", "error", err, "invoice_id", invoiceID)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		// Частая причина — PAYMENTS_BACKEND_URL смотрит в недостижимое место
		// с точки зрения бота (тот же класс ошибки, что и у настоящих
		// мерчантов, см. config.PaymentsConfig.IsLoopbackURL).
		p.log.Errorw("dummy_provider: self-webhook call failed", "error", err, "invoice_id", invoiceID)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		p.log.Errorw("dummy_provider: self-webhook returned non-200", "status", resp.StatusCode, "invoice_id", invoiceID)
	}
}

// CheckStatus не вызывается ни для одного сценария: DummyWebhook уже знает
// статус (сам его и породил), перепроверка через CheckStatus нужна только
// CrystalPay. Существует ради полноты интерфейса PaymentProvider.
func (p *DummyProvider) CheckStatus(context.Context, string) (domainpayment.PaymentStatus, error) {
	return "", fmt.Errorf("dummy: CheckStatus is not supported")
}
