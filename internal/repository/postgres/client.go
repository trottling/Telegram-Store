package postgres

import (
	"fmt"

	"github.com/trottling/Telegram-Store/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewClient(cfg *config.PostgresConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)

	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
