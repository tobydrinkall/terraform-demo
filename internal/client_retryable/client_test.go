package client_retryable

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func newTestNote() KnowledgeNote {
	return KnowledgeNote{
		NoteID:     "note-test-123",
		FolderPath: "/",
		Name:       "Test Note",
		Body:       "Test body content",
		Trigger:    "test trigger",
		IsEnabled:  true,
		CreatedAt:  1700000000,
		UpdatedAt:  1700000000,
		AccessType: "org",
		OrgID:      "org-test",
	}
}

func newTestServer(handler http.HandlerFunc) (*httptest.Server, *Client) {
	server := httptest.NewServer(handler)
	client := NewClient("test-api-key", "org-test",
		WithBaseURL(server.URL),
		WithMaxRetries(2),
		WithBaseDelay(10*time.Millisecond),
		WithMaxDelay(50*time.Millisecond),
	)
	return server, client
}

func TestCreateKnowledgeNote(t *testing.T) {
	note := newTestNote()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/organizations/org-test/knowledge/notes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("missing or wrong Authorization header")
		}
		if r.Header.Get("User-Agent") != defaultUserAgent {
			t.Errorf("missing or wrong User-Agent header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("missing Content-Type header")
		}

		var req CreateKnowledgeNoteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if req.Name != "Test Note" {
			t.Errorf("expected name 'Test Note', got '%s'", req.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(note)
	})
	defer server.Close()

	result, err := client.CreateKnowledgeNote(context.Background(), CreateKnowledgeNoteRequest{
		Name:    "Test Note",
		Body:    "Test body content",
		Trigger: "test trigger",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NoteID != "note-test-123" {
		t.Errorf("expected note ID 'note-test-123', got '%s'", result.NoteID)
	}
	if result.Name != "Test Note" {
		t.Errorf("expected name 'Test Note', got '%s'", result.Name)
	}
}

func TestGetKnowledgeNote(t *testing.T) {
	note := newTestNote()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/organizations/org-test/knowledge/notes/note-test-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(note)
	})
	defer server.Close()

	result, err := client.GetKnowledgeNote(context.Background(), "note-test-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NoteID != "note-test-123" {
		t.Errorf("expected note ID 'note-test-123', got '%s'", result.NoteID)
	}
}

func TestUpdateKnowledgeNote(t *testing.T) {
	note := newTestNote()
	note.Name = "Updated Note"
	note.Body = "Updated body"
	note.UpdatedAt = 1700000100

	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(note)
	})
	defer server.Close()

	result, err := client.UpdateKnowledgeNote(context.Background(), "note-test-123", UpdateKnowledgeNoteRequest{
		Name:    "Updated Note",
		Body:    "Updated body",
		Trigger: "test trigger",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated Note" {
		t.Errorf("expected name 'Updated Note', got '%s'", result.Name)
	}
}

func TestDeleteKnowledgeNote(t *testing.T) {
	note := newTestNote()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/organizations/org-test/knowledge/notes/note-test-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(note)
	})
	defer server.Close()

	err := client.DeleteKnowledgeNote(context.Background(), "note-test-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListKnowledgeNotes(t *testing.T) {
	listResp := ListKnowledgeNotesResponse{
		Items:       []KnowledgeNote{newTestNote()},
		EndCursor:   "cursor-abc",
		HasNextPage: false,
		Total:       1,
	}

	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("first") != "10" {
			t.Errorf("expected first=10, got %s", r.URL.Query().Get("first"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listResp)
	})
	defer server.Close()

	result, err := client.ListKnowledgeNotes(context.Background(), 10, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
}

func TestRetry429(t *testing.T) {
	var attempts int32
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count <= 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"detail": "rate limited"})
			return
		}
		note := newTestNote()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(note)
	})
	defer server.Close()

	result, err := client.GetKnowledgeNote(context.Background(), "note-test-123")
	if err != nil {
		t.Fatalf("unexpected error after retries: %v", err)
	}
	if result.NoteID != "note-test-123" {
		t.Errorf("expected note ID 'note-test-123', got '%s'", result.NoteID)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestRetry429Exhausted(t *testing.T) {
	var attempts int32
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"detail": "rate limited"})
	})
	defer server.Close()

	_, err := client.GetKnowledgeNote(context.Background(), "note-test-123")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	// retryablehttp: 1 initial + 2 retries = 3 total
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts (1 initial + 2 retries), got %d", atomic.LoadInt32(&attempts))
	}
}

func TestRetry5xx(t *testing.T) {
	var attempts int32
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count <= 1 {
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]string{"detail": "bad gateway"})
			return
		}
		note := newTestNote()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(note)
	})
	defer server.Close()

	result, err := client.GetKnowledgeNote(context.Background(), "note-test-123")
	if err != nil {
		t.Fatalf("unexpected error after retries: %v", err)
	}
	if result.NoteID != "note-test-123" {
		t.Errorf("expected note ID 'note-test-123', got '%s'", result.NoteID)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("expected 2 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestRetry5xxExhausted(t *testing.T) {
	var attempts int32
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"detail": "internal error"})
	})
	defer server.Close()

	_, err := client.GetKnowledgeNote(context.Background(), "note-test-123")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestErrorParsing404(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"detail": "Note not found"})
	})
	defer server.Close()

	_, err := client.GetKnowledgeNote(context.Background(), "note-nonexistent")
	if err == nil {
		t.Fatal("expected error for 404")
	}

	var notFoundErr *NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestErrorParsing403(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"detail": "Unauthorized"})
	})
	defer server.Close()

	_, err := client.GetKnowledgeNote(context.Background(), "note-test-123")
	if err == nil {
		t.Fatal("expected error for 403")
	}

	var unauthErr *UnauthorizedError
	if !errors.As(err, &unauthErr) {
		t.Errorf("expected UnauthorizedError, got %T: %v", err, err)
	}
}

func TestErrorParsing422(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		resp := map[string]interface{}{
			"detail": []map[string]interface{}{
				{"type": "missing", "loc": []string{"body", "name"}, "msg": "Field required"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	_, err := client.CreateKnowledgeNote(context.Background(), CreateKnowledgeNoteRequest{})
	if err == nil {
		t.Fatal("expected error for 422")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("expected APIError, got %T: %v", err, err)
	}
}

func TestContextCancellation(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		json.NewEncoder(w).Encode(newTestNote())
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.GetKnowledgeNote(ctx, "note-test-123")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Logf("got error (acceptable): %v", err)
	}
}

func TestContextCancellationDuringRetry(t *testing.T) {
	var attempts int32
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"detail": "rate limited"})
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.GetKnowledgeNote(ctx, "note-test-123")
	if err == nil {
		t.Fatal("expected error from cancelled context during retry")
	}
}

func TestUserAgentHeader(t *testing.T) {
	customUA := "my-custom-agent/1.0"
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != customUA {
			t.Errorf("expected User-Agent '%s', got '%s'", customUA, r.Header.Get("User-Agent"))
		}
		json.NewEncoder(w).Encode(newTestNote())
	})
	defer server.Close()
	client.userAgent = customUA

	_, err := client.GetKnowledgeNote(context.Background(), "note-test-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
