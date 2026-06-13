// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand_DefaultHelp(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "hyp")
	assert.Contains(t, out, "monitor")
	assert.Contains(t, out, "incident")
	assert.Contains(t, out, "statuspage")
}

func TestRootCommand_APIKeyFromFlag(t *testing.T) {
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "flagkey", "--base-url", server.URL, "monitor", "list"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Bearer flagkey", capturedAuth)
}

func TestRootCommand_APIKeyFromEnv(t *testing.T) {
	t.Setenv("HYPERPING_API_KEY", "envkey")

	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--base-url", server.URL, "monitor", "list"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Bearer envkey", capturedAuth)
}

func TestRootCommand_FlagOverridesEnv(t *testing.T) {
	t.Setenv("HYPERPING_API_KEY", "envkey")

	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "flagkey", "--base-url", server.URL, "monitor", "list"})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Bearer flagkey", capturedAuth)
}

func TestRootCommand_MissingAPIKey(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"monitor", "list"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API key required")
}
