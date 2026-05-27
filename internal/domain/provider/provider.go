package provider

type AuthType string

const (
	AuthTypeOAuth  AuthType = "oauth"
	AuthTypeAPIKey AuthType = "api_key"
)

type AdapterType string

const (
	AdapterTypeOpenAICompatible    AdapterType = "openai_compatible"
	AdapterTypeAnthropicCompatible AdapterType = "anthropic_compatible"
)

type Provider struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	AuthType     AuthType    `json:"auth_type"`
	Category     string      `json:"category"`
	AdapterType  AdapterType `json:"adapter_type,omitempty"`
	BaseURL      string      `json:"base_url,omitempty"`
	DefaultModel string      `json:"default_model"`
	Models       []Model     `json:"models"`
}

type Record struct {
	ID           string      `json:"id" gorm:"column:id;primaryKey"`
	Name         string      `json:"name" gorm:"column:name;not null"`
	AuthType     AuthType    `json:"auth_type" gorm:"column:auth_type;not null"`
	Category     string      `json:"category" gorm:"column:category;not null"`
	AdapterType  AdapterType `json:"adapter_type" gorm:"column:adapter_type;not null"`
	BaseURL      string      `json:"base_url" gorm:"column:base_url;not null"`
	DefaultModel string      `json:"default_model" gorm:"column:default_model;not null"`
	CreatedAt    int64       `json:"-" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt    int64       `json:"-" gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (Record) TableName() string {
	return "providers"
}

func (r Record) Provider() Provider {
	return Provider{
		ID:           r.ID,
		Name:         r.Name,
		AuthType:     r.AuthType,
		Category:     r.Category,
		AdapterType:  r.AdapterType,
		BaseURL:      r.BaseURL,
		DefaultModel: r.DefaultModel,
	}
}

type Model struct {
	ID                       string  `json:"id"`
	Name                     string  `json:"name"`
	Description              string  `json:"description"`
	InputPricePerMillionUSD  float64 `json:"input_price_per_million_usd"`
	OutputPricePerMillionUSD float64 `json:"output_price_per_million_usd"`
	Source                   string  `json:"source"`
}

type ModelRecord struct {
	ID                       string  `json:"id" gorm:"primaryKey"`
	ProviderID               string  `json:"provider_id" gorm:"index;not null"`
	Name                     string  `json:"name"`
	Description              string  `json:"description"`
	InputPricePerMillionUSD  float64 `json:"input_price_per_million_usd"`
	OutputPricePerMillionUSD float64 `json:"output_price_per_million_usd"`
}

func (ModelRecord) TableName() string {
	return "provider_models"
}
