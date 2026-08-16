// Command migrate — одноразовый прогон: схема (AutoMigrate), два разовых
// клинапа старых колонок и бутстрап root-admin. Отдельный от cmd/bot и
// cmd/backend бинарник, чтобы два долгоживущих сервиса не гоняли миграцию
// параллельно — docker-compose ждёт его завершения перед стартом остальных.
package main

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/config"
	"github.com/trottling/Telegram-Store/internal/logger"
	pgdb "github.com/trottling/Telegram-Store/internal/repository/postgres"
)

func main() {
	ctx := context.Background()

	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	log := logger.New(cfg.Logger)
	log.Info("migrate: config loaded")

	db, err := pgdb.NewClient(cfg.Postgres)
	if err != nil {
		log.Fatalf("migrate: failed to connect to postgres: %v", err)
	}

	if err = pgdb.AutoMigrate(db); err != nil {
		log.Fatalf("migrate: failed to run migrations: %v", err)
	}
	log.Info("migrate: schema migrated")

	if err = pgdb.BackfillUserRoles(ctx, db, log); err != nil {
		log.Fatalf("migrate: failed to backfill user roles: %v", err)
	}

	if err = pgdb.DropLegacyAdminTokenColumn(db, log); err != nil {
		log.Fatalf("migrate: failed to drop legacy admin_token_hash column: %v", err)
	}

	userRepo := pgdb.NewUserRepo(db, log)
	if err = userRepo.EnsureRootAdminExists(ctx, cfg.Telegram.RootAdminID); err != nil {
		log.Fatalf("migrate: failed to ensure root admin exists: %v", err)
	}
	log.Info("migrate: root admin ensured, done")
}
