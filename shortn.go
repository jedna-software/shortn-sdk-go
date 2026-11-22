// Package shortn is a Go client for the Shortn Link public API
// (https://app.shortn.link/api/v1).
//
// Create a client with an API key and call the resource groups hanging off
// it: client.Links, client.Analytics, client.Workspaces, client.Tags,
// client.UTMTemplates, client.Destinations, client.Teams, client.Domains,
// client.Reports, client.Conversions and client.Public. The client.Public
// helpers also work without an API key.
//
// Every response type keeps the whole decoded JSON object in its Raw field,
// so fields the API adds later are readable before this SDK models them.
package shortn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Version is the SDK version, sent in the User-Agent header.
const Version = "0.1.0"

const (
	// DefaultBaseURL is the Shortn Link origin the client talks to by default.
	// The client appends /api/v1 to it.
	DefaultBaseURL = "https://app.shortn.link"
	defaultTimeout = 30 * time.Second
	// defaultMaxRetryAfter matches the other Shortn Link SDKs.
	defaultMaxRetryAfter = 60 * time.Second
	userAgent            = "shortn-sdk-go/" + Version
)

// ErrAPIKeyRequired is returned, without sending a request, when a method
// that needs an API key is called on a client created without one.
var ErrAPIKeyRequired = errors.New("shortn: an API key is required for this endpoint")

// Client is a Shortn Link API client for trusted server-side code. It is safe
// for concurrent use.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	retry      RetryOptions
	userAgent  string
	// sleep waits between retries; tests replace it.
	sleep func(ctx context.Context, delay time.Duration) error

	Links        *LinksService
	Analytics    *AnalyticsService
	Workspaces   *WorkspacesService
	Tags         *TagsService
	UTMTemplates *UTMTemplatesService
	Destinations *DestinationsService
	Teams        *TeamsService
	Domains      *DomainsService
	Reports      *ReportsService
	Conversions  *ConversionsService
	Public       *PublicService
}

type service struct{ client *Client }

// Option configures a Client.
type Option func(*Client) error

// RetryOptions controls retries. The client retries HTTP 429 and 503
// answers (waiting for Retry-After when the server sends it, up to
// MaxRetryAfter) and network errors, but only for GET, PUT and DELETE
// requests and for POST requests whose body carries an idempotency_key (the
// conversion endpoints). A POST whose body has test_mode set to true is never
// retried: the API does not deduplicate test events, so a retry after a lost
// answer would record a second one.
type RetryOptions struct {
	// MaxRetries is how many times a request is retried after the first
	// attempt. Default 2.
	MaxRetries int
	// InitialDelay is the wait before the first retry when the server sends
	// no Retry-After. Default 500ms.
	InitialDelay time.Duration
	// MaxDelay caps the exponential backoff (not Retry-After). Default 5s.
	MaxDelay time.Duration
	// BackoffFactor multiplies the delay after every retry. Default 2.
	BackoffFactor float64
	// Jitter randomises each backoff delay by ±20%, still within MaxDelay.
	Jitter bool
	// MaxRetryAfter is the longest Retry-After the client waits for. When
	// the server asks for a longer wait, such as the seconds until the
	// monthly quota resets (code api_monthly_limit_exceeded), the client
	// does not wait or retry and returns the *APIError at once; its
	// RetryAfter field holds the header. Default 60s.
	MaxRetryAfter time.Duration
	// Disabled turns retries off.
	Disabled bool
}

// DefaultRetryOptions returns the retry policy a new client uses.
func DefaultRetryOptions() RetryOptions {
	return RetryOptions{
		MaxRetries:    2,
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2,
		Jitter:        true,
		MaxRetryAfter: defaultMaxRetryAfter,
	}
}

// NewClient creates a client. apiKey may be empty: such a client can call
// only the client.Public helpers; every other method returns
// ErrAPIKeyRequired.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	c := &Client{
		apiKey:     strings.TrimSpace(apiKey),
		baseURL:    DefaultBaseURL,
		httpClient: newDefaultHTTPClient(),
		retry:      DefaultRetryOptions(),
		userAgent:  userAgent,
		sleep:      sleep,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	s := &service{client: c}
	c.Links = (*LinksService)(s)
	c.Analytics = (*AnalyticsService)(s)
	c.Workspaces = (*WorkspacesService)(s)
	c.Tags = (*TagsService)(s)
	c.UTMTemplates = (*UTMTemplatesService)(s)
	c.Destinations = (*DestinationsService)(s)
	c.Teams = (*TeamsService)(s)
	c.Domains = (*DomainsService)(s)
	c.Reports = (*ReportsService)(s)
	c.Conversions = (*ConversionsService)(s)
	c.Public = (*PublicService)(s)
	return c, nil
}

// newDefaultHTTPClient is the client NewClient uses without WithHTTPClient.
// It does not follow redirects: Go would turn a POST answered with 301 or
// 302 into a GET, so a redirect surfaces as an *APIError instead.
func newDefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: defaultTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// NewPublicClient creates a client without an API key, for the client.Public
// helpers (URL validation, QR codes, the plan catalog).
func NewPublicClient(opts ...Option) (*Client, error) {
	return NewClient("", opts...)
}

// WithBaseURL overrides the Shortn Link origin, such as
// "https://app.shortn.link". The client appends /api/v1.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) error {
		baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		if baseURL == "" {
			return errors.New("shortn: base url is required")
		}
		u, err := url.Parse(baseURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("shortn: invalid base url %q", baseURL)
		}
		c.baseURL = strings.TrimSuffix(baseURL, "/api/v1")
		return nil
	}
}

// WithHTTPClient replaces the HTTP client, for proxies, custom transports or
// tests. Its own Timeout applies unless WithTimeout comes after it.
//
// The SDK's default client does not follow redirects, so a 3xx answer is
// returned as an *APIError. A client passed here is used as it is,
// including its CheckRedirect policy (Go's default follows up to 10
// redirects, turning a redirected POST into a GET); set CheckRedirect to
// return http.ErrUseLastResponse to keep the SDK's behaviour.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) error {
		if httpClient == nil {
			return errors.New("shortn: http client is required")
		}
		c.httpClient = httpClient
		return nil
	}
}

// WithTimeout sets the timeout of each HTTP attempt. Zero means no timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) error {
		if timeout < 0 {
			return errors.New("shortn: timeout must be zero or greater")
		}
		next := *c.httpClient
		next.Timeout = timeout
		c.httpClient = &next
		return nil
	}
}

// WithRetry sets the retry policy. Unset fields take their defaults; set
// Disabled to turn retries off.
func WithRetry(retry RetryOptions) Option {
	return func(c *Client) error {
		c.retry = normalizeRetry(retry)
		return nil
	}
}

// WithUserAgent appends an application identifier to the User-Agent header.
func WithUserAgent(suffix string) Option {
	return func(c *Client) error {
		if suffix = strings.TrimSpace(suffix); suffix != "" {
			c.userAgent = userAgent + " " + suffix
		}
		return nil
	}
}

type request struct {
	method         string
	path           string
	query          url.Values
	body           any
	public         bool
	idempotencyKey string
	accept         string
	// testMode is set by send when a POST body has "test_mode": true.
	testMode bool
}

// pathEscape joins path segments, escaping each one.
func pathEscape(segments ...string) string {
	var b strings.Builder
	for _, s := range segments {
		b.WriteByte('/')
		b.WriteString(url.PathEscape(s))
	}
	return b.String()
}

func (c *Client) do(ctx context.Context, req request, out any) error {
	raw, err := c.send(ctx, req)
	if err != nil {
		return err
	}
	if out == nil || len(bytes.TrimSpace(raw.body)) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw.body, out); err != nil {
		return &APIError{
			StatusCode: raw.status,
			Message:    "shortn: could not decode the response: " + err.Error(),
			Body:       raw.body,
		}
	}
	return nil
}

type rawResponse struct {
	status int
	header http.Header
	body   []byte
}

func (c *Client) send(ctx context.Context, req request) (*rawResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !req.public && c.apiKey == "" {
		return nil, ErrAPIKeyRequired
	}
	var encoded []byte
	if req.body != nil {
		var err error
		if encoded, err = json.Marshal(req.body); err != nil {
			return nil, fmt.Errorf("shortn: encode request: %w", err)
		}
		req.testMode = req.method == http.MethodPost && isTestModeBody(encoded)
	}
	endpoint := c.baseURL + "/api/v1" + req.path
	if len(req.query) > 0 {
		endpoint += "?" + req.query.Encode()
	}
	accept := req.accept
	if accept == "" {
		accept = "application/json"
	}
	for attempt := 0; ; attempt++ {
		var body io.Reader
		if encoded != nil {
			body = bytes.NewReader(encoded)
		}
		httpReq, err := http.NewRequestWithContext(ctx, req.method, endpoint, body)
		if err != nil {
			return nil, fmt.Errorf("shortn: build request: %w", err)
		}
		httpReq.Header.Set("Accept", accept)
		httpReq.Header.Set("User-Agent", c.userAgent)
		if c.apiKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		}
		if encoded != nil {
			httpReq.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(httpReq)
		var data []byte
		if err == nil {
			data, err = io.ReadAll(resp.Body)
			resp.Body.Close()
		}
		if err != nil {
			if ctx.Err() == nil && canRetry(req, c.retry, attempt) {
				if waitErr := c.sleep(ctx, backoff(c.retry, attempt)); waitErr != nil {
					return nil, waitErr
				}
				continue
			}
			return nil, fmt.Errorf("shortn: request failed: %w", err)
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return &rawResponse{status: resp.StatusCode, header: resp.Header, body: data}, nil
		}
		retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable
		if retryable && canRetry(req, c.retry, attempt) {
			delay, ok := parseRetryAfter(resp.Header.Get("Retry-After"))
			if !ok {
				delay = backoff(c.retry, attempt)
			} else if delay > c.retry.MaxRetryAfter {
				return nil, newAPIError(resp, data)
			}
			if waitErr := c.sleep(ctx, delay); waitErr != nil {
				return nil, waitErr
			}
			continue
		}
		return nil, newAPIError(resp, data)
	}
}

func normalizeRetry(retry RetryOptions) RetryOptions {
	if retry.Disabled {
		return RetryOptions{Disabled: true}
	}
	if retry.MaxRetries < 0 {
		retry.MaxRetries = 0
	}
	if retry.InitialDelay <= 0 {
		retry.InitialDelay = 500 * time.Millisecond
	}
	if retry.MaxDelay <= 0 {
		retry.MaxDelay = 5 * time.Second
	}
	if retry.BackoffFactor < 1 {
		retry.BackoffFactor = 2
	}
	if retry.MaxRetryAfter <= 0 {
		retry.MaxRetryAfter = defaultMaxRetryAfter
	}
	return retry
}

func canRetry(req request, retry RetryOptions, attempt int) bool {
	if retry.Disabled || attempt >= retry.MaxRetries {
		return false
	}
	switch req.method {
	case http.MethodGet, http.MethodPut, http.MethodDelete:
		return true
	case http.MethodPost:
		// The API never deduplicates test events, even with an
		// idempotency_key, so a test-mode POST is sent once.
		return strings.TrimSpace(req.idempotencyKey) != "" && !req.testMode
	}
	return false
}

// isTestModeBody reports whether a JSON request body has "test_mode": true.
func isTestModeBody(body []byte) bool {
	var probe struct {
		TestMode bool `json:"test_mode"`
	}
	return json.Unmarshal(body, &probe) == nil && probe.TestMode
}

func backoff(retry RetryOptions, attempt int) time.Duration {
	delay := float64(retry.InitialDelay) * math.Pow(retry.BackoffFactor, float64(attempt))
	if retry.Jitter && delay > 0 {
		delay *= 0.8 + rand.Float64()*0.4
	}
	// Cap after the jitter, so no wait exceeds MaxDelay.
	if delay > float64(retry.MaxDelay) {
		delay = float64(retry.MaxDelay)
	}
	return time.Duration(delay)
}

func parseRetryAfter(value string) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.ParseFloat(value, 64); err == nil && seconds >= 0 {
		return time.Duration(seconds * float64(time.Second)), true
	}
	if ts, err := http.ParseTime(value); err == nil {
		return max(time.Until(ts), 0), true
	}
	return 0, false
}

func sleep(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// decodeObject decodes data into v and keeps every field, known or not, in
// raw, so callers can read response fields this SDK does not model yet.
func decodeObject(data []byte, v any, raw *map[string]any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return err
	}
	var m map[string]any
	if json.Unmarshal(data, &m) == nil {
		*raw = m
	}
	return nil
}

// String returns a pointer to s, for optional request fields.
func String(s string) *string { return &s }

// Bool returns a pointer to b, for optional request fields.
func Bool(b bool) *bool { return &b }

// Int returns a pointer to i, for optional request fields.
func Int(i int) *int { return &i }

// Int64 returns a pointer to i, for optional request fields.
func Int64(i int64) *int64 { return &i }

// Strings returns a pointer to a non-nil list, for list fields where an empty
// list means something: Strings() sends [] (for example to make a tag
// workspace-wide again), while a nil pointer leaves the field out.
func Strings(values ...string) *[]string {
	list := append([]string{}, values...)
	return &list
}

// FormatTime formats t as RFC 3339 in UTC, as the API expects for times such
// as starts_at, expires_at and occurred_at.
func FormatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func setString(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}

func setInt(q url.Values, key string, value int) {
	if value != 0 {
		q.Set(key, strconv.Itoa(value))
	}
}

func setList(q url.Values, key string, values []string) {
	if len(values) > 0 {
		q.Set(key, strings.Join(values, ","))
	}
}
