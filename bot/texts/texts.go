// Package texts — все пользовательские строки бота. Сообщения, отправляемые
// с ParseMode=MarkdownV2, содержат literal MarkdownV2-спецсимволы (., !, (, ),
// |) уже экранированными в статической части — динамические значения
// экранирует вызывающий код через bot/utils.EscapeMarkdown/FormatAmount/
// FormatDate/EscapeMarkdownCode перед подстановкой. Сообщения без ParseMode
// (см. комментарий у каждой группы) экранировать не нужно и нельзя — в
// plain-text режиме обратный слэш показывается пользователю буквально.
package texts

var (
	CatalogBtn        = "📦 Каталог"
	ProfileBtn        = "👤 Профиль"
	HelpBtn           = "☎️ Помощь"
	PurchasesBtn      = "📜 Мои покупки"
	ReplenishmentsBtn = "💳 Мои пополнения"
	RefillBalanceBtn  = "💰 Пополнить баланс"
	StartMenuBtn      = "↩️ На главную"
	AdminPanelBtn     = "♿️ Админ панель"

	CrystalPayBtn = "💎 CrystalPay"
	YooKassaBtn   = "🟣 ЮKassa"
	TinkoffBtn    = "🟡 Тинькофф"
	ReferralName  = "🤝 Реферальная программа"
	ReferralBtn   = "👫 Реферальная программа"
	CloseBtn      = "❌ Закрыть"
	ShareBtn      = "📤 Поделиться"

	PurchaseInlineBtn = "%.2f₽ | %d x %s"
	BuyBtn            = "🛒 Купить"
	BackBtn           = "⬅️ Назад"
	MainMenuInlineBtn = "↩️ В главное меню"
	CatalogRootBtn    = "↩️ В корень каталога"
	CancelBtn         = "❌ Отмена"
	ConfirmBtn        = "✅ Подтвердить"
	PrevPageBtn       = "⬅️"
	NextPageBtn       = "➡️"
	ProfileRefreshBtn = "🔄 Обновить"

	// ParseMode=MarkdownV2. %s — FirstName, экранировать EscapeMarkdown.
	StartMsg = "🤙 Привет, *%s*\\!"
	// ParseMode=MarkdownV2. %s — Username (EscapeMarkdown), %d — telegram_id
	// (сырое число), %s — Balance (FormatAmount), %d — count, %s — Spent (FormatAmount).
	ProfileMsg = "👤 *@%s*\nℹ️ `%d`\n\n💰 Баланс: *%s* ₽\n📝 Покупок: *%d*\n💵 Потрачено: *%s* ₽"
	// ParseMode=MarkdownV2. %s — SupportUsername, экранировать EscapeMarkdown.
	HelpMsg      = "☎️ Саппорт *%s*"
	PurchasesMsg = "🛒 Последние покупки:"
	// ParseMode=MarkdownV2. %s — Name (EscapeMarkdown), %s — Price (FormatAmount),
	// %s — StockIndicator (свой фиксированный emoji, без экранирования), %d — count,
	// %s — Description (EscapeMarkdown), %s — Balance (FormatAmount).
	ProductMsg = "📦 *%s*\n💰 %s ₽\n%s Доступно: *%d*\n\n%s\n\n💰 Доступный баланс: %s ₽"
	CatalogMsg = "*📦 Каталог товаров*"
	// ParseMode=MarkdownV2. %s — Name (EscapeMarkdown), %s — Description
	// (EscapeMarkdown), %s — Balance (FormatAmount).
	CategoryMsg     = "📦 Категория: *%s*\n%s\n\n💰 Доступный баланс: %s ₽"
	CatalogEmptyMsg = "_📦 Пока здесь пусто_"
	// ParseMode=MarkdownV2. %d — count, %s — Description (EscapeMarkdown),
	// %s — уже собранные строки `content` (EscapeMarkdownCode на каждый Content
	// до оборачивания в бэктики, см. вызывающий код).
	ProductBoughtMsg = "✅ Покупка совершена\\! Куплено: *%d* шт\\.\n\nℹ️ %s\n\n*📦 Товар\\(ы\\):*\n%s"
	// ParseMode=MarkdownV2. %s — Name (EscapeMarkdown), %s — Amount (FormatAmount),
	// %d — count, %s — дата (FormatDate), %s — Description (EscapeMarkdown),
	// %s — уже собранные строки `content` (EscapeMarkdownCode).
	PurchaseDetailMsg = "📦 *%s*\n💰 %s ₽ \\(%d шт\\.\\)\n🗓%s\nℹ️ %s\n\n*🛒 Товар\\(ы\\):*\n%s"
	PurchasesEmptyMsg = "📜 У вас пока нет покупок"
	// Без ParseMode.
	RefillMsg = "💰 Пополнение баланса сейчас недоступно"

	ReplenishmentsMsg      = "💳 Последние пополнения:"
	ReplenishmentsEmptyMsg = "💳 У вас пока нет пополнений"
	// ParseMode=MarkdownV2. %s — Amount (FormatAmount), %s — MerchantName,
	// %s — ReplenishmentStatusName (оба — свои фиксированные строки без
	// спецсимволов, экранировать не нужно), %s — дата (FormatDate).
	ReplenishmentLineMsg = "%s ₽ \\| %s \\| %s \\| %s"

	ReplenishmentStatusPending   = "⏳ В обработке"
	ReplenishmentStatusPaid      = "✅ Оплачено"
	ReplenishmentStatusFailed    = "❌ Не удалось"
	ReplenishmentStatusCancelled = "🚫 Отменено"

	// Без ParseMode.
	RefillMerchantPickerMsg = "💰 Выберите способ пополнения:"

	// Без ParseMode (ссылка с username бота может содержать "_" — см.
	// историю бага в ReferralHandler). %d — процент, %s — ссылка, %d —
	// приглашено чел., %.2f — начислено ₽.
	ReferralMsg = "👫 Реферальная программа\n\nС каждой покупки реферала начисляется %d%%. Просто отправь ссылку друзьям 👇🏽\n\n🔗 Ссылка для приглашения — %s\n\n👥 Всего приглашено: %d чел.\n💵 Всего начислено: %.2f ₽"
	// ParseMode=MarkdownV2. %s — Amount (FormatAmount).
	ReferralCreditMsg      = "🎉 Ваш реферал совершил покупку\\!\n💵 Начислено: *%s* ₽"
	ReferralUnavailableMsg = "👫 Реферальная программа сейчас недоступна"

	// ParseMode=MarkdownV2. %s — код (внутренний, только цифры; EscapeMarkdownCode
	// применяется defensively).
	AdminMsg = "🔐 Панель администратора:\n\n🔑 Код для входа: `%s`\n⏳ Действителен 30 секунд"
	// ParseMode=MarkdownV2. %s — код (EscapeMarkdownCode), %s — FrontendURL (EscapeMarkdown).
	AdminMsgWithLink = "🔐 Панель администратора:\n\n🔑 Код для входа: `%s`\n⏳ Действителен 30 секунд\n\n🔗 Ссылка: %s"
	// Без ParseMode.
	AdminCodeErrMsg = "❌ Не удалось выпустить код для входа, попробуйте ещё раз."
	NotAdminMsg     = "⛔️ Команда доступна только администраторам."

	// Без ParseMode.
	PleaseStartMsg = "👋 Сначала нажмите /start"
	BannedMsg      = "🚫 Вы заблокированы."

	// ParseMode=MarkdownV2. %s — Name (EscapeMarkdown).
	AskQuantityMsg = "🔢 Сколько штук «%s» купить?\n\n_Введите число или выберите на клавиатуре_"
	// ParseMode=MarkdownV2, статический текст.
	InsufficientStockMsg = "❌ В наличии только *%d* шт\\. Введите другое количество\\."
	// ParseMode=MarkdownV2. %s — Name (EscapeMarkdown), %d — qty, %s — сумма (FormatAmount).
	ConfirmPurchaseMsg = "🧾 Подтвердите покупку:\n\n📦 *%s*\n🔢 Количество: *%d*\n💰 Сумма: *%s ₽*"
	// ParseMode=MarkdownV2, статический текст.
	InvalidQuantityMsg = "⚠️ Введите целое число от 1 до 20\\."
	// Без ParseMode.
	AskRefillAmountMsg = "💰 Введите сумму пополнения (%s):"
	// ParseMode=MarkdownV2, статический текст.
	InvalidAmountMsg = "⚠️ Введите положительное число\\."
	// ParseMode=MarkdownV2. %s — сумма (FormatAmount).
	RefillInvoiceMsg = "💳 Оплатите *%s ₽* по кнопке ниже\\."
	PayBtn           = "💳 Оплатить"
	// Без ParseMode.
	ErrInsufficientBalanceMsg = "❌ Не хватает баланса на покупку."
	ErrOutOfStockMsg          = "❌ Товар закончился."
	ErrProductInactiveMsg     = "❌ Товар больше не продаётся."
	ErrTooManyProductsMsg     = "❌ Слишком большое количество за раз (максимум 20)."
	ErrGenericMsg             = "❌ Что-то пошло не так, попробуйте позже."
)
