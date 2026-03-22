package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func Run(cfg Config) error {
	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		return fmt.Errorf("rabbitmq dial failed: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq channel failed: %w", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(cfg.QueueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("queue declare failed: %w", err)
	}

	report := ProjectExecutionReport{
		ProjectID:       cfg.ProjectID,
		QueueName:       cfg.QueueName,
		BaseURL:         strings.TrimRight(cfg.BaseURL, "/"),
		StartedAt:       time.Now().UTC(),
		EndpointResults: make([]JobExecutionResult, 0),
	}

	client := &http.Client{Timeout: cfg.RequestTimeout}
	lastMessageAt := time.Now()

	for {
		msg, ok, getErr := ch.Get(cfg.QueueName, false)
		if getErr != nil {
			return fmt.Errorf("queue read failed: %w", getErr)
		}

		if !ok {
			if report.JobsProcessed > 0 && time.Since(lastMessageAt) >= cfg.IdleTimeout {
				break
			}
			time.Sleep(cfg.PollInterval)
			continue
		}

		lastMessageAt = time.Now()

		result, processErr := processMessage(cfg, client, msg.Body)
		if processErr != nil {
			report.JobsFailed++
			_ = msg.Ack(false)
			continue
		}

		report.JobsProcessed++
		if result.TestResult.Error != "" || result.TestResult.StatusCode >= 400 {
			report.JobsFailed++
		}
		report.EndpointResults = append(report.EndpointResults, result)

		if ackErr := msg.Ack(false); ackErr != nil {
			return fmt.Errorf("ack failed: %w", ackErr)
		}
	}

	report.CompletedAt = time.Now().UTC()

	if err := writeReport(report, cfg.OutputFile); err != nil {
		return err
	}

	return nil
}

func processMessage(cfg Config, client *http.Client, body []byte) (JobExecutionResult, error) {
	var message QueueMessage
	if err := json.Unmarshal(body, &message); err != nil {
		return JobExecutionResult{}, fmt.Errorf("invalid message payload: %w", err)
	}

	projectID := message.Metadata.ProjectID
	if projectID == "" {
		projectID = cfg.ProjectID
	}

	result := JobExecutionResult{
		ProjectID:    projectID,
		MessageID:    message.MessageID,
		BatchID:      message.BatchID,
		OriginalVerb: strings.ToUpper(message.Payload.Test.Method),
		OriginalPath: message.Payload.Test.URLPath,
		StartedAt:    time.Now().UTC(),
	}

	setupID := ""
	if message.Payload.Setup != nil {
		setupResult := executeRequest(client, cfg.BaseURL, *message.Payload.Setup, "")
		result.SetupResult = &setupResult
		if setupResult.Error == "" {
			setupID = extractResourceID(setupResult.ResponseBody)
		}
	}

	testResult := executeRequest(client, cfg.BaseURL, message.Payload.Test, setupID)
	result.TestResult = testResult
	result.CompletedAt = time.Now().UTC()

	return result, nil
}

func executeRequest(client *http.Client, baseURL string, req RequestDetails, setupID string) RequestExecutionResult {
	resolvedPath := req.URLPath
	if setupID != "" {
		resolvedPath = strings.ReplaceAll(resolvedPath, "{{setup_id}}", setupID)
	}

	fullURL, urlErr := buildURL(baseURL, resolvedPath, req.Params)
	if urlErr != nil {
		return RequestExecutionResult{
			Method: req.Method,
			URL:    resolvedPath,
			Error:  urlErr.Error(),
		}
	}

	var bodyReader io.Reader
	if req.Body != nil {
		payload, marshalErr := json.Marshal(req.Body)
		if marshalErr != nil {
			return RequestExecutionResult{Method: req.Method, URL: fullURL, Error: marshalErr.Error()}
		}
		bodyReader = bytes.NewReader(payload)
	}

	httpReq, err := http.NewRequest(strings.ToUpper(req.Method), fullURL, bodyReader)
	if err != nil {
		return RequestExecutionResult{Method: req.Method, URL: fullURL, Error: err.Error()}
	}

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	httpResp, err := client.Do(httpReq)
	if err != nil {
		return RequestExecutionResult{
			Method:      req.Method,
			URL:         fullURL,
			Headers:     req.Headers,
			RequestBody: req.Body,
			Error:       err.Error(),
		}
	}
	defer httpResp.Body.Close()

	respBytes, _ := io.ReadAll(httpResp.Body)

	return RequestExecutionResult{
		Method:       req.Method,
		URL:          fullURL,
		Headers:      req.Headers,
		RequestBody:  req.Body,
		StatusCode:   httpResp.StatusCode,
		ResponseBody: string(respBytes),
	}
}

func buildURL(baseURL string, path string, params map[string]interface{}) (string, error) {
	base := strings.TrimRight(baseURL, "/")
	rel := path
	if !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}

	u, err := url.Parse(base + rel)
	if err != nil {
		return "", err
	}

	if len(params) == 0 {
		return u.String(), nil
	}

	query := u.Query()
	for key, value := range params {
		query.Set(key, fmt.Sprintf("%v", value))
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func extractResourceID(responseBody string) string {
	if strings.TrimSpace(responseBody) == "" {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(responseBody), &payload); err != nil {
		return ""
	}

	candidateKeys := []string{"id", "_id", "user_id", "resource_id"}
	for _, key := range candidateKeys {
		if value, ok := payload[key]; ok {
			return fmt.Sprintf("%v", value)
		}
	}
	if nested, ok := payload["data"].(map[string]any); ok {
		for _, key := range candidateKeys {
			if value, exists := nested[key]; exists {
				return fmt.Sprintf("%v", value)
			}
		}
	}

	return ""
}

func writeReport(report ProjectExecutionReport, outputPath string) error {
	if outputPath == "" {
		outputPath = fmt.Sprintf("executor-report-%s.json", sanitizeName(report.ProjectID))
	}

	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize report: %w", err)
	}

	if err := os.WriteFile(outputPath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	fmt.Printf("Executor completed project %s with %d jobs (%d failed). Report: %s\n", report.ProjectID, report.JobsProcessed, report.JobsFailed, outputPath)
	return nil
}

func sanitizeName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-")
	return replacer.Replace(trimmed)
}
