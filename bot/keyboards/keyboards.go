package keyboards

import (
	"fmt"
	"strconv"

	"github.com/go-telegram/bot/models"
	"github.com/trottling/Telegram-Store/bot/texts"
	"github.com/trottling/Telegram-Store/bot/utils"
	"github.com/trottling/Telegram-Store/internal/config"
	domain "github.com/trottling/Telegram-Store/internal/domain/models"
)

// Keyboards — клавиатуры, не зависящие от запроса, собираются один раз при
// старте (AdminKb нужен URL панели из конфига). Остальные — функциями ниже.
type Keyboards struct {
	MainMenuKb *models.ReplyKeyboardMarkup
	ProfileKb  *models.ReplyKeyboardMarkup
	AdminKb    *models.InlineKeyboardMarkup
}

func New(adminPanelConfig *config.AdminPanelConfig) *Keyboards {
	return &Keyboards{
		MainMenuKb: &models.ReplyKeyboardMarkup{
			ResizeKeyboard: true,
			Keyboard: [][]models.KeyboardButton{
				{{Text: texts.HelpBtn}, {Text: texts.CatalogBtn}, {Text: texts.ProfileBtn}},
			},
		},
		ProfileKb: &models.ReplyKeyboardMarkup{
			ResizeKeyboard: true,
			Keyboard: [][]models.KeyboardButton{
				{{Text: texts.PurchasesBtn}, {Text: texts.RefillBalanceBtn}},
				{{Text: texts.StartMenuBtn}},
			},
		},
		AdminKb: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: texts.AdminPanelBtn, URL: adminPanelConfig.FrontendURL}},
			},
		},
	}
}

// BuildPurchasesKb рисует страницу истории покупок плюс кнопки вперёд/назад.
func BuildPurchasesKb(batches []domain.PurchaseBatchSummary, offset, limit int, total int64) *models.InlineKeyboardMarkup {
	// nil-слайс уходит в JSON как null, а Bot API требует массив — начинаем с пустого, но не nil.
	rows := make([][]models.InlineKeyboardButton, 0)
	for _, batch := range batches {
		rows = append(rows, []models.InlineKeyboardButton{{
			Text:         fmt.Sprintf(texts.PurchaseInlineBtn, batch.TotalAmount, batch.Quantity, batch.ProductName),
			CallbackData: utils.BuildPurchaseBatchCallback(batch.BatchID),
		}})
	}

	var navRow []models.InlineKeyboardButton
	if offset > 0 {
		prevOffset := max(offset-limit, 0)
		navRow = append(navRow, models.InlineKeyboardButton{Text: texts.PrevPageBtn, CallbackData: utils.BuildPurchasesPageCallback(prevOffset)})
	}
	if int64(offset+limit) < total {
		navRow = append(navRow, models.InlineKeyboardButton{Text: texts.NextPageBtn, CallbackData: utils.BuildPurchasesPageCallback(offset + limit)})
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// BuildCatalogNavKb рисует экран дерева категорий: подкатегории, товары в
// сетке по 2 колонки, затем строка навигации (разная в корне и внутри).
func BuildCatalogNavKb(children []domain.Category, products []domain.Product, backCallbackData string) *models.InlineKeyboardMarkup {
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
			models.InlineKeyboardButton{Text: texts.BackBtn, CallbackData: backCallbackData},
			models.InlineKeyboardButton{Text: texts.CatalogRootBtn, CallbackData: utils.CatalogRootCallback},
		)
	} else {
		navRow = append(navRow, models.InlineKeyboardButton{Text: texts.MainMenuInlineBtn, CallbackData: utils.MainMenuCallback})
	}
	rows = append(rows, navRow)

	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// BuildQuantityKb — быстрый выбор количества 1..max плюс кнопка отмены.
func BuildQuantityKb(max int) *models.InlineKeyboardMarkup {
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
			{{Text: texts.CancelBtn, CallbackData: utils.BuyCancelCallback}},
		},
	}
}

// BuildBuyConfirmKb — кнопки подтвердить/отменить перед списанием.
func BuildBuyConfirmKb() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: texts.ConfirmBtn, CallbackData: utils.BuyConfirmCallback},
				{Text: texts.CancelBtn, CallbackData: utils.BuyCancelCallback},
			},
		},
	}
}
