# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.0] - 2026-05-21

### Fixed

- MCP transport now captures `Mcp-Session-Id` from the `initialize` response and echoes it on every subsequent JSON-RPC request, per MCP 2025-03-26 Streamable HTTP spec. Servers that enforce session ownership (Hyperping among them) were previously treating every `tools/call` as a sessionless fresh attempt and rate-limiting them as `initialize`-bucket violations. Fixes hyperping-exporter#60.

### Added

- One-shot session-loss recovery: when the server returns HTTP 404 to a request that carried a session id, the transport clears its local session state, re-initializes once, and retries the failing call. The existing init mutex serializes concurrent recovery so a worker pool does not stampede the server with parallel initialize attempts.
- `ErrSessionLost` sentinel error so callers can match the recovered-too-late case via `errors.Is`.

### Notes

- Backward compatible: servers that do not issue `Mcp-Session-Id` continue to work without sending the header.

## [0.4.0] - 2026-04-25

### Breaking Changes

- `NewMcpTransport` now returns `(*McpTransport, error)` instead of `*McpTransport`. Callers must handle the returned error. The constructor validates the base URL (must be HTTPS or `http://localhost`) before returning.

### Changed

- MCP transport now uses `buildTransportChain` to enforce TLS 1.2+, AEAD cipher suites, and connection pooling (matches the REST client).
- `Initialize` uses mutex-based double-checked locking to prevent concurrent goroutines from racing into the handshake.
- All JSON unmarshal errors in `mcp_client.go` are now propagated to callers instead of being silently discarded.
- Unchecked type assertions in MCP response parsing replaced with comma-ok checks that return descriptive errors.

### Fixed

- HTTP response bodies are now drained before close on error paths, preventing connection pool exhaustion.
- `context.Done()` is respected in retry sleep loops, allowing in-flight MCP calls to cancel promptly.

### Dependency

- Bumped `github.com/stretchr/testify` from 1.3.0 to 1.11.1.

## [0.3.0] - 2026-04-22

### Added

- MCP transport (`McpTransport`): JSON-RPC 2.0 client for the Hyperping MCP server with retry logic on 5xx errors and error mapping to typed error types.
- MCP client (`MCPClient`): high-level typed wrapper exposing all 30 MCP tools (25 read, 5 write).
- `mcp_models.go`: typed structs for all MCP responses (`StatusSummary`, `ResponseTimeReport`, `MttaReport`, `MttrReport`, `MonitorAnomaly`, `ProbeLogResponse`, `AlertHistory`, `OnCallSchedule`, `EscalationPolicy`, `Integration`, and more).

## [0.2.1] - 2026-04-14

### Fixed

- `parseOutageListResponse` and `parseMaintenanceListResponse` now preserve `HasNextPage` from the API response instead of always returning `false` on empty pages.
- Circuit breaker timeout increased from 5s to 30s to match realistic API response times.

## [0.2.0] - 2026-04-10

### Added

- Circuit breaker via `gobreaker` to protect against cascading failures on repeated API errors.
- Retry with exponential backoff on 5xx responses across all REST client methods.
- Pagination helpers: `HasNextPage` field propagated correctly through list responses.

## [0.1.0] - 2026-04-01

### Added

- Initial release with full REST API client for Hyperping.
- Resources: monitors, incidents, outages, maintenance windows, escalation policies, on-call schedules, integrations, status pages.
- Error types: `HyperpingError`, `NotFoundError`, `AuthError`, `RateLimitError`, `ValidationError`.
- `validateBaseURL`, `IsNotFound`, `IsRateLimit` helpers.
