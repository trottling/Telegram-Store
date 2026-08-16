package models

import (
	"time"

	"gorm.io/datatypes"
)

type AdminLog struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	AdminID   int64          `gorm:"index;not null" json:"admin_id"`
	Action    string         `gorm:"size:50;not null" json:"action"` // ban, unban, balance_add, make_admin, product_create, product_update, product_delete, product_add_items, category_create, category_update, category_delete
	TargetID  *int64         `gorm:"index" json:"target_id,omitempty"`
	Details   datatypes.JSON `gorm:"type:jsonb" json:"details,omitempty"`
	CreatedAt time.Time      `json:"created_at"`

	Admin User `gorm:"belongsTo:User;foreignKey:AdminID" json:"-"`
}
