package models

import (
	"database/sql/driver"
	"encoding/json/v2"
	"fmt"

	"uuid"
)

// baseID — общая реализация поверх стандартного пакета uuid (Go 1.27+, тот
// же, которым уже пользуется dummy.go для invoiceID — это не
// github.com/google/uuid, а часть стандартной библиотеки, отдельной
// зависимости в go.mod не требует). Сам uuid.UUID не реализует
// sql.Scanner/driver.Valuer (только MarshalText/UnmarshalText) — Value/Scan
// здесь дописаны через общий хелпер uuidScan.
//
// ProductID/PurchaseID/... встраивают baseID анонимно, а не определяются как
// `type X uuid.UUID`: обычное переопределение типа (`type A B`) в Go не
// наследует методы B, так что пришлось бы дублировать String/MarshalJSON/
// Value/Scan на каждой из 6 сущностей. Встраивание — единственный штатный
// механизм Go, который одновременно даёт (а) несовместимые на уровне
// компилятора типы (ProductID нельзя перепутать с CategoryID) и (б) общую
// реализацию методов, написанную один раз. GORM это не путает: ProductID (и
// *ProductID) реализует driver.Valuer/sql.Scanner через промоушен методов
// baseID, а GORM проверяет именно это перед тем, как решить, разворачивать
// структуру в отдельные колонки — тот же принцип, что уже работает для Money
// (внутри тоже одно приватное поле + Value/Scan, тоже не разворачивается).
//
// Генерация — uuid.NewV7() (не New()/NewV4()): v7 монотонно возрастает по
// времени создания (гарантированно строго в рамках процесса), что заодно
// сохраняет естественный порядок вставки для B-tree индексов Postgres — в
// отличие от случайного v4, не фрагментирует их.
type baseID struct {
	u uuid.UUID
}

func newBaseID() baseID { return baseID{u: uuid.NewV7()} }

func parseBaseID(s string) (baseID, error) {
	u, err := uuid.Parse(s)
	return baseID{u: u}, err
}

func (id baseID) String() string { return id.u.String() }
func (id baseID) IsZero() bool   { return id.u == uuid.UUID{} }

func (id baseID) MarshalJSON() ([]byte, error) { return json.Marshal(id.String()) }

func (id *baseID) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return err
	}
	id.u = u
	return nil
}

func (id baseID) Value() (driver.Value, error) { return id.String(), nil }

func (id *baseID) Scan(v any) error {
	switch v := v.(type) {
	case nil:
		return nil
	case string:
		return id.u.UnmarshalText([]byte(v))
	case []byte:
		return id.u.UnmarshalText(v)
	default:
		return fmt.Errorf("uuid: unable to scan type %T", v)
	}
}

type ProductID struct{ baseID }
type ProductItemID struct{ baseID }
type PurchaseID struct{ baseID }
type CategoryID struct{ baseID }
type AdminLogID struct{ baseID }
type ReplenishmentID struct{ baseID }

// BatchID — не первичный ключ, а группирующее значение: все Purchase-строки
// одного вызова Buy() делят один BatchID. Тип всё равно нужен по той же
// причине, что и остальным: не перепутать его на вызове с ProductID/PurchaseID.
type BatchID struct{ baseID }

func NewProductID() ProductID { return ProductID{newBaseID()} }
func ParseProductID(s string) (ProductID, error) {
	b, err := parseBaseID(s)
	return ProductID{b}, err
}

func NewProductItemID() ProductItemID { return ProductItemID{newBaseID()} }
func ParseProductItemID(s string) (ProductItemID, error) {
	b, err := parseBaseID(s)
	return ProductItemID{b}, err
}

func NewPurchaseID() PurchaseID { return PurchaseID{newBaseID()} }
func ParsePurchaseID(s string) (PurchaseID, error) {
	b, err := parseBaseID(s)
	return PurchaseID{b}, err
}

func NewCategoryID() CategoryID { return CategoryID{newBaseID()} }
func ParseCategoryID(s string) (CategoryID, error) {
	b, err := parseBaseID(s)
	return CategoryID{b}, err
}

func NewAdminLogID() AdminLogID { return AdminLogID{newBaseID()} }
func ParseAdminLogID(s string) (AdminLogID, error) {
	b, err := parseBaseID(s)
	return AdminLogID{b}, err
}

func NewReplenishmentID() ReplenishmentID { return ReplenishmentID{newBaseID()} }
func ParseReplenishmentID(s string) (ReplenishmentID, error) {
	b, err := parseBaseID(s)
	return ReplenishmentID{b}, err
}

func NewBatchID() BatchID { return BatchID{newBaseID()} }
func ParseBatchID(s string) (BatchID, error) {
	b, err := parseBaseID(s)
	return BatchID{b}, err
}
