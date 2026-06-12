// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"expvar"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
)

// debugClientSeq generates unique expvar names when multiple clients call
// WithDebugStats in the same process.
var debugClientSeq atomic.Uint64

// DebugStats holds expvar counters for a single Client instance. Obtain one
// by calling WithDebugStats on a Client; pass it to NewMcpTransport via
// WithTransportStats to expose MCP session-refresh counts on the same endpoint.
type DebugStats struct {
	inFlight     *expvar.Int
	totalReqs    *expvar.Int
	errors       *expvar.Map
	cbState      *expvar.String
	retries      *expvar.Map
	mcpRefreshes *expvar.Int

	mapName    string
	listenAddr string
	server     *http.Server
}

func newDebugStats() *DebugStats {
	id := debugClientSeq.Add(1) - 1
	name := fmt.Sprintf("hyperping.client.%d", id)
	m := expvar.NewMap(name)

	s := &DebugStats{
		mapName:      name,
		inFlight:     new(expvar.Int),
		totalReqs:    new(expvar.Int),
		errors:       new(expvar.Map),
		cbState:      new(expvar.String),
		retries:      new(expvar.Map),
		mcpRefreshes: new(expvar.Int),
	}
	s.cbState.Set("closed")

	m.Set("in_flight", s.inFlight)
	m.Set("total_requests", s.totalReqs)
	m.Set("errors", s.errors)
	m.Set("circuit_breaker_state", s.cbState)
	m.Set("retries", s.retries)
	m.Set("mcp_session_refreshes", s.mcpRefreshes)

	return s
}

// Addr returns the TCP address the debug server is actually listening on.
// Useful when addr was ":0" (random port assignment).
func (s *DebugStats) Addr() string {
	return s.listenAddr
}

// MapName returns the expvar map key under which this client's stats are
// registered (e.g. "hyperping.client.0"). Use it to look up this client's
// entry in the /debug/vars output when multiple clients are active.
func (s *DebugStats) MapName() string {
	return s.mapName
}

func (s *DebugStats) recordStart() {
	if s == nil {
		return
	}
	s.inFlight.Add(1)
	s.totalReqs.Add(1)
}

func (s *DebugStats) recordEnd() {
	if s == nil {
		return
	}
	s.inFlight.Add(-1)
}

func (s *DebugStats) recordRetry(attempt int) {
	if s == nil {
		return
	}
	s.retries.Add(fmt.Sprintf("attempt_%d", attempt), 1)
}

func (s *DebugStats) incError(errType string) {
	if s == nil {
		return
	}
	s.errors.Add(errType, 1)
}

func (s *DebugStats) setCBState(state string) {
	if s == nil {
		return
	}
	s.cbState.Set(state)
}

func (s *DebugStats) incMCPRefreshes() {
	if s == nil {
		return
	}
	s.mcpRefreshes.Add(1)
}

// WithDebugStats starts a dedicated HTTP server on addr that serves /debug/vars
// (expvar format) with in-flight requests, total requests, error counts by type,
// circuit-breaker state, retry distribution, and MCP session-id refresh count.
// Pass an empty string to disable. After construction, call (*Client).Stats() to
// obtain the *DebugStats and pass it to NewMcpTransport via WithTransportStats to
// include MCP session-refresh tracking on the same endpoint.
func WithDebugStats(addr string) Option {
	return func(c *Client) {
		if addr == "" {
			return
		}
		s := newDebugStats()

		ln, err := net.Listen("tcp", addr)
		if err != nil {
			c.setupErr = fmt.Errorf("WithDebugStats: listen %s: %w", addr, err)
			return
		}
		s.listenAddr = ln.Addr().String()

		mux := http.NewServeMux()
		mux.Handle("/debug/vars", expvar.Handler())

		srv := &http.Server{
			Addr:    addr,
			Handler: mux,
		}
		s.server = srv
		go srv.Serve(ln) //nolint:errcheck

		c.debugStats = s
	}
}

// Stats returns the DebugStats wired to this client, or nil if WithDebugStats
// was not called. Pass the result to NewMcpTransport via WithTransportStats to
// surface MCP session-refresh counts on the same /debug/vars endpoint.
func (c *Client) Stats() *DebugStats {
	return c.debugStats
}
