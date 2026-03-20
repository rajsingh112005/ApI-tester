package gateway

import (
	"api-tester/backend/discovery"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type ParseRequest struct {
	FilePath string `json:"file_path"`
}

func ParseController(c *gin.Context) {
	var req ParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	if _, err := os.Stat(req.FilePath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("file not found: %s", req.FilePath),
		})
		return
	}
	res := discovery.Handler(req.FilePath)
	c.JSON(200, res)
}
