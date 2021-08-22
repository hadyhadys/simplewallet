package model

import "gorm.io/gorm"

type Saldo struct {
	gorm.Model
	UserID  uint `json:"userid" gorm:"index:,unique"`
	User    User
	Balance uint64 `json:"balance"`
}

type ResponseSaldo struct {
	UserID  uint   `json:"userid"`
	Balance uint64 `json:"balance"`
}

type Topup struct {
	Amount uint64 `json:"amount"`
}
