package service

import (
	"context"
	"time"

	initdata "github.com/telegram-mini-apps/init-data-golang"
	"github.com/trottling/Telegram-Store/internal/auth/web"
	"github.com/trottling/Telegram-Store/internal/config"
	"github.com/trottling/Telegram-Store/internal/domain/adminsession"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	"go.uber.org/zap"
)

// initDataTTL — как давно создана initData, если больше - ошибка.
const initDataExpIn = 24 * time.Hour

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
	userRepo    repository.UserRepository
	store       adminsession.Store
	adminConfig *config.AdminPanelConfig
	log         *zap.SugaredLogger
}

func NewAdminAuthSrv(userRepo repository.UserRepository,
	store adminsession.Store,
	adminConfig *config.AdminPanelConfig,
	log *zap.SugaredLogger,
) *AdminAuthSrv {
	return &AdminAuthSrv{userRepo: userRepo, store: store, adminConfig: adminConfig, log: log}
}

func (s *AdminAuthSrv) ExchangeInitData(ctx context.Context, initData string) (string, *models.User, error) {
	if err := initdata.Validate(initData, s.adminConfig.BotToken, initDataExpIn); err != nil {
		s.log.Errorw("admin_auth_service: initData exchange error on validating", "error", err)
		return "", nil, domainerrors.ErrInvalidInitData
	}

	data, err := initdata.Parse(initData)
	if err != nil {
		s.log.Errorw("admin_auth_service: initData exchange error on parsing", "error", err)
		return "", nil, domainerrors.ErrInvalidInitData
	}

	telegramID := data.User.ID

	user, err := s.userRepo.GetByID(ctx, telegramID)
	if err != nil {
		return "", nil, err
	}
	if !user.IsAdmin() {
		return "", nil, domainerrors.ErrNotAdmin
	}

	plaintext, err := admintoken.GenerateSessionJWT(telegramID, sessionTTL, s.adminConfig.JWTSecret)
	if err != nil {
		return "", nil, err
	}
	if err = s.store.SetSession(ctx, admintoken.Hash(plaintext, s.adminConfig.JWTSecret), telegramID, sessionTTL); err != nil {
		return "", nil, err
	}

	s.log.Infow("admin_auth_service: initData exchanged for session", "telegram_id", telegramID)
	return plaintext, user, nil
}

// ValidateSession проверяет токен тремя способами: подпись/срок JWT →
// живая сессия в Redis (отзыв через Logout) → пользователь всё ещё админ в
// Postgres. Любой из трёх — ErrInvalidToken.
func (s *AdminAuthSrv) ValidateSession(ctx context.Context, sessionToken string) (*models.User, error) {
	claims, err := admintoken.ParseSessionJWT(sessionToken, s.adminConfig.JWTSecret)
	if err != nil {
		return nil, domainerrors.ErrInvalidToken
	}

	storedTelegramID, err := s.store.GetSession(ctx, admintoken.Hash(sessionToken, s.adminConfig.JWTSecret))
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
	return s.store.DeleteSession(ctx, admintoken.Hash(sessionToken, s.adminConfig.JWTSecret))
}

func (s *AdminAuthSrv) AllowExchangeAttempt(ctx context.Context, key string) (bool, error) {
	attempt, err := s.store.IncrExchangeAttempts(ctx, key, exchangeAttemptWindow)
	if err != nil {
		return false, err
	}
	return attempt <= exchangeAttemptLimit, nil
}
