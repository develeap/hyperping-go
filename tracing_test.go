//go:build ignore

// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func newTestTP() (*sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	return tp, exp
}

func findAttr(attrs []attribute.KeyValue, key string) (attribute.Value, bool) {
	for _, a := range attrs {
		if string(a.Key) == key {
			return a.Value, true
		}
	}
	return attribute.Value{}, false
}

func newMCPTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	initialized := false
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		method, _ := req["method"].(string)
		idRaw := req["id"]
		switch method {
		case "initialize":
			initialized = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
				"jsonrpc": "2.0",
				"id":      idRaw,
				"result": map[string]any{
					"protocolVersion": "2025-03-26",
					"serverInfo":      map[string]any{"name": "test", "version": "0"},
					"capabilities":    map[string]any{},
				},
			})
		case "tools/call":
			if !initialized {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
				"jsonrpc": "2.0",
				"id":      idRaw,
				"result":  map[string]any{"content": []any{}},
			})
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
}

// ==== REST tracing tests ====

func TestWithTracerProvider_DefaultIsNoOp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}")) //nolint:errcheck
	}))
	defer srv.Close()

	c := NewClient("sk_test123", WithBaseURL(srv.URL), WithNoCircuitBreaker())
	_ = c.doRequest(context.Background(), http.MethodGet, "/v1/monitors", nil, nil)
	// no panic, no span — test passes if we get here
}

func TestWithTracerProvider_RESTSpanCreated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}")) //nolint:errcheck
	}))
	defer srv.Close()

	tp, exp := newTestTP()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	c := NewClient("sk_test123", WithBaseURL(srv.URL), WithTracerProvider(tp), WithNoCircuitBreaker())
	err := c.doRequest(context.Background(), http.MethodGet, "/v1/monitors", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span, got none")
	}
}

func TestWithTracerProvider_RESTSpanAttributes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}")) //nolint:errcheck
	}))
	defer srv.Close()

	tp, exp := newTestTP()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	c := NewClient("sk_test123", WithBaseURL(srv.URL), WithTracerProvider(tp), WithNoCircuitBreaker())
	_ = c.doRequest(context.Background(), http.MethodGet, "/v1/monitors", nil, nil)

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("no spans recorded")
	}
	s := spans[0]
	attrs := s.Attributes

	if v, ok := findAttr(attrs, "hyperping.method"); !ok || v.AsString() != http.MethodGet {
		t.Errorf("hyperping.method: want %q, got %q", http.MethodGet, v.AsString())
	}
	if v, ok := findAttr(attrs, "hyperping.endpoint"); !ok || v.AsString() != "/v1/monitors" {
		t.Errorf("hyperping.endpoint: want %q, got %q", "/v1/monitors", v.AsString())
	}
	if v, ok := findAttr(attrs, "http.status_code"); !ok || v.AsInt64() != 200 {
		t.Errorf("http.status_code: want 200, got %v", v)
	}
}

func TestWithTracerProvider_RESTSpanOnError_404(t *testing.T) {
	reqID := "req-test-404"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", reqID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`)) //nolint:errcheck
	}))
	defer srv.Close()

	tp, exp := newTestTP()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	c := NewClient("sk_test123", WithBaseURL(srv.URL), WithTracerProvider(tp), WithNoCircuitBreaker())
	_ = c.doRequest(context.Background(), http.MethodGet, "/v1/monitors/mon_notexist", nil, nil)

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("no spans recorded")
	}
	s := spans[0]

	if v, ok := findAttr(s.Attributes, "http.status_code"); !ok || v.AsInt64() != 404 {
		t.Errorf("http.status_code: want 404, got %v", v)
	}
	if v, ok := findAttr(s.Attributes, "hyperping.request_id"); !ok || v.AsString() != reqID {
		t.Errorf("hyperping.request_id: want %q, got %q", reqID, v.AsString())
	}
}

func TestWithTracerProvider_RESTSpanOnError_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server error"}`)) //nolint:errcheck
	}))
	defer srv.Close()

	tp, exp := newTestTP()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	// Set maxRetries=0 to avoid multiple spans from retry attempts
	c := NewClient("sk_test123", WithBaseURL(srv.URL), WithTracerProvider(tp), WithNoCircuitBreaker(), WithMaxRetries(0))
	_ = c.doRequest(context.Background(), http.MethodGet, "/v1/monitors", nil, nil)

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("no spans recorded")
	}
	s := spans[0]

	if v, ok := findAttr(s.Attributes, "http.status_code"); !ok || v.AsInt64() != 500 {
		t.Errorf("http.status_code: want 500, got %v", v)
	}
}

func TestWithTracerProvider_RESTSpanName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}")) //nolint:errcheck
	}))
	defer srv.Close()

	tp, exp := newTestTP()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	c := NewClient("sk_test123", WithBaseURL(srv.URL), WithTracerProvider(tp), WithNoCircuitBreaker())
	_ = c.doRequest(context.Background(), http.MethodGet, "/v1/monitors", nil, nil)

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("no spans recorded")
	}
	want := "hyperping.GET /v1/monitors"
	if got := spans[0].Name; got != want {
		t.Errorf("span name: want %q, got %q", want, got)
	}
}

// ==== MCP tracing tests ====

func TestWithMCPTracerProvider_DefaultIsNoOp(t *testing.T) {
	srv := newMCPTestServer(t)
	defer srv.Close()

	tr, err := NewMcpTransport("sk_test123", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	// no tracer configured — should not panic
	_, _ = tr.CallTool(context.Background(), "test_tool", nil)
}

func TestWithMCPTracerProvider_CallToolCreatesSpan(t *testing.T) {
	srv := newMCPTestServer(t)
	defer srv.Close()

	tp, exp := newTestTP()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	tr, err := NewMcpTransport("sk_test123", srv.URL, WithMCPTracerProvider(tp))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = tr.CallTool(context.Background(), "get_monitors", nil)

	spans := exp.GetSpans()
	// Filter to only tools/call spans (exclude any initialize spans if those were traced)
	var toolSpans []tracetest.SpanStub
	for _, s := range spans {
		if s.SpanKind == trace.SpanKindClient {
			toolSpans = append(toolSpans, s)
		}
	}
	if len(toolSpans) == 0 {
		t.Fatalf("expected at least one client span, got spans: %v", spans)
	}
}

func TestWithMCPTracerProvider_CallToolAttributes(t *testing.T) {
	srv := newMCPTestServer(t)
	defer srv.Close()

	tp, exp := newTestTP()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	tr, err := NewMcpTransport("sk_test123", srv.URL, WithMCPTracerProvider(tp))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = tr.CallTool(context.Background(), "get_monitors", nil)

	var toolSpan *tracetest.SpanStub
	for i, s := range exp.GetSpans() {
		if s.SpanKind == trace.SpanKindClient {
			toolSpan = &exp.GetSpans()[i]
			break
		}
	}
	if toolSpan == nil {
		t.Fatal("no client span found")
	}

	if v, ok := findAttr(toolSpan.Attributes, "hyperping.method"); !ok || v.AsString() != "tools/call" {
		t.Errorf("hyperping.method: want %q, got %q", "tools/call", v.AsString())
	}
	if v, ok := findAttr(toolSpan.Attributes, "hyperping.endpoint"); !ok || v.AsString() != "get_monitors" {
		t.Errorf("hyperping.endpoint: want %q, got %q", "get_monitors", v.AsString())
	}
}

func TestWithMCPTracerProvider_CallToolSpanName(t *testing.T) {
	srv := newMCPTestServer(t)
	defer srv.Close()

	tp, exp := newTestTP()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	tr, err := NewMcpTransport("sk_test123", srv.URL, WithMCPTracerProvider(tp))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = tr.CallTool(context.Background(), "list_monitors", nil)

	var toolSpan *tracetest.SpanStub
	for i, s := range exp.GetSpans() {
		if s.SpanKind == trace.SpanKindClient {
			toolSpan = &exp.GetSpans()[i]
			break
		}
	}
	if toolSpan == nil {
		t.Fatal("no client span found")
	}

	want := "hyperping.tools/call list_monitors"
	if got := toolSpan.Name; got != want {
		t.Errorf("span name: want %q, got %q", want, got)
	}
}
