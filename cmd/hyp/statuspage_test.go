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

func TestStatuspageList_Table(t *testing.T) {
	resp := hyperping.StatusPagePaginatedResponse{
		StatusPages: []hyperping.StatusPage{
			{UUID: "sp_001", Name: "Acme Status", HostedSubdomain: "acme.hyperping.app", URL: "https://acme.hyperping.app"},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	out, _, err := execCmd(server, "statuspage", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"UUID", "sp_001", "Acme Status"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q; got: %q", want, out)
		}
	}
}

func TestStatuspageList_JSON(t *testing.T) {
	resp := hyperping.StatusPagePaginatedResponse{
		StatusPages: []hyperping.StatusPage{
			{UUID: "sp_001", Name: "Acme Status"},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	out, _, err := execCmd(server, "--output", "json", "statuspage", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, out)
	}
	if len(result) == 0 {
		t.Error("expected at least one status page in JSON output")
	}
}

func TestStatuspageShow_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"statuspage": hyperping.StatusPage{UUID: "sp_001", Name: "Acme Status", HostedSubdomain: "acme.hyperping.app"},
		})
	}))
	defer server.Close()

	out, _, err := execCmd(server, "statuspage", "show", "sp_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "sp_001") {
		t.Errorf("expected sp_001 in output, got: %q", out)
	}
}

func TestStatuspageShow_MissingUUID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _, err := execCmd(server, "statuspage", "show")
	if err == nil {
		t.Error("expected error for missing UUID, got nil")
	}
}
