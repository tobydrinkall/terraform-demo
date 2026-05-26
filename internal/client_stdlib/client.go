package client_stdlib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL   = "https://api.devin.ai/v3"
	defaultUserAgent = "terraform-provider-devin/0.1.0"
	defaultMaxRetries = 3
	defaultBaseDelay  = 1 * time.Second
	defaultMaxDelay   = 30 * time.Second
)

// Logger defines the interface for structured logging compatible with
// terraform-plugin-log (hclog).
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// defaultLogger wraps the standard library logger.
type defaultLogger struct{}

func (l *defaultLogger) Debug(msg string, keysAndValues ...interface{}) {
	log.Printf("[DEBUG] %s %v", msg, keysAndValues)
}
func (l *defaultLogger) Info(msg string, keysAndValues ...interface{}) {
	log.Printf("[INFO] %s %v", msg, keysAndValues)
}
func (l *defaultLogger) Warn(msg string, keysAndValues ...interface{}) {
	log.Printf("[WARN] %s %v", msg, keysAndValues)
}
func (l *defaultLogger) Error(msg string, keysAndValues ...interface{}) {
	log.Printf("[ERROR] %s %v", msg, keysAndValues)
}

// Client is the Devin API v3 HTTP client using net/http.
type Client struct {
	apiKey     string
	baseURL    string
	orgID      string
	httpClient *http.Client
	userAgent  string
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
	logger     Logger
}

// Option configures the Client.
type Option func(*Client)

// WithHTTPClient sets a custom http.Client.
func WithHTTPClient(c *http.Client) Option {
	return func(client *Client) {
		client.httpClient = c
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(client *Client) {
		client.userAgent = ua
	}
}

// WithMaxRetries sets the maximum number of retries for retryable errors.
func WithMaxRetries(n int) Option {
	return func(client *Client) {
		client.maxRetries = n
	}
}

// WithBaseDelay sets the base delay for exponential backoff.
func WithBaseDelay(d time.Duration) Option {
	return func(client *Client) {
		client.baseDelay = d
	}
}

// WithMaxDelay sets the maximum delay for exponential backoff.
func WithMaxDelay(d time.Duration) Option {
	return func(client *Client) {
		client.maxDelay = d
	}
}

// WithLogger sets the logger for request/response logging.
func WithLogger(l Logger) Option {
	return func(client *Client) {
		client.logger = l
	}
}

// NewClient creates a new Devin API v3 client using net/http.
func NewClient(apiKey, orgID string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		orgID:      orgID,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		userAgent:  defaultUserAgent,
		maxRetries: defaultMaxRetries,
		baseDelay:  defaultBaseDelay,
		maxDelay:   defaultMaxDelay,
		logger:     &defaultLogger{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithBaseURL sets the base URL (useful for testing).
func WithBaseURL(u string) Option {
	return func(client *Client) {
		client.baseURL = strings.TrimRight(u, "/")
	}
}

func (c *Client) knowledgeBasePath() string {
	return fmt.Sprintf("%s/organizations/%s/knowledge/notes", c.baseURL, c.orgID)
}

func (c *Client) knowledgeNotePath(noteID string) string {
	return fmt.Sprintf("%s/%s", c.knowledgeBasePath(), noteID)
}

// doRequest executes an HTTP request with retry logic for 429 and 5xx errors.
func (c *Client) doRequest(ctx context.Context, method, urlStr string, body interface{}) ([]byte, int, error) {
	var bodyReader io.Reader
	var bodyBytes []byte

	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request body: %w", err)
		}
	}

	var lastErr error
	retryAfterUsed := false
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 && !retryAfterUsed {
			delay := c.calculateBackoff(attempt)
			c.logger.Warn("retrying request",
				"attempt", attempt,
				"max_retries", c.maxRetries,
				"delay", delay.String(),
				"method", method,
				"url", urlStr,
			)
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(delay):
			}
		}
		retryAfterUsed = false

		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
		if err != nil {
			return nil, 0, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("User-Agent", c.userAgent)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		c.logger.Debug("sending request",
			"method", method,
			"url", urlStr,
			"attempt", attempt,
		)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("executing request: %w", err)
			if isRetryableNetworkError(err) {
				continue
			}
			return nil, 0, lastErr
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("reading response body: %w", err)
			continue
		}

		c.logger.Debug("received response",
			"status", resp.StatusCode,
			"body_length", len(respBody),
			"method", method,
			"url", urlStr,
		)

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := resp.Header.Get("Retry-After")
			lastErr = &RateLimitError{
				APIError:   &APIError{StatusCode: resp.StatusCode},
				RetryAfter: retryAfter,
			}
			parseAPIErrorDetail(respBody, lastErr.(*RateLimitError).APIError)
			if retryAfter != "" {
				if secs, err := strconv.Atoi(retryAfter); err == nil {
					c.logger.Warn("rate limited, using Retry-After header",
						"retry_after_seconds", secs,
					)
					select {
					case <-ctx.Done():
						return nil, 0, ctx.Err()
					case <-time.After(time.Duration(secs) * time.Second):
					}
					retryAfterUsed = true
				}
			}
			continue
		}

		if resp.StatusCode >= 500 {
			apiErr := &APIError{StatusCode: resp.StatusCode}
			parseAPIErrorDetail(respBody, apiErr)
			lastErr = &ServerError{APIError: apiErr}
			continue
		}

		if resp.StatusCode >= 400 {
			return nil, resp.StatusCode, parseErrorResponse(resp.StatusCode, respBody)
		}

		return respBody, resp.StatusCode, nil
	}

	return nil, 0, fmt.Errorf("max retries (%d) exceeded: %w", c.maxRetries, lastErr)
}

func (c *Client) calculateBackoff(attempt int) time.Duration {
	backoff := float64(c.baseDelay) * math.Pow(2, float64(attempt-1))
	jitter := rand.Float64() * float64(c.baseDelay)
	delay := time.Duration(backoff + jitter)
	if delay > c.maxDelay {
		delay = c.maxDelay
	}
	return delay
}

func isRetryableNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "i/o timeout")
}

func parseAPIErrorDetail(body []byte, apiErr *APIError) {
	var raw struct {
		Detail interface{} `json:"detail"`
	}
	if err := json.Unmarshal(body, &raw); err == nil {
		apiErr.Detail = raw.Detail
	} else {
		apiErr.Detail = string(body)
	}
}

func parseErrorResponse(statusCode int, body []byte) error {
	apiErr := &APIError{StatusCode: statusCode}
	parseAPIErrorDetail(body, apiErr)

	switch statusCode {
	case http.StatusNotFound:
		return &NotFoundError{APIError: apiErr}
	case http.StatusUnauthorized, http.StatusForbidden:
		return &UnauthorizedError{APIError: apiErr}
	default:
		return apiErr
	}
}

// CreateKnowledgeNote creates a new knowledge note.
func (c *Client) CreateKnowledgeNote(ctx context.Context, req CreateKnowledgeNoteRequest) (*KnowledgeNote, error) {
	respBody, _, err := c.doRequest(ctx, http.MethodPost, c.knowledgeBasePath(), req)
	if err != nil {
		return nil, fmt.Errorf("creating knowledge note: %w", err)
	}

	var note KnowledgeNote
	if err := json.Unmarshal(respBody, &note); err != nil {
		return nil, fmt.Errorf("decoding create response: %w", err)
	}
	return &note, nil
}

// GetKnowledgeNote retrieves a knowledge note by ID.
func (c *Client) GetKnowledgeNote(ctx context.Context, noteID string) (*KnowledgeNote, error) {
	respBody, _, err := c.doRequest(ctx, http.MethodGet, c.knowledgeNotePath(noteID), nil)
	if err != nil {
		return nil, fmt.Errorf("getting knowledge note: %w", err)
	}

	var note KnowledgeNote
	if err := json.Unmarshal(respBody, &note); err != nil {
		return nil, fmt.Errorf("decoding get response: %w", err)
	}
	return &note, nil
}

// UpdateKnowledgeNote updates an existing knowledge note (full replace).
func (c *Client) UpdateKnowledgeNote(ctx context.Context, noteID string, req UpdateKnowledgeNoteRequest) (*KnowledgeNote, error) {
	respBody, _, err := c.doRequest(ctx, http.MethodPut, c.knowledgeNotePath(noteID), req)
	if err != nil {
		return nil, fmt.Errorf("updating knowledge note: %w", err)
	}

	var note KnowledgeNote
	if err := json.Unmarshal(respBody, &note); err != nil {
		return nil, fmt.Errorf("decoding update response: %w", err)
	}
	return &note, nil
}

// DeleteKnowledgeNote deletes a knowledge note by ID.
func (c *Client) DeleteKnowledgeNote(ctx context.Context, noteID string) error {
	_, _, err := c.doRequest(ctx, http.MethodDelete, c.knowledgeNotePath(noteID), nil)
	if err != nil {
		return fmt.Errorf("deleting knowledge note: %w", err)
	}
	return nil
}

// ListKnowledgeNotes lists knowledge notes with optional pagination.
func (c *Client) ListKnowledgeNotes(ctx context.Context, first int, after string) (*ListKnowledgeNotesResponse, error) {
	u, err := url.Parse(c.knowledgeBasePath())
	if err != nil {
		return nil, fmt.Errorf("parsing base URL: %w", err)
	}

	q := u.Query()
	if first > 0 {
		q.Set("first", strconv.Itoa(first))
	}
	if after != "" {
		q.Set("after", after)
	}
	u.RawQuery = q.Encode()

	respBody, _, err := c.doRequest(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("listing knowledge notes: %w", err)
	}

	var resp ListKnowledgeNotesResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("decoding list response: %w", err)
	}
	return &resp, nil
}
