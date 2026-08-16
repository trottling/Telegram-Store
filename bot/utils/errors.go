package utils

import (
	"errors"

	"github.com/trottling/TG-Store/bot/texts"
	domainerrors "github.com/trottling/TG-Store/internal/domain/errors"
)

// UserFacingError превращает доменную ошибку в текст для пользователя;
// нераспознанное — в общее сообщение, реальная ошибка остаётся в логах.
func UserFacingError(err error) string {
	switch {
	case errors.Is(err, domainerrors.ErrNotEnoughBalance):
		return texts.ErrInsufficientBalanceMsg
	case errors.Is(err, domainerrors.ErrProductOutOfStock):
		return texts.ErrOutOfStockMsg
	case errors.Is(err, domainerrors.ErrProductInactive):
		return texts.ErrProductInactiveMsg
	case errors.Is(err, domainerrors.ErrProductNotFound):
		return texts.ErrProductInactiveMsg
	case errors.Is(err, domainerrors.ErrTooManyProducts):
		return texts.ErrTooManyProductsMsg
	case errors.Is(err, domainerrors.ErrInvalidQuantity):
		return texts.InvalidQuantityMsg
	default:
		return texts.ErrGenericMsg
	}
}
