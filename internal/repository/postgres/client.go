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

	return gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLog})
}
