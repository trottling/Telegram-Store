package texts

import (
	"embed"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.uber.org/zap"
	"golang.org/x/text/language"
)

//go:embed locales/*.toml
var localeFS embed.FS

// Поддерживаемые языки бота — единственный источник правды: используется
// и здесь (Normalize), и в bot/keyboards (прогрев обеих раскладок), и в
// bot.go/fsm.go (распознавание нажатой кнопки на любом языке).
const (
	LangRU = "ru"
	LangEN = "en"
)

var SupportedLanguages = []string{LangRU, LangEN}

var supportedSet = map[string]bool{LangRU: true, LangEN: true}

var (
	bundle     = mustBuildBundle()
	localizers = buildLocalizers(bundle)
)

func mustBuildBundle() *i18n.Bundle {
	b := i18n.NewBundle(language.Russian) // ru — язык бандла по умолчанию/fallback
	b.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	for _, f := range []string{"locales/active.ru.toml", "locales/active.en.toml"} {
		if _, err := b.LoadMessageFileFS(localeFS, f); err != nil {
			// Паника на старте — как и остальные fail-fast инициализации в
			// проекте (AutoMigrate, fx.Provide): битый TOML не должен
			// молча деградировать в проде.
			panic("texts: failed to load " + f + ": " + err.Error())
		}
	}
	return b
}

func buildLocalizers(b *i18n.Bundle) map[string]*i18n.Localizer {
	m := make(map[string]*i18n.Localizer, len(SupportedLanguages))
	for _, lang := range SupportedLanguages {
		m[lang] = i18n.NewLocalizer(b, lang)
	}
	return m
}

// Normalize сводит произвольный код Telegram (BCP-47: "en-US", "uk", "") к
// одному из поддерживаемых языков, ru — если код не распознан или не покрыт.
func Normalize(code string) string {
	tag, err := language.Parse(code)
	if err == nil {
		if base, _ := tag.Base(); supportedSet[base.String()] {
			return base.String()
		}
	}
	return LangRU
}

// IsSupported — код уже нормализован вызывающим кодом (пришёл из БД/callback'а)?
func IsSupported(lang string) bool {
	return supportedSet[lang]
}

// T возвращает локализованную строку. lang нормализуется на каждый вызов —
// вызывающему коду не нужно заранее валидировать User.Language.
// Не паникует: отсутствующий messageID — программная ошибка (опечатка в ID
// или пропуск в TOML), логируется и отдаёт сам ID как видимый маркер, чтобы
// не ронять весь бот на одном сломанном экране (ловится в CI — см. texts_test.go).
func T(lang, messageID string, data map[string]any) string {
	loc := localizers[Normalize(lang)]
	s, err := loc.Localize(&i18n.LocalizeConfig{MessageID: messageID, TemplateData: data})
	if err != nil {
		zap.S().Errorw("texts: localize failed", "error", err, "message_id", messageID)
		return messageID
	}
	return s
}
