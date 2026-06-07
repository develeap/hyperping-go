// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MPL-2.0

package hyperping

import (
	"crypto/tls"
	"net/http"
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

// These tests guard against regression of the v0.6.2 HTTP/2 ALPN bug.
//
// Background: enforceTLS calls httpTransport.Clone() to avoid mutating a
// caller-owned *http.Transport. Clone() triggers Go's stdlib lazy
// initialization on the SOURCE transport, which populates the source's
// TLSClientConfig.NextProtos with ["h2", "http/1.1"] and the source's
// TLSNextProto["h2"] with an h2 upgrader. The clone deep-copies
// TLSClientConfig (so NextProtos carries over) but does NOT copy
// TLSNextProto when the source's TLSNextProto was originally nil
// (the documented tlsNextProtoWasNil guard in net/http.Transport.Clone).
//
// On the clone's own lazy init, stdlib's protocols() returns HTTP/1-only
// because the clone now has a non-nil TLSClientConfig (set by enforceTLS)
// with ForceAttemptHTTP2 unset; that path skips h2 setup entirely. The
// result: the clone advertises h2 in ALPN with no h2 handler registered,
// causing "malformed HTTP response \x00\x00..." errors against any
// h2-capable server.
//
// The invariant these tests guard: any *http.Transport that exits
// enforceTLS (or that the SDK installs on its http.Client) MUST NOT
// advertise h2 in TLSClientConfig.NextProtos without also having a
// matching handler in TLSNextProto["h2"].
//
// Reproduction requires the production transport shape: source with
// nil TLSClientConfig, nil Dial/DialContext/DialTLS/DialTLSContext,
// nil Protocols, ForceAttemptHTTP2 unset. This is what the SDK's own
// default transport looks like; it is also why a pre-configured
// httptest server (which forces a custom DialContext) cannot be used
// to trigger the bug end-to-end. These tests therefore assert the
// invariant directly on the transport state after a forced lazy init
// to a dead local address.

// forceLazyInit triggers Go stdlib's nextProtoOnce.Do on a transport by
// attempting a RoundTrip to a dead local address. The dial fails, but the
// h2 auto-init logic runs before the dial.
func forceLazyInit(t *testing.T, rt http.RoundTripper) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, "https://127.0.0.1:1/", nil)
	_, _ = rt.RoundTrip(req)
}

// assertH2Invariant fails the test if the transport advertises h2 in ALPN
// (TLSClientConfig.NextProtos contains "h2") without a matching handler
// in TLSNextProto["h2"].
func assertH2Invariant(t *testing.T, label string, tr *http.Transport) {
	t.Helper()
	advertisesH2 := false
	if tr.TLSClientConfig != nil {
		for _, p := range tr.TLSClientConfig.NextProtos {
			if p == "h2" {
				advertisesH2 = true
				break
			}
		}
	}
	if !advertisesH2 {
		return
	}
	require.NotNilf(t, tr.TLSNextProto["h2"],
		"%s: h2 advertised in ALPN but no handler registered in TLSNextProto; "+
			"see https://pkg.go.dev/net/http#Transport.ForceAttemptHTTP2",
		label)
}

// TestEnforceTLS_HTTP2InvariantIfAdvertisedThenHandlerExists is the
// deterministic regression test for the bug. It calls enforceTLS directly
// with a default *http.Transport{} (matching the SDK's production transport
// shape) and asserts the h2 ALPN invariant after forced lazy init.
func TestEnforceTLS_HTTP2InvariantIfAdvertisedThenHandlerExists(t *testing.T) {
	src := &http.Transport{}
	// NON-localhost URL to bypass the localhost early-return path. The
	// localhost path returns the transport unchanged, which is why the
	// SDK's own httptest-based tests (127.0.0.1) never caught the bug.
	result := enforceTLS(src, "https://api.example.com")
	cloned, ok := result.(*http.Transport)
	require.True(t, ok, "expected *http.Transport from enforceTLS on https non-localhost URL")

	forceLazyInit(t, cloned)
	assertH2Invariant(t, "enforceTLS direct", cloned)
}

// extractInnerHTTPTransport walks an http.RoundTripper chain (e.g. the
// authTransport wrapper installed by NewClient) using reflection+unsafe to
// reach unexported fields, returning the inner *http.Transport. Returns nil
// if not found within a small depth bound.
func extractInnerHTTPTransport(rt http.RoundTripper) *http.Transport {
	cur := rt
	for i := 0; i < 8 && cur != nil; i++ {
		if t, ok := cur.(*http.Transport); ok {
			return t
		}
		v := reflect.ValueOf(cur)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return nil
		}
		found := false
		for j := 0; j < v.NumField(); j++ {
			f := v.Field(j)
			if !f.CanInterface() {
				f = reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
			}
			val := f.Interface()
			if next, ok := val.(http.RoundTripper); ok && next != nil {
				cur = next
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return nil
}

// clientHTTPClient extracts the unexported httpClient field from a *Client.
func clientHTTPClient(c *Client) *http.Client {
	cv := reflect.ValueOf(c).Elem()
	f := cv.FieldByName("httpClient")
	f = reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	return f.Interface().(*http.Client)
}

// TestNewClient_HTTP2Invariant_RESTPath asserts the h2 ALPN invariant on the
// inner *http.Transport that NewClient installs, when configured with a
// non-localhost HTTPS base URL. This exercises the same buildTransportChain
// -> enforceTLS path as production calls to api.hyperping.io.
func TestNewClient_HTTP2Invariant_RESTPath(t *testing.T) {
	c := NewClient("test-api-key-1234567890",
		WithBaseURL("https://api.example.com"),
	)
	hc := clientHTTPClient(c)
	inner := extractInnerHTTPTransport(hc.Transport)
	require.NotNil(t, inner, "expected to extract *http.Transport from client chain")

	forceLazyInit(t, inner)
	assertH2Invariant(t, "NewClient (REST)", inner)
}

// TestNewMcpTransport_HTTP2Invariant_MCPPath asserts the h2 ALPN invariant on
// the inner *http.Transport that NewMcpTransport installs. The MCP path also
// runs through buildTransportChain -> enforceTLS, so the single enforceTLS
// fix must cover it.
func TestNewMcpTransport_HTTP2Invariant_MCPPath(t *testing.T) {
	tr, err := NewMcpTransport("test-api-key-1234567890", "https://mcp.example.com")
	require.NoError(t, err)

	// McpTransport.client is an *http.Client; reach into it via reflection.
	tv := reflect.ValueOf(tr).Elem()
	cf := tv.FieldByName("client")
	cf = reflect.NewAt(cf.Type(), unsafe.Pointer(cf.UnsafeAddr())).Elem()
	mc := cf.Interface().(*http.Client)

	inner := extractInnerHTTPTransport(mc.Transport)
	require.NotNil(t, inner, "expected to extract *http.Transport from MCP client chain")

	forceLazyInit(t, inner)
	assertH2Invariant(t, "NewMcpTransport (MCP)", inner)

	// Belt-and-suspenders: the TLS config installed by enforceTLS must still
	// carry the SDK's TLS hardening (MinVersion >= TLS 1.2, AEAD ciphers).
	// This is the security floor that ForceAttemptHTTP2 must not weaken.
	require.NotNil(t, inner.TLSClientConfig)
	require.GreaterOrEqual(t, int(inner.TLSClientConfig.MinVersion), int(tls.VersionTLS12),
		"enforceTLS must preserve MinVersion TLS 1.2 floor")
	require.NotEmpty(t, inner.TLSClientConfig.CipherSuites,
		"enforceTLS must preserve cipher suite restrictions")
}
