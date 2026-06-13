// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hyperping "github.com/develeap/hyperping-go"
)

func TestMonitorList_Table(t *testing.T) {
	monitors := []hyperping.Monitor{
		{UUID: "mon_001", Name: "API Check", URL: "https://api.example.com", Status: "up", Paused: false},
		{UUID: "mon_002", Name: "Web Check", URL: "https://web.example.com", Status: "down", Paused: true},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(monitors)
	}))
	defer server.Close()

	out, _, err := execCmd(server, "monitor", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"UUID", "mon_001", "API Check", "mon_002"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestMonitorList_JSON(t *testing.T) {
	monitors := []hyperping.Monitor{
		{UUID: "mon_001", Name: "API Check", URL: "https://api.example.com"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(monitors)
	}))
	defer server.Close()

	out, _, err := execCmd(server, "--output", "json", "monitor", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, out)
	}
	if len(result) == 0 {
		t.Error("expected at least one monitor in JSON output")
	}
}

func TestMonitorList_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	out, _, err := execCmd(server, "monitor", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "no monitors found") {
		t.Errorf("expected empty message, got: %q", out)
	}
}

func TestMonitorPause_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(hyperping.Monitor{UUID: "mon_001", Name: "API Check", Paused: true})
	}))
	defer server.Close()

	out, _, err := execCmd(server, "monitor", "pause", "mon_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "mon_001") || !strings.Contains(out, "paused") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestMonitorPause_MissingUUID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _, err := execCmd(server, "monitor", "pause")
	if err == nil {
		t.Error("expected error for missing UUID, got nil")
	}
}

func TestMonitorResume_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(hyperping.Monitor{UUID: "mon_001", Name: "API Check", Paused: false})
	}))
	defer server.Close()

	out, _, err := execCmd(server, "monitor", "resume", "mon_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "mon_001") || !strings.Contains(out, "resumed") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestMonitorResume_MissingUUID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _, err := execCmd(server, "monitor", "resume")
	if err == nil {
		t.Error("expected error for missing UUID, got nil")
	}
}
