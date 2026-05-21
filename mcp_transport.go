// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultMCPURL    = "https://api.hyperping.io/v1/mcp"
	DefaultMCPTimeout = 30 * time.Second
	// Using the existing DefaultMaxRetries from client.go
	mcpProtocolVersion = "2025-03-26"
	mcpBaseDelay    = 1 * time.Second
	mcpMaxDelay   = 10 * time.Second
)

// ==================== Types ====================

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID     any            `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID     any            `json:"id"`
	Result any            `json:"result,omitempty"`
	Error  *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ErrSessionLost indicates the MCP server returned HTTP 404 for a request that
// carried an Mcp-Session-Id header. The transport clears its session state and
// performs one re-initialize + retry automatically; callers see this error only
// when the retried call also fails with a session-loss response.
var ErrSessionLost = errors.New("MCP session lost; re-initialize required")

// McpTransport is the low-level JSON-RPC 2.0 client for the Hyperping MCP server
type McpTransport struct {
	client      *http.Client
	url         string
	token       []byte
	maxRetries  int
	initialized atomic.Bool
	initMu      sync.Mutex
	reqID       atomic.Int64
	// sessionID holds the Mcp-Session-Id captured from the initialize response,
	// per MCP 2025-03-26 Streamable HTTP spec. nil means "no session"; the
	// transport then omits the header entirely (servers that do not issue a
	// session id continue to work unchanged). Written only under initMu;
	// read lock-free in callToolOnce.
	sessionID atomic.Pointer[string]
}

func (t *McpTransport) loadSessionID() string {
	if p := t.sessionID.Load(); p != nil {
		return *p
	}
	return ""
}

func (t *McpTransport) storeSessionID(s string) {
	if s == "" {
		t.sessionID.Store(nil)
		return
	}
	t.sessionID.Store(&s)
}

// TransportOption configures McpTransport
type TransportOption func(*McpTransport)

// WithMCPTimeout sets the request timeout
func WithMCPTimeout(d time.Duration) TransportOption {
	return func(t *McpTransport) {
		t.client.Timeout = d
	}
}

// WithMCPMaxRetries sets the maximum retry attempts
func WithMCPMaxRetries(n int) TransportOption {
	return func(t *McpTransport) {
		t.maxRetries = n
	}
}

// WithMCPHTTPClient sets a custom HTTP client
func WithMCPHTTPClient(h *http.Client) TransportOption {
	return func(t *McpTransport) {
		t.client = h
	}
}

// NewMcpTransport creates a new MCP transport.
// Returns an error if baseURL fails validation (must be HTTPS or localhost).
func NewMcpTransport(apiKey string, baseURL string, opts ...TransportOption) (*McpTransport, error) {
	resolvedURL := DefaultMCPURL
	if baseURL != "" {
		if err := validateBaseURL(baseURL); err != nil {
			return nil, fmt.Errorf("NewMcpTransport: %w", err)
		}
		resolvedURL = baseURL
	}

	baseTransport := &http.Transport{
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     20,
		IdleConnTimeout:     90 * time.Second,
	}

	t := &McpTransport{
		client: &http.Client{
			Timeout:   DefaultMCPTimeout,
			Transport: buildTransportChain([]byte(apiKey), baseTransport, resolvedURL),
		},
		url:        resolvedURL,
		token:      []byte(apiKey),
		maxRetries: DefaultMaxRetries,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t, nil
}

// ==================== Request ID ====================

func (t *McpTransport) nextID() int64 {
	return t.reqID.Add(1)
}

// ==================== Error Handling ====================

func (t *McpTransport) handleHTTPError(resp *http.Response) error {
	// Drain and discard the body so the connection can be reused.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20)) //nolint:errcheck

	statusCode := resp.StatusCode
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		retryAfter := 0
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			fmt.Sscanf(ra, "%d", &retryAfter) //nolint:errcheck
		}
		return &APIError{
			StatusCode: 429,
			Message:    "rate limit exceeded",
			RetryAfter: retryAfter,
		}
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return ErrValidation
	default:
		if statusCode >= 500 {
			return &APIError{
				StatusCode: statusCode,
				Message:    "server error",
			}
		}
		return &APIError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("HTTP %d", statusCode),
		}
	}
}

func (t *McpTransport) handleJSONRPCError(resp *JSONRPCResponse) error {
	if resp.Error == nil {
		return nil
	}

	switch resp.Error.Code {
	case -32700:
		return fmt.Errorf("%w: %s", ErrValidation, resp.Error.Message)
	case -32600:
		return fmt.Errorf("%w: %s", ErrValidation, resp.Error.Message)
	case -32601:
		return fmt.Errorf("%w: %s", ErrNotFound, resp.Error.Message)
	case -32602:
		return fmt.Errorf("%w: %s", ErrValidation, resp.Error.Message)
	case -32603:
		return fmt.Errorf("%w: %s", ErrServerError, resp.Error.Message)
	default:
		return fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
	}
}

// ==================== Initialize ====================

func (t *McpTransport) Initialize(ctx context.Context) (map[string]any, error) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      t.nextID(),
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": mcpProtocolVersion,
			"capabilities":  map[string]any{},
			"clientInfo": map[string]any{
				"name":    "hyperping-go",
				"version": "0.5.0",
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var lastErr error
	maxRetries := t.maxRetries

	for attempt := 0; attempt <= maxRetries; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+string(t.token))

		resp, err := t.client.Do(httpReq)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close() //nolint:errcheck

		if resp.StatusCode != http.StatusOK {
			err := t.handleHTTPError(resp)
			if isServerError(err) && attempt < maxRetries {
				lastErr = err
				select {
				case <-time.After(mcpBackoff(attempt)):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				continue
			}
			return nil, err
		}

		var rpcResp JSONRPCResponse
		if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
			return nil, err
		}

		if rpcResp.Error != nil {
			err := t.handleJSONRPCError(&rpcResp)
			if isServerError(err) && attempt < maxRetries {
				lastErr = err
				select {
				case <-time.After(mcpBackoff(attempt)):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				continue
			}
			return nil, err
		}

		// Capture Mcp-Session-Id per MCP 2025-03-26 Streamable HTTP spec. If
		// the server did not issue a session id, this stores "" (treated as
		// "no session") and the transport stays backward compatible with
		// servers that do not enforce sessions.
		t.storeSessionID(resp.Header.Get("Mcp-Session-Id"))
		t.initialized.Store(true)

		if result, ok := rpcResp.Result.(map[string]any); ok {
			return result, nil
		}
		return nil, nil
	}

	return nil, lastErr
}

// ==================== CallTool ====================

func (t *McpTransport) CallTool(ctx context.Context, toolName string, args map[string]any) (any, error) {
	var lastErr error
	var sessionRecoveryAttempted bool
	maxRetries := t.maxRetries

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Init guard inside the loop so session-loss recovery (which clears
		// t.initialized) re-fires Initialize on the next iteration. The
		// fast-path atomic load keeps the common case lock-free.
		if !t.initialized.Load() {
			t.initMu.Lock()
			if !t.initialized.Load() {
				_, initErr := t.Initialize(ctx)
				t.initMu.Unlock()
				if initErr != nil {
					return nil, initErr
				}
			} else {
				t.initMu.Unlock()
			}
		}

		result, err := t.callToolOnce(ctx, toolName, args)
		if err == nil {
			return result, nil
		}
		lastErr = err

		// Session-loss recovery: server dropped our session. callToolOnce has
		// already cleared t.initialized and t.sessionID, so the next iteration
		// re-initializes under initMu (concurrent callers serialize through
		// the same mutex, so the server sees one re-initialize, not N). At
		// most one recovery attempt per CallTool to avoid infinite loops.
		if errors.Is(err, ErrSessionLost) && !sessionRecoveryAttempted {
			sessionRecoveryAttempted = true
			continue
		}

		// Retryable server errors (500, 502, 503, 504)
		if isServerError(err) && attempt < maxRetries {
			select {
			case <-time.After(mcpBackoff(attempt)):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}

		return nil, err
	}

	return nil, lastErr
}

func (t *McpTransport) callToolOnce(ctx context.Context, toolName string, args map[string]any) (any, error) {
	params := map[string]any{
		"name": toolName,
	}
	if args != nil {
		params["arguments"] = args
	}

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      t.nextID(),
		Method:  "tools/call",
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+string(t.token))
	// Snapshot the session id for the lifetime of this attempt so the 404
	// detection below knows whether the request carried one.
	sid := t.loadSessionID()
	if sid != "" {
		httpReq.Header.Set("Mcp-Session-Id", sid)
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	// Session-loss handling on HTTP 404.
	//
	// Case A — request DID carry a session id: the server dropped our
	// session. Clear local state under initMu, paired with a sid-match
	// check so a stale 404 (from a request sent before another goroutine
	// already recovered) does not clobber a freshly re-initialized
	// session and cause a re-init cascade under heavy concurrency.
	//
	// Case B — request did NOT carry a session id, but the live session
	// state has changed since our snapshot: a concurrent goroutine cleared
	// or replaced the session between our init-guard fast-load and the
	// sessionID snapshot at the top of this function. The 404 reflects
	// that race, not a missing-tool error. Signal session loss so the
	// caller's recovery path can re-initialize and retry. Checking both
	// "session id now non-empty" and "transport no longer initialized"
	// catches both halves of the clear+re-init window.
	//
	// Case C — request had no session id AND the live state is unchanged
	// (still no session, still initialized): the server simply does not
	// enforce sessions (backward-compat path) and the 404 is a genuine
	// missing-tool / bad-URL response. Fall through to the existing
	// error path.
	if resp.StatusCode == http.StatusNotFound {
		if sid != "" {
			t.initMu.Lock()
			if t.loadSessionID() == sid {
				t.initialized.Store(false)
				t.storeSessionID("")
			}
			t.initMu.Unlock()
			return nil, ErrSessionLost
		}
		if t.loadSessionID() != "" || !t.initialized.Load() {
			return nil, ErrSessionLost
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, t.handleHTTPError(resp)
	}

	var rpcResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}

	if rpcResp.Error != nil {
		return nil, t.handleJSONRPCError(&rpcResp)
	}

	if rpcResp.Result == nil {
		return nil, nil
	}

	resultMap, ok := rpcResp.Result.(map[string]any)
	if !ok {
		return rpcResp.Result, nil
	}

	rawContent, exists := resultMap["content"]
	if !exists {
		return nil, nil
	}
	content, ok := rawContent.([]any)
	if !ok {
		return nil, fmt.Errorf("MCP response: expected \"content\" to be []any, got %T", rawContent)
	}
	if len(content) == 0 {
		return nil, nil
	}

	rawFirst := content[0]
	firstContent, ok := rawFirst.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("MCP response: expected content[0] to be map[string]any, got %T", rawFirst)
	}

	rawText, exists := firstContent["text"]
	if !exists {
		return nil, nil
	}
	text, ok := rawText.(string)
	if !ok {
		return nil, fmt.Errorf("MCP response: expected content[0][\"text\"] to be string, got %T", rawText)
	}
	if text == "" {
		return nil, nil
	}

	var parsed any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse MCP tool response: %w", err)
	}

	return parsed, nil
}

// isServerError checks if an error is a server error that should be retried
func isServerError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrServerError) {
		return true
	}
	if ae, ok := err.(*APIError); ok {
		return ae.StatusCode >= 500 && ae.StatusCode < 600
	}
	errStr := err.Error()
	return strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "server error") ||
		strings.Contains(errStr, "Internal Server Error")
}

func mcpBackoff(attempt int) time.Duration {
	delay := mcpBaseDelay * time.Duration(1<<attempt)
	if delay > mcpMaxDelay {
		return mcpMaxDelay
	}
	return delay
}
