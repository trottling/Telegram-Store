package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// Префиксы callback_data для регистрации хендлеров.
const (
	ProductCallbackPrefix       = "product_"
	BuyCallbackPrefix           = "buy_"
	BuyQtyCallbackPrefix        = "buyqty_"
	PurchaseCallbackPrefix      = "purchase_"
	PurchasesPageCallbackPrefix = "purchasespage_"
	CategoryCallbackPrefix      = "category_"
)

// CatalogRootCallback — без id-суффикса, матчится точно, не через ParseCallbackQuery.
const CatalogRootCallback = "catalog_root"

const (
	MainMenuCallback     = "main_menu"
	StartProfileCallback = "start_profile"
	BuyCancelCallback    = "buycancel"
	BuyConfirmCallback   = "buyconfirm"
)

func BuildProductCallback(productId int64) string {
	return fmt.Sprintf("%s%d", ProductCallbackPrefix, productId)
}

func BuildBuyCallback(productId int64) string {
	return fmt.Sprintf("%s%d", BuyCallbackPrefix, productId)
}

func BuildCategoryCallback(categoryId int64) string {
	return fmt.Sprintf("%s%d", CategoryCallbackPrefix, categoryId)
}

// BuildBuyQtyCallback несёт только число — товар лежит в состоянии FSM.
func BuildBuyQtyCallback(qty int) string {
	return fmt.Sprintf("%s%d", BuyQtyCallbackPrefix, qty)
}

func BuildPurchaseBatchCallback(batchID string) string {
	return fmt.Sprintf("%s%s", PurchaseCallbackPrefix, batchID)
}

func BuildPurchasesPageCallback(offset int) string {
	return fmt.Sprintf("%s%d", PurchasesPageCallbackPrefix, offset)
}

// ParseCallbackQuery извлекает числовой ID из хвоста "prefix_<id>".
func ParseCallbackQuery(query string) (int64, error) {
	parts := strings.Split(query, "_")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid callback format")
	}
	return strconv.ParseInt(parts[len(parts)-1], 10, 64)
}

// ParseBatchCallbackQuery извлекает UUID батча — в UUID дефисы, не
// подчёркивания, так что хвост после split("_") остаётся целым.
func ParseBatchCallbackQuery(query string) (string, error) {
	parts := strings.Split(query, "_")
	if len(parts) < 2 || parts[len(parts)-1] == "" {
		return "", fmt.Errorf("invalid callback format")
	}
	return parts[len(parts)-1], nil
}
