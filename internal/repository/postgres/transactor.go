package postgres

import (
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// txCtxKey — ключ ctx для активной транзакции, dbFromCtx её оттуда достаёт.
type txCtxKey struct{}

type GormTransactor struct {
	db  *gorm.DB
	log *logrus.Logger
}

func NewGormTransactor(db *gorm.DB, log *logrus.Logger) *GormTransactor {
	return &GormTransactor{db: db, log: log}
}

func (t *GormTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txCtxKey{}, tx))
	})
	if err != nil {
		t.log.WithError(err).Debug("transactor: transaction rolled back")
	}
	return err
}

// dbFromCtx возвращает транзакцию из ctx, иначе — обычный db.
func dbFromCtx(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txCtxKey{}).(*gorm.DB); ok {
		return tx
	}
	return db
}
