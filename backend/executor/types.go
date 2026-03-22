package executor

import "time"

type QueueMessage struct {
	MessageID string         `json:"message_id"`
	BatchID   string         `json:"batch_id"`
	Priority  string         `json:"priority"`
	Action    string         `json:"action"`
	Payload   JobPayload     `json:"payload"`
	Metadata  MessageContext `json:"metadata"`
}

type MessageContext struct {
	OriginalPlanID any    `json:"original_plan_id,omitempty"`
	ProjectID      string `json:"project_id,omitempty"`
}

type JobPayload struct {
	Setup *RequestDetails `json:"setup,omitempty"`
	Test  RequestDetails  `json:"test"`
}

type RequestDetails struct {
	Method  string                 `json:"method"`
	URLPath string                 `json:"url_path"`
	Headers map[string]string      `json:"headers,omitempty"`
	Body    map[string]any         `json:"body,omitempty"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

type RequestExecutionResult struct {
	Method       string            `json:"method"`
	URL          string            `json:"url"`
	Headers      map[string]string `json:"headers,omitempty"`
	RequestBody  map[string]any    `json:"request_body,omitempty"`
	StatusCode   int               `json:"status_code"`
	ResponseBody string            `json:"response_body"`
	Error        string            `json:"error,omitempty"`
}

type JobExecutionResult struct {
	ProjectID    string                  `json:"project_id"`
	MessageID    string                  `json:"message_id"`
	BatchID      string                  `json:"batch_id"`
	OriginalPath string                  `json:"original_path"`
	OriginalVerb string                  `json:"original_method"`
	SetupResult  *RequestExecutionResult `json:"setup_result,omitempty"`
	TestResult   RequestExecutionResult  `json:"test_result"`
	StartedAt    time.Time               `json:"started_at"`
	CompletedAt  time.Time               `json:"completed_at"`
}

type ProjectExecutionReport struct {
	ProjectID       string               `json:"project_id"`
	QueueName       string               `json:"queue_name"`
	BaseURL         string               `json:"base_url"`
	StartedAt       time.Time            `json:"started_at"`
	CompletedAt     time.Time            `json:"completed_at"`
	JobsProcessed   int                  `json:"jobs_processed"`
	JobsFailed      int                  `json:"jobs_failed"`
	EndpointResults []JobExecutionResult `json:"endpoint_results"`
}

type Config struct {
	RabbitURL      string
	QueueName      string
	BaseURL        string
	ProjectID      string
	OutputFile     string
	PollInterval   time.Duration
	IdleTimeout    time.Duration
	RequestTimeout time.Duration
}
