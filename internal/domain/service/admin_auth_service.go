package service

import (
	"context"

	"github.com/trottling/TG-Store/internal/domain/models"
)

// AdminAuthService — вход в админ-панель: /admin выдаёт одноразовый код
// (IssueLoginCode), логин-страница меняет его на сессионный токен
// (ExchangeLoginCode), дальше сессия живёт через ValidateSession/Logout.
type AdminAuthService interface {
	// IssueLoginCode выдаёт код на 30 секунд для уже-админа telegramID.
	IssueLoginCode(ctx context.Context, telegramID int64) (code string, err error)

	// ExchangeLoginCode одноразово меняет код на токен, если пользователь
	// всё ещё админ. ErrInvalidLoginCode — код неверный/истёк/использован,
	// ErrNotAdmin — права сняли после выдачи кода.
	ExchangeLoginCode(ctx context.Context, code string) (sessionToken string, admin *models.User, err error)

	// ValidateSession резолвит токен в админа. ErrInvalidToken — сессия не
	// найдена/истекла или права сняты.
	ValidateSession(ctx context.Context, sessionToken string) (*models.User, error)

	Logout(ctx context.Context, sessionToken string) error
}
