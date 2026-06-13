// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestTableOutput_Monitors(t *testing.T) {
	buf := &bytes.Buffer{}
	headers := []string{"UUID", "NAME", "URL"}
	rows := [][]string{
		{"mon_001", "My Monitor", "https://example.com"},
		{"mon_002", "Other", "https://other.com"},
	}
	writeTable(buf, headers, rows)
	out := buf.String()

	if !strings.Contains(out, "UUID") {
		t.Error("missing UUID header")
	}
	if !strings.Contains(out, "mon_001") {
		t.Error("missing mon_001 row")
	}
	if !strings.Contains(out, "My Monitor") {
		t.Error("missing My Monitor row")
	}
}

func TestJSONOutput_Monitors(t *testing.T) {
	buf := &bytes.Buffer{}
	data := []map[string]string{
		{"uuid": "mon_001", "name": "My Monitor"},
	}
	if err := writeJSON(buf, data); err != nil {
		t.Fatalf("writeJSON error: %v", err)
	}
	var result []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if len(result) != 1 || result[0]["uuid"] != "mon_001" {
		t.Errorf("unexpected JSON output: %v", result)
	}
}
