// Command migrate — одноразовый прогон pgdb.AutoMigrate (схема, легаси-клинапы,
// бутстрап root-admin и дефолтных настроек). Отдельный от cmd/bot,
// cmd/admin_backend и cmd/payments_backend бинарник, чтобы долгоживущие
// сервисы не гоняли миграцию параллельно — docker-compose ждёт его
// завершения перед стартом остальных.
//
// Не долгоживущее приложение — в отличие от остальных cmd/*, тут нет
// fx.Lifecycle/Run(): fx.New строит граф и сразу же (синхронно, во время
// самого fx.New — Invoke-функции выполняются при построении графа, до
// какого-либо Start) выполняет fx.Invoke(runMigrate), после чего процесс
// просто завершается. app.Err() — ошибка сборки графа ИЛИ ошибка,
// возвращённая runMigrate; при ошибке дублируем её в stderr напрямую (не
// через настроенный логгер) — если сборка упала рано, самого логгера может
// ещё не быть, а уровень LOG_LEVEL не должен скрывать фатальный сбой миграции.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/trottling/Telegram-Store/internal/config"
	"github.com/trottling/Telegram-Store/internal/logger"
	pgdb "github.com/trottling/Telegram-Store/internal/repository/postgres"
)

func providePostgresConfig(cfg *config.Config) *config.PostgresConfig { return cfg.Postgres }
func provideLoggerConfig(cfg *config.Config) *config.LoggerConfig     { return cfg.Logger }

// migrateTimeout — потолок на всю миграцию. Без дедлайна AutoMigrate, упёршийся
// в блокировку на занятой таблице, висел бы вечно, а вместе с ним и весь стек:
// bot/admin_backend/payments_backend ждут этот контейнер через
// service_completed_successfully и без диагностики просто не стартуют.
const migrateTimeout = 5 * time.Minute

func runMigrate(cfg *config.Config, log *logrus.Logger, db *gorm.DB) error {
	log.Info("migrate: config loaded")

	ctx, cancel := context.WithTimeout(context.Background(), migrateTimeout)
	defer cancel()

	if err := pgdb.AutoMigrate(ctx, db, log, cfg.Telegram.RootAdminID); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	log.Info("migrate: done")
	return nil
}

func main() {
	app := fx.New(
		fx.WithLogger(logger.NewFxLogger),
		fx.Provide(
			config.New,
			logger.New,
			providePostgresConfig,
			provideLoggerConfig,
			pgdb.NewClient,
		),
		fx.Invoke(runMigrate),
	)

	if err := app.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: failed: %v\n", err)
		os.Exit(1)
	}
}
