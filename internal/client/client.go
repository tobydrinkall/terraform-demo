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
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("parsing base URL: %w", err)
	}
	ref, err := url.Parse(path)
	if err != nil {
		return fmt.Errorf("parsing path: %w", err)
	}
	u := base.ResolveReference(ref).String()

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
	PinnedRepo *string `json:"pinned_repo"`
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

// --- Playbooks ---

// PlaybookCreateRequest is the request body for creating/updating a playbook.
type PlaybookCreateRequest struct {
	Title   string  `json:"title"`
	Content string  `json:"content"`
	Macro   *string `json:"macro"`
}

// PlaybookResponse is the API response for a playbook.
type PlaybookResponse struct {
	PlaybookID string  `json:"playbook_id"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Macro      *string `json:"macro"`
	CreatedAt  int64   `json:"created_at"`
	UpdatedAt  int64   `json:"updated_at"`
}

// PaginatedPlaybooksResponse is the paginated list response.
type PaginatedPlaybooksResponse struct {
	Playbooks   []PlaybookResponse `json:"playbooks"`
	EndCursor   *string            `json:"end_cursor"`
	HasNextPage bool               `json:"has_next_page"`
}

func (c *Client) playbooksPath(orgID string) string {
	return fmt.Sprintf("/v3/organizations/%s/playbooks", orgID)
}

func (c *Client) playbookPath(orgID, playbookID string) string {
	return fmt.Sprintf("/v3/organizations/%s/playbooks/%s", orgID, playbookID)
}

// CreatePlaybook creates a new playbook.
func (c *Client) CreatePlaybook(ctx context.Context, orgID string, req *PlaybookCreateRequest) (*PlaybookResponse, error) {
	var resp PlaybookResponse
	if err := c.do(ctx, http.MethodPost, c.playbooksPath(orgID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetPlaybook retrieves a playbook by ID.
func (c *Client) GetPlaybook(ctx context.Context, orgID, playbookID string) (*PlaybookResponse, error) {
	var resp PlaybookResponse
	if err := c.do(ctx, http.MethodGet, c.playbookPath(orgID, playbookID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdatePlaybook updates a playbook (full replace via PUT).
func (c *Client) UpdatePlaybook(ctx context.Context, orgID, playbookID string, req *PlaybookCreateRequest) (*PlaybookResponse, error) {
	var resp PlaybookResponse
	if err := c.do(ctx, http.MethodPut, c.playbookPath(orgID, playbookID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListPlaybooks lists playbooks with pagination.
func (c *Client) ListPlaybooks(ctx context.Context, orgID string, after string, first int) (*PaginatedPlaybooksResponse, error) {
	path := c.playbooksPath(orgID)
	params := url.Values{}
	if after != "" {
		params.Set("after", after)
	}
	if first > 0 {
		params.Set("first", strconv.Itoa(first))
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var resp PaginatedPlaybooksResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeletePlaybook deletes a playbook.
func (c *Client) DeletePlaybook(ctx context.Context, orgID, playbookID string) error {
	return c.do(ctx, http.MethodDelete, c.playbookPath(orgID, playbookID), nil, nil)
}

// --- Schedules ---

// ScheduleCreateRequest is the request body for creating a schedule.
type ScheduleCreateRequest struct {
	Name           string  `json:"name"`
	Prompt         string  `json:"prompt"`
	PlaybookID     *string `json:"playbook_id,omitempty"`
	Frequency      *string `json:"frequency,omitempty"`
	ScheduleType   string  `json:"schedule_type"`
	ScheduledAt    *string `json:"scheduled_at,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
	NotifyOn       *string `json:"notify_on,omitempty"`
	Agent          *string `json:"agent,omitempty"`
	BypassApproval *bool   `json:"bypass_approval,omitempty"`
}

// ScheduleUpdateRequest is the request body for updating a schedule (PATCH).
type ScheduleUpdateRequest struct {
	Name           *string `json:"name,omitempty"`
	Prompt         *string `json:"prompt,omitempty"`
	PlaybookID     *string `json:"playbook_id,omitempty"`
	Frequency      *string `json:"frequency,omitempty"`
	ScheduledAt    *string `json:"scheduled_at,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
	NotifyOn       *string `json:"notify_on,omitempty"`
	Agent          *string `json:"agent,omitempty"`
	BypassApproval *bool   `json:"bypass_approval,omitempty"`
}

// ScheduleResponse is the API response for a schedule.
type ScheduleResponse struct {
	ScheduleID     string  `json:"schedule_id"`
	Name           string  `json:"name"`
	Prompt         string  `json:"prompt"`
	PlaybookID     *string `json:"playbook_id"`
	Frequency      *string `json:"frequency"`
	ScheduleType   string  `json:"schedule_type"`
	ScheduledAt    *string `json:"scheduled_at"`
	Enabled        bool    `json:"enabled"`
	NotifyOn       string  `json:"notify_on"`
	Agent          string  `json:"agent"`
	BypassApproval bool    `json:"bypass_approval"`
	CreatedAt      int64   `json:"created_at"`
	UpdatedAt      int64   `json:"updated_at"`
}

// PaginatedSchedulesResponse is the paginated list response.
type PaginatedSchedulesResponse struct {
	Schedules []ScheduleResponse `json:"schedules"`
	Total     int                `json:"total"`
}

func (c *Client) schedulesPath(orgID string) string {
	return fmt.Sprintf("/v3/organizations/%s/schedules", orgID)
}

func (c *Client) schedulePath(orgID, scheduleID string) string {
	return fmt.Sprintf("/v3/organizations/%s/schedules/%s", orgID, scheduleID)
}

// CreateSchedule creates a new schedule.
func (c *Client) CreateSchedule(ctx context.Context, orgID string, req *ScheduleCreateRequest) (*ScheduleResponse, error) {
	var resp ScheduleResponse
	if err := c.do(ctx, http.MethodPost, c.schedulesPath(orgID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetSchedule retrieves a schedule by ID.
func (c *Client) GetSchedule(ctx context.Context, orgID, scheduleID string) (*ScheduleResponse, error) {
	var resp ScheduleResponse
	if err := c.do(ctx, http.MethodGet, c.schedulePath(orgID, scheduleID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateSchedule updates a schedule (PATCH semantics — partial update).
func (c *Client) UpdateSchedule(ctx context.Context, orgID, scheduleID string, req *ScheduleUpdateRequest) (*ScheduleResponse, error) {
	var resp ScheduleResponse
	if err := c.do(ctx, http.MethodPatch, c.schedulePath(orgID, scheduleID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateScheduleRaw updates a schedule using a raw map body, allowing explicit null values
// to clear fields under PATCH semantics.
func (c *Client) UpdateScheduleRaw(ctx context.Context, orgID, scheduleID string, body map[string]interface{}) (*ScheduleResponse, error) {
	var resp ScheduleResponse
	if err := c.do(ctx, http.MethodPatch, c.schedulePath(orgID, scheduleID), body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteSchedule deletes a schedule.
func (c *Client) DeleteSchedule(ctx context.Context, orgID, scheduleID string) error {
	return c.do(ctx, http.MethodDelete, c.schedulePath(orgID, scheduleID), nil, nil)
}

// --- Secrets ---

// SecretCreateRequest is the request body for creating a secret.
type SecretCreateRequest struct {
	Key         string  `json:"key"`
	Value       string  `json:"value"`
	SecretType  string  `json:"type"`
	IsSensitive bool    `json:"is_sensitive"`
	Note        *string `json:"note,omitempty"`
}

// SecretResponse is the API response for a secret (metadata only, no value).
type SecretResponse struct {
	SecretID    string  `json:"secret_id"`
	Key         *string `json:"key"`
	SecretType  string  `json:"secret_type"`
	IsSensitive bool    `json:"is_sensitive"`
	Note        *string `json:"note"`
	AccessType  string  `json:"access_type"`
	CreatedBy   string  `json:"created_by"`
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   *int64  `json:"updated_at"`
	UpdatedBy   *string `json:"updated_by"`
}

// PaginatedSecretsResponse is the paginated list response.
type PaginatedSecretsResponse struct {
	Items       []SecretResponse `json:"items"`
	EndCursor   *string          `json:"end_cursor"`
	HasNextPage bool             `json:"has_next_page"`
	Total       *int             `json:"total"`
}

func (c *Client) secretsPath(orgID string) string {
	return fmt.Sprintf("/v3/organizations/%s/secrets", orgID)
}

func (c *Client) secretPath(orgID, secretID string) string {
	return fmt.Sprintf("/v3/organizations/%s/secrets/%s", orgID, secretID)
}

// CreateSecret creates a new secret.
func (c *Client) CreateSecret(ctx context.Context, orgID string, req *SecretCreateRequest) (*SecretResponse, error) {
	var resp SecretResponse
	if err := c.do(ctx, http.MethodPost, c.secretsPath(orgID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// FindSecretByID finds a secret by ID via list+filter (no GET-by-ID endpoint).
func (c *Client) FindSecretByID(ctx context.Context, orgID, secretID string) (*SecretResponse, error) {
	var cursor string
	for {
		path := c.secretsPath(orgID) + "?first=100"
		if cursor != "" {
			path += "&after=" + url.QueryEscape(cursor)
		}

		var resp PaginatedSecretsResponse
		if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
			return nil, err
		}

		for i := range resp.Items {
			if resp.Items[i].SecretID == secretID {
				return &resp.Items[i], nil
			}
		}

		if !resp.HasNextPage || resp.EndCursor == nil {
			break
		}
		cursor = *resp.EndCursor
	}

	return nil, &APIError{StatusCode: http.StatusNotFound, Message: "secret not found: " + secretID}
}

// DeleteSecret deletes a secret.
func (c *Client) DeleteSecret(ctx context.Context, orgID, secretID string) error {
	return c.do(ctx, http.MethodDelete, c.secretPath(orgID, secretID), nil, nil)
}
