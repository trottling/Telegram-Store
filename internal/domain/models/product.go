package models

import "time"

type Product struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	CategoryID  *int64    `gorm:"index" json:"category_id,omitempty"` // nil = uncategorized
	Name        string    `gorm:"size:255;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Price       Money     `gorm:"type:decimal(12,2);not null" json:"price"`
	IsActive    bool      `gorm:"default:true;not null" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`

	Category *Category     `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Items    []ProductItem `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

type ProductItem struct {
	ID        int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	ProductID int64      `gorm:"index;not null" json:"product_id"`
	Content   string     `gorm:"type:text;not null" json:"-"` // сам товар (ключ, ссылка)
	IsSold    bool       `gorm:"default:false;not null" json:"is_sold"`
	SoldAt    *time.Time `json:"sold_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`

	Product  Product   `gorm:"belongsTo:Product;foreignKey:ProductID" json:"-"`
	Purchase *Purchase `gorm:"foreignKey:ItemID" json:"-"`
}

// ProductAdminSummary — вид Product для админ-листинга: все товары
// независимо от IsActive/остатка, плюс имя категории и живой счётчик остатка.
type ProductAdminSummary struct {
	ID             int64     `json:"id"`
	CategoryID     *int64    `json:"category_id,omitempty"`
	CategoryName   string    `json:"category_name,omitempty"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Price          Money     `json:"price"`
	IsActive       bool      `json:"is_active"`
	AvailableCount int       `json:"available_count"`
	CreatedAt      time.Time `json:"created_at"`
}
