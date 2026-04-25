# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
