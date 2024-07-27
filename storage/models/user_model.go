package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	TelegramID        int `json:"unique"`
	TwitterSubscribed bool
	TGSubscribed      bool
	JoinDate          string
	WalletConnected   bool
}

type WalletConnection struct {
	gorm.Model
	UserID int `gorm:"index"`
	Wallet string
}
