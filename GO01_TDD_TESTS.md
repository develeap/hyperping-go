# GO-01 TDD Test Specification

## Test-Driven Development Plan

This document specifies exact test cases to write FIRST, then implement to make pass.

---

## Phase 1: Transport Layer Tests

### Test 1.1: JSON-RPC Request Encoding
```go
func TestMCPTransport_EncodeRequest(t *testing.T) {
    // Given: A transport with test API key
    transport := NewMCPTransport("test-key", withTestURL(t))

    // When: Encoding a request for "get_status_summary"
    req, err := transport.encodeRequest("tools/call", map[string]any{"name": "get_status_summary"})

    // Then:
    require.NoError(t, err)
    require.JSONEq(t, `{
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/call",
        "params": {"name": "get_status_summary"}
    }`, string(req))
}
```

### Test 1.2: JSON-RPC Response Decoding - Success
```go
func TestMCPTransport_DecodeResponse_Success(t *testing.T) {
    // Given: Response JSON from MCP server
    respJSON := `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"total\":10}"}]}}`

    // When: Decoding
    result, err := decodeResponse([]byte(respJSON))

    // Then:
    require.NoError(t, err)
    require.Equal(t, "ok", result.Status)
}
```

### Test 1.3: JSON-RPC Response Decoding - Error
```go
func TestMCPTransport_DecodeResponse_Error(t *testing.T) {
    // Given: Error response
    respJSON := `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`

    // When: Decoding
    result, err := decodeResponse([]byte(respJSON))

    // Then:
    require.Error(t, err)
    require.Contains(t, err.Error(), "Method not found")
}
```

### Test 1.4: Initialize Handshake
```go
func TestMCPTransport_Initialize(t *testing.T) {
    // Given: Mock MCP server returning initialize response
    server := newMCPMockServer(t, mockInitializeResponse)
    transport := NewMCPTransport("test-key", server.URL)

    // When: Initializing
    result, err := transport.Initialize(context.Background())

    // Then:
    require.NoError(t, err)
    require.Equal(t, "2025-03-26", result.ProtocolVersion)
}
```

### Test 1.5: CallTool - Success
```go
func TestMCPTransport_CallTool_Success(t *testing.T) {
    // Given: Initialized transport, mock server
    server := newMCPMockServer(t, mockCallToolResponse)
    transport := NewMCPTransport("test-key", server.URL)
    transport.Initialize(context.Background())

    // When: Calling get_status_summary tool
    result, err := transport.CallTool(context.Background(), "get_status_summary", nil)

    // Then:
    require.NoError(t, err)
    require.Equal(t, 10, result.Total)
}
```

### Test 1.6: HTTP 401 Unauthorized
```go
func TestMCPTransport_HTTPError_401(t *testing.T) {
    // Given: Mock server returning 401
    server := newMCPMockServer(t, "", 401)
    transport := NewMCPTransport("invalid-key", server.URL)

    // When: Initializing
    _, err := transport.Initialize(context.Background())

    // Then: Should return ErrUnauthorized
    require.Error(t, err)
    require.True(t, errors.Is(err, ErrUnauthorized))
}
```

### Test 1.7: HTTP 429 Rate Limited
```go
func TestMCPTransport_HTTPError_429(t *testing.T) {
    // Given: Mock server returning 429 with Retry-After
    server := newMCPMockServer(t, "", 429, "Retry-After: 60")
    transport := NewMCPTransport("test-key", server.URL, WithMaxRetries(2))

    // When: Calling tool with retry
    _, err := transport.CallTool(context.Background(), "get_status_summary", nil)

    // Then: Should return ErrRateLimited with retry info
    require.Error(t, err)
    require.True(t, errors.Is(err, ErrRateLimited))
    var rateLimitErr *APIError
    if errors.As(err, &rateLimitErr) {
        require.Equal(t, 60, rateLimitErr.RetryAfter)
    }
}
```

### Test 1.8: Retry on 500
```go
func TestMCPTransport_RetryOn500(t *testing.T) {
    // Given: Mock server that fails twice then succeeds
    callCount := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        callCount++
        if callCount < 3 {
            w.WriteHeader(500)
        } else {
            json.NewEncoder(w).Encode(map[string]any{
                "jsonrpc": "2.0", "id": 1,
                "result": map[string]any{"content": []any{}},
            })
        }
    }))
    transport := NewMCPTransport("test-key", server.URL, WithMaxRetries(2))

    // When: Calling tool
    _, err := transport.CallTool(context.Background(), "test", nil)

    // Then: Should succeed after retries
    require.NoError(t, err)
    require.Equal(t, 3, callCount)
}
```

### Test 1.9: Thread-Safe Request ID
```go
func TestMCPTransport_RequestID_Concurrent(t *testing.T) {
    // Given: Transport
    transport := NewMCPTransport("test-key", "http://unused")

    // When: Many goroutines request IDs concurrently
    var wg sync.WaitGroup
    ids := make(map[int64]bool)
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

    // Then: All IDs should be unique
    require.Equal(t, 100, len(ids))
}
```

### Test 1.10: Thread-Safe Initialize
```go
func TestMCPTransport_Initialize_Concurrent(t *testing.T) {
    // Given: Mock server
    server := newMCPMockServer(t, mockInitializeResponse)
    transport := NewMCPTransport("test-key", server.URL)

    // When: Multiple goroutines call Initialize concurrently
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            transport.Initialize(context.Background())
            wg.Done()
        }()
    }
    wg.Wait()

    // Then: Should not panic, initialized should be true
    require.True(t, transport.initialized.Load())
}
```

---

## Phase 2: Client Layer Tests

### Test 2.1: GetStatusSummary
```go
func TestMCPClient_GetStatusSummary(t *testing.T) {
    // Given: Mock transport returning StatusSummary data
    transport := &mockTransport{
        result: map[string]any{
            "total": 10, "up": 9, "down": 1,
        },
    }
    client := NewMCPClient(transport)

    // When: Calling GetStatusSummary
    result, err := client.GetStatusSummary(context.Background())

    // Then:
    require.NoError(t, err)
    require.Equal(t, 10, result.Total)
    require.Equal(t, 9, result.Up)
    require.Equal(t, 1, result.Down)
}
```

### Test 2.2: All 16 Tools Available
```go
func TestMCPClient_AllTools(t *testing.T) {
    // Verify all 16 tools are accessible
    client := NewMCPClient(&mockTransport{})
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
```

---

## Phase 3: Model Tests

### Test 3.1: StatusSummary Unmarshal
```go
func TestStatusSummary_Unmarshal(t *testing.T) {
    // Given: JSON response
    data := []byte(`{"total":10,"up":9,"down":1}`)

    // When: Unmarshaling into StatusSummary
    var result StatusSummary
    err := json.Unmarshal(data, &result)

    // Then:
    require.NoError(t, err)
    require.Equal(t, 10, result.Total)
}
```

### Test 3.2: Null Field Handling
```go
func TestStatusSummary_NullFields(t *testing.T) {
    // Given: JSON with null fields
    data := []byte(`{"total":null,"up":null,"down":null}`)

    // When: Unmarshaling
    var result StatusSummary
    err := json.Unmarshal(data, &result)

    // Then: Should not error, fields should be zero
    require.NoError(t, err)
    require.Equal(t, 0, result.Total)
}
```

---

## Implementation Order (TDD)

1. Write Test 1.1 → Implement encodeRequest
2. Write Test 1.2 → Implement decodeResponse  
3. Write Test 1.3 → Handle decode error
4. Write Test 1.4 → Implement Initialize
5. Write Test 1.5 → Implement CallTool
6. Write Tests 1.6-1.8 → Implement error handling + retry
7. Write Tests 1.9-1.10 → Implement thread safety
8. Write Phase 2 tests → Implement client methods
9. Write Phase 3 tests → Implement models

## Running Tests
```bash
go test -v -race -coverprofile=coverage.out ./...
```