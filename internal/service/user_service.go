package service

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
)

type UserSrv struct {
	userRepo repository.UserRepository
	cache    domaincache.UserCache
	log      *logrus.Logger
}

func NewUserSrv(userRepo repository.UserRepository, cache domaincache.UserCache, log *logrus.Logger) *UserSrv {
	return &UserSrv{userRepo: userRepo, cache: cache, log: log}
}

// GetOrCreate ищет пользователя по Telegram ID, создаёт при первом контакте.
// referrerID учитывается только на ветке создания — уже существующий
// пользователь рефералом не становится, даже если пришёл по ссылке.
func (s *UserSrv) GetOrCreate(ctx context.Context, telegramID int64, username string, referrerID *int64) (*models.User, error) {
	if user, err := s.cache.GetUser(ctx, telegramID); err == nil {
		return user, nil
	}

	user, err := s.userRepo.GetByID(ctx, telegramID)
	if err != nil {
		if !errors.Is(err, domainerrors.ErrUserNotFound) {
			return nil, err
		}
		user = &models.User{TelegramID: telegramID, Username: username, ReferrerID: s.validReferrer(ctx, telegramID, referrerID)}
		if err = s.userRepo.Create(ctx, user); err != nil {
			return nil, err
		}
		s.log.WithFields(logrus.Fields{"telegram_id": telegramID, "referrer_id": user.ReferrerID}).Info("user_service: new user registered")
	}

	_ = s.cache.SetUser(ctx, user)
	return user, nil
}

// validReferrer — nil, если рефка на себя или на несуществующего пользователя.
func (s *UserSrv) validReferrer(ctx context.Context, telegramID int64, referrerID *int64) *int64 {
	if referrerID == nil || *referrerID == telegramID {
		return nil
	}
	if _, err := s.userRepo.GetByID(ctx, *referrerID); err != nil {
		return nil
	}
	return referrerID
}

// GetProfile — то же, что GetOrCreate, но не создаёт строку.
func (s *UserSrv) GetProfile(ctx context.Context, telegramID int64) (*models.User, error) {
	if user, err := s.cache.GetUser(ctx, telegramID); err == nil {
		return user, nil
	}

	user, err := s.userRepo.GetByID(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	_ = s.cache.SetUser(ctx, user)
	return user, nil
}

// RefreshProfile — то же, что GetProfile, но без чтения кэша: сразу идёт в
// Postgres и перезаписывает кэш свежим значением (для инлайн-кнопки «Обновить»).
func (s *UserSrv) RefreshProfile(ctx context.Context, telegramID int64) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	_ = s.cache.SetUser(ctx, user)
	return user, nil
}

// ListAdmin/CountAdmin намеренно мимо кэша.
func (s *UserSrv) ListAdmin(ctx context.Context, offset, limit int) ([]models.User, error) {
	return s.userRepo.List(ctx, offset, limit)
}

func (s *UserSrv) CountAdmin(ctx context.Context) (int64, error) {
	return s.userRepo.Count(ctx)
}

func (s *UserSrv) CountReferrals(ctx context.Context, telegramID int64) (int64, error) {
	return s.userRepo.CountReferrals(ctx, telegramID)
}

func (s *UserSrv) IsBanned(ctx context.Context, telegramID int64) (bool, error) {
	if user, err := s.cache.GetUser(ctx, telegramID); err == nil {
		return user.IsBanned(), nil
	}

	user, err := s.userRepo.GetByID(ctx, telegramID)
	if err != nil {
		return false, err
	}

	_ = s.cache.SetUser(ctx, user)
	return user.IsBanned(), nil
}
