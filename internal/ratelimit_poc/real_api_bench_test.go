package ratelimit_poc

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRealAPIConcurrentBurst(t *testing.T) {
	apiKey := os.Getenv("DEVIN_API_KEY_TERRAFORM")
	if apiKey == "" {
		t.Skip("DEVIN_API_KEY_TERRAFORM not set, skipping real API test")
	}

	baseURL := "https://api.devin.ai"
	orgID := "org-8d158f07ee0f4678a467078b323880a4"
	apiEndpoint := baseURL + "/v3/organizations/" + orgID + "/knowledge/notes?first=1"
	ctx := context.Background()

	concurrencyLevels := []int{5, 10, 20}

	for _, concurrency := range concurrencyLevels {
		t.Run("concurrent_burst", func(t *testing.T) {
			var (
				wg         sync.WaitGroup
				success    int64
				count429   int64
				errors     int64
				otherCodes int64
			)

			totalRequests := 30
			start := time.Now()

			sem := make(chan struct{}, concurrency)
			for i := 0; i < totalRequests; i++ {
				wg.Add(1)
				go func(reqNum int) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					resp, err := doRealRequest(ctx, apiEndpoint, apiKey)
					if err != nil {
						atomic.AddInt64(&errors, 1)
						t.Logf("  Request %d: error=%v", reqNum, err)
						return
					}

					switch resp.StatusCode {
					case 200:
						atomic.AddInt64(&success, 1)
					case 429:
						atomic.AddInt64(&count429, 1)
						t.Logf("  Request %d: 429! retry-after=%s", reqNum, resp.Headers.Get("Retry-After"))
					default:
						atomic.AddInt64(&otherCodes, 1)
						t.Logf("  Request %d: status=%d", reqNum, resp.StatusCode)
					}
				}(i + 1)
			}

			wg.Wait()
			elapsed := time.Since(start)

			t.Logf("Concurrency=%d: %d total, %d success, %d 429s, %d errors, %d other | %.1f rps | %s",
				concurrency, totalRequests, success, count429, errors, otherCodes,
				float64(success)/elapsed.Seconds(), elapsed.Round(time.Millisecond))
		})

		time.Sleep(2 * time.Second)
	}
}
