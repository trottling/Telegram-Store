package postgres

import (
	"context"
	"fmt"

	"github.com/trottling/Telegram-Store/internal/domain/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// refreshCollationVersions — постоянный шум в логах, а не признак реальной
// проблемы: postgres:*-alpine собран на musl, а он вообще не версионирует
// collation. Postgres видит расхождение с версией, записанной при создании
// базы, и предупреждает на каждое новое подключение ("no actual collation
// version, but a version was recorded") — штатный ALTER DATABASE ... REFRESH
// COLLATION VERSION тут даже падает ("invalid collation version change"),
// сравнивать нечего. Обнулить datcollversion — единственный способ снять
// предупреждение; поведение сортировки строк это не меняет ни на бит.
//
// Ошибку не считаем фатальной для всей миграции и просто предупреждаем:
// на managed Postgres (не наш bundled образ) роль из POSTGRES_USER может не
// быть суперпользователем — обновлять системный каталог pg_database тогда
// нечем, но это не должно останавливать реальную схему.
func refreshCollationVersions(db *gorm.DB, log *zap.SugaredLogger) {
	if err := db.Table("pg_database").
		Where("datcollversion IS NOT NULL").
		Update("datcollversion", gorm.Expr("NULL")).Error; err != nil {
		log.Warnw("postgres: failed to clear stale collation versions, the warning will keep appearing in logs", "error", err)
	}
}

// AutoMigrate — единственная точка входа для cmd/migrate: схема (DDL), бутстрап root-admin + дефолтных настроек.
// cmd/migrate/main.go - тонкая обвязка
func AutoMigrate(ctx context.Context, db *gorm.DB, log *zap.SugaredLogger, rootAdminID models.TelegramID, rootAdminUsername string) error {
	if err := db.AutoMigrate(
		&userRecord{},
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

	refreshCollationVersions(db, log)

	userRepo := NewUserRepo(db, log)
	if err := userRepo.EnsureRootAdminExists(ctx, rootAdminID, rootAdminUsername); err != nil {
		return fmt.Errorf("ensure root admin exists: %w", err)
	}
	log.Info("postgres: root admin ensured")

	settingsRepo := NewSettingsRepo(db, log)
	if err := settingsRepo.EnsureExists(ctx, &models.Settings{SupportUsername: "@#####", CatalogRefreshIntervalSeconds: 30}); err != nil {
		return fmt.Errorf("ensure settings exist: %w", err)
	}
	log.Info("postgres: settings ensured")

	return nil
}
