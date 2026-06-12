// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// WithTracerProvider sets an OpenTelemetry TracerProvider on the Client.
// When set, doRequest creates a client span for each API call.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return func(c *Client) {
		c.tracerProvider = tp
	}
}

// WithMCPTracerProvider sets an OpenTelemetry TracerProvider on the McpTransport.
// When set, CallTool creates a client span for each tool call.
func WithMCPTracerProvider(tp trace.TracerProvider) TransportOption {
	return func(t *McpTransport) {
		t.tracerProvider = tp
	}
}

// recordResponseOnSpan sets http.status_code and hyperping.request_id on the
// span carried by ctx. When no recording span is present the calls are no-ops
// because otel returns a noop span from SpanFromContext in that case.
func recordResponseOnSpan(ctx context.Context, resp *http.Response) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Int64("http.status_code", int64(resp.StatusCode)))
	if reqID := resp.Header.Get("X-Request-Id"); reqID != "" {
		span.SetAttributes(attribute.String("hyperping.request_id", reqID))
	}
}
