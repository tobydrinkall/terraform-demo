# Terraform Provider for Devin API v3 — Execution Plan

## Executive Summary

Build a Terraform provider (`terraform-provider-devin`) targeting the Devin API v3, starting with a single resource: **`devin_knowledge_note`**. The provider will use the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework) (Go) and authenticate via Service User API keys.

Knowledge Notes are the ideal first resource because they have clean full CRUD semantics, a simple schema, no dependencies on other resources, and represent a genuine IaC use case (managing Devin's knowledge base as version-controlled code).

---

## Table of Contents

1. [API Surface](#1-api-surface)
2. [Provider Design](#2-provider-design)
3. [API Client Layer](#3-api-client-layer)
4. [Resource: `devin_knowledge_note`](#4-resource-devin_knowledge_note)
5. [Data Sources](#5-data-sources)
6. [File Structure](#6-file-structure)
7. [Implementation Phases](#7-implementation-phases)
8. [Testing Strategy](#8-testing-strategy)
9. [CI/CD Pipeline](#9-cicd-pipeline)
10. [Future Expansion](#10-future-expansion)

---

## 1. API Surface

### Authentication

- All requests use `Authorization: Bearer cog_xxx` header
- Credentials are Service User API keys (created in Devin Settings > Service Users)
- The service user must have the `ManageAccountKnowledge` permission (for create/list/delete) and `ManageKnowledge` (for get-by-ID)

### Knowledge Notes CRUD Endpoints

| Operation | Method   | Endpoint                                                        | Request Body                     | Response                  |
|-----------|----------|-----------------------------------------------------------------|----------------------------------|---------------------------|
| **Create**| `POST`   | `/v3/organizations/{org_id}/knowledge/notes`                    | `KnowledgeNoteCreateRequest`     | `KnowledgeNoteResponse`   |
| **Read**  | `GET`    | `/v3/organizations/{org_id}/knowledge/notes/{note_id}`          | —                                | `KnowledgeNoteResponse`   |
| **Update**| `PUT`    | `/v3/organizations/{org_id}/knowledge/notes/{note_id}`          | `KnowledgeNoteCreateRequest`     | `KnowledgeNoteResponse`   |
| **Delete**| `DELETE` | `/v3/organizations/{org_id}/knowledge/notes/{note_id}`          | —                                | `KnowledgeNoteResponse`   |
| **List**  | `GET`    | `/v3/organizations/{org_id}/knowledge/notes`                    | — (query params)                 | `PaginatedResponse`       |

### Request Schema: `KnowledgeNoteCreateRequest`

```json
{
  "name": "string (required)",
  "body": "string (required)",
  "trigger": "string (required)",
  "pinned_repo": "string | null (optional)"
}
```

### Response Schema: `KnowledgeNoteResponse`

```json
{
  "note_id": "note-abc123def456",
  "name": "string",
  "body": "string",
  "trigger": "string",
  "pinned_repo": "string | null",
  "folder_id": "string | null",
  "folder_path": "string",
  "is_enabled": true,
  "access_type": "org | enterprise",
  "org_id": "string | null",
  "macro": "string | null",
  "created_at": 1716000000,
  "updated_at": 1716000000
}
```

### List Endpoint Query Parameters

| Param         | Type              | Description                                    |
|---------------|-------------------|------------------------------------------------|
| `after`       | `string \| null`  | Cursor for pagination                          |
| `first`       | `integer`         | Page size (1–200, default 100)                 |
| `search`      | `string \| null`  | Case-insensitive substring match across fields |
| `folder_path` | `string \| null`  | Filter by folder                               |
| `pinned_repo` | `string \| null`  | Filter by pinned repository                    |

### Key API Behaviours

- **Update is full-replace**: The `PUT` endpoint uses the same `KnowledgeNoteCreateRequest` schema as `POST`. All fields must be sent — it is NOT a partial patch. This aligns perfectly with Terraform's model where the full desired state is always sent.
- **Note ID format**: Prefixed with `note-` (e.g., `note-abc123def456`).
- **Delete returns the deleted resource**: The DELETE response returns the full `KnowledgeNoteResponse` of the deleted note.
- **Pagination**: Cursor-based with `after` + `first` + `has_next_page` + `end_cursor`.

---

## 2. Provider Design

### Provider Configuration

```hcl
terraform {
  required_providers {
    devin = {
      source  = "cognition-ai/devin"
      version = "~> 0.1"
    }
  }
}

provider "devin" {
  api_key         = var.devin_api_key       # or env DEVIN_API_KEY
  organization_id = var.devin_org_id        # or env DEVIN_ORG_ID
  base_url        = "https://api.devin.ai"  # optional, for enterprise/self-hosted
}
```

### Provider Schema (Go)

| Attribute         | Type   | Required | Env Var Fallback | Description                              |
|-------------------|--------|----------|------------------|------------------------------------------|
| `api_key`         | String | Yes*     | `DEVIN_API_KEY`  | Service User API key (`cog_` prefix)     |
| `organization_id` | String | Yes*     | `DEVIN_ORG_ID`   | Organization ID to manage                |
| `base_url`        | String | No       | `DEVIN_BASE_URL` | API base URL (default: `https://api.devin.ai`) |

\* Required either in config or via environment variable.

### Technology Choices

| Choice                      | Rationale                                                     |
|-----------------------------|---------------------------------------------------------------|
| Go 1.22+                    | Standard for Terraform providers                              |
| Terraform Plugin Framework  | Recommended over SDKv2 for new providers — better types, protocol 6 |
| `net/http` (stdlib)         | No need for a heavy HTTP framework for 5 endpoints            |
| `encoding/json` (stdlib)    | Simple JSON marshaling                                        |

---

## 3. API Client Layer

### Client Structure

```go
// internal/client/client.go

type Client struct {
    BaseURL    string
    OrgID      string
    APIKey     string
    HTTPClient *http.Client
    UserAgent  string  // "terraform-provider-devin/0.1.0"
}

func NewClient(baseURL, orgID, apiKey string) *Client { ... }

// Generic request helper
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) { ... }
```

### Knowledge Notes Methods

```go
// internal/client/knowledge_notes.go

func (c *Client) CreateKnowledgeNote(ctx context.Context, req KnowledgeNoteCreateRequest) (*KnowledgeNote, error)
func (c *Client) GetKnowledgeNote(ctx context.Context, noteID string) (*KnowledgeNote, error)
func (c *Client) UpdateKnowledgeNote(ctx context.Context, noteID string, req KnowledgeNoteCreateRequest) (*KnowledgeNote, error)
func (c *Client) DeleteKnowledgeNote(ctx context.Context, noteID string) error
func (c *Client) ListKnowledgeNotes(ctx context.Context, opts ListKnowledgeNotesOptions) (*PaginatedResponse[KnowledgeNote], error)
```

### Error Handling

```go
// internal/client/errors.go

type APIError struct {
    StatusCode int
    Message    string
    Detail     []ValidationError  // from 422 responses
}

func (e *APIError) Error() string { ... }
func IsNotFound(err error) bool { ... }  // checks for 404
```

Key behaviours:
- **404 on Read** → return nil (Terraform interprets this as "resource was deleted externally" and removes it from state)
- **422 Validation Error** → surface the detail array in the Terraform error message
- **401/403** → clear error message about authentication/permissions
- **Rate limiting (429)** → retry with exponential backoff (3 retries, 1s/2s/4s)

---

## 4. Resource: `devin_knowledge_note`

### Terraform Schema

```go
// internal/provider/knowledge_note_resource.go

func (r *KnowledgeNoteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Manages a Devin Knowledge Note.",
        Attributes: map[string]schema.Attribute{
            // User-provided (required)
            "name": schema.StringAttribute{
                Required:    true,
                Description: "The name/title of the knowledge note.",
            },
            "body": schema.StringAttribute{
                Required:    true,
                Description: "The content of the knowledge note (supports markdown).",
            },
            "trigger": schema.StringAttribute{
                Required:    true,
                Description: "When this knowledge should be retrieved (the scope/trigger description).",
            },

            // User-provided (optional)
            "pinned_repo": schema.StringAttribute{
                Optional:    true,
                Description: "Pin this note to a specific repo (owner/repo format).",
            },

            // Computed (read-only, from API)
            "id": schema.StringAttribute{
                Computed:    true,
                Description: "The note ID (e.g., note-abc123def456).",
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "folder_path": schema.StringAttribute{
                Computed:    true,
                Description: "The folder path of the note.",
            },
            "is_enabled": schema.BoolAttribute{
                Computed:    true,
                Description: "Whether the note is enabled.",
            },
            "access_type": schema.StringAttribute{
                Computed:    true,
                Description: "Access type: 'org' or 'enterprise'.",
            },
            "created_at": schema.Int64Attribute{
                Computed:    true,
                Description: "Unix timestamp of creation.",
            },
            "updated_at": schema.Int64Attribute{
                Computed:    true,
                Description: "Unix timestamp of last update.",
            },
        },
    }
}
```

### CRUD Lifecycle Methods

#### Create

```
1. Read name, body, trigger, pinned_repo from plan
2. POST /v3/organizations/{org_id}/knowledge/notes
3. On success: set id = response.note_id, populate all computed fields into state
4. On error: return diagnostics with API error detail
```

#### Read

```
1. Get note_id from state (id attribute)
2. GET /v3/organizations/{org_id}/knowledge/notes/{note_id}
3. On 200: refresh all attributes in state from response
4. On 404: call resp.State.RemoveResource() — TF will plan a recreate
5. On other error: return diagnostics
```

#### Update

```
1. Read name, body, trigger, pinned_repo from plan
2. Get note_id from state
3. PUT /v3/organizations/{org_id}/knowledge/notes/{note_id}
4. On success: refresh all attributes in state from response
5. On 404: return error (shouldn't happen if Read passed)
6. On other error: return diagnostics
```

#### Delete

```
1. Get note_id from state
2. DELETE /v3/organizations/{org_id}/knowledge/notes/{note_id}
3. On success or 404: return (both mean the resource is gone)
4. On other error: return diagnostics
```

#### ImportState

```
1. Accept the note_id string as the import identifier
2. Set it as the "id" attribute
3. Terraform automatically calls Read to populate the rest of state
```

### Example Usage

```hcl
resource "devin_knowledge_note" "coding_standards" {
  name        = "Coding Standards"
  trigger     = "When writing or reviewing code in any repository"
  body        = file("knowledge/coding-standards.md")
  pinned_repo = "myorg/myrepo"
}

resource "devin_knowledge_note" "triage_playbook" {
  name    = "Triage Playbook"
  trigger = "When triaging alerts or investigating incidents"
  body    = <<-EOT
    ## Steps
    1. Check logs
    2. Identify root cause
    3. Create JIRA ticket
  EOT
}

output "coding_standards_note_id" {
  value = devin_knowledge_note.coding_standards.id
}
```

---

## 5. Data Sources

### `devin_knowledge_note` (singular)

Look up a single note by its ID.

```hcl
data "devin_knowledge_note" "existing" {
  note_id = "note-abc123def456"
}

output "note_body" {
  value = data.devin_knowledge_note.existing.body
}
```

### `devin_knowledge_notes` (plural)

List/search notes with optional filters.

```hcl
data "devin_knowledge_notes" "all" {
  search      = "coding"        # optional
  folder_path = "/standards"    # optional
  pinned_repo = "myorg/myrepo"  # optional
}

output "note_ids" {
  value = [for n in data.devin_knowledge_notes.all.notes : n.note_id]
}
```

Implementation note: the data source handles pagination internally, fetching all pages and returning the full list.

---

## 6. File Structure

```
terraform-provider-devin/
├── main.go                                        # Provider entry point (plugin server)
├── go.mod
├── go.sum
├── GNUmakefile                                    # build, test, install targets
├── .goreleaser.yml                                # Release automation
│
├── internal/
│   ├── client/
│   │   ├── client.go                              # HTTP client, auth, base request helper
│   │   ├── client_test.go                         # Unit tests for HTTP client (mocked)
│   │   ├── knowledge_notes.go                     # CRUD methods for knowledge notes
│   │   ├── knowledge_notes_test.go                # Unit tests with httptest server
│   │   ├── models.go                              # Go structs for API request/response
│   │   └── errors.go                              # APIError type, IsNotFound helper
│   │
│   └── provider/
│       ├── provider.go                            # Provider definition + resource/data source registry
│       ├── provider_test.go                       # Provider config validation tests
│       ├── knowledge_note_resource.go             # Resource: devin_knowledge_note
│       ├── knowledge_note_resource_test.go        # Acceptance tests for the resource
│       ├── knowledge_note_data_source.go          # Data source: devin_knowledge_note (singular)
│       ├── knowledge_note_data_source_test.go
│       ├── knowledge_notes_data_source.go         # Data source: devin_knowledge_notes (plural)
│       └── knowledge_notes_data_source_test.go
│
├── examples/
│   ├── provider/
│   │   └── main.tf                                # Minimal provider config example
│   └── resources/
│       └── devin_knowledge_note/
│           ├── main.tf                            # Basic resource example
│           └── import.sh                          # Import example
│
├── docs/                                          # Auto-generated by tfplugindocs
│   ├── index.md
│   ├── resources/
│   │   └── knowledge_note.md
│   └── data-sources/
│       ├── knowledge_note.md
│       └── knowledge_notes.md
│
└── .github/
    └── workflows/
        ├── test.yml                               # Unit + acceptance tests
        └── release.yml                            # GoReleaser publish
```

---

## 7. Implementation Phases

### Phase 1: Skeleton + Client (Est. ~1 hour)

| Step | Task | Deliverable |
|------|------|-------------|
| 1.1 | Initialize Go module, add Plugin Framework dependency | `go.mod` |
| 1.2 | Implement `internal/client/client.go` — HTTP client with auth | Client struct |
| 1.3 | Implement `internal/client/models.go` — request/response structs | Go types |
| 1.4 | Implement `internal/client/errors.go` — error types | Error helpers |
| 1.5 | Implement `internal/client/knowledge_notes.go` — 5 CRUD methods | API methods |
| 1.6 | Write unit tests for client with `httptest` mock server | `*_test.go` |
| 1.7 | Implement `internal/provider/provider.go` — provider config + schema | Provider |
| 1.8 | Wire up `main.go` entry point | Compilable binary |

**Exit criteria**: `go build` succeeds, `go test ./internal/client/...` passes.

### Phase 2: Resource Implementation (Est. ~1.5 hours)

| Step | Task | Deliverable |
|------|------|-------------|
| 2.1 | Implement `knowledge_note_resource.go` — Schema method | Schema definition |
| 2.2 | Implement Create method | POST + state write |
| 2.3 | Implement Read method (with 404 → remove from state) | GET + state refresh |
| 2.4 | Implement Update method | PUT + state refresh |
| 2.5 | Implement Delete method | DELETE |
| 2.6 | Implement ImportState method | Import by note_id |
| 2.7 | Register resource in provider.go | Resource available |

**Exit criteria**: `go build` succeeds, resource compiles and is registered.

### Phase 3: Data Sources (Est. ~45 min)

| Step | Task | Deliverable |
|------|------|-------------|
| 3.1 | Implement singular data source (`knowledge_note_data_source.go`) | Read by ID |
| 3.2 | Implement plural data source (`knowledge_notes_data_source.go`) | List with pagination |
| 3.3 | Register data sources in provider.go | Data sources available |

**Exit criteria**: `go build` succeeds, data sources compile.

### Phase 4: Testing (Est. ~2 hours)

| Step | Task | Deliverable |
|------|------|-------------|
| 4.1 | Write unit tests for client layer (mock HTTP) | `internal/client/*_test.go` |
| 4.2 | Write acceptance test: basic CRUD lifecycle | `TestAcc_basic` |
| 4.3 | Write acceptance test: with `pinned_repo` | `TestAcc_with_pinned_repo` |
| 4.4 | Write acceptance test: import state | `TestAcc_import` |
| 4.5 | Write acceptance test: resource disappears externally | `TestAcc_disappears` |
| 4.6 | Write acceptance tests for data sources | `TestAcc_data_*` |
| 4.7 | Write GNUmakefile with `test`, `testacc`, `build`, `install` targets | Makefile |

**Exit criteria**: `make test` (unit) passes. `make testacc` (acceptance) passes with valid credentials.

### Phase 5: Documentation + Examples (Est. ~30 min)

| Step | Task | Deliverable |
|------|------|-------------|
| 5.1 | Add description annotations to all schema attributes | In-code docs |
| 5.2 | Create `examples/` directory with working `.tf` files | Examples |
| 5.3 | Generate docs via `tfplugindocs generate` | `docs/` directory |
| 5.4 | Write README.md with quickstart | Root README |

**Exit criteria**: `tfplugindocs validate` passes. Examples are syntactically valid.

### Phase 6: CI/CD + Release (Est. ~30 min)

| Step | Task | Deliverable |
|------|------|-------------|
| 6.1 | Create `.github/workflows/test.yml` | CI pipeline |
| 6.2 | Create `.github/workflows/release.yml` with GoReleaser | Release pipeline |
| 6.3 | Create `.goreleaser.yml` | Release config |
| 6.4 | Tag `v0.1.0` and verify release | First release |

**Exit criteria**: CI passes on push. Tagged release produces binaries.

---

## 8. Testing Strategy

### Layer 1: Unit Tests (No API Calls)

**Scope**: Client HTTP logic, error handling, JSON serialization, provider config validation.

**Approach**: Use Go's `httptest.NewServer` to mock API responses.

```go
func TestCreateKnowledgeNote_Success(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "POST", r.Method)
        assert.Equal(t, "/v3/organizations/org-123/knowledge/notes", r.URL.Path)
        assert.Equal(t, "Bearer cog_test_key", r.Header.Get("Authorization"))

        var req KnowledgeNoteCreateRequest
        json.NewDecoder(r.Body).Decode(&req)
        assert.Equal(t, "test-note", req.Name)

        w.WriteHeader(200)
        json.NewEncoder(w).Encode(KnowledgeNote{NoteID: "note-abc123", Name: req.Name, ...})
    }))
    defer server.Close()

    client := NewClient(server.URL, "org-123", "cog_test_key")
    note, err := client.CreateKnowledgeNote(ctx, KnowledgeNoteCreateRequest{Name: "test-note", ...})
    assert.NoError(t, err)
    assert.Equal(t, "note-abc123", note.NoteID)
}
```

**Test cases**:

| Test | What it validates |
|------|-------------------|
| `TestCreateKnowledgeNote_Success` | Correct request, response parsing |
| `TestCreateKnowledgeNote_ValidationError` | 422 → `APIError` with detail |
| `TestGetKnowledgeNote_Success` | Read by ID works |
| `TestGetKnowledgeNote_NotFound` | 404 → `IsNotFound(err)` returns true |
| `TestUpdateKnowledgeNote_Success` | PUT with full body |
| `TestDeleteKnowledgeNote_Success` | DELETE returns no error |
| `TestDeleteKnowledgeNote_AlreadyDeleted` | 404 on delete → no error (idempotent) |
| `TestListKnowledgeNotes_Pagination` | Multi-page fetch, cursor handling |
| `TestClient_AuthHeader` | Bearer token sent correctly |
| `TestClient_UserAgent` | User-Agent header set |
| `TestClient_RateLimitRetry` | 429 → retries with backoff |

**Run with**: `go test ./internal/client/... -v`

### Layer 2: Acceptance Tests (Real API)

**Scope**: Full Terraform resource lifecycle against the live Devin API.

**Prerequisites**:
- `TF_ACC=1` environment variable
- `DEVIN_API_KEY` — service user key with `ManageAccountKnowledge` permission
- `DEVIN_ORG_ID` — target organization

**Test cases**:

```go
func TestAccKnowledgeNoteResource_basic(t *testing.T) {
    // Step 1: Create note, verify all attributes
    // Step 2: Update name + body, verify in-place update
    // Step 3: ImportState, verify all fields match
    // Implicit: Destroy after test (Terraform test framework handles this)
}

func TestAccKnowledgeNoteResource_withPinnedRepo(t *testing.T) {
    // Step 1: Create with pinned_repo set
    // Step 2: Remove pinned_repo (set to null)
    // Step 3: Re-add pinned_repo
}

func TestAccKnowledgeNoteResource_disappears(t *testing.T) {
    // Step 1: Create note
    // Between steps: delete note via API directly (simulating external deletion)
    // Step 2: Plan should show note needs to be recreated
}

func TestAccKnowledgeNoteResource_multiple(t *testing.T) {
    // Create 3 notes, verify they don't interfere with each other
    // Destroy all 3
}

func TestAccKnowledgeNoteDataSource_basic(t *testing.T) {
    // Create a note via resource, then read it back via data source
    // Verify all fields match
}

func TestAccKnowledgeNotesDataSource_search(t *testing.T) {
    // Create 2 notes with distinct names
    // Use data source with search filter
    // Verify only matching notes returned
}
```

**Cleanup strategy**: Each test creates notes with a unique random suffix (e.g., `tf-acc-test-abc123`) and uses `CheckDestroy` to verify the note was deleted after the test. A sweeper function can also be registered to clean up leaked test notes.

```go
func testAccCheckKnowledgeNoteDestroy(s *terraform.State) error {
    client := testAccProvider.Meta().(*client.Client)
    for _, rs := range s.RootModule().Resources {
        if rs.Type != "devin_knowledge_note" { continue }
        _, err := client.GetKnowledgeNote(ctx, rs.Primary.ID)
        if err == nil { return fmt.Errorf("note %s still exists", rs.Primary.ID) }
        if !client.IsNotFound(err) { return err }
    }
    return nil
}
```

**Run with**: `TF_ACC=1 go test ./internal/provider/... -v -timeout 30m`

### Layer 3: End-to-End Smoke Test

A shell script that exercises the full Terraform CLI workflow:

```bash
#!/bin/bash
set -euo pipefail

# Build and install
make install

# Init + Apply
cd examples/resources/devin_knowledge_note
terraform init
terraform plan -out=tfplan
terraform apply tfplan

# Verify state
NOTE_ID=$(terraform output -raw note_id)
echo "Created note: $NOTE_ID"

# Verify via curl
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $DEVIN_API_KEY" \
  "https://api.devin.ai/v3/organizations/$DEVIN_ORG_ID/knowledge/notes/$NOTE_ID")
[ "$STATUS" = "200" ] || { echo "FAIL: expected 200, got $STATUS"; exit 1; }

# Import test
terraform state rm devin_knowledge_note.example
terraform import devin_knowledge_note.example "$NOTE_ID"
terraform plan -detailed-exitcode  # exit code 0 = no changes (import was clean)

# Destroy
terraform destroy -auto-approve

# Verify deletion
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $DEVIN_API_KEY" \
  "https://api.devin.ai/v3/organizations/$DEVIN_ORG_ID/knowledge/notes/$NOTE_ID")
[ "$STATUS" = "404" ] || { echo "FAIL: expected 404, got $STATUS"; exit 1; }

echo "PASS: smoke test complete"
```

### Testing Summary

| Layer | Calls Real API? | Runs When | What It Catches | Est. Runtime |
|-------|----------------|-----------|-----------------|--------------|
| Unit tests | No (httptest mock) | Every PR / every `go test` | JSON bugs, nil panics, auth header, retry logic, error parsing | ~5 seconds |
| Acceptance tests | Yes | Merge to main, or manual `TF_ACC=1` | API contract drift, real CRUD, import, drift detection | ~2–3 minutes |
| Smoke test | Yes | Pre-release / manual | Full CLI workflow, binary packaging, real-world usage | ~1 minute |

---

## 9. CI/CD Pipeline

### Test Workflow (`.github/workflows/test.yml`)

```yaml
name: Tests
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go vet ./...
      - uses: golangci/golangci-lint-action@v4

  unit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go test ./internal/client/... -v -race -count=1

  acceptance:
    runs-on: ubuntu-latest
    needs: [lint, unit]
    if: github.ref == 'refs/heads/main'  # Only on main to conserve API calls
    env:
      TF_ACC: "1"
      DEVIN_API_KEY: ${{ secrets.DEVIN_API_KEY }}
      DEVIN_ORG_ID: ${{ secrets.DEVIN_ORG_ID }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go test ./internal/provider/... -v -timeout 30m -count=1
```

### Release Workflow (`.github/workflows/release.yml`)

```yaml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - uses: goreleaser/goreleaser-action@v5
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ secrets.GPG_FINGERPRINT }}  # For Terraform Registry signing
```

---

## 10. Future Expansion

After `devin_knowledge_note` is stable, the provider can grow incrementally:

| Priority | Resource | CRUD | Complexity |
|----------|----------|------|------------|
| **P1** | `devin_playbook` | Full CRUD | Low — nearly identical pattern to knowledge notes |
| **P2** | `devin_secret` | Create/List/Delete | Medium — no update, no read-value (write-only) |
| **P3** | `devin_schedule` | Full CRUD | Medium — references playbooks, cron expressions |
| **P4** | `devin_repository_index` | Idempotent PUT/DELETE | Medium — different CRUD pattern |
| **P5** | `devin_organization` | Full CRUD (enterprise) | High — enterprise-only, different base URL |

Each new resource follows the same pattern:
1. Add client methods to `internal/client/`
2. Add resource/data source to `internal/provider/`
3. Add acceptance tests
4. Generate docs

The client layer and provider framework established in Phase 1 are reusable for all future resources.

---

## Appendix: Design Decisions

| Decision | Rationale |
|----------|-----------|
| Plugin Framework over SDKv2 | SDKv2 is in maintenance mode; Framework is the future and supports protocol 6 |
| Single `organization_id` in provider | Keeps things simple; multi-org can be done with provider aliases |
| `id` attribute = `note_id` | Standard Terraform convention; enables `terraform import` |
| Delete treats 404 as success | Idempotent deletion — already gone is fine |
| Full-replace PUT (not PATCH) | Matches the API semantics and Terraform's "desired state" model perfectly |
| Paginated list in data source | Users shouldn't need to think about pagination; fetch all pages internally |
| Acceptance tests on main only | Avoids burning API rate limits on every PR push |
| Random suffixes on test resources | Prevents name collisions if tests run in parallel |
