package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/trottling/Telegram-Store/internal/domain/models"
	"go.uber.org/zap"
)

// fakeDB — минимальная модель того, что делает Postgres: fakeTransactor снимает
// с неё копию и восстанавливает, если функция транзакции вернула ошибку.
// Без этого нельзя проверить главное свойство Confirm — что упавшее начисление
// откатывает и отметку "оплачено", оставляя счёт пригодным для ретрая.
type fakeDB struct {
	status         models.ReplenishmentStatus
	balance        float64
	balanceCalls   int
	invalidateCall int
}

type fakeTransactor struct{ db *fakeDB }

func (t *fakeTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	snapshot := *t.db
	if err := fn(ctx); err != nil {
		balanceCalls, invalidateCall := t.db.balanceCalls, t.db.invalidateCall
		*t.db = snapshot
		// Счётчики вызовов — наблюдение за тестом, а не данные: их откат
		// скрыл бы от проверок сам факт попытки начисления.
		t.db.balanceCalls, t.db.invalidateCall = balanceCalls, invalidateCall
		return err
	}
	return nil
}

type fakeReplRepo struct {
	db *fakeDB
}

func (r *fakeReplRepo) GetByMerchantInvoiceID(context.Context, models.Merchant, string) (*models.Replenishment, error) {
	return &models.Replenishment{ID: 7, UserID: 100, Amount: 250, Status: r.db.status}, nil
}

func (r *fakeReplRepo) UpdateStatus(_ context.Context, _ int64, status models.ReplenishmentStatus, _ *time.Time) (bool, error) {
	if r.db.status != models.ReplenishmentStatusPending {
		return false, nil
	}
	r.db.status = status
	return true, nil
}

func (r *fakeReplRepo) Create(context.Context, *models.Replenishment) error {
	panic("не используется")
}
func (r *fakeReplRepo) GetByID(context.Context, int64) (*models.Replenishment, error) {
	panic("не используется")
}
func (r *fakeReplRepo) ListByUserID(context.Context, int64, int, int) ([]models.Replenishment, error) {
	panic("не используется")
}
func (r *fakeReplRepo) CountByUserID(context.Context, int64) (int64, error) {
	panic("не используется")
}
func (r *fakeReplRepo) SumPaidByUserMerchant(context.Context, int64, models.Merchant) (float64, error) {
	panic("не используется")
}
func (r *fakeReplRepo) ListAllAdmin(context.Context, models.ReplenishmentAdminFilter, int, int) ([]models.ReplenishmentAdminItem, error) {
	panic("не используется")
}
func (r *fakeReplRepo) CountAllAdmin(context.Context, models.ReplenishmentAdminFilter) (int64, error) {
	panic("не используется")
}

type fakeUserRepo struct {
	db      *fakeDB
	failErr error
}

func (r *fakeUserRepo) UpdateBalance(_ context.Context, _ int64, delta float64) error {
	r.db.balanceCalls++
	if r.failErr != nil {
		return r.failErr
	}
	r.db.balance += delta
	return nil
}

func (r *fakeUserRepo) GetByID(context.Context, int64) (*models.User, error) {
	panic("не используется")
}
func (r *fakeUserRepo) Create(context.Context, *models.User) error {
	panic("не используется")
}
func (r *fakeUserRepo) Update(context.Context, *models.User) error {
	panic("не используется")
}
func (r *fakeUserRepo) List(context.Context, int, int) ([]models.User, error) {
	panic("не используется")
}
func (r *fakeUserRepo) Count(context.Context) (int64, error) { panic("не используется") }
func (r *fakeUserRepo) CountReferrals(context.Context, int64) (int64, error) {
	panic("не используется")
}
func (r *fakeUserRepo) ListReferrals(context.Context, int64, int, int) ([]models.User, error) {
	panic("не используется")
}
func (r *fakeUserRepo) EnsureRootAdminExists(context.Context, int64) error {
	panic("не используется")
}

type fakeUserCache struct{ db *fakeDB }

func (c *fakeUserCache) InvalidateUser(context.Context, int64) error {
	c.db.invalidateCall++
	return nil
}
func (c *fakeUserCache) GetUser(context.Context, int64) (*models.User, error) {
	panic("не используется")
}
func (c *fakeUserCache) SetUser(context.Context, *models.User) error {
	panic("не используется")
}

func newTestSrv(db *fakeDB, balanceErr error) *ReplenishmentSrv {
	log := zap.NewNop().Sugar()
	return NewReplenishmentSrv(
		&fakeReplRepo{db: db},
		&fakeUserRepo{db: db, failErr: balanceErr},
		&fakeTransactor{db: db},
		nil,
		&fakeUserCache{db: db},
		nil, // checkCooldown — CheckInvoice в этих тестах не вызывается
		log,
	)
}

// TestConfirmCreditsOnce — повторный вебхук мерчанта не начисляет второй раз.
func TestConfirmCreditsOnce(t *testing.T) {
	db := &fakeDB{status: models.ReplenishmentStatusPending}
	srv := newTestSrv(db, nil)

	if err := srv.Confirm(context.Background(), models.MerchantCrystalPay, "inv-1", 0); err != nil {
		t.Fatalf("первый Confirm вернул ошибку: %v", err)
	}
	if err := srv.Confirm(context.Background(), models.MerchantCrystalPay, "inv-1", 0); err != nil {
		t.Fatalf("повторный Confirm вернул ошибку: %v", err)
	}

	if db.balance != 250 {
		t.Errorf("баланс = %v, ожидалось 250 (начислено ровно один раз)", db.balance)
	}
	if db.status != models.ReplenishmentStatusPaid {
		t.Errorf("статус = %q, ожидалось paid", db.status)
	}
	if db.invalidateCall != 1 {
		t.Errorf("инвалидаций кэша = %d, ожидалась 1 (только после реального начисления)", db.invalidateCall)
	}
}

// TestConfirmRollsBackOnBalanceFailure — регрессия на баг, из-за которого
// деньги терялись безвозвратно: статус коммитился отдельно от начисления, и
// после сбоя ретрай вебхука получал changed=false и молча возвращал успех.
func TestConfirmRollsBackOnBalanceFailure(t *testing.T) {
	db := &fakeDB{status: models.ReplenishmentStatusPending}
	balanceErr := errors.New("соединение с БД потеряно")
	srv := newTestSrv(db, balanceErr)

	if err := srv.Confirm(context.Background(), models.MerchantCrystalPay, "inv-1", 0); !errors.Is(err, balanceErr) {
		t.Fatalf("Confirm вернул %v, ожидалась ошибка начисления", err)
	}

	if db.status != models.ReplenishmentStatusPending {
		t.Fatalf("статус = %q, ожидался pending: откат обязателен, иначе ретрай уже не начислит", db.status)
	}
	if db.balance != 0 {
		t.Errorf("баланс = %v, ожидался 0", db.balance)
	}
	if db.invalidateCall != 0 {
		t.Errorf("инвалидаций кэша = %d, ожидалось 0 (начисления не было)", db.invalidateCall)
	}

	// Счёт остался pending, поэтому ретрай мерчанта доводит дело до конца.
	srv = newTestSrv(db, nil)
	if err := srv.Confirm(context.Background(), models.MerchantCrystalPay, "inv-1", 0); err != nil {
		t.Fatalf("ретрай после сбоя вернул ошибку: %v", err)
	}
	if db.balance != 250 {
		t.Errorf("баланс после ретрая = %v, ожидалось 250", db.balance)
	}
}
