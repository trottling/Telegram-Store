package models

import "time"

// Category — узел дерева каталога произвольной глубины; ParentID == nil — корень.
type Category struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ParentID    *int64    `gorm:"index" json:"parent_id,omitempty"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`

	Parent   *Category  `gorm:"foreignKey:ParentID" json:"-"`
	Children []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Products []Product  `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
}
