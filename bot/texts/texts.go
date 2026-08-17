// Package texts — все пользовательские строки бота. Значения переменных
// ниже — это message ID для bot/texts.T(lang, id, data), сам текст на каждом
// поддерживаемом языке лежит в locales/active.<lang>.toml (см. bundle.go).
// Сообщения, отправляемые с ParseMode=MarkdownV2, содержат literal
// MarkdownV2-спецсимволы (., !, (, ), |) уже экранированными в статической
// части TOML-строки — динамические значения экранирует вызывающий код через
// bot/utils.EscapeMarkdown/FormatAmount/FormatDate/EscapeMarkdownCode перед
// подстановкой в TemplateData. Сообщения без ParseMode (см. комментарий у
// каждой группы) экранировать не нужно и нельзя — в plain-text режиме
// обратный слэш показывается пользователю буквально.
package texts

const (
	CatalogBtn        = "CatalogBtn"
	ProfileBtn        = "ProfileBtn"
	HelpBtn           = "HelpBtn"
	PurchasesBtn      = "PurchasesBtn"
	ReplenishmentsBtn = "ReplenishmentsBtn"
	RefillBalanceBtn  = "RefillBalanceBtn"
	StartMenuBtn      = "StartMenuBtn"
	AdminPanelBtn     = "AdminPanelBtn"
	SettingsBtn       = "SettingsBtn"
	LanguageBtn       = "LanguageBtn"

	CrystalPayBtn = "CrystalPayBtn"
	YooKassaBtn   = "YooKassaBtn"
	TinkoffBtn    = "TinkoffBtn"
	ReferralName  = "ReferralName"
	ReferralBtn   = "ReferralBtn"
	CloseBtn      = "CloseBtn"
	ShareBtn      = "ShareBtn"

	// TemplateData: Amount (строка, plain fmt.Sprintf("%.2f", ...) — не
	// MarkdownV2, экранировать не нужно), Quantity (int), ProductName (string).
	PurchaseInlineBtn = "PurchaseInlineBtn"
	BuyBtn            = "BuyBtn"
	BackBtn           = "BackBtn"
	MainMenuInlineBtn = "MainMenuInlineBtn"
	CatalogRootBtn    = "CatalogRootBtn"
	CancelBtn         = "CancelBtn"
	ConfirmBtn        = "ConfirmBtn"
	PrevPageBtn       = "PrevPageBtn"
	NextPageBtn       = "NextPageBtn"
	ProfileRefreshBtn = "ProfileRefreshBtn"

	// ParseMode=MarkdownV2. TemplateData: Name — экранировать EscapeMarkdown.
	StartMsg = "StartMsg"
	// ParseMode=MarkdownV2. TemplateData: Username (EscapeMarkdown), TelegramID
	// (сырое число), Balance (FormatAmount), Count (int), Spent (FormatAmount).
	ProfileMsg = "ProfileMsg"
	// ParseMode=MarkdownV2. TemplateData: SupportUsername — экранировать EscapeMarkdown.
	HelpMsg      = "HelpMsg"
	PurchasesMsg = "PurchasesMsg"
	// ParseMode=MarkdownV2. TemplateData: Name (EscapeMarkdown), Price (FormatAmount),
	// StockIndicator (свой фиксированный emoji, без экранирования), Available (int),
	// Description (EscapeMarkdown), Balance (FormatAmount).
	ProductMsg = "ProductMsg"
	CatalogMsg = "CatalogMsg"
	// ParseMode=MarkdownV2. TemplateData: Name (EscapeMarkdown), Description
	// (EscapeMarkdown), Balance (FormatAmount).
	CategoryMsg     = "CategoryMsg"
	CatalogEmptyMsg = "CatalogEmptyMsg"
	// ParseMode=MarkdownV2. TemplateData: Count (int), Description (EscapeMarkdown),
	// Content — уже собранные строки (EscapeMarkdownCode на каждый Content до
	// оборачивания в бэктики, см. вызывающий код).
	ProductBoughtMsg = "ProductBoughtMsg"
	// ParseMode=MarkdownV2. TemplateData: Name (EscapeMarkdown), Amount (FormatAmount),
	// Count (int), Date (FormatDate), Description (EscapeMarkdown),
	// Content — уже собранные строки (EscapeMarkdownCode).
	PurchaseDetailMsg = "PurchaseDetailMsg"
	PurchasesEmptyMsg = "PurchasesEmptyMsg"
	// Без ParseMode.
	RefillMsg = "RefillMsg"

	ReplenishmentsMsg      = "ReplenishmentsMsg"
	ReplenishmentsEmptyMsg = "ReplenishmentsEmptyMsg"
	// ParseMode=MarkdownV2. TemplateData: Amount (FormatAmount), Merchant,
	// Status (оба — свои фиксированные строки без спецсимволов, экранировать
	// не нужно), Date (FormatDate).
	ReplenishmentLineMsg = "ReplenishmentLineMsg"

	ReplenishmentStatusPending   = "ReplenishmentStatusPending"
	ReplenishmentStatusPaid      = "ReplenishmentStatusPaid"
	ReplenishmentStatusFailed    = "ReplenishmentStatusFailed"
	ReplenishmentStatusCancelled = "ReplenishmentStatusCancelled"

	// Без ParseMode.
	RefillMerchantPickerMsg = "RefillMerchantPickerMsg"

	// Без ParseMode (ссылка с username бота может содержать "_" — см. историю
	// бага в ReferralHandler). TemplateData: Percent (int), Link (string),
	// Invited (int), Credited (строка, уже отформатированная "%.2f", без
	// экранирования — не MarkdownV2).
	ReferralMsg = "ReferralMsg"
	// ParseMode=MarkdownV2. TemplateData: Amount (FormatAmount).
	ReferralCreditMsg      = "ReferralCreditMsg"
	ReferralUnavailableMsg = "ReferralUnavailableMsg"

	// ParseMode=MarkdownV2. TemplateData: Code (внутренний, только цифры;
	// EscapeMarkdownCode применяется defensively).
	AdminMsg = "AdminMsg"
	// ParseMode=MarkdownV2. TemplateData: Code (EscapeMarkdownCode), URL (EscapeMarkdown).
	AdminMsgWithLink = "AdminMsgWithLink"
	// Без ParseMode.
	AdminCodeErrMsg = "AdminCodeErrMsg"
	NotAdminMsg     = "NotAdminMsg"

	// Без ParseMode.
	PleaseStartMsg = "PleaseStartMsg"
	BannedMsg      = "BannedMsg"

	// ParseMode=MarkdownV2. TemplateData: Name — экранировать EscapeMarkdown.
	AskQuantityMsg = "AskQuantityMsg"
	// ParseMode=MarkdownV2. TemplateData: Available (int).
	InsufficientStockMsg = "InsufficientStockMsg"
	// ParseMode=MarkdownV2. TemplateData: Name (EscapeMarkdown), Quantity (int), Amount (FormatAmount).
	ConfirmPurchaseMsg = "ConfirmPurchaseMsg"
	// ParseMode=MarkdownV2, статический текст.
	InvalidQuantityMsg = "InvalidQuantityMsg"
	// Без ParseMode. TemplateData: Hint (string, уже локализован через
	// AmountRange*Msg ниже — см. bot/handlers/refill.go:amountRangeHint).
	AskRefillAmountMsg = "AskRefillAmountMsg"
	// Без ParseMode. Куски подсказки amountRangeHint — какая из четырёх
	// собирается, зависит от того, заданы ли Min/Max в настройках мерчанта.
	AmountRangeBothMsg = "AmountRangeBothMsg" // TemplateData: Min, Max (строки, "%.2f")
	AmountRangeMinMsg  = "AmountRangeMinMsg"  // TemplateData: Min
	AmountRangeMaxMsg  = "AmountRangeMaxMsg"  // TemplateData: Max
	AmountRangeAnyMsg  = "AmountRangeAnyMsg"
	// ParseMode=MarkdownV2, статический текст.
	InvalidAmountMsg = "InvalidAmountMsg"
	// ParseMode=MarkdownV2. TemplateData: Amount (FormatAmount).
	RefillInvoiceMsg = "RefillInvoiceMsg"
	PayBtn           = "PayBtn"
	// Без ParseMode.
	ErrInsufficientBalanceMsg = "ErrInsufficientBalanceMsg"
	ErrOutOfStockMsg          = "ErrOutOfStockMsg"
	ErrProductInactiveMsg     = "ErrProductInactiveMsg"
	ErrTooManyProductsMsg     = "ErrTooManyProductsMsg"
	ErrGenericMsg             = "ErrGenericMsg"

	// ParseMode=MarkdownV2, статический текст.
	SettingsMsg = "SettingsMsg"
	// Без ParseMode.
	LanguagePickerMsg = "LanguagePickerMsg"
	LanguageSetMsg    = "LanguageSetMsg"
	// Названия языков в меню выбора — нативные эндонимы (как язык называет
	// себя сам), значение одинаковое в обоих TOML: пункт "🇷🇺 Русский" не
	// переводится на английский, даже если сейчас выбран английский.
	LanguageRUBtn = "LanguageRUBtn"
	LanguageENBtn = "LanguageENBtn"
)
