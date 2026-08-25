package models

import "time"

// Category — узел дерева каталога произвольной глубины; ParentID == nil — корень.
//
// Видимость (есть ли остаток у самой категории или где-то в её поддереве) не
// хранится в этой строке — она считается фоново, см. CategorySrv.RefreshCatalogSnapshot.
type Category struct {
	ID          CategoryID  `gorm:"type:uuid;primaryKey" json:"id"`
	ParentID    *CategoryID `gorm:"type:uuid;index" json:"parent_id,omitempty"`
	Name        string      `gorm:"size:255;not null" json:"name"`
	Description string      `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}
