package postgres

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"gorm.io/gorm"
)

// AutoMigrate — единственная точка входа для cmd/migrate: схема (DDL), бутстрап root-admin + дефолтных настроек.
// cmd/migrate/main.go - тонкая обвязка
func AutoMigrate(ctx context.Context, db *gorm.DB, log *logrus.Logger, rootAdminID int64) error {
	if err := db.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Product{},
		&models.ProductItem{},
		&models.Purchase{},
		&models.AdminLog{},
		&models.Settings{},
	); err != nil {
		return err
	}
	log.Info("postgres: schema migrated")

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
