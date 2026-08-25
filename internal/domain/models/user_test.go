package models

import (
	"testing"
	"time"

	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
)

func newTestUser(id TelegramID, role Role) *User {
	return HydrateUser(id, "user", Money{}, role, time.Now(), time.Now(), nil, true, "ru")
}

func TestUser_Ban_Self(t *testing.T) {
	u := newTestUser(1, RoleAdmin)
	if err := u.Ban(u); err != domainerrors.ErrCannotBanSelf {
		t.Fatalf("expected ErrCannotBanSelf, got %v", err)
	}
}

func TestUser_Ban_RootAdmin(t *testing.T) {
	target := newTestUser(1, RoleRootAdmin)
	actor := newTestUser(2, RoleAdmin)
	if err := target.Ban(actor); err != domainerrors.ErrCannotBanRootAdmin {
		t.Fatalf("expected ErrCannotBanRootAdmin, got %v", err)
	}
}

func TestUser_Ban_Idempotent(t *testing.T) {
	target := newTestUser(1, RoleBanned)
	actor := newTestUser(2, RoleAdmin)
	if err := target.Ban(actor); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !target.IsBanned() {
		t.Fatalf("expected target to stay banned")
	}
}

func TestUser_Ban_Success(t *testing.T) {
	target := newTestUser(1, RoleAdmin)
	actor := newTestUser(2, RoleAdmin)
	if err := target.Ban(actor); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !target.IsBanned() {
		t.Fatalf("expected target to be banned")
	}
	if target.IsAdmin() {
		t.Fatalf("ban must strip admin rights too (role is a single field)")
	}
}

func TestUser_Unban_AlwaysPlainUser(t *testing.T) {
	target := newTestUser(1, RoleBanned)
	target.Unban()
	if target.IsBanned() || target.IsAdmin() || target.IsRootAdmin() {
		t.Fatalf("expected plain user after unban, got role state banned=%v admin=%v root=%v", target.IsBanned(), target.IsAdmin(), target.IsRootAdmin())
	}
}

func TestUser_Promote_RequiresRootAdmin(t *testing.T) {
	target := newTestUser(1, RoleUser)
	actor := newTestUser(2, RoleAdmin)
	if err := target.Promote(actor); err != domainerrors.ErrOnlyRootAdminCanPromote {
		t.Fatalf("expected ErrOnlyRootAdminCanPromote, got %v", err)
	}
}

func TestUser_Promote_AlreadyAdmin(t *testing.T) {
	target := newTestUser(1, RoleAdmin)
	actor := newTestUser(2, RoleRootAdmin)
	if err := target.Promote(actor); err != domainerrors.ErrAlreadyAdmin {
		t.Fatalf("expected ErrAlreadyAdmin, got %v", err)
	}
}

func TestUser_Promote_Success(t *testing.T) {
	target := newTestUser(1, RoleUser)
	actor := newTestUser(2, RoleRootAdmin)
	if err := target.Promote(actor); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !target.IsAdmin() {
		t.Fatalf("expected target to become admin")
	}
}

func TestUser_Demote_Self(t *testing.T) {
	u := newTestUser(1, RoleAdmin)
	if err := u.Demote(u); err != domainerrors.ErrCannotRevokeSelf {
		t.Fatalf("expected ErrCannotRevokeSelf, got %v", err)
	}
}

func TestUser_Demote_RootAdmin(t *testing.T) {
	target := newTestUser(1, RoleRootAdmin)
	actor := newTestUser(2, RoleRootAdmin)
	if err := target.Demote(actor); err != domainerrors.ErrCannotRevokeRootAdmin {
		t.Fatalf("expected ErrCannotRevokeRootAdmin, got %v", err)
	}
}

func TestUser_Demote_NotAdmin(t *testing.T) {
	target := newTestUser(1, RoleUser)
	actor := newTestUser(2, RoleRootAdmin)
	if err := target.Demote(actor); err != domainerrors.ErrNotAdmin {
		t.Fatalf("expected ErrNotAdmin, got %v", err)
	}
}

func TestUser_Demote_Success(t *testing.T) {
	target := newTestUser(1, RoleAdmin)
	actor := newTestUser(2, RoleRootAdmin)
	if err := target.Demote(actor); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.IsAdmin() {
		t.Fatalf("expected target to lose admin rights")
	}
}

func TestUser_Debit_Insufficient(t *testing.T) {
	u := newTestUser(1, RoleUser)
	amount, _ := NewMoney("50.00")
	if err := u.Debit(amount); err != domainerrors.ErrNotEnoughBalance {
		t.Fatalf("expected ErrNotEnoughBalance, got %v", err)
	}
}

func TestUser_CreditDebit(t *testing.T) {
	u := newTestUser(1, RoleUser)
	credit, _ := NewMoney("100.00")
	u.Credit(credit)
	if u.Balance().String() != "100.00" {
		t.Fatalf("expected balance 100.00, got %s", u.Balance().String())
	}

	debit, _ := NewMoney("30.00")
	if err := u.Debit(debit); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Balance().String() != "70.00" {
		t.Fatalf("expected balance 70.00, got %s", u.Balance().String())
	}
}
