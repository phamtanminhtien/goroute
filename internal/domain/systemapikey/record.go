package systemapikey

type Record struct {
	ID                     string        `json:"id" gorm:"column:id;primaryKey"`
	Name                   string        `json:"name" gorm:"column:name;not null"`
	Key                    string        `json:"key" gorm:"column:key;not null;uniqueIndex"`
	Enabled                bool          `json:"enabled" gorm:"column:enabled;not null;default:true"`
	RequestsPerMinuteLimit *int          `json:"requests_per_minute_limit" gorm:"column:requests_per_minute_limit"`
	DailyTokenLimit        *int          `json:"daily_token_limit" gorm:"column:daily_token_limit"`
	MonthlyTokenLimit      *int          `json:"monthly_token_limit" gorm:"column:monthly_token_limit"`
	Usage                  *UsageSummary `json:"usage,omitempty" gorm:"-"`
	CreatedAt              int64         `json:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt              int64         `json:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime"`
	LastUsedAt             int64         `json:"last_used_at" gorm:"column:last_used_at;not null;default:0"`
}

func (Record) TableName() string {
	return "system_api_keys"
}

type UsageSummary struct {
	CurrentMinuteRequests int        `json:"current_minute_requests"`
	DailyTokens           TokenUsage `json:"daily_tokens"`
	MonthlyTokens         TokenUsage `json:"monthly_tokens"`
	RateLimitReached      bool       `json:"rate_limit_reached"`
	DailyLimitReached     bool       `json:"daily_limit_reached"`
	MonthlyLimitReached   bool       `json:"monthly_limit_reached"`
}

type TokenUsage struct {
	Used      int  `json:"used"`
	Limit     *int `json:"limit"`
	Remaining *int `json:"remaining"`
}

type RequestEvent struct {
	ID             uint   `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	SystemAPIKeyID string `json:"system_api_key_id" gorm:"column:system_api_key_id;not null;index:idx_system_api_key_request_events_key_created_at,priority:1"`
	RequestID      string `json:"request_id" gorm:"column:request_id;not null;default:'';index"`
	Path           string `json:"path" gorm:"column:path;not null;default:''"`
	CreatedAt      int64  `json:"created_at" gorm:"column:created_at;not null;autoCreateTime:milli;index:idx_system_api_key_request_events_key_created_at,priority:2"`
}

func (RequestEvent) TableName() string {
	return "system_api_key_request_events"
}
