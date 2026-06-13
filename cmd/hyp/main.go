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
	root := newRootCmd(version, commit, date)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
