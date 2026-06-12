// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clientDebugVars mirrors the shape of a hyperping.client.N expvar entry.
type clientDebugVars struct {
	InFlight            int64            `json:"in_flight"`
	TotalRequests       int64            `json:"total_requests"`
	Errors              map[string]int64 `json:"errors"`
	CircuitBreakerState string           `json:"circuit_breaker_state"`
	Retries             map[string]int64 `json:"retries"`
	MCPSessionRefreshes int64            `json:"mcp_session_refreshes"`
}

// fetchDebugStats calls /debug/vars and extracts the entry identified by mapName.
func fetchDebugStats(t *testing.T, addr, mapName string) clientDebugVars {
	t.Helper()
	resp, err := http.Get("http://" + addr + "/debug/vars")
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &raw), "parse /debug/vars JSON")

	v, ok := raw[mapName]
	require.True(t, ok, "key %q not found in /debug/vars output", mapName)

	var cv clientDebugVars
	require.NoError(t, json.Unmarshal(v, &cv))
	return cv
}

func TestWithDebugStats_ServeEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewClient("test_key",
		WithBaseURL(server.URL),
		WithMaxRetries(0),
		WithNoCircuitBreaker(),
		WithDebugStats("127.0.0.1:0"),
	)
	require.NotNil(t, client.debugStats)
	addr := client.debugStats.Addr()
	mapName := client.debugStats.MapName()
	require.NotEmpty(t, addr)
	require.NotEmpty(t, mapName)

	require.Eventually(t, func() bool {
		resp, err := http.Get("http://" + addr + "/debug/vars")
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 2*time.Second, 20*time.Millisecond)

	cv := fetchDebugStats(t, addr, mapName)
	assert.Equal(t, int64(0), cv.InFlight)
	assert.Equal(t, int64(0), cv.TotalRequests)
	assert.Equal(t, "closed", cv.CircuitBreakerState)
}

func TestWithDebugStats_TotalRequestsAndRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewClient("test_key",
		WithBaseURL(server.URL),
		WithMaxRetries(3),
		WithRetryWait(0, 0),
		WithNoCircuitBreaker(),
		WithDebugStats("127.0.0.1:0"),
	)
	addr := client.debugStats.Addr()
	mapName := client.debugStats.MapName()

	err := client.doRequest(context.Background(), http.MethodGet, "/v1/monitors", nil, nil)
	require.NoError(t, err)

	cv := fetchDebugStats(t, addr, mapName)
	assert.Equal(t, int64(1), cv.TotalRequests, "one top-level call")
	assert.Equal(t, int64(0), cv.InFlight, "in-flight decremented after completion")
	assert.Equal(t, int64(1), cv.Retries["attempt_1"], "first retry")
	assert.Equal(t, int64(1), cv.Retries["attempt_2"], "second retry")
	assert.Empty(t, cv.Errors, "no terminal error")
}

func TestWithDebugStats_ErrorClassification(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		errKey  string
	}{
		{"rate_limit", http.StatusTooManyRequests, "rate_limit"},
		{"server_error", http.StatusInternalServerError, "server_error"},
		{"client_error", http.StatusBadRequest, "client_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			}))
			defer server.Close()

			client := NewClient("test_key",
				WithBaseURL(server.URL),
				WithMaxRetries(0),
				WithNoCircuitBreaker(),
				WithDebugStats("127.0.0.1:0"),
			)
			addr := client.debugStats.Addr()
			mapName := client.debugStats.MapName()

			_ = client.doRequest(context.Background(), http.MethodGet, "/v1/monitors", nil, nil)

			cv := fetchDebugStats(t, addr, mapName)
			assert.Equal(t, int64(1), cv.TotalRequests)
			assert.Equal(t, int64(1), cv.Errors[tc.errKey], "expected %s error", tc.errKey)
		})
	}
}

func TestWithDebugStats_InFlightDecrement(t *testing.T) {
	unblock := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-unblock
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewClient("test_key",
		WithBaseURL(server.URL),
		WithMaxRetries(0),
		WithNoCircuitBreaker(),
		WithDebugStats("127.0.0.1:0"),
	)
	addr := client.debugStats.Addr()
	mapName := client.debugStats.MapName()

	done := make(chan error, 1)
	go func() {
		done <- client.doRequest(context.Background(), http.MethodGet, "/v1/monitors", nil, nil)
	}()

	require.Eventually(t, func() bool {
		return client.debugStats.inFlight.Value() == 1
	}, 2*time.Second, 5*time.Millisecond)

	cv := fetchDebugStats(t, addr, mapName)
	assert.Equal(t, int64(1), cv.InFlight)

	close(unblock)
	require.NoError(t, <-done)

	cv = fetchDebugStats(t, addr, mapName)
	assert.Equal(t, int64(0), cv.InFlight)
}

func TestWithDebugStats_CircuitBreakerState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient("test_key",
		WithBaseURL(server.URL),
		WithMaxRetries(0),
		WithDebugStats("127.0.0.1:0"),
		WithCircuitBreakerSettings(gobreaker.Settings{
			Name:        "test-cb",
			MaxRequests: 1,
			Interval:    0,
			Timeout:     30 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 3
			},
		}),
	)
	addr := client.debugStats.Addr()
	mapName := client.debugStats.MapName()

	for i := 0; i < 3; i++ {
		_ = client.doRequest(context.Background(), http.MethodGet, "/v1/monitors", nil, nil)
	}

	cv := fetchDebugStats(t, addr, mapName)
	assert.Equal(t, "open", cv.CircuitBreakerState)
}

func TestWithTransportStats_MCPSessionRefreshes(t *testing.T) {
	sessionDropped := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Mcp-Session-Id") != "" && !sessionDropped {
			sessionDropped = true
			// Drain body so server doesn't log errors.
			_, _ = io.Copy(io.Discard, r.Body) //nolint:errcheck
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if isMCPInitialize(r) {
			w.Header().Set("Mcp-Session-Id", "sess-abc")
			_, _ = fmt.Fprintln(w, `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26","capabilities":{},"serverInfo":{"name":"test","version":"1.0"}}}`)
			return
		}
		_, _ = fmt.Fprintln(w, `{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"{}"}],"isError":false}}`)
	}))
	defer server.Close()

	debugClient := NewClient("unused_key", WithNoCircuitBreaker(), WithDebugStats("127.0.0.1:0"))
	stats := debugClient.Stats()
	require.NotNil(t, stats)

	transport, err := NewMcpTransport("test_key", server.URL, WithTransportStats(stats))
	require.NoError(t, err)

	_, _ = transport.CallTool(context.Background(), "get_status_summary", nil)

	cv := fetchDebugStats(t, stats.Addr(), stats.MapName())
	assert.Equal(t, int64(1), cv.MCPSessionRefreshes)
}

// isMCPInitialize peeks at the JSON-RPC body to detect an initialize call.
func isMCPInitialize(r *http.Request) bool {
	if r.Body == nil {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
	if err != nil {
		return false
	}
	var req struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return false
	}
	return req.Method == "initialize"
}

func TestWithDebugStats_EmptyAddr(t *testing.T) {
	client := NewClient("test_key",
		WithNoCircuitBreaker(),
		WithDebugStats(""),
	)
	assert.Nil(t, client.debugStats, "empty addr disables debug stats")
}

func TestWithDebugStats_InvalidAddr(t *testing.T) {
	client := NewClient("test_key",
		WithNoCircuitBreaker(),
		WithDebugStats("invalid-addr-no-port-%%"),
	)
	assert.Nil(t, client.debugStats)
	assert.Error(t, client.setupErr)
}
