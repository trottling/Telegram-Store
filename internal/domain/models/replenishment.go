package models

import "time"

// Merchant — источник пополнения. MerchantReferral зарезервирован под
// будущий процент с рефералов — начисления оттуда тоже будут писаться сюда
// же, отдельной логики начисления пока нет.
type Merchant string

const (
	MerchantCrystalPay Merchant = "crystalpay"
	MerchantYooKassa   Merchant = "yookassa"
	MerchantTinkoff    Merchant = "tinkoff"
	MerchantReferral   Merchant = "referral"
	// MerchantDummy — тестовый провайдер без реальной оплаты (см.
	// internal/service/payment.DummyProvider), для разработки/демо.
	MerchantDummy Merchant = "dummy"
)

type ReplenishmentStatus string

const (
	ReplenishmentStatusPending   ReplenishmentStatus = "pending"
	ReplenishmentStatusPaid      ReplenishmentStatus = "paid"
	ReplenishmentStatusFailed    ReplenishmentStatus = "failed"
	ReplenishmentStatusCancelled ReplenishmentStatus = "cancelled"
)

// Replenishment — одна попытка пополнения баланса (один счёт у мерчанта).
type Replenishment struct {
	ID        int64               `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64               `gorm:"index;not null" json:"user_id"`
	Merchant  Merchant            `gorm:"type:varchar(20);not null" json:"merchant"`
	InvoiceID string              `gorm:"size:128;index" json:"invoice_id"` // ID счёта у мерчанта
	Amount    float64             `gorm:"type:decimal(12,2);not null" json:"amount"`
	Status    ReplenishmentStatus `gorm:"type:varchar(20);default:'pending';not null" json:"status"`
	// index — межпользовательский листинг админки сортирует по этой колонке.
	CreatedAt   time.Time  `gorm:"index" json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	User User `gorm:"belongsTo:User;foreignKey:UserID" json:"-"`
}

// ReplenishmentAdminFilter — фильтр для ListAllAdmin/CountAllAdmin, оба поля опциональны.
type ReplenishmentAdminFilter struct {
	UserID   *int64
	Merchant *Merchant
}

// ReplenishmentAdminItem — Replenishment с подмешанным Username, для админки.
type ReplenishmentAdminItem struct {
	ID          int64               `json:"id"`
	UserID      int64               `json:"user_id"`
	Username    string              `json:"username"`
	Merchant    Merchant            `json:"merchant"`
	InvoiceID   string              `json:"invoice_id"`
	Amount      float64             `json:"amount"`
	Status      ReplenishmentStatus `json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	CompletedAt *time.Time          `json:"completed_at,omitempty"`
}
