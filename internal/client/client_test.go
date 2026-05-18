package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_DefaultBaseURL(t *testing.T) {
	c := NewClient("", "org-123", "cog_test")
	if c.BaseURL != defaultBaseURL {
		t.Errorf("expected default base URL %q, got %q", defaultBaseURL, c.BaseURL)
	}
}

func TestNewClient_CustomBaseURL(t *testing.T) {
	c := NewClient("https://custom.api.com", "org-123", "cog_test")
	if c.BaseURL != "https://custom.api.com" {
		t.Errorf("expected custom base URL, got %q", c.BaseURL)
	}
}

func TestDoRequest_AuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer cog_test_key" {
			t.Errorf("expected Bearer cog_test_key, got %q", auth)
		}
		ua := r.Header.Get("User-Agent")
		if ua != defaultUserAgent {
			t.Errorf("expected User-Agent %q, got %q", defaultUserAgent, ua)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, "org-123", "cog_test_key")
	_, err := c.doRequest(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoRequest_4xxError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"detail": []map[string]interface{}{
				{"loc": []string{"body", "name"}, "msg": "field required", "type": "value_error.missing"},
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "org-123", "cog_test_key")
	_, err := c.doRequest(context.Background(), "POST", "/test", map[string]string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 422 {
		t.Errorf("expected status 422, got %d", apiErr.StatusCode)
	}
}

func TestDoRequest_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"detail": "Not found"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, "org-123", "cog_test_key")
	_, err := c.doRequest(context.Background(), "GET", "/test", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Error("expected IsNotFound to return true")
	}
}

func TestDoRequest_RateLimitRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(429)
			w.Write([]byte(`{"message": "rate limited"}`))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, "org-123", "cog_test_key")
	_, err := c.doRequest(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error after retries: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestDoRequest_RateLimitRetry_PreservesBody(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Errorf("attempt %d: received empty body, expected JSON payload", attempts)
		}
		if r.ContentLength == 0 && r.Method == "POST" {
			t.Errorf("attempt %d: ContentLength is 0, expected non-zero", attempts)
		}
		if attempts <= 1 {
			w.WriteHeader(429)
			w.Write([]byte(`{"message": "rate limited"}`))
			return
		}
		w.WriteHeader(200)
		w.Write(body)
	}))
	defer server.Close()

	c := NewClient(server.URL, "org-123", "cog_test_key")
	payload := map[string]string{"name": "test", "body": "content"}
	resp, err := c.doRequest(context.Background(), "POST", "/test", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) == 0 {
		t.Error("expected non-empty response body")
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}
