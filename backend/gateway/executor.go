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
		OutputFile:     envOrDefault("EXECUTOR_OUTPUT_FILE", ""),
		PollInterval:   time.Second,
		IdleTimeout:    12 * time.Second,
		RequestTimeout: 20 * time.Second,
	}

	go func(config executor.Config) {
		_ = executor.Run(config)
	}(cfg)

	c.JSON(http.StatusAccepted, gin.H{
		"status":     "started",
		"project_id": cfg.ProjectID,
		"base_url":   cfg.BaseURL,
		"queue_name": cfg.QueueName,
	})
}

func defaultString(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
