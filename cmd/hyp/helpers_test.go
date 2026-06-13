// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"net/http/httptest"
)

// execCmd runs the root command with the given args against the given test server.
// It returns stdout, stderr, and the execution error.
func execCmd(server *httptest.Server, args ...string) (string, string, error) {
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	root := newRootCmd("dev", "none", "unknown")
	root.SetOut(outBuf)
	root.SetErr(errBuf)

	fullArgs := []string{"--api-key", "test_key", "--base-url", server.URL}
	fullArgs = append(fullArgs, args...)
	root.SetArgs(fullArgs)

	err := root.Execute()
	return outBuf.String(), errBuf.String(), err
}

// execCmdNoServer runs the root command without a server (for commands that don't need one).
func execCmdNoServer(args ...string) (string, string, error) {
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	root := newRootCmd("dev", "none", "unknown")
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs(args)

	err := root.Execute()
	return outBuf.String(), errBuf.String(), err
}
