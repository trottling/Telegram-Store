package payment

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/nikita-vanyasin/tinkoff"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	domainpayment "github.com/trottling/Telegram-Store/internal/domain/service/payment"
)

// TinkoffProvider — обёртка над github.com/nikita-vanyasin/tinkoff (Tinkoff
// Acquiring API, одностадийная оплата PayTypeOneStep). Креды читаются из
// Settings на каждый вызов, как у остальных провайдеров.
type TinkoffProvider struct {
	settingsService service.SettingsService
	backendURL      string
}

func NewTinkoffProvider(settingsService service.SettingsService, backendURL string) *TinkoffProvider {
	return &TinkoffProvider{settingsService: settingsService, backendURL: backendURL}
}

func (p *TinkoffProvider) CreateInvoice(ctx context.Context, userID int64, amount float64, description string) (string, string, error) {
	settings, err := p.settingsService.Get(ctx)
	if err != nil {
		return "", "", err
	}
	cfg := settings.Tinkoff
	if !cfg.Enabled {
		return "", "", domainerrors.ErrMerchantDisabled
	}
	if amount < cfg.MinAmount || (cfg.MaxAmount > 0 && amount > cfg.MaxAmount) {
		return "", "", domainerrors.ErrAmountOutOfRange
	}

	client := tinkoff.NewClient(cfg.TerminalKey, cfg.Password)

	resp, err := client.InitWithContext(ctx, &tinkoff.InitRequest{
		Amount:          uint64(math.Round(amount * 100)), // копейки
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

	client := tinkoff.NewClient(cfg.TerminalKey, cfg.Password)
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
func orderID(userID int64) string {
	return fmt.Sprintf("%d-%d", userID, time.Now().UnixNano())
}
