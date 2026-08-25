package keyboards

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/texts"
	"github.com/trottling/Telegram-Store/bot/utils"
	"github.com/trottling/Telegram-Store/internal/config"
	domain "github.com/trottling/Telegram-Store/internal/domain/models"
)

// Keyboards — клавиатуры, не зависящие от запроса, кроме языка пользователя.
// Обе языковые раскладки прогреваются один раз при старте (языков всего два,
// перестраивать на каждый реквест незачем), доступ — через методы-аксессоры
// ниже. AdminKb нужен URL панели из конфига. Остальные клавиатуры (зависящие
// от данных запроса — пагинация, каталог, количество) строятся функциями ниже.
type Keyboards struct {
	mainMenuKb map[string]*models.ReplyKeyboardMarkup
	profileKb  map[string]*models.ReplyKeyboardMarkup
	adminKb    map[string]*models.InlineKeyboardMarkup
	settingsKb map[string]*models.InlineKeyboardMarkup
}

func New(adminPanelConfig *config.AdminPanelConfig) *Keyboards {
	k := &Keyboards{
		mainMenuKb: make(map[string]*models.ReplyKeyboardMarkup, len(texts.SupportedLanguages)),
		profileKb:  make(map[string]*models.ReplyKeyboardMarkup, len(texts.SupportedLanguages)),
		adminKb:    make(map[string]*models.InlineKeyboardMarkup, len(texts.SupportedLanguages)),
		settingsKb: make(map[string]*models.InlineKeyboardMarkup, len(texts.SupportedLanguages)),
	}
	for _, lang := range texts.SupportedLanguages {
		k.mainMenuKb[lang] = buildMainMenuKb(lang)
		k.profileKb[lang] = buildProfileKb(lang)
		k.adminKb[lang] = buildAdminKb(lang, adminPanelConfig.FrontendURL, adminPanelConfig.TechDashboardUID, adminPanelConfig.BusinessDashboardUID)
		k.settingsKb[lang] = buildSettingsKb(lang)
	}
	return k
}

func buildMainMenuKb(lang string) *models.ReplyKeyboardMarkup {
	return &models.ReplyKeyboardMarkup{
		ResizeKeyboard: true,
		Keyboard: [][]models.KeyboardButton{
			{{Text: texts.T(lang, texts.HelpBtn, nil)}, {Text: texts.T(lang, texts.CatalogBtn, nil)}, {Text: texts.T(lang, texts.ProfileBtn, nil)}},
		},
	}
}

func buildProfileKb(lang string) *models.ReplyKeyboardMarkup {
	return &models.ReplyKeyboardMarkup{
		ResizeKeyboard: true,
		Keyboard: [][]models.KeyboardButton{
			{{Text: texts.T(lang, texts.PurchasesBtn, nil)}, {Text: texts.T(lang, texts.ReplenishmentsBtn, nil)}},
			{{Text: texts.T(lang, texts.RefillBalanceBtn, nil)}, {Text: texts.T(lang, texts.ReferralBtn, nil)}},
			{{Text: texts.T(lang, texts.ProfileRefreshBtn, nil)}, {Text: texts.T(lang, texts.StartMenuBtn, nil)}},
			{{Text: texts.T(lang, texts.SettingsBtn, nil)}},
		},
	}
}

// buildAdminKb — все три кнопки открываются как Telegram Mini App (web_app).
// Обе кнопки статистики ведут не прямо в Grafana, а на страницу обмена
// initData (см. statsStartURL) — иначе первый неаутентифицированный тап
// падал бы в Caddy'шный forward_auth и терял выбор дашборда, см. его комментарий.
// techDashboardUID/businessDashboardUID — из config.AdminPanelConfig
// (ADMIN_PANEL_TECH_DASHBOARD_UID/ADMIN_PANEL_BUSINESS_DASHBOARD_UID),
// провалидированы уже в config.New().
func buildAdminKb(lang, frontendURL, techDashboardUID, businessDashboardUID string) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: texts.T(lang, texts.AdminPanelBtn, nil), WebApp: &models.WebAppInfo{URL: frontendURL}}},
			{{Text: texts.T(lang, texts.TechStatsBtn, nil), WebApp: &models.WebAppInfo{URL: statsStartURL(frontendURL, techDashboardUID)}}},
			{{Text: texts.T(lang, texts.BusinessStatsBtn, nil), WebApp: &models.WebAppInfo{URL: statsStartURL(frontendURL, businessDashboardUID)}}},
		},
	}
}

// statsStartURL — ссылка на /start?to=stats&dashboard=<uid>, а не сразу на
// Grafana: StartPage.tsx сама меняет initData на сессию и только потом ведёт
// на конкретный дашборд
func statsStartURL(frontendURL, dashboardUID string) string {
	return strings.TrimRight(frontendURL, "/") + "/start?to=stats&dashboard=" + dashboardUID
}

// buildSettingsKb — инлайн-меню настроек (кнопка "⚙️ Настройки" в профиле).
// Один пункт сегодня, структура готова под будущие настройки — просто
// добавляется новая строка в InlineKeyboard.
func buildSettingsKb(lang string) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: texts.T(lang, texts.LanguageBtn, nil), CallbackData: utils.SettingsLanguageCallback}},
		},
	}
}

func (k *Keyboards) MainMenuKb(lang string) *models.ReplyKeyboardMarkup {
	return k.mainMenuKb[texts.Normalize(lang)]
}

func (k *Keyboards) ProfileKb(lang string) *models.ReplyKeyboardMarkup {
	return k.profileKb[texts.Normalize(lang)]
}

func (k *Keyboards) AdminKb(lang string) *models.InlineKeyboardMarkup {
	return k.adminKb[texts.Normalize(lang)]
}

func (k *Keyboards) SettingsKb(lang string) *models.InlineKeyboardMarkup {
	return k.settingsKb[texts.Normalize(lang)]
}

// BuildPurchasesKb рисует страницу истории покупок плюс кнопки вперёд/назад.
func BuildPurchasesKb(lang string, batches []domain.PurchaseBatchSummary, offset, limit int, total int64) *models.InlineKeyboardMarkup {
	// nil-слайс уходит в JSON как null, а Bot API требует массив — начинаем с пустого, но не nil.
	rows := make([][]models.InlineKeyboardButton, 0)
	for _, batch := range batches {
		rows = append(rows, []models.InlineKeyboardButton{{
			Text: texts.T(lang, texts.PurchaseInlineBtn, map[string]any{
				"Amount":      fmt.Sprintf("%.2f", batch.TotalAmount), // подпись кнопки, не MarkdownV2 — без FormatAmount
				"Quantity":    batch.Quantity,
				"ProductName": batch.ProductName,
			}),
			CallbackData: utils.BuildPurchaseBatchCallback(offset, batch.BatchID),
		}})
	}

	var navRow []models.InlineKeyboardButton
	if offset > 0 {
		prevOffset := max(offset-limit, 0)
		navRow = append(navRow, models.InlineKeyboardButton{Text: texts.T(lang, texts.PrevPageBtn, nil), CallbackData: utils.BuildPurchasesPageCallback(prevOffset)})
	}
	if int64(offset+limit) < total {
		navRow = append(navRow, models.InlineKeyboardButton{Text: texts.T(lang, texts.NextPageBtn, nil), CallbackData: utils.BuildPurchasesPageCallback(offset + limit)})
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// BuildReplenishmentsKb — по кнопке на пополнение (эмодзи статуса + сумма,
// см. ReplenishmentInlineBtn/ReplenishmentStatusEmoji), тот же паттерн, что
// у BuildPurchasesKb.
func BuildReplenishmentsKb(lang string, items []domain.Replenishment, offset, limit int, total int64) *models.InlineKeyboardMarkup {
	rows := make([][]models.InlineKeyboardButton, 0)
	for _, r := range items {
		rows = append(rows, []models.InlineKeyboardButton{{
			Text: texts.T(lang, texts.ReplenishmentInlineBtn, map[string]any{
				"Status": texts.ReplenishmentStatusEmoji(r.Status),
				"Amount": fmt.Sprintf("%.2f", r.Amount), // подпись кнопки, не MarkdownV2 — без FormatAmount
			}),
			CallbackData: utils.BuildReplenishmentDetailCallback(offset, r.ID),
		}})
	}

	var navRow []models.InlineKeyboardButton
	if offset > 0 {
		prevOffset := max(offset-limit, 0)
		navRow = append(navRow, models.InlineKeyboardButton{Text: texts.T(lang, texts.PrevPageBtn, nil), CallbackData: utils.BuildReplenishmentsPageCallback(prevOffset)})
	}
	if int64(offset+limit) < total {
		navRow = append(navRow, models.InlineKeyboardButton{Text: texts.T(lang, texts.NextPageBtn, nil), CallbackData: utils.BuildReplenishmentsPageCallback(offset + limit)})
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// BuildCatalogNavKb рисует экран дерева категорий: подкатегории, товары в
// сетке по 2 колонки, затем строка навигации (разная в корне и внутри).
func BuildCatalogNavKb(lang string, children []domain.Category, products []domain.Product, backCallbackData string) *models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton
	var currentRow []models.InlineKeyboardButton

	flush := func() {
		if len(currentRow) > 0 {
			rows = append(rows, currentRow)
			currentRow = nil
		}
	}

	for _, c := range children {
		currentRow = append(currentRow, models.InlineKeyboardButton{
			Text:         c.Name,
			CallbackData: utils.BuildCategoryCallback(c.ID),
		})
		if len(currentRow) == 2 {
			flush()
		}
	}
	flush()

	for _, p := range products {
		currentRow = append(currentRow, models.InlineKeyboardButton{
			Text:         p.Name,
			CallbackData: utils.BuildProductCallback(p.ID),
		})
		if len(currentRow) == 2 {
			flush()
		}
	}
	flush()

	var navRow []models.InlineKeyboardButton
	if backCallbackData != "" {
		navRow = append(navRow,
			models.InlineKeyboardButton{Text: texts.T(lang, texts.BackBtn, nil), CallbackData: backCallbackData},
			models.InlineKeyboardButton{Text: texts.T(lang, texts.CatalogRootBtn, nil), CallbackData: utils.CatalogRootCallback},
		)
	} else {
		navRow = append(navRow, models.InlineKeyboardButton{Text: texts.T(lang, texts.MainMenuInlineBtn, nil), CallbackData: utils.MainMenuCallback})
	}
	rows = append(rows, navRow)

	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// BuildQuantityKb — быстрый выбор количества 1..max плюс кнопка отмены.
func BuildQuantityKb(lang string, max int) *models.InlineKeyboardMarkup {
	row := make([]models.InlineKeyboardButton, 0, max)
	for i := 1; i <= max; i++ {
		row = append(row, models.InlineKeyboardButton{
			Text:         strconv.Itoa(i),
			CallbackData: utils.BuildBuyQtyCallback(i),
		})
	}
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			row,
			{{Text: texts.T(lang, texts.CancelBtn, nil), CallbackData: utils.BuyCancelCallback}},
		},
	}
}

// BuildBuyConfirmKb — кнопки подтвердить/отменить перед списанием.
func BuildBuyConfirmKb(lang string) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: texts.T(lang, texts.ConfirmBtn, nil), CallbackData: utils.BuyConfirmCallback},
				{Text: texts.T(lang, texts.CancelBtn, nil), CallbackData: utils.BuyCancelCallback},
			},
		},
	}
}
