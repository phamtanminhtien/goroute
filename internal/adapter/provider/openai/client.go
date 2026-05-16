package openai

import (
	"net/http"

	upstreamopenai "github.com/phamtanminhtien/goroute/internal/adapter/upstream/openai"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
)

type Client struct {
	*upstreamopenai.Client
}

func NewClient(httpClient *http.Client, connection connection.Record) *Client {
	return &Client{Client: upstreamopenai.NewClient(httpClient, connection)}
}
