package models

import (
	"database/sql/driver"
	"fmt"

	"github.com/shopspring/decimal"

	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
)

// Money — неотрицательная денежная сумма (цена, баланс, сумма покупки/
// пополнения). Immutable, на shopspring/decimal — float64 для денег не
// используется нигде, это защита от ошибок округления. Знаковые
// корректировки (дельта UserRepository.UpdateBalance, ручная правка баланса
// админом) — это не "сумма денег", а сигнатурное изменение к ней, поэтому
// они остаются обычным decimal.Decimal, а не Money.
type Money struct {
	d decimal.Decimal
}

// NewMoney парсит строку (пользовательский ввод, строковые суммы из вебхуков
// платёжных провайдеров) и отклоняет отрицательное/нечисловое значение.
func NewMoney(s string) (Money, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Money{}, err
	}
	return newMoney(d)
}

// NewMoneyFromFloat — для float64-границ (запросы admin_backend, уже
// распарсенные суммы из SDK платёжных провайдеров).
func NewMoneyFromFloat(f float64) (Money, error) {
	return newMoney(decimal.NewFromFloat(f))
}

// MoneyFromCents — для мерчантов, отдающих сумму в копейках (Tinkoff).
func MoneyFromCents(cents int64) (Money, error) {
	return newMoney(decimal.New(cents, -2))
}

func newMoney(d decimal.Decimal) (Money, error) {
	if d.IsNegative() {
		return Money{}, domainerrors.ErrNegativeAmount
	}
	return Money{d: d}, nil
}

func (m Money) Add(other Money) Money {
	return Money{d: m.d.Add(other.d)}
}

// Sub вычитает other, возвращая domainerrors.ErrNotEnoughBalance, если
// результат ушёл бы в минус.
func (m Money) Sub(other Money) (Money, error) {
	res := m.d.Sub(other.d)
	if res.IsNegative() {
		return Money{}, domainerrors.ErrNotEnoughBalance
	}
	return Money{d: res}, nil
}

func (m Money) Mul(qty int) Money {
	return Money{d: m.d.Mul(decimal.NewFromInt(int64(qty)))}
}

// Percent — pct% от суммы (используется для расчёта реферального начисления).
func (m Money) Percent(pct int) Money {
	return Money{d: m.d.Mul(decimal.NewFromInt(int64(pct))).Div(decimal.NewFromInt(100))}
}

func (m Money) LessThan(other Money) bool {
	return m.d.LessThan(other.d)
}

func (m Money) GreaterThan(other Money) bool {
	return m.d.GreaterThan(other.d)
}

func (m Money) IsZero() bool {
	return m.d.IsZero()
}

// Equal — decimal.Decimal хранит значение через *big.Int, поэтому обычное
// "==" сравнивает указатели, а не сумму; используйте Equal вместо "==".
func (m Money) Equal(other Money) bool {
	return m.d.Equal(other.d)
}

// String форматирует сумму с двумя знаками после запятой — формат, который
// весь бот и панель показывают пользователю сегодня.
func (m Money) String() string {
	return m.d.StringFixed(2)
}

// Float64 — только для границ, которым нужен именно float64 (метрики
// Prometheus): наблюдение по сути и так приблизительное.
func (m Money) Float64() float64 {
	f, _ := m.d.Float64()
	return f
}

// Decimal — доступ к внутреннему представлению для знаковых конверсий на
// границе репозитория (например .Neg() перед UserRepository.UpdateBalance).
func (m Money) Decimal() decimal.Decimal {
	return m.d
}

func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(m.d.StringFixed(2)), nil
}

func (m *Money) UnmarshalJSON(b []byte) error {
	var d decimal.Decimal
	if err := d.UnmarshalJSON(b); err != nil {
		return err
	}
	money, err := newMoney(d)
	if err != nil {
		return err
	}
	*m = money
	return nil
}

// Value/Scan — Money хранится в decimal(12,2) колонке; GORM/pgx трактуют его
// как обычное скалярное значение, отдельная persistence-модель не нужна.
func (m Money) Value() (driver.Value, error) {
	return m.d.Value()
}

func (m *Money) Scan(v any) error {
	var d decimal.Decimal
	if err := d.Scan(v); err != nil {
		return fmt.Errorf("money: scan: %w", err)
	}
	money, err := newMoney(d)
	if err != nil {
		return err
	}
	*m = money
	return nil
}
