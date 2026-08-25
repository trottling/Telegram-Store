// Package dto — тела запросов и пагинация admin API. Ответы в основном
// переиспользуют типы из internal/domain/models напрямую.
package dto

import domainmodels "github.com/trottling/Telegram-Store/internal/domain/models"

// Paginated — обёртка, в которой отвечают все list-эндпоинты.
type Paginated[T any] struct {
	Items  []T   `json:"items"`
	Total  int64 `json:"total"`
	Offset int   `json:"offset"`
	Limit  int   `json:"limit"`
}

// binding-теги повторяют проверки сервисного слоя (AdminSrv), а не заменяют их:
// сервис остаётся последней инстанцией, теги лишь дают 400 сразу и с понятным
// ответом вместо доменной ошибки после лишнего похода в БД. Отступать от
// серверных условий нельзя — разойдясь, они начнут отвергать валидные запросы.
// На bool теги не вешаем: для required значение false неотличимо от «не задано».

type CreateCategoryRequest struct {
	ParentID    *domainmodels.CategoryID `json:"parent_id"`
	Name        string                   `json:"name" binding:"required"`
	Description string                   `json:"description"`
}

type UpdateCategoryRequest struct {
	ParentID    *domainmodels.CategoryID `json:"parent_id"`
	Name        string                   `json:"name" binding:"required"`
	Description string                   `json:"description"`
}

type CreateProductRequest struct {
	CategoryID  *domainmodels.CategoryID `json:"category_id"`
	Name        string                   `json:"name" binding:"required"`
	Description string                   `json:"description"`
	Price       float64                  `json:"price" binding:"required,gt=0"`
}

type UpdateProductRequest struct {
	CategoryID  *domainmodels.CategoryID `json:"category_id"`
	Name        string                   `json:"name" binding:"required"`
	Description string                   `json:"description"`
	Price       float64                  `json:"price" binding:"required,gt=0"`
	IsActive    bool                     `json:"is_active"`
}

type AddItemsRequest struct {
	Contents []string `json:"contents" binding:"required,min=1"`
}

// UpdateBalanceRequest — required здесь и означает «не ноль»: для числа нулевое
// значение неотличимо от отсутствующего, а нулевая корректировка баланса
// бессмысленна (AdminSrv.AddBalance отвергает её же).
type UpdateBalanceRequest struct {
	Amount float64 `json:"amount" binding:"required"` // может быть отрицательной — списание
}

// UpdateSettingsRequest — на вложенные структуры теги не вешаем: это типы из
// internal/domain/models, и транспортным ограничениям там не место. Диапазон
// Referral.Percent проверяет AdminSrv.UpdateSettings.
type UpdateSettingsRequest struct {
	SupportUsername               string                          `json:"support_username" binding:"required"`
	CatalogRefreshIntervalSeconds int                             `json:"catalog_refresh_interval_seconds" binding:"required"`
	CrystalPay                    domainmodels.CrystalPaySettings `json:"crystalpay"`
	YooKassa                      domainmodels.YooKassaSettings   `json:"yookassa"`
	Tinkoff                       domainmodels.TinkoffSettings    `json:"tinkoff"`
	Dummy                         domainmodels.DummySettings      `json:"dummy"`
	Referral                      domainmodels.ReferralSettings   `json:"referral"`
}

// ExchangeRequest — тело POST /api/auth/exchange, initData от Telegram.
type ExchangeRequest struct {
	InitData string `json:"init_data" binding:"required"`
}

// TokenResponse — сессионный токен, выдаётся в обмен на initData.
type TokenResponse struct {
	Token string `json:"token"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
