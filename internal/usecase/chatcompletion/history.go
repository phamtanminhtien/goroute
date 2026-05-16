package chatcompletion

import (
	"fmt"
	"sync"
	"time"
)

const defaultHistoryLimit = 100

const (
	RequestStatusStarted  = "started"
	RequestStatusRetrying = "retrying"
	RequestStatusSuccess  = "success"
	RequestStatusError    = "error"
)

type HistoryStore interface {
	CreateRequestAttemptHistory(RequestAttemptHistory) (RequestAttemptHistory, error)
	UpdateRequestAttemptHistory(RequestAttemptHistory) error
	RecentRequestAttempts(limit int) ([]RequestAttemptHistory, error)
}

type RequestAttempt struct {
	ConnectionID   string    `json:"connection_id"`
	ConnectionName string    `json:"connection_name"`
	AttemptIndex   int       `json:"attempt_index"`
	Outcome        string    `json:"outcome"`
	ErrorCategory  string    `json:"error_category"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	WillFallback   bool      `json:"will_fallback"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at"`
	LatencyMillis  int64     `json:"latency_ms"`
}

type RequestAttemptHistory struct {
	HistoryID          int64            `json:"history_id,omitempty"`
	RequestID          string           `json:"request_id,omitempty"`
	RequestPath        string           `json:"request_path"`
	RequestedModel     string           `json:"requested_model"`
	ResolvedTarget     string           `json:"resolved_target"`
	ProviderID         string           `json:"provider_id"`
	ProviderName       string           `json:"provider_name"`
	Stream             bool             `json:"stream"`
	Status             string           `json:"status"`
	FinalStatus        string           `json:"final_status,omitempty"`
	FinalErrorCategory string           `json:"final_error_category,omitempty"`
	LastErrorCategory  string           `json:"last_error_category,omitempty"`
	LastErrorMessage   string           `json:"last_error_message,omitempty"`
	MessageCount       int              `json:"message_count"`
	ToolCount          int              `json:"tool_count"`
	AttemptCount       int              `json:"attempt_count"`
	LastConnectionID   string           `json:"last_connection_id,omitempty"`
	LastConnectionName string           `json:"last_connection_name,omitempty"`
	StartedAt          time.Time        `json:"started_at"`
	LastAttemptAt      time.Time        `json:"last_attempt_at,omitempty"`
	CompletedAt        time.Time        `json:"completed_at,omitempty"`
	UpdatedAt          time.Time        `json:"updated_at"`
	Attempts           []RequestAttempt `json:"attempts"`
}

type memoryHistoryStore struct {
	mu      sync.Mutex
	limit   int
	nextID  int64
	records []RequestAttemptHistory
}

func newMemoryHistoryStore(limit int) *memoryHistoryStore {
	if limit <= 0 {
		limit = defaultHistoryLimit
	}

	return &memoryHistoryStore{limit: limit}
}

func (s *memoryHistoryStore) CreateRequestAttemptHistory(record RequestAttemptHistory) (RequestAttemptHistory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	record.HistoryID = s.nextID
	s.records = append([]RequestAttemptHistory{record}, s.records...)
	if len(s.records) > s.limit {
		s.records = s.records[:s.limit]
	}

	return cloneHistoryRecord(record), nil
}

func (s *memoryHistoryStore) UpdateRequestAttemptHistory(record RequestAttemptHistory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for index, current := range s.records {
		if current.HistoryID == record.HistoryID {
			s.records[index] = cloneHistoryRecord(record)
			return nil
		}
	}

	return fmt.Errorf("request history %d not found", record.HistoryID)
}

func (s *memoryHistoryStore) RecentRequestAttempts(limit int) ([]RequestAttemptHistory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit > len(s.records) {
		limit = len(s.records)
	}

	out := make([]RequestAttemptHistory, limit)
	for i := 0; i < limit; i++ {
		out[i] = cloneHistoryRecord(s.records[i])
	}
	return out, nil
}

func cloneHistoryRecord(record RequestAttemptHistory) RequestAttemptHistory {
	record.Attempts = append([]RequestAttempt(nil), record.Attempts...)
	return record
}
