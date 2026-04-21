# GO-01: MCP Transport Implementation Plan (TDD) - REVISED

## Overview
Implement an MCP (Model Context Protocol) client in Go for hyperping-go that exposes 16 tools not available via REST API.

## Architecture

### Package Structure
```
hyperping-go/
├── mcp_transport.go       # Low-level JSON-RPC 2.0 transport
├── mcp_client.go         # High-level typed client
├── mcp_models.go         # Response models
├── mcp_transport_test.go # Transport layer tests
├── mcp_client_test.go    # Client layer tests
└── mcp_models_test.go    # Model tests
```

## Constants
```go
const (
    DefaultMCPURL         = "https://api.hyperping.io/v1/mcp"
    MCPProtocolVersion    = "2025-03-26"
    DefaultMCPTimeout     = 30 * time.Second
    DefaultMaxRetries     = 2
)
```

## JSON-RPC Types
```go
// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      any             `json:"id"`
    Method  string          `json:"method"`
    Params  map[string]any  `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      any             `json:"id"`
    Result  any             `json:"result,omitempty"`
    Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
    Code    int             `json:"code"`
    Message string          `json:"message"`
    Data    any             `json:"data,omitempty"`
}
```

## MCP Tools (16 total)

| Tool | Category | Response Type |
|------|----------|---------------|
| get_status_summary | Status | StatusSummary |
| get_monitor_response_time | Reporting | ResponseTimeReport |
| get_monitor_mtta | Reporting | MttaReport |
| get_monitor_mttr | Reporting | MttrReport |
| get_monitor_anomalies | Observability | []MonitorAnomaly |
| get_monitor_http_logs | Observability | ProbeLogResponse |
| list_recent_alerts | Alerts | AlertHistory |
| list_on_call_schedules | On-Call | []OnCallSchedule |
| get_on_call_schedule | On-Call | OnCallSchedule |
| list_escalation_policies | Escalation | []EscalationPolicy |
| get_escalation_policy | Escalation | EscalationPolicy |
| list_team_members | Team | []TeamMember |
| list_integrations | Integrations | []Integration |
| get_integration | Integrations | Integration |
| get_outage_timeline | Outages | OutageTimeline |
| search_monitors_by_name | Monitors | []Monitor |

## Transport Layer Design

### MCPTransport Struct
```go
type MCPTransport struct {
    client       *http.Client
    url          string
    token        []byte
    maxRetries   int
    initialized  atomic.Bool  // thread-safe flag
    reqID        atomic.Int64  // thread-safe counter
}
```

### Option Pattern (matching hyperping-go)
```go
type TransportOption func(*MCPTransport)

// Options:
// - WithTimeout(d time.Duration)
// - WithMaxRetries(n int)
// - WithHTTPClient(h *http.Client)
```

### Error Handling
Map HTTP status codes to existing hyperping errors:
- 401, 403 → ErrUnauthorized
- 404 → ErrNotFound
- 429 → ErrRateLimited
- 400, 422 → ErrValidation
- 500, 502, 503, 504 → retry with backoff

MCP JSON-RPC errors:
- -32700 → ErrValidation (parse error)
- -32600 → ErrValidation (invalid request)
- -32601 → ErrNotFound (method not found)
- -32602 → ErrValidation (invalid params)
- -32603 → ErrServerError (internal error)

### Retry Logic
- Max retries: 2 (configurable)
- Backoff: min(2^attempt, 10) seconds
- Jitter: +/- 1 second
- Only retry on 500, 502, 503, 504

## TDD Implementation Order

### Phase 1: Transport Layer Tests (Write First)
1. Test JSON-RPC request encoding with valid request
2. Test JSON-RPC response decoding with success
3. Test JSON-RPC error response decoding
4. Test initialize handshake
5. Test call_tool method
6. Test HTTP error handling (401, 403, 404, 429, 400, 422, 500, 502, 503, 504)
7. Test retry with backoff
8. Test thread-safe request ID counter (concurrent)
9. Test initialized flag thread safety
10. Test close during in-flight requests

### Phase 2: Client Layer Tests
1. Test each MCP tool method exists
2. Test response parsing into models
3. Test error propagation
4. Test with various argument combinations

### Phase 3: Model Tests
1. Test each model JSON unmarshaling
2. Test null/undefined field handling
3. Test unknown field handling (forward compat)
4. Test empty arrays

## Test Infrastructure

### Test Data Constants
```go
const (
    validRequestJSON = `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_status_summary","arguments":{}}}`
    validResponseJSON = `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"total\":10,\"up\":9,\"down\":1}"}]}}`
    errorResponseJSON = `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`
    initializeResponseJSON = `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26","capabilities":{}}}`
)
```

### Mock Server Patterns
- Use httptest.Server for all tests
- Return appropriate JSON-RPC responses
- Test all error scenarios with proper HTTP status codes

## Model Definitions

### StatusSummary
```go
type StatusSummary struct {
    Total int `json:"total"`
    Up    int `json:"up"`
    Down  int `json:"down"`
}
```

### ResponseTimeReport
```go
type ResponseTimeReport struct {
    UUID       string  `json:"uuid"`
    Avg        float64 `json:"avg"`
    Min        float64 `json:"min"`
    Max        float64 `json:"max"`
    // ... other fields
}
```

### MttaReport, MttrReport, MonitorAnomaly, ProbeLogResponse, AlertHistory, OnCallSchedule, EscalationPolicy, TeamMember, Integration, OutageTimeline, Monitor
// Define all with proper JSON tags matching Python implementation

## Acceptance Criteria
1. All 16 MCP tools callable via typed client
2. Error handling matches existing hyperping-go patterns
3. Thread-safe request ID generation (atomic)
4. Thread-safe initialized flag (atomic)
5. Retry logic with exponential backoff + jitter
6. Context support for cancellation/timeouts
7. Tests cover all error scenarios
8. 85%+ test coverage on new code
9. Uses option pattern for configuration
10. Interface defined for mocking

## Naming Conventions
- Structs: `MCPTransport`, `MCPClient` (MCP as one word)
- Methods: `GetStatusSummary`, `ListOnCallSchedules`
- Files: `mcp_transport.go`, `mcp_client.go`, `mcp_models.go`
- Tests: `mcp_transport_test.go`, etc.
