// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import "os"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd := newRootCmd(version, commit, date)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
