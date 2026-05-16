package config

type Config struct {
	Server ServerConfig `json:"server"`
}

type ServerConfig struct {
	Listen    string `json:"listen"`
	AuthToken string `json:"auth_token"`
	WebUIDir  string `json:"web_ui_dir,omitempty"`
}
