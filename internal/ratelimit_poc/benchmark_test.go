package ratelimit_poc

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

const (
	mockMaxReqsPerWindow = 10
	mockWindow           = 1 * time.Second
	requestsPerTest      = 50
)

func TestMockServer(t *testing.T) {
	server := NewMockRateLimitServer(mockMaxReqsPerWindow, mockWindow)
	defer server.Close()

	client := NewRetryOn429Client(5)
	ctx := context.Background()

	for i := 0; i < 15; i++ {
		resp, err := client.Do(ctx, server.URL()+"/api/test")
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		t.Logf("Request %d: status=%d retries=%d", i, resp.StatusCode, resp.Retries)
	}

	totalReqs, total429s := server.Stats()
	t.Logf("Server stats: total=%d, 429s=%d", totalReqs, total429s)
}

func TestOptionA_ClientSideLimiter(t *testing.T) {
	server := NewMockRateLimitServer(mockMaxReqsPerWindow, mockWindow)
	defer server.Close()

	client := NewClientSideLimiter(float64(mockMaxReqsPerWindow), 1)
	ctx := context.Background()

	concurrencyLevels := []int{1, 5, 10, 20}
	for _, c := range concurrencyLevels {
		server.ResetStats()
		result := RunBenchmark(ctx, client, server.URL(), c, requestsPerTest)
		serverReqs, server429s := server.Stats()
		result.ServerRequests = serverReqs
		result.Server429s = server429s
		t.Logf("Option A [concurrency=%d]: success=%d, 429s=%d, retries=%d, duration=%s, throughput=%.1f rps, server429s=%d",
			c, result.SuccessCount, result.Total429s, result.TotalRetries,
			result.TotalDuration.Round(time.Millisecond), result.Throughput, server429s)
	}
}

func TestOptionB_RetryOn429(t *testing.T) {
	server := NewMockRateLimitServer(mockMaxReqsPerWindow, mockWindow)
	defer server.Close()

	client := NewRetryOn429Client(5)
	ctx := context.Background()

	concurrencyLevels := []int{1, 5, 10, 20}
	for _, c := range concurrencyLevels {
		server.ResetStats()
		result := RunBenchmark(ctx, client, server.URL(), c, requestsPerTest)
		serverReqs, server429s := server.Stats()
		result.ServerRequests = serverReqs
		result.Server429s = server429s
		t.Logf("Option B [concurrency=%d]: success=%d, 429s=%d, retries=%d, duration=%s, throughput=%.1f rps, server429s=%d",
			c, result.SuccessCount, result.Total429s, result.TotalRetries,
			result.TotalDuration.Round(time.Millisecond), result.Throughput, server429s)
	}
}

func TestOptionC_Combined(t *testing.T) {
	server := NewMockRateLimitServer(mockMaxReqsPerWindow, mockWindow)
	defer server.Close()

	client := NewCombinedClient(float64(mockMaxReqsPerWindow), 1, 5)
	ctx := context.Background()

	concurrencyLevels := []int{1, 5, 10, 20}
	for _, c := range concurrencyLevels {
		server.ResetStats()
		result := RunBenchmark(ctx, client, server.URL(), c, requestsPerTest)
		serverReqs, server429s := server.Stats()
		result.ServerRequests = serverReqs
		result.Server429s = server429s
		t.Logf("Option C [concurrency=%d]: success=%d, 429s=%d, retries=%d, duration=%s, throughput=%.1f rps, server429s=%d",
			c, result.SuccessCount, result.Total429s, result.TotalRetries,
			result.TotalDuration.Round(time.Millisecond), result.Throughput, server429s)
	}
}

func TestFullBenchmarkSuite(t *testing.T) {
	server := NewMockRateLimitServer(mockMaxReqsPerWindow, mockWindow)
	defer server.Close()

	ctx := context.Background()
	concurrencyLevels := []int{1, 5, 10, 20}

	clients := []RateLimitClient{
		NewClientSideLimiter(float64(mockMaxReqsPerWindow), 1),
		NewRetryOn429Client(5),
		NewCombinedClient(float64(mockMaxReqsPerWindow), 1, 5),
	}

	var allResults []BenchmarkResult

	for _, client := range clients {
		for _, c := range concurrencyLevels {
			server.ResetStats()
			result := RunBenchmark(ctx, client, server.URL(), c, requestsPerTest)
			serverReqs, server429s := server.Stats()
			result.ServerRequests = serverReqs
			result.Server429s = server429s
			allResults = append(allResults, result)
		}
	}

	t.Log("\n" + FormatResults(allResults))
}

func TestRealDevinAPI(t *testing.T) {
	apiKey := os.Getenv("DEVIN_API_KEY_TERRAFORM")
	if apiKey == "" {
		t.Skip("DEVIN_API_KEY_TERRAFORM not set, skipping real API test")
	}

	baseURL := "https://api.devin.ai"
	orgID := "org-8d158f07ee0f4678a467078b323880a4"
	apiEndpoint := baseURL + "/v3/organizations/" + orgID + "/knowledge/notes?first=1"
	ctx := context.Background()

	t.Log("=== Real Devin API Rate Limit Observation ===")

	t.Run("burst_requests", func(t *testing.T) {
		var results []realAPIResult
		for i := 0; i < 20; i++ {
			start := time.Now()
			resp, err := makeRealAPIRequest(ctx, apiEndpoint, apiKey)
			elapsed := time.Since(start)
			r := realAPIResult{
				requestNum: i + 1,
				duration:   elapsed,
			}
			if err != nil {
				r.err = err
				t.Logf("Request %d: error=%v duration=%s", i+1, err, elapsed.Round(time.Millisecond))
			} else {
				r.statusCode = resp.StatusCode
				r.retryAfter = resp.Headers.Get("Retry-After")
				r.rateLimitRemaining = resp.Headers.Get("X-RateLimit-Remaining")
				t.Logf("Request %d: status=%d duration=%s remaining=%s retry-after=%s",
					i+1, resp.StatusCode, elapsed.Round(time.Millisecond),
					r.rateLimitRemaining, r.retryAfter)
			}
			results = append(results, r)
		}

		var count429 int
		for _, r := range results {
			if r.statusCode == 429 {
				count429++
			}
		}
		t.Logf("Summary: %d/20 requests got 429", count429)
	})

	t.Run("with_option_a", func(t *testing.T) {
		client := NewRealAPIClientA(5, 1, baseURL, apiKey)
		for i := 0; i < 10; i++ {
			start := time.Now()
			resp, err := client.Do(ctx, apiEndpoint)
			elapsed := time.Since(start)
			if err != nil {
				t.Logf("Option A request %d: error=%v duration=%s", i+1, err, elapsed.Round(time.Millisecond))
			} else {
				t.Logf("Option A request %d: status=%d duration=%s retries=%d",
					i+1, resp.StatusCode, elapsed.Round(time.Millisecond), resp.Retries)
			}
		}
	})

	t.Run("with_option_b", func(t *testing.T) {
		client := NewRealAPIClientB(5, baseURL, apiKey)
		for i := 0; i < 10; i++ {
			start := time.Now()
			resp, err := client.Do(ctx, apiEndpoint)
			elapsed := time.Since(start)
			if err != nil {
				t.Logf("Option B request %d: error=%v duration=%s", i+1, err, elapsed.Round(time.Millisecond))
			} else {
				t.Logf("Option B request %d: status=%d duration=%s retries=%d",
					i+1, resp.StatusCode, elapsed.Round(time.Millisecond), resp.Retries)
			}
		}
	})

	t.Run("with_option_c", func(t *testing.T) {
		client := NewRealAPIClientC(5, 1, 5, baseURL, apiKey)
		for i := 0; i < 10; i++ {
			start := time.Now()
			resp, err := client.Do(ctx, apiEndpoint)
			elapsed := time.Since(start)
			if err != nil {
				t.Logf("Option C request %d: error=%v duration=%s", i+1, err, elapsed.Round(time.Millisecond))
			} else {
				t.Logf("Option C request %d: status=%d duration=%s retries=%d",
					i+1, resp.StatusCode, elapsed.Round(time.Millisecond), resp.Retries)
			}
		}
	})
}

type realAPIResult struct {
	requestNum         int
	statusCode         int
	duration           time.Duration
	retryAfter         string
	rateLimitRemaining string
	err                error
}

func makeRealAPIRequest(ctx context.Context, url, apiKey string) (*Response, error) {
	client := &noLimitClient{apiKey: apiKey}
	return client.Do(ctx, url)
}

type noLimitClient struct {
	apiKey string
}

func (c *noLimitClient) Do(ctx context.Context, url string) (*Response, error) {
	return doRealRequest(ctx, url, c.apiKey)
}

func (c *noLimitClient) Name() string {
	return "No Rate Limiting"
}

func TestGenerateBenchmarkReport(t *testing.T) {
	server := NewMockRateLimitServer(mockMaxReqsPerWindow, mockWindow)
	defer server.Close()

	ctx := context.Background()
	concurrencyLevels := []int{1, 5, 10, 20}

	clients := []RateLimitClient{
		NewClientSideLimiter(float64(mockMaxReqsPerWindow), 1),
		NewRetryOn429Client(5),
		NewCombinedClient(float64(mockMaxReqsPerWindow), 1, 5),
	}

	var allResults []BenchmarkResult

	for _, client := range clients {
		for _, c := range concurrencyLevels {
			server.ResetStats()
			result := RunBenchmark(ctx, client, server.URL(), c, requestsPerTest)
			serverReqs, server429s := server.Stats()
			result.ServerRequests = serverReqs
			result.Server429s = server429s
			allResults = append(allResults, result)
		}
	}

	report := "# Rate Limiting Benchmark Results\n\n"
	report += "## Mock Server Configuration\n"
	report += fmt.Sprintf("- Max requests per window: %d\n", mockMaxReqsPerWindow)
	report += fmt.Sprintf("- Window duration: %s\n", mockWindow)
	report += fmt.Sprintf("- Requests per test: %d\n\n", requestsPerTest)
	report += "## Results\n\n"
	report += FormatResults(allResults)
	report += "\n## Server-Side Stats\n\n"
	report += "| Strategy | Concurrency | Server Requests | Server 429s |\n"
	report += "|----------|-------------|-----------------|-------------|\n"
	for _, r := range allResults {
		report += fmt.Sprintf("| %s | %d | %d | %d |\n",
			r.Strategy, r.Concurrency, r.ServerRequests, r.Server429s)
	}

	t.Log("\n" + report)
}
