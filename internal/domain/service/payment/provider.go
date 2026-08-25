package payment

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusPaid      Status = "paid"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Provider interface {
	CreateInvoice(ctx context.Context, userID models.TelegramID, amount models.Money, description string) (paymentURL, invoiceID string, err error)
	CheckStatus(ctx context.Context, invoiceID string) (Status, error)
}
