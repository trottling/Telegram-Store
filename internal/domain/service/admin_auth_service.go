package service

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

// AdminAuthService — вход в админ-панель: Mini App открывает /start с
// initData, который Telegram подписывает сам, страница меняет его на
// сессионный токен (ExchangeInitData), дальше сессия живёт через
// ValidateSession/Logout.
type AdminAuthService interface {
	// ExchangeInitData одноразово меняет initData на токен, если пользователь
	// всё ещё админ. ErrInvalidInitData — initData неверная/истёкла,
	// ErrNotAdmin — прав нет или их сняли после выдачи initData.
	ExchangeInitData(ctx context.Context, initData string) (sessionToken string, admin *models.User, err error)

	// ValidateSession резолвит токен в админа. ErrInvalidToken — сессия не
	// найдена/истекла или права сняты.
	ValidateSession(ctx context.Context, sessionToken string) (*models.User, error)

	Logout(ctx context.Context, sessionToken string) error

	// AllowExchangeAttempt — можно ли клиенту key (IP) ещё раз попытаться
	// обменять initData. false — лимит исчерпан, ошибка — Redis недоступен.
	AllowExchangeAttempt(ctx context.Context, key string) (bool, error)
}
