package models

import "time"

type PurchaseStatus string

const PurchaseStatusCompleted PurchaseStatus = "completed"

type Purchase struct {
	ID        PurchaseID     `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    TelegramID     `gorm:"index;not null" json:"user_id"`
	ProductID ProductID      `gorm:"type:uuid;index;not null" json:"product_id"`
	ItemID    *ProductItemID `gorm:"type:uuid;uniqueIndex" json:"item_id,omitempty"` // конкретный выданный товар
	BatchID   BatchID        `gorm:"type:uuid;index" json:"batch_id"`
	Amount    Money          `gorm:"type:decimal(12,2);not null" json:"amount"`
	Status    PurchaseStatus `gorm:"size:20;default:'pending';not null" json:"status"`
	// index — межпользовательский листинг админки сортирует по этой колонке.
	CreatedAt   time.Time  `gorm:"index" json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	Product Product      `gorm:"belongsTo:Product;foreignKey:ProductID" json:"-"`
	Item    *ProductItem `gorm:"belongsTo:ProductItem;foreignKey:ItemID" json:"item,omitempty"`
}

// PurchaseBatchSummary — агрегат одного вызова Buy(), одна строка на батч
// ("5x Товар"), не на единицу.
type PurchaseBatchSummary struct {
	BatchID     BatchID   `json:"batch_id"`
	ProductID   ProductID `json:"product_id"`
	ProductName string    `json:"product_name"`
	UnitPrice   Money     `json:"unit_price"`
	Quantity    int       `json:"quantity"`
	TotalAmount Money     `json:"total_amount"`
	CreatedAt   time.Time `json:"created_at"`
}

// PurchaseAdminFilter — фильтр для ListAll/CountAll, все поля опциональны.
type PurchaseAdminFilter struct {
	UserID *TelegramID
	Status *PurchaseStatus
	From   *time.Time
	To     *time.Time
}

// PurchaseAdminItem — Purchase с подмешанными Username/ProductName для админки.
type PurchaseAdminItem struct {
	ID          PurchaseID     `json:"id"`
	UserID      TelegramID     `json:"user_id"`
	Username    string         `json:"username"`
	ProductID   ProductID      `json:"product_id"`
	ProductName string         `json:"product_name"`
	ItemID      *ProductItemID `json:"item_id,omitempty"`
	BatchID     BatchID        `json:"batch_id"`
	Amount      Money          `json:"amount"`
	Status      PurchaseStatus `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}
