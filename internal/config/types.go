package config

type Config struct {
	Server      ServerConfig       `json:"server"`
	Connections []ConnectionConfig `json:"connections"`
}

type ServerConfig struct {
	Listen    string `json:"listen"`
	AuthToken string `json:"auth_token"`
	WebUIDir  string `json:"web_ui_dir,omitempty"`
}

type ConnectionConfig struct {
	ID                   string `json:"id"`
	ProviderID           string `json:"provider_id"`
	APIKey               string `json:"api_key,omitempty"`
	AccessToken          string `json:"access_token,omitempty"`
	RefreshToken         string `json:"refresh_token,omitempty"`
	TokenType            string `json:"token_type,omitempty"`
	ExpiresIn            int    `json:"expires_in,omitempty"`
	AccessTokenExpiresAt int64  `json:"access_token_expires_at,omitempty"`
	Name                 string `json:"name"`
}
