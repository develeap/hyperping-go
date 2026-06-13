// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCommand_Output(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := newRootCmd("1.2.3", "abc1234", "2026-01-01")
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"version"})
	err := cmd.Execute()
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "1.2.3")
	assert.Contains(t, out, "abc1234")
	assert.Contains(t, out, "2026-01-01")
}
