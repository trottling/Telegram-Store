package payment

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusPaid      PaymentStatus = "paid"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

type PaymentProvider interface {
	CreateInvoice(ctx context.Context, userID models.TelegramID, amount models.Money, description string) (paymentURL, invoiceID string, err error)
	CheckStatus(ctx context.Context, invoiceID string) (PaymentStatus, error)
}
