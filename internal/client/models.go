package client

// KnowledgeNoteCreateRequest is the request body for creating or updating a knowledge note.
type KnowledgeNoteCreateRequest struct {
	Name       string  `json:"name"`
	Body       string  `json:"body"`
	Trigger    string  `json:"trigger"`
	PinnedRepo *string `json:"pinned_repo"`
}

// KnowledgeNote is the API response for a single knowledge note.
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
	OrgID      *string `json:"org_id"`
	Macro      *string `json:"macro"`
	PinnedRepo *string `json:"pinned_repo"`
}

// PaginatedResponse wraps a paginated list of knowledge notes.
type PaginatedResponse struct {
	Items       []KnowledgeNote `json:"items"`
	EndCursor   *string         `json:"end_cursor"`
	HasNextPage bool            `json:"has_next_page"`
	Total       *int            `json:"total"`
}

// ListKnowledgeNotesOptions contains optional filters for listing notes.
type ListKnowledgeNotesOptions struct {
	After      *string
	First      *int
	Search     *string
	FolderPath *string
	PinnedRepo *string
}
