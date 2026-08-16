package models

// SettingsID — единственная строка настроек бота, ключ фиксирован.
const SettingsID int64 = 1

// CrystalPaySettings — учётные данные кассы CrystalPay: Login/Secret — для
// запросов к API, Salt — отдельный секрет только для подписи вебхуков.
type CrystalPaySettings struct {
	Enabled   bool    `json:"enabled"`
	Login     string  `json:"login"`
	Secret    string  `json:"secret"`
	Salt      string  `json:"salt"`
	MinAmount float64 `json:"min_amount"`
	MaxAmount float64 `json:"max_amount"`
}

// YooKassaSettings — ShopID + SecretKey, Basic Auth для API ЮKassa.
type YooKassaSettings struct {
	Enabled   bool    `json:"enabled"`
	ShopID    string  `json:"shop_id"`
	SecretKey string  `json:"secret_key"`
	MinAmount float64 `json:"min_amount"`
	MaxAmount float64 `json:"max_amount"`
}

// TinkoffSettings — TerminalKey + Password для подписи запросов Tinkoff Acquiring.
type TinkoffSettings struct {
	Enabled     bool    `json:"enabled"`
	TerminalKey string  `json:"terminal_key"`
	Password    string  `json:"password"`
	MinAmount   float64 `json:"min_amount"`
	MaxAmount   float64 `json:"max_amount"`
}

// ReferralSettings — Percent начисляется рефереру с каждой покупки его
// реферала; Enabled=false выключает программу целиком, независимо от Percent.
type ReferralSettings struct {
	Enabled bool `json:"enabled"`
	Percent int  `json:"percent"`
}

// Settings — общие настройки бота, редактируются через веб-панель.
type Settings struct {
	ID              int64  `gorm:"primaryKey" json:"id"`
	SupportUsername string `gorm:"size:64" json:"support_username"`

	CrystalPay CrystalPaySettings `gorm:"embedded;embeddedPrefix:crystalpay_" json:"crystalpay"`
	YooKassa   YooKassaSettings   `gorm:"embedded;embeddedPrefix:yookassa_" json:"yookassa"`
	Tinkoff    TinkoffSettings    `gorm:"embedded;embeddedPrefix:tinkoff_" json:"tinkoff"`
	Referral   ReferralSettings   `gorm:"embedded;embeddedPrefix:referral_" json:"referral"`
}
