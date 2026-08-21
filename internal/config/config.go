package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	// minJWTSecretLen — HS256, поэтому секрет короче 32 байт ослабляет подпись.
	minJWTSecretLen = 32
	// secretPlaceholder — значение из .env.example (JWT_SECRET, REDIS_USERNAME/
	// PASSWORD), его нельзя пускать в работу: оно лежит в git, то есть известно всем.
	secretPlaceholder = "change_me_generate_with_openssl_rand_base64_32"
)

type Config struct {
	Telegram   *TelegramConfig
	Postgres   *PostgresConfig
	Redis      *RedisConfig
	AdminPanel *AdminPanelConfig
	Payments   *PaymentsConfig
	Metrics    *MetricsConfig
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
	RedisAddr string
	// RedisUsername/RedisPassword — ACL-пользователь (Redis 6+), обязательны:
	// bundled redis-сервис в docker-compose отключает user default и пускает
	// только по этой паре (см. docker-compose.yml).
	RedisUsername string
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
	// BotToken нужен для работы с TMA и initData
	// Должен совпадать с TelegramConfig.Token
	BotToken string
	// MetricsPort — свой /metrics-сервер admin_backend (бизнес-метрики
	// админских действий + агрегаты для Grafana, см. internal/metrics/admin),
	// отдельно от собственного порта API (Port).
	MetricsPort int
}

// PaymentsConfig — конфиг payments_backend (вебхуки мерчантов).
type PaymentsConfig struct {
	Port int
	// URL — внешний адрес payments_backend, на него указывают webhook-колбэки
	// CrystalPay/Tinkoff (строится в cmd/bot/providers.go при создании провайдеров).
	URL string
	// MetricsPort — свой /metrics-сервер payments_backend (пополнения по
	// мерчантам, см. internal/metrics/payments), отдельно от порта вебхуков (Port).
	MetricsPort int
}

// MetricsConfig — конфиг /metrics-сервера бота (Prometheus, см. internal/metrics/bot).
type MetricsConfig struct {
	Port int
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
	pgPort, err := getEnvInt("POSTGRES_PORT", 5432)
	if err != nil {
		return nil, err
	}
	pgUser := os.Getenv("POSTGRES_USER")
	pgPassword := os.Getenv("POSTGRES_PASSWORD")
	pgName := os.Getenv("POSTGRES_NAME")
	pgSSLMode := getEnv("POSTGRES_SSLMODE", "disable")

	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisUsername := os.Getenv("REDIS_USERNAME")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB, err := getEnvInt("REDIS_DB", 0)
	if err != nil {
		return nil, err
	}

	adminPanelPort, err := getEnvInt("ADMIN_PANEL_BACKEND_PORT", 8080)
	if err != nil {
		return nil, err
	}
	adminPanelFrontendURL := getEnv("ADMIN_PANEL_FRONTEND_URL", "http://localhost:3000")
	// localhost и 127.0.0.1 разрешены оба — браузер считает их разными origin.
	adminPanelCORSOrigin := getEnv("ADMIN_PANEL_CORS_ORIGIN", "http://localhost:3000,http://127.0.0.1:3000")
	// Дефолт — подсеть public-network из docker-compose.yml.
	adminPanelTrustedProxies := getEnv("ADMIN_PANEL_TRUSTED_PROXIES", "172.28.0.0/16")
	adminJWTSecret := os.Getenv("ADMIN_JWT_SECRET")

	paymentsPort, err := getEnvInt("PAYMENTS_BACKEND_PORT", 8081)
	if err != nil {
		return nil, err
	}
	paymentsURL := getEnv("PAYMENTS_BACKEND_URL", "http://localhost:8081")
	paymentsMetricsPort, err := getEnvInt("PAYMENTS_METRICS_PORT", 9102)
	if err != nil {
		return nil, err
	}

	metricsPort, err := getEnvInt("BOT_METRICS_PORT", 9100)
	if err != nil {
		return nil, err
	}
	adminMetricsPort, err := getEnvInt("ADMIN_METRICS_PORT", 9101)
	if err != nil {
		return nil, err
	}

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
			RedisUsername: redisUsername,
			RedisPassword: redisPassword,
			RedisDB:       redisDB,
		},

		AdminPanel: &AdminPanelConfig{
			Port:           adminPanelPort,
			FrontendURL:    adminPanelFrontendURL,
			CORSOrigin:     adminPanelCORSOrigin,
			TrustedProxies: adminPanelTrustedProxies,
			JWTSecret:      []byte(adminJWTSecret),
			BotToken:       botToken,
			MetricsPort:    adminMetricsPort,
		},

		Payments: &PaymentsConfig{
			Port:        paymentsPort,
			URL:         paymentsURL,
			MetricsPort: paymentsMetricsPort,
		},

		Metrics: &MetricsConfig{
			Port: metricsPort,
		},

		Logger: &LoggerConfig{
			Level: logLevel,
		},
	}

	if cfg.Telegram.Token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	// Telegram ID всегда положительный. Ноль проходил проверку выше, а
	// cmd/migrate бутстрапил root-админа на несуществующего пользователя —
	// и выдать права дальше становилось некому.
	if cfg.Telegram.RootAdminID <= 0 {
		return nil, fmt.Errorf("TELEGRAM_ROOT_ADMIN_ID must be a positive Telegram user ID, got %d", cfg.Telegram.RootAdminID)
	}
	if cfg.Postgres.DBUser == "" || cfg.Postgres.DBPassword == "" || cfg.Postgres.DBName == "" {
		return nil, fmt.Errorf("POSTGRES_USER, POSTGRES_PASSWORD and POSTGRES_NAME are required")
	}
	// bundled redis-сервис в docker-compose поднимает ACL-пользователя из этих
	// двух переменных и отключает default (см. docker-compose.yml) — пустое
	// или дефолтное значение означает, что подключиться к нему нечем.
	if cfg.Redis.RedisUsername == "" || cfg.Redis.RedisPassword == "" {
		return nil, fmt.Errorf("REDIS_USERNAME and REDIS_PASSWORD are required")
	}
	if cfg.Redis.RedisUsername == secretPlaceholder || cfg.Redis.RedisPassword == secretPlaceholder {
		return nil, fmt.Errorf("REDIS_USERNAME/REDIS_PASSWORD are still the .env.example placeholder, generate real ones: openssl rand -base64 32")
	}
	if len(cfg.AdminPanel.JWTSecret) == 0 {
		return nil, fmt.Errorf("ADMIN_JWT_SECRET is required")
	}
	if adminJWTSecret == secretPlaceholder {
		return nil, fmt.Errorf("ADMIN_JWT_SECRET is still the .env.example placeholder, generate a real one: openssl rand -base64 32")
	}
	if len(cfg.AdminPanel.JWTSecret) < minJWTSecretLen {
		return nil, fmt.Errorf("ADMIN_JWT_SECRET must be at least %d characters, got %d", minJWTSecretLen, len(cfg.AdminPanel.JWTSecret))
	}

	ports := []struct {
		key   string
		value int
	}{
		{"POSTGRES_PORT", cfg.Postgres.DBPort},
		{"ADMIN_PANEL_BACKEND_PORT", cfg.AdminPanel.Port},
		{"PAYMENTS_BACKEND_PORT", cfg.Payments.Port},
		{"BOT_METRICS_PORT", cfg.Metrics.Port},
		{"ADMIN_METRICS_PORT", cfg.AdminPanel.MetricsPort},
		{"PAYMENTS_METRICS_PORT", cfg.Payments.MetricsPort},
	}
	for _, p := range ports {
		if err = validatePort(p.key, p.value); err != nil {
			return nil, err
		}
	}
	// Локально и в docker-compose.debug.yml все эти сервисы слушают один хост,
	// так что совпадение любых двух портов вылезало бы только в рантайме как
	// "address in use" — проверяем попарную уникальность сразу здесь.
	seenPorts := make(map[int]string, len(ports))
	for _, p := range ports {
		if other, ok := seenPorts[p.value]; ok {
			return nil, fmt.Errorf("%s and %s must differ, both are %d", other, p.key, p.value)
		}
		seenPorts[p.value] = p.key
	}
	// Верхнюю границу не проверяем: число баз задаётся конфигом самого Redis.
	if cfg.Redis.RedisDB < 0 {
		return nil, fmt.Errorf("invalid REDIS_DB: %d must not be negative", cfg.Redis.RedisDB)
	}

	if err = validateURL("PAYMENTS_BACKEND_URL", cfg.Payments.URL); err != nil {
		return nil, err
	}
	if err = validateURL("ADMIN_PANEL_FRONTEND_URL", cfg.AdminPanel.FrontendURL); err != nil {
		return nil, err
	}
	if err = validateOriginList("ADMIN_PANEL_CORS_ORIGIN", cfg.AdminPanel.CORSOrigin); err != nil {
		return nil, err
	}
	if err = validateProxyList("ADMIN_PANEL_TRUSTED_PROXIES", cfg.AdminPanel.TrustedProxies); err != nil {
		return nil, err
	}

	return cfg, nil
}

// getEnvInt — как getEnv, но для чисел и с ошибкой вместо молчаливого дефолта:
// опечатка в номере порта иначе тихо превращалась в 8080, а caddy проксировал
// в пустоту, и причину приходилось искать по 502.
func getEnvInt(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %q is not a number", key, raw)
	}
	return value, nil
}

// validateURL проверяет структуру: парсится, есть схема и хост. Забытую
// переменную так не поймать — в проде и в разработке она одна и та же, — но
// опечатку поймать можно. Про адрес, на который мерчант не сможет доставить
// вебхук, предупреждает уже cmd/bot (см. providePaymentProviders).
func validateURL(key, raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", key, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("invalid %s: %q must include scheme and host", key, raw)
	}
	return nil
}

// IsLoopbackURL — URL смотрит на localhost. Для PAYMENTS_BACKEND_URL это
// значит, что вебхуки мерчантов не придут: они стучатся из интернета.
func (c *PaymentsConfig) IsLoopbackURL() bool {
	parsed, err := url.Parse(c.URL)
	if err != nil {
		return false
	}

	host := parsed.Hostname()
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// validatePort — порт вне допустимого диапазона раньше доезжал до рантайма и
// падал там невнятной ошибкой драйвера или листенера.
func validatePort(key string, port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid %s: %d is out of range 1-65535", key, port)
	}
	return nil
}

// validateOriginList — middleware.CORS сравнивает Origin посимвольно, а браузер
// присылает его без пути и без слеша на конце. Поэтому "http://localhost:3000/"
// не совпадёт никогда: панель молча упрётся в CORS, притом что переменная
// выглядит правильно. Ловим здесь, а не по жалобе в консоли браузера.
func validateOriginList(key, raw string) error {
	for _, origin := range strings.Split(raw, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}

		parsed, err := url.Parse(origin)
		if err != nil {
			return fmt.Errorf("invalid %s: %q: %w", key, origin, err)
		}
		if parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("invalid %s: %q must include scheme and host", key, origin)
		}
		if parsed.Path != "" || parsed.RawQuery != "" {
			return fmt.Errorf("invalid %s: %q must be a bare origin, without path or trailing slash", key, origin)
		}
	}
	return nil
}

// validateProxyList — это security-контроль, а не косметика: на кривом списке
// gin.SetTrustedProxies лишь возвращает ошибку, которую router.go может только
// записать в лог, и сервис продолжает работать с недостоверным ClientIP —
// то есть с обходимым лимитом попыток входа. Падать сразу честнее.
func validateProxyList(key, raw string) error {
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if _, _, err := net.ParseCIDR(entry); err == nil {
			continue
		}
		if net.ParseIP(entry) != nil {
			continue
		}
		return fmt.Errorf("invalid %s: %q is neither an IP address nor a CIDR block", key, entry)
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
