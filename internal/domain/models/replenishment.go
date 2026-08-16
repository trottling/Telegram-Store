package models

type Replenishment struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`
}
