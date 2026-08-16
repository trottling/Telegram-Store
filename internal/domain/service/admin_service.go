package service

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// AdminService: adminID — Telegram ID действующего админа, targetTelegramID — объект действия.
type AdminService interface {
	AddBalance(ctx context.Context, adminID, targetTelegramID int64, amount float64) error
	BanUser(ctx context.Context, adminID, targetTelegramID int64) error
	UnbanUser(ctx context.Context, adminID, targetTelegramID int64) error

	// MakeAdmin — только root-admin, кредит не возвращает.
	MakeAdmin(ctx context.Context, adminID, targetTelegramID int64) error

	// RevokeAdmin: ErrNotAdmin — цель не админ, ErrCannotRevokeRootAdmin — цель root.
	RevokeAdmin(ctx context.Context, adminID, targetTelegramID int64) error

	// SetReferralsEnabled — вкл/выкл начисления targetTelegramID как рефереру.
	SetReferralsEnabled(ctx context.Context, adminID, targetTelegramID int64, enabled bool) error

	// CRUD товаров — categoryID может быть nil (без категории)
	CreateProduct(ctx context.Context, adminID int64, categoryID *int64, name, description string, price float64) (*models.Product, error)
	UpdateProduct(ctx context.Context, adminID int64, productID int64, categoryID *int64, name, description string, price float64, isActive bool) (*models.Product, error)
	DeleteProduct(ctx context.Context, adminID int64, productID int64) error
	AddProductItems(ctx context.Context, adminID int64, productID int64, contents []string) error

	// CRUD категорий — parentID nil = верхний уровень
	CreateCategory(ctx context.Context, adminID int64, parentID *int64, name, description string) (*models.Category, error)
	UpdateCategory(ctx context.Context, adminID int64, categoryID int64, name, description string, parentID *int64) (*models.Category, error)
	DeleteCategory(ctx context.Context, adminID int64, categoryID int64) error

	// UpdateSettings перезаписывает единственную строку настроек бота
	// (models.SettingsID) целиком — Username поддержки и конфиг каждого мерчанта.
	UpdateSettings(ctx context.Context, adminID int64, settings *models.Settings) (*models.Settings, error)

	GetLogs(ctx context.Context, adminID int64, offset, limit int) ([]models.AdminLog, error)

	// ListLogs/CountLogs — журнал по всем админам, adminID nil = без фильтра.
	ListLogs(ctx context.Context, adminID *int64, offset, limit int) ([]models.AdminLog, error)
	CountLogs(ctx context.Context, adminID *int64) (int64, error)
}
