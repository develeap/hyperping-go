// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"strings"
	"testing"
)

func TestVersionCommand_Output(t *testing.T) {
	out, _, err := execCmdNoServer("version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "dev") {
		t.Errorf("expected version 'dev' in output, got: %q", out)
	}
	if !strings.Contains(out, "none") {
		t.Errorf("expected commit 'none' in output, got: %q", out)
	}
	if !strings.Contains(out, "unknown") {
		t.Errorf("expected date 'unknown' in output, got: %q", out)
	}
}
