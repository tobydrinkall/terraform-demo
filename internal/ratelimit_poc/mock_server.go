package ratelimit_poc

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// MockRateLimitServer simulates an API server that enforces rate limits.
// After maxRequestsPerWindow requests within windowDuration, it returns 429.
type MockRateLimitServer struct {
	mu                   sync.Mutex
	requestCount         int
	windowStart          time.Time
	maxRequestsPerWindow int
	windowDuration       time.Duration
	total429s            int
	totalRequests        int
	server               *httptest.Server
}

// NewMockRateLimitServer creates a mock server that allows maxReqs requests per window.
func NewMockRateLimitServer(maxReqs int, window time.Duration) *MockRateLimitServer {
	m := &MockRateLimitServer{
		maxRequestsPerWindow: maxReqs,
		windowDuration:       window,
		windowStart:          time.Now(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/test", m.handleRequest)
	m.server = httptest.NewServer(mux)
	return m
}

func (m *MockRateLimitServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests++
	now := time.Now()

	if now.Sub(m.windowStart) >= m.windowDuration {
		m.requestCount = 0
		m.windowStart = now
	}

	m.requestCount++
	if m.requestCount > m.maxRequestsPerWindow {
		m.total429s++
		retryAfter := m.windowDuration - now.Sub(m.windowStart)
		if retryAfter < 100*time.Millisecond {
			retryAfter = 100 * time.Millisecond
		}
		w.Header().Set("Retry-After", fmt.Sprintf("%.1f", retryAfter.Seconds()))
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", m.maxRequestsPerWindow))
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprintf(w, `{"error":"rate_limit_exceeded","retry_after":%.1f}`, retryAfter.Seconds())
		return
	}

	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", m.maxRequestsPerWindow))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", m.maxRequestsPerWindow-m.requestCount))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","request_num":%d}`, m.requestCount)
}

func (m *MockRateLimitServer) URL() string {
	return m.server.URL
}

func (m *MockRateLimitServer) Close() {
	m.server.Close()
}

func (m *MockRateLimitServer) Stats() (totalRequests, total429s int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.totalRequests, m.total429s
}

func (m *MockRateLimitServer) ResetStats() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalRequests = 0
	m.total429s = 0
	m.requestCount = 0
	m.windowStart = time.Now()
}
