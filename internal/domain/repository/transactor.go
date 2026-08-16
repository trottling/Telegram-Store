package repository

import "context"

// Transactor выполняет fn в одной транзакции. Репозитории сами достают
// активную транзакцию из ctx — сервисы не видят *gorm.DB.
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
