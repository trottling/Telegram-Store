package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Telegram   *TelegramConfig
	Postgres   *PostgresConfig
	Redis      *RedisConfig
	AdminPanel *AdminPanelConfig
	Logger     *LoggerConfig
}

type TelegramConfig struct {
	Token       string
	RootAdminID int64
}

type PostgresConfig struct {
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
}

type RedisConfig struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

type AdminPanelConfig struct {
	Port int
	// URL — адрес самого API (VITE_API_BASE_URL), не панели.
	URL string
	// FrontendURL — где отдаётся React-панель, на неё ссылается /admin.
	FrontendURL string
	// CORSOrigin — разрешённые origin'ы через запятую.
	CORSOrigin string
	// JWTSecret подписывает сессионные токены, обязателен.
	JWTSecret []byte
}

type LoggerConfig struct {
	Level string
}

func New() (*Config, error) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	rootAdminID, err := strconv.ParseInt(os.Getenv("TELEGRAM_ROOT_ADMIN_ID"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid TELEGRAM_ROOT_ADMIN_ID: %w", err)
	}

	pgHost := getEnv("POSTGRES_HOST", "localhost")
	pgPort, _ := strconv.Atoi(os.Getenv("POSTGRES_PORT"))
	if pgPort == 0 {
		pgPort = 5432
	}
	pgUser := os.Getenv("POSTGRES_USER")
	pgPassword := os.Getenv("POSTGRES_PASSWORD")
	pgName := os.Getenv("POSTGRES_NAME")
	pgSSLMode := getEnv("POSTGRES_SSLMODE", "disable")

	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB, _ := strconv.Atoi(os.Getenv("REDIS_DB"))

	adminPanelPort, _ := strconv.Atoi(os.Getenv("ADMIN_PANEL_BACKEND_PORT"))
	if adminPanelPort == 0 {
		adminPanelPort = 8080
	}
	adminPanelURL := getEnv("ADMIN_PANEL_BACKEND_URL", "http://localhost:8080")
	adminPanelFrontendURL := getEnv("ADMIN_PANEL_FRONTEND_URL", "http://localhost:3000")
	// localhost и 127.0.0.1 разрешены оба — браузер считает их разными origin.
	adminPanelCORSOrigin := getEnv("ADMIN_PANEL_CORS_ORIGIN", "http://localhost:3000,http://127.0.0.1:3000")
	adminJWTSecret := os.Getenv("ADMIN_JWT_SECRET")

	logLevel := getEnv("LOG_LEVEL", "info")

	cfg := &Config{
		Telegram: &TelegramConfig{
			Token:       botToken,
			RootAdminID: rootAdminID,
		},

		Postgres: &PostgresConfig{
			DBHost:     pgHost,
			DBPort:     pgPort,
			DBUser:     pgUser,
			DBPassword: pgPassword,
			DBName:     pgName,
			DBSSLMode:  pgSSLMode,
		},

		Redis: &RedisConfig{
			RedisAddr:     redisAddr,
			RedisPassword: redisPassword,
			RedisDB:       redisDB,
		},

		AdminPanel: &AdminPanelConfig{
			Port:        adminPanelPort,
			URL:         adminPanelURL,
			FrontendURL: adminPanelFrontendURL,
			CORSOrigin:  adminPanelCORSOrigin,
			JWTSecret:   []byte(adminJWTSecret),
		},

		Logger: &LoggerConfig{
			Level: logLevel,
		},
	}

	if cfg.Telegram.Token == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}
	if cfg.Postgres.DBUser == "" || cfg.Postgres.DBPassword == "" || cfg.Postgres.DBName == "" {
		return nil, fmt.Errorf("DB_USER, DB_PASSWORD and DB_NAME are required")
	}
	if len(cfg.AdminPanel.JWTSecret) == 0 {
		return nil, fmt.Errorf("ADMIN_JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
