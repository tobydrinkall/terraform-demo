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

const (
	defaultBaseURL   = "https://api.devin.ai"
	defaultUserAgent = "terraform-provider-devin/0.1.0"
	maxRetries       = 3
)

// Client is an HTTP client for the Devin API v3.
type Client struct {
	BaseURL    string
	OrgID      string
	APIKey     string
	HTTPClient *http.Client
	UserAgent  string
}

// NewClient creates a new Devin API client.
func NewClient(baseURL, orgID, apiKey string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		BaseURL: baseURL,
		OrgID:   orgID,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		UserAgent: defaultUserAgent,
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := fmt.Sprintf("%s%s", c.BaseURL, path)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		// Reset the body reader for retries so ContentLength is computed correctly
		if reqBody != nil {
			if seeker, ok := reqBody.(io.Seeker); ok {
				seeker.Seek(0, io.SeekStart)
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
		req.Header.Set("User-Agent", c.UserAgent)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		if resp.StatusCode == 429 {
			lastErr = &APIError{StatusCode: 429, Message: "rate limited"}
			continue
		}

		if resp.StatusCode >= 400 {
			apiErr := &APIError{
				StatusCode: resp.StatusCode,
				Message:    string(respBody),
			}
			// Try to parse structured validation errors
			json.Unmarshal(respBody, apiErr)
			return nil, apiErr
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

// orgPath returns the base path for organization-scoped endpoints.
func (c *Client) orgPath() string {
	return fmt.Sprintf("/v3/organizations/%s", c.OrgID)
}
