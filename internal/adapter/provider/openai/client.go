package openai

import (
	"net/http"

	upstreamopenai "github.com/phamtanminhtien/goroute/internal/adapter/upstream/openai"
	"github.com/phamtanminhtien/goroute/internal/config"
)

type Client struct {
	*upstreamopenai.Client
}

func NewClient(httpClient *http.Client, connection config.ConnectionConfig) *Client {
	return &Client{Client: upstreamopenai.NewClient(httpClient, connection)}
}
