# hyperping-go

Go client library for the [Hyperping](https://hyperping.io) uptime monitoring API.

Used as the shared HTTP client by:

- [terraform-provider-hyperping](https://github.com/develeap/terraform-provider-hyperping)
- [hyperping-exporter](https://github.com/develeap/hyperping-exporter)

## Installation

```bash
go get github.com/develeap/hyperping-go@latest
```

## CLI

The `hyp` binary is a standalone CLI for Hyperping. No Python runtime required.

**Homebrew (macOS / Linux):**

```bash
brew install develeap/tap/hyp
```

**Go install:**

```bash
go install github.com/develeap/hyperping-go/cmd/hyp@latest
```

**Direct download:** grab the pre-built binary for your platform from the [GitHub releases page](https://github.com/develeap/hyperping-go/releases).

### CLI Usage

```bash
# Authentication: set once, use everywhere
export HYPERPING_API_KEY=sk_your_api_key

# Monitors
hyp monitor list
hyp monitor pause  <uuid>
hyp monitor resume <uuid>

# Incidents
hyp incident create --title "Elevated error rate" --text "p95 latency spiked" \
  --type monitoring --statuspage <uuid>
hyp incident resolve <uuid> --message "All clear"

# Status pages
hyp statuspage list
hyp statuspage show <uuid>

# Version info
hyp version

# Global flags (available on every subcommand)
hyp --api-key sk_...  monitor list   # override env var
hyp --output json     monitor list   # machine-readable JSON
hyp --base-url http://localhost:8080 monitor list
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

// Query response time over the last 24h for two monitors. Pass zero-value
// from/to to let the server use its default window (currently 30 days).
// Omit uuids entirely to query every monitor in the project.
now := time.Now().UTC()
resp, err := mcpClient.GetMonitorResponseTime(ctx, now.Add(-24*time.Hour), now,
    "mon_skuPqyDxN9MScu", "mon_IwBwebrfw2S2Q9")
if err != nil {
    log.Fatal(err)
}
if resp != nil {
    fmt.Printf("avg: %.0fms, p95: %.0fms (across %d monitors)\n",
        resp.AvgResponseTime, resp.P95ResponseTime, len(resp.Monitors))
}
```

Pass an empty string for the URL to use the official endpoint (`https://api.hyperping.io/v1/mcp`). Pass `http://localhost:PORT` for local development.

> **v0.7.0 breaking change:** `GetMonitorMtta`, `GetMonitorMttr`, `GetMonitorResponseTime`, and `GetMonitorUptime` migrated to a canonical windowed signature `(ctx, from, to time.Time, uuids ...string)` and new typed response models. The pre-v0.7.0 methods sent the wrong arg name to the server (`uuid` / `monitor_uuid` string instead of `monitor_uuids` array) and decoded into structs whose fields the server never returned — every call returned all-zero values silently. See `CHANGELOG.md` for the full migration table.
>
> **v0.4.0 breaking change:** `NewMcpTransport` now returns `(*McpTransport, error)`. Callers must handle the returned error.

### Integration tests

The SDK ships a separate integration test suite that hits the live Hyperping MCP server, gated behind a `integration` build tag:

```bash
HYPERPING_TEST_API_KEY=sk_... go test -tags=integration -race -count=1 ./...
```

CI runs this job automatically when the `HYPERPING_TEST_API_KEY` repository secret is configured; PRs from forks skip cleanly. The schema-snapshot unit test (`testdata/mcp_tools_list.json` + `schema_contract_test.go`) runs on every build and provides drift detection without requiring network access.

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
