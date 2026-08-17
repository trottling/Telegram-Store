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
