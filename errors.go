// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/sony/gobreaker"
)

// Common errors returned by the client.
var (
	// ErrNotFound is returned when a resource is not found (404).
	ErrNotFound = errors.New("resource not found")

	// ErrUnauthorized is returned when authentication fails (401/403).
	ErrUnauthorized = errors.New("unauthorized: invalid or missing API key")

	// ErrRateLimited is returned when rate limit is exceeded (429).
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrValidation is returned when request validation fails (400/422).
	ErrValidation = errors.New("validation error")

	// ErrServerError is returned for server errors (5xx).
	ErrServerError = errors.New("server error")
)

// APIError represents an error returned by the Hyperping API.
type APIError struct {
	StatusCode int
	Message    string
	Details    []ValidationDetail
	RetryAfter int // seconds to wait before retrying (for rate limits)
}

// ValidationDetail represents a field-level validation error.
type ValidationDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface.
// The message is sanitized to remove sensitive information like API keys.
func (e *APIError) Error() string {
	sanitized := sanitizeMessage(e.Message)

	// Enhanced rate limit error messages
	if e.StatusCode == 429 && e.RetryAfter > 0 {
		return fmt.Sprintf("API error (status %d): %s - retry after %d seconds", e.StatusCode, sanitized, e.RetryAfter)
	}

	if len(e.Details) > 0 {
		return fmt.Sprintf("API error (status %d): %s - %d validation errors", e.StatusCode, sanitized, len(e.Details))
	}
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, sanitized)
}

// Is checks if the error matches a target error.
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrNotFound:
		return e.StatusCode == 404
	case ErrUnauthorized:
		return e.StatusCode == 401 || e.StatusCode == 403
	case ErrRateLimited:
		return e.StatusCode == 429
	case ErrValidation:
		return e.StatusCode == 400 || e.StatusCode == 422
	case ErrServerError:
		return e.StatusCode >= 500
	}
	return false
}

// Unwrap returns the underlying error based on status code.
func (e *APIError) Unwrap() error {
	switch {
	case e.StatusCode == 404:
		return ErrNotFound
	case e.StatusCode == 401 || e.StatusCode == 403:
		return ErrUnauthorized
	case e.StatusCode == 429:
		return ErrRateLimited
	case e.StatusCode == 400 || e.StatusCode == 422:
		return ErrValidation
	case e.StatusCode >= 500:
		return ErrServerError
	}
	return nil
}

// NewAPIError creates a new APIError from an HTTP response.
func NewAPIError(statusCode int, message string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// NewValidationError creates a new APIError with validation details.
func NewValidationError(statusCode int, message string, details []ValidationDetail) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    message,
		Details:    details,
	}
}

// NewRateLimitError creates a new APIError for rate limiting with retry-after.
func NewRateLimitError(retryAfter int) *APIError {
	return &APIError{
		StatusCode: 429,
		Message:    "rate limit exceeded",
		RetryAfter: retryAfter,
	}
}

// Compile regexes once for performance.
// bearerPattern matches any Bearer token of 8+ non-whitespace chars (VULN-019).
// authHeaderPattern matches any Authorization header, regardless of scheme
// (Bearer, Basic, Digest, AWS SigV4, etc.). The terminator is line-bounded
// rather than comma-bounded because multi-field schemes such as RFC 7616
// Digest and AWS4-HMAC-SHA256 embed their credential (response="<hash>",
// Signature=<hex>) after one or more commas. RE2 has no lookahead, so the
// only safe primitive is "redact to end of line"; this over-redacts when
// callers flatten multiple headers onto one comma-separated line, which is
// an acceptable tradeoff (security beats prettier debug output).
//
// proxyAuthHeaderPattern, cookieHeaderPattern and apiKeyHeaderPattern cover
// the other common credential-bearing headers that downstream callers
// (e.g. terraform-provider-hyperping) let users set on monitor probes.
// This set is not exhaustive; any header not listed here should be treated
// as potentially sensitive until proven otherwise.
var (
	// bearerPattern matches any "Bearer <token>" sequence where the token is
	// 6 or more non-whitespace characters. The original variant required a
	// digit/underscore/dash anywhere in the token OR a 32+ char run, which
	// left letters-only tokens of 8-31 chars (real-world: opaque session
	// ids, first segment of an unsigned JWT) unredacted. 6 is the floor for
	// any plausible production credential; shorter sequences are treated as
	// placeholders / examples and pass through.
	// bearerPattern captures the post-Bearer token in group 1 so a
	// ReplaceAllStringFunc callback can decide whether to redact. Without
	// the capture group, the prior `Bearer\s+\S{6,}` matched
	// RFC 6750 WWW-Authenticate challenge parameters (`Bearer realm="api"`,
	// `Bearer error="invalid_token"`) and clobbered legitimate diagnostic
	// output. RE2 has no lookahead, so the discriminator runs in the
	// callback (see sanitizeMessage).
	bearerPattern = regexp.MustCompile(`Bearer\s+(\S{6,})`)
	// bearerChallengeParamPattern enumerates the RFC 6750 section 3
	// challenge-parameter names. If the captured group starts with one of
	// these followed by '=', the match is a challenge parameter, not a
	// credential, and is left intact.
	bearerChallengeParamPattern = regexp.MustCompile(`^(?:realm|scope|error|error_description|error_uri)=`)
	urlCredPattern              = regexp.MustCompile(`://[^:]+:[^@]+@`)
	proxyAuthHeaderPattern      = regexp.MustCompile(`(?i)Proxy-Authorization:\s+[^\r\n]+`)
	cookieHeaderPattern         = regexp.MustCompile(`(?i)(?:Set-)?Cookie:\s+[^\r\n]+`)
	apiKeyHeaderPattern         = regexp.MustCompile(`(?i)X-(?:Api-Key|Auth-Token|Access-Token):\s+[^\r\n]+`)
	authHeaderPattern           = regexp.MustCompile(`(?i)Authorization:\s+[^\r\n]+`)
)

// sanitizeMessage removes sensitive information from error messages.
// This prevents API keys, tokens, and credentials from being exposed in logs or error output.
func sanitizeMessage(msg string) string {
	// Replace Hyperping API keys (sk_alphanumeric) with redacted placeholder
	msg = APIKeyPattern.ReplaceAllString(msg, APIKeyPrefix+"***REDACTED***")

	// Replace credential-bearing headers. Order matters: more specific names
	// run first so that Proxy-Authorization and Set-Cookie are not partially
	// clobbered by the broader Authorization / Cookie patterns.
	msg = proxyAuthHeaderPattern.ReplaceAllString(msg, "Proxy-Authorization: ***REDACTED***")
	msg = cookieHeaderPattern.ReplaceAllStringFunc(msg, func(match string) string {
		// Preserve the original header name (Cookie vs Set-Cookie) so the
		// redacted output still tells the reader which header leaked.
		if len(match) >= 11 && (match[0] == 'S' || match[0] == 's') {
			return "Set-Cookie: ***REDACTED***"
		}
		return "Cookie: ***REDACTED***"
	})
	msg = apiKeyHeaderPattern.ReplaceAllStringFunc(msg, func(match string) string {
		// Find the colon to recover the original header name.
		for i := 0; i < len(match); i++ {
			if match[i] == ':' {
				return match[:i] + ": ***REDACTED***"
			}
		}
		return match
	})
	msg = authHeaderPattern.ReplaceAllString(msg, "Authorization: ***REDACTED***")

	// Replace Bearer tokens, but preserve RFC 6750 challenge parameters
	// (realm/scope/error/...). The capture group holds the post-Bearer
	// run of non-whitespace; if it starts with a known challenge prefix
	// the match is returned unchanged. Short opaque tokens that happen to
	// collide with a challenge name (e.g. literal "Cookie", "value") are
	// still redacted because they do not match the param= prefix.
	msg = bearerPattern.ReplaceAllStringFunc(msg, func(match string) string {
		sub := bearerPattern.FindStringSubmatch(match)
		if len(sub) >= 2 && bearerChallengeParamPattern.MatchString(sub[1]) {
			return match
		}
		return "Bearer ***REDACTED***"
	})

	// Replace credentials in URLs (https://user:pass@domain.com)
	msg = urlCredPattern.ReplaceAllString(msg, "://***REDACTED***@")

	return msg
}

// IsNotFound checks if an error is a not found error.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsUnauthorized checks if an error is an unauthorized error.
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsRateLimited checks if an error is a rate limit error.
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited)
}

// IsValidation checks if an error is a validation error.
func IsValidation(err error) bool {
	return errors.Is(err, ErrValidation)
}

// IsServerError checks if an error is a server error.
func IsServerError(err error) bool {
	return errors.Is(err, ErrServerError)
}

// IsCircuitBreakerOpen checks if an error is due to the circuit breaker being open.
// This occurs when too many recent API calls failed, triggering the circuit breaker
// to stop all requests for a recovery period (30 seconds by default).
func IsCircuitBreakerOpen(err error) bool {
	return errors.Is(err, gobreaker.ErrOpenState)
}

// HyperpingRateLimitError is returned when the API responds with 429 Too Many Requests.
// RequestID is the X-Request-Id from the response header.
// RetryAfter is the number of seconds the caller should wait before retrying.
type HyperpingRateLimitError struct {
	RequestID  string
	RetryAfter int
	apiError   *APIError
}

func (e *HyperpingRateLimitError) Error() string { return e.apiError.Error() }
func (e *HyperpingRateLimitError) Unwrap() error  { return e.apiError }

// HyperpingAuthError is returned when the API responds with 401 Unauthorized or 403 Forbidden.
// RequestID is the X-Request-Id from the response header.
type HyperpingAuthError struct {
	RequestID string
	apiError  *APIError
}

func (e *HyperpingAuthError) Error() string { return e.apiError.Error() }
func (e *HyperpingAuthError) Unwrap() error  { return e.apiError }

// HyperpingNotFoundError is returned when the API responds with 404 Not Found.
// RequestID is the X-Request-Id from the response header.
type HyperpingNotFoundError struct {
	RequestID string
	apiError  *APIError
}

func (e *HyperpingNotFoundError) Error() string { return e.apiError.Error() }
func (e *HyperpingNotFoundError) Unwrap() error  { return e.apiError }

// HyperpingValidationError is returned when the API responds with 400 Bad Request
// or 422 Unprocessable Entity. RequestID is the X-Request-Id from the response header.
// Details contains field-level validation errors when the API provides them.
type HyperpingValidationError struct {
	RequestID string
	Details   []ValidationDetail
	apiError  *APIError
}

func (e *HyperpingValidationError) Error() string { return e.apiError.Error() }
func (e *HyperpingValidationError) Unwrap() error  { return e.apiError }
