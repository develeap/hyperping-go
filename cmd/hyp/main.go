// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	root := newRootCmd(version, commit, date)
	root.Execute()
}
