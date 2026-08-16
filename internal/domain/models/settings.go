package models

// SettingsID — единственная строка настроек бота, ключ фиксирован.
const SettingsID int64 = 1

// Settings — общие настройки бота, редактируются через веб-панель.
// Пока только Username поддержки (см. texts.HelpMsg), позже сюда добавятся другие поля.
type Settings struct {
	ID              int64  `gorm:"primaryKey" json:"id"`
	SupportUsername string `gorm:"size:64" json:"support_username"`
}
