package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tobydrinkall/terraform-provider-devin/internal/client"
)

func TestCreateNote(t *testing.T) {
	t.Parallel()

	expected := &client.KnowledgeNoteResponse{
		NoteID:     "note-123",
		Name:       "Test Note",
		Body:       "Test body content",
		Trigger:    "When testing",
		IsEnabled:  true,
		AccessType: "org",
		CreatedAt:  1700000000,
		UpdatedAt:  1700000000,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v3/organizations/org-abc/knowledge/notes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		var req client.KnowledgeNoteCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %s", err)
		}
		if req.Name != "Test Note" {
			t.Errorf("expected name 'Test Note', got '%s'", req.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	c := client.New("test-key", client.WithBaseURL(server.URL))
	note, err := c.CreateNote(context.Background(), "org-abc", &client.KnowledgeNoteCreateRequest{
		Name:    "Test Note",
		Body:    "Test body content",
		Trigger: "When testing",
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if note.NoteID != "note-123" {
		t.Errorf("expected note ID 'note-123', got '%s'", note.NoteID)
	}
}

func TestGetNote(t *testing.T) {
	t.Parallel()

	expected := &client.KnowledgeNoteResponse{
		NoteID:     "note-456",
		Name:       "Retrieved Note",
		Body:       "Some content",
		Trigger:    "Always",
		IsEnabled:  true,
		AccessType: "org",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v3/organizations/org-abc/knowledge/notes/note-456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	c := client.New("test-key", client.WithBaseURL(server.URL))
	note, err := c.GetNote(context.Background(), "org-abc", "note-456")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if note.Name != "Retrieved Note" {
		t.Errorf("expected name 'Retrieved Note', got '%s'", note.Name)
	}
}

func TestNotFoundError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"detail": "not found"})
	}))
	defer server.Close()

	c := client.New("test-key", client.WithBaseURL(server.URL))
	_, err := c.GetNote(context.Background(), "org-abc", "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected not found error, got: %v", err)
	}
}

func TestDeleteNote(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&client.KnowledgeNoteResponse{NoteID: "note-789"})
	}))
	defer server.Close()

	c := client.New("test-key", client.WithBaseURL(server.URL))
	note, err := c.DeleteNote(context.Background(), "org-abc", "note-789")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if note.NoteID != "note-789" {
		t.Errorf("expected note ID 'note-789', got '%s'", note.NoteID)
	}
}
