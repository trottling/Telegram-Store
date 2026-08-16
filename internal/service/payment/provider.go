package payment

import (
	"context"
	"errors"

	domainpayment "github.com/trottling/TG-Store/internal/domain/service/payment"
)

// StubProvider — заглушка PaymentProvider, все вызовы намеренно падают.
type StubProvider struct{}

func NewStubProvider() *StubProvider {
	return &StubProvider{}
}

func (StubProvider) CreateInvoice(ctx context.Context, userID int64, amount float64, description string) (paymentURL, invoiceID string, err error) {
	return "", "", errors.New("payment provider is not implemented yet")
}

func (StubProvider) CheckStatus(ctx context.Context, invoiceID string) (domainpayment.PaymentStatus, error) {
	return "", errors.New("payment provider is not implemented yet")
}
