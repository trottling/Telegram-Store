package models

import "time"

// Category — узел дерева каталога произвольной глубины; ParentID == nil — корень.
type Category struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ParentID    *int64    `gorm:"index" json:"parent_id,omitempty"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	// HasStock — денормализованный агрегат: true, если у самой категории или
	// у любого её потомка есть активный товар в наличии. Поддерживается
	// репозиторием (RecomputeStock) на каждой операции, меняющей остаток или
	// дерево, — ListChildren фильтрует по нему напрямую вместо рекурсивного
	// CTE по всему поддереву на каждое чтение.
	HasStock bool `gorm:"not null;default:false" json:"has_stock"`

	Parent   *Category  `gorm:"foreignKey:ParentID" json:"-"`
	Children []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Products []Product  `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
}
