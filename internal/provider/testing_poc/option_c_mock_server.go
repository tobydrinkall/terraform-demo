package testing_poc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// MockDevinAPI is an in-memory mock of the Devin API knowledge notes endpoints.
type MockDevinAPI struct {
	mu       sync.RWMutex
	notes    map[string]mockNote
	nextSeq  int
	orgID    string
}

type mockNote struct {
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

type createNoteRequest struct {
	Name       string  `json:"name"`
	Body       string  `json:"body"`
	Trigger    string  `json:"trigger"`
	PinnedRepo *string `json:"pinned_repo"`
}

type paginatedResponse struct {
	Items       []mockNote `json:"items"`
	EndCursor   *string    `json:"end_cursor"`
	HasNextPage bool       `json:"has_next_page"`
	Total       *int       `json:"total"`
}

// NewMockDevinAPI creates a new mock API with an empty data store.
func NewMockDevinAPI(orgID string) *MockDevinAPI {
	return &MockDevinAPI{
		notes: make(map[string]mockNote),
		orgID: orgID,
	}
}

// Handler returns an http.Handler that implements the knowledge notes API.
func (m *MockDevinAPI) Handler() http.Handler {
	mux := http.NewServeMux()

	basePath := fmt.Sprintf("/v3/organizations/%s/knowledge/notes", m.orgID)

	mux.HandleFunc(basePath+"/", func(w http.ResponseWriter, r *http.Request) {
		// Extract note ID from path: /v3/organizations/{org}/knowledge/notes/{noteID}
		noteID := strings.TrimPrefix(r.URL.Path, basePath+"/")
		if noteID == "" {
			http.Error(w, `{"detail":"Not Found"}`, http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			m.handleGetNote(w, noteID)
		case http.MethodPut:
			m.handleUpdateNote(w, r, noteID)
		case http.MethodDelete:
			m.handleDeleteNote(w, noteID)
		default:
			http.Error(w, `{"detail":"Method Not Allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc(basePath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			m.handleCreateNote(w, r)
		case http.MethodGet:
			m.handleListNotes(w, r)
		default:
			http.Error(w, `{"detail":"Method Not Allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	return mux
}

func (m *MockDevinAPI) handleCreateNote(w http.ResponseWriter, r *http.Request) {
	var req createNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid JSON"})
		return
	}
	if req.Name == "" || req.Body == "" || req.Trigger == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"detail": []map[string]interface{}{
				{"loc": []string{"body"}, "msg": "field required", "type": "value_error.missing"},
			},
		})
		return
	}

	m.mu.Lock()
	m.nextSeq++
	noteID := fmt.Sprintf("note-mock-%06d", m.nextSeq)
	now := time.Now().Unix()
	note := mockNote{
		NoteID:     noteID,
		FolderPath: "/",
		Name:       req.Name,
		Body:       req.Body,
		Trigger:    req.Trigger,
		IsEnabled:  true,
		CreatedAt:  now,
		UpdatedAt:  now,
		AccessType: "org",
		OrgID:      &m.orgID,
		PinnedRepo: req.PinnedRepo,
	}
	m.notes[noteID] = note
	m.mu.Unlock()

	writeJSON(w, http.StatusOK, note)
}

func (m *MockDevinAPI) handleGetNote(w http.ResponseWriter, noteID string) {
	m.mu.RLock()
	note, ok := m.notes[noteID]
	m.mu.RUnlock()

	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Not Found"})
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func (m *MockDevinAPI) handleUpdateNote(w http.ResponseWriter, r *http.Request, noteID string) {
	var req createNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid JSON"})
		return
	}

	m.mu.Lock()
	note, ok := m.notes[noteID]
	if !ok {
		m.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Not Found"})
		return
	}
	note.Name = req.Name
	note.Body = req.Body
	note.Trigger = req.Trigger
	note.PinnedRepo = req.PinnedRepo
	note.UpdatedAt = time.Now().Unix()
	m.notes[noteID] = note
	m.mu.Unlock()

	writeJSON(w, http.StatusOK, note)
}

func (m *MockDevinAPI) handleDeleteNote(w http.ResponseWriter, noteID string) {
	m.mu.Lock()
	note, ok := m.notes[noteID]
	delete(m.notes, noteID)
	m.mu.Unlock()

	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Not Found"})
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func (m *MockDevinAPI) handleListNotes(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	var items []mockNote
	for _, note := range m.notes {
		items = append(items, note)
	}
	total := len(items)
	m.mu.RUnlock()

	resp := paginatedResponse{
		Items:       items,
		HasNextPage: false,
		Total:       &total,
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
