package postgres

import (
	"context"
	"fmt"

	"github.com/trottling/Telegram-Store/internal/domain/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// partialIndexes — то, что не выражается тегами GORM.
//
// Непроданных единиц у товара всегда мало, а проданные копятся и не удаляются.
// Обычный индекс по product_id со временем заставляет пробегать все проданные
// строки, чтобы отфильтровать их по is_sold; частичный содержит только
// непроданные и остаётся маленьким независимо от оборота. Задействован в самых
// горячих местах: ReserveItem (денежный путь под локами), проверка наличия в
// листинге товаров, рекурсивный CTE видимости категорий, счётчик на карточке.
//
// IF NOT EXISTS — cmd/migrate прогоняется на каждый деплой.
var partialIndexes = []string{
	`CREATE INDEX IF NOT EXISTS idx_product_items_unsold
	 ON product_items (product_id) WHERE is_sold = false`,
}

func createPartialIndexes(db *gorm.DB) error {
	for _, stmt := range partialIndexes {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

// AutoMigrate — единственная точка входа для cmd/migrate: схема (DDL), бутстрап root-admin + дефолтных настроек.
// cmd/migrate/main.go - тонкая обвязка
func AutoMigrate(ctx context.Context, db *gorm.DB, log *zap.SugaredLogger, rootAdminID int64) error {
	if err := db.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Product{},
		&models.ProductItem{},
		&models.Purchase{},
		&models.AdminLog{},
		&models.Settings{},
		&models.Replenishment{},
	); err != nil {
		return err
	}
	log.Info("postgres: schema migrated")

	if err := createPartialIndexes(db); err != nil {
		return fmt.Errorf("create partial indexes: %w", err)
	}
	log.Info("postgres: partial indexes ensured")

	userRepo := NewUserRepo(db, log)
	if err := userRepo.EnsureRootAdminExists(ctx, rootAdminID); err != nil {
		return fmt.Errorf("ensure root admin exists: %w", err)
	}
	log.Info("postgres: root admin ensured")

	settingsRepo := NewSettingsRepo(db, log)
	if err := settingsRepo.EnsureExists(ctx, &models.Settings{SupportUsername: "@#####"}); err != nil {
		return fmt.Errorf("ensure settings exist: %w", err)
	}
	log.Info("postgres: settings ensured")

	return nil
}
