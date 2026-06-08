// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

//go:build integration

// Integration tests for the MCP client. Run with:
//
//	HYPERPING_TEST_API_KEY=sk_... go test -tags=integration -race -count=1 ./...
//
// These tests hit the live Hyperping MCP server at https://api.hyperping.io/v1/mcp.
// They are gated behind the `integration` build tag so the normal `go test`
// invocation (and the unit-test CI matrix) never touches the network.
//
// Why integration tests exist alongside the schema snapshot:
//
//   - The snapshot test catches "we send a key the server does not declare"
//     by reading testdata/mcp_tools_list.json. It does NOT catch silent
//     decode-into-empty-struct regressions, because the snapshot is the
//     declared schema, not a sample response.
//
//   - These tests issue real CallTool requests against the live server,
//     decode the response into the v0.7.0 typed models, and assert the
//     decode succeeded. If the server changes a field name or type, this
//     test fails on the next CI run that has the secret set.
//
// If HYPERPING_TEST_API_KEY is not set, every test in this file t.Skips
// with a clear message. CI guards the integration job behind a repository
// variable so PRs from forks (where secrets are unavailable) do not appear
// red.

package hyperping

import (
	"context"
	"os"
	"testing"
	"time"
)

const integrationEnvVar = "HYPERPING_TEST_API_KEY"

func liveMCPClient(t *testing.T) *MCPClient {
	t.Helper()
	key := os.Getenv(integrationEnvVar)
	if key == "" {
		t.Skipf("integration test skipped: %s not set", integrationEnvVar)
	}
	tr, err := NewMcpTransport(key, "")
	if err != nil {
		t.Fatalf("NewMcpTransport: %v", err)
	}
	return NewMCPClient(tr)
}

// withTimeout returns a child context with a 30s ceiling so a stuck server
// cannot wedge the whole CI job.
func withTimeout(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// integrationWindow returns a small known-good window (last 24h). The four
// windowed tools accept this and either return populated data (response
// time, often uptime) or empty arrays + zero scalars (mtta/mttr if no
// outages in the window). Decode-success is the load-bearing assertion;
// the actual numerical values are not pinned because they change over
// time.
func integrationWindow() (time.Time, time.Time) {
	now := time.Now().UTC()
	return now.Add(-24 * time.Hour), now
}

func TestIntegration_GetMonitorMtta_DecodesLive(t *testing.T) {
	c := liveMCPClient(t)
	ctx, cancel := withTimeout(t)
	defer cancel()
	from, to := integrationWindow()
	resp, err := c.GetMonitorMtta(ctx, from, to)
	if err != nil {
		t.Fatalf("GetMonitorMtta failed: %v", err)
	}
	if resp == nil {
		t.Fatal("GetMonitorMtta returned nil response")
	}
	// resp.Monitors may legitimately be empty when there are no alerts in
	// the window; we only assert the decode itself succeeded.
}

func TestIntegration_GetMonitorMttr_DecodesLive(t *testing.T) {
	c := liveMCPClient(t)
	ctx, cancel := withTimeout(t)
	defer cancel()
	from, to := integrationWindow()
	resp, err := c.GetMonitorMttr(ctx, from, to)
	if err != nil {
		t.Fatalf("GetMonitorMttr failed: %v", err)
	}
	if resp == nil {
		t.Fatal("GetMonitorMttr returned nil response")
	}
}

func TestIntegration_GetMonitorResponseTime_DecodesLive(t *testing.T) {
	c := liveMCPClient(t)
	ctx, cancel := withTimeout(t)
	defer cancel()
	from, to := integrationWindow()
	resp, err := c.GetMonitorResponseTime(ctx, from, to)
	if err != nil {
		t.Fatalf("GetMonitorResponseTime failed: %v", err)
	}
	if resp == nil {
		t.Fatal("GetMonitorResponseTime returned nil response")
	}
}

func TestIntegration_GetMonitorUptime_DecodesLive(t *testing.T) {
	c := liveMCPClient(t)
	ctx, cancel := withTimeout(t)
	defer cancel()
	from, to := integrationWindow()
	resp, err := c.GetMonitorUptime(ctx, from, to)
	if err != nil {
		t.Fatalf("GetMonitorUptime failed: %v", err)
	}
	if resp == nil {
		t.Fatal("GetMonitorUptime returned nil response")
	}
}
