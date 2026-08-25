package service

import (
	"context"
	"errors"

	domaincache "github.com/trottling/Telegram-Store/internal/domain/cache"
	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
	"github.com/trottling/Telegram-Store/internal/domain/models"
	"github.com/trottling/Telegram-Store/internal/domain/repository"
	domainservice "github.com/trottling/Telegram-Store/internal/domain/service"
	"go.uber.org/zap"
)

type UserSrv struct {
	userRepo repository.UserRepository
	cache    domaincache.UserCache
	log      *zap.SugaredLogger
}

func NewUserSrv(userRepo repository.UserRepository, cache domaincache.UserCache, log *zap.SugaredLogger) *UserSrv {
	return &UserSrv{userRepo: userRepo, cache: cache, log: log}
}

// GetOrCreate ищет пользователя по Telegram ID, создаёт при первом контакте.
// referrerID/language учитываются только на ветке создания — уже существующий
// пользователь рефералом не становится и язык не меняет, даже если пришёл по
// реф-ссылке или сменил локаль в Telegram.
func (s *UserSrv) GetOrCreate(ctx context.Context, telegramID int64, username string, referrerID *int64, language string) (*models.User, bool, error) {
	if user, err := s.cache.GetUser(ctx, telegramID); err == nil {
		return user, false, nil
	}

	user, err := s.userRepo.GetByID(ctx, telegramID)
	created := false
	if err != nil {
		if !errors.Is(err, domainerrors.ErrUserNotFound) {
			return nil, false, err
		}
		user = models.NewUser(telegramID, username, language, s.validReferrer(ctx, telegramID, referrerID))
		if err = s.userRepo.Create(ctx, user); err != nil {
			return nil, false, err
		}
		created = true
		s.log.Infow("user_service: new user registered", "telegram_id", telegramID, "referrer_id", user.ReferrerID, "language", language)
	}

	_ = s.cache.SetUser(ctx, user)
	return user, created, nil
}

// SetLanguage — ручное переключение языка интерфейса, перекрывает то, что
// было определено автоматически по Telegram-локали при /start.
func (s *UserSrv) SetLanguage(ctx context.Context, telegramID int64, language string) error {
	user, err := s.userRepo.GetByID(ctx, telegramID)
	if err != nil {
		return err
	}
	if user.Language == language {
		return nil
	}

	user.Language = language
	if err = s.userRepo.Update(ctx, user); err != nil {
		return err
	}
	_ = s.cache.InvalidateUser(ctx, telegramID)
	return nil
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
	// Свежего пользователя уже мог положить bot/middleware.BanCheck (см.
	// domainservice.WithUser) — тогда это вообще без похода в кэш/Postgres.
	if user, ok := domainservice.UserFromContext(ctx); ok && user.TelegramID == telegramID {
		return user, nil
	}
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

// GetFreshProfile — RefreshProfile без записи в кэш, см. doc-комментарий интерфейса.
func (s *UserSrv) GetFreshProfile(ctx context.Context, telegramID int64) (*models.User, error) {
	return s.userRepo.GetByID(ctx, telegramID)
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

func (s *UserSrv) ListReferrals(ctx context.Context, telegramID int64, offset, limit int) ([]models.User, error) {
	return s.userRepo.ListReferrals(ctx, telegramID, offset, limit)
}
