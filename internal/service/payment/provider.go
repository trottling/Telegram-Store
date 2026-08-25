package payment

import (
	"context"
	"errors"

	"github.com/trottling/Telegram-Store/internal/domain/models"
	domainpayment "github.com/trottling/Telegram-Store/internal/domain/service/payment"
)

// StubProvider — заглушка PaymentProvider, все вызовы намеренно падают.
type StubProvider struct{}

func NewStubProvider() *StubProvider {
	return &StubProvider{}
}

func (StubProvider) CreateInvoice(ctx context.Context, userID models.TelegramID, amount models.Money, description string) (paymentURL, invoiceID string, err error) {
	return "", "", errors.New("payment provider is not implemented yet")
}

func (StubProvider) CheckStatus(ctx context.Context, invoiceID string) (domainpayment.PaymentStatus, error) {
	return "", errors.New("payment provider is not implemented yet")
}
