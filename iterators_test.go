// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// =============================================================================
// IterMonitors
// =============================================================================

func TestIterMonitors_All(t *testing.T) {
	monitors := []Monitor{
		{UUID: "mon_001", Name: "Monitor 1"},
		{UUID: "mon_002", Name: "Monitor 2"},
		{UUID: "mon_003", Name: "Monitor 3"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != MonitorsBasePath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(monitors)
	}))
	defer server.Close()

	c := NewClient("test_key", WithBaseURL(server.URL), WithMaxRetries(0))

	var got []Monitor
	for m, err := range c.IterMonitors(context.Background()) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got = append(got, m)
	}

	if len(got) != 3 {
		t.Errorf("expected 3 monitors, got %d", len(got))
	}
	if got[0].UUID != "mon_001" {
		t.Errorf("expected mon_001, got %s", got[0].UUID)
	}
	if got[2].UUID != "mon_003" {
		t.Errorf("expected mon_003, got %s", got[2].UUID)
	}
}

func TestIterMonitors_EarlyTermination(t *testing.T) {
	requestCount := 0
	monitors := []Monitor{
		{UUID: "mon_001"}, {UUID: "mon_002"}, {UUID: "mon_003"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(monitors)
	}))
	defer server.Close()

	c := NewClient("test_key", WithBaseURL(server.URL), WithMaxRetries(0))

	var got []Monitor
	for m, err := range c.IterMonitors(context.Background()) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got = append(got, m)
		break
	}

	if len(got) != 1 {
		t.Errorf("expected 1 monitor (early termination), got %d", len(got))
	}
	if got[0].UUID != "mon_001" {
		t.Errorf("expected mon_001, got %s", got[0].UUID)
	}
	if requestCount != 1 {
		t.Errorf("expected exactly 1 server request, got %d", requestCount)
	}
}

func TestIterMonitors_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewClient("test_key", WithBaseURL(server.URL), WithMaxRetries(0))

	var items []Monitor
	var gotErr error
	var zeroOnError Monitor
	for m, err := range c.IterMonitors(context.Background()) {
		if err != nil {
			gotErr = err
			zeroOnError = m
		} else {
			items = append(items, m)
		}
	}

	if len(items) != 0 {
		t.Errorf("expected no items before error, got %d", len(items))
	}
	if gotErr == nil {
		t.Error("expected an error, got nil")
	}
	if zeroOnError.UUID != "" || zeroOnError.Name != "" {
		t.Errorf("expected zero Monitor on error, got UUID=%q Name=%q", zeroOnError.UUID, zeroOnError.Name)
	}
}

// =============================================================================
// IterIncidents
// =============================================================================

func TestIterIncidents_All(t *testing.T) {
	incidents := []Incident{
		{UUID: "inci_001", Type: "outage"},
		{UUID: "inci_002", Type: "incident"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != IncidentsBasePath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(incidents)
	}))
	defer server.Close()

	c := NewClient("test_key", WithBaseURL(server.URL), WithMaxRetries(0))

	var got []Incident
	for inc, err := range c.IterIncidents(context.Background()) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got = append(got, inc)
	}

	if len(got) != 2 {
		t.Errorf("expected 2 incidents, got %d", len(got))
	}
	if got[0].UUID != "inci_001" {
		t.Errorf("expected inci_001, got %s", got[0].UUID)
	}
	if got[1].UUID != "inci_002" {
		t.Errorf("expected inci_002, got %s", got[1].UUID)
	}
}

func TestIterIncidents_EarlyTermination(t *testing.T) {
	requestCount := 0
	incidents := []Incident{
		{UUID: "inci_001"}, {UUID: "inci_002"}, {UUID: "inci_003"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(incidents)
	}))
	defer server.Close()

	c := NewClient("test_key", WithBaseURL(server.URL), WithMaxRetries(0))

	var got []Incident
	for inc, err := range c.IterIncidents(context.Background()) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got = append(got, inc)
		break
	}

	if len(got) != 1 {
		t.Errorf("expected 1 incident (early termination), got %d", len(got))
	}
	if got[0].UUID != "inci_001" {
		t.Errorf("expected inci_001, got %s", got[0].UUID)
	}
	if requestCount != 1 {
		t.Errorf("expected exactly 1 server request, got %d", requestCount)
	}
}

func TestIterIncidents_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewClient("test_key", WithBaseURL(server.URL), WithMaxRetries(0))

	var items []Incident
	var gotErr error
	for inc, err := range c.IterIncidents(context.Background()) {
		if err != nil {
			gotErr = err
		} else {
			items = append(items, inc)
		}
	}

	if len(items) != 0 {
		t.Errorf("expected no items before error, got %d", len(items))
	}
	if gotErr == nil {
		t.Error("expected an error, got nil")
	}
}

// =============================================================================
// IterStatusPages
// =============================================================================

func TestIterStatusPages_SinglePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := StatusPagePaginatedResponse{
			StatusPages: []StatusPage{
				{UUID: "sp_001", Name: "Page 1"},
				{UUID: "sp_002", Name: "Page 2"},
			},
			HasNextPage: false,
			Total:       2,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("test_key", WithBaseURL(server.URL), WithMaxRetries(0))

	var got []StatusPage
	for sp, err := range c.IterStatusPages(context.Background(), nil) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got = append(got, sp)
	}

	if len(got) != 2 {
		t.Errorf("expected 2 status pages, got %d", len(got))
	}
	if got[0].UUID != "sp_001" {
		t.Errorf("expected sp_001, got %s", got[0].UUID)
	}
}

func TestIterStatusPages_MultiPage(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		pageParam := r.URL.Query().Get("page")

		var resp StatusPagePaginatedResponse
		switch pageParam {
		case "0", "":
			resp = StatusPagePaginatedResponse{
				StatusPages: []StatusPage{{UUID: "sp_001"}, {UUID: "sp_002"}},
				HasNextPage: true,
			}
		case "1":
			resp = StatusPagePaginatedResponse{
				StatusPages: []StatusPage{{UUID: "sp_003"}, {UUID: "sp_004"}},
				HasNextPage: false,
			}
		default:
			http.Error(w, "unexpected page", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("test_key", WithBaseURL(server.URL), WithMaxRetries(0))

	var got []StatusPage
	for sp, err := range c.IterStatusPages(context.Background(), nil) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got = append(got, sp)
	}

	if len(got) != 4 {
		t.Errorf("expected 4 status pages across 2 pages, got %d", len(got))
	}
	if requestCount != 2 {
		t.Errorf("expected exactly 2 HTTP requests, got %d", requestCount)
	}
	if got[0].UUID != "sp_001" {
		t.Errorf("expected sp_001 first, got %s", got[0].UUID)
	}
	if got[2].UUID != "sp_003" {
		t.Errorf("expected sp_003 third, got %s", got[2].UUID)
	}
}

func TestIterStatusPages_EarlyTermination(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		resp := StatusPagePaginatedResponse{
			StatusPages: []StatusPage{{UUID: "sp_001"}, {UUID: "sp_002"}, {UUID: "sp_003"}},
			HasNextPage: true,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("test_key", WithBaseURL(server.URL), WithMaxRetries(0))

	var got []StatusPage
	for sp, err := range c.IterStatusPages(context.Background(), nil) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got = append(got, sp)
		break
	}

	if len(got) != 1 {
		t.Errorf("expected 1 status page (early termination), got %d", len(got))
	}
	if requestCount != 1 {
		t.Errorf("expected exactly 1 HTTP request (second page not fetched), got %d", requestCount)
	}
}

func TestIterStatusPages_ErrorMidStream(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		pageParam := r.URL.Query().Get("page")

		switch pageParam {
		case "0", "":
			resp := StatusPagePaginatedResponse{
				StatusPages: []StatusPage{{UUID: "sp_001"}, {UUID: "sp_002"}},
				HasNextPage: true,
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	c := NewClient("test_key", WithBaseURL(server.URL), WithMaxRetries(0))

	var items []StatusPage
	var gotErr error
	for sp, err := range c.IterStatusPages(context.Background(), nil) {
		if err != nil {
			gotErr = err
		} else {
			items = append(items, sp)
		}
	}

	if len(items) != 2 {
		t.Errorf("expected 2 items from page 0, got %d", len(items))
	}
	if gotErr == nil {
		t.Error("expected an error from page 1, got nil")
	}
}

func TestIterStatusPages_WithSearch(t *testing.T) {
	var receivedSearch string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSearch = r.URL.Query().Get("search")
		resp := StatusPagePaginatedResponse{
			StatusPages: []StatusPage{{UUID: "sp_001"}},
			HasNextPage: false,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient("test_key", WithBaseURL(server.URL), WithMaxRetries(0))
	search := "production"

	for _, err := range c.IterStatusPages(context.Background(), &search) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if receivedSearch != "production" {
		t.Errorf("expected search param 'production', got %q", receivedSearch)
	}
}
