package models

import (
	"encoding/json/v2"
	"time"

	domainerrors "github.com/trottling/Telegram-Store/internal/domain/errors"
)

// Role — единственная роль пользователя вместо старых IsBanned/IsAdmin.
// RootAdmin ровно один — TELEGRAM_ROOT_ADMIN_ID из конфига, см. cmd/migrate.
type Role string

const (
	RoleBanned    Role = "banned"
	RoleUser      Role = "user"
	RoleAdmin     Role = "admin"
	RoleRootAdmin Role = "root_admin"
)

// User ключуется по TelegramID — отдельного auto-increment ID нет.
//
// balance/role неэкспортируемые: это единственные два поля, за которыми стоит
// реальный инвариант (баланс не уходит в минус, роль меняется только через
// разрешённые переходы) — менять их можно только через Debit/Credit/Ban/
// Unban/Promote/Demote, минуя прямое присваивание. Persistence (GORM) для
// этой структуры — отдельный тип userRecord в internal/repository/postgres,
// т.к. GORM не умеет писать в неэкспортируемые поля; MarshalJSON/UnmarshalJSON
// ниже нужны по той же причине для JSON-границ (admin_backend-ответы,
// internal/cache/redis — оба гоняют *User через encoding/json напрямую).
// Остальные поля (TelegramID, Username, ...) инвариантов не несут и остаются
// обычными экспортируемыми полями.
type User struct {
	TelegramID int64
	Username   string
	balance    Money
	role       Role
	CreatedAt  time.Time
	UpdatedAt  time.Time

	// ReferrerID — кто пригласил (Telegram ID), nil если пришёл не по
	// реф-ссылке или ссылка вела на уже существующего пользователя
	// (см. UserSrv.GetOrCreate — выставляется только при создании строки).
	ReferrerID *int64
	// ReferralsEnabled — админ может отключить этому юзеру начисления как
	// рефереру (см. AdminSrv.SetReferralsEnabled), по умолчанию включено.
	ReferralsEnabled bool

	// Language — язык интерфейса бота ("ru"/"en"), нормализованное значение
	// (см. bot/texts.Normalize). Определяется автоматически по Telegram-локали
	// при первом /start, дальше меняется только вручную через UserSrv.SetLanguage.
	Language string
}

// NewUser — новый пользователь (ветка создания в UserSrv.GetOrCreate). Баланс
// нулевой, роль — нулевое значение Role(""), а не RoleUser: так же, как и
// раньше, INSERT с пустой ролью отдаёт выставление default'а 'user' самому
// Postgres (см. userRecord).
func NewUser(telegramID int64, username, language string, referrerID *int64) *User {
	return &User{
		TelegramID:       telegramID,
		Username:         username,
		ReferralsEnabled: true,
		Language:         language,
		ReferrerID:       referrerID,
	}
}

// HydrateUser восстанавливает User из хранилища. Только для репозитория —
// в отличие от NewUser, здесь role/balance приходят уже готовыми из БД.
func HydrateUser(
	telegramID int64,
	username string,
	balance Money,
	role Role,
	createdAt, updatedAt time.Time,
	referrerID *int64,
	referralsEnabled bool,
	language string,
) *User {
	return &User{
		TelegramID:       telegramID,
		Username:         username,
		balance:          balance,
		role:             role,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		ReferrerID:       referrerID,
		ReferralsEnabled: referralsEnabled,
		Language:         language,
	}
}

func (u *User) Balance() Money {
	return u.balance
}

// Role — нужен репозиторию для маппинга в persistence-строку; сравнения по
// роли снаружи пакета всё равно должны идти через IsBanned/IsAdmin/IsRootAdmin.
func (u *User) Role() Role {
	return u.role
}

func (u *User) IsBanned() bool {
	return u.role == RoleBanned
}

func (u *User) IsAdmin() bool {
	return u.role == RoleAdmin || u.role == RoleRootAdmin
}

func (u *User) IsRootAdmin() bool {
	return u.role == RoleRootAdmin
}

// Ban переводит пользователя в RoleBanned, отбирая заодно права админа (роль
// одна на всех). Идемпотентно. actor — действующий админ: нельзя забанить
// себя или root admin, иначе некому будет вернуть права обратно.
func (u *User) Ban(actor *User) error {
	if u.TelegramID == actor.TelegramID {
		return domainerrors.ErrCannotBanSelf
	}
	if u.IsRootAdmin() {
		return domainerrors.ErrCannotBanRootAdmin
	}
	if u.IsBanned() {
		return nil
	}
	u.role = RoleBanned
	return nil
}

// Unban всегда возвращает роль User, а не ту, что была до бана — повторно
// выдать права нужно через Promote. Идемпотентно.
func (u *User) Unban() {
	if !u.IsBanned() {
		return
	}
	u.role = RoleUser
}

// Promote выдаёт права админа — только если actor сам root admin, иначе
// цепочка promote была бы неконтролируемой.
func (u *User) Promote(actor *User) error {
	if !actor.IsRootAdmin() {
		return domainerrors.ErrOnlyRootAdminCanPromote
	}
	if u.IsAdmin() {
		return domainerrors.ErrAlreadyAdmin
	}
	u.role = RoleAdmin
	return nil
}

// Demote снимает права админа. Нельзя снять с root admin или с себя.
func (u *User) Demote(actor *User) error {
	if u.TelegramID == actor.TelegramID {
		return domainerrors.ErrCannotRevokeSelf
	}
	if u.IsRootAdmin() {
		return domainerrors.ErrCannotRevokeRootAdmin
	}
	if !u.IsAdmin() {
		return domainerrors.ErrNotAdmin
	}
	u.role = RoleUser
	return nil
}

// Credit — начисление на баланс. Только для согласованности уже загруженного
// в память агрегата (например, чтобы вызвавший код увидел актуальный баланс
// без повторного чтения) — реальное начисление всегда идёт через атомарный
// UserRepository.UpdateBalance, см. его doc-комментарий.
func (u *User) Credit(amount Money) {
	u.balance = u.balance.Add(amount)
}

// Debit — списание с баланса, fast-fail на нехватку средств для уже
// загруженного агрегата. НЕ заменяет атомарный guard
// UserRepository.UpdateBalance (balance >= -delta) — тот единственный
// безопасен при конкуренции, этот метод лишь не даёт in-memory копии уйти в
// рассинхрон с ним.
func (u *User) Debit(amount Money) error {
	newBalance, err := u.balance.Sub(amount)
	if err != nil {
		return err
	}
	u.balance = newBalance
	return nil
}

// userJSON — теневая структура для JSON-границ (admin_backend-ответы,
// internal/cache/redis) — своя, а не теги на User: неэкспортируемые
// balance/role обычный encoding/json всё равно не увидит.
type userJSON struct {
	TelegramID       int64     `json:"telegram_id"`
	Username         string    `json:"username"`
	Balance          Money     `json:"balance"`
	Role             Role      `json:"role"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	ReferrerID       *int64    `json:"referrer_id,omitempty"`
	ReferralsEnabled bool      `json:"referrals_enabled"`
	Language         string    `json:"language"`
}

func (u *User) MarshalJSON() ([]byte, error) {
	return json.Marshal(userJSON{
		TelegramID:       u.TelegramID,
		Username:         u.Username,
		Balance:          u.balance,
		Role:             u.role,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
		ReferrerID:       u.ReferrerID,
		ReferralsEnabled: u.ReferralsEnabled,
		Language:         u.Language,
	})
}

func (u *User) UnmarshalJSON(data []byte) error {
	var j userJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	u.TelegramID = j.TelegramID
	u.Username = j.Username
	u.balance = j.Balance
	u.role = j.Role
	u.CreatedAt = j.CreatedAt
	u.UpdatedAt = j.UpdatedAt
	u.ReferrerID = j.ReferrerID
	u.ReferralsEnabled = j.ReferralsEnabled
	u.Language = j.Language
	return nil
}

// ReferralCredit — что начислено рефереру за покупку его реферала, для
// уведомления в боте (см. PurchaseService.Buy).
type ReferralCredit struct {
	ReferrerID int64
	Amount     Money
}
