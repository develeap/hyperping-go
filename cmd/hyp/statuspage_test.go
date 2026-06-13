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

func TestStatuspageShow_Success(t *testing.T) {
	page := hyperping.StatusPage{
		UUID: "sp_abc123",
		Name: "Production Status",
		URL:  "https://status.example.com",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GetStatusPage returns {"statuspage": {...}}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"statuspage": page})
	}))
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "test_key", "--base-url", server.URL, "statuspage", "show", "sp_abc123"})
	err := cmd.Execute()
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "sp_abc123")
	assert.Contains(t, out, "Production Status")
	assert.Contains(t, out, "https://status.example.com")
}

func TestStatuspageShow_MissingUUID(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "test_key", "statuspage", "show"})
	err := cmd.Execute()
	require.Error(t, err)
}

func TestStatuspageList_Table(t *testing.T) {
	resp := hyperping.StatusPagePaginatedResponse{
		StatusPages: []hyperping.StatusPage{
			{UUID: "sp_001", Name: "Status Alpha", URL: "https://alpha.status.io"},
			{UUID: "sp_002", Name: "Status Beta", URL: "https://beta.status.io"},
		},
		Total: 2,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "test_key", "--base-url", server.URL, "statuspage", "list"})
	err := cmd.Execute()
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "sp_001")
	assert.Contains(t, out, "Status Alpha")
	assert.Contains(t, out, "sp_002")
	assert.Contains(t, out, "Status Beta")
}

func TestStatuspageList_JSON(t *testing.T) {
	resp := hyperping.StatusPagePaginatedResponse{
		StatusPages: []hyperping.StatusPage{
			{UUID: "sp_001", Name: "Status Alpha", URL: "https://alpha.status.io"},
		},
		Total: 1,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "test_key", "--base-url", server.URL, "--output", "json", "statuspage", "list"})
	err := cmd.Execute()
	require.NoError(t, err)

	var result []hyperping.StatusPage
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Len(t, result, 1)
	assert.Equal(t, "sp_001", result[0].UUID)
}
