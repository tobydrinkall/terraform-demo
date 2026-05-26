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

func doRealRequest(ctx context.Context, url, apiKey string) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
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

// RealAPIClientA wraps Option A for real API calls with auth.
type RealAPIClientA struct {
	limiter *rate.Limiter
	baseURL string
	apiKey  string
}

func NewRealAPIClientA(rps float64, burst int, baseURL, apiKey string) *RealAPIClientA {
	return &RealAPIClientA{
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

func (c *RealAPIClientA) Do(ctx context.Context, url string) (*Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}
	return doRealRequest(ctx, url, c.apiKey)
}

func (c *RealAPIClientA) Name() string {
	return "Option A: Client-Side Rate Limiter (Real API)"
}

// RealAPIClientB wraps Option B for real API calls with auth.
type RealAPIClientB struct {
	maxRetries int
	baseURL    string
	apiKey     string
}

func NewRealAPIClientB(maxRetries int, baseURL, apiKey string) *RealAPIClientB {
	return &RealAPIClientB{
		maxRetries: maxRetries,
		baseURL:    baseURL,
		apiKey:     apiKey,
	}
}

func (c *RealAPIClientB) Do(ctx context.Context, url string) (*Response, error) {
	var lastErr error
	totalRetries := 0

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		resp, err := doRealRequest(ctx, url, c.apiKey)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			resp.Retries = totalRetries
			return resp, nil
		}

		totalRetries++
		sleepDuration := calculateRealBackoff(attempt, resp.Headers)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(sleepDuration):
		}
	}

	return nil, fmt.Errorf("max retries (%d) exceeded: %w", c.maxRetries, lastErr)
}

func (c *RealAPIClientB) Name() string {
	return "Option B: Retry on 429 (Real API)"
}

// RealAPIClientC wraps Option C for real API calls with auth.
type RealAPIClientC struct {
	limiter    *rate.Limiter
	maxRetries int
	baseURL    string
	apiKey     string
}

func NewRealAPIClientC(rps float64, burst int, maxRetries int, baseURL, apiKey string) *RealAPIClientC {
	return &RealAPIClientC{
		limiter:    rate.NewLimiter(rate.Limit(rps), burst),
		maxRetries: maxRetries,
		baseURL:    baseURL,
		apiKey:     apiKey,
	}
}

func (c *RealAPIClientC) Do(ctx context.Context, url string) (*Response, error) {
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

		resp, err := doRealRequest(ctx, url, c.apiKey)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			resp.Retries = totalRetries
			return resp, nil
		}

		totalRetries++
		sleepDuration := calculateRealBackoff(attempt, resp.Headers)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(sleepDuration):
		}
	}

	return nil, fmt.Errorf("max retries (%d) exceeded: %w", c.maxRetries, lastErr)
}

func (c *RealAPIClientC) Name() string {
	return "Option C: Combined (Real API)"
}

func calculateRealBackoff(attempt int, headers http.Header) time.Duration {
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
