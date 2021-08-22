package model

import "gorm.io/gorm"

type Transactions struct {
	gorm.Model
	UserID uint `json:"userid"`
	User   User

	ProductID uint `json:"productid"`
	Product   Product

	Amount uint64 `json:"amount"`
}

type ResponseTransaction struct {
	ID          uint   `json:"transactionid"`
	UserID      uint   `json:"userid"`
	Username    string `json:"username"`
	ProductID   uint   `json:"productid"`
	ProductName string `json:"productname"`
	Amount      uint64 `json:"amount"`
}
