package httpclient

import (
	"net"
	"net/http"
	"time"

	"github.com/phamtanminhtien/goroute/internal/config"
)

const (
	DefaultDialTimeout           = time.Duration(config.DefaultDialTimeoutMs) * time.Millisecond
	DefaultTLSHandshakeTimeout   = time.Duration(config.DefaultTLSHandshakeTimeoutMs) * time.Millisecond
	DefaultResponseHeaderTimeout = time.Duration(config.DefaultResponseHeaderTimeoutMs) * time.Millisecond
	DefaultKeepAlive             = 30 * time.Second
)

func NewStreamingClient() *http.Client {
	return NewStreamingClientWithSettings(config.DefaultProviderRuntimeSettings())
}

func NewStreamingClientWithSettings(settings config.ProviderRuntimeSettings) *http.Client {
	return &http.Client{Transport: NewStreamingTransportWithSettings(settings)}
}

func NewStreamingTransport() *http.Transport {
	return NewStreamingTransportWithSettings(config.DefaultProviderRuntimeSettings())
}

func NewStreamingTransportWithSettings(settings config.ProviderRuntimeSettings) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{
		Timeout:   time.Duration(settings.DialTimeoutMs) * time.Millisecond,
		KeepAlive: DefaultKeepAlive,
	}).DialContext
	transport.TLSHandshakeTimeout = time.Duration(settings.TLSHandshakeTimeoutMs) * time.Millisecond
	transport.ResponseHeaderTimeout = time.Duration(settings.ResponseHeaderTimeoutMs) * time.Millisecond
	return transport
}
