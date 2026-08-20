package service

import (
	"context"
	"errors"
	"time"

	"github.com/trottling/Telegram-Store/internal/auth/web"
	"github.com/trottling/Telegram-Store/internal/domain/adminsession"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	"go.uber.org/zap"
)

// loginCodeTTL — срок жизни одноразового кода от /admin.
const loginCodeTTL = 30 * time.Second

// sessionTTL — срок жизни сессии панели после обмена кода.
const sessionTTL = 24 * time.Hour

// Лимит на обмен кода. Режет автоподбор, а не человека: код 6-значный и живёт
// 30 секунд, живому админу хватает одной попытки на вход. Атакующему остаётся
// порядка пяти догадок на срок жизни кода из миллиона возможных.
const (
	exchangeAttemptLimit  = 10
	exchangeAttemptWindow = time.Minute
)

type AdminAuthSrv struct {
	userRepo  repository.UserRepository
	store     adminsession.Store
	jwtSecret []byte
	log       *zap.SugaredLogger
}

func NewAdminAuthSrv(userRepo repository.UserRepository, store adminsession.Store, jwtSecret []byte, log *zap.SugaredLogger) *AdminAuthSrv {
	return &AdminAuthSrv{userRepo: userRepo, store: store, jwtSecret: jwtSecret, log: log}
}

func (s *AdminAuthSrv) IssueLoginCode(ctx context.Context, telegramID int64) (string, error) {
	code, hash, err := admintoken.GenerateCode(s.jwtSecret)
	if err != nil {
		return "", err
	}
	if err = s.store.SetLoginCode(ctx, hash, telegramID, loginCodeTTL); err != nil {
		return "", err
	}
	return code, nil
}

func (s *AdminAuthSrv) ExchangeLoginCode(ctx context.Context, code string) (string, *models.User, error) {
	telegramID, err := s.store.ConsumeLoginCode(ctx, admintoken.Hash(code, s.jwtSecret))
	if err != nil {
		if errors.Is(err, adminsession.ErrNotFound) {
			return "", nil, domainerrors.ErrInvalidLoginCode
		}
		return "", nil, err
	}

	user, err := s.userRepo.GetByID(ctx, telegramID)
	if err != nil {
		return "", nil, err
	}
	if !user.IsAdmin() {
		return "", nil, domainerrors.ErrNotAdmin
	}

	plaintext, err := admintoken.GenerateSessionJWT(telegramID, sessionTTL, s.jwtSecret)
	if err != nil {
		return "", nil, err
	}
	if err = s.store.SetSession(ctx, admintoken.Hash(plaintext, s.jwtSecret), telegramID, sessionTTL); err != nil {
		return "", nil, err
	}

	s.log.Infow("admin_auth_service: login code exchanged for session", "telegram_id", telegramID)
	return plaintext, user, nil
}

// ValidateSession проверяет токен тремя способами: подпись/срок JWT →
// живая сессия в Redis (отзыв через Logout) → пользователь всё ещё админ в
// Postgres. Любой из трёх — ErrInvalidToken.
func (s *AdminAuthSrv) ValidateSession(ctx context.Context, sessionToken string) (*models.User, error) {
	claims, err := admintoken.ParseSessionJWT(sessionToken, s.jwtSecret)
	if err != nil {
		return nil, domainerrors.ErrInvalidToken
	}

	storedTelegramID, err := s.store.GetSession(ctx, admintoken.Hash(sessionToken, s.jwtSecret))
	if err != nil || storedTelegramID != claims.TelegramID {
		return nil, domainerrors.ErrInvalidToken
	}

	user, err := s.userRepo.GetByID(ctx, claims.TelegramID)
	if err != nil {
		return nil, domainerrors.ErrInvalidToken
	}
	if !user.IsAdmin() {
		return nil, domainerrors.ErrInvalidToken
	}
	return user, nil
}

func (s *AdminAuthSrv) Logout(ctx context.Context, sessionToken string) error {
	return s.store.DeleteSession(ctx, admintoken.Hash(sessionToken, s.jwtSecret))
}

func (s *AdminAuthSrv) AllowExchangeAttempt(ctx context.Context, key string) (bool, error) {
	attempt, err := s.store.IncrExchangeAttempts(ctx, key, exchangeAttemptWindow)
	if err != nil {
		return false, err
	}
	return attempt <= exchangeAttemptLimit, nil
}
