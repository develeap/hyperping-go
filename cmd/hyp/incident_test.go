// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hyperping "github.com/develeap/hyperping-go"
)

func TestIncidentCreate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(hyperping.Incident{UUID: "inci_001"})
	}))
	defer server.Close()

	out, _, err := execCmd(server,
		"incident", "create",
		"--title", "DB outage",
		"--type", "outage",
		"--statuspage", "sp_abc",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "inci_001") {
		t.Errorf("expected incident UUID in output, got: %q", out)
	}
}

func TestIncidentCreate_MissingFlags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _, err := execCmd(server, "incident", "create")
	if err == nil {
		t.Error("expected error for missing flags, got nil")
	}
}

func TestIncidentResolve_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(hyperping.Incident{UUID: "inci_001"})
	}))
	defer server.Close()

	out, _, err := execCmd(server, "incident", "resolve", "inci_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "inci_001") || !strings.Contains(out, "resolved") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestIncidentResolve_MissingUUID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _, err := execCmd(server, "incident", "resolve")
	if err == nil {
		t.Error("expected error for missing UUID, got nil")
	}
}

func TestIncidentResolve_WithMessage(t *testing.T) {
	var capturedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/updates") {
			b, _ := io.ReadAll(r.Body)
			capturedBody = string(b)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(hyperping.Incident{UUID: "inci_001"})
	}))
	defer server.Close()

	_, _, err := execCmd(server, "incident", "resolve", "inci_001", "--message", "All clear")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedBody, "All clear") {
		t.Errorf("expected resolution message in request body, got: %s", capturedBody)
	}
}
