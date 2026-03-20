package main

import (
	"api-tester/backend/gateway"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.POST("/parse", gateway.ParseController)
	router.Run(":8080")
}
