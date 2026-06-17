package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles communication with the Devin API v3.
type Client struct {
	BaseURL    string
	OrgID      string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new Devin API client.
func NewClient(baseURL, orgID, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		OrgID:   orgID,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// KnowledgeNote represents a knowledge note resource.
type KnowledgeNote struct {
	NoteID     string `json:"note_id"`
	Name       string `json:"name"`
	Trigger    string `json:"trigger"`
	Body       string `json:"body"`
	FolderID   string `json:"folder_id,omitempty"`
	FolderPath string `json:"folder_path,omitempty"`
	IsEnabled  bool   `json:"is_enabled"`
	Macro      string `json:"macro,omitempty"`
	OrgID      string `json:"org_id,omitempty"`
	PinnedRepo string `json:"pinned_repo,omitempty"`
	AccessType string `json:"access_type,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

// KnowledgeNoteRequest is the request body for create/update operations.
type KnowledgeNoteRequest struct {
	Name       string  `json:"name"`
	Trigger    string  `json:"trigger"`
	Body       string  `json:"body"`
	FolderID   *string `json:"folder_id"`
	IsEnabled  *bool   `json:"is_enabled"`
	PinnedRepo *string `json:"pinned_repo"`
}

// CreateKnowledgeNote creates a new knowledge note.
func (c *Client) CreateKnowledgeNote(ctx context.Context, req KnowledgeNoteRequest) (*KnowledgeNote, error) {
	url := fmt.Sprintf("%s/organizations/%s/knowledge/notes", c.BaseURL, c.OrgID)
	return c.doNoteRequest(ctx, http.MethodPost, url, req)
}

// GetKnowledgeNote retrieves a knowledge note by ID.
func (c *Client) GetKnowledgeNote(ctx context.Context, noteID string) (*KnowledgeNote, error) {
	url := fmt.Sprintf("%s/organizations/%s/knowledge/notes/%s", c.BaseURL, c.OrgID, noteID)
	return c.doNoteRequest(ctx, http.MethodGet, url, nil)
}

// UpdateKnowledgeNote updates an existing knowledge note.
func (c *Client) UpdateKnowledgeNote(ctx context.Context, noteID string, req KnowledgeNoteRequest) (*KnowledgeNote, error) {
	url := fmt.Sprintf("%s/organizations/%s/knowledge/notes/%s", c.BaseURL, c.OrgID, noteID)
	return c.doNoteRequest(ctx, http.MethodPut, url, req)
}

// DeleteKnowledgeNote deletes a knowledge note.
func (c *Client) DeleteKnowledgeNote(ctx context.Context, noteID string) error {
	url := fmt.Sprintf("%s/organizations/%s/knowledge/notes/%s", c.BaseURL, c.OrgID, noteID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) doNoteRequest(ctx context.Context, method, url string, body interface{}) (*KnowledgeNote, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("note not found")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var note KnowledgeNote
	if err := json.NewDecoder(resp.Body).Decode(&note); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &note, nil
}
