// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hyperping "github.com/develeap/hyperping-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// slugify unit tests (WI-2)
// ---------------------------------------------------------------------------

func TestSlugify_Simple(t *testing.T) {
	assert.Equal(t, "my-company", slugify("My Company"))
}

func TestSlugify_SpecialChars(t *testing.T) {
	assert.Equal(t, "acme-corp-2024", slugify("Acme Corp! (2024)"))
}

func TestSlugify_Truncate(t *testing.T) {
	input := strings.Repeat("a", 70)
	result := slugify(input)
	assert.Len(t, result, 63)
}

func TestSlugify_LeadingTrailingHyphens(t *testing.T) {
	assert.Equal(t, "foo", slugify(" --foo-- "))
}

func TestSlugify_Empty(t *testing.T) {
	assert.Equal(t, "", slugify(""))
}

// ---------------------------------------------------------------------------
// tenant onboard command tests (WI-4)
// ---------------------------------------------------------------------------

// spWrapped wraps a StatusPage in the envelope the Hyperping API returns.
func spWrapped(page hyperping.StatusPage) map[string]interface{} {
	return map[string]interface{}{"statuspage": page}
}

// tenantTestServer builds a test server that handles the three API calls
// made by tenant onboard. It records call order for sequencing assertions.
func tenantTestServer(
	t *testing.T,
	spResp hyperping.StatusPage,
	monResp func(n int) hyperping.Monitor, // called per monitor creation; n is 1-based counter
	updateResp hyperping.StatusPage,
	statusPageCode int,
	monitorCodeFn func(n int) int,
	calls *[]string,
) *httptest.Server {
	t.Helper()
	monCount := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/statuspages":
			*calls = append(*calls, "create_sp")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusPageCode)
			if statusPageCode == http.StatusOK {
				json.NewEncoder(w).Encode(spWrapped(spResp))
			} else {
				w.Write([]byte(`{"error":"subdomain taken"}`))
			}

		case r.Method == http.MethodPost && r.URL.Path == "/v1/monitors":
			monCount++
			n := monCount
			*calls = append(*calls, "create_mon")
			code := http.StatusOK
			if monitorCodeFn != nil {
				code = monitorCodeFn(n)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(code)
			if code == http.StatusOK && monResp != nil {
				json.NewEncoder(w).Encode(monResp(n))
			} else {
				w.Write([]byte(`{"error":"server error"}`))
			}

		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/v2/statuspages/"):
			*calls = append(*calls, "update_sp")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(spWrapped(updateResp))

		default:
			t.Logf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestTenantOnboard_StatusPageOnly(t *testing.T) {
	spResp := hyperping.StatusPage{
		UUID:            "sp_001",
		Name:            "Acme Corp",
		HostedSubdomain: "acme-corp",
	}
	var calls []string
	server := tenantTestServer(t, spResp, nil, hyperping.StatusPage{}, http.StatusOK, nil, &calls)
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "test_key", "--base-url", server.URL, "tenant", "onboard", "Acme Corp"})
	err := cmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "sp_001")
	assert.Contains(t, out, "Acme Corp")
	assert.Equal(t, []string{"create_sp"}, calls)
}

func TestTenantOnboard_WithMonitors(t *testing.T) {
	spResp := hyperping.StatusPage{UUID: "sp_001", Name: "Acme Corp", HostedSubdomain: "acme-corp"}
	monResponses := []hyperping.Monitor{
		{UUID: "mon_001"},
		{UUID: "mon_002"},
	}
	updateResp := hyperping.StatusPage{UUID: "sp_001", Name: "Acme Corp", HostedSubdomain: "acme-corp", Monitors: []string{"mon_001", "mon_002"}}

	var calls []string
	var capturedUpdateBody []byte

	monIdx := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/statuspages":
			calls = append(calls, "create_sp")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(spWrapped(spResp))

		case r.Method == http.MethodPost && r.URL.Path == "/v1/monitors":
			n := monIdx
			monIdx++
			calls = append(calls, "create_mon")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(monResponses[n])

		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/v2/statuspages/"):
			calls = append(calls, "update_sp")
			body, _ := readBody(r)
			capturedUpdateBody = body
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(spWrapped(updateResp))

		default:
			t.Logf("unexpected: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{
		"--api-key", "test_key", "--base-url", server.URL,
		"tenant", "onboard", "Acme Corp",
		"--monitor-url", "https://api.acme.com",
		"--monitor-url", "https://web.acme.com",
	})
	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, []string{"create_sp", "create_mon", "create_mon", "update_sp"}, calls)

	var updateReq map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedUpdateBody, &updateReq))
	monitors, ok := updateReq["monitors"].([]interface{})
	require.True(t, ok, "monitors field missing from update body")
	assert.Len(t, monitors, 2)
	assert.Equal(t, "mon_001", monitors[0])
	assert.Equal(t, "mon_002", monitors[1])

	out := buf.String()
	assert.Contains(t, out, "sp_001")
	assert.Contains(t, out, "Acme Corp")
}

func TestTenantOnboard_WithMonitors_JSON(t *testing.T) {
	spResp := hyperping.StatusPage{UUID: "sp_001", Name: "Acme Corp", HostedSubdomain: "acme-corp"}
	monResponses := []hyperping.Monitor{{UUID: "mon_001"}, {UUID: "mon_002"}}
	updateResp := hyperping.StatusPage{UUID: "sp_001", Name: "Acme Corp", HostedSubdomain: "acme-corp", Monitors: []string{"mon_001", "mon_002"}}

	monIdx2 := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/statuspages":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(spWrapped(spResp))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/monitors":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(monResponses[monIdx2])
			monIdx2++
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/v2/statuspages/"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(spWrapped(updateResp))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{
		"--api-key", "test_key", "--base-url", server.URL, "--output", "json",
		"tenant", "onboard", "Acme Corp",
		"--monitor-url", "https://api.acme.com",
		"--monitor-url", "https://web.acme.com",
	})
	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "sp_001", result["uuid"])
	assert.Equal(t, "Acme Corp", result["name"])
	assert.Equal(t, "acme-corp", result["subdomain"])
	monitors, ok := result["monitors"].([]interface{})
	require.True(t, ok)
	assert.Len(t, monitors, 2)
}

func TestTenantOnboard_StatusPageError(t *testing.T) {
	var calls []string
	server := tenantTestServer(t, hyperping.StatusPage{}, nil, hyperping.StatusPage{}, http.StatusUnprocessableEntity, nil, &calls)
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "test_key", "--base-url", server.URL, "tenant", "onboard", "Acme Corp"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Equal(t, []string{"create_sp"}, calls)
}

func TestTenantOnboard_MonitorError(t *testing.T) {
	spResp := hyperping.StatusPage{UUID: "sp_001", Name: "Acme Corp", HostedSubdomain: "acme-corp"}

	monErrCount := 0
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/statuspages":
			calls = append(calls, "create_sp")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(spWrapped(spResp))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/monitors":
			monErrCount++
			calls = append(calls, "create_mon")
			if monErrCount == 1 {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(hyperping.Monitor{UUID: "mon_001"})
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"server error"}`))
			}
		default:
			calls = append(calls, "update_sp")
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{
		"--api-key", "test_key", "--base-url", server.URL,
		"tenant", "onboard", "Acme Corp",
		"--monitor-url", "https://api.acme.com",
		"--monitor-url", "https://web.acme.com",
	})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Equal(t, []string{"create_sp", "create_mon", "create_mon"}, calls)
}

func TestTenantOnboard_MissingName(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "test_key", "tenant", "onboard"})
	err := cmd.Execute()
	require.Error(t, err)
}
