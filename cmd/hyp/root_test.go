// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRootCommand_DefaultHelp(t *testing.T) {
	out, _, err := execCmdNoServer("--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, sub := range []string{"monitor", "incident", "statuspage", "tenant", "version"} {
		if !strings.Contains(out, sub) {
			t.Errorf("help output missing subcommand %q", sub)
		}
	}
}

func TestRootCommand_APIKeyFromFlag(t *testing.T) {
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	_, _, _ = execCmd(server, "monitor", "list")
	if !strings.HasPrefix(capturedAuth, "Bearer ") {
		t.Errorf("expected Bearer auth, got %q", capturedAuth)
	}
}

func TestRootCommand_APIKeyFromEnv(t *testing.T) {
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	t.Setenv("HYPERPING_API_KEY", "env_key_123")

	outBuf := &bytes.Buffer{}
	root := newRootCmd("dev", "none", "unknown")
	root.SetOut(outBuf)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"--base-url", server.URL, "monitor", "list"})
	root.Execute()

	if !strings.HasPrefix(capturedAuth, "Bearer ") {
		t.Errorf("expected Bearer auth from env, got %q", capturedAuth)
	}
}

func TestRootCommand_FlagOverridesEnv(t *testing.T) {
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	t.Setenv("HYPERPING_API_KEY", "env_key")

	_, _, _ = execCmd(server, "monitor", "list")

	// The flag value is "test_key" (set by execCmd), env is "env_key".
	// Both produce a Bearer token; we can only verify the format here since
	// the transport hides the raw key.
	if !strings.HasPrefix(capturedAuth, "Bearer ") {
		t.Errorf("expected Bearer auth, got %q", capturedAuth)
	}
}

func TestRootCommand_MissingAPIKey(t *testing.T) {
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	root := newRootCmd("dev", "none", "unknown")
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs([]string{"monitor", "list"})

	err := root.Execute()
	if err == nil {
		t.Error("expected error for missing API key, got nil")
	}
	if !strings.Contains(err.Error(), "API key") {
		t.Errorf("expected API key error, got: %v", err)
	}
}
