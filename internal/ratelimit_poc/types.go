package ratelimit_poc

import (
	"context"
	"net/http"
	"time"
)

// Response represents the result of an HTTP request.
type Response struct {
	StatusCode int
	Body       string
	Duration   time.Duration
	Headers    http.Header
	Retries    int
}

// RateLimitClient is the interface all strategies implement.
type RateLimitClient interface {
	Do(ctx context.Context, url string) (*Response, error)
	Name() string
}

// BenchmarkResult holds the results of a benchmark run.
type BenchmarkResult struct {
	Strategy       string        `json:"strategy"`
	Concurrency    int           `json:"concurrency"`
	TotalRequests  int           `json:"total_requests"`
	SuccessCount   int           `json:"success_count"`
	FailureCount   int           `json:"failure_count"`
	Total429s      int           `json:"total_429s"`
	TotalRetries   int           `json:"total_retries"`
	TotalDuration  time.Duration `json:"total_duration"`
	AvgLatency     time.Duration `json:"avg_latency"`
	P99Latency     time.Duration `json:"p99_latency"`
	Throughput     float64       `json:"throughput_rps"`
	Server429s     int           `json:"server_429s"`
	ServerRequests int           `json:"server_requests"`
}
