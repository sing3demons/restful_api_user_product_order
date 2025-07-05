package logger

import "time"

type InComing struct {
	Headers map[string]string `json:"headers,omitempty"`
	Body    interface{}       `json:"body,omitempty"`
	Query   map[string]string `json:"query,omitempty"`
	Params  map[string]string `json:"params,omitempty"`
}

type Metadata struct {
	Topic         string `json:"topic,omitempty"`
	MessageValue  string `json:"messageValue,omitempty"`
	Key           string `json:"key,omitempty"`
	ConsumerGroup string `json:"consumerGroup,omitempty"`
	Broker        string `json:"broker,omitempty"`
	TraceId       string `json:"traceId,omitempty"`
	SpanId        string `json:"spanId,omitempty"`
	ClientIP      string `json:"clientIP,omitempty"`  // IP address of the client making the request
	UserAgent     string `json:"userAgent,omitempty"` // User-Agent header from the request
	Referer       string `json:"referer,omitempty"`   // Referer header from the request
	Method        string `json:"method,omitempty"`    // HTTP method of the request (GET, POST, etc.)
	URL           string `json:"url,omitempty"`       // Full URL of the request
	Source        string `json:"source,omitempty"`
}

type SequenceResult struct {
	Result  string `json:"result_code"`
	Desc    string `json:"result_desc"`
	ResTime int64  `json:"res_time,omitempty"` // Response time in milliseconds
}
type Sequence struct {
	Node    string           `json:"node"`
	Command string           `json:"command"`
	Result  []SequenceResult `json:"result"` // List of results for the sequence
}

type EventSummary struct {
	Event  string           `json:"event"`
	Result []SequenceResult `json:"result"` // List of results for the event
}

// LogDto struct
type LogDto struct {
	LogType          string     `json:"logType"` // "Detail"
	ServiceName      string     `json:"serviceName,omitempty"`
	Environment      string     `json:"environment,omitempty"`      // "production", "staging", "development"
	Component        string     `json:"component,omitempty"`        // "API", "Database", "Cache", etc.
	ComponentVersion string     `json:"componentVersion,omitempty"` // Version of the component, e.g., "1.0.0"
	StartTime        *time.Time `json:"-"`                          // Start time of the action

	Action            string     `json:"action"` // Action type, e.g., "HTTP_REQUEST", "DB_REQUEST", etc.
	ActionDescription string     `json:"actionDescription"`
	SubAction         string     `json:"subAction,omitempty"`
	Timestamp         *time.Time `json:"timestamp,omitempty"`

	Sequences []Sequence `json:"-"` // List of event tags

	Metadata Metadata `json:"metadata,omitzero"`
	Instance string   `json:"instance,omitempty"`
	Host     string   `json:"host,omitempty"`

	RequestId string `json:"requestId,omitempty"`
	SessionId string `json:"sessionId,omitempty"`
	TraceId   string `json:"traceId,omitempty"`
	SpanId    string `json:"spanId,omitempty"`

	ResponseTime int64                  `json:"responseTime,omitempty"`
	Level        string                 `json:"level,omitempty"` // "info", "warn", "error", "debug"
	Tags         []string               `json:"tags,omitempty"`  //
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
	Message      string                 `json:"message,omitempty"`

	AppResult           string `json:"appResult,omitempty"`
	AppResultCode       string `json:"appResultCode,omitempty"`
	AppResultHttpStatus string `json:"appResultHttpStatus,omitempty"`
	AppResultType       string `json:"appResultType,omitempty"`
	Severity            string `json:"severity,omitempty"`

	ThreadId    uint64 `json:"threadId,omitempty"` // ระบุ thread หรือ goroutine
	UseCase     string `json:"useCase,omitempty"`  // e.g., "user-authentication", "data-processing"
	UseCaseStep string `json:"useCaseStep,omitempty"`
}
