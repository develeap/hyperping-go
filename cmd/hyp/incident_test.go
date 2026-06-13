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

func TestIncidentCreate_Success(t *testing.T) {
	created := hyperping.Incident{
		UUID:        "inci_001",
		Title:       hyperping.LocalizedText{En: "Elevated error rates"},
		Text:        hyperping.LocalizedText{En: "We are investigating."},
		Type:        "monitoring",
		StatusPages: []string{"sp_abc"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(created)
	}))
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{
		"--api-key", "test_key",
		"--base-url", server.URL,
		"incident", "create",
		"--title", "Elevated error rates",
		"--text", "We are investigating.",
		"--type", "monitoring",
		"--statuspage", "sp_abc",
	})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "inci_001")
}

func TestIncidentCreate_MissingFlags(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{
		"--api-key", "test_key",
		"incident", "create",
		"--title", "Partial",
	})
	err := cmd.Execute()
	require.Error(t, err)
}

func TestIncidentResolve_Success(t *testing.T) {
	resolved := hyperping.Incident{
		UUID: "inci_001",
		Updates: []hyperping.IncidentUpdate{
			{UUID: "upd_001", Type: "resolved"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resolved)
	}))
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{
		"--api-key", "test_key",
		"--base-url", server.URL,
		"incident", "resolve", "inci_001",
	})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "inci_001")
}

func TestIncidentResolve_MissingUUID(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--api-key", "test_key", "incident", "resolve"})
	err := cmd.Execute()
	require.Error(t, err)
}

func TestIncidentResolve_WithMessage(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = readBody(r)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(hyperping.Incident{UUID: "inci_001"})
	}))
	defer server.Close()

	buf := &bytes.Buffer{}
	cmd := newRootCmd("dev", "none", "unknown")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{
		"--api-key", "test_key",
		"--base-url", server.URL,
		"incident", "resolve", "inci_001",
		"--message", "All clear",
	})
	err := cmd.Execute()
	require.NoError(t, err)

	var update hyperping.AddIncidentUpdateRequest
	require.NoError(t, json.Unmarshal(capturedBody, &update))
	assert.Equal(t, "resolved", update.Type)
	assert.Equal(t, "All clear", update.Text.En)
}
