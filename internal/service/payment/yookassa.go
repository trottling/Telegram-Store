package payment

import (
	"context"

	"github.com/rvinnie/yookassa-sdk-go/yookassa"
	yoocommon "github.com/rvinnie/yookassa-sdk-go/yookassa/common"
	yoopayment "github.com/rvinnie/yookassa-sdk-go/yookassa/payment"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/service"
	domainpayment "github.com/trottling/Telegram-Store/internal/domain/service/payment"
)

// YooKassaProvider — обёртка над github.com/rvinnie/yookassa-sdk-go. Креды
// читаются из Settings на каждый вызов — админ меняет их в панели без
// перезапуска бота.
type YooKassaProvider struct {
	settingsService service.SettingsService
}

func NewYooKassaProvider(settingsService service.SettingsService) *YooKassaProvider {
	return &YooKassaProvider{settingsService: settingsService}
}

func (p *YooKassaProvider) CreateInvoice(ctx context.Context, _ int64, amount models.Money, description string) (string, string, error) {
	settings, err := p.settingsService.Get(ctx)
	if err != nil {
		return "", "", err
	}
	cfg := settings.YooKassa
	if !cfg.Enabled {
		return "", "", domainerrors.ErrMerchantDisabled
	}
	if amount.LessThan(cfg.MinAmount) || (!cfg.MaxAmount.IsZero() && amount.GreaterThan(cfg.MaxAmount)) {
		return "", "", domainerrors.ErrAmountOutOfRange
	}

	handler := yookassa.NewPaymentHandler(yookassa.NewClient(cfg.ShopID, cfg.SecretKey))

	pay, err := handler.CreatePayment(ctx, &yoopayment.Payment{
		Amount:       &yoocommon.Amount{Value: amount.String(), Currency: "RUB"},
		Description:  description,
		Capture:      true,
		Confirmation: &yoopayment.Redirect{Type: yoopayment.TypeRedirect},
	})
	if err != nil {
		return "", "", err
	}

	link, err := handler.ParsePaymentLink(pay)
	if err != nil {
		return "", "", err
	}
	return link, pay.ID, nil
}

func (p *YooKassaProvider) CheckStatus(ctx context.Context, invoiceID string) (domainpayment.PaymentStatus, error) {
	settings, err := p.settingsService.Get(ctx)
	if err != nil {
		return "", err
	}
	cfg := settings.YooKassa

	handler := yookassa.NewPaymentHandler(yookassa.NewClient(cfg.ShopID, cfg.SecretKey))
	pay, err := handler.FindPayment(ctx, invoiceID)
	if err != nil {
		return "", err
	}
	return yooKassaStatus(pay.Status), nil
}

func yooKassaStatus(status yoopayment.Status) domainpayment.PaymentStatus {
	switch status {
	case yoopayment.Succeeded:
		return domainpayment.PaymentStatusPaid
	case yoopayment.Canceled:
		return domainpayment.PaymentStatusFailed
	default: // pending, waiting_for_capture
		return domainpayment.PaymentStatusPending
	}
}
