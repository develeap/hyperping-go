// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	hyperping "github.com/develeap/hyperping-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableOutput_Monitors(t *testing.T) {
	monitors := []hyperping.Monitor{
		{UUID: "mon_001", Name: "Alpha", URL: "https://alpha.example.com", Status: "up", Paused: false},
		{UUID: "mon_002", Name: "Beta", URL: "https://beta.example.com", Status: "down", Paused: true},
	}

	buf := &bytes.Buffer{}
	writeTable(buf,
		[]string{"UUID", "NAME", "URL", "STATUS", "PAUSED"},
		monitorsToRows(monitors),
	)

	out := buf.String()
	assert.Contains(t, out, "UUID")
	assert.Contains(t, out, "mon_001")
	assert.Contains(t, out, "Alpha")
	assert.Contains(t, out, "mon_002")
	assert.Contains(t, out, "Beta")
	assert.Contains(t, out, "up")
	assert.Contains(t, out, "down")
}

func TestJSONOutput_Monitors(t *testing.T) {
	monitors := []hyperping.Monitor{
		{UUID: "mon_001", Name: "Alpha", URL: "https://alpha.example.com"},
	}

	buf := &bytes.Buffer{}
	err := writeJSON(buf, monitors)
	require.NoError(t, err)

	var result []hyperping.Monitor
	err = json.NewDecoder(strings.NewReader(buf.String())).Decode(&result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "mon_001", result[0].UUID)
}
