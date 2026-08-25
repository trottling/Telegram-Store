package payment

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/nikita-vanyasin/tinkoff"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	domainpayment "github.com/trottling/Telegram-Store/internal/domain/service/payment"
)

// tinkoffTimeout — таймаут запросов к Tinkoff. Задаём явно: NewClient в SDK
// берёт http.DefaultClient, у которого таймаута нет вовсе, и зависшее
// соединение держало бы горутину хендлера бота бесконечно (ctx долгого
// поллинга дедлайна не имеет).
const tinkoffTimeout = 15 * time.Second

// TinkoffProvider — обёртка над github.com/nikita-vanyasin/tinkoff (Tinkoff
// Acquiring API, одностадийная оплата PayTypeOneStep). Креды читаются из
// Settings на каждый вызов, как у остальных провайдеров.
type TinkoffProvider struct {
	settingsService service.SettingsService
	backendURL      string
	httpClient      *http.Client
}

func NewTinkoffProvider(settingsService service.SettingsService, backendURL string) *TinkoffProvider {
	return &TinkoffProvider{
		settingsService: settingsService,
		backendURL:      backendURL,
		httpClient:      &http.Client{Timeout: tinkoffTimeout},
	}
}

// newClient — креды приходят из Settings на каждый вызов, поэтому клиент
// собирается заново, а http-клиент с таймаутом переиспользуется.
func (p *TinkoffProvider) newClient(cfg models.TinkoffSettings) *tinkoff.Client {
	return tinkoff.NewClientWithOptions(
		tinkoff.WithTerminalKey(cfg.TerminalKey),
		tinkoff.WithPassword(cfg.Password),
		tinkoff.WithHTTPClient(p.httpClient),
	)
}

func (p *TinkoffProvider) CreateInvoice(ctx context.Context, userID models.TelegramID, amount models.Money, description string) (string, string, error) {
	settings, err := p.settingsService.Get(ctx)
	if err != nil {
		return "", "", err
	}
	cfg := settings.Tinkoff
	if !cfg.Enabled {
		return "", "", domainerrors.ErrMerchantDisabled
	}
	if amount.LessThan(cfg.MinAmount) || (!cfg.MaxAmount.IsZero() && amount.GreaterThan(cfg.MaxAmount)) {
		return "", "", domainerrors.ErrAmountOutOfRange
	}

	client := p.newClient(cfg)

	resp, err := client.InitWithContext(ctx, &tinkoff.InitRequest{
		Amount:          uint64(math.Round(amount.Float64() * 100)), // копейки
		OrderID:         orderID(userID),
		Description:     description,
		PayType:         tinkoff.PayTypeOneStep,
		NotificationURL: p.backendURL + "/api/webhooks/tinkoff",
	})
	if err != nil {
		return "", "", err
	}

	return resp.PaymentURL, resp.PaymentID, nil
}

func (p *TinkoffProvider) CheckStatus(ctx context.Context, invoiceID string) (domainpayment.PaymentStatus, error) {
	settings, err := p.settingsService.Get(ctx)
	if err != nil {
		return "", err
	}
	cfg := settings.Tinkoff

	client := p.newClient(cfg)
	resp, err := client.GetStateWithContext(ctx, &tinkoff.GetStateRequest{PaymentID: invoiceID})
	if err != nil {
		return "", err
	}
	return tinkoffStatus(resp.Status), nil
}

func tinkoffStatus(status string) domainpayment.PaymentStatus {
	switch status {
	case tinkoff.StatusConfirmed:
		return domainpayment.PaymentStatusPaid
	case tinkoff.StatusRejected, tinkoff.StatusAuthFail, tinkoff.StatusCanceled, tinkoff.StatusDeadlineExpired, tinkoff.StatusReversed:
		return domainpayment.PaymentStatusFailed
	default:
		return domainpayment.PaymentStatusPending
	}
}

// orderID — уникален на каждый вызов, Tinkoff требует несовпадающий OrderId на каждый Init.
func orderID(userID models.TelegramID) string {
	return fmt.Sprintf("%d-%d", userID, time.Now().UnixNano())
}
