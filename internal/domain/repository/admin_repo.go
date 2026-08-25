package repository

import (
	"context"

	"github.com/trottling/Telegram-Store/internal/domain/models"
)

type AdminLogRepository interface {
	Create(ctx context.Context, log *models.AdminLog) error
	ListByAdmin(ctx context.Context, adminID models.TelegramID, offset, limit int) ([]models.AdminLog, error)

	// ListAll/CountAll — журнал по всем админам, adminID nil = без фильтра.
	ListAll(ctx context.Context, adminID *models.TelegramID, offset, limit int) ([]models.AdminLog, error)
	CountAll(ctx context.Context, adminID *models.TelegramID) (int64, error)
}
