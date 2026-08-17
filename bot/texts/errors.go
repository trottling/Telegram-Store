package texts

import (
	"errors"

	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
)

// UserFacingError превращает доменную ошибку в текст для пользователя на
// языке lang; нераспознанное — в общее сообщение, реальная ошибка остаётся в логах.
func UserFacingError(lang string, err error) string {
	switch {
	case errors.Is(err, domainerrors.ErrNotEnoughBalance):
		return T(lang, ErrInsufficientBalanceMsg, nil)
	case errors.Is(err, domainerrors.ErrProductOutOfStock):
		return T(lang, ErrOutOfStockMsg, nil)
	case errors.Is(err, domainerrors.ErrProductInactive):
		return T(lang, ErrProductInactiveMsg, nil)
	case errors.Is(err, domainerrors.ErrProductNotFound):
		return T(lang, ErrProductInactiveMsg, nil)
	case errors.Is(err, domainerrors.ErrTooManyProducts):
		return T(lang, ErrTooManyProductsMsg, nil)
	case errors.Is(err, domainerrors.ErrInvalidQuantity):
		return T(lang, InvalidQuantityMsg, nil)
	default:
		return T(lang, ErrGenericMsg, nil)
	}
}
