package executor

import (
	"api-tester/backend/helpers"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func Run(cfg Config) (ProjectExecutionReport, error) {
	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		return ProjectExecutionReport{}, fmt.Errorf("rabbitmq dial failed: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return ProjectExecutionReport{}, fmt.Errorf("rabbitmq channel failed: %w", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(cfg.QueueName, true, false, false, false, nil)
	if err != nil {
		return ProjectExecutionReport{}, fmt.Errorf("queue declare failed: %w", err)
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
			return ProjectExecutionReport{}, fmt.Errorf("queue read failed: %w", getErr)
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
			return ProjectExecutionReport{}, fmt.Errorf("ack failed: %w", ackErr)
		}
	}

	report.CompletedAt = time.Now().UTC()
	return report, nil
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
			setupID = helpers.ExtractResourceID(setupResult.ResponseBody)
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

	fullURL, urlErr := helpers.BuildURL(baseURL, resolvedPath, req.Params)
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
