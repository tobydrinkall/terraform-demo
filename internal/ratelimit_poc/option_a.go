package ratelimit_poc

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

// ClientSideLimiter implements Option A: proactive token bucket rate limiting.
type ClientSideLimiter struct {
	client  *http.Client
	limiter *rate.Limiter
}

// NewClientSideLimiter creates a new client with a token bucket limiter.
// rps is requests per second, burst is the maximum burst size.
func NewClientSideLimiter(rps float64, burst int) *ClientSideLimiter {
	return &ClientSideLimiter{
		client:  &http.Client{Timeout: 30 * time.Second},
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
	}
}

func (c *ClientSideLimiter) Do(ctx context.Context, url string) (*Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       string(body),
		Duration:   time.Since(start),
		Headers:    resp.Header,
	}, nil
}

func (c *ClientSideLimiter) Name() string {
	return "Option A: Client-Side Rate Limiter"
}
