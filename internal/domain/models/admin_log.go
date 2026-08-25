package models

import (
	"time"

	"gorm.io/datatypes"
)

// AdminLog.TargetID — полиморфная ссылка: в зависимости от Action это либо
// User.TelegramID (ban/unban/balance_add/make_admin/revoke_admin/referral_*),
// либо ProductID/CategoryID (product_*/category_*), либо Settings.ID
// (settings_update) — четыре разных типовых пространства в одном поле.
// Ни один запрос по нему не фильтрует (чистый аудит-лог), поэтому строка, а
// не типизированный ID: каждый вызывающий сам форматирует своё значение
// (см. AdminSrv.logAction).
type AdminLog struct {
	ID       AdminLogID     `gorm:"type:uuid;primaryKey" json:"id"`
	AdminID  TelegramID     `gorm:"index;not null" json:"admin_id"`
	Action   string         `gorm:"size:50;not null" json:"action"` // ban, unban, balance_add, make_admin, product_create, product_update, product_delete, product_add_items, category_create, category_update, category_delete
	TargetID *string        `gorm:"index" json:"target_id,omitempty"`
	Details  datatypes.JSON `gorm:"type:jsonb" json:"details,omitempty"`
	// index — аудит-лог всегда отдаётся отсортированным по этой колонке.
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}
