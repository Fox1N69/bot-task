package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	TelegramID        int `json:"unique"`
	TwitterSubscribed bool
	TGSubscribed      bool
	JoinDate          string
}
