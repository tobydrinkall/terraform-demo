package sessions_poc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// SessionClient handles Devin API v3 session operations.
type SessionClient struct {
	APIKey  string
	BaseURL string // e.g. "https://api.devin.ai/v3"
	OrgID   string
	HTTP    *http.Client
}

// Session represents a Devin session as returned by the API.
type Session struct {
	SessionID       string   `json:"session_id"`
	URL             string   `json:"url"`
	Status          string   `json:"status"`
	StatusDetail    *string  `json:"status_detail"`
	Title           *string  `json:"title"`
	Tags            []string `json:"tags"`
	PlaybookID      *string  `json:"playbook_id"`
	UserID          string   `json:"user_id"`
	OrgID           string   `json:"org_id"`
	CreatedAt       int64    `json:"created_at"`
	UpdatedAt       int64    `json:"updated_at"`
	IsArchived      bool     `json:"is_archived"`
	ACUsConsumed    float64  `json:"acus_consumed"`
	Origin          string   `json:"origin"`
	ServiceUserID   *string  `json:"service_user_id"`
	ParentSessionID *string  `json:"parent_session_id"`
}

// CreateSessionRequest is the payload for creating a new session.
type CreateSessionRequest struct {
	Prompt         string  `json:"prompt"`
	IdempotencyKey string  `json:"idempotency_key,omitempty"`
	CreateAsUserID string  `json:"create_as_user_id,omitempty"`
	PlaybookID     string  `json:"playbook_id,omitempty"`
}

// SessionList is the response from listing sessions.
type SessionList struct {
	Items      []Session `json:"items"`
	NextCursor *string   `json:"next_cursor"`
}

func NewSessionClient(apiKey, baseURL, orgID string) *SessionClient {
	return &SessionClient{
		APIKey:  apiKey,
		BaseURL: baseURL,
		OrgID:   orgID,
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *SessionClient) sessionsURL() string {
	return fmt.Sprintf("%s/organizations/%s/sessions", c.BaseURL, c.OrgID)
}

func (c *SessionClient) sessionURL(sessionID string) string {
	devinID := "devin-" + sessionID
	return fmt.Sprintf("%s/organizations/%s/sessions/%s", c.BaseURL, c.OrgID, devinID)
}

func (c *SessionClient) doRequest(method, url string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// CreateSession creates a new Devin session.
func (c *SessionClient) CreateSession(req CreateSessionRequest) (*Session, error) {
	body, status, err := c.doRequest("POST", c.sessionsURL(), req)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("API returned status %d: %s", status, string(body))
	}

	var session Session
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &session, nil
}

// GetSession retrieves a session by its ID.
func (c *SessionClient) GetSession(sessionID string) (*Session, error) {
	body, status, err := c.doRequest("GET", c.sessionURL(sessionID), nil)
	if err != nil {
		return nil, err
	}
	if status == 404 {
		return nil, nil
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("API returned status %d: %s", status, string(body))
	}

	var session Session
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &session, nil
}

// TerminateSession sends a DELETE to terminate the session.
func (c *SessionClient) TerminateSession(sessionID string) (*Session, error) {
	body, status, err := c.doRequest("DELETE", c.sessionURL(sessionID), nil)
	if err != nil {
		return nil, err
	}
	if status == 404 {
		return nil, nil
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("API returned status %d: %s", status, string(body))
	}

	var session Session
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &session, nil
}

// ListSessions retrieves sessions with optional filters.
func (c *SessionClient) ListSessions(limit int, status string, cursor string) (*SessionList, error) {
	u, err := url.Parse(c.sessionsURL())
	if err != nil {
		return nil, err
	}

	q := u.Query()
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if status != "" {
		q.Set("status", status)
	}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	u.RawQuery = q.Encode()

	body, httpStatus, err := c.doRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	if httpStatus < 200 || httpStatus >= 300 {
		return nil, fmt.Errorf("API returned status %d: %s", httpStatus, string(body))
	}

	var list SessionList
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &list, nil
}

// IsTerminalStatus returns true if the session has reached a terminal state.
func IsTerminalStatus(status string) bool {
	switch status {
	case "exit", "stopped", "failed", "terminated":
		return true
	}
	return false
}

// WaitForCompletion polls the session until it reaches a terminal state or timeout.
func (c *SessionClient) WaitForCompletion(sessionID string, timeout time.Duration, pollInterval time.Duration) (*Session, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		session, err := c.GetSession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("polling session status: %w", err)
		}
		if session == nil {
			return nil, fmt.Errorf("session %s not found during polling", sessionID)
		}

		if IsTerminalStatus(session.Status) {
			return session, nil
		}

		// Check if status_detail indicates completion while status is still "running"
		if session.Status == "running" && session.StatusDetail != nil && *session.StatusDetail == "finished" {
			return session, nil
		}

		remaining := time.Until(deadline)
		if remaining < pollInterval {
			time.Sleep(remaining)
		} else {
			time.Sleep(pollInterval)
		}
	}

	// Final check before returning timeout
	session, err := c.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("final poll: %w", err)
	}
	return session, fmt.Errorf("timeout waiting for session %s to complete (last status: %s)", sessionID, session.Status)
}
