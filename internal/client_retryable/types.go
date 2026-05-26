package client_retryable

import "fmt"

// KnowledgeNote represents a knowledge note from the Devin API v3.
type KnowledgeNote struct {
	NoteID     string  `json:"note_id"`
	FolderID   *string `json:"folder_id"`
	FolderPath string  `json:"folder_path"`
	Name       string  `json:"name"`
	Body       string  `json:"body"`
	Trigger    string  `json:"trigger"`
	IsEnabled  bool    `json:"is_enabled"`
	CreatedAt  int64   `json:"created_at"`
	UpdatedAt  int64   `json:"updated_at"`
	AccessType string  `json:"access_type"`
	OrgID      string  `json:"org_id"`
	Macro      *string `json:"macro"`
	PinnedRepo *string `json:"pinned_repo"`
}

// CreateKnowledgeNoteRequest is the request body for creating a knowledge note.
type CreateKnowledgeNoteRequest struct {
	Name       string  `json:"name"`
	Body       string  `json:"body"`
	Trigger    string  `json:"trigger"`
	PinnedRepo *string `json:"pinned_repo,omitempty"`
}

// UpdateKnowledgeNoteRequest is the request body for updating a knowledge note (full replace).
type UpdateKnowledgeNoteRequest struct {
	Name       string  `json:"name"`
	Body       string  `json:"body"`
	Trigger    string  `json:"trigger"`
	PinnedRepo *string `json:"pinned_repo,omitempty"`
}

// ListKnowledgeNotesResponse is the response for listing knowledge notes.
type ListKnowledgeNotesResponse struct {
	Items       []KnowledgeNote `json:"items"`
	EndCursor   string          `json:"end_cursor"`
	HasNextPage bool            `json:"has_next_page"`
	Total       int             `json:"total"`
}

// APIError represents an error response from the Devin API.
type APIError struct {
	StatusCode int
	Detail     interface{} `json:"detail"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("devin api error (status %d): %v", e.StatusCode, e.Detail)
}

// ValidationErrorDetail represents a single validation error from the API.
type ValidationErrorDetail struct {
	Type string   `json:"type"`
	Loc  []string `json:"loc"`
	Msg  string   `json:"msg"`
}

// RateLimitError is returned when the API returns 429.
type RateLimitError struct {
	*APIError
	RetryAfter string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("devin api rate limited (retry-after: %s): %v", e.RetryAfter, e.Detail)
}

// ServerError is returned when the API returns 5xx.
type ServerError struct {
	*APIError
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("devin api server error (status %d): %v", e.StatusCode, e.Detail)
}

// NotFoundError is returned when the API returns 404.
type NotFoundError struct {
	*APIError
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("devin api not found: %v", e.Detail)
}

// UnauthorizedError is returned when the API returns 401 or 403.
type UnauthorizedError struct {
	*APIError
}

func (e *UnauthorizedError) Error() string {
	return fmt.Sprintf("devin api unauthorized: %v", e.Detail)
}
