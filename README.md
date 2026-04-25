# hyperping-go

Go client library for the [Hyperping](https://hyperping.io) uptime monitoring API.

Used as the shared HTTP client by:

- [terraform-provider-hyperping](https://github.com/develeap/terraform-provider-hyperping)
- [hyperping-exporter](https://github.com/develeap/hyperping-exporter)

## Installation

```bash
go get github.com/develeap/hyperping-go@latest
```

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    hyperping "github.com/develeap/hyperping-go"
)

func main() {
    client := hyperping.NewClient("sk_your_api_key")

    monitors, err := client.ListMonitors(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    for _, m := range monitors {
        fmt.Printf("%s: down=%v\n", m.Name, m.Down)
    }
}
```

## API Coverage

| Resource | Operations |
|----------|-----------|
| Monitors | List, Get, Create, Update, Delete, Pause, Resume |
| Incidents | List, Get, Create, Update, Resolve, AddUpdate, Delete |
| Maintenance | List, Get, Create, Update, Delete |
| Status Pages | List, Get, Create, Update, Delete |
| Subscribers | List, Add, Delete |
| Healthchecks | List, Get, Create, Update, Delete, Pause, Resume |
| Outages | List, Get, Create, Acknowledge, Resolve, Escalate, Unacknowledge, Delete |
| Reports | ListMonitorReports, GetMonitorReport |

## Configuration

```go
client := hyperping.NewClient(
    "sk_your_api_key",
    hyperping.WithBaseURL("https://api.hyperping.io"),
    hyperping.WithMaxRetries(3),
)
```

## MCP Client

The library also exposes a JSON-RPC 2.0 MCP client for the Hyperping MCP server (25 read tools, 5 write tools).

```go
// Initialize transport (validates URL, enforces TLS 1.2+)
transport, err := hyperping.NewMcpTransport("sk_your_api_key", "")
if err != nil {
    log.Fatal(err)
}
mcpClient := hyperping.NewMCPClient(transport)

// Query monitor response time
report, err := mcpClient.GetMonitorResponseTime(ctx, "monitor-uuid")
if err != nil {
    log.Fatal(err)
}
if report != nil {
    fmt.Printf("avg: %.3fs\n", report.Avg)
}
```

Pass an empty string for the URL to use the official endpoint (`https://api.hyperping.io/v1/mcp`). Pass `http://localhost:PORT` for local development.

> **v0.4.0 breaking change:** `NewMcpTransport` now returns `(*McpTransport, error)`. Callers must handle the returned error.

## Features

- Automatic retry with exponential backoff and jitter on 5xx and 429 responses
- Retry-After header respected on rate limit responses
- Circuit breaker (via [gobreaker](https://github.com/sony/gobreaker)) to prevent cascading failures
- Context propagation on all API calls; in-flight MCP calls respect context cancellation
- Structured error types: `*APIError` with status code and message
- MCP JSON-RPC 2.0 client with 30 typed tools (25 read, 5 write)
- TLS 1.2+ enforced on all transport connections with AEAD cipher suites
- Mutex double-checked locking on MCP handshake prevents concurrent initialization races

## Testing

Interfaces for all resource types are defined in `interface.go`, enabling straightforward mock injection in tests:

```go
type MockMonitorAPI struct{}

func (m *MockMonitorAPI) ListMonitors(ctx context.Context) ([]hyperping.Monitor, error) {
    return []hyperping.Monitor{{Name: "test"}}, nil
}
```

## License

MIT. Maintained by [Develeap](https://develeap.com).
