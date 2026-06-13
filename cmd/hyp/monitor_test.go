// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	hyperping "github.com/develeap/hyperping-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMonitorServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func TestMonitorList_Table(t *testing.T) {
	monitors := []hyperping.Monitor{
		{UUID: "mon_001", Name: "Prod API", URL: "https://api.example.com", Status: "up", Paused: false},
		{UUID: "mon_002", Name: "Staging", URL: "https://staging.example.com", Status: "down", Paused: true},
	}

	server := newTestMonitorServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(monitors)
	})
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "test_key", "--base-url", server.URL, "monitor", "list"})
	err := cmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "mon_001")
	assert.Contains(t, out, "Prod API")
	assert.Contains(t, out, "mon_002")
	assert.Contains(t, out, "Staging")
}

func TestMonitorList_JSON(t *testing.T) {
	monitors := []hyperping.Monitor{
		{UUID: "mon_001", Name: "Prod API", URL: "https://api.example.com"},
	}

	server := newTestMonitorServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(monitors)
	})
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "test_key", "--base-url", server.URL, "--output", "json", "monitor", "list"})
	err := cmd.Execute()
	require.NoError(t, err)

	var result []hyperping.Monitor
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "mon_001", result[0].UUID)
}

func TestMonitorList_Empty(t *testing.T) {
	server := newTestMonitorServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	})
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "test_key", "--base-url", server.URL, "monitor", "list"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "no monitors found")
}

func TestMonitorPause_Success(t *testing.T) {
	paused := hyperping.Monitor{UUID: "mon_001", Name: "Prod API", Paused: true}

	server := newTestMonitorServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(paused)
	})
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "test_key", "--base-url", server.URL, "monitor", "pause", "mon_001"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "mon_001")
}

func TestMonitorPause_MissingUUID(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "test_key", "monitor", "pause"})
	err := cmd.Execute()
	require.Error(t, err)
}

func TestMonitorResume_Success(t *testing.T) {
	resumed := hyperping.Monitor{UUID: "mon_001", Name: "Prod API", Paused: false}

	server := newTestMonitorServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resumed)
	})
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "test_key", "--base-url", server.URL, "monitor", "resume", "mon_001"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "mon_001")
}

func TestMonitorResume_MissingUUID(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "test_key", "monitor", "resume"})
	err := cmd.Execute()
	require.Error(t, err)
}
