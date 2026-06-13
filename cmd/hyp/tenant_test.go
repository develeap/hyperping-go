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

func TestSlugify_Simple(t *testing.T) {
	if got := slugify("Acme Corp"); got != "acme-corp" {
		t.Errorf("want %q, got %q", "acme-corp", got)
	}
}

func TestSlugify_SpecialChars(t *testing.T) {
	if got := slugify("Acme & Corp!"); got != "acme-corp" {
		t.Errorf("want %q, got %q", "acme-corp", got)
	}
}

func TestSlugify_Truncate(t *testing.T) {
	long := strings.Repeat("a", 80)
	got := slugify(long)
	if len(got) > 63 {
		t.Errorf("slugify should truncate to 63 chars, got %d", len(got))
	}
}

func TestSlugify_LeadingTrailingHyphens(t *testing.T) {
	if got := slugify("  --hello-- world--  "); got != "hello-world" {
		t.Errorf("want %q, got %q", "hello-world", got)
	}
}

func TestSlugify_Empty(t *testing.T) {
	if got := slugify(""); got != "" {
		t.Errorf("want empty string, got %q", got)
	}
}

func TestTenantOnboard_StatusPageOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Status page created",
			"statuspage": hyperping.StatusPage{
				UUID:            "sp_001",
				Name:            "Acme Corp",
				HostedSubdomain: "acme-corp.hyperping.app",
			},
		})
	}))
	defer server.Close()

	out, _, err := execCmd(server, "tenant", "onboard", "Acme Corp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "sp_001") {
		t.Errorf("expected status page UUID in output, got: %q", out)
	}
}

func TestTenantOnboard_WithMonitors(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "statuspages"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"statuspage": hyperping.StatusPage{UUID: "sp_001", Name: "Acme"},
			})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "monitors"):
			json.NewEncoder(w).Encode(hyperping.Monitor{UUID: "mon_001", Name: "Acme - https://acme.com"})
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "statuspages"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"statuspage": hyperping.StatusPage{UUID: "sp_001"},
			})
		}
	}))
	defer server.Close()

	out, _, err := execCmd(server, "tenant", "onboard", "Acme", "--monitor-url", "https://acme.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "sp_001") {
		t.Errorf("expected status page UUID, got: %q", out)
	}
	if !strings.Contains(out, "mon_001") {
		t.Errorf("expected monitor UUID, got: %q", out)
	}
}

func TestTenantOnboard_WithMonitors_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "statuspages"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"statuspage": hyperping.StatusPage{UUID: "sp_001", Name: "Acme", HostedSubdomain: "acme.hyperping.app"},
			})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "monitors"):
			json.NewEncoder(w).Encode(hyperping.Monitor{UUID: "mon_001"})
		case r.Method == http.MethodPut:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"statuspage": hyperping.StatusPage{UUID: "sp_001"},
			})
		}
	}))
	defer server.Close()

	out, _, err := execCmd(server, "--output", "json", "tenant", "onboard", "Acme", "--monitor-url", "https://acme.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, out)
	}
	if result["uuid"] != "sp_001" {
		t.Errorf("expected uuid=sp_001, got: %v", result["uuid"])
	}
	monitors, ok := result["monitors"].([]interface{})
	if !ok || len(monitors) == 0 {
		t.Errorf("expected monitors array in JSON, got: %v", result["monitors"])
	}
}

func TestTenantOnboard_StatusPageError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer server.Close()

	_, _, err := execCmd(server, "tenant", "onboard", "Acme")
	if err == nil {
		t.Error("expected error on status page creation failure, got nil")
	}
}

func TestTenantOnboard_MonitorError(t *testing.T) {
	monitorCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "statuspages") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"statuspage": hyperping.StatusPage{UUID: "sp_001"},
			})
			return
		}
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "monitors") {
			monitorCalls++
			if monitorCalls == 1 {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(hyperping.Monitor{UUID: "mon_001"})
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"failed"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _, err := execCmd(server, "tenant", "onboard", "Acme",
		"--monitor-url", "https://ok.com",
		"--monitor-url", "https://fail.com",
	)
	if err == nil {
		t.Error("expected error on monitor creation failure, got nil")
	}
}

func TestTenantOnboard_MissingName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _, err := execCmd(server, "tenant", "onboard")
	if err == nil {
		t.Error("expected error for missing name argument, got nil")
	}
}
