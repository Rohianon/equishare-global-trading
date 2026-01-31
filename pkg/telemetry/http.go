package telemetry

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// WrapHTTPClient wraps an HTTP client with OpenTelemetry tracing
func WrapHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}

	// Wrap the transport with otelhttp
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	client.Transport = otelhttp.NewTransport(transport)
	return client
}

// NewTracedHTTPClient creates a new HTTP client with tracing enabled
func NewTracedHTTPClient() *http.Client {
	return &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
}
