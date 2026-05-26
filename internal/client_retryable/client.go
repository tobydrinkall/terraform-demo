package client_retryable

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	defaultBaseURL    = "https://api.devin.ai/v3"
	defaultUserAgent  = "terraform-provider-devin/0.1.0"
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

// hclogAdapter adapts our Logger interface to retryablehttp.LeveledLogger.
type hclogAdapter struct {
	logger Logger
}

func (a *hclogAdapter) Error(msg string, keysAndValues ...interface{}) {
	a.logger.Error(msg, keysAndValues...)
}
func (a *hclogAdapter) Info(msg string, keysAndValues ...interface{}) {
	a.logger.Info(msg, keysAndValues...)
}
func (a *hclogAdapter) Debug(msg string, keysAndValues ...interface{}) {
	a.logger.Debug(msg, keysAndValues...)
}
func (a *hclogAdapter) Warn(msg string, keysAndValues ...interface{}) {
	a.logger.Warn(msg, keysAndValues...)
}

// Client is the Devin API v3 HTTP client using hashicorp/go-retryablehttp.
type Client struct {
	apiKey     string
	baseURL    string
	orgID      string
	httpClient *retryablehttp.Client
	userAgent  string
	logger     Logger
}

// Option configures the Client.
type Option func(*Client)

// WithHTTPClient sets a custom underlying http.Client.
func WithHTTPClient(c *http.Client) Option {
	return func(client *Client) {
		client.httpClient.HTTPClient = c
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(client *Client) {
		client.userAgent = ua
	}
}

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(n int) Option {
	return func(client *Client) {
		client.httpClient.RetryMax = n
	}
}

// WithBaseDelay sets the minimum wait time for retries.
func WithBaseDelay(d time.Duration) Option {
	return func(client *Client) {
		client.httpClient.RetryWaitMin = d
	}
}

// WithMaxDelay sets the maximum wait time for retries.
func WithMaxDelay(d time.Duration) Option {
	return func(client *Client) {
		client.httpClient.RetryWaitMax = d
	}
}

// WithLogger sets the logger.
func WithLogger(l Logger) Option {
	return func(client *Client) {
		client.logger = l
		client.httpClient.Logger = &hclogAdapter{logger: l}
	}
}

// WithBaseURL sets the base URL (useful for testing).
func WithBaseURL(u string) Option {
	return func(client *Client) {
		client.baseURL = strings.TrimRight(u, "/")
	}
}

// NewClient creates a new Devin API v3 client using hashicorp/go-retryablehttp.
func NewClient(apiKey, orgID string, opts ...Option) *Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = defaultMaxRetries
	retryClient.RetryWaitMin = defaultBaseDelay
	retryClient.RetryWaitMax = defaultMaxDelay
	retryClient.HTTPClient.Timeout = 30 * time.Second

	retryClient.CheckRetry = customCheckRetry
	retryClient.ErrorHandler = retryablehttp.PassthroughErrorHandler

	c := &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		orgID:      orgID,
		httpClient: retryClient,
		userAgent:  defaultUserAgent,
		logger:     nil,
	}
	for _, opt := range opts {
		opt(c)
	}

	if c.logger != nil {
		retryClient.Logger = &hclogAdapter{logger: c.logger}
	} else {
		retryClient.Logger = nil
	}

	return c
}

// customCheckRetry retries on 429 and 5xx responses.
func customCheckRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return true, nil
	}

	if resp.StatusCode >= 500 {
		return true, nil
	}

	return false, nil
}

func (c *Client) knowledgeBasePath() string {
	return fmt.Sprintf("%s/organizations/%s/knowledge/notes", c.baseURL, c.orgID)
}

func (c *Client) knowledgeNotePath(noteID string) string {
	return fmt.Sprintf("%s/%s", c.knowledgeBasePath(), noteID)
}

// doRequest executes an HTTP request. Retries are handled by retryablehttp.
func (c *Client) doRequest(ctx context.Context, method, urlStr string, body interface{}) ([]byte, int, error) {
	var bodyReader io.Reader

	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.logger != nil {
		c.logger.Debug("sending request",
			"method", method,
			"url", urlStr,
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("reading response body: %w", err)
	}

	if c.logger != nil {
		c.logger.Debug("received response",
			"status", resp.StatusCode,
			"body_length", len(respBody),
			"method", method,
			"url", urlStr,
		)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		parseAPIErrorDetail(respBody, apiErr)
		return nil, resp.StatusCode, &RateLimitError{
			APIError:   apiErr,
			RetryAfter: resp.Header.Get("Retry-After"),
		}
	}

	if resp.StatusCode >= 500 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		parseAPIErrorDetail(respBody, apiErr)
		return nil, resp.StatusCode, &ServerError{APIError: apiErr}
	}

	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, parseErrorResponse(resp.StatusCode, respBody)
	}

	return respBody, resp.StatusCode, nil
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
