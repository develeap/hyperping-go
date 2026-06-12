// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerate_ModelsDotGenExists(t *testing.T) {
	_, err := os.Stat("models_gen.go")
	require.NoError(t, err,
		"models_gen.go must be committed; run: go generate ./...")
}

func TestGenerate_GeneratedFileHeaderIsPresent(t *testing.T) {
	data, err := os.ReadFile("models_gen.go")
	require.NoError(t, err)
	require.Contains(t, string(data), "DO NOT EDIT",
		"models_gen.go must carry the oapi-codegen DO NOT EDIT header")
}

func TestGenerate_FlexibleStringNotInGeneratedFile(t *testing.T) {
	data, err := os.ReadFile("models_gen.go")
	require.NoError(t, err)
	require.NotContains(t, string(data), "type FlexibleString",
		"FlexibleString must remain hand-coded; check -exclude-schemas in generate.go")
}

func TestGenerate_MonitorNotInGeneratedFile(t *testing.T) {
	data, err := os.ReadFile("models_gen.go")
	require.NoError(t, err)
	require.NotContains(t, string(data), "type Monitor struct",
		"Monitor must remain hand-coded; check -exclude-schemas in generate.go")
}
