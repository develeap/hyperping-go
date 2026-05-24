// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MPL-2.0

package hyperping

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestListOutages_NoOpts_OmitsStatusParam verifies that when ListOutages is
// invoked without options, no "status" query parameter is sent. This preserves
// the server-side default behaviour and keeps the call backward-compatible.
func TestListOutages_NoOpts_OmitsStatusParam(t *testing.T) {
	var observed []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed = append(observed, r.URL.RawQuery)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"outages":     []Outage{},
			"hasNextPage": false,
		})
	}))
	defer server.Close()

	client := NewClient("test_key", WithBaseURL(server.URL))
	if _, err := client.ListOutages(context.Background()); err != nil {
		t.Fatalf("ListOutages() error = %v", err)
	}

	if len(observed) == 0 {
		t.Fatal("expected at least one request, got none")
	}
	for i, q := range observed {
		if strings.Contains(q, "status=") {
			t.Errorf("request[%d] query %q must not contain status=", i, q)
		}
	}
}

// TestListOutages_WithStatusOngoing_AddsParam verifies that WithStatus("ongoing")
// produces a request URL containing status=ongoing.
func TestListOutages_WithStatusOngoing_AddsParam(t *testing.T) {
	var observed string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"outages":     []Outage{},
			"hasNextPage": false,
		})
	}))
	defer server.Close()

	client := NewClient("test_key", WithBaseURL(server.URL))
	if _, err := client.ListOutages(context.Background(), WithStatus("ongoing")); err != nil {
		t.Fatalf("ListOutages() error = %v", err)
	}

	if !strings.Contains(observed, "status=ongoing") {
		t.Errorf("expected status=ongoing in query %q", observed)
	}
}

// TestListOutages_WithStatusAll_AddsParam verifies status=all is sent verbatim.
func TestListOutages_WithStatusAll_AddsParam(t *testing.T) {
	var observed string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"outages":     []Outage{},
			"hasNextPage": false,
		})
	}))
	defer server.Close()

	client := NewClient("test_key", WithBaseURL(server.URL))
	if _, err := client.ListOutages(context.Background(), WithStatus("all")); err != nil {
		t.Fatalf("ListOutages() error = %v", err)
	}

	if !strings.Contains(observed, "status=all") {
		t.Errorf("expected status=all in query %q", observed)
	}
}

// TestListOutages_WithStatusResolved_AddsParam verifies status=resolved is sent.
func TestListOutages_WithStatusResolved_AddsParam(t *testing.T) {
	var observed string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"outages":     []Outage{},
			"hasNextPage": false,
		})
	}))
	defer server.Close()

	client := NewClient("test_key", WithBaseURL(server.URL))
	if _, err := client.ListOutages(context.Background(), WithStatus("resolved")); err != nil {
		t.Fatalf("ListOutages() error = %v", err)
	}

	if !strings.Contains(observed, "status=resolved") {
		t.Errorf("expected status=resolved in query %q", observed)
	}
}

// TestListOutages_WithInvalidStatus_ReturnsError verifies that an invalid status
// value yields an error before any HTTP request is issued.
func TestListOutages_WithInvalidStatus_ReturnsError(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"outages": []Outage{}})
	}))
	defer server.Close()

	client := NewClient("test_key", WithBaseURL(server.URL))
	_, err := client.ListOutages(context.Background(), WithStatus("bogus"))
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if called {
		t.Error("server must not be called when status is invalid")
	}
}

// TestListOutages_PaginationStillWorks_WithStatus verifies that pagination
// continues to combine pages correctly while the status filter is set.
func TestListOutages_PaginationStillWorks_WithStatus(t *testing.T) {
	t.Parallel()

	pages := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if got := r.URL.Query().Get("status"); got != "ongoing" {
			t.Errorf("expected status=ongoing on every page, got %q (page=%s)", got, page)
		}
		pages++
		w.WriteHeader(http.StatusOK)
		switch page {
		case "0":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"outages": []map[string]interface{}{
					{"uuid": "out_p0_1"},
					{"uuid": "out_p0_2"},
				},
				"hasNextPage": true,
			})
		case "1":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"outages": []map[string]interface{}{
					{"uuid": "out_p1_1"},
				},
				"hasNextPage": false,
			})
		default:
			t.Errorf("unexpected page request: %s", page)
		}
	}))
	defer server.Close()

	client := NewClient("test_key", WithBaseURL(server.URL))
	outages, err := client.ListOutages(context.Background(), WithStatus("ongoing"))
	if err != nil {
		t.Fatalf("ListOutages() error = %v", err)
	}
	if len(outages) != 3 {
		t.Errorf("got %d outages, want 3", len(outages))
	}
	if pages != 2 {
		t.Errorf("got %d page requests, want 2", pages)
	}
	wantOrder := []string{"out_p0_1", "out_p0_2", "out_p1_1"}
	for i, want := range wantOrder {
		if outages[i].UUID != want {
			t.Errorf("outage[%d].UUID = %q, want %q", i, outages[i].UUID, want)
		}
	}
}

// TestListOutages_BackwardCompat verifies that the no-opts call still hits
// /v2/outages with only page= in the query string (no other unexpected params).
// This mirrors what the previous implementation did.
func TestListOutages_BackwardCompat(t *testing.T) {
	var path, query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		query = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"outages":     []Outage{},
			"hasNextPage": false,
		})
	}))
	defer server.Close()

	client := NewClient("test_key", WithBaseURL(server.URL))
	if _, err := client.ListOutages(context.Background()); err != nil {
		t.Fatalf("ListOutages() error = %v", err)
	}

	if path != OutagesBasePath {
		t.Errorf("path = %q, want %q", path, OutagesBasePath)
	}
	// Expect exactly page=0 (no other params).
	if query != "page=0" {
		t.Errorf("query = %q, want %q", query, "page=0")
	}
}
