// Package errors переводит доменные ошибки в HTTP-статус + JSON.
package errors

import (
	"errors"
	"net/http"

	"github.com/trottling/Telegram-Store/admin_backend/dto"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
)

func DomainErrorToResponse(err error) (int, *dto.ErrorResponse) {
	switch {
	case errors.Is(err, domainerrors.ErrUserNotFound),
		errors.Is(err, domainerrors.ErrProductNotFound),
		errors.Is(err, domainerrors.ErrCategoryNotFound),
		errors.Is(err, domainerrors.ErrPurchaseNotFound),
		errors.Is(err, domainerrors.ErrSettingsNotFound):
		return http.StatusNotFound, &dto.ErrorResponse{Code: "not_found", Message: err.Error()}
	case errors.Is(err, domainerrors.ErrInvalidToken):
		return http.StatusUnauthorized, &dto.ErrorResponse{Code: "unauthorized", Message: "invalid or expired session"}
	case errors.Is(err, domainerrors.ErrInvalidLoginCode):
		return http.StatusUnauthorized, &dto.ErrorResponse{Code: "unauthorized", Message: "invalid or expired login code"}
	case errors.Is(err, domainerrors.ErrTooManyAttempts):
		return http.StatusTooManyRequests, &dto.ErrorResponse{Code: "too_many_requests", Message: "too many attempts, try again later"}
	case errors.Is(err, domainerrors.ErrAlreadyAdmin),
		errors.Is(err, domainerrors.ErrNotAdmin),
		errors.Is(err, domainerrors.ErrCannotRevokeRootAdmin),
		errors.Is(err, domainerrors.ErrCannotRevokeSelf),
		errors.Is(err, domainerrors.ErrOnlyRootAdminCanPromote),
		errors.Is(err, domainerrors.ErrCannotBanRootAdmin),
		errors.Is(err, domainerrors.ErrCannotBanSelf),
		errors.Is(err, domainerrors.ErrInvalidAmount),
		errors.Is(err, domainerrors.ErrInvalidProductInput),
		errors.Is(err, domainerrors.ErrNoItemsProvided),
		errors.Is(err, domainerrors.ErrCategoryNotEmpty),
		errors.Is(err, domainerrors.ErrProductHasPurchases),
		errors.Is(err, domainerrors.ErrInvalidSettingsInput):
		return http.StatusBadRequest, &dto.ErrorResponse{Code: "bad_request", Message: err.Error()}
	default:
		return http.StatusInternalServerError, &dto.ErrorResponse{Code: "internal_error", Message: "internal server error"}
	}
}
