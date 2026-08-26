package postgres

import (
	"fmt"
	"strings"
	"time"

	"github.com/trottling/Telegram-Store/internal/config"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// gormSlowQueryThreshold — с какой длительности GORM считает запрос медленным.
const gormSlowQueryThreshold = 200 * time.Millisecond

// Пул соединений: без явных лимитов database/sql держит их без ограничения
// сверху, и всплеск одновременно обрабатываемых update'ов бота (см.
// bot/middleware.Track) мог бы забрать себе куда больше своей доли
// max_connections самого Postgres (дефолт 100), которую делят ещё
// admin_backend и payments_backend — каждый со своим отдельным пулом (эта
// функция общая, но каждый процесс вызывает её один раз и получает свой
// *sql.DB). Цифры подобраны под целевой деплой — 1 CPU/2 GB VPS: суммарно
// три постоянных потребителя пула укладываются в 45 соединений, с запасом
// до дефолтного лимита сервера.
const (
	dbMaxOpenConns    = 15
	dbMaxIdleConns    = 5
	dbConnMaxLifetime = 30 * time.Minute
	dbConnMaxIdleTime = 5 * time.Minute
)

// gormLogWriter направляет вывод GORM в zap. Своим логгером GORM пишет в
// stderr через стандартный log: с ANSI-раскраской независимо от того, терминал
// там или pipe, и мимо LOG_LEVEL. В контейнере это была вторая струя логов
// чужого формата, попадавшая в те же json-file файлы.
type gormLogWriter struct{ log *zap.SugaredLogger }

// Printf получает уже собранную GORM строку. Уровень один — Warn: при
// gormlogger.Warn GORM отдаёт только медленные запросы и ошибки, рядовые
// запросы сюда не доходят.
func (w gormLogWriter) Printf(format string, args ...any) {
	w.log.Warnf(strings.TrimSpace(format), args...)
}

func NewClient(cfg *config.PostgresConfig, log *zap.SugaredLogger) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)

	gormLog := gormlogger.New(gormLogWriter{log: log}, gormlogger.Config{
		SlowThreshold: gormSlowQueryThreshold,
		LogLevel:      gormlogger.Warn,
		Colorful:      false,
	})

	// SkipDefaultTransaction: GORM иначе оборачивает любую запись (даже
	// однострочный Create/Update/Delete) в отдельную транзакцию — лишний
	// round-trip там, где атомарность и так не нужна (один statement в
	// Postgres атомарен сам по себе). Там, где реально нужна атомарность
	// нескольких запросов (ReserveItems+CreateBatch, Replenishment.Confirm),
	// код уже явно оборачивает их в Transactor.WithinTransaction — эта опция
	// его не отключает и не касается.
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLog, SkipDefaultTransaction: true})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(dbMaxOpenConns)
	sqlDB.SetMaxIdleConns(dbMaxIdleConns)
	sqlDB.SetConnMaxLifetime(dbConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(dbConnMaxIdleTime)

	return db, nil
}
