package ratelimit_poc

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// RunBenchmark runs a concurrency benchmark for a given strategy against a URL.
func RunBenchmark(ctx context.Context, client RateLimitClient, url string, concurrency, totalRequests int) BenchmarkResult {
	var (
		wg           sync.WaitGroup
		successCount int64
		failureCount int64
		total429s    int64
		totalRetries int64
		latencies    []time.Duration
		latMu        sync.Mutex
	)

	requestCh := make(chan int, totalRequests)
	for i := 0; i < totalRequests; i++ {
		requestCh <- i
	}
	close(requestCh)

	start := time.Now()

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range requestCh {
				reqStart := time.Now()
				resp, err := client.Do(ctx, url+"/api/test")
				elapsed := time.Since(reqStart)

				latMu.Lock()
				latencies = append(latencies, elapsed)
				latMu.Unlock()

				if err != nil {
					atomic.AddInt64(&failureCount, 1)
					continue
				}

				if resp.StatusCode == 200 {
					atomic.AddInt64(&successCount, 1)
				} else if resp.StatusCode == 429 {
					atomic.AddInt64(&total429s, 1)
				} else {
					atomic.AddInt64(&failureCount, 1)
				}
				atomic.AddInt64(&totalRetries, int64(resp.Retries))
			}
		}()
	}

	wg.Wait()
	totalDuration := time.Since(start)

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	var avgLatency time.Duration
	if len(latencies) > 0 {
		var sum time.Duration
		for _, l := range latencies {
			sum += l
		}
		avgLatency = sum / time.Duration(len(latencies))
	}

	var p99Latency time.Duration
	if len(latencies) > 0 {
		idx := int(float64(len(latencies)) * 0.99)
		if idx >= len(latencies) {
			idx = len(latencies) - 1
		}
		p99Latency = latencies[idx]
	}

	throughput := float64(successCount) / totalDuration.Seconds()

	return BenchmarkResult{
		Strategy:      client.Name(),
		Concurrency:   concurrency,
		TotalRequests: totalRequests,
		SuccessCount:  int(successCount),
		FailureCount:  int(failureCount),
		Total429s:     int(total429s),
		TotalRetries:  int(totalRetries),
		TotalDuration: totalDuration,
		AvgLatency:    avgLatency,
		P99Latency:    p99Latency,
		Throughput:    throughput,
	}
}

// FormatResults formats benchmark results as a markdown table.
func FormatResults(results []BenchmarkResult) string {
	s := "| Strategy | Concurrency | Total Reqs | Success | 429s | Retries | Duration | Avg Latency | P99 Latency | Throughput (rps) |\n"
	s += "|----------|-------------|------------|---------|------|---------|----------|-------------|-------------|------------------|\n"
	for _, r := range results {
		s += fmt.Sprintf("| %s | %d | %d | %d | %d | %d | %s | %s | %s | %.1f |\n",
			r.Strategy, r.Concurrency, r.TotalRequests,
			r.SuccessCount, r.Total429s, r.TotalRetries,
			r.TotalDuration.Round(time.Millisecond),
			r.AvgLatency.Round(time.Microsecond),
			r.P99Latency.Round(time.Microsecond),
			r.Throughput,
		)
	}
	return s
}
