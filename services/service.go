package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"wallet/model"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/inconshreveable/log15"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB
var Log log15.Logger
var client *redis.Client

func SetupRedis() {
	client = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	pong, err := client.Ping().Result()
	if err != nil {
		Log.Debug("Getting error when create client redis", "error", err.Error())
		panic(err)
	}

	if pong == "PONG" {
		Log.Debug("Redis connected!")
	} else {
		Log.Debug("Redis not connected!")
	}
}

func Connect() {
	var product model.Product
	var user model.User
	var saldo model.Saldo
	var transaction model.Transactions
	var err error

	dsn := "hady:hadyhadys@(localhost:3307)/wallet?charset=utf8mb4&parseTime=True&loc=Local"
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		Log.Debug("Getting error when create connection to database", "error", err.Error())
		panic(err)
	}

	// //Migrate the schema
	DB.AutoMigrate(
		&user,
		&saldo,
		&product,
		&transaction,
	)
}

// ===========================================================================================================
// =========================================== USER ==========================================================

// GetAllUser to get all user
func GetAllUser(c *gin.Context) {
	var users []model.User
	var usersRedis model.UserRedis

	cache, err := client.Get("AllUser").Result()
	if err != nil && err != redis.Nil {
		Log.Debug("Getting error when get AllUser to redis", "error", err.Error())
	}

	if cache == "" {
		err = DB.Find(&users).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			Log.Debug("Getting error when find users", "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}

		if len(users) <= 0 {
			c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "Users not found!"})
			return
		}

		// Start input to redis
		usersRedis.Data = users
		json, err := json.Marshal(usersRedis)
		if err != nil {
			Log.Debug("Getting error when marshal cache AllUser", "error", err.Error())
		}

		err = client.Set("AllUser", json, 0).Err()
		if err != nil {
			Log.Debug("Getting error when set cache AllUser", "error", err.Error())
		}
	} else {
		Log.Debug("Cache exist!")
		err := json.Unmarshal([]byte(cache), &usersRedis)
		if err != nil {
			Log.Debug("Getting error when unmarshal cache AllUser", "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": usersRedis.Data})
}

// GetUserByID to get specific user
func GetUserByID(c *gin.Context) {
	var user model.User

	userID := c.Param("id")

	cache, err := client.Get("User" + userID).Result()
	if err != nil && err != redis.Nil {
		Log.Debug("Getting error when get user"+userID+" to redis", "error", err.Error())
	}

	if cache == "" {
		err = DB.Find(&user, userID).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			Log.Debug("Getting error when find users", "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}

		if user.ID == 0 {
			c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "ID not found!"})
			return
		}

		// Start input to redis
		json, err := json.Marshal(user)
		if err != nil {
			Log.Debug("Getting error when marshal cache User"+userID, "error", err.Error())
		}

		err = client.Set("User"+userID, json, 0).Err()
		if err != nil {
			Log.Debug("Getting error when set cache User"+userID, "error", err.Error())
		}
		Log.Debug("User"+userID, string(json))
	} else {
		Log.Debug("Cache exist!")
		err := json.Unmarshal([]byte(cache), &user)
		if err != nil {
			Log.Debug("Getting error when unmarshal cache User"+userID, "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": user})
}

// AddUser to add new user
func AddUser(c *gin.Context) {
	var user model.User
	var saldo model.Saldo
	var findUser model.User
	err := c.BindJSON(&user)
	if err != nil {
		Log.Debug("Getting error when bind request body", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if user.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Username field is required!"})
		return
	}

	// check exist user
	err = DB.Where("username = ?", user.Username).First(&findUser).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		Log.Debug("Getting error when check exist user", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if findUser.ID > 0 {
		c.JSON(http.StatusConflict, gin.H{"status": http.StatusConflict, "message": "User already exist!"})
		return
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			// Return err will rollback
			Log.Debug("Getting error when insert user to database", "error", err.Error())
			return err
		}

		saldo.UserID = user.ID

		if err := tx.Create(&saldo).Error; err != nil {
			Log.Debug("Getting error when insert saldo to database", "error", err.Error())
			return err
		}

		return nil

	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	// Remove Cache Redis
	client.Del("AllUser")

	c.JSON(http.StatusCreated, gin.H{"status": http.StatusCreated, "message": "User successfully added!", "resourceId": user.ID})
}

func UpdateUser(c *gin.Context) {
	var user model.User
	var req model.User

	userID := c.Param("id")

	// Check user exist
	err := DB.First(&user, userID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		Log.Debug("Getting error when check exist user", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if user.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "ID not found!"})
		return
	}

	err = c.BindJSON(&req)
	if err != nil {
		Log.Debug("Getting error when bind request body", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if req.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Username field is required!"})
		return
	} else {
		err = DB.Model(&user).Update("username", req.Username).Error
		if err != nil {
			Log.Debug("Getting error when update user data", "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
	}

	// Remove Cache Redis
	client.Del("AllUser")
	client.Del("User" + userID)

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "User successfully updated!"})
}

// DeleteUser for delete user
func DeleteUser(c *gin.Context) {
	var user model.User
	userID := c.Param("id")

	// Check exist user
	err := DB.First(&user, userID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		Log.Debug("Getting error when check exist user", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if user.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "ID not found!"})
		return
	}

	err = DB.Delete(&user).Error
	if err != nil {
		Log.Debug("Getting error when delete user", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	// Remove Cache Redis
	client.Del("AllUser")
	client.Del("User" + userID)

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "User deleted successfully!"})
}

// ===========================================================================================================
// =========================================== PRODUCT========================================================

// GetAllProduct to get all product
func GetAllProduct(c *gin.Context) {
	var products []model.Product
	var productRedis model.ProductRedis

	cache, err := client.Get("AllProduct").Result()
	if err != nil && err != redis.Nil {
		Log.Debug("Getting error when get AllProduct to redis", "error", err.Error())
	}

	if cache == "" {
		err = DB.Find(&products).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			Log.Debug("Getting error when check exist products", "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}

		if len(products) <= 0 {
			c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "Products not found!"})
			return
		}

		// Start input to redis
		productRedis.Data = products
		json, err := json.Marshal(productRedis)
		if err != nil {
			Log.Debug("Getting error when marshal cache AllProduct", "error", err.Error())
		}

		err = client.Set("AllProduct", json, 0).Err()
		if err != nil {
			Log.Debug("Getting error when set cache AllProduct", "error", err.Error())
		}
	} else {
		Log.Debug("Cache exist!")
		err := json.Unmarshal([]byte(cache), &productRedis)
		if err != nil {
			Log.Debug("Getting error when unmarshal cache AllProduct", "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": productRedis.Data})
}

// GetProductByID to get specific product
func GetProductByID(c *gin.Context) {
	var product model.Product

	productID := c.Param("id")

	cache, err := client.Get("Product" + productID).Result()
	if err != nil && err != redis.Nil {
		Log.Debug("Getting error when get Product"+productID+" to redis", "error", err.Error())
	}

	if cache == "" {
		err = DB.Find(&product, productID).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			Log.Debug("Getting error when check exist product", "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}

		if product.ID == 0 {
			c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "ID not found!"})
			return
		}

		// Start input to redis
		json, err := json.Marshal(product)
		if err != nil {
			Log.Debug("Getting error when marshal cache Product"+productID, "error", err.Error())
		}

		err = client.Set("Product"+productID, json, 0).Err()
		if err != nil {
			Log.Debug("Getting error when set cache Product"+productID, "error", err.Error())
		}
		Log.Debug("Product"+productID, string(json))
	} else {
		Log.Debug("Cache exist!")
		err := json.Unmarshal([]byte(cache), &product)
		if err != nil {
			Log.Debug("Getting error when unmarshal cache Product"+productID, "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": product})
}

// AddProduct to add new product
func AddProduct(c *gin.Context) {
	var product model.Product
	var findProduct model.Product

	err := c.BindJSON(&product)
	if err != nil {
		Log.Debug("Getting error when bind request body", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if product.ProductName == "" || product.Price == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Field productname and price is required!"})
		return
	}

	// check exist product
	err = DB.Where("product_name = ?", product.ProductName).First(&findProduct).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		Log.Debug("Getting error when check exist product", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if findProduct.ID > 0 {
		c.JSON(http.StatusConflict, gin.H{"status": http.StatusConflict, "message": "Product already exist!"})
		return
	} else {
		err = DB.Save(&product).Error
		if err != nil {
			Log.Debug("Getting error when add product to database", "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
	}

	// Remove Cache Redis
	client.Del("AllProduct")

	c.JSON(http.StatusCreated, gin.H{"status": http.StatusCreated, "message": "Product successfully added!", "resourceId": product.ID})
}

// UpdateProduct to update product
func UpdateProduct(c *gin.Context) {
	var product model.Product
	var req model.Product

	productID := c.Param("id")

	// Check product exist
	err := DB.First(&product, productID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		Log.Debug("Getting error when check exist product", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if product.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "ID not found!"})
		return
	}

	err = c.BindJSON(&req)
	if err != nil {
		Log.Debug("Getting error when bind request body", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if req.ProductName == "" || req.Price == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Field productname and price is required!"})
		return
	} else {
		product.ProductName = req.ProductName
		product.Price = req.Price

		err = DB.Model(&product).Updates(product).Error
		if err != nil {
			Log.Debug("Getting error when update product data", "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
	}

	// Remove Cache Redis
	client.Del("AllProduct")
	client.Del("Product" + productID)

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "Product successfully updated!"})
}

// PartialUpdateProduct to partial update product
func PartialUpdateProduct(c *gin.Context) {
	var product model.Product
	var req model.Product

	productID := c.Param("id")

	// Check product exist
	err := DB.First(&product, productID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		Log.Debug("Getting error when check exist product", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if product.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "ID not found!"})
		return
	}

	err = c.BindJSON(&req)
	if err != nil {
		Log.Debug("Getting error when bind request body", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if req.ProductName == "" && req.Price == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Field productname or price is required!"})
		return
	}

	if req.ProductName != "" {
		product.ProductName = req.ProductName
	}

	if req.Price > 0 {
		product.Price = req.Price
	}

	err = DB.Model(&product).Updates(product).Error
	if err != nil {
		Log.Debug("Getting error when update product data", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	// Remove Cache Redis
	client.Del("AllProduct")
	client.Del("Product" + productID)

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "Product successfully updated!"})
}

// DeleteProduct for delete product
func DeleteProduct(c *gin.Context) {
	var product model.Product
	productID := c.Param("id")

	// Check exist product
	err := DB.First(&product, productID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		Log.Debug("Getting error when check exist product", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if product.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "ID not found!"})
		return
	}

	err = DB.Delete(&product).Error
	if err != nil {
		Log.Debug("Getting error when delete product", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	// Remove Cache Redis
	client.Del("AllProduct")
	client.Del("Product" + productID)

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "Product deleted successfully!"})
}

// ===========================================================================================================
// ============================================ SALDO ========================================================

func Topup(c *gin.Context) {
	var saldo model.Saldo
	var req model.Topup

	userID := c.Param("id")

	// check exist user's saldo
	err := DB.Where("user_id = ?", userID).First(&saldo).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		Log.Debug("Getting error when check exist user's saldo", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if saldo.UserID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "User not found!"})
		return
	}

	err = c.BindJSON(&req)
	if err != nil {
		Log.Debug("Getting error when bind request body", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if req.Amount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Field amount is required!"})
		return
	}

	saldo.Balance = saldo.Balance + req.Amount

	err = DB.Model(&saldo).Update("balance", saldo.Balance).Error
	if err != nil {
		Log.Debug("Getting error when update balance user's saldo", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	// Remove Cache Redis
	client.Del("SaldoUser" + userID)

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "Topup saldo successfully!"})

}

func CheckSaldo(c *gin.Context) {
	var saldo model.Saldo
	var response model.ResponseSaldo

	userID := c.Param("id")

	cache, err := client.Get("SaldoUser" + userID).Result()
	if err != nil && err != redis.Nil {
		Log.Debug("Getting error when get SaldoUser"+userID+" to redis", "error", err.Error())
	}

	if cache == "" {
		// check exist user
		err = DB.Where("user_id = ?", userID).First(&saldo).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			Log.Debug("Getting error when check exist user's saldo", "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}

		if saldo.UserID == 0 {
			c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "User not found!"})
			return
		}

		response.UserID = saldo.UserID
		response.Balance = saldo.Balance

		// Start input to redis
		json, err := json.Marshal(response)
		if err != nil {
			Log.Debug("Getting error when marshal cache SaldoUser"+userID, "error", err.Error())
		}

		err = client.Set("SaldoUser"+userID, json, 0).Err()
		if err != nil {
			Log.Debug("Getting error when set cache SaldoUser"+userID, "error", err.Error())
		}
		Log.Debug("SaldoUser"+userID, string(json))
	} else {
		Log.Debug("Cache exist!")
		err := json.Unmarshal([]byte(cache), &response)
		if err != nil {
			Log.Debug("Getting error when unmarshal cache SaldoUser"+userID, "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": response})
}

// ===========================================================================================================
// ========================================== Transaction ====================================================

func Transaction(c *gin.Context) {
	var user model.User
	var product model.Product
	var saldo model.Saldo
	var transaction model.Transactions

	err := c.BindJSON(&transaction)
	if err != nil {
		Log.Debug("Getting error when bind request body", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if transaction.UserID == 0 || transaction.ProductID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Invalid request body!"})
		return
	}

	// check exist user
	err = DB.Find(&user, transaction.UserID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		Log.Debug("Getting error when check exist user", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}
	if user.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "User not found!"})
		return
	}

	// check exist product
	err = DB.Find(&product, transaction.ProductID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		Log.Debug("Getting error when check exist product", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}
	if product.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "Product not found!"})
		return
	}

	// get user's saldo
	err = DB.Where("user_id = ?", user.ID).First(&saldo).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		Log.Debug("Getting error when get user's saldo", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}

	if saldo.Balance < product.Price {
		c.JSON(http.StatusNotAcceptable, gin.H{"status": http.StatusNotAcceptable, "message": "Not enough saldo!"})
		return
	} else {
		// Credit balance
		saldo.Balance = saldo.Balance - product.Price
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		transaction.Amount = product.Price
		if err := tx.Create(&transaction).Error; err != nil {
			// Return err will rollback
			Log.Debug("Getting error when insert transaction to database", "error", err.Error())
			return err
		}

		if err := tx.Model(&saldo).Update("balance", saldo.Balance).Error; err != nil {
			// Return err will rollback
			Log.Debug("Getting error when update user's balance saldo", "error", err.Error())
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
		return
	}
	// Remove Cache Redis
	client.Del("SaldoUser" + fmt.Sprint(user.ID))
	client.Del("AllTransaction")

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "Transaction successfully!", "resourceId": transaction.ID})
}

func GetAllTransaction(c *gin.Context) {
	var transactions []model.Transactions
	var response []model.ResponseTransaction

	cache, err := client.Get("AllTransaction").Result()
	if err != nil && err != redis.Nil {
		Log.Debug("Getting error when get AllTransaction to redis", "error", err.Error())
	}

	if cache == "" {
		err := DB.Model(&transactions).
			Select("transactions.id, transactions.user_id, users.username, transactions.product_id, products.product_name, transactions.amount").
			Joins("left join users on users.id = transactions.user_id").
			Joins("left join products on products.id = transactions.product_id").Scan(&response).Error

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			Log.Debug("Getting error when get transaction data", "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}

		// Start input to redis
		json, err := json.Marshal(response)
		if err != nil {
			Log.Debug("Getting error when marshal cache AllTransaction", "error", err.Error())
		}

		err = client.Set("AllTransaction", json, 0).Err()
		if err != nil {
			Log.Debug("Getting error when set cache AllTransaction", "error", err.Error())
		}
		Log.Debug("AllTransaction", string(json))
	} else {
		Log.Debug("Cache exist!")
		err := json.Unmarshal([]byte(cache), &response)
		if err != nil {
			Log.Debug("Getting error when unmarshal cache AllTransaction", "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": response})
}
