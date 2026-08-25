package service

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type userContextKey struct{}

// WithUser кладёт уже полученного пользователя в ctx. Единственный
// вызывающий сейчас — bot/middleware.BanCheck: тот и так вынужден читать
// пользователя свежим, не через кэш, чтобы бан срабатывал на этом же update'е
// (см. RefreshProfile). Кладя результат в ctx, UserSrv.GetProfile ниже по
// цепочке отдаёт его без повторного похода в кэш/Postgres — раньше на каждый
// update уходило два похода за одним и тем же пользователем, теперь один.
func WithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

// UserFromContext — см. WithUser. ok=false, если в ctx ничего не положили
// (например, до BanCheck — команда /start её сознательно пропускает).
func UserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userContextKey{}).(*models.User)
	return user, ok
}
