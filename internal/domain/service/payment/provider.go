package payment

import "context"

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusPaid      PaymentStatus = "paid"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

type PaymentProvider interface {
	CreateInvoice(ctx context.Context, userID int64, amount float64, description string) (paymentURL, invoiceID string, err error)
	CheckStatus(ctx context.Context, invoiceID string) (PaymentStatus, error)
}
