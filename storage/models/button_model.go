package models

type Button struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"index"`
	Flag bool
}
