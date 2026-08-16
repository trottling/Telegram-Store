package service

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	domaincache "github.com/trottling/TG-Store/internal/domain/cache"
	domainerrors "github.com/trottling/TG-Store/internal/domain/errors"
	"github.com/trottling/TG-Store/internal/domain/models"
	"github.com/trottling/TG-Store/internal/domain/repository"
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
func (s *UserSrv) GetOrCreate(ctx context.Context, telegramID int64, username string) (*models.User, error) {
	if user, err := s.cache.GetUser(ctx, telegramID); err == nil {
		return user, nil
	}

	user, err := s.userRepo.GetByID(ctx, telegramID)
	if err != nil {
		if !errors.Is(err, domainerrors.ErrUserNotFound) {
			return nil, err
		}
		user = &models.User{TelegramID: telegramID, Username: username}
		if err = s.userRepo.Create(ctx, user); err != nil {
			return nil, err
		}
		s.log.WithField("telegram_id", telegramID).Info("user_service: new user registered")
	}

	_ = s.cache.SetUser(ctx, user)
	return user, nil
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

// ListAdmin/CountAdmin намеренно мимо кэша.
func (s *UserSrv) ListAdmin(ctx context.Context, offset, limit int) ([]models.User, error) {
	return s.userRepo.List(ctx, offset, limit)
}

func (s *UserSrv) CountAdmin(ctx context.Context) (int64, error) {
	return s.userRepo.Count(ctx)
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
