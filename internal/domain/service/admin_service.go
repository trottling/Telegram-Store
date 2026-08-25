package service

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// AdminService: adminID — Telegram ID действующего админа, targetTelegramID — объект действия.
type AdminService interface {
	// AddBalance — amount знаковый (админ может и списать, не только начислить),
	// поэтому decimal.Decimal, а не models.Money: Money отрицательным не бывает.
	AddBalance(ctx context.Context, adminID, targetTelegramID models.TelegramID, amount decimal.Decimal) error
	BanUser(ctx context.Context, adminID, targetTelegramID models.TelegramID) error
	UnbanUser(ctx context.Context, adminID, targetTelegramID models.TelegramID) error

	// MakeAdmin — только root-admin, кредит не возвращает.
	MakeAdmin(ctx context.Context, adminID, targetTelegramID models.TelegramID) error

	// RevokeAdmin: ErrNotAdmin — цель не админ, ErrCannotRevokeRootAdmin — цель root.
	RevokeAdmin(ctx context.Context, adminID, targetTelegramID models.TelegramID) error

	// SetReferralsEnabled — вкл/выкл начисления targetTelegramID как рефереру.
	SetReferralsEnabled(ctx context.Context, adminID, targetTelegramID models.TelegramID, enabled bool) error

	// CRUD товаров — categoryID может быть nil (без категории)
	CreateProduct(ctx context.Context, adminID models.TelegramID, categoryID *models.CategoryID, name, description string, price models.Money) (*models.Product, error)
	UpdateProduct(ctx context.Context, adminID models.TelegramID, productID models.ProductID, categoryID *models.CategoryID, name, description string, price models.Money, isActive bool) (*models.Product, error)
	DeleteProduct(ctx context.Context, adminID models.TelegramID, productID models.ProductID) error
	AddProductItems(ctx context.Context, adminID models.TelegramID, productID models.ProductID, contents []string) error

	// CRUD категорий — parentID nil = верхний уровень
	CreateCategory(ctx context.Context, adminID models.TelegramID, parentID *models.CategoryID, name, description string) (*models.Category, error)
	UpdateCategory(ctx context.Context, adminID models.TelegramID, categoryID models.CategoryID, name, description string, parentID *models.CategoryID) (*models.Category, error)
	DeleteCategory(ctx context.Context, adminID models.TelegramID, categoryID models.CategoryID) error

	// UpdateSettings перезаписывает единственную строку настроек бота
	// (models.SettingsID) целиком — Username поддержки и конфиг каждого мерчанта.
	UpdateSettings(ctx context.Context, adminID models.TelegramID, settings *models.Settings) (*models.Settings, error)

	GetLogs(ctx context.Context, adminID models.TelegramID, offset, limit int) ([]models.AdminLog, error)

	// ListLogs/CountLogs — журнал по всем админам, adminID nil = без фильтра.
	ListLogs(ctx context.Context, adminID *models.TelegramID, offset, limit int) ([]models.AdminLog, error)
	CountLogs(ctx context.Context, adminID *models.TelegramID) (int64, error)
}
