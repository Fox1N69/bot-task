package storage

import "gorm.io/gorm"

type Storage interface {
}
type storage struct {
	db *gorm.DB
}

func NewSotrage(db *gorm.DB) Storage {
	return &storage{
		db: db,
	}
}
