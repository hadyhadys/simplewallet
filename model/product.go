package model

import "gorm.io/gorm"

type Product struct {
	gorm.Model
	ProductName string `json:"productname"`
	Price       uint64 `json:"price"`
}

type ProductRedis struct {
	Data []Product `json:"data"`
}
