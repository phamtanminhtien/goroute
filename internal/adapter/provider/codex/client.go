package codex

import (
	"net/http"
	"strings"

	adapterhttpclient "github.com/phamtanminhtien/goroute/internal/adapter/httpclient"
	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
)

const (
	defaultBaseURL   = "https://chatgpt.com/backend-api/codex"
	defaultUserAgent = "codex-cli/1.0.18 (macOS; arm64)"
)

type Client struct {
	httpClient *http.Client
	connection connection.Record
	baseURL    string
}

func NewClient(connection connection.Record) *Client {
	return NewClientWithHTTPClient(nil, connection)
}

func NewClientWithHTTPClient(httpClient *http.Client, connection connection.Record) *Client {
	if httpClient == nil {
		httpClient = adapterhttpclient.NewStreamingClient()
	}

	return &Client{
		httpClient: httpClient,
		connection: connection,
		baseURL:    defaultBaseURL,
	}
}

func NewClientWithRuntimeSettings(connection connection.Record, settings config.ProviderRuntimeSettings) *Client {
	return NewClientWithHTTPClient(adapterhttpclient.NewStreamingClientWithSettings(settings), connection)
}

func (c *Client) resolveAccessToken(forceRefresh bool) (string, error) {
	connection := c.connection
	if forceRefresh {
		var err error
		connection, err = refreshConnectionToken(connection, true)
		if err != nil {
			return "", err
		}
		c.connection = connection
		return GetAccessToken(connection)
	}

	if strings.TrimSpace(connection.APIKey) != "" && strings.TrimSpace(connection.AccessToken) == "" {
		return strings.TrimSpace(connection.APIKey), nil
	}

	var err error
	connection, err = refreshConnectionToken(connection, false)
	if err != nil {
		return "", err
	}
	c.connection = connection

	token, err := GetAccessToken(connection)
	if err == nil {
		return token, nil
	}
	if strings.TrimSpace(connection.APIKey) != "" {
		return strings.TrimSpace(connection.APIKey), nil
	}

	return "", err
}

func shouldRetryWithTokenRefresh(statusCode int, connection connection.Record) bool {
	if statusCode != http.StatusUnauthorized && statusCode != http.StatusForbidden {
		return false
	}

	return strings.TrimSpace(connection.RefreshToken) != ""
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
