// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test constants
const (
	testMCPURL         = "https://api.hyperping.io/v1/mcp"
	testMCPProtocolVer = "2025-03-26"
)

// Mock responses
var (
	mockInitializeResponse = JSONRPCResponse{
		JSONRPC: "2.0",
		ID:     1,
		Result: map[string]any{
			"protocolVersion": testMCPProtocolVer,
			"capabilities":   map[string]any{},
		},
	}

	mockStatusSummaryResponse = JSONRPCResponse{
		JSONRPC: "2.0",
		ID:     1,
		Result: map[string]any{
			"content": []any{
				map[string]any{
					"type": "text",
					"text": `{"total":10,"up":9,"down":1}`,
				},
			},
		},
	}
)

// ==================== Phase 1: Transport Layer Tests ====================

// Test 1.1: JSON-RPC request encoding
func TestMCPTransport_EncodeRequest(t *testing.T) {
	_, err := NewMcpTransport("test-key", testMCPURL)
	require.NoError(t, err)

	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":     1,
		"method": "tools/call",
		"params": map[string]any{
			"name":      "get_status_summary",
			"arguments": map[string]any{},
		},
	})
	require.NoError(t, err)

	var req JSONRPCRequest
	err = json.Unmarshal(body, &req)
	require.NoError(t, err)
	require.Equal(t, "2.0", req.JSONRPC)
	require.Equal(t, "tools/call", req.Method)
	require.Equal(t, "get_status_summary", req.Params["name"])
}

// Test 1.2: JSON-RPC response decoding - success
func TestMCPTransport_DecodeResponse_Success(t *testing.T) {
	respJSON := `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"total\":10}"}]}}`

	var resp JSONRPCResponse
	err := json.Unmarshal([]byte(respJSON), &resp)
	require.NoError(t, err)
	require.Equal(t, "2.0", resp.JSONRPC)
	require.NotNil(t, resp.Result)
}

// Test 1.3: JSON-RPC response decoding - error
func TestMCPTransport_DecodeResponse_Error(t *testing.T) {
	respJSON := `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`

	var resp JSONRPCResponse
	err := json.Unmarshal([]byte(respJSON), &resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Error)
	require.Equal(t, -32601, resp.Error.Code)
	require.Equal(t, "Method not found", resp.Error.Message)
}

// Test 1.4: Initialize handshake
func TestMCPTransport_Initialize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.Contains(t, r.Header.Get("Authorization"), "Bearer")

		resp := mockInitializeResponse
		resp.ID = nil
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL)
	require.NoError(t, err)
	result, err := transport.Initialize(context.Background())

	require.NoError(t, err)
	require.NotNil(t, result)
}

// Test 1.5: CallTool - success
func TestMCPTransport_CallTool_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mockStatusSummaryResponse)
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL)
	require.NoError(t, err)
	transport.Initialize(context.Background())

	result, err := transport.CallTool(context.Background(), "get_status_summary", nil)
	require.NoError(t, err)
	require.NotNil(t, result)
}

// Test 1.6: HTTP 401 Unauthorized
func TestMCPTransport_HTTPError_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	transport, err := NewMcpTransport("invalid-key", server.URL)
	require.NoError(t, err)

	_, err = transport.Initialize(context.Background())
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrUnauthorized))
}

// Test 1.7: HTTP 404 Not Found
func TestMCPTransport_HTTPError_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL)
	require.NoError(t, err)
	transport.Initialize(context.Background())

	_, err = transport.CallTool(context.Background(), "nonexistent", nil)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotFound))
}

// Test 1.8: HTTP 429 Rate Limited
func TestMCPTransport_HTTPError_429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL, WithMCPMaxRetries(2))
	require.NoError(t, err)

	_, err = transport.CallTool(context.Background(), "get_status_summary", nil)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrRateLimited))
}

// Test 1.9: HTTP 422 Validation Error
func TestMCPTransport_HTTPError_422(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL)
	require.NoError(t, err)

	_, err = transport.CallTool(context.Background(), "get_status_summary", nil)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrValidation))
}

// Test 1.10: Retry on 500
func TestMCPTransport_RetryOn500(t *testing.T) {
	callCount := atomic.Int64{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := callCount.Add(1)
		if count == 1 {
			json.NewEncoder(w).Encode(mockInitializeResponse)
			return
		}
		if count < 4 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(mockStatusSummaryResponse)
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL, WithMCPMaxRetries(2))
	require.NoError(t, err)

	result, err := transport.CallTool(context.Background(), "test", nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(4), callCount.Load())
}

// Test 1.11: Retry on 502, 503, 504
func TestMCPTransport_RetryOnServerErrors(t *testing.T) {
	for _, status := range []int{502, 503, 504} {
		callCount := atomic.Int64{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := callCount.Add(1)
			if count == 1 {
				json.NewEncoder(w).Encode(mockInitializeResponse)
				return
			}
			if count < 3 {
				w.WriteHeader(status)
				return
			}
			json.NewEncoder(w).Encode(mockStatusSummaryResponse)
		}))
		defer server.Close()

		transport, err := NewMcpTransport("test-key", server.URL, WithMCPMaxRetries(1))
		require.NoError(t, err)

		result, err := transport.CallTool(context.Background(), "test", nil)
		require.NoError(t, err, "Should retry on %d", status)
		require.NotNil(t, result)
	}
}

// Test 1.12: Thread-safe request ID
func TestMCPTransport_RequestID_Concurrent(t *testing.T) {
	transport, err := NewMcpTransport("test-key", testMCPURL)
	require.NoError(t, err)

	var wg sync.WaitGroup
	ids := make(map[int64]bool)
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			id := transport.nextID()
			mu.Lock()
			ids[id] = true
			mu.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()

	require.Equal(t, 100, len(ids), "All IDs should be unique")
}

// Test 1.13: Thread-safe initialized flag
func TestMCPTransport_Initialize_Concurrent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		json.NewEncoder(w).Encode(mockInitializeResponse)
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			transport.Initialize(context.Background())
			wg.Done()
		}()
	}
	wg.Wait()

	require.True(t, transport.initialized.Load(), "Should be initialized")
}

// Test 1.14: No retry on 400
func TestMCPTransport_NoRetryOn400(t *testing.T) {
	callCount := atomic.Int64{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL, WithMCPMaxRetries(2))
	require.NoError(t, err)

	_, err = transport.CallTool(context.Background(), "test", nil)
	require.Error(t, err)
	require.Equal(t, int64(1), callCount.Load(), "Should not retry on 400")
}

// ==================== Phase 2: Client Layer Tests ====================

// mockMCPTransport implements transport interface for testing
type mockMCPTransport struct {
	result    any
	callCount int
}

func (m *mockMCPTransport) Initialize(ctx context.Context) (map[string]any, error) {
	m.callCount++
	return map[string]any{"protocolVersion": "2025-03-26"}, nil
}

func (m *mockMCPTransport) CallTool(ctx context.Context, toolName string, args map[string]any) (any, error) {
	m.callCount++
	return m.result, nil
}

// Test 2.1: MCPClient GetStatusSummary
func TestMCPClient_GetStatusSummary(t *testing.T) {
	transport := &mockMCPTransport{
		result: map[string]any{
			"total": 10,
			"up":    9,
			"down":  1,
		},
	}
	client := NewMCPClient(transport)

	result, err := client.GetStatusSummary(context.Background())
	require.NoError(t, err)
	require.Equal(t, 10, result.Total)
}

// Test 2.2: All 16 tools available
func TestMCPClient_AllTools(t *testing.T) {
	transport := &mockMCPTransport{result: map[string]any{}}
	client := NewMCPClient(transport)

	require.NotNil(t, client.GetStatusSummary)
	require.NotNil(t, client.GetMonitorResponseTime)
	require.NotNil(t, client.GetMonitorMtta)
	require.NotNil(t, client.GetMonitorMttr)
	require.NotNil(t, client.GetMonitorAnomalies)
	require.NotNil(t, client.GetMonitorHttpLogs)
	require.NotNil(t, client.ListRecentAlerts)
	require.NotNil(t, client.ListOnCallSchedules)
	require.NotNil(t, client.GetOnCallSchedule)
	require.NotNil(t, client.ListEscalationPolicies)
	require.NotNil(t, client.GetEscalationPolicy)
	require.NotNil(t, client.ListTeamMembers)
	require.NotNil(t, client.ListIntegrations)
	require.NotNil(t, client.GetIntegration)
	require.NotNil(t, client.GetOutageTimeline)
	require.NotNil(t, client.SearchMonitorsByName)
}

// ==================== Phase 3: Model Tests ====================

// Test 3.1: StatusSummary unmarshal
func TestStatusSummary_Unmarshal(t *testing.T) {
	data := []byte(`{"total":10,"up":9,"down":1}`)

	var result StatusSummary
	err := json.Unmarshal(data, &result)

	require.NoError(t, err)
	require.Equal(t, 10, result.Total)
	require.Equal(t, 9, result.Up)
	require.Equal(t, 1, result.Down)
}

// Test 3.2: StatusSummary null fields
func TestStatusSummary_NullFields(t *testing.T) {
	data := []byte(`{"total":null,"up":null,"down":null}`)

	var result StatusSummary
	err := json.Unmarshal(data, &result)

	require.NoError(t, err)
	require.Equal(t, 0, result.Total)
}

// Test 3.3: ResponseTimeReport unmarshal
func TestResponseTimeReport_Unmarshal(t *testing.T) {
	data := []byte(`{"uuid":"abc","avg":100.5,"min":50,"max":200}`)

	var result ResponseTimeReport
	err := json.Unmarshal(data, &result)

	require.NoError(t, err)
	require.Equal(t, "abc", result.UUID)
	require.Equal(t, 100.5, result.Avg)
}

// Test 1.15: NewMcpTransport rejects invalid URL scheme
func TestNewMcpTransport_InvalidURL(t *testing.T) {
	_, err := NewMcpTransport("key", "ftp://example.com")
	require.Error(t, err, "non-HTTPS non-localhost URL should be rejected")
}

// ==================== Phase 4: Session-ID Tests ====================

// classifyRequest reads the body and returns the JSON-RPC method ("initialize"
// or "tools/call" or other). Restores the body via io.NopCloser so handlers
// remain free to inspect or echo it if needed.
func classifyRequest(t *testing.T, r *http.Request) string {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	var req JSONRPCRequest
	require.NoError(t, json.Unmarshal(body, &req))
	return req.Method
}

// Test 1.16: Mcp-Session-Id captured on initialize.
func TestMCPTransport_SessionID_CapturedOnInitialize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Mcp-Session-Id", "test-sess-abc")
		require.NoError(t, json.NewEncoder(w).Encode(mockInitializeResponse))
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL)
	require.NoError(t, err)
	_, err = transport.Initialize(context.Background())
	require.NoError(t, err)

	require.Equal(t, "test-sess-abc", transport.loadSessionID())
}

// Test 1.17: Mcp-Session-Id propagated on every subsequent tools/call.
func TestMCPTransport_SessionID_PropagatedOnCallTool(t *testing.T) {
	const expectedSID = "test-sess-abc"
	var (
		mu              sync.Mutex
		toolCallHeaders []string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := classifyRequest(t, r)
		if method == "initialize" {
			w.Header().Set("Mcp-Session-Id", expectedSID)
			require.NoError(t, json.NewEncoder(w).Encode(mockInitializeResponse))
			return
		}
		mu.Lock()
		toolCallHeaders = append(toolCallHeaders, r.Header.Get("Mcp-Session-Id"))
		mu.Unlock()
		require.NoError(t, json.NewEncoder(w).Encode(mockStatusSummaryResponse))
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		_, err := transport.CallTool(context.Background(), "get_status_summary", nil)
		require.NoError(t, err)
	}

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, toolCallHeaders, 5)
	for i, got := range toolCallHeaders {
		require.Equal(t, expectedSID, got, "call %d carried wrong session id", i+1)
	}
}

// Test 1.18: backward compatible when server issues no Mcp-Session-Id.
func TestMCPTransport_NoSessionID_BackwardCompat(t *testing.T) {
	var (
		mu              sync.Mutex
		toolCallHeaders []string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := classifyRequest(t, r)
		if method == "initialize" {
			// Deliberately no Mcp-Session-Id header.
			require.NoError(t, json.NewEncoder(w).Encode(mockInitializeResponse))
			return
		}
		mu.Lock()
		toolCallHeaders = append(toolCallHeaders, r.Header.Get("Mcp-Session-Id"))
		mu.Unlock()
		require.NoError(t, json.NewEncoder(w).Encode(mockStatusSummaryResponse))
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		_, err := transport.CallTool(context.Background(), "get_status_summary", nil)
		require.NoError(t, err)
	}

	require.Equal(t, "", transport.loadSessionID())
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, toolCallHeaders, 3)
	for i, got := range toolCallHeaders {
		require.Equal(t, "", got, "call %d should not have set Mcp-Session-Id", i+1)
	}
}

// Test 1.19: HTTP 404 on a session-bearing request triggers one re-initialize
// and a retry, both observable in the server's request sequence.
func TestMCPTransport_SessionLossRecovery(t *testing.T) {
	var (
		initCount atomic.Int64
		callCount atomic.Int64
		seqMu     sync.Mutex
		sequence  []string
	)
	record := func(method string) {
		seqMu.Lock()
		sequence = append(sequence, method)
		seqMu.Unlock()
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := classifyRequest(t, r)
		record(method)
		if method == "initialize" {
			n := initCount.Add(1)
			w.Header().Set("Mcp-Session-Id", fmt.Sprintf("sess-%d", n))
			require.NoError(t, json.NewEncoder(w).Encode(mockInitializeResponse))
			return
		}
		// tools/call
		n := callCount.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		require.NoError(t, json.NewEncoder(w).Encode(mockStatusSummaryResponse))
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL)
	require.NoError(t, err)

	result, err := transport.CallTool(context.Background(), "get_status_summary", nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "sess-2", transport.loadSessionID())

	seqMu.Lock()
	defer seqMu.Unlock()
	require.Equal(t, []string{"initialize", "tools/call", "initialize", "tools/call"}, sequence)
}

// Test 1.20: 20 concurrent callers see at most ONE re-initialize after a
// session loss. The init mutex serializes recovery so the server is not
// stampeded with parallel initialize attempts.
//
// "No stampede" is verified two ways:
//   - server total initialize count is exactly 2 (initial + one recovery);
//   - server never observes two initialize requests concurrently in-flight
//     (an inFlight gauge incremented at handler entry, decremented at exit).
//     The total-count assertion alone could pass even if 19 goroutines all
//     entered the recovery init concurrently and the server happened to
//     serialize them; the inFlight gauge catches that subtler pattern.
func TestMCPTransport_SessionLoss_NoStampede(t *testing.T) {
	var (
		initCount    atomic.Int64
		callCount    atomic.Int64
		inFlightInit atomic.Int64
		stampede     atomic.Bool
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := classifyRequest(t, r)
		if method == "initialize" {
			if inFlightInit.Add(1) > 1 {
				stampede.Store(true)
			}
			defer inFlightInit.Add(-1)
			n := initCount.Add(1)
			w.Header().Set("Mcp-Session-Id", fmt.Sprintf("sess-%d", n))
			// Hold the handler briefly so concurrent inits would actually
			// overlap if the init mutex wasn't serializing them.
			time.Sleep(5 * time.Millisecond)
			require.NoError(t, json.NewEncoder(w).Encode(mockInitializeResponse))
			return
		}
		// tools/call: 404 until the second initialize has landed; then accept
		// only the latest session id (sess-2) and respond 200.
		callCount.Add(1)
		if initCount.Load() < 2 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		want := fmt.Sprintf("sess-%d", initCount.Load())
		if r.Header.Get("Mcp-Session-Id") != want {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		require.NoError(t, json.NewEncoder(w).Encode(mockStatusSummaryResponse))
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL)
	require.NoError(t, err)

	const N = 20
	var wg sync.WaitGroup
	errs := make([]error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, e := transport.CallTool(context.Background(), "get_status_summary", nil)
			errs[idx] = e
		}(i)
	}
	wg.Wait()

	require.Equal(t, int64(2), initCount.Load(),
		"server saw %d initialize requests; want exactly 2", initCount.Load())
	require.False(t, stampede.Load(),
		"two initialize requests overlapped on the server; init mutex did not serialize recovery")
	require.Equal(t, "sess-2", transport.loadSessionID())
	for i, e := range errs {
		require.NoError(t, e, "goroutine %d failed", i)
	}
}

// Test 1.21a: backward-compat server returns HTTP 404 for an unknown tool
// (the request carried no session id because the server never issued one).
// The transport must surface this as ErrNotFound, NOT ErrSessionLost. This
// codifies Case C in callToolOnce's 404 handler.
func TestMCPTransport_BackwardCompat_NotFound_StaysNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := classifyRequest(t, r)
		if method == "initialize" {
			// No Mcp-Session-Id header on response — backward-compat server.
			require.NoError(t, json.NewEncoder(w).Encode(mockInitializeResponse))
			return
		}
		// tools/call → unknown tool, server returns 404.
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL)
	require.NoError(t, err)

	_, err = transport.CallTool(context.Background(), "missing_tool", nil)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotFound),
		"backward-compat 404 must surface as ErrNotFound, not %T: %v", err, err)
	require.False(t, errors.Is(err, ErrSessionLost),
		"backward-compat 404 must NOT surface as ErrSessionLost")
}

// Test 1.21b: each McpTransport instance owns its own session id. A session
// captured on one transport must not leak to another instance.
func TestMCPTransport_SessionID_DoesNotLeakAcrossInstances(t *testing.T) {
	makeServer := func(sid string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Mcp-Session-Id", sid)
			require.NoError(t, json.NewEncoder(w).Encode(mockInitializeResponse))
		}))
	}
	srvA := makeServer("sess-A")
	defer srvA.Close()
	srvB := makeServer("sess-B")
	defer srvB.Close()

	tA, err := NewMcpTransport("key-A", srvA.URL)
	require.NoError(t, err)
	tB, err := NewMcpTransport("key-B", srvB.URL)
	require.NoError(t, err)

	_, err = tA.Initialize(context.Background())
	require.NoError(t, err)
	_, err = tB.Initialize(context.Background())
	require.NoError(t, err)

	require.Equal(t, "sess-A", tA.loadSessionID())
	require.Equal(t, "sess-B", tB.loadSessionID())
}

// Test 1.21c: ErrSessionLost remains matchable via errors.Is when callers
// wrap it for additional context. Locks in the sentinel-error contract so
// a future refactor that switches to a typed error or fmt.Errorf wrapping
// does not silently break consumer code.
func TestMCPTransport_SessionLost_IsMatchable_WhenWrapped(t *testing.T) {
	require.True(t, errors.Is(ErrSessionLost, ErrSessionLost), "sanity")

	wrapped := fmt.Errorf("CallTool retry exhausted: %w", ErrSessionLost)
	require.True(t, errors.Is(wrapped, ErrSessionLost),
		"errors.Is must traverse wrapping; got %v", wrapped)

	doubleWrapped := fmt.Errorf("monitor poll failed: %w", wrapped)
	require.True(t, errors.Is(doubleWrapped, ErrSessionLost),
		"errors.Is must traverse double-wrapping; got %v", doubleWrapped)
}

// Test 1.21d: concurrent direct calls to Initialize (rare but supported) do
// not deadlock or corrupt state, even with simultaneous CallTool traffic.
// Documents that the transport tolerates the M1 race window described in
// Initialize's doc; this is a smoke test, not a correctness assertion about
// individual error returns.
func TestMCPTransport_Initialize_RacingWithCallTool(t *testing.T) {
	var initCount atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := classifyRequest(t, r)
		if method == "initialize" {
			n := initCount.Add(1)
			w.Header().Set("Mcp-Session-Id", fmt.Sprintf("sess-%d", n))
			require.NoError(t, json.NewEncoder(w).Encode(mockInitializeResponse))
			return
		}
		require.NoError(t, json.NewEncoder(w).Encode(mockStatusSummaryResponse))
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	// Initialize-spammer goroutine.
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			if _, e := transport.Initialize(ctx); e != nil {
				return
			}
		}
	}()
	// CallTool-spammer goroutine.
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			// CallTool may surface ErrSessionLost spuriously per the M1 doc;
			// we tolerate but do not assert on it. The point is no deadlock,
			// no panic, no goroutine leak.
			_, _ = transport.CallTool(ctx, "get_status_summary", nil)
		}
	}()

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("concurrent Initialize + CallTool did not complete within timeout")
	}

	require.True(t, transport.initialized.Load(),
		"transport should be initialized after the spam settles")
}

// Test 1.21: when the retried call also returns session-loss, CallTool stops
// after exactly one recovery attempt and surfaces ErrSessionLost.
func TestMCPTransport_SessionLossRecoveryFailsTwice(t *testing.T) {
	var (
		initCount atomic.Int64
		callCount atomic.Int64
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := classifyRequest(t, r)
		if method == "initialize" {
			n := initCount.Add(1)
			w.Header().Set("Mcp-Session-Id", fmt.Sprintf("sess-%d", n))
			require.NoError(t, json.NewEncoder(w).Encode(mockInitializeResponse))
			return
		}
		callCount.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL)
	require.NoError(t, err)

	_, err = transport.CallTool(context.Background(), "get_status_summary", nil)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrSessionLost),
		"expected ErrSessionLost, got %T: %v", err, err)
	require.Equal(t, int64(2), initCount.Load(),
		"server saw %d initialize requests; want exactly 2 (one initial + one recovery)", initCount.Load())
	require.Equal(t, int64(2), callCount.Load(),
		"server saw %d tools/call requests; want exactly 2", callCount.Load())
}

// ==================== Phase 5: Audit Fixes ====================

// Test (audit CRITICAL-1): MCP CallTool must cap response body to prevent OOM.
//
// A malicious or compromised MCP server (or a MITM at a proxy) could stream
// arbitrarily many bytes of JSON to exhaust client memory. The REST path
// already caps at maxResponseBodyBytes (10 MB) via readResponseBody; the MCP
// path must reuse the same cap rather than handing resp.Body straight to
// json.NewDecoder().
//
// The handler below streams ~50 MB of payload as a single large JSON string
// inside a valid JSON-RPC envelope. CallTool must return an error within
// bounded memory, NOT decode the full payload.
func TestMCPTransport_CallTool_RejectsOversizedResponse(t *testing.T) {
	const oversizeBytes = 50 * 1024 * 1024
	// Server: respond to initialize normally; on tools/call, stream a huge JSON
	// payload that is bigger than the cap.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := classifyRequest(t, r)
		if method == "initialize" {
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(mockInitializeResponse))
			return
		}
		// tools/call: write a JSON-RPC envelope whose result.content[0].text
		// is a huge embedded JSON string. We do not need the inner text to be
		// valid JSON for the test: the outer envelope decoder must abort
		// before reaching that depth because the body exceeds the cap.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Write a valid JSON prefix then a long run of filler bytes inside a
		// JSON string. We deliberately never close the JSON string so a
		// no-cap decoder would keep reading until EOF / OOM.
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"`)
		buf := make([]byte, 64*1024)
		for i := range buf {
			buf[i] = 'A'
		}
		written := 0
		for written < oversizeBytes {
			n, err := w.Write(buf)
			if err != nil {
				return
			}
			written += n
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL, WithMCPTimeout(30*time.Second))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	_, err = transport.CallTool(ctx, "get_status_summary", nil)
	require.Error(t, err, "CallTool must reject an oversized response, not OOM")
	require.True(t,
		strings.Contains(err.Error(), "response") ||
			strings.Contains(err.Error(), "too large") ||
			strings.Contains(err.Error(), "exceeds"),
		"error should mention oversized response; got %v", err)
}

// Test (audit CRITICAL-2): WithMCPHTTPClient must not bypass the TLS
// enforcement + auth-transport chain.
//
// A user passing a custom *http.Client (e.g. for metrics or proxy support)
// would previously have its default transport substituted in wholesale,
// dropping defaultTLSConfig (TLS 1.2+ floor and AEAD cipher suites) AND
// dropping the authTransport that injects the Bearer header. The Bearer
// is still manually set on the request in initializeAttempt/callToolOnce,
// so the practical risk is that a downgrade-to-cleartext attack ships the
// token over plain HTTP.
//
// We assert one of two contracts:
//   - the constructor errors, OR
//   - after the option runs, the http.Client.Transport is wrapped through
//     buildTransportChain so the top-level RoundTripper is *authTransport
//     and the inner layer enforces TLS for HTTPS URLs.
func TestNewMcpTransport_CustomHTTPClient_PreservesTLSAndAuthChain(t *testing.T) {
	custom := &http.Client{Transport: &http.Transport{}}
	transport, err := NewMcpTransport(
		"test-key",
		"https://api.hyperping.io/v1/mcp",
		WithMCPHTTPClient(custom),
	)
	require.NoError(t, err)
	require.NotNil(t, transport)

	// The transport's http.Client must have its Transport re-wrapped so the
	// top layer is the auth/transport chain, not the bare *http.Transport
	// the caller supplied. Without the fix, transport.client.Transport ==
	// custom.Transport (a plain *http.Transport) and both checks fail.
	rt := transport.client.Transport
	require.NotNil(t, rt, "client transport must not be nil")
	authT, ok := rt.(*authTransport)
	require.True(t, ok,
		"expected top-level transport to be *authTransport after WithMCPHTTPClient, got %T", rt)
	require.Equal(t, []byte("test-key"), authT.token, "auth token must propagate")

	// For an HTTPS base URL on a non-localhost host, the inner transport
	// must be the standard *http.Transport with our defaultTLSConfig applied
	// (TLS 1.2 minimum, restricted cipher suites). For custom non-*http.Transport
	// inners, it would be wrapped in *tlsEnforcedTransport. Either path is
	// acceptable; both prove the chain was rebuilt.
	switch inner := authT.next.(type) {
	case *http.Transport:
		require.NotNil(t, inner.TLSClientConfig,
			"HTTPS transport must have TLSClientConfig set after option re-wrap")
		require.Equal(t, uint16(0x0303), inner.TLSClientConfig.MinVersion,
			"TLSClientConfig.MinVersion must be TLS 1.2")
	case *tlsEnforcedTransport:
		// acceptable
	default:
		t.Fatalf("expected inner transport to be *http.Transport with TLS config or *tlsEnforcedTransport, got %T", inner)
	}
}

// Test (audit CRITICAL-2, b): WithMCPHTTPClient with a non-*http.Transport
// custom RoundTripper must result in either an error from the constructor
// OR a chain that blocks cleartext HTTP for non-localhost URLs.
func TestNewMcpTransport_CustomHTTPClient_BlocksCleartextHTTP(t *testing.T) {
	// A mockRoundTripper records the request; we then attempt an HTTP probe.
	mockRT := &mockRoundTripper{
		response: &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))},
	}
	custom := &http.Client{Transport: mockRT}

	// HTTPS base URL so the constructor doesn't reject up front.
	transport, err := NewMcpTransport(
		"test-key",
		"https://api.hyperping.io/v1/mcp",
		WithMCPHTTPClient(custom),
	)
	require.NoError(t, err)
	require.NotNil(t, transport)

	// Now issue a probe HTTP request through the resulting client. If the
	// transport chain was correctly re-wrapped, this should be blocked at
	// the tlsEnforcedTransport layer.
	req, err := http.NewRequest(http.MethodGet, "http://api.hyperping.io/probe", nil)
	require.NoError(t, err)
	_, err = transport.client.Do(req)
	require.Error(t, err,
		"cleartext HTTP probe should be blocked by tlsEnforcedTransport after re-wrap")
	require.True(t,
		strings.Contains(err.Error(), "HTTPS required") ||
			strings.Contains(err.Error(), "tlsEnforcedTransport"),
		"error should come from the TLS enforcement layer; got %v", err)
}

// Test (audit HIGH-3): initializeAttempt must hold initMu while writing
// the (sessionID, initialized) pair, so the 404-clear path cannot interleave
// between those two writes and produce a torn state.
//
// Direct structural assertion: hold initMu in the test goroutine, then
// kick off Initialize concurrently. With the fix, initializeAttempt blocks
// on initMu before publishing either write, so until the test releases the
// mutex, the transport state stays clean. Without the fix, initializeAttempt
// publishes both writes immediately and the test goroutine sees the published
// state while still holding the mutex (proving the writes happen outside the
// critical section).
func TestMCPTransport_InitializeAttempt_WritesUnderInitMu(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Mcp-Session-Id", "sess-test")
		require.NoError(t, json.NewEncoder(w).Encode(mockInitializeResponse))
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL)
	require.NoError(t, err)

	// Take initMu in the test goroutine. With the fix, any concurrent
	// Initialize must block on this mutex before publishing the sessionID
	// and initialized writes. Without the fix, Initialize completes both
	// writes regardless.
	transport.initMu.Lock()

	done := make(chan error, 1)
	go func() {
		_, e := transport.Initialize(context.Background())
		done <- e
	}()

	// Give the background goroutine time to run its HTTP request and reach
	// the write step. With the fix, it will block on initMu. Without the
	// fix, it will complete the writes.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if transport.initialized.Load() || transport.loadSessionID() != "" {
			// State was published while we still hold initMu.
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	gotInit := transport.initialized.Load()
	gotSID := transport.loadSessionID()

	// Release the mutex so the background Initialize can finish.
	transport.initMu.Unlock()
	require.NoError(t, <-done)

	require.Falsef(t, gotInit,
		"initialized was published while initMu was held by another goroutine; write happened outside the critical section")
	require.Emptyf(t, gotSID,
		"sessionID was published while initMu was held by another goroutine; got %q", gotSID)
}

// Test (audit HIGH-3, b): concurrent Initialize calls do not deadlock and
// always leave the transport in a consistent (initialized=true, sessionID!="")
// state. Runs with -race to also catch any data race in the publish step.
func TestMCPTransport_ConcurrentInitialize_ConsistentFinalState(t *testing.T) {
	var seq atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := seq.Add(1)
		time.Sleep(5 * time.Millisecond)
		w.Header().Set("Mcp-Session-Id", fmt.Sprintf("sess-%d", n))
		require.NoError(t, json.NewEncoder(w).Encode(mockInitializeResponse))
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL)
	require.NoError(t, err)

	const N = 16
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = transport.Initialize(context.Background())
		}()
	}
	wg.Wait()

	require.True(t, transport.initialized.Load(),
		"after concurrent Initialize, transport.initialized must be true")
	require.NotEmpty(t, transport.loadSessionID(),
		"after concurrent Initialize with initialized=true, sessionID must be non-empty")
}

// Test (audit CRITICAL-1, b): Initialize must also cap response body.
func TestMCPTransport_Initialize_RejectsOversizedResponse(t *testing.T) {
	const oversizeBytes = 50 * 1024 * 1024
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Open a JSON object and stream a long key string we never close.
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":{"k":"`)
		buf := make([]byte, 64*1024)
		for i := range buf {
			buf[i] = 'A'
		}
		written := 0
		for written < oversizeBytes {
			n, err := w.Write(buf)
			if err != nil {
				return
			}
			written += n
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	defer server.Close()

	transport, err := NewMcpTransport("test-key", server.URL, WithMCPTimeout(30*time.Second))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	_, err = transport.Initialize(ctx)
	require.Error(t, err, "Initialize must reject an oversized response, not OOM")
	require.True(t,
		strings.Contains(err.Error(), "response") ||
			strings.Contains(err.Error(), "too large") ||
			strings.Contains(err.Error(), "exceeds"),
		"error should mention oversized response; got %v", err)
}
