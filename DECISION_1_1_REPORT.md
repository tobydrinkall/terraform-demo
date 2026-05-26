# Decision 1.1: HTTP Client Library for Devin API v3

## Executive Summary

Two complete Devin API v3 client implementations were built and tested:
- **Option A**: `net/http` (Go standard library) with hand-rolled retry logic
- **Option B**: `hashicorp/go-retryablehttp` with built-in retry infrastructure

**Recommendation: Option B (`hashicorp/go-retryablehttp`)** — it provides battle-tested retry logic, integrates natively with the HashiCorp ecosystem, and requires ~6% fewer lines of code.

---

## Side-by-Side Code Comparison

### Line Counts

| Component | Option A (stdlib) | Option B (retryablehttp) |
|---|---|---|
| `client.go` (main logic) | 380 | 351 |
| `types.go` (shared types) | 98 | 98 |
| **Total implementation** | **478** | **449** |
| `client_test.go` (unit tests) | 395 | 395 |
| `integration_test.go` (live API) | 111 | 108 |
| **Total tests** | **506** | **503** |
| **Grand total** | **984** | **952** |

Option B is **29 lines shorter** (~6%) because retry logic, backoff calculation, and network error classification are handled by the library.

### Structural Comparison

Both implementations share an identical public API surface:

```go
type Client struct {
    apiKey     string
    baseURL    string
    orgID      string
    httpClient *http.Client  // or *retryablehttp.Client
    userAgent  string
    logger     Logger
    // stdlib only: maxRetries, baseDelay, maxDelay
}

func NewClient(apiKey, orgID string, opts ...Option) *Client
func (c *Client) CreateKnowledgeNote(ctx, req) (*KnowledgeNote, error)
func (c *Client) GetKnowledgeNote(ctx, noteID) (*KnowledgeNote, error)
func (c *Client) UpdateKnowledgeNote(ctx, noteID, req) (*KnowledgeNote, error)
func (c *Client) DeleteKnowledgeNote(ctx, noteID) error
func (c *Client) ListKnowledgeNotes(ctx, first, after) (*ListKnowledgeNotesResponse, error)
```

### Key Code Differences

#### 1. Retry Loop

**Option A (stdlib)** — Hand-rolled retry loop (~50 lines):
```go
for attempt := 0; attempt <= c.maxRetries; attempt++ {
    if attempt > 0 {
        delay := c.calculateBackoff(attempt)
        select {
        case <-ctx.Done():
            return nil, 0, ctx.Err()
        case <-time.After(delay):
        }
    }
    // ... make request, check status, decide retry ...
}
```

**Option B (retryablehttp)** — Delegate to library (~5 lines of config):
```go
retryClient := retryablehttp.NewClient()
retryClient.RetryMax = defaultMaxRetries
retryClient.RetryWaitMin = defaultBaseDelay
retryClient.RetryWaitMax = defaultMaxDelay
retryClient.CheckRetry = customCheckRetry
```

#### 2. Backoff Calculation

**Option A** — Manual exponential backoff with jitter:
```go
func (c *Client) calculateBackoff(attempt int) time.Duration {
    backoff := float64(c.baseDelay) * math.Pow(2, float64(attempt-1))
    jitter := rand.Float64() * float64(c.baseDelay)
    delay := time.Duration(backoff + jitter)
    if delay > c.maxDelay { delay = c.maxDelay }
    return delay
}
```

**Option B** — Built into library (uses `retryablehttp.LinearJitterBackoff` or `DefaultBackoff`).

#### 3. Request Creation

**Option A** — `http.NewRequestWithContext(ctx, method, url, body)`
**Option B** — `retryablehttp.NewRequestWithContext(ctx, method, url, body)` (wraps body for re-reads)

---

## Test Results

### Unit Tests (httptest.Server)

Both implementations pass all 15 identical unit tests:

| Test | Option A | Option B |
|---|---|---|
| TestCreateKnowledgeNote | PASS | PASS |
| TestGetKnowledgeNote | PASS | PASS |
| TestUpdateKnowledgeNote | PASS | PASS |
| TestDeleteKnowledgeNote | PASS | PASS |
| TestListKnowledgeNotes | PASS | PASS |
| TestRetry429 | PASS | PASS |
| TestRetry429Exhausted | PASS | PASS |
| TestRetry5xx | PASS | PASS |
| TestRetry5xxExhausted | PASS | PASS |
| TestErrorParsing404 | PASS | PASS |
| TestErrorParsing403 | PASS | PASS |
| TestErrorParsing422 | PASS | PASS |
| TestContextCancellation | PASS | PASS |
| TestContextCancellationDuringRetry | PASS | PASS |
| TestUserAgentHeader | PASS | PASS |

### Integration Tests (Live Devin API v3)

Both clients successfully performed the full CRUD lifecycle against the live API:

| Operation | Option A (stdlib) | Option B (retryablehttp) |
|---|---|---|
| Create note | PASS (note-04172faa...) | PASS (note-6251d662...) |
| Read note | PASS | PASS |
| Update note | PASS | PASS |
| List notes | PASS (4 total) | PASS (5 total) |
| Delete note | PASS | PASS |
| Verify deletion (404) | PASS | PASS |

**Total integration time**: Option A = 4.98s, Option B = 5.62s (negligible difference, dominated by network latency).

---

## Retry Behavior Comparison

| Aspect | Option A (stdlib) | Option B (retryablehttp) |
|---|---|---|
| 429 handling | Manual: parse Retry-After, sleep, then enter next iteration | Library: automatic via CheckRetry returning true; Retry-After supported via Backoff func |
| 5xx handling | Manual: classify status >= 500, continue loop | Library: automatic via CheckRetry |
| Backoff algorithm | Hand-rolled exponential + random jitter | Library default: exponential + jitter (or linear jitter) |
| Max retries config | Stored on Client struct, checked in loop | `retryClient.RetryMax` |
| Network errors | Manual `isRetryableNetworkError()` string matching | Library `DefaultRetryPolicy` (uses `net.Error` interface) |
| Context respect | Manual `select` on ctx.Done between retries | Built-in context propagation |
| Body re-reading | Manual `bytes.NewReader` reset per attempt | Library handles via `retryablehttp.Request` (seekable body) |

**Key finding**: The stdlib retry loop has a subtle correctness risk — the `Retry-After` wait happens *inside* the current iteration, then the next iteration's backoff wait happens *again*. The retryablehttp library handles this cleanly with its `Backoff` function, which can inspect `Retry-After` headers.

---

## Logging Output Comparison

### Option A (stdlib) — Custom Logger Interface

```
[DEBUG] sending request [method GET url https://api.devin.ai/v3/... attempt 0]
[DEBUG] received response [status 200 body_length 416 method GET url ...]
[WARN]  retrying request [attempt 1 max_retries 2 delay 18.701529ms ...]
[WARN]  rate limited, using Retry-After header [retry_after_seconds 1]
```

Uses a simple `Logger` interface with `Debug/Info/Warn/Error` methods that accept key-value pairs, directly compatible with `hclog.Logger` from terraform-plugin-log.

### Option B (retryablehttp) — LeveledLogger Adapter

```
[DEBUG] sending request [method GET url https://api.devin.ai/v3/...]
[DEBUG] received response [status 200 body_length 416 ...]
```

Uses a `hclogAdapter` struct that adapts our `Logger` interface to `retryablehttp.LeveledLogger`. The library emits its own retry log messages (e.g., "retrying request after error" or "request failed, retrying").

**Key finding**: Option B gets retry logging for free from the library. Option A requires manual log statements at each retry decision point. Both are fully compatible with terraform-plugin-log's `hclog.Logger`.

---

## Dependency Analysis

### Option A (stdlib)

```
go.mod dependencies: 0 (zero external dependencies)
```

Standard library only: `net/http`, `context`, `encoding/json`, `math`, `math/rand`, `time`, `io`, `bytes`, `fmt`, `net/url`, `strconv`, `strings`, `log`.

### Option B (retryablehttp)

```
go.mod dependencies:
  github.com/hashicorp/go-retryablehttp v0.7.8
  github.com/hashicorp/go-cleanhttp v0.5.2  (transitive)
```

Both are HashiCorp-maintained, well-audited, widely used in the Terraform ecosystem:
- `go-retryablehttp`: 2.2k GitHub stars, used by Terraform core and every official provider
- `go-cleanhttp`: Sensible HTTP client defaults, zero external deps itself

### Dependency Risk Assessment

| Factor | Option A | Option B |
|---|---|---|
| External deps | 0 | 2 |
| Supply chain risk | None | Minimal (HashiCorp-maintained) |
| License | N/A | MPL-2.0 (same as Terraform) |
| Maintenance | Self-maintained | HashiCorp-maintained |
| Breaking changes | N/A | Stable API, follows semver |
| Ecosystem fit | Generic | Native Terraform ecosystem |

---

## Recommendation

**Use Option B: `hashicorp/go-retryablehttp`**

### Rationale

1. **Battle-tested retry logic**: The library handles edge cases (body re-reading, connection draining, TLS handshake failures) that the hand-rolled stdlib version must handle manually and risks getting wrong.

2. **Ecosystem alignment**: Every official Terraform provider uses `go-retryablehttp`. Reviewers and contributors will recognize the patterns instantly. The library integrates natively with `hclog` (terraform-plugin-log).

3. **Less code to maintain**: 29 fewer lines may seem small, but the *complexity* delta is larger — the retry loop, backoff calculation, and network error classification in Option A are all subtle code paths that need ongoing testing and maintenance.

4. **Correct Retry-After handling**: The library's `Backoff` function cleanly integrates `Retry-After` header parsing without the double-wait bug risk present in the hand-rolled version.

5. **Minimal dependency cost**: Only 2 transitive deps, both HashiCorp-maintained under MPL-2.0 (same license as Terraform itself). Zero supply chain risk delta vs. using Terraform SDK.

### When to choose Option A instead

- If building a non-Terraform Go project where minimizing external dependencies is critical
- If you need HTTP/2 push or other features not supported by retryablehttp's wrapper
- If the project has strict "zero external deps" policies

---

## Files Produced

### Option A: `internal/client_stdlib/`
- `client.go` — HTTP client with hand-rolled retry (380 lines)
- `types.go` — Request/response types and error types (98 lines)
- `client_test.go` — 15 unit tests using httptest.Server (395 lines)
- `integration_test.go` — Live API CRUD test (111 lines)

### Option B: `internal/client_retryable/`
- `client.go` — HTTP client using go-retryablehttp (351 lines)
- `types.go` — Request/response types and error types (98 lines)
- `client_test.go` — 15 unit tests using httptest.Server (395 lines)
- `integration_test.go` — Live API CRUD test (108 lines)

### Shared
- `go.mod` / `go.sum` — Module definition with dependencies
- `DECISION_1_1_REPORT.md` — This report
