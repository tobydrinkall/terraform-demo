package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	defaultBaseURL   = "https://api.devin.ai"
	defaultUserAgent = "terraform-provider-devin"
	defaultMaxRetries = 3
)

// Client is the Devin API v3 HTTP client.
type Client struct {
	baseURL    string
	apiKey     string
	userAgent  string
	httpClient *http.Client
}

// Option configures the Client.
type Option func(*Client)

// WithBaseURL overrides the default API base URL.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = u }
}

// WithHTTPClient sets a custom http.Client (useful for testing).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithUserAgent sets the User-Agent header value.
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.userAgent = ua }
}

// New creates a new Devin API client.
func New(apiKey string, opts ...Option) *Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = defaultMaxRetries
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 30 * time.Second
	retryClient.Logger = nil
	retryClient.CheckRetry = checkRetryPolicy

	c := &Client{
		baseURL:    defaultBaseURL,
		apiKey:     apiKey,
		userAgent:  defaultUserAgent,
		httpClient: retryClient.StandardClient(),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func checkRetryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if err != nil {
		return true, nil
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return true, nil
	}
	if resp.StatusCode >= 500 {
		return true, nil
	}
	return false, nil
}

// APIError represents an error response from the Devin API.
type APIError struct {
	StatusCode int
	Message    string
	Details    []ValidationDetail
}

// ValidationDetail is a single validation error from the API.
type ValidationDetail struct {
	Loc  []interface{} `json:"loc"`
	Msg  string        `json:"msg"`
	Type string        `json:"type"`
}

func (e *APIError) Error() string {
	if len(e.Details) > 0 {
		return fmt.Sprintf("Devin API error (HTTP %d): %s (details: %v)", e.StatusCode, e.Message, e.Details)
	}
	return fmt.Sprintf("Devin API error (HTTP %d): %s", e.StatusCode, e.Message)
}

func (c *Client) do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	u, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return fmt.Errorf("building URL: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Message: string(respBody)}
		var errResp struct {
			Detail interface{} `json:"detail"`
		}
		if json.Unmarshal(respBody, &errResp) == nil {
			switch d := errResp.Detail.(type) {
			case string:
				apiErr.Message = d
			case []interface{}:
				raw, _ := json.Marshal(d)
				var details []ValidationDetail
				if json.Unmarshal(raw, &details) == nil {
					apiErr.Details = details
					apiErr.Message = "validation error"
				}
			}
		}
		return apiErr
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}
	}

	return nil
}

// --- Knowledge Notes ---

// KnowledgeNoteCreateRequest is the request body for creating/updating a note.
type KnowledgeNoteCreateRequest struct {
	Name       string  `json:"name"`
	Body       string  `json:"body"`
	Trigger    string  `json:"trigger"`
	PinnedRepo *string `json:"pinned_repo,omitempty"`
}

// KnowledgeNoteResponse is the API response for a knowledge note.
type KnowledgeNoteResponse struct {
	NoteID     string  `json:"note_id"`
	Name       string  `json:"name"`
	Body       string  `json:"body"`
	Trigger    string  `json:"trigger"`
	PinnedRepo *string `json:"pinned_repo"`
	FolderID   *string `json:"folder_id"`
	FolderPath string  `json:"folder_path"`
	IsEnabled  bool    `json:"is_enabled"`
	Macro      *string `json:"macro"`
	AccessType string  `json:"access_type"`
	OrgID      *string `json:"org_id"`
	CreatedAt  int64   `json:"created_at"`
	UpdatedAt  int64   `json:"updated_at"`
}

// PaginatedNotesResponse is the paginated list response.
type PaginatedNotesResponse struct {
	Notes       []KnowledgeNoteResponse `json:"notes"`
	EndCursor   *string                 `json:"end_cursor"`
	HasNextPage bool                    `json:"has_next_page"`
}

func (c *Client) notesPath(orgID string) string {
	return fmt.Sprintf("/v3/organizations/%s/knowledge/notes", orgID)
}

func (c *Client) notePath(orgID, noteID string) string {
	return fmt.Sprintf("/v3/organizations/%s/knowledge/notes/%s", orgID, noteID)
}

// CreateNote creates a new knowledge note.
func (c *Client) CreateNote(ctx context.Context, orgID string, req *KnowledgeNoteCreateRequest) (*KnowledgeNoteResponse, error) {
	var resp KnowledgeNoteResponse
	if err := c.do(ctx, http.MethodPost, c.notesPath(orgID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetNote retrieves a knowledge note by ID.
func (c *Client) GetNote(ctx context.Context, orgID, noteID string) (*KnowledgeNoteResponse, error) {
	var resp KnowledgeNoteResponse
	if err := c.do(ctx, http.MethodGet, c.notePath(orgID, noteID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateNote updates a knowledge note (full replace via PUT).
func (c *Client) UpdateNote(ctx context.Context, orgID, noteID string, req *KnowledgeNoteCreateRequest) (*KnowledgeNoteResponse, error) {
	var resp KnowledgeNoteResponse
	if err := c.do(ctx, http.MethodPut, c.notePath(orgID, noteID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteNote deletes a knowledge note.
func (c *Client) DeleteNote(ctx context.Context, orgID, noteID string) (*KnowledgeNoteResponse, error) {
	var resp KnowledgeNoteResponse
	if err := c.do(ctx, http.MethodDelete, c.notePath(orgID, noteID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListNotesOptions configures the list request.
type ListNotesOptions struct {
	After      string
	First      int
	Search     string
	FolderPath string
	PinnedRepo string
}

// ListNotes lists knowledge notes with pagination.
func (c *Client) ListNotes(ctx context.Context, orgID string, opts *ListNotesOptions) (*PaginatedNotesResponse, error) {
	path := c.notesPath(orgID)
	params := url.Values{}
	if opts != nil {
		if opts.After != "" {
			params.Set("after", opts.After)
		}
		if opts.First > 0 {
			params.Set("first", strconv.Itoa(opts.First))
		}
		if opts.Search != "" {
			params.Set("search", opts.Search)
		}
		if opts.FolderPath != "" {
			params.Set("folder_path", opts.FolderPath)
		}
		if opts.PinnedRepo != "" {
			params.Set("pinned_repo", opts.PinnedRepo)
		}
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var resp PaginatedNotesResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// IsNotFound returns true if the error is a 404.
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}
