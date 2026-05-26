package testutil

import (
	"testing"
)

const testOrgID = "org-test"

// RegisterCreateNoteStub registers a WireMock stub for POST /knowledge/notes.
func RegisterCreateNoteStub(t *testing.T, wm *WireMockContainer, responseBody interface{}, statusCode int) {
	t.Helper()
	wm.RegisterStub(t, StubMapping{
		Request: StubRequest{
			Method:  "POST",
			URLPath: "/v3/organizations/" + testOrgID + "/knowledge/notes",
		},
		Response: StubResponse{
			Status:   statusCode,
			JSONBody: responseBody,
			Headers:  map[string]string{"Content-Type": "application/json"},
		},
	})
}

// RegisterGetNoteStub registers a WireMock stub for GET /knowledge/notes/{noteID}.
func RegisterGetNoteStub(t *testing.T, wm *WireMockContainer, noteID string, responseBody interface{}, statusCode int) {
	t.Helper()
	wm.RegisterStub(t, StubMapping{
		Request: StubRequest{
			Method:  "GET",
			URLPath: "/v3/organizations/" + testOrgID + "/knowledge/notes/" + noteID,
		},
		Response: StubResponse{
			Status:   statusCode,
			JSONBody: responseBody,
			Headers:  map[string]string{"Content-Type": "application/json"},
		},
	})
}

// RegisterUpdateNoteStub registers a WireMock stub for PUT /knowledge/notes/{noteID}.
func RegisterUpdateNoteStub(t *testing.T, wm *WireMockContainer, noteID string, responseBody interface{}, statusCode int) {
	t.Helper()
	wm.RegisterStub(t, StubMapping{
		Request: StubRequest{
			Method:  "PUT",
			URLPath: "/v3/organizations/" + testOrgID + "/knowledge/notes/" + noteID,
		},
		Response: StubResponse{
			Status:   statusCode,
			JSONBody: responseBody,
			Headers:  map[string]string{"Content-Type": "application/json"},
		},
	})
}

// RegisterDeleteNoteStub registers a WireMock stub for DELETE /knowledge/notes/{noteID}.
func RegisterDeleteNoteStub(t *testing.T, wm *WireMockContainer, noteID string, responseBody interface{}, statusCode int) {
	t.Helper()
	wm.RegisterStub(t, StubMapping{
		Request: StubRequest{
			Method:  "DELETE",
			URLPath: "/v3/organizations/" + testOrgID + "/knowledge/notes/" + noteID,
		},
		Response: StubResponse{
			Status:   statusCode,
			JSONBody: responseBody,
			Headers:  map[string]string{"Content-Type": "application/json"},
		},
	})
}

// RegisterListNotesStub registers a WireMock stub for GET /knowledge/notes (list).
func RegisterListNotesStub(t *testing.T, wm *WireMockContainer, responseBody interface{}, statusCode int) {
	t.Helper()
	wm.RegisterStub(t, StubMapping{
		Request: StubRequest{
			Method:         "GET",
			URLPathPattern: "/v3/organizations/" + testOrgID + "/knowledge/notes(\\?.*)?",
		},
		Response: StubResponse{
			Status:   statusCode,
			JSONBody: responseBody,
			Headers:  map[string]string{"Content-Type": "application/json"},
		},
	})
}

// NoteResponse is a convenience struct for building WireMock response bodies.
type NoteResponse struct {
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

// DefaultNoteResponse returns a NoteResponse with sensible defaults for testing.
func DefaultNoteResponse(noteID, name, body, trigger string) NoteResponse {
	orgID := testOrgID
	return NoteResponse{
		NoteID:     noteID,
		FolderPath: "/",
		Name:       name,
		Body:       body,
		Trigger:    trigger,
		IsEnabled:  true,
		CreatedAt:  1716000000,
		UpdatedAt:  1716000000,
		AccessType: "org",
		OrgID:      &orgID,
	}
}

// PaginatedNoteResponse wraps notes in a paginated response.
type PaginatedNoteResponse struct {
	Items       []NoteResponse `json:"items"`
	EndCursor   *string        `json:"end_cursor"`
	HasNextPage bool           `json:"has_next_page"`
	Total       *int           `json:"total"`
}
