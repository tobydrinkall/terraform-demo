package ratelimit_poc

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

// CombinedClient implements Option C: client-side limiter + 429 retry fallback.
type CombinedClient struct {
	client     *http.Client
	limiter    *rate.Limiter
	maxRetries int
}

// NewCombinedClient creates a client with both proactive limiting and reactive retries.
func NewCombinedClient(rps float64, burst int, maxRetries int) *CombinedClient {
	return &CombinedClient{
		client:     &http.Client{Timeout: 30 * time.Second},
		limiter:    rate.NewLimiter(rate.Limit(rps), burst),
		maxRetries: maxRetries,
	}
}

func (c *CombinedClient) Do(ctx context.Context, url string) (*Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}

	var lastErr error
	totalRetries := 0

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := c.limiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limiter wait on retry: %w", err)
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		start := time.Now()
		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("do request (attempt %d): %w", attempt+1, err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusTooManyRequests {
			return &Response{
				StatusCode: resp.StatusCode,
				Body:       string(body),
				Duration:   time.Since(start),
				Headers:    resp.Header,
				Retries:    totalRetries,
			}, nil
		}

		totalRetries++
		sleepDuration := c.calculateBackoff(attempt, resp.Header)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(sleepDuration):
		}
	}

	return nil, fmt.Errorf("max retries (%d) exceeded: %w", c.maxRetries, lastErr)
}

func (c *CombinedClient) calculateBackoff(attempt int, headers http.Header) time.Duration {
	retryAfter := headers.Get("Retry-After")
	if retryAfter != "" {
		if seconds, err := strconv.ParseFloat(retryAfter, 64); err == nil {
			jitter := time.Duration(rand.Float64()*200) * time.Millisecond
			return time.Duration(seconds*1000)*time.Millisecond + jitter
		}
	}

	baseDelay := 100 * time.Millisecond
	backoff := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
	maxDelay := 10 * time.Second
	if backoff > maxDelay {
		backoff = maxDelay
	}

	jitter := time.Duration(rand.Float64()*float64(backoff)*0.3) * time.Nanosecond
	return backoff + jitter
}

func (c *CombinedClient) Name() string {
	return "Option C: Combined (Limiter + Retry)"
}
