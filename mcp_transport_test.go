// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
	_ = NewMcpTransport("test-key", testMCPURL)

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

	transport := NewMcpTransport("test-key", server.URL)
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

	transport := NewMcpTransport("test-key", server.URL)
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

	transport := NewMcpTransport("invalid-key", server.URL)

	_, err := transport.Initialize(context.Background())
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrUnauthorized))
}

// Test 1.7: HTTP 404 Not Found
func TestMCPTransport_HTTPError_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	transport := NewMcpTransport("test-key", server.URL)
	transport.Initialize(context.Background())

	_, err := transport.CallTool(context.Background(), "nonexistent", nil)
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

	transport := NewMcpTransport("test-key", server.URL, WithMCPMaxRetries(2))

	_, err := transport.CallTool(context.Background(), "get_status_summary", nil)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrRateLimited))
}

// Test 1.9: HTTP 422 Validation Error
func TestMCPTransport_HTTPError_422(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer server.Close()

	transport := NewMcpTransport("test-key", server.URL)

	_, err := transport.CallTool(context.Background(), "get_status_summary", nil)
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

	transport := NewMcpTransport("test-key", server.URL, WithMCPMaxRetries(2))

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

		transport := NewMcpTransport("test-key", server.URL, WithMCPMaxRetries(1))

		result, err := transport.CallTool(context.Background(), "test", nil)
		require.NoError(t, err, "Should retry on %d", status)
		require.NotNil(t, result)
	}
}

// Test 1.12: Thread-safe request ID
func TestMCPTransport_RequestID_Concurrent(t *testing.T) {
	transport := NewMcpTransport("test-key", testMCPURL)

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

	transport := NewMcpTransport("test-key", server.URL)

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

	transport := NewMcpTransport("test-key", server.URL, WithMCPMaxRetries(2))

	_, err := transport.CallTool(context.Background(), "test", nil)
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


