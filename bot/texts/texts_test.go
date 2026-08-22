package texts

import "testing"

// allMessageIDs — явный список всех message ID из texts.go. Поддерживается
// руками (Go не даёт перечислить var-идентификаторы пакета рефлексией) —
// цена простого map-based T() вместо генерации типизированных обёрток.
var allMessageIDs = []string{
	CatalogBtn, ProfileBtn, HelpBtn, PurchasesBtn, ReplenishmentsBtn, RefillBalanceBtn,
	StartMenuBtn, AdminPanelBtn, StatsBtn, SettingsBtn, LanguageBtn,
	CrystalPayBtn, YooKassaBtn, TinkoffBtn, DummyBtn, ReferralName, ReferralBtn, CloseBtn, ShareBtn,
	PurchaseInlineBtn, BuyBtn, BackBtn, MainMenuInlineBtn, CatalogRootBtn, CancelBtn,
	ConfirmBtn, PrevPageBtn, NextPageBtn, ProfileRefreshBtn,
	StartMsg, ProfileMsg, HelpMsg, PurchasesMsg, ProductMsg, CatalogMsg, CategoryMsg,
	CatalogEmptyMsg, ProductBoughtMsg, PurchaseDetailMsg, PurchasesEmptyMsg, RefillMsg,
	ReplenishmentsMsg, ReplenishmentsEmptyMsg, ReplenishmentLineMsg,
	ReplenishmentStatusPending, ReplenishmentStatusPaid, ReplenishmentStatusFailed, ReplenishmentStatusCancelled,
	RefillMerchantPickerMsg,
	ReferralMsg, ReferralCreditMsg, ReferralUnavailableMsg,
	AdminMsg, AdminPanelUnavailableMsg, NotAdminMsg,
	PleaseStartMsg, BannedMsg,
	AskQuantityMsg, InsufficientStockMsg, ConfirmPurchaseMsg, InvalidQuantityMsg,
	AskRefillAmountMsg, AmountRangeBothMsg, AmountRangeMinMsg, AmountRangeMaxMsg, AmountRangeAnyMsg,
	InvalidAmountMsg, RefillInvoiceMsg, PayBtn, CheckPaymentBtn,
	CheckPaymentPendingMsg, CheckPaymentPaidMsg, CheckPaymentFailedMsg, RefillCheckErrorMsg,
	ErrInsufficientBalanceMsg, ErrOutOfStockMsg, ErrProductInactiveMsg, ErrTooManyProductsMsg, ErrGenericMsg,
	SettingsMsg, LanguagePickerMsg, LanguageSetMsg, LanguageRUBtn, LanguageENBtn,
}

// dummyData — заглушки под все встречающиеся в шаблонах поля, чтобы
// text/template не падал на выполнении для сообщений с плейсхолдерами.
var dummyData = map[string]any{
	"Name": "x", "Username": "x", "TelegramID": 1, "Balance": "0.00", "Count": 1, "Spent": "0.00",
	"SupportUsername": "x", "Price": "0.00", "StockIndicator": "🟢", "Available": 1, "Description": "x",
	"Content": "x", "Amount": "0.00", "Date": "x", "Merchant": "x", "Status": "x",
	"Percent": 1, "Link": "x", "Invited": 1, "Credited": "0.00", "Code": "x", "URL": "x",
	"Quantity": 1, "Hint": "x", "ProductName": "x", "Min": "0.00", "Max": "0.00",
}

// TestAllMessagesTranslated гоняет каждый ID через оба языка и проверяет,
// что T() не откатывается на сам ID — иначе перевод для этого языка
// отсутствует в TOML (опечатка/пропуск), лучше поймать это в CI, чем в проде.
func TestAllMessagesTranslated(t *testing.T) {
	for _, id := range allMessageIDs {
		for _, lang := range SupportedLanguages {
			if got := T(lang, id, dummyData); got == id {
				t.Errorf("message %q missing translation for lang %q", id, lang)
			}
		}
	}
}
