package utils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// Префиксы callback_data для регистрации хендлеров.
const (
	ProductCallbackPrefix             = "product_"
	BuyCallbackPrefix                 = "buy_"
	BuyQtyCallbackPrefix              = "buyqty_"
	PurchaseCallbackPrefix            = "purchase_"
	PurchasesPageCallbackPrefix       = "purchasespage_"
	CategoryCallbackPrefix            = "category_"
	RefillMerchantCallbackPrefix      = "refillmerchant_"
	ReplenishmentsPageCallbackPrefix  = "replenishmentspage_"
	CheckPaymentCallbackPrefix        = "checkpay_"
	ReplenishmentDetailCallbackPrefix = "replenishmentdetail_"
)

// CatalogRootCallback — без id-суффикса, матчится точно, не через ParseCallbackQuery.
const CatalogRootCallback = "catalog_root"

const (
	MainMenuCallback         = "main_menu"
	StartProfileCallback     = "start_profile"
	BuyCancelCallback        = "buycancel"
	BuyConfirmCallback       = "buyconfirm"
	ReferralCloseCallback    = "referral_close"
	SettingsLanguageCallback = "settings_language"
)

// LanguageCallbackPrefix — выбор языка в меню языка ("lang_ru"/"lang_en"), сам код языка берется как есть (не число).
const LanguageCallbackPrefix = "lang_"

func BuildLanguageCallback(lang string) string {
	return LanguageCallbackPrefix + lang
}

func ParseLanguageCallback(query string) (string, error) {
	lang, ok := strings.CutPrefix(query, LanguageCallbackPrefix)
	if !ok || lang == "" {
		return "", fmt.Errorf("invalid callback format")
	}
	return lang, nil
}

// compactUUID/expandUUID — UUID в callback_data кодируется без дефисов (32
// hex-символа вместо 36): экономит 4 байта на каждый ID, а лимит Telegram на
// callback_data — 64 байта, и составные callback'ы (offset+UUID) иначе
// подходят к нему вплотную. models.Parse*ID принимает обе формы напрямую
// (стандартный uuid.Parse понимает 32-символьную форму без дефисов), так что
// разворачивать обратно перед парсингом не нужно.
func compactUUID(s string) string {
	return strings.ReplaceAll(s, "-", "")
}

func BuildProductCallback(id models.ProductID) string {
	return ProductCallbackPrefix + compactUUID(id.String())
}

func BuildBuyCallback(id models.ProductID) string {
	return BuyCallbackPrefix + compactUUID(id.String())
}

func BuildCategoryCallback(id models.CategoryID) string {
	return CategoryCallbackPrefix + compactUUID(id.String())
}

// BuildBuyQtyCallback несёт только число — товар лежит в состоянии FSM.
func BuildBuyQtyCallback(qty int) string {
	return fmt.Sprintf("%s%d", BuyQtyCallbackPrefix, qty)
}

// BuildPurchaseBatchCallback несёт и offset страницы списка — чтобы кнопка
// «назад» на карточке батча знала, на какую страницу списка возвращаться.
func BuildPurchaseBatchCallback(offset int, batchID models.BatchID) string {
	return fmt.Sprintf("%s%d_%s", PurchaseCallbackPrefix, offset, batchID.String())
}

func BuildPurchasesPageCallback(offset int) string {
	return fmt.Sprintf("%s%d", PurchasesPageCallbackPrefix, offset)
}

func BuildReplenishmentsPageCallback(offset int) string {
	return fmt.Sprintf("%s%d", ReplenishmentsPageCallbackPrefix, offset)
}

func BuildCheckPaymentCallback(id models.ReplenishmentID) string {
	return CheckPaymentCallbackPrefix + compactUUID(id.String())
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

// tailSegment извлекает хвост из "prefix_<tail>" — общая часть для всех
// однозначных (без второго "_"-сегмента) callback'ов.
func tailSegment(query string) (string, error) {
	parts := strings.Split(query, "_")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid callback format")
	}
	return parts[len(parts)-1], nil
}

// ParseCallbackQuery извлекает числовой ID из хвоста "prefix_<id>" — для
// callback'ов, не несущих ID сущности (offset пагинации, buyqty_ с
// количеством).
func ParseCallbackQuery(query string) (int64, error) {
	tail, err := tailSegment(query)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(tail, 10, 64)
}

func ParseProductCallback(query string) (models.ProductID, error) {
	tail, err := tailSegment(query)
	if err != nil {
		return models.ProductID{}, err
	}
	return models.ParseProductID(tail)
}

func ParseCategoryCallback(query string) (models.CategoryID, error) {
	tail, err := tailSegment(query)
	if err != nil {
		return models.CategoryID{}, err
	}
	return models.ParseCategoryID(tail)
}

func ParseReplenishmentCallback(query string) (models.ReplenishmentID, error) {
	tail, err := tailSegment(query)
	if err != nil {
		return models.ReplenishmentID{}, err
	}
	return models.ParseReplenishmentID(tail)
}

// ParseBatchCallbackQuery извлекает offset страницы и UUID батча из
// "purchase_<offset>_<uuid>" — в UUID дефисы, не подчёркивания, так что
// достаточно разрезать по первому "_" после offset.
func ParseBatchCallbackQuery(query string) (offset int, batchID models.BatchID, err error) {
	rest, ok := strings.CutPrefix(query, PurchaseCallbackPrefix)
	if !ok {
		return 0, models.BatchID{}, fmt.Errorf("invalid callback format")
	}

	offsetStr, id, ok := strings.Cut(rest, "_")
	if !ok || id == "" {
		return 0, models.BatchID{}, fmt.Errorf("invalid callback format")
	}

	offset, err = strconv.Atoi(offsetStr)
	if err != nil {
		return 0, models.BatchID{}, fmt.Errorf("invalid callback format")
	}
	batchID, err = models.ParseBatchID(id)
	if err != nil {
		return 0, models.BatchID{}, fmt.Errorf("invalid callback format")
	}
	return offset, batchID, nil
}

// BuildReplenishmentDetailCallback несёт offset страницы (для кнопки «назад»,
// как у покупок) и внутренний ID Replenishment.
func BuildReplenishmentDetailCallback(offset int, id models.ReplenishmentID) string {
	return fmt.Sprintf("%s%d_%s", ReplenishmentDetailCallbackPrefix, offset, compactUUID(id.String()))
}

func ParseReplenishmentDetailCallback(query string) (offset int, id models.ReplenishmentID, err error) {
	rest, ok := strings.CutPrefix(query, ReplenishmentDetailCallbackPrefix)
	if !ok {
		return 0, models.ReplenishmentID{}, fmt.Errorf("invalid callback format")
	}

	offsetStr, idStr, ok := strings.Cut(rest, "_")
	if !ok || idStr == "" {
		return 0, models.ReplenishmentID{}, fmt.Errorf("invalid callback format")
	}

	offset, err = strconv.Atoi(offsetStr)
	if err != nil {
		return 0, models.ReplenishmentID{}, fmt.Errorf("invalid callback format")
	}
	id, err = models.ParseReplenishmentID(idStr)
	if err != nil {
		return 0, models.ReplenishmentID{}, fmt.Errorf("invalid callback format")
	}
	return offset, id, nil
}
