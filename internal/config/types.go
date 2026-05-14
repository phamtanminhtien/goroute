package config

type Config struct {
	Server    ServerConfig     `json:"server"`
	Providers []ProviderConfig `json:"providers"`
}

type ServerConfig struct {
	Listen    string `json:"listen"`
	AuthToken string `json:"auth_token"`
}

type ProviderConfig struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	APIKey       string `json:"api_key,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Name         string `json:"name"`
}
