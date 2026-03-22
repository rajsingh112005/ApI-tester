package gateway

import (
	"api-tester/backend/executor"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type RunExecutorRequest struct {
	BaseURL string `json:"base_url"`
}

func RunExecutorController(c *gin.Context) {
	var req RunExecutorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	if strings.TrimSpace(req.BaseURL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "base_url is required"})
		return
	}

	cfg := executor.Config{
		RabbitURL:      envOrDefault("RABBIT_URL", "amqp://guest:guest@localhost:5672/"),
		QueueName:      envOrDefault("RABBIT_QUEUE", "test_jobs_queue"),
		BaseURL:        req.BaseURL,
		ProjectID:      envOrDefault("EXECUTOR_PROJECT_ID", "default_project"),
		PollInterval:   time.Second,
		IdleTimeout:    12 * time.Second,
		RequestTimeout: 20 * time.Second,
	}

	report, err := executor.Run(cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
