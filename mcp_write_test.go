// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// errorMCPTransport is a transport stub that always returns the configured error.
type errorMCPTransport struct {
	err error
}

func (e *errorMCPTransport) Initialize(ctx context.Context) (map[string]any, error) {
	return nil, e.err
}

func (e *errorMCPTransport) CallTool(ctx context.Context, toolName string, args map[string]any) (any, error) {
	return nil, e.err
}

func float64Ptr(f float64) *float64 { return &f }

// ============================================================
// CreateMonitor
// ============================================================

func TestCreateMonitor_SendsCorrectArgs(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{
		"uuid": "mon_abc",
		"name": "My Monitor",
		"url":  "https://example.com",
	}}
	c := NewMCPClient(tr)

	req := MCPCreateMonitorRequest{
		Name:               "My Monitor",
		URL:                "https://example.com",
		Protocol:           strPtr("http"),
		HTTPMethod:         strPtr("GET"),
		CheckFrequency:     float64Ptr(60),
		ExpectedStatusCode: strPtr("2xx"),
		RequiredKeyword:    strPtr("ok"),
		EscalationPolicy:   strPtr("ep-uuid"),
	}
	_, err := c.CreateMonitor(context.Background(), req)
	require.NoError(t, err)

	require.Equal(t, "create_monitor", tr.lastTool)
	require.Equal(t, "My Monitor", tr.lastArgs["name"])
	require.Equal(t, "https://example.com", tr.lastArgs["url"])
	require.Equal(t, "http", tr.lastArgs["protocol"])
	require.Equal(t, "GET", tr.lastArgs["http_method"])
	require.Equal(t, float64(60), tr.lastArgs["check_frequency"])
	require.Equal(t, "2xx", tr.lastArgs["expected_status_code"])
	require.Equal(t, "ok", tr.lastArgs["required_keyword"])
	require.Equal(t, "ep-uuid", tr.lastArgs["escalation_policy"])

	// Old incorrect arg names must NOT be present.
	_, hasMethod := tr.lastArgs["method"]
	require.False(t, hasMethod, "must not send legacy 'method' key")
	_, hasFreq := tr.lastArgs["frequency"]
	require.False(t, hasFreq, "must not send legacy 'frequency' key")
	_, hasStatus := tr.lastArgs["expected_status"]
	require.False(t, hasStatus, "must not send legacy 'expected_status' key")
	_, hasKw := tr.lastArgs["keyword"]
	require.False(t, hasKw, "must not send legacy 'keyword' key")
}

func TestCreateMonitor_MinimalArgs(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{
		"uuid": "mon_abc",
		"name": "Minimal",
		"url":  "https://example.com",
	}}
	c := NewMCPClient(tr)

	req := MCPCreateMonitorRequest{
		Name: "Minimal",
		URL:  "https://example.com",
	}
	_, err := c.CreateMonitor(context.Background(), req)
	require.NoError(t, err)

	require.Equal(t, "create_monitor", tr.lastTool)
	require.Equal(t, "Minimal", tr.lastArgs["name"])
	require.Equal(t, "https://example.com", tr.lastArgs["url"])
	require.Len(t, tr.lastArgs, 2, "only name+url should be sent for minimal request")
}

func TestCreateMonitor_RequestHeaders(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{}}
	c := NewMCPClient(tr)

	req := MCPCreateMonitorRequest{
		Name: "With Headers",
		URL:  "https://example.com",
		RequestHeaders: []MCPRequestHeader{
			{Name: "Authorization", Value: "Bearer token"},
			{Name: "X-Custom", Value: "val"},
		},
	}
	_, err := c.CreateMonitor(context.Background(), req)
	require.NoError(t, err)

	headers, ok := tr.lastArgs["request_headers"].([]MCPRequestHeader)
	require.True(t, ok, "request_headers must be []MCPRequestHeader, got %T", tr.lastArgs["request_headers"])
	require.Len(t, headers, 2)
	require.Equal(t, "Authorization", headers[0].Name)
	require.Equal(t, "Bearer token", headers[0].Value)
	require.Equal(t, "X-Custom", headers[1].Name)
}

func TestCreateMonitor_DecodesResponse(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{
		"uuid":   "mon_xyz",
		"name":   "Decoded Monitor",
		"url":    "https://example.com",
		"status": "up",
	}}
	c := NewMCPClient(tr)

	resp, err := c.CreateMonitor(context.Background(), MCPCreateMonitorRequest{
		Name: "Decoded Monitor",
		URL:  "https://example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "mon_xyz", resp.UUID)
	require.Equal(t, "Decoded Monitor", resp.Name)
}

func TestCreateMonitor_NilResponse(t *testing.T) {
	tr := &recordingMCPTransport{result: "not a map"}
	c := NewMCPClient(tr)

	resp, err := c.CreateMonitor(context.Background(), MCPCreateMonitorRequest{
		Name: "Test",
		URL:  "https://example.com",
	})
	require.NoError(t, err)
	require.Nil(t, resp)
}

// ============================================================
// UpdateMonitor
// ============================================================

func TestUpdateMonitor_SendsCorrectArgs(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{
		"uuid": "mon_upd",
		"name": "Updated",
	}}
	c := NewMCPClient(tr)

	req := MCPUpdateMonitorRequest{
		Name:           strPtr("Updated"),
		CheckFrequency: float64Ptr(30),
	}
	_, err := c.UpdateMonitor(context.Background(), "mon_upd", req)
	require.NoError(t, err)

	require.Equal(t, "update_monitor", tr.lastTool)
	require.Equal(t, "mon_upd", tr.lastArgs["uuid"])
	require.Equal(t, "Updated", tr.lastArgs["name"])
	require.Equal(t, float64(30), tr.lastArgs["check_frequency"])

	// Old incorrect arg names must NOT be present.
	_, hasFreq := tr.lastArgs["frequency"]
	require.False(t, hasFreq, "must not send legacy 'frequency' key")
}

func TestUpdateMonitor_UUIDAlwaysPresent(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{}}
	c := NewMCPClient(tr)

	_, err := c.UpdateMonitor(context.Background(), "mon_only", MCPUpdateMonitorRequest{})
	require.NoError(t, err)

	require.Equal(t, "update_monitor", tr.lastTool)
	require.Equal(t, "mon_only", tr.lastArgs["uuid"])
	require.Len(t, tr.lastArgs, 1, "only uuid should be sent when no fields changed")
}

// ============================================================
// PauseMonitor
// ============================================================

func TestPauseMonitor_SendsCorrectArgs(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{}}
	c := NewMCPClient(tr)

	err := c.PauseMonitor(context.Background(), "mon_pause")
	require.NoError(t, err)

	require.Equal(t, "pause_monitor", tr.lastTool)
	require.Equal(t, "mon_pause", tr.lastArgs["uuid"])
	require.Len(t, tr.lastArgs, 1)
}

func TestPauseMonitor_TransportError(t *testing.T) {
	tr := &errorMCPTransport{err: errors.New("connection refused")}
	c := NewMCPClient(tr)

	err := c.PauseMonitor(context.Background(), "mon_x")
	require.Error(t, err)
	require.Contains(t, err.Error(), "connection refused")
}

// ============================================================
// ResumeMonitor
// ============================================================

func TestResumeMonitor_SendsCorrectArgs(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{}}
	c := NewMCPClient(tr)

	err := c.ResumeMonitor(context.Background(), "mon_resume")
	require.NoError(t, err)

	require.Equal(t, "resume_monitor", tr.lastTool)
	require.Equal(t, "mon_resume", tr.lastArgs["uuid"])
	require.Len(t, tr.lastArgs, 1)
}

func TestResumeMonitor_TransportError(t *testing.T) {
	tr := &errorMCPTransport{err: errors.New("timeout")}
	c := NewMCPClient(tr)

	err := c.ResumeMonitor(context.Background(), "mon_y")
	require.Error(t, err)
	require.Contains(t, err.Error(), "timeout")
}

// ============================================================
// DeleteMonitor
// ============================================================

func TestDeleteMonitor_SendsCorrectArgs(t *testing.T) {
	tr := &recordingMCPTransport{result: map[string]any{}}
	c := NewMCPClient(tr)

	err := c.DeleteMonitor(context.Background(), "mon_del")
	require.NoError(t, err)

	require.Equal(t, "delete_monitor", tr.lastTool)
	require.Equal(t, "mon_del", tr.lastArgs["uuid"])
	require.Len(t, tr.lastArgs, 1)
}

func TestDeleteMonitor_TransportError(t *testing.T) {
	tr := &errorMCPTransport{err: errors.New("not found")}
	c := NewMCPClient(tr)

	err := c.DeleteMonitor(context.Background(), "mon_z")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// ============================================================
// Transport-error coverage for Create and Update
// ============================================================

func TestCreateMonitor_TransportError(t *testing.T) {
	tr := &errorMCPTransport{err: errors.New("dial tcp: connection refused")}
	c := NewMCPClient(tr)

	_, err := c.CreateMonitor(context.Background(), MCPCreateMonitorRequest{
		Name: "Test",
		URL:  "https://example.com",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "connection refused")
}

func TestUpdateMonitor_TransportError(t *testing.T) {
	tr := &errorMCPTransport{err: errors.New("unauthorized")}
	c := NewMCPClient(tr)

	_, err := c.UpdateMonitor(context.Background(), "mon_upd", MCPUpdateMonitorRequest{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")
}
