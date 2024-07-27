package models

import "gorm.io/gorm"

type Button struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"index"`
	Flag bool
}

type ButtonPress struct {
	gorm.Model
	UserID  int    `gorm:"index"`
	Button  string `gorm:"size:255"`
	Pressed bool
}
