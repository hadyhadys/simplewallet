package main

import (
	"os"
	"time"
	"wallet/services"

	"github.com/gin-gonic/gin"
	"github.com/inconshreveable/log15"
)

func init() {

	// Setup gin logger
	now := time.Now()
	logFile, _ := os.Create("log/request-" + now.Format("2006-01-02") + ".log")
	logErrFile, _ := os.Create("log/error-" + now.Format("2006-01-02") + ".log")
	gin.DefaultWriter = logFile
	gin.DefaultErrorWriter = logErrFile

	// Set Production Mode
	gin.SetMode(gin.ReleaseMode)

	if gin.Mode() == "debug" {
		hfilter := log15.LvlFilterHandler(log15.LvlDebug, log15.CallerFileHandler(log15.StreamHandler(os.Stdout, log15.JsonFormat())))
		log15.Root().SetHandler(log15.MultiHandler(hfilter))
	} else {
		hfilter := log15.LvlFilterHandler(log15.LvlDebug, log15.CallerFileHandler(log15.StreamHandler(gin.DefaultErrorWriter, log15.JsonFormat())))
		log15.Root().SetHandler(log15.MultiHandler(hfilter))
	}

	services.Log = log15.New(log15.Ctx{"module": "wallet"})

	// Create connection to redis
	services.SetupRedis()

	// Create connection to database
	services.Connect()
}

func main() {
	router := gin.Default()

	userAPI := router.Group("/api/v1/user")
	{
		userAPI.POST("/", services.AddUser)
		userAPI.GET("/", services.GetAllUser)
		userAPI.GET("/:id", services.GetUserByID)
		userAPI.PUT("/:id", services.UpdateUser)
		userAPI.DELETE("/:id", services.DeleteUser)
	}

	productAPI := router.Group("/api/v1/product")
	{
		productAPI.POST("/", services.AddProduct)
		productAPI.GET("/", services.GetAllProduct)
		productAPI.GET("/:id", services.GetProductByID)
		productAPI.PUT("/:id", services.UpdateProduct)
		productAPI.PATCH("/:id", services.PartialUpdateProduct)
		productAPI.DELETE("/:id", services.DeleteProduct)
	}

	saldoAPI := router.Group("/api/v1/saldo")
	{
		saldoAPI.GET("/:id", services.CheckSaldo)
		saldoAPI.PATCH("/:id", services.Topup)
	}

	transactionAPI := router.Group("/api/v1/transaction")
	{
		transactionAPI.GET("/", services.GetAllTransaction)
		transactionAPI.POST("/", services.Transaction)
	}

	router.Run()
}
