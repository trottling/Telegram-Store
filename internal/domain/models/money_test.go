package models

import (
	"encoding/json"
	"errors"
	"testing"

	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
)

func TestMoney_NewMoney_RejectsNegative(t *testing.T) {
	if _, err := NewMoney("-1.00"); !errors.Is(err, domainerrors.ErrNegativeAmount) {
		t.Fatalf("expected ErrNegativeAmount, got %v", err)
	}
}

func TestMoney_AddSub(t *testing.T) {
	a, _ := NewMoney("10.50")
	b, _ := NewMoney("3.25")

	sum := a.Add(b)
	if sum.String() != "13.75" {
		t.Fatalf("Add: expected 13.75, got %s", sum.String())
	}

	diff, err := a.Sub(b)
	if err != nil {
		t.Fatalf("Sub: unexpected error: %v", err)
	}
	if diff.String() != "7.25" {
		t.Fatalf("Sub: expected 7.25, got %s", diff.String())
	}
}

func TestMoney_Sub_NegativeResult(t *testing.T) {
	a, _ := NewMoney("5.00")
	b, _ := NewMoney("10.00")

	if _, err := a.Sub(b); !errors.Is(err, domainerrors.ErrNotEnoughBalance) {
		t.Fatalf("expected ErrNotEnoughBalance, got %v", err)
	}
}

func TestMoney_MulPercent(t *testing.T) {
	price, _ := NewMoney("99.99")
	total := price.Mul(3)
	if total.String() != "299.97" {
		t.Fatalf("Mul: expected 299.97, got %s", total.String())
	}

	amount, _ := NewMoney("200.00")
	credit := amount.Percent(10)
	if credit.String() != "20.00" {
		t.Fatalf("Percent: expected 20.00, got %s", credit.String())
	}
}

func TestMoney_JSONRoundTrip(t *testing.T) {
	m, _ := NewMoney("1234.5")

	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(raw) != "1234.50" {
		t.Fatalf("expected plain JSON number 1234.50, got %s", raw)
	}

	var back Money
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !back.d.Equal(m.d) {
		t.Fatalf("round-trip mismatch: got %s, want %s", back.String(), m.String())
	}
}

func TestMoney_ScanValueRoundTrip(t *testing.T) {
	m, _ := NewMoney("42.10")

	v, err := m.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}

	var back Money
	if err := back.Scan(v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if back.String() != "42.10" {
		t.Fatalf("expected 42.10, got %s", back.String())
	}
}

func TestMoney_MoneyFromCents(t *testing.T) {
	m, err := MoneyFromCents(12345)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.String() != "123.45" {
		t.Fatalf("expected 123.45, got %s", m.String())
	}
}
