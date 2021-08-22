package model

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string `json:"username" gorm:"index:,unique"`
}

type UserRedis struct {
	Data []User `json:"data"`
}
