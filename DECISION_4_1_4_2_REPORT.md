# Decision 4.1 & 4.2: Session Representation in Terraform

## Decision Summary

| ID | Question | Recommendation |
|----|----------|---------------|
| 4.1 | Should sessions be a managed resource, data source only, or both? | **Both** (Approach D) |
| 4.2 | If a resource, should create be fire-and-forget or wait for completion? | **Configurable** — default fire-and-forget with opt-in `wait_for_completion` |

---

## Session Lifecycle (Observed via API)

Real API calls were made using the Devin API v3 to map the full session lifecycle.

### State Machine

```
                              ┌─────────────────────────────────────────────────────┐
                              │              Session Lifecycle                       │
                              │                                                     │
   POST /sessions ──────►  ┌──┴──┐     ┌─────────┐     ┌─────────┐     ┌─────────┐ │
                           │ new │────►│ claimed │────►│ running │────►│  exit   │ │
                           └─────┘     └─────────┘     └────┬────┘     └─────────┘ │
                              │                              │              ▲       │
                              │                              │              │       │
                              │                         status_detail:      │       │
                              │                         ┌──────────┐        │       │
                              │                         │ working  │────────┘       │
                              │                         ├──────────┤   (completes   │
                              │                         │waiting_  │   or DELETE)   │
                              │                         │for_user  │                │
                              │                         ├──────────┤                │
                              │                         │ finished │────────────────┘
                              │                         └──────────┘
                              │
                              │     DELETE /sessions/{devin_id}
                              │     ┌─────────────────────────────┐
                              │     │ Can be called at any state  │
                              │     │ Async: status transitions   │
                              │     │ to "exit" over ~10 seconds  │
                              │     └─────────────────────────────┘
                              └─────────────────────────────────────────────────────┘
```

### Observed Transitions (Real API Calls)

```
Time  │ Action                                    │ Status        │ Detail
──────┼───────────────────────────────────────────┼───────────────┼──────────────
 0s   │ POST /sessions (create)                   │ new           │ null
 2s   │ GET  /sessions/devin-{id} (poll)          │ claimed       │ null
 7s   │ GET  /sessions/devin-{id} (poll)          │ claimed       │ null
17s   │ GET  /sessions/devin-{id} (poll)          │ running       │ waiting_for_user
20s   │ GET  /sessions/devin-{id} (full details)  │ running       │ waiting_for_user
23s   │ DELETE /sessions/devin-{id} (terminate)   │ running       │ null
26s   │ GET  /sessions/devin-{id} (post-delete)   │ running       │ finished
36s   │ GET  /sessions/devin-{id} (final)         │ exit          │ null
```

**Key observations:**
1. **Create returns immediately** — status is `new`, no session_id prefix needed in POST
2. **`claimed` is a transitional state** — session is being assigned to a worker (2-10s)
3. **`running` has sub-states** via `status_detail`: `working`, `waiting_for_user`, `finished`
4. **DELETE is asynchronous** — returns immediately but session takes ~10s to reach `exit`
5. **`devin-` prefix required** — GET/DELETE paths use `devin-{session_id}`, not raw UUID
6. **Sessions are NOT deleted** — they persist in `exit` state indefinitely (queryable)

### API Endpoints Used

| Operation | Method | Path |
|-----------|--------|------|
| Create | `POST` | `/v3/organizations/{org_id}/sessions` |
| Get | `GET` | `/v3/organizations/{org_id}/sessions/devin-{session_id}` |
| List | `GET` | `/v3/organizations/{org_id}/sessions?limit=N` |
| Terminate | `DELETE` | `/v3/organizations/{org_id}/sessions/devin-{session_id}` |

---

## Destroy Behavior Analysis

### What happens when you `terraform destroy` an active session?

```
┌──────────────────────────────────────────────────────────────────────────┐
│                    terraform destroy Flow                                │
│                                                                          │
│  State of Session          │  Destroy Behavior                           │
│  ─────────────────────────┼───────────────────────────────────────────── │
│  new / claimed / running  │  DELETE → poll until exit → remove state    │
│  running (finished)       │  DELETE → poll until exit → remove state    │
│  exit                     │  No API call → remove from state only       │
│  not found (404)          │  No API call → remove from state only       │
└──────────────────────────────────────────────────────────────────────────┘
```

**Critical insight:** `terraform destroy` of an active session **terminates ongoing work**. 
This is the correct behavior for infrastructure (you destroy what you created), but is 
**potentially dangerous** for sessions since:

- Devin may be mid-task (writing code, deploying, etc.)
- Termination is immediate and irreversible
- Any in-progress PRs or deployments become orphaned

**Mitigation strategies implemented:**
1. Poll after DELETE to confirm termination (not just fire-and-forget)
2. Warn in docs about active session termination
3. Approach D suggests `prevent_destroy` lifecycle for critical sessions

---

## Approach Comparison

### Approach A: Managed Resource (Fire-and-Forget)

```
terraform apply                          terraform destroy
    │                                         │
    ▼                                         ▼
POST /sessions ─────► return immediately   DELETE /sessions/{id}
    │                                         │
    ▼                                         ▼
State: session_id,                        Poll until exit
       url, status="new"                      │
                                              ▼
                                         Remove from state
```

| Aspect | Assessment |
|--------|------------|
| **Complexity** | Low (~100 LoC) |
| **UX** | Simple — matches Terraform's async resource model |
| **terraform plan** | Shows session creation, no state drift from status changes |
| **Limitation** | No way to know when session completes |
| **Best for** | Background tasks, batch operations, "launch and forget" |

### Approach B: Managed Resource (Wait for Completion)

```
terraform apply (wait=true)                   terraform apply (wait=false)
    │                                              │
    ▼                                              ▼
POST /sessions                                POST /sessions
    │                                              │
    ▼                                              ▼
Poll status every 30s ◄──── timeout?          Return immediately
    │                    │     │               (same as Approach A)
    │                    │     ▼
    │                    │  Return with
    │                    │  timed_out=true
    ▼                    │
status == exit?          │
    │                    │
    ▼                    │
Return with full         │
session results          │
(acus, title, etc)       │
```

| Aspect | Assessment |
|--------|------------|
| **Complexity** | Medium (~200 LoC) |
| **UX** | Flexible — subsumes Approach A |
| **terraform plan** | Same as A when wait=false; when wait=true, terraform apply blocks |
| **Timeout handling** | Graceful — sets `timed_out=true`, doesn't error |
| **Best for** | CI/CD pipelines, sequential tasks, deployment gates |

### Approach C: Data Sources Only

```
terraform plan/apply
    │
    ▼
GET /sessions (list)          GET /sessions/{id} (single)
    │                              │
    ▼                              ▼
Read-only snapshot             Read-only snapshot
of session state               of session state

No CREATE, UPDATE, or DELETE operations
```

| Aspect | Assessment |
|--------|------------|
| **Complexity** | Low (~150 LoC) |
| **UX** | Read-only — familiar pattern for Terraform users |
| **terraform plan** | Always refreshes, no state management concerns |
| **Limitation** | Cannot create or manage sessions |
| **Best for** | Monitoring dashboards, audit, referencing externally-created sessions |

### Approach D: Resource + Data Sources Combined

```
┌─────────────────────────────────────────────────────────────────┐
│                      Approach D                                  │
│                                                                  │
│   Resource (from B)              Data Sources (from C)           │
│   ┌───────────────────┐         ┌──────────────────────┐        │
│   │ devin_session     │         │ devin_session (data)  │        │
│   │  - create         │         │  - get by ID          │        │
│   │  - read           │         ├──────────────────────┤        │
│   │  - delete         │         │ devin_sessions (data) │        │
│   │  - wait option    │         │  - list/filter        │        │
│   └───────────────────┘         └──────────────────────┘        │
│                                                                  │
│   resource "devin_session"      data "devin_session"             │
│   "my_task" {                   "lookup" {                       │
│     prompt = "..."                session_id = "abc123"          │
│     wait   = true               }                                │
│   }                                                              │
│                                 data "devin_sessions"            │
│                                 "running" {                      │
│                                   status_filter = "running"      │
│                                 }                                │
└─────────────────────────────────────────────────────────────────┘
```

| Aspect | Assessment |
|--------|------------|
| **Complexity** | Medium (reuses B + C, ~350 LoC total) |
| **UX** | Complete — covers all use cases |
| **terraform plan** | Full lifecycle + read-only queries |
| **prevent_destroy** | User opt-in via `lifecycle` block (NOT default) |
| **Best for** | Production provider — serves all user personas |

---

## UX Comparison Matrix

```
                    ┌─────────┬──────────┬──────────┬──────────┐
                    │    A    │    B     │    C     │    D     │
                    │Fire&Fgt │Wait+Opt │Data Only │Combined  │
┌───────────────────┼─────────┼──────────┼──────────┼──────────┤
│ Create sessions   │   ✓     │    ✓     │    ✗     │    ✓     │
│ Wait for result   │   ✗     │    ✓     │    ✗     │    ✓     │
│ Query sessions    │   ✗     │    ✗     │    ✓     │    ✓     │
│ Destroy sessions  │   ✓     │    ✓     │   N/A    │    ✓     │
│ CI/CD pipelines   │   ~     │    ✓     │    ✗     │    ✓     │
│ Batch operations  │   ✓     │    ✓     │    ✗     │    ✓     │
│ Audit/monitoring  │   ✗     │    ✗     │    ✓     │    ✓     │
│ Sequential tasks  │   ✗     │    ✓     │    ✗     │    ✓     │
│ Terraform idiom   │   ✓     │    ~     │    ✓     │    ✓     │
│ State drift risk  │  Low    │   Low    │   None   │   Low    │
│ Accidental kill   │  Risk   │   Risk   │   None   │   Risk   │
└───────────────────┴─────────┴──────────┴──────────┴──────────┘
```

---

## Use Case Mapping

### Use Case 1: "Launch 5 Devins to review PRs"
**Best approach: A or D** — Fire-and-forget, use `for_each` to launch batch sessions.

```hcl
resource "devin_session" "reviews" {
  for_each = toset(var.pr_numbers)
  prompt   = "Review PR #${each.value}"
}
```

### Use Case 2: "Run Devin, wait for it to finish, then deploy"
**Best approach: B or D** — Sequential pipeline with `wait_for_completion`.

```hcl
resource "devin_session" "test" {
  prompt              = "Run test suite"
  wait_for_completion = true
  timeout             = "30m"
}

resource "devin_session" "deploy" {
  depends_on          = [devin_session.test]
  prompt              = "Deploy to staging"
  wait_for_completion = true
}
```

### Use Case 3: "Check how many sessions are running"
**Best approach: C or D** — Data source query, no session management.

```hcl
data "devin_sessions" "active" {
  status_filter = "running"
}

output "active_count" {
  value = length(data.devin_sessions.active.sessions)
}
```

### Use Case 4: "Reference a session created via Slack/webapp"
**Best approach: C or D** — Look up by ID, read-only.

```hcl
data "devin_session" "slack_session" {
  session_id = var.session_from_slack
}

output "session_status" {
  value = data.devin_session.slack_session.status
}
```

---

## The Ephemeral Resource Question

Sessions are fundamentally different from typical Terraform resources:

```
Traditional Infrastructure          Devin Sessions
─────────────────────────           ──────────────────
Long-lived (months/years)           Short-lived (minutes/hours)
Stable state                        Constantly changing state
Idempotent updates                  Immutable after creation
Destroy = remove                    Destroy = terminate work
State drift = problem               State drift = expected
```

**Why a resource still makes sense:**
1. Terraform's lifecycle model (create/read/delete) maps cleanly to session lifecycle
2. `terraform destroy` = terminate is intuitive and useful
3. `for_each` enables batch session creation
4. State tracking enables `depends_on` for sequential workflows
5. The `wait_for_completion` pattern is well-established (see `aws_instance` with `user_data`)

**Why data sources are also needed:**
1. Most sessions are created outside Terraform (webapp, Slack, API)
2. Monitoring/audit use cases don't need lifecycle management
3. Cross-referencing sessions between Terraform configs

---

## Recommendation

### Decision 4.1: **Both resource AND data sources (Approach D)**

The provider should ship with:
- `devin_session` resource — for creating and managing sessions
- `devin_session` data source — for looking up a single session by ID
- `devin_sessions` data source — for listing/filtering sessions

This covers all use cases without forcing users into a single paradigm.

### Decision 4.2: **Configurable with fire-and-forget default (Approach B pattern)**

The resource should:
- Default to fire-and-forget (`wait_for_completion = false`)
- Support opt-in waiting (`wait_for_completion = true`)
- Handle timeout gracefully (warning, not error)
- Poll with configurable interval (default 30s)

**Rationale:**
- Fire-and-forget default matches Terraform's async model and is safest
- Wait-for-completion is essential for CI/CD pipelines
- Making it configurable avoids needing two separate resources
- Graceful timeout prevents terraform from hanging indefinitely

### Additional Recommendations

1. **Do NOT set `prevent_destroy` by default** — sessions are ephemeral; document the pattern for users who want it
2. **Add `org_id` to provider config** — required for all session API paths
3. **Consider `ignore_changes` on `status`** — prevent constant state drift on reads
4. **Add `tags` as a mutable attribute** — the API supports `PUT /sessions/{id}/tags`

---

## Implementation Files

| File | LoC | Purpose |
|------|-----|---------|
| `client.go` | ~190 | Shared API client for all approaches |
| `approach_a_fire_and_forget.go` | ~150 | Approach A: fire-and-forget resource |
| `approach_b_wait_for_completion.go` | ~220 | Approach B: wait-for-completion resource |
| `approach_c_data_sources.go` | ~200 | Approach C: data sources (single + list) |
| `approach_d_combined.go` | ~50 | Approach D: architecture documentation |
| `examples/approach_*/main.tf` | ~40ea | Example Terraform configurations |

All implementations are in `internal/provider/sessions_poc/` and compile successfully.
