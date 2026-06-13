// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"io"
	"net/http"
)

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}
