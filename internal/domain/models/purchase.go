package models

import "time"

type PurchaseStatus string

const (
	PurchaseStatusPending   PurchaseStatus = "pending"
	PurchaseStatusCompleted PurchaseStatus = "completed"
	PurchaseStatusCancelled PurchaseStatus = "cancelled"
)

type Purchase struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64          `gorm:"index;not null" json:"user_id"`
	ProductID   int64          `gorm:"index;not null" json:"product_id"`
	ItemID      *int64         `gorm:"uniqueIndex" json:"item_id,omitempty"` // конкретный выданный товар
	BatchID     string         `gorm:"size:36;index" json:"batch_id"`
	Amount      float64        `gorm:"type:decimal(12,2);not null" json:"amount"`
	Status      PurchaseStatus `gorm:"size:20;default:'pending';not null" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`

	User    User         `gorm:"belongsTo:User;foreignKey:UserID" json:"-"`
	Product Product      `gorm:"belongsTo:Product;foreignKey:ProductID" json:"-"`
	Item    *ProductItem `gorm:"belongsTo:ProductItem;foreignKey:ItemID" json:"item,omitempty"`
}

// PurchaseBatchSummary — агрегат одного вызова Buy(), одна строка на батч
// ("5x Товар"), не на единицу.
type PurchaseBatchSummary struct {
	BatchID     string    `json:"batch_id"`
	ProductID   int64     `json:"product_id"`
	ProductName string    `json:"product_name"`
	UnitPrice   float64   `json:"unit_price"`
	Quantity    int       `json:"quantity"`
	TotalAmount float64   `json:"total_amount"`
	CreatedAt   time.Time `json:"created_at"`
}

// PurchaseAdminFilter — фильтр для ListAll/CountAll, все поля опциональны.
type PurchaseAdminFilter struct {
	UserID *int64
	Status *PurchaseStatus
	From   *time.Time
	To     *time.Time
}

// PurchaseAdminItem — Purchase с подмешанными Username/ProductName для админки.
type PurchaseAdminItem struct {
	ID          int64          `json:"id"`
	UserID      int64          `json:"user_id"`
	Username    string         `json:"username"`
	ProductID   int64          `json:"product_id"`
	ProductName string         `json:"product_name"`
	ItemID      *int64         `json:"item_id,omitempty"`
	BatchID     string         `json:"batch_id"`
	Amount      float64        `json:"amount"`
	Status      PurchaseStatus `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}
