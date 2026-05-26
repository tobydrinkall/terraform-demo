package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// CreateKnowledgeNote creates a new knowledge note.
func (c *Client) CreateKnowledgeNote(ctx context.Context, req KnowledgeNoteCreateRequest) (*KnowledgeNote, error) {
	path := fmt.Sprintf("%s/knowledge/notes", c.orgPath())

	respBody, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var note KnowledgeNote
	if err := json.Unmarshal(respBody, &note); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &note, nil
}

// GetKnowledgeNote retrieves a knowledge note by ID.
func (c *Client) GetKnowledgeNote(ctx context.Context, noteID string) (*KnowledgeNote, error) {
	path := fmt.Sprintf("%s/knowledge/notes/%s", c.orgPath(), noteID)

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var note KnowledgeNote
	if err := json.Unmarshal(respBody, &note); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &note, nil
}

// UpdateKnowledgeNote updates an existing knowledge note (full replace).
func (c *Client) UpdateKnowledgeNote(ctx context.Context, noteID string, req KnowledgeNoteCreateRequest) (*KnowledgeNote, error) {
	path := fmt.Sprintf("%s/knowledge/notes/%s", c.orgPath(), noteID)

	respBody, err := c.doRequest(ctx, "PUT", path, req)
	if err != nil {
		return nil, err
	}

	var note KnowledgeNote
	if err := json.Unmarshal(respBody, &note); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &note, nil
}

// DeleteKnowledgeNote deletes a knowledge note. Returns nil on 404 (already deleted).
func (c *Client) DeleteKnowledgeNote(ctx context.Context, noteID string) error {
	path := fmt.Sprintf("%s/knowledge/notes/%s", c.orgPath(), noteID)

	_, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}

// ListKnowledgeNotes lists knowledge notes with optional filters. Handles pagination internally.
func (c *Client) ListKnowledgeNotes(ctx context.Context, opts ListKnowledgeNotesOptions) ([]KnowledgeNote, error) {
	var allNotes []KnowledgeNote
	var cursor *string

	for {
		params := url.Values{}
		if cursor != nil {
			params.Set("after", *cursor)
		}
		if opts.First != nil {
			params.Set("first", strconv.Itoa(*opts.First))
		}
		if opts.Search != nil {
			params.Set("search", *opts.Search)
		}
		if opts.FolderPath != nil {
			params.Set("folder_path", *opts.FolderPath)
		}
		if opts.PinnedRepo != nil {
			params.Set("pinned_repo", *opts.PinnedRepo)
		}

		path := fmt.Sprintf("%s/knowledge/notes?%s", c.orgPath(), params.Encode())

		respBody, err := c.doRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page PaginatedResponse
		if err := json.Unmarshal(respBody, &page); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		allNotes = append(allNotes, page.Items...)

		if !page.HasNextPage || page.EndCursor == nil {
			break
		}
		cursor = page.EndCursor
	}

	return allNotes, nil
}
