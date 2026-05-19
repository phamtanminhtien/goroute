package airequestlog

type RunRecord struct {
	ID                  uint   `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	RequestID           string `json:"request_id" gorm:"column:request_id;not null;uniqueIndex"`
	Type                string `json:"type" gorm:"column:type;not null;index:idx_ai_request_runs_type_mode_created_at,priority:1"`
	RequestMode         string `json:"request_mode" gorm:"column:request_mode;not null;index:idx_ai_request_runs_type_mode_created_at,priority:2"`
	ProviderRequestMode string `json:"provider_request_mode" gorm:"column:provider_request_mode;not null;default:''"`
	Method              string `json:"method" gorm:"column:method;not null"`
	Path                string `json:"path" gorm:"column:path;not null"`
	RequestedModel      string `json:"requested_model" gorm:"column:requested_model;not null;default:'';index:idx_ai_request_runs_requested_model_created_at,priority:1"`
	ResolvedModel       string `json:"resolved_model" gorm:"column:resolved_model;not null;default:''"`
	ProviderID          string `json:"provider_id" gorm:"column:provider_id;not null;default:'';index:idx_ai_request_runs_provider_created_at,priority:1"`
	ProviderName        string `json:"provider_name" gorm:"column:provider_name;not null;default:''"`
	FinalConnectionID   string `json:"final_connection_id" gorm:"column:final_connection_id;not null;default:''"`
	FinalConnectionName string `json:"final_connection_name" gorm:"column:final_connection_name;not null;default:''"`
	AttemptCount        int    `json:"attempt_count" gorm:"column:attempt_count;not null;default:0"`
	StatusCode          int    `json:"status_code" gorm:"column:status_code;not null;default:0"`
	PromptTokens        int    `json:"prompt_tokens" gorm:"column:prompt_tokens;not null;default:0"`
	CompletionTokens    int    `json:"completion_tokens" gorm:"column:completion_tokens;not null;default:0"`
	TotalTokens         int    `json:"total_tokens" gorm:"column:total_tokens;not null;default:0"`
	ErrorType           string `json:"error_type" gorm:"column:error_type;not null;default:''"`
	ErrorMessage        string `json:"error_message" gorm:"column:error_message;not null;default:''"`
	FinalErrorCategory  string `json:"final_error_category" gorm:"column:final_error_category;not null;default:''"`
	StartedAt           int64  `json:"started_at" gorm:"column:started_at;not null;default:0"`
	CompletedAt         int64  `json:"completed_at" gorm:"column:completed_at;not null;default:0"`
	DurationMs          int64  `json:"duration_ms" gorm:"column:duration_ms;not null;default:0"`
	CreatedAt           int64  `json:"created_at" gorm:"column:created_at;not null;autoCreateTime:milli;index:idx_ai_request_runs_created_at;index:idx_ai_request_runs_type_mode_created_at,priority:3;index:idx_ai_request_runs_provider_created_at,priority:2;index:idx_ai_request_runs_requested_model_created_at,priority:2"`
	UpdatedAt           int64  `json:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime:milli"`
}

func (RunRecord) TableName() string {
	return "ai_request_runs"
}

type FlowRecord struct {
	ID                     uint    `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	RunID                  uint    `json:"run_id" gorm:"column:run_id;not null;index"`
	RequestID              string  `json:"request_id" gorm:"column:request_id;not null;index"`
	Type                   string  `json:"type" gorm:"column:type;not null;index:idx_ai_request_flows_type_mode_created_at,priority:1"`
	RequestMode            string  `json:"request_mode" gorm:"column:request_mode;not null;index:idx_ai_request_flows_type_mode_created_at,priority:2"`
	ProviderRequestMode    string  `json:"provider_request_mode" gorm:"column:provider_request_mode;not null;default:''"`
	Method                 string  `json:"method" gorm:"column:method;not null"`
	Path                   string  `json:"path" gorm:"column:path;not null"`
	Query                  string  `json:"query" gorm:"column:query;not null;default:''"`
	RemoteAddr             string  `json:"remote_addr" gorm:"column:remote_addr;not null;default:''"`
	UserAgent              string  `json:"user_agent" gorm:"column:user_agent;not null;default:''"`
	RequestHeaders         string  `json:"request_headers" gorm:"column:request_headers;not null;default:''"`
	RequestBody            string  `json:"request_body" gorm:"column:request_body;not null;default:''"`
	TranslatedRequestBody  string  `json:"translated_request_body" gorm:"column:translated_request_body;not null;default:''"`
	RequestedModel         string  `json:"requested_model" gorm:"column:requested_model;not null;default:''"`
	ResolvedModel          string  `json:"resolved_model" gorm:"column:resolved_model;not null;default:''"`
	ProviderID             string  `json:"provider_id" gorm:"column:provider_id;not null;default:''"`
	ProviderName           string  `json:"provider_name" gorm:"column:provider_name;not null;default:''"`
	AttemptTrace           string  `json:"attempt_trace" gorm:"column:attempt_trace;not null;default:''"`
	ResponseStatusCode     int     `json:"response_status_code" gorm:"column:response_status_code;not null;default:0"`
	ResponseHeaders        string  `json:"response_headers" gorm:"column:response_headers;not null;default:''"`
	ResponseBody           *string `json:"response_body" gorm:"column:response_body"`
	TranslatedResponseBody *string `json:"translated_response_body" gorm:"column:translated_response_body"`
	ErrorType              string  `json:"error_type" gorm:"column:error_type;not null;default:''"`
	ErrorMessage           string  `json:"error_message" gorm:"column:error_message;not null;default:''"`
	StartedAt              int64   `json:"started_at" gorm:"column:started_at;not null;default:0"`
	CompletedAt            int64   `json:"completed_at" gorm:"column:completed_at;not null;default:0"`
	DurationMs             int64   `json:"duration_ms" gorm:"column:duration_ms;not null;default:0"`
	CreatedAt              int64   `json:"created_at" gorm:"column:created_at;not null;autoCreateTime:milli;index:idx_ai_request_flows_created_at;index:idx_ai_request_flows_type_mode_created_at,priority:3"`
	UpdatedAt              int64   `json:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime:milli"`
}

func (FlowRecord) TableName() string {
	return "ai_request_flows"
}

type ThirdPartyRequestLogRecord struct {
	ID                  uint    `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	RunID               uint    `json:"run_id" gorm:"column:run_id;not null;index:idx_third_party_request_logs_run_attempt,priority:1"`
	RequestID           string  `json:"request_id" gorm:"column:request_id;not null;index"`
	Type                string  `json:"type" gorm:"column:type;not null"`
	RequestMode         string  `json:"request_mode" gorm:"column:request_mode;not null"`
	ProviderRequestMode string  `json:"provider_request_mode" gorm:"column:provider_request_mode;not null;default:''"`
	ProviderID          string  `json:"provider_id" gorm:"column:provider_id;not null;default:'';index:idx_third_party_request_logs_provider_created_at,priority:1"`
	ProviderName        string  `json:"provider_name" gorm:"column:provider_name;not null;default:''"`
	ConnectionID        string  `json:"connection_id" gorm:"column:connection_id;not null;default:''"`
	ConnectionName      string  `json:"connection_name" gorm:"column:connection_name;not null;default:''"`
	AttemptIndex        int     `json:"attempt_index" gorm:"column:attempt_index;not null;default:0;index:idx_third_party_request_logs_run_attempt,priority:2"`
	RequestMethod       string  `json:"request_method" gorm:"column:request_method;not null"`
	RequestURL          string  `json:"request_url" gorm:"column:request_url;not null;default:''"`
	RequestHeaders      string  `json:"request_headers" gorm:"column:request_headers;not null;default:''"`
	RequestBody         string  `json:"request_body" gorm:"column:request_body;not null;default:''"`
	ResponseStatusCode  int     `json:"response_status_code" gorm:"column:response_status_code;not null;default:0"`
	ResponseHeaders     string  `json:"response_headers" gorm:"column:response_headers;not null;default:''"`
	ResponseBody        *string `json:"response_body" gorm:"column:response_body"`
	ErrorType           string  `json:"error_type" gorm:"column:error_type;not null;default:''"`
	ErrorMessage        string  `json:"error_message" gorm:"column:error_message;not null;default:''"`
	StartedAt           int64   `json:"started_at" gorm:"column:started_at;not null;default:0"`
	CompletedAt         int64   `json:"completed_at" gorm:"column:completed_at;not null;default:0"`
	DurationMs          int64   `json:"duration_ms" gorm:"column:duration_ms;not null;default:0"`
	CreatedAt           int64   `json:"created_at" gorm:"column:created_at;not null;autoCreateTime:milli;index:idx_third_party_request_logs_provider_created_at,priority:2"`
	UpdatedAt           int64   `json:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime:milli"`
}

func (ThirdPartyRequestLogRecord) TableName() string {
	return "third_party_request_logs"
}

type RTKRecord struct {
	ID           uint   `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	RunID        uint   `json:"run_id" gorm:"column:run_id;not null;uniqueIndex"`
	RequestID    string `json:"request_id" gorm:"column:request_id;not null;index"`
	Applied      bool   `json:"applied" gorm:"column:applied;not null;default:false"`
	BytesBefore  int    `json:"bytes_before" gorm:"column:bytes_before;not null;default:0"`
	BytesAfter   int    `json:"bytes_after" gorm:"column:bytes_after;not null;default:0"`
	SavedBytes   int    `json:"saved_bytes" gorm:"column:saved_bytes;not null;default:0"`
	SavedPercent int    `json:"saved_percent" gorm:"column:saved_percent;not null;default:0"`
	FilterChain  string `json:"filter_chain" gorm:"column:filter_chain;not null;default:''"`
	HitCount     int    `json:"hit_count" gorm:"column:hit_count;not null;default:0"`
	FieldCount   int    `json:"field_count" gorm:"column:field_count;not null;default:0"`
	CreatedAt    int64  `json:"created_at" gorm:"column:created_at;not null;autoCreateTime:milli;index"`
	UpdatedAt    int64  `json:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime:milli"`
}

func (RTKRecord) TableName() string {
	return "rtk_records"
}
