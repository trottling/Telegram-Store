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
	Payments   *PaymentsConfig
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
	// FrontendURL — где отдаётся React-панель, на неё ссылается /admin.
	FrontendURL string
	// CORSOrigin — разрешённые origin'ы через запятую.
	CORSOrigin string
	// TrustedProxies — CIDR/адреса через запятую, чьему X-Forwarded-For можно
	// верить. Наружу admin_backend виден только через caddy, поэтому список —
	// подсеть public-network. Без него gin доверяет заголовку от кого угодно,
	// и rate-limit по IP перестаёт что-либо значить.
	TrustedProxies string
	// JWTSecret подписывает сессионные токены, обязателен.
	JWTSecret []byte
}

// PaymentsConfig — конфиг payments_backend (вебхуки мерчантов).
type PaymentsConfig struct {
	Port int
	// URL — внешний адрес payments_backend, на него указывают webhook-колбэки
	// CrystalPay/Tinkoff (строится в cmd/bot/providers.go при создании провайдеров).
	URL string
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
	adminPanelFrontendURL := getEnv("ADMIN_PANEL_FRONTEND_URL", "http://localhost:3000")
	// localhost и 127.0.0.1 разрешены оба — браузер считает их разными origin.
	adminPanelCORSOrigin := getEnv("ADMIN_PANEL_CORS_ORIGIN", "http://localhost:3000,http://127.0.0.1:3000")
	// Дефолт — подсеть public-network из docker-compose.yml.
	adminPanelTrustedProxies := getEnv("ADMIN_PANEL_TRUSTED_PROXIES", "172.28.0.0/16")
	adminJWTSecret := os.Getenv("ADMIN_JWT_SECRET")

	paymentsPort, _ := strconv.Atoi(os.Getenv("PAYMENTS_BACKEND_PORT"))
	if paymentsPort == 0 {
		paymentsPort = 8081
	}
	paymentsURL := getEnv("PAYMENTS_BACKEND_URL", "http://localhost:8081")

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
			Port:           adminPanelPort,
			FrontendURL:    adminPanelFrontendURL,
			CORSOrigin:     adminPanelCORSOrigin,
			TrustedProxies: adminPanelTrustedProxies,
			JWTSecret:      []byte(adminJWTSecret),
		},

		Payments: &PaymentsConfig{
			Port: paymentsPort,
			URL:  paymentsURL,
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
