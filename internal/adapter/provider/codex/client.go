package codex

import (
	upstreamopenai "github.com/phamtanminhtien/goroute/internal/adapter/upstream/openai"
	"github.com/phamtanminhtien/goroute/internal/config"
)

type Client struct {
	*upstreamopenai.Client
}

func NewClient(provider config.ProviderConfig) *Client {
	return &Client{Client: upstreamopenai.NewClient(nil, provider)}
}
