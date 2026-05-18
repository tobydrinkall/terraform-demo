package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateKnowledgeNote_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v3/organizations/org-123/knowledge/notes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req KnowledgeNoteCreateRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Name != "test-note" {
			t.Errorf("expected name 'test-note', got %q", req.Name)
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(KnowledgeNote{
			NoteID:     "note-abc123",
			Name:       req.Name,
			Body:       req.Body,
			Trigger:    req.Trigger,
			IsEnabled:  true,
			AccessType: "org",
			FolderPath: "/",
			CreatedAt:  1716000000,
			UpdatedAt:  1716000000,
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "org-123", "cog_test_key")
	note, err := c.CreateKnowledgeNote(context.Background(), KnowledgeNoteCreateRequest{
		Name:    "test-note",
		Body:    "test body",
		Trigger: "test trigger",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if note.NoteID != "note-abc123" {
		t.Errorf("expected note ID 'note-abc123', got %q", note.NoteID)
	}
	if note.Name != "test-note" {
		t.Errorf("expected name 'test-note', got %q", note.Name)
	}
}

func TestGetKnowledgeNote_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v3/organizations/org-123/knowledge/notes/note-abc123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(KnowledgeNote{
			NoteID:     "note-abc123",
			Name:       "test-note",
			Body:       "test body",
			Trigger:    "test trigger",
			IsEnabled:  true,
			AccessType: "org",
			FolderPath: "/",
			CreatedAt:  1716000000,
			UpdatedAt:  1716000000,
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "org-123", "cog_test_key")
	note, err := c.GetKnowledgeNote(context.Background(), "note-abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if note.NoteID != "note-abc123" {
		t.Errorf("expected note ID 'note-abc123', got %q", note.NoteID)
	}
}

func TestGetKnowledgeNote_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"detail": "Not found"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, "org-123", "cog_test_key")
	_, err := c.GetKnowledgeNote(context.Background(), "note-nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Error("expected IsNotFound to return true")
	}
}

func TestUpdateKnowledgeNote_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/v3/organizations/org-123/knowledge/notes/note-abc123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req KnowledgeNoteCreateRequest
		json.NewDecoder(r.Body).Decode(&req)

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(KnowledgeNote{
			NoteID:     "note-abc123",
			Name:       req.Name,
			Body:       req.Body,
			Trigger:    req.Trigger,
			IsEnabled:  true,
			AccessType: "org",
			FolderPath: "/",
			CreatedAt:  1716000000,
			UpdatedAt:  1716000001,
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "org-123", "cog_test_key")
	note, err := c.UpdateKnowledgeNote(context.Background(), "note-abc123", KnowledgeNoteCreateRequest{
		Name:    "updated-note",
		Body:    "updated body",
		Trigger: "updated trigger",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if note.Name != "updated-note" {
		t.Errorf("expected name 'updated-note', got %q", note.Name)
	}
	if note.UpdatedAt != 1716000001 {
		t.Errorf("expected updated_at 1716000001, got %d", note.UpdatedAt)
	}
}

func TestDeleteKnowledgeNote_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(KnowledgeNote{NoteID: "note-abc123"})
	}))
	defer server.Close()

	c := NewClient(server.URL, "org-123", "cog_test_key")
	err := c.DeleteKnowledgeNote(context.Background(), "note-abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteKnowledgeNote_AlreadyDeleted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"detail": "Not found"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, "org-123", "cog_test_key")
	err := c.DeleteKnowledgeNote(context.Background(), "note-abc123")
	if err != nil {
		t.Fatalf("expected no error on 404 delete, got: %v", err)
	}
}

func TestListKnowledgeNotes_SinglePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(PaginatedResponse{
			Items: []KnowledgeNote{
				{NoteID: "note-1", Name: "Note 1"},
				{NoteID: "note-2", Name: "Note 2"},
			},
			HasNextPage: false,
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "org-123", "cog_test_key")
	notes, err := c.ListKnowledgeNotes(context.Background(), ListKnowledgeNotesOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notes))
	}
}

func TestListKnowledgeNotes_Pagination(t *testing.T) {
	page := 0
	cursor := "cursor-page2"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.WriteHeader(200)
		if page == 1 {
			json.NewEncoder(w).Encode(PaginatedResponse{
				Items:       []KnowledgeNote{{NoteID: "note-1"}},
				HasNextPage: true,
				EndCursor:   &cursor,
			})
		} else {
			if r.URL.Query().Get("after") != "cursor-page2" {
				t.Errorf("expected cursor 'cursor-page2', got %q", r.URL.Query().Get("after"))
			}
			json.NewEncoder(w).Encode(PaginatedResponse{
				Items:       []KnowledgeNote{{NoteID: "note-2"}},
				HasNextPage: false,
			})
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "org-123", "cog_test_key")
	notes, err := c.ListKnowledgeNotes(context.Background(), ListKnowledgeNotesOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("expected 2 notes across pages, got %d", len(notes))
	}
	if page != 2 {
		t.Errorf("expected 2 pages fetched, got %d", page)
	}
}

func TestCreateKnowledgeNote_WithPinnedRepo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req KnowledgeNoteCreateRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.PinnedRepo == nil || *req.PinnedRepo != "myorg/myrepo" {
			t.Errorf("expected pinned_repo 'myorg/myrepo', got %v", req.PinnedRepo)
		}

		pinnedRepo := "myorg/myrepo"
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(KnowledgeNote{
			NoteID:     "note-abc123",
			Name:       req.Name,
			PinnedRepo: &pinnedRepo,
			IsEnabled:  true,
			AccessType: "org",
			FolderPath: "/",
		})
	}))
	defer server.Close()

	pinnedRepo := "myorg/myrepo"
	c := NewClient(server.URL, "org-123", "cog_test_key")
	note, err := c.CreateKnowledgeNote(context.Background(), KnowledgeNoteCreateRequest{
		Name:       "test-note",
		Body:       "body",
		Trigger:    "trigger",
		PinnedRepo: &pinnedRepo,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if note.PinnedRepo == nil || *note.PinnedRepo != "myorg/myrepo" {
		t.Errorf("expected pinned_repo 'myorg/myrepo', got %v", note.PinnedRepo)
	}
}
