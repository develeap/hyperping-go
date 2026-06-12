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
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

const integrationEnvVar = "HYPERPING_TEST_API_KEY"

// sharedLiveMCPClient is a package-scoped singleton transport so the
// whole integration suite shares one MCP session. The Hyperping server
// rate-limits initialize at 5/min per project; constructing a fresh
// transport per test case blew that budget once the suite grew past
// four tests. The McpTransport itself is safe for concurrent use; the
// sync.Once guards construction without forcing tests to run serially.
var (
	sharedClientOnce sync.Once
	sharedClient     *MCPClient
	sharedClientErr  error
)

func liveMCPClient(t *testing.T) *MCPClient {
	t.Helper()
	key := os.Getenv(integrationEnvVar)
	if key == "" {
		t.Skipf("integration test skipped: %s not set", integrationEnvVar)
	}
	sharedClientOnce.Do(func() {
		tr, err := NewMcpTransport(key, "")
		if err != nil {
			sharedClientErr = err
			return
		}
		sharedClient = NewMCPClient(tr)
	})
	if sharedClientErr != nil {
		t.Fatalf("NewMcpTransport: %v", sharedClientErr)
	}
	return sharedClient
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

// The following tests cover the six MCPClient methods that pre-v0.7.1
// passed nil for args. They are the regression guard for the v0.7.1
// nil-args serialization fix (callToolOnce now sends arguments:{}
// instead of omitting the key) AND for the response-shape corrections
// to AlertHistory, TeamMember, and StatusSummary. A red CI here on the
// next run after a rollback would surface the regression immediately.

func TestIntegration_GetStatusSummary_DecodesLive(t *testing.T) {
	c := liveMCPClient(t)
	ctx, cancel := withTimeout(t)
	defer cancel()
	resp, err := c.GetStatusSummary(ctx)
	if err != nil {
		t.Fatalf("GetStatusSummary failed: %v", err)
	}
	if resp == nil {
		t.Fatal("GetStatusSummary returned nil response")
	}
}

func TestIntegration_ListRecentAlerts_DecodesLive(t *testing.T) {
	c := liveMCPClient(t)
	ctx, cancel := withTimeout(t)
	defer cancel()
	resp, err := c.ListRecentAlerts(ctx)
	if err != nil {
		t.Fatalf("ListRecentAlerts failed: %v", err)
	}
	if resp == nil {
		t.Fatal("ListRecentAlerts returned nil response")
	}
	// The Total() accessor must be callable without panic on whatever
	// the live server returned, even for an empty rawAlerts.
	_ = resp.Total()
}

func TestIntegration_ListOnCallSchedules_DecodesLive(t *testing.T) {
	c := liveMCPClient(t)
	ctx, cancel := withTimeout(t)
	defer cancel()
	_, err := c.ListOnCallSchedules(ctx)
	if err != nil {
		t.Fatalf("ListOnCallSchedules failed: %v", err)
	}
}

func TestIntegration_ListEscalationPolicies_DecodesLive(t *testing.T) {
	c := liveMCPClient(t)
	ctx, cancel := withTimeout(t)
	defer cancel()
	_, err := c.ListEscalationPolicies(ctx)
	if err != nil {
		t.Fatalf("ListEscalationPolicies failed: %v", err)
	}
}

func TestIntegration_ListTeamMembers_DecodesLive(t *testing.T) {
	c := liveMCPClient(t)
	ctx, cancel := withTimeout(t)
	defer cancel()
	members, err := c.ListTeamMembers(ctx)
	if err != nil {
		t.Fatalf("ListTeamMembers failed: %v", err)
	}
	// Every team has at least one member (the owner that issued the
	// API key); zero members would imply the server stopped enumerating
	// the project's roster.
	if len(members) == 0 {
		t.Fatal("ListTeamMembers returned empty slice; expected >=1 member")
	}
	for i, m := range members {
		if m.UUID == "" {
			t.Errorf("members[%d].UUID empty; live server should populate it", i)
		}
		if m.AccountRole == "" {
			t.Errorf("members[%d].AccountRole empty; pre-v0.7.1 Role field was dropped at decode time, so empty here means the v0.7.1 rename did not take effect", i)
		}
	}
}

func TestIntegration_ListIntegrations_DecodesLive(t *testing.T) {
	c := liveMCPClient(t)
	ctx, cancel := withTimeout(t)
	defer cancel()
	_, err := c.ListIntegrations(ctx)
	if err != nil {
		t.Fatalf("ListIntegrations failed: %v", err)
	}
}

// serverHasTool checks whether toolName appears in the pinned tools-list
// snapshot (testdata/mcp_tools_list.json). Returns false when the snapshot
// has not been updated to include the tool yet.
func serverHasTool(t *testing.T, toolName string) bool {
	t.Helper()
	snapshot := loadToolsListSnapshot(t)
	for _, tool := range snapshot.Result.Tools {
		if tool.Name == toolName {
			return true
		}
	}
	return false
}

// TestIntegration_DeleteMonitor_LiveRoundTrip verifies the full create-then-delete
// cycle against the live MCP server.
//
// The test is gated on two conditions:
//  1. HYPERPING_TEST_API_KEY must be set (standard integration gate).
//  2. delete_monitor must appear in testdata/mcp_tools_list.json (the pinned
//     server snapshot). The Hyperping MCP server did not expose delete_monitor
//     as of 2026-06-12; once the server adds the tool, re-capture the snapshot
//     with tools/list and the test will run automatically.
//
// Cleanup: the defer block calls DeleteMonitor a second time so a mid-test
// panic does not orphan the created monitor. The second call is expected to
// return ErrNotFound and is intentionally ignored.
func TestIntegration_DeleteMonitor_LiveRoundTrip(t *testing.T) {
	if !serverHasTool(t, "delete_monitor") {
		t.Skip("delete_monitor not in testdata/mcp_tools_list.json; re-capture snapshot when the server exposes the tool")
	}

	c := liveMCPClient(t)
	ctx, cancel := withTimeout(t)
	defer cancel()

	// Create a disposable monitor that will be deleted by this test.
	created, err := c.CreateMonitor(ctx, MCPCreateMonitorRequest{
		Name:      "integration-test-delete-98a932",
		URL:       "https://httpbin.org/status/200",
		Method:    "GET",
		Frequency: 60,
	})
	if err != nil {
		t.Fatalf("CreateMonitor failed: %v", err)
	}
	if created == nil || created.UUID == "" {
		t.Fatal("CreateMonitor returned nil or empty UUID")
	}

	// Best-effort cleanup: covers the case where the test fails after Create
	// but before Delete so the monitor is not left orphaned indefinitely.
	defer func() {
		_ = c.DeleteMonitor(context.Background(), created.UUID)
	}()

	// Delete the monitor.
	if err := c.DeleteMonitor(ctx, created.UUID); err != nil {
		t.Fatalf("DeleteMonitor failed: %v", err)
	}

	// Verify: a subsequent GetMonitor must return ErrNotFound.
	_, err = c.GetMonitor(ctx, created.UUID)
	if err == nil {
		t.Fatal("GetMonitor after delete: expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetMonitor after delete: expected ErrNotFound, got %v", err)
	}
}
