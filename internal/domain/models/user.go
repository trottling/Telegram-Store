package models

import (
	"time"

	"gorm.io/gorm"
)

// Role — единственная роль пользователя вместо старых IsBanned/IsAdmin.
// RootAdmin ровно один — TELEGRAM_ROOT_ADMIN_ID из конфига, см. cmd/migrate.
type Role string

const (
	RoleBanned    Role = "banned"
	RoleUser      Role = "user"
	RoleAdmin     Role = "admin"
	RoleRootAdmin Role = "root_admin"
)

// User ключуется по TelegramID — отдельного auto-increment ID нет.
type User struct {
	TelegramID int64          `gorm:"primaryKey" json:"telegram_id"`
	Username   string         `gorm:"size:32" json:"username"`
	Balance    float64        `gorm:"type:decimal(12,2);default:0;not null" json:"balance"`
	Role       Role           `gorm:"type:varchar(20);default:'user';not null" json:"role"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	Purchases []Purchase `gorm:"foreignKey:UserID" json:"purchases,omitempty"`
}

func (u *User) IsBanned() bool {
	return u.Role == RoleBanned
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin || u.Role == RoleRootAdmin
}

func (u *User) IsRootAdmin() bool {
	return u.Role == RoleRootAdmin
}
