package chatcompletion

import (
	"sync"
	"time"
)

const defaultHistoryLimit = 100

type RequestAttempt struct {
	ProviderID    string `json:"provider_id"`
	ProviderName  string `json:"provider_name"`
	AttemptIndex  int    `json:"attempt_index"`
	Outcome       string `json:"outcome"`
	ErrorCategory string `json:"error_category"`
	LatencyMillis int64  `json:"latency_ms"`
}

type RequestAttemptHistory struct {
	RequestID          string           `json:"request_id,omitempty"`
	RequestedModel     string           `json:"requested_model"`
	ResolvedTarget     string           `json:"resolved_target"`
	ProviderType       string           `json:"provider_type"`
	Stream             bool             `json:"stream"`
	FinalStatus        string           `json:"final_status"`
	FinalErrorCategory string           `json:"final_error_category,omitempty"`
	StartedAt          time.Time        `json:"started_at"`
	CompletedAt        time.Time        `json:"completed_at"`
	Attempts           []RequestAttempt `json:"attempts"`
}

type requestHistoryStore struct {
	mu      sync.Mutex
	limit   int
	records []RequestAttemptHistory
}

func newRequestHistoryStore(limit int) *requestHistoryStore {
	if limit <= 0 {
		limit = defaultHistoryLimit
	}

	return &requestHistoryStore{limit: limit}
}

func (s *requestHistoryStore) add(record RequestAttemptHistory) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = append([]RequestAttemptHistory{record}, s.records...)
	if len(s.records) > s.limit {
		s.records = s.records[:s.limit]
	}
}

func (s *requestHistoryStore) recent(limit int) []RequestAttemptHistory {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit > len(s.records) {
		limit = len(s.records)
	}

	out := make([]RequestAttemptHistory, limit)
	copy(out, s.records[:limit])
	return out
}
