// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const (
	DefaultMCPURL    = "https://api.hyperping.io/v1/mcp"
	DefaultMCPTimeout = 30 * time.Second
	// Using the existing DefaultMaxRetries from client.go
	mcpProtocolVersion = "2025-03-26"
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

// McpTransport is the low-level JSON-RPC 2.0 client for the Hyperping MCP server
type McpTransport struct {
	client      *http.Client
	url         string
	token       []byte
	maxRetries  int
	initialized atomic.Bool
	reqID       atomic.Int64
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

// NewMcpTransport creates a new MCP transport
func NewMcpTransport(apiKey string, baseURL string, opts ...TransportOption) *McpTransport {
	t := &McpTransport{
		client: &http.Client{
			Timeout: DefaultMCPTimeout,
		},
		url:        DefaultMCPURL,
		token:      []byte(apiKey),
		maxRetries: DefaultMaxRetries,
	}

	for _, opt := range opts {
		opt(t)
	}

	if baseURL != "" {
		t.url = baseURL
	}

	return t
}

// ==================== Request ID ====================

func (t *McpTransport) nextID() int64 {
	return t.reqID.Add(1)
}

// ==================== Error Handling ====================

func (t *McpTransport) handleHTTPError(resp *http.Response) error {
	statusCode := resp.StatusCode
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		retryAfter := 0
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			fmt.Sscanf(ra, "%d", &retryAfter)
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
				"version": "0.3.0",
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
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			err := t.handleHTTPError(resp)
			if isServerError(err) && attempt < maxRetries {
				lastErr = err
				time.Sleep(time.Duration(1<<attempt) * time.Second)
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
				time.Sleep(time.Duration(1<<attempt) * time.Second)
				continue
			}
			return nil, err
		}

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
	if !t.initialized.Load() {
		if _, err := t.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	var lastErr error
	maxRetries := t.maxRetries

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := t.callToolOnce(ctx, toolName, args)
		if err == nil {
			return result, nil
		}

		// Check if error is retryable (server errors 500, 502, 503, 504)
		if isServerError(err) && attempt < maxRetries {
			lastErr = err
			sleepDur := time.Duration(1<<attempt+rand.IntN(2)) * time.Second
			if sleepDur > 10*time.Second {
				sleepDur = 10 * time.Second
			}
			time.Sleep(sleepDur)
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

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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

	content, ok := resultMap["content"].([]any)
	if !ok || len(content) == 0 {
		return nil, nil
	}

	firstContent, ok := content[0].(map[string]any)
	if !ok {
		return nil, nil
	}

	text, ok := firstContent["text"].(string)
	if !ok || text == "" {
		return nil, nil
	}

	var parsed any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse MCP tool response: %w", err)
	}

	return parsed, nil
}

func randIntn(n int) int {
	return rand.IntN(n)
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
