package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// Префиксы callback_data для регистрации хендлеров.
const (
	ProductCallbackPrefix            = "product_"
	BuyCallbackPrefix                = "buy_"
	BuyQtyCallbackPrefix             = "buyqty_"
	PurchaseCallbackPrefix           = "purchase_"
	PurchasesPageCallbackPrefix      = "purchasespage_"
	CategoryCallbackPrefix           = "category_"
	RefillMerchantCallbackPrefix     = "refillmerchant_"
	ReplenishmentsPageCallbackPrefix = "replenishmentspage_"
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

// BuildPurchaseBatchCallback несёт и offset страницы списка — чтобы кнопка
// «назад» на карточке батча знала, на какую страницу списка возвращаться.
func BuildPurchaseBatchCallback(offset int, batchID string) string {
	return fmt.Sprintf("%s%d_%s", PurchaseCallbackPrefix, offset, batchID)
}

func BuildPurchasesPageCallback(offset int) string {
	return fmt.Sprintf("%s%d", PurchasesPageCallbackPrefix, offset)
}

func BuildReplenishmentsPageCallback(offset int) string {
	return fmt.Sprintf("%s%d", ReplenishmentsPageCallbackPrefix, offset)
}

// BuildRefillMerchantCallback — merchant буквенный (crystalpay/yookassa/...), без underscore внутри.
func BuildRefillMerchantCallback(merchant string) string {
	return RefillMerchantCallbackPrefix + merchant
}

func ParseRefillMerchantCallback(query string) (string, error) {
	merchant, ok := strings.CutPrefix(query, RefillMerchantCallbackPrefix)
	if !ok || merchant == "" {
		return "", fmt.Errorf("invalid callback format")
	}
	return merchant, nil
}

// ParseCallbackQuery извлекает числовой ID из хвоста "prefix_<id>".
func ParseCallbackQuery(query string) (int64, error) {
	parts := strings.Split(query, "_")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid callback format")
	}
	return strconv.ParseInt(parts[len(parts)-1], 10, 64)
}

// ParseBatchCallbackQuery извлекает offset страницы и UUID батча из
// "purchase_<offset>_<uuid>" — в UUID дефисы, не подчёркивания, так что
// достаточно разрезать по первому "_" после offset.
func ParseBatchCallbackQuery(query string) (offset int, batchID string, err error) {
	rest, ok := strings.CutPrefix(query, PurchaseCallbackPrefix)
	if !ok {
		return 0, "", fmt.Errorf("invalid callback format")
	}

	offsetStr, id, ok := strings.Cut(rest, "_")
	if !ok || id == "" {
		return 0, "", fmt.Errorf("invalid callback format")
	}

	offset, err = strconv.Atoi(offsetStr)
	if err != nil {
		return 0, "", fmt.Errorf("invalid callback format")
	}
	return offset, id, nil
}
