package config

import "testing"

// TestIsLoopbackURL — от этой проверки зависит единственное предупреждение о
// забытом PAYMENTS_BACKEND_URL (см. cmd/bot/providePaymentProviders). Ложное
// «нет, не loopback» означает, что прод молча не будет подтверждать платежи.
func TestIsLoopbackURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"http://localhost:8081", true},
		{"http://127.0.0.1:8081", true},
		{"http://127.1.2.3:8081", true},
		{"http://[::1]:8081", true},
		{"https://pay.example.com", false},
		{"https://pay.example.com:443/api/webhooks", false},
		{"http://192.168.1.10:8081", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			cfg := &PaymentsConfig{URL: tt.url}
			if got := cfg.IsLoopbackURL(); got != tt.want {
				t.Errorf("IsLoopbackURL(%q) = %v, ожидалось %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"полный URL", "https://pay.example.com", false},
		{"с портом и путём", "http://localhost:8081/api", false},
		{"без схемы", "pay.example.com", true},
		{"только схема", "https://", true},
		{"пустой", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateURL("TEST_URL", tt.url); (err != nil) != tt.wantErr {
				t.Errorf("validateURL(%q) = %v, ожидалась ошибка: %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	t.Run("не задано — дефолт", func(t *testing.T) {
		got, err := getEnvInt("CONFIG_TEST_UNSET_PORT", 5432)
		if err != nil || got != 5432 {
			t.Errorf("получено (%d, %v), ожидалось (5432, nil)", got, err)
		}
	})

	t.Run("задано числом", func(t *testing.T) {
		t.Setenv("CONFIG_TEST_PORT", "6432")
		got, err := getEnvInt("CONFIG_TEST_PORT", 5432)
		if err != nil || got != 6432 {
			t.Errorf("получено (%d, %v), ожидалось (6432, nil)", got, err)
		}
	})

	// Раньше опечатка молча превращалась в дефолт, и caddy проксировал не туда.
	t.Run("мусор — ошибка, а не дефолт", func(t *testing.T) {
		t.Setenv("CONFIG_TEST_PORT", "8O80")
		if _, err := getEnvInt("CONFIG_TEST_PORT", 5432); err == nil {
			t.Error("ожидалась ошибка на нечисловое значение")
		}
	})
}

func TestValidatePort(t *testing.T) {
	valid := []int{1, 8080, 65535}
	for _, port := range valid {
		if err := validatePort("TEST_PORT", port); err != nil {
			t.Errorf("порт %d отвергнут: %v", port, err)
		}
	}

	invalid := []int{0, -1, 65536, 99999}
	for _, port := range invalid {
		if err := validatePort("TEST_PORT", port); err == nil {
			t.Errorf("порт %d принят, ожидался отказ", port)
		}
	}
}

// TestValidateOriginList — CORS сравнивает Origin посимвольно, поэтому лишний
// слеш или путь означают, что панель молча не пройдёт проверку.
func TestValidateOriginList(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"один origin", "https://admin.example.com", false},
		{"несколько через запятую", "http://localhost:3000,http://127.0.0.1:3000", false},
		{"пробелы вокруг запятой", "http://localhost:3000 , http://127.0.0.1:3000", false},
		{"слеш на конце", "http://localhost:3000/", true},
		{"с путём", "https://admin.example.com/panel", true},
		{"без схемы", "localhost:3000", true},
		{"один из двух битый", "http://localhost:3000,localhost:3001", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateOriginList("TEST_ORIGINS", tt.raw); (err != nil) != tt.wantErr {
				t.Errorf("validateOriginList(%q) = %v, ожидалась ошибка: %v", tt.raw, err, tt.wantErr)
			}
		})
	}
}

// TestValidateProxyList — список доверенных прокси это security-контроль:
// gin на кривом значении только вернёт ошибку, а сервис продолжит работать с
// недостоверным ClientIP.
func TestValidateProxyList(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"CIDR", "172.28.0.0/16", false},
		{"одиночный IP", "10.0.0.1", false},
		{"несколько", "172.28.0.0/16,10.0.0.1", false},
		{"IPv6 CIDR", "fd00::/8", false},
		{"мусор", "not-an-address", true},
		{"кривая маска", "172.28.0.0/99", true},
		{"один из двух битый", "172.28.0.0/16,мусор", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateProxyList("TEST_PROXIES", tt.raw); (err != nil) != tt.wantErr {
				t.Errorf("validateProxyList(%q) = %v, ожидалась ошибка: %v", tt.raw, err, tt.wantErr)
			}
		})
	}
}
