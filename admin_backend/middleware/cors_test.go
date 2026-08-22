package middleware

import "testing"

// TestNormalizeOrigin — PUBLIC_PORT=443 (дефолт из .env.example) делает
// ADMIN_PANEL_CORS_ORIGIN буквально "https://domain:443", а браузер в
// заголовке Origin порт по умолчанию для схемы никогда не шлёт. Без этой
// нормализации конфиг и реальный запрос расходились бы навсегда, и CORS
// отваливался бы у всех в стандартной (443) конфигурации.
func TestNormalizeOrigin(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"https с портом по умолчанию — порт убирается", "https://example.com:443", "https://example.com"},
		{"http с портом по умолчанию — порт убирается", "http://example.com:80", "http://example.com"},
		{"https без порта — без изменений", "https://example.com", "https://example.com"},
		{"нестандартный порт остаётся значимым", "https://example.com:8443", "https://example.com:8443"},
		{"http-порт не убирается со схемой https", "https://example.com:80", "https://example.com:80"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeOrigin(tt.in); got != tt.want {
				t.Errorf("normalizeOrigin(%q) = %q, ожидалось %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestNormalizeOriginRoundTrip — прямая проверка того, что реально решает
// баг: значение из ADMIN_PANEL_CORS_ORIGIN с явным :443 и заголовок Origin,
// который в этом случае реально шлёт браузер (без порта), после нормализации
// совпадают — то есть карта allowed[normalizeOrigin(cfg)] содержит ключ,
// под которым его найдёт normalizeOrigin(origin) на каждый запрос.
func TestNormalizeOriginRoundTrip(t *testing.T) {
	configured := normalizeOrigin("https://example.com:443")
	fromBrowser := normalizeOrigin("https://example.com")
	if configured != fromBrowser {
		t.Errorf("configured=%q fromBrowser=%q — не совпадают, CORS отклонит легитимный запрос", configured, fromBrowser)
	}
}
