// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// recordingMCPTransport captures the toolName and args of the last CallTool
// invocation so request-side assertions can verify the client sends the right
// arg names + types. Returns a configurable result or error.
type recordingMCPTransport struct {
	result       any
	callErr      error // returned by CallTool when non-nil
	lastTool     string
	lastArgs     map[string]any
	callCount    int
	initCount    int
}

func (m *recordingMCPTransport) Initialize(ctx context.Context) (map[string]any, error) {
	m.initCount++
	return map[string]any{"protocolVersion": "2025-03-26"}, nil
}

func (m *recordingMCPTransport) CallTool(ctx context.Context, toolName string, args map[string]any) (any, error) {
	m.callCount++
	m.lastTool = toolName
	m.lastArgs = args
	return m.result, m.callErr
}

// ============================================================
// Request-arg correctness (the v0.6.3 bug: wrong arg names)
// ============================================================

func TestGetMonitorMtta_SendsMonitorUuidsArray(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{
		"monitors":          []any{},
		"totalAcknowledged": 0,
		"mtta":              0,
	}}
	c := NewMCPClient(tr)

	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	_, err := c.GetMonitorMtta(context.Background(), from, to, "mon_a", "mon_b")
	require.NoError(t, err)

	require.Equal(t, "get_monitor_mtta", tr.lastTool)
	uuids, ok := tr.lastArgs["monitor_uuids"].([]string)
	require.True(t, ok, "monitor_uuids must be []string, got %T", tr.lastArgs["monitor_uuids"])
	require.Equal(t, []string{"mon_a", "mon_b"}, uuids)
	require.Equal(t, "2026-06-01T00:00:00Z", tr.lastArgs["from"])
	require.Equal(t, "2026-06-08T00:00:00Z", tr.lastArgs["to"])
	_, hasUUID := tr.lastArgs["uuid"]
	require.False(t, hasUUID, "client must not send the legacy 'uuid' key")
}

func TestGetMonitorMttr_SendsMonitorUuidsArray(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{
		"monitors":           []any{},
		"totalOutages":       0,
		"totalOutagesLength": 0,
		"mttr":               0,
		"mtta":               0,
	}}
	c := NewMCPClient(tr)

	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	_, err := c.GetMonitorMttr(context.Background(), from, to, "mon_a")
	require.NoError(t, err)

	require.Equal(t, "get_monitor_mttr", tr.lastTool)
	uuids, ok := tr.lastArgs["monitor_uuids"].([]string)
	require.True(t, ok)
	require.Equal(t, []string{"mon_a"}, uuids)
	_, hasLegacy := tr.lastArgs["uuid"]
	require.False(t, hasLegacy)
}

func TestGetMonitorResponseTime_SendsMonitorUuidsArray(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{
		"timeGroups":      []any{},
		"avgResponseTime": 0,
		"p95ResponseTime": 0,
		"monitors":        []any{},
	}}
	c := NewMCPClient(tr)

	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	_, err := c.GetMonitorResponseTime(context.Background(), from, to, "mon_a", "mon_b")
	require.NoError(t, err)

	require.Equal(t, "get_monitor_response_time", tr.lastTool)
	uuids, ok := tr.lastArgs["monitor_uuids"].([]string)
	require.True(t, ok)
	require.Equal(t, []string{"mon_a", "mon_b"}, uuids)
	_, hasLegacy := tr.lastArgs["uuid"]
	require.False(t, hasLegacy)
}

func TestGetMonitorUptime_SendsMonitorUuidsArray(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{
		"monitors":           []any{},
		"periodAverages":     []any{},
		"totalOutages":       0,
		"totalOutagesLength": 0,
		"MTTR":               0,
		"averageUptime":      "100%",
	}}
	c := NewMCPClient(tr)

	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	_, err := c.GetMonitorUptime(context.Background(), from, to, "mon_a")
	require.NoError(t, err)

	require.Equal(t, "get_monitor_uptime", tr.lastTool)
	uuids, ok := tr.lastArgs["monitor_uuids"].([]string)
	require.True(t, ok)
	require.Equal(t, []string{"mon_a"}, uuids)
	_, hasLegacy := tr.lastArgs["monitor_uuid"]
	require.False(t, hasLegacy, "client must not send the legacy 'monitor_uuid' key")
}

// ============================================================
// Zero-value window semantics: omit from/to/monitor_uuids
// ============================================================

func TestGetMonitorMtta_ZeroWindowOmitsFromTo(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{"monitors": []any{}, "totalAcknowledged": 0, "mtta": 0}}
	c := NewMCPClient(tr)

	_, err := c.GetMonitorMtta(context.Background(), time.Time{}, time.Time{})
	require.NoError(t, err)
	_, hasFrom := tr.lastArgs["from"]
	_, hasTo := tr.lastArgs["to"]
	_, hasUuids := tr.lastArgs["monitor_uuids"]
	require.False(t, hasFrom, "zero from should be omitted, got %v", tr.lastArgs["from"])
	require.False(t, hasTo, "zero to should be omitted")
	require.False(t, hasUuids, "empty uuids should be omitted (server-default = all monitors)")
}

func TestGetMonitorMttr_ZeroWindowOmitsFromTo(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{
		"monitors": []any{}, "totalOutages": 0, "totalOutagesLength": 0, "mttr": 0, "mtta": 0,
	}}
	c := NewMCPClient(tr)

	_, err := c.GetMonitorMttr(context.Background(), time.Time{}, time.Time{})
	require.NoError(t, err)
	_, hasFrom := tr.lastArgs["from"]
	require.False(t, hasFrom)
}

func TestGetMonitorResponseTime_ZeroWindowOmitsFromTo(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{
		"timeGroups": []any{}, "avgResponseTime": 0, "p95ResponseTime": 0, "monitors": []any{},
	}}
	c := NewMCPClient(tr)

	_, err := c.GetMonitorResponseTime(context.Background(), time.Time{}, time.Time{})
	require.NoError(t, err)
	_, hasFrom := tr.lastArgs["from"]
	require.False(t, hasFrom)
}

func TestGetMonitorUptime_ZeroWindowOmitsFromTo(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{
		"monitors": []any{}, "periodAverages": []any{}, "totalOutages": 0,
		"totalOutagesLength": 0, "MTTR": 0, "averageUptime": "100%",
	}}
	c := NewMCPClient(tr)

	_, err := c.GetMonitorUptime(context.Background(), time.Time{}, time.Time{})
	require.NoError(t, err)
	_, hasFrom := tr.lastArgs["from"]
	require.False(t, hasFrom)
}

// ============================================================
// Response decoding into the new (correct) shape
// ============================================================

func TestGetMonitorMtta_DecodesRealServerShape(t *testing.T) {
	// Verbatim shape from /v1/mcp tools/call get_monitor_mtta (2026-06-08 probe).
	tr := &recordingMCPTransport{result: map[string]any{
		"monitors":          []any{},
		"totalAcknowledged": float64(5),
		"mtta":              float64(120.5),
	}}
	c := NewMCPClient(tr)

	r, err := c.GetMonitorMtta(context.Background(), time.Time{}, time.Time{})
	require.NoError(t, err)
	require.NotNil(t, r)
	require.Equal(t, 5, r.TotalAcknowledged)
	require.Equal(t, 120.5, r.Mtta)
	require.NotNil(t, r.Monitors)
}

func TestGetMonitorMttr_DecodesRealServerShape(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{
		"monitors": []any{
			map[string]any{
				"uuid":          "mon_a",
				"name":          "[X]",
				"protocol":      "http",
				"outageCount":   float64(2),
				"totalDowntime": float64(80401),
				"mttr":          float64(40201),
				"mtta":          float64(0),
				"longestOutage": float64(45420),
			},
		},
		"totalOutages":       float64(46),
		"totalOutagesLength": float64(972855),
		"mttr":               float64(21149),
		"mtta":               float64(0),
	}}
	c := NewMCPClient(tr)

	r, err := c.GetMonitorMttr(context.Background(), time.Time{}, time.Time{})
	require.NoError(t, err)
	require.NotNil(t, r)
	require.Len(t, r.Monitors, 1)
	require.Equal(t, "mon_a", r.Monitors[0].UUID)
	require.Equal(t, 40201.0, r.Monitors[0].Mttr)
	require.Equal(t, 45420.0, r.Monitors[0].LongestOutage)
	require.Equal(t, 46, r.TotalOutages)
	require.Equal(t, 21149.0, r.Mttr)
}

func TestGetMonitorResponseTime_DecodesRealServerShape(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{
		"timeGroups": []any{
			map[string]any{"time": "2026-06-07", "avgResponseTime": float64(459), "count": float64(26)},
		},
		"avgResponseTime": float64(462),
		"p95ResponseTime": float64(505),
		"monitors": []any{
			map[string]any{
				"uuid": "mon_a", "name": "[X]", "protocol": "http",
				"avgResponseTime": float64(474),
				"avgResponseTimeByRegion": map[string]any{
					"na": nil, "eu": float64(450),
				},
				"count": float64(25),
			},
		},
	}}
	c := NewMCPClient(tr)

	r, err := c.GetMonitorResponseTime(context.Background(), time.Time{}, time.Time{})
	require.NoError(t, err)
	require.NotNil(t, r)
	require.Equal(t, 462.0, r.AvgResponseTime)
	require.Equal(t, 505.0, r.P95ResponseTime)
	require.Len(t, r.TimeGroups, 1)
	require.Equal(t, "2026-06-07", r.TimeGroups[0].Time)
	require.Equal(t, 459.0, r.TimeGroups[0].AvgResponseTime)
	require.Equal(t, 26, r.TimeGroups[0].Count)
	require.Len(t, r.Monitors, 1)
	require.Equal(t, "mon_a", r.Monitors[0].UUID)
	require.Equal(t, 474.0, r.Monitors[0].AvgResponseTime)
	// Verify null region encoded as nil pointer
	require.Nil(t, r.Monitors[0].AvgResponseTimeByRegion["na"])
	require.NotNil(t, r.Monitors[0].AvgResponseTimeByRegion["eu"])
	require.Equal(t, 450.0, *r.Monitors[0].AvgResponseTimeByRegion["eu"])
}

func TestGetMonitorUptime_DecodesRealServerShape(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{
		"monitors": []any{
			map[string]any{
				"uuid": "mon_a", "name": "[X]", "protocol": "http",
				"uptimePeriods": []any{
					map[string]any{"date": "2026-06-07", "uptime": float64(100)},
				},
				"averageUptime": float64(100),
				"outageCount":   float64(0),
				"totalDowntime": float64(0),
				"mttr":          float64(0),
				"longestOutage": float64(0),
			},
		},
		"periodAverages": []any{
			map[string]any{"date": "2026-06-07", "uptime": float64(100)},
		},
		"totalOutages":       float64(0),
		"totalOutagesLength": float64(0),
		"MTTR":               float64(0),
		"averageUptime":      "100%",
	}}
	c := NewMCPClient(tr)

	r, err := c.GetMonitorUptime(context.Background(), time.Time{}, time.Time{})
	require.NoError(t, err)
	require.NotNil(t, r)
	require.Len(t, r.Monitors, 1)
	require.Equal(t, "mon_a", r.Monitors[0].UUID)
	require.Equal(t, 100.0, r.Monitors[0].AverageUptime)
	require.Len(t, r.Monitors[0].UptimePeriods, 1)
	require.Equal(t, "2026-06-07", r.Monitors[0].UptimePeriods[0].Date)
	require.Equal(t, "100%", r.AverageUptime)
	require.Equal(t, 0, r.TotalOutages)
}

// ============================================================
// Multi-uuid batch + ctx cancel + server error propagation
// ============================================================

func TestGetMonitorMtta_MultiUUIDBatch(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{
		"monitors": []any{}, "totalAcknowledged": 0, "mtta": 0,
	}}
	c := NewMCPClient(tr)

	uuids := []string{"mon_1", "mon_2", "mon_3", "mon_4", "mon_5"}
	_, err := c.GetMonitorMtta(context.Background(), time.Time{}, time.Time{}, uuids...)
	require.NoError(t, err)
	got, _ := tr.lastArgs["monitor_uuids"].([]string)
	require.Equal(t, uuids, got)
}

// TestWindowedMethods_CtxCancelAndServerError covers ctx cancellation and
// server-error propagation symmetrically across all four windowed methods,
// rather than just one of them. A refactor that drops error propagation
// in any of them would trip this matrix.
func TestWindowedMethods_CtxCancelAndServerError(t *testing.T) {
	type call func(ctx context.Context, c *MCPClient) error
	cases := map[string]call{
		"GetMonitorMtta": func(ctx context.Context, c *MCPClient) error {
			_, err := c.GetMonitorMtta(ctx, time.Time{}, time.Time{}, "mon_a")
			return err
		},
		"GetMonitorMttr": func(ctx context.Context, c *MCPClient) error {
			_, err := c.GetMonitorMttr(ctx, time.Time{}, time.Time{}, "mon_a")
			return err
		},
		"GetMonitorResponseTime": func(ctx context.Context, c *MCPClient) error {
			_, err := c.GetMonitorResponseTime(ctx, time.Time{}, time.Time{}, "mon_a")
			return err
		},
		"GetMonitorUptime": func(ctx context.Context, c *MCPClient) error {
			_, err := c.GetMonitorUptime(ctx, time.Time{}, time.Time{}, "mon_a")
			return err
		},
	}
	for name, fn := range cases {
		t.Run(name+"/ctx_cancel", func(t *testing.T) {
			c := NewMCPClient(&errMCPTransport{err: context.Canceled})
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			require.Error(t, fn(ctx, c))
		})
		t.Run(name+"/server_error", func(t *testing.T) {
			c := NewMCPClient(&errMCPTransport{err: ErrServerError})
			require.Error(t, fn(context.Background(), c))
		})
	}
}

// errMCPTransport implements MCPTransport and returns a configured error.
type errMCPTransport struct{ err error }

func (e *errMCPTransport) Initialize(ctx context.Context) (map[string]any, error) {
	return nil, e.err
}
func (e *errMCPTransport) CallTool(ctx context.Context, toolName string, args map[string]any) (any, error) {
	return nil, e.err
}
