package shortn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// APIError is returned for every non-2xx answer from the API.
//
// The API answers errors as
//
//	{"code": "...", "error": "...", "hint": "...", "details": {"field": "..."}}
//
// where code is stable and safe to branch on. Some endpoints answer some
// errors with only "error" (Code is then empty): see the README for the list. Plan-limit errors (code "plan_limit_exceeded") also carry feature,
// limit, current, required_plan and upgrade_url at the top level: read them
// from Raw.
type APIError struct {
	// StatusCode is the HTTP status, such as 404.
	StatusCode int
	// Status is the HTTP status line, such as "404 Not Found".
	Status string
	// Code is the API's error code, such as "link_not_found". Empty when the
	// endpoint sent none.
	Code string
	// Message is the API's readable message (its "error" field).
	Message string
	// Hint suggests how to fix the request, when the API sends one.
	Hint string
	// Field names the request field at fault (details.field), when known.
	Field string
	// Details is the "details" object, when present.
	Details map[string]any
	// RequestID is the conversion endpoints' request_id (also the
	// Shortn-Request-Id header).
	RequestID string
	// RetryAfter is the Retry-After header of a 429 or 503 answer.
	RetryAfter string
	// Raw is the decoded JSON error body, when it was a JSON object.
	Raw map[string]any
	// Body is the raw response body.
	Body []byte
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("shortn: ")
	if e.StatusCode > 0 {
		fmt.Fprintf(&b, "%d ", e.StatusCode)
	}
	if e.Code != "" {
		b.WriteString(e.Code)
		b.WriteString(": ")
	}
	b.WriteString(e.Message)
	return b.String()
}

func newAPIError(resp *http.Response, body []byte) *APIError {
	e := &APIError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Body:       body,
		RequestID:  resp.Header.Get("Shortn-Request-Id"),
		RetryAfter: resp.Header.Get("Retry-After"),
	}
	var payload map[string]any
	if json.Unmarshal(body, &payload) == nil && payload != nil {
		e.Raw = payload
		e.Code = stringField(payload, "code")
		e.Message = stringField(payload, "error")
		if e.Message == "" {
			e.Message = stringField(payload, "message")
		}
		e.Hint = stringField(payload, "hint")
		if id := stringField(payload, "request_id"); id != "" {
			e.RequestID = id
		}
		if details, ok := payload["details"].(map[string]any); ok {
			e.Details = details
			e.Field = stringField(details, "field")
		}
		if e.Field == "" {
			e.Field = stringField(payload, "field")
		}
	}
	if e.Message == "" {
		text := strings.TrimSpace(string(body))
		if text == "" || len(text) > 200 || e.Raw != nil {
			text = http.StatusText(resp.StatusCode)
		}
		e.Message = text
	}
	return e
}

func stringField(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return strings.TrimSpace(s)
}
