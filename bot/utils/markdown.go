package utils

import (
	"strings"
	"time"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// mdV2Escaper экранирует спецсимволы MarkdownV2 вне сущностей:
// https://core.telegram.org/bots/api#markdownv2-style
var mdV2Escaper = strings.NewReplacer(
	"\\", "\\\\",
	"_", "\\_",
	"*", "\\*",
	"[", "\\[",
	"]", "\\]",
	"(", "\\(",
	")", "\\)",
	"~", "\\~",
	"`", "\\`",
	">", "\\>",
	"#", "\\#",
	"+", "\\+",
	"-", "\\-",
	"=", "\\=",
	"|", "\\|",
	"{", "\\{",
	"}", "\\}",
	".", "\\.",
	"!", "\\!",
)

// EscapeMarkdown экранирует произвольный текст для безопасной подстановки в
// сообщение с ParseMode=MarkdownV2 (models.ParseModeMarkdown) — обязательно
// для любых динамических значений (username, названия, описания, ссылки,
// отформатированные даты/суммы), включая те, что стоят внутри *bold*/_italic_ —
// MarkdownV2 поддерживает вложенные сущности, так что непарные спецсимволы
// внутри них тоже ломают парсинг.
func EscapeMarkdown(s string) string {
	return mdV2Escaper.Replace(s)
}

// mdV2CodeEscaper — внутри `code`-сущности экранировать нужно только сам
// backtick и обратный слэш, остальные спецсимволы не мешают парсингу.
var mdV2CodeEscaper = strings.NewReplacer(
	"\\", "\\\\",
	"`", "\\`",
)

// EscapeMarkdownCode экранирует текст перед оборачиванием в `...` (inline code).
func EscapeMarkdownCode(s string) string {
	return mdV2CodeEscaper.Replace(s)
}

// FormatAmount форматирует сумму до сотых с экранированной точкой — вне
// сущности литеральная "." ломает парсинг MarkdownV2 так же, как и любой
// другой спецсимвол.
func FormatAmount(v models.Money) string {
	return EscapeMarkdown(v.String())
}

// FormatDate — дата/время в привычном виде, с экранированными точками под MarkdownV2.
func FormatDate(t time.Time) string {
	return EscapeMarkdown(t.Format("02.01.2006 15:04"))
}
