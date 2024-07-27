package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	TelegramID        int64 `json:"unique"`
	TwitterSubscribed bool
	TGSubscribed      bool
	JoinDate          string
}
