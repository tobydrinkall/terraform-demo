# Decision 6.1: Testing Strategy for Terraform Devin API Provider

## Executive Summary

Three testing strategies were built and validated against the `devin_knowledge_note` resource (CRUD + import). All three approaches produce **passing** test suites. **Recommendation: Option C (httptest mock server) as the primary test strategy, supplemented by a thin Option A acceptance test suite gated behind `TF_ACC=1`.** Option B (go-vcr) adds complexity without proportional value for this use case.

---

## What Was Built

All test code lives in `internal/provider/testing_poc/`:

| File | Lines | Purpose |
|------|-------|---------|
| `option_a_acceptance_test.go` | 191 | Live API acceptance tests |
| `option_b_vcr_test.go` | 212 | go-vcr cassette record/replay tests |
| `option_c_mock_server.go` | 207 | In-memory mock Devin API server |
| `option_c_mock_test.go` | 282 | Provider tests against mock server |
| `helpers_test.go` | 39 | Shared HTTP test helpers |
| `testdata/cassettes/*.yaml` | 437 | Recorded HTTP cassettes (3 files) |

---

## Option A: Live API Acceptance Tests

### How It Works
- Uses `resource.Test()` from `terraform-plugin-testing`
- `PreCheck` ensures `DEVIN_API_KEY` and `DEVIN_ORG_ID` are set
- Each test creates real knowledge notes via the Devin API, verifies attributes, and cleans up via Terraform destroy
- `CheckDestroy` independently verifies the resource was deleted from the API
- Gated on `TF_ACC=1`

### Tests Implemented
| Test | Description |
|------|-------------|
| `TestAccKnowledgeNote_basic` | Create → verify all attributes → destroy |
| `TestAccKnowledgeNote_update` | Create → update name/body/trigger → verify → destroy |
| `TestAccKnowledgeNote_import` | Create → `terraform import` → verify state matches |

### Metrics
- **Test code**: 191 lines
- **Infrastructure code**: 0 lines (uses built-in test framework)
- **Setup complexity**: Low — just needs 2 env vars
- **Runtime**: ~10s (3 tests, sequential API calls)
- **Dependencies added**: None

### Test Output
```
=== RUN   TestAccKnowledgeNote_basic
--- PASS: TestAccKnowledgeNote_basic (1.18s)
=== RUN   TestAccKnowledgeNote_update
--- PASS: TestAccKnowledgeNote_update (1.98s)
=== RUN   TestAccKnowledgeNote_import
--- PASS: TestAccKnowledgeNote_import (6.85s)
PASS
ok      github.com/tobydrinkall/terraform-demo/internal/provider/testing_poc    10.017s
```

### CI Pipeline Requirements
```yaml
env:
  TF_ACC: "1"
  DEVIN_API_KEY: ${{ secrets.DEVIN_API_KEY }}
  DEVIN_ORG_ID: ${{ secrets.DEVIN_ORG_ID }}
```
- Requires **2 secrets** in CI
- Creates **real API resources** during test runs
- Tests must run **serially** (parallel runs risk resource conflicts)
- Rate limiting may cause flaky failures

---

## Option B: Recorded HTTP Cassettes (go-vcr)

### How It Works
- Uses `gopkg.in/dnaeon/go-vcr.v4` to record and replay HTTP interactions
- `VCR_RECORD=1` mode records real API calls to YAML cassette files
- Default (replay) mode plays back from cassettes — no secrets or network needed
- `BeforeSaveHook` sanitizes auth headers and scrubs org_id to a stable placeholder
- Custom `MatcherFunc` compares method + full URL for sequential interaction matching

### Tests Implemented
| Test | Description |
|------|-------------|
| `TestVCR_KnowledgeNote_basic` | Create → verify → destroy (recorded) |
| `TestVCR_KnowledgeNote_update` | Create → update → verify (recorded) |
| `TestVCR_KnowledgeNote_import` | Create → import → verify (recorded) |

### Metrics
- **Test code**: 212 lines
- **Infrastructure code**: 437 lines of YAML cassettes
- **Setup complexity**: Medium — VCR hooks, matcher config, org_id scrubbing
- **Runtime**: ~2s (replay mode), ~9s (record mode)
- **Dependencies added**: `gopkg.in/dnaeon/go-vcr.v4`

### Test Output (Replay Mode)
```
=== RUN   TestVCR_KnowledgeNote_basic
--- PASS: TestVCR_KnowledgeNote_basic (0.51s)
=== RUN   TestVCR_KnowledgeNote_update
--- PASS: TestVCR_KnowledgeNote_update (0.83s)
=== RUN   TestVCR_KnowledgeNote_import
--- PASS: TestVCR_KnowledgeNote_import (0.67s)
PASS
ok      github.com/tobydrinkall/terraform-demo/internal/provider/testing_poc    2.018s
```

### CI Pipeline Requirements
```yaml
# Replay mode (default): no secrets needed
- run: go test ./internal/provider/testing_poc/ -run TestVCR_

# Record mode (manual): needs secrets
env:
  VCR_RECORD: "1"
  DEVIN_API_KEY: ${{ secrets.DEVIN_API_KEY }}
  DEVIN_ORG_ID: ${{ secrets.DEVIN_ORG_ID }}
```
- Replay needs **0 secrets** in CI
- Cassette files must be **committed to the repo** (437 lines of YAML)
- Cassettes must be **re-recorded** when API behavior changes
- Cassettes are **brittle** — if the number of HTTP requests changes (e.g., due to retries or provider framework changes), the cassettes break

### Challenges Encountered During PoC
1. **org_id scrubbing**: Real org_id must be scrubbed from URLs and response bodies to make cassettes portable
2. **Matcher complexity**: Default matcher checks proto version, headers, and body — all of which differ between record/replay. Required custom method+URL matcher.
3. **Multi-step tests**: The update test required careful cassette recording — VCR replays interactions in FIFO order, so the exact sequence of HTTP calls must be deterministic
4. **Destroy verification**: VCR intercepts the destroy-phase API calls too; destroy verification must be disabled for VCR tests

---

## Option C: httptest Mock Server

### How It Works
- `MockDevinAPI` implements the knowledge notes REST endpoints with in-memory storage
- Each test starts a fresh `httptest.Server` with the mock handler
- Provider is configured to point at the mock via `DEVIN_BASE_URL`
- No secrets, no network, no external dependencies

### Tests Implemented
| Test | Description |
|------|-------------|
| `TestMock_KnowledgeNote_basic` | Create → verify all attributes → destroy |
| `TestMock_KnowledgeNote_update` | Create → update → verify all fields |
| `TestMock_KnowledgeNote_import` | Create → import → verify state matches |
| `TestMock_KnowledgeNote_deleteIdempotent` | Create → destroy (exercises delete path) |
| `TestMock_KnowledgeNote_multiple` | Create 2 notes simultaneously |
| `TestMock_KnowledgeNote_pinnedRepo` | Create with pinned_repo attribute |
| `TestMockServer_CRUD` | Direct unit test of mock server endpoints |

### Metrics
- **Test code**: 282 lines (mock tests) + 39 lines (helpers)
- **Infrastructure code**: 207 lines (mock server)
- **Setup complexity**: Low — `setupMockServer(t)` is 8 lines
- **Runtime**: ~3.5s (7 tests)
- **Dependencies added**: None (stdlib `net/http/httptest`)

### Test Output
```
=== RUN   TestMock_KnowledgeNote_basic
--- PASS: TestMock_KnowledgeNote_basic (0.51s)
=== RUN   TestMock_KnowledgeNote_update
--- PASS: TestMock_KnowledgeNote_update (0.83s)
=== RUN   TestMock_KnowledgeNote_import
--- PASS: TestMock_KnowledgeNote_import (0.67s)
=== RUN   TestMock_KnowledgeNote_deleteIdempotent
--- PASS: TestMock_KnowledgeNote_deleteIdempotent (0.50s)
=== RUN   TestMock_KnowledgeNote_multiple
--- PASS: TestMock_KnowledgeNote_multiple (0.52s)
=== RUN   TestMock_KnowledgeNote_pinnedRepo
--- PASS: TestMock_KnowledgeNote_pinnedRepo (0.50s)
=== RUN   TestMockServer_CRUD
--- PASS: TestMockServer_CRUD (0.00s)
PASS
ok      github.com/tobydrinkall/terraform-demo/internal/provider/testing_poc    3.536s
```

### CI Pipeline Requirements
```yaml
# No secrets, no env vars, no network access needed
- run: go test ./internal/provider/testing_poc/ -run "TestMock_|TestMockServer_"
```
- **0 secrets** in CI
- **0 external dependencies**
- Tests run in **complete isolation**
- Safe to run in **parallel**

---

## Comparison Matrix

| Dimension | Option A (Live API) | Option B (go-vcr) | Option C (httptest) |
|-----------|--------------------|--------------------|---------------------|
| **Test LoC** | 191 | 212 | 321 (282+39) |
| **Infrastructure LoC** | 0 | 437 (cassettes) | 207 (mock server) |
| **Total LoC** | 191 | 649 | 528 |
| **External deps** | 0 | 1 (go-vcr) | 0 |
| **CI secrets needed** | 2 | 0 (replay) / 2 (record) | 0 |
| **Network required** | Yes | No (replay) | No |
| **Runtime** | ~10s | ~2s (replay) | ~3.5s |
| **API drift detection** | Immediate | Only when re-recording | Never (by design) |
| **Setup complexity** | Low | Medium-High | Low |
| **Maintenance burden** | Low | High (cassette rot) | Medium (mock fidelity) |
| **Test coverage** | 3 tests | 3 tests | 7 tests |
| **Parallelizable** | No (shared API state) | Yes | Yes |
| **Flakiness risk** | Medium (rate limits, network) | Low | None |

---

## Maintenance Burden Analysis

### Option A
- **When API changes**: Tests auto-adapt — they call the real API
- **When provider changes**: Just update the test config
- **Risk**: Rate limiting, network failures, test pollution (orphaned resources)
- **Operational burden**: Secrets must be rotated; tests create real resources

### Option B
- **When API changes**: Must re-record ALL cassettes (requires credentials, manual step)
- **When provider changes**: If HTTP call sequence changes, cassettes break silently
- **Risk**: Stale cassettes give false confidence — tests pass but the real API has diverged
- **Operational burden**: 437 lines of YAML cassettes to maintain; developers must understand VCR lifecycle

### Option C
- **When API changes**: Must update the mock server (207 lines of Go)
- **When provider changes**: Tests adapt automatically (mock is stateful)
- **Risk**: Mock may not match real API behavior — but this is mitigated by the fact that the mock is simple (just CRUD)
- **Operational burden**: Low — mock is Go code, reviewed alongside provider changes

---

## Recommendation

**Use Option C (httptest mock) as the primary testing layer, with a thin Option A (live API) suite for smoke testing.**

### Rationale

1. **Option C provides the best developer experience**: No secrets, no network, sub-second tests, safe to run in parallel, zero flakiness. Developers can run `go test ./...` locally without any setup.

2. **Option A is essential but should be thin**: 2-3 acceptance tests gated behind `TF_ACC=1` catch real API integration issues. Run these in CI on a schedule (nightly) or gated behind a manual trigger, not on every PR.

3. **Option B (go-vcr) is not recommended**: The maintenance burden outweighs the benefits. Cassettes are a snapshot in time — they give false confidence when the API changes. The org_id scrubbing, custom matchers, and FIFO replay ordering make VCR fragile for multi-step Terraform tests. The 437 lines of YAML cassettes are opaque and hard to review.

### Suggested CI Configuration

```yaml
jobs:
  unit-tests:
    # Runs on every PR — fast, no secrets
    steps:
      - run: go test ./... -run "TestMock_|TestMockServer_|TestUnit_"

  acceptance-tests:
    # Runs nightly or on manual trigger — requires secrets
    if: github.event_name == 'schedule' || github.event_name == 'workflow_dispatch'
    env:
      TF_ACC: "1"
      DEVIN_API_KEY: ${{ secrets.DEVIN_API_KEY }}
      DEVIN_ORG_ID: ${{ secrets.DEVIN_ORG_ID }}
    steps:
      - run: go test ./... -run "TestAcc" -timeout 300s
```

### Additional Notes

- The mock server (`option_c_mock_server.go`) should be promoted to a shared `testutil` package and extended as new resources are added
- Consider adding fuzz tests using the mock server — since it's stateful and in-process, it's ideal for property-based testing
- The acceptance test suite should include a `TestAccSweeper` to clean up orphaned resources from failed test runs
