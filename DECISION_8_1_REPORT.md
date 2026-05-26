# Decision 8.1: Initial Release Scope for Terraform Provider

## Executive Summary

This report catalogs the entire Devin API v3 surface, classifies each endpoint group by Terraform resource type and implementation complexity, maps dependencies, and proposes three release scopes (Minimal, Core, Full) with effort estimates. The recommendation is **Scope B (Core)** — shipping Knowledge Notes, Playbooks, and Schedules in v0.1.0.

---

## 1. Full API Surface Catalog

### 1.1 Organization-Scope Endpoints (primary Terraform target)

#### Knowledge Notes

| # | Method | Path | Terraform Type | Complexity |
|---|--------|------|----------------|------------|
| 1 | `GET` | `/v3/organizations/{org_id}/knowledge/notes` | Data Source (`devin_knowledge_notes`) | Simple |
| 2 | `POST` | `/v3/organizations/{org_id}/knowledge/notes` | Resource (Create) | Simple |
| 3 | `GET` | `/v3/organizations/{org_id}/knowledge/notes/{note_id}` | Resource (Read) | Simple |
| 4 | `PUT` | `/v3/organizations/{org_id}/knowledge/notes/{note_id}` | Resource (Update) | Simple |
| 5 | `DELETE` | `/v3/organizations/{org_id}/knowledge/notes/{note_id}` | Resource (Delete) | Simple |
| 6 | `GET` | `/v3/organizations/{org_id}/knowledge/folders` | Data Source (`devin_knowledge_folders`) | Simple |

**Resource:** `devin_knowledge_note` — Standard CRUD. Fields: `name`, `body`, `trigger`, `pinned_repo`, `is_enabled`, `folder_path`, `macro`.
**Data Sources:** `devin_knowledge_notes` (list with search/filter), `devin_knowledge_folders` (folder tree).
**Estimated hours:** 8h (resource already scaffolded, needs real API client wiring + tests)

#### Playbooks

| # | Method | Path | Terraform Type | Complexity |
|---|--------|------|----------------|------------|
| 7 | `GET` | `/v3/organizations/{org_id}/playbooks` | Data Source (`devin_playbooks`) | Simple |
| 8 | `POST` | `/v3/organizations/{org_id}/playbooks` | Resource (Create) | Simple |
| 9 | `GET` | `/v3/organizations/{org_id}/playbooks/{playbook_id}` | Resource (Read) | Simple |
| 10 | `PUT` | `/v3/organizations/{org_id}/playbooks/{playbook_id}` | Resource (Update) | Simple |
| 11 | `DELETE` | `/v3/organizations/{org_id}/playbooks/{playbook_id}` | Resource (Delete) | Simple |

**Resource:** `devin_playbook` — Standard CRUD. Fields: `title`, `body`, `macro`.
**Data Source:** `devin_playbooks` (paginated list).
**Estimated hours:** 12h (new resource from scratch, similar pattern to knowledge notes)

#### Schedules

| # | Method | Path | Terraform Type | Complexity |
|---|--------|------|----------------|------------|
| 12 | `GET` | `/v3/organizations/{org_id}/schedules` | Data Source (`devin_schedules`) | Medium |
| 13 | `POST` | `/v3/organizations/{org_id}/schedules` | Resource (Create) | Medium |
| 14 | `GET` | `/v3/organizations/{org_id}/schedules/{schedule_id}` | Resource (Read) | Medium |
| 15 | `PATCH` | `/v3/organizations/{org_id}/schedules/{schedule_id}` | Resource (Update) | Medium |
| 16 | `DELETE` | `/v3/organizations/{org_id}/schedules/{schedule_id}` | Resource (Delete) | Medium |

**Resource:** `devin_schedule` — CRUD with conditional logic. Fields: `name`, `prompt`, `frequency` (cron), `schedule_type` (recurring vs one_time), `scheduled_at`, `agent` (devin/data_analyst), `playbook_id`, `notify_on`, `bypass_approval`, `tags`, `slack_channel_id`, `slack_team_id`, `target_devin_id`, `create_as_user_id`, `interval_count`. Computed: `enabled`, `consecutive_failures`, `last_executed_at`, `last_error_at`, `last_error_message`.
**Complexity rationale:** Conditional required fields (cron vs scheduled_at), enum validation, optional playbook reference, Slack integration fields, cron expression validation.
**Estimated hours:** 20h

#### Secrets

| # | Method | Path | Terraform Type | Complexity |
|---|--------|------|----------------|------------|
| 17 | `GET` | `/v3/organizations/{org_id}/secrets` | Data Source (`devin_secrets`) | Medium |
| 18 | `POST` | `/v3/organizations/{org_id}/secrets` | Resource (Create) | Medium |
| 19 | `DELETE` | `/v3/organizations/{org_id}/secrets/{secret_id}` | Resource (Delete) | Medium |

**Resource:** `devin_secret` — Create + Delete only (no Read of value, no Update). Fields: `type` (cookie/key-value/totp), `key`, `value` (write-only/sensitive), `is_sensitive`, `note`.
**Complexity rationale:** Write-only `value` field (never returned by API, must use `ignore_changes` pattern or ForceNew). No GET-by-ID endpoint visible — may need to match by key from list endpoint. No Update endpoint — secret must be recreated on change (ForceNew on all fields).
**Estimated hours:** 16h (write-only pattern is non-trivial)

#### Sessions

| # | Method | Path | Terraform Type | Complexity |
|---|--------|------|----------------|------------|
| 20 | `GET` | `/v3/organizations/{org_id}/sessions` | Data Source (`devin_sessions`) | Medium |
| 21 | `POST` | `/v3/organizations/{org_id}/sessions` | Resource (Create) | Complex |
| 22 | `GET` | `/v3/organizations/{org_id}/sessions/{session_id}` | Resource (Read) | Medium |
| 23 | `DELETE` | `/v3/organizations/{org_id}/sessions/{session_id}` | Resource (Delete/Terminate) | Medium |
| 24 | `POST` | `/v3/organizations/{org_id}/sessions/{session_id}/messages` | N/A (ephemeral action) | Complex |
| 25 | `GET` | `/v3/organizations/{org_id}/sessions/{session_id}/messages` | Data Source (`devin_session_messages`) | Medium |
| 26 | `GET` | `/v3/organizations/{org_id}/sessions/{session_id}/attachments` | Data Source (`devin_session_attachments`) | Simple |
| 27 | `GET` | `/v3/organizations/{org_id}/sessions/{session_id}/insights` | Data Source (`devin_session_insights`) | Simple |
| 28 | `POST` | `/v3/organizations/{org_id}/sessions/{session_id}/insights/generate` | N/A (trigger action) | Medium |
| 29 | `GET` | `/v3/organizations/{org_id}/sessions/{session_id}/tags` | Part of session resource | Simple |
| 30 | `POST` | `/v3/organizations/{org_id}/sessions/{session_id}/tags` | Part of session resource | Simple |
| 31 | `PUT` | `/v3/organizations/{org_id}/sessions/{session_id}/tags` | Part of session resource | Simple |
| 32 | `POST` | `/v3/organizations/{org_id}/sessions/archive` | N/A (action) | Simple |
| 33 | `GET` | `/v3/organizations/{org_id}/sessions?insights=true` | Data Source variant | Medium |

**Resource:** `devin_session` — Lifecycle management. Create-only with terminate on destroy. Many optional fields: `prompt`, `repos`, `playbook_id`, `secret_ids`, `knowledge_ids`, `attachment_urls`, `structured_output_schema`, `devin_mode`, `platform`, `create_as_user_id`, `tags`, `max_acu_limit`, `bypass_approval`, `session_links`, `session_secrets`.
**Complexity rationale:** Async operation (session starts running after create), lifecycle management (running → suspended → terminated), many optional parameters, structured output schema (arbitrary JSON), session secrets (inline sensitive data). Tags are sub-resource operations.
**Estimated hours:** 32h

#### Attachments

| # | Method | Path | Terraform Type | Complexity |
|---|--------|------|----------------|------------|
| 34 | `POST` | `/v3/organizations/{org_id}/attachments` | Resource (Create) | Medium |
| 35 | `GET` | `/v3/organizations/{org_id}/attachments/{attachment_id}` | Data Source (Download redirect) | Medium |

**Resource:** `devin_attachment` — Upload-only (multipart/form-data). No update, no list. Download returns presigned URL redirect.
**Complexity rationale:** File upload via multipart, no standard CRUD lifecycle, presigned URL handling.
**Estimated hours:** 12h

#### PR Reviews (Devin Review)

| # | Method | Path | Terraform Type | Complexity |
|---|--------|------|----------------|------------|
| 36 | `POST` | `/v3/organizations/{org_id}/pr-reviews` | Resource (trigger) | Medium |
| 37 | `GET` | `/v3/organizations/{org_id}/pr-reviews` | Data Source (status) | Simple |

**Resource:** `devin_pr_review` — Trigger-only. Not a traditional CRUD resource; creates a review job that transitions through states (pending → running → completed/errored).
**Complexity rationale:** Async state machine, no update/delete, polling for completion.
**Estimated hours:** 12h

#### Repositories

| # | Method | Path | Terraform Type | Complexity |
|---|--------|------|----------------|------------|
| 38 | `GET` | `/v3beta1/organizations/{org_id}/repositories` | Data Source (`devin_repositories`) | Simple |
| 39 | `GET` | `/v3beta1/organizations/{org_id}/repositories/indexed` | Data Source (`devin_indexed_repositories`) | Simple |
| 40 | `PUT` | `/v3beta1/organizations/{org_id}/repositories/{path}/indexing` | Resource (idempotent) | Medium |
| 41 | `DELETE` | `/v3beta1/organizations/{org_id}/repositories/{path}` | Resource (remove indexing) | Simple |
| 42 | `DELETE` | `/v3beta1/organizations/{org_id}/repositories/{path}/branches` | Sub-operation | Simple |
| 43 | `PUT` | `/v3beta1/organizations/{org_id}/repositories/bulk-index` | N/A (bulk action) | Medium |
| 44 | `DELETE` | `/v3beta1/organizations/{org_id}/repositories/bulk-remove` | N/A (bulk action) | Medium |

**Resource:** `devin_repository_indexing` — Idempotent PUT to enable indexing, DELETE to remove. Fields: `repository_path`, `branch_names`.
**Complexity rationale:** Beta API (v3beta1), idempotent PUT (not POST), async indexing jobs, nested status response.
**Note:** Beta status means API may change.
**Estimated hours:** 16h

#### Snapshot Setup (Blueprints & Builds)

| # | Method | Path | Terraform Type | Complexity |
|---|--------|------|----------------|------------|
| 45 | `GET` | `/v3beta1/organizations/{org_id}/snapshot-setup/blueprints` | Data Source | Simple |
| 46 | `POST` | `/v3beta1/organizations/{org_id}/snapshot-setup/blueprints` | Resource (Create) | Medium |
| 47 | `GET` | `/v3beta1/organizations/{org_id}/snapshot-setup/blueprints/{id}` | Resource (Read) | Simple |
| 48 | `PATCH` | `/v3beta1/organizations/{org_id}/snapshot-setup/blueprints/{id}` | Resource (Update) | Medium |
| 49 | `DELETE` | `/v3beta1/organizations/{org_id}/snapshot-setup/blueprints/{id}` | Resource (Delete) | Simple |
| 50 | `GET` | `/v3beta1/organizations/{org_id}/snapshot-setup/blueprints/{id}/contents` | Sub-operation (presigned URL) | Medium |
| 51 | `POST` | `/v3beta1/organizations/{org_id}/snapshot-setup/blueprints/{id}/files` | Sub-operation (file upload) | Medium |
| 52 | `GET` | `/v3beta1/organizations/{org_id}/snapshot-setup/blueprints/{id}/files` | Sub-operation (list files) | Simple |
| 53 | `DELETE` | `/v3beta1/organizations/{org_id}/snapshot-setup/blueprints/{id}/files/{file_id}` | Sub-operation | Simple |
| 54 | `GET` | `/v3beta1/organizations/{org_id}/snapshot-setup/builds` | Data Source (`devin_builds`) | Simple |
| 55 | `POST` | `/v3beta1/organizations/{org_id}/snapshot-setup/builds` | Resource (trigger) | Medium |
| 56 | `GET` | `/v3beta1/organizations/{org_id}/snapshot-setup/builds/{id}` | Data Source | Simple |
| 57 | `GET` | `/v3beta1/organizations/{org_id}/snapshot-setup/builds/{id}/logs` | Data Source (presigned URL) | Simple |
| 58 | `POST` | `/v3beta1/organizations/{org_id}/snapshot-setup/builds/{id}/cancel` | Action | Simple |
| 59 | `POST` | `/v3beta1/organizations/{org_id}/snapshot-setup/builds/{id}/pin` | Action | Simple |
| 60 | `DELETE` | `/v3beta1/organizations/{org_id}/snapshot-setup/builds/{id}/pin` | Action | Simple |

**Resources:** `devin_blueprint` (CRUD with YAML contents + file attachments), `devin_build` (trigger + track).
**Complexity rationale:** Beta API, YAML content fetched via separate presigned URL endpoint, file attachments are sub-resources, builds are async with status tracking, pinning is a separate operation.
**Estimated hours:** 32h

#### Git Permissions

| # | Method | Path | Terraform Type | Complexity |
|---|--------|------|----------------|------------|
| 61 | `GET` | `/v3/organizations/{org_id}/git-providers/permissions` | Data Source | Simple |
| 62 | `POST` | `/v3/organizations/{org_id}/git-providers/permissions` | Resource (Create) | Simple |
| 63 | `DELETE` | `/v3/organizations/{org_id}/git-providers/permissions/{id}` | Resource (Delete) | Simple |
| 64 | `PUT` | `/v3/organizations/{org_id}/git-providers/permissions` | Resource (Replace all) | Medium |
| 65 | `DELETE` | `/v3/organizations/{org_id}/git-providers/permissions` (clear all) | Action | Simple |

**Resource:** `devin_git_permission` — Create + Delete (no Update per-item). Full replace via PUT.
**Estimated hours:** 12h

#### Tags

| # | Method | Path | Terraform Type | Complexity |
|---|--------|------|----------------|------------|
| 66 | `GET` | `/v3/organizations/{org_id}/tags` | Data Source | Simple |
| 67 | `POST` | `/v3/organizations/{org_id}/tags` | Resource (append) | Simple |
| 68 | `PUT` | `/v3/organizations/{org_id}/tags` | Resource (replace) | Simple |
| 69 | `DELETE` | `/v3/organizations/{org_id}/tags` | Resource (clear) | Simple |
| 70 | `DELETE` | `/v3/organizations/{org_id}/tags/{tag}` | Resource (remove one) | Simple |

**Resource:** `devin_organization_tags` — Set-based resource (manage the full set of allowed tags). Also `devin_organization_default_tag` (enterprise-scope).
**Estimated hours:** 8h

### 1.2 Enterprise-Scope Endpoints

These endpoints operate at the enterprise level (`/v3/enterprise/*`) and are relevant for enterprise customers managing multiple organizations.

#### Enterprise Knowledge Notes
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/knowledge/notes` | Simple |
| `POST` | `/v3/enterprise/knowledge/notes` | Simple |
| `GET/PUT/DELETE` | `/v3/enterprise/knowledge/notes/{note_id}` | Simple |
| `GET` | `/v3/enterprise/knowledge/folders` | Simple |

Mirror of org-level notes. **Resource:** `devin_enterprise_knowledge_note`. **Hours:** 6h (reuse org client code)

#### Enterprise Playbooks
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/playbooks` | Simple |
| `POST` | `/v3/enterprise/playbooks` | Simple |
| `GET/PUT/DELETE` | `/v3/enterprise/playbooks/{playbook_id}` | Simple |

Mirror of org-level. **Resource:** `devin_enterprise_playbook`. **Hours:** 6h

#### Enterprise Sessions
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/sessions` | Medium |
| `GET` | `/v3/enterprise/sessions?insights=true` | Medium |
| `GET` | `/v3/enterprise/sessions/{session_id}` | Medium |
| `GET` | `/v3/enterprise/sessions/{session_id}/messages` | Medium |
| `GET` | `/v3/enterprise/sessions/{session_id}/attachments` | Simple |
| `GET` | `/v3/enterprise/sessions/{session_id}/insights` | Simple |
| `POST` | `/v3/enterprise/sessions/{session_id}/insights/generate` | Medium |
| `GET/POST/PUT` | `/v3/enterprise/sessions/{session_id}/tags` | Simple |

Read-only views + tag management. **Data Sources only** (sessions are created at org level). **Hours:** 12h

#### Enterprise Snapshot Setup
| Method | Path | Complexity |
|--------|------|------------|
| `GET/POST` | `/v3/enterprise/snapshot-setup/blueprints` | Medium |
| `GET/PUT/DELETE` | `/v3/enterprise/snapshot-setup/blueprints/{id}` | Medium |
| `GET/POST/DELETE` | `/v3/enterprise/snapshot-setup/blueprints/{id}/files` | Medium |
| `GET` | `/v3/enterprise/snapshot-setup/blueprints/{id}/contents` | Medium |

**Resource:** `devin_enterprise_blueprint`. **Hours:** 16h

#### Enterprise PR Reviews (Devin Review)
| Method | Path | Complexity |
|--------|------|------------|
| `POST` | `/v3/enterprise/pr-reviews` | Medium |
| `GET` | `/v3/enterprise/pr-reviews` | Simple |

**Hours:** 6h (reuse org pattern)

#### Audit Logs
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/audit-logs` | Medium |
| `GET` | `/v3/organizations/{org_id}/audit-logs` | Medium |

**Data Source only** (read-only, paginated, filterable). **Hours:** 8h

#### Guardrail Violations (Beta)
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/guardrail-violations` | Medium |
| `GET` | `/v3/organizations/{org_id}/guardrail-violations` | Medium |

**Data Source only**. Beta status. **Hours:** 6h

#### Consumption
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/consumption/cycles` | Simple |
| `GET` | `/v3/enterprise/consumption/daily` | Medium |
| `GET` | `/v3/enterprise/consumption/daily/organizations/{id}` | Medium |
| `GET` | `/v3/enterprise/consumption/daily/users/{id}` | Medium |
| `GET` | `/v3/enterprise/consumption/daily/service-users/{id}` | Medium |
| `GET` | `/v3/enterprise/consumption/daily/sessions/{id}` | Medium |
| `GET` | `/v3/organizations/{org_id}/consumption/daily` | Medium |
| `GET` | `/v3/organizations/{org_id}/consumption/daily/users/{id}` | Medium |
| `GET` | `/v3/organizations/{org_id}/consumption/daily/service-users/{id}` | Medium |
| `GET` | `/v3/organizations/{org_id}/consumption/daily/sessions/{id}` | Medium |

**Data Sources only** (analytics, no mutations). **Hours:** 12h

#### Metrics
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/metrics/active-users` | Simple |
| `GET` | `/v3/enterprise/metrics/dau` | Simple |
| `GET` | `/v3/enterprise/metrics/wau` | Simple |
| `GET` | `/v3/enterprise/metrics/mau` | Simple |
| `GET` | `/v3/enterprise/metrics/sessions` | Simple |
| `GET` | `/v3/enterprise/metrics/sessions-by-category` | Simple |
| `GET` | `/v3/enterprise/metrics/prs` | Simple |
| `GET` | `/v3/enterprise/metrics/searches` | Simple |
| `GET` | `/v3/enterprise/metrics/usage` | Simple |
| `GET` | `/v3/organizations/{org_id}/metrics/*` (8 endpoints) | Simple |

**Data Sources only**. **Hours:** 10h

#### Queue
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/queue` | Simple |

**Data Source only**. **Hours:** 2h

#### Hypervisors
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/hypervisors` | Simple |

**Data Source only**. **Hours:** 2h

#### Git Connections (Enterprise)
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/git-providers/connections` | Simple |
| `GET` | `/v3/enterprise/git-providers/connections/{id}/repositories` | Simple |

**Data Sources only**. **Hours:** 4h

#### Organizations (Enterprise)
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/organizations` | Simple |
| `POST` | `/v3/enterprise/organizations` | Medium |
| `GET` | `/v3/enterprise/organizations/{id}` | Simple |
| `PATCH` | `/v3/enterprise/organizations/{id}` | Simple |
| `DELETE` | `/v3/enterprise/organizations/{id}` | Medium |

**Resource:** `devin_organization`. **Hours:** 16h

#### Users & Membership (Enterprise)
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/members/users` | Medium |
| `POST` | `/v3/enterprise/members/users` | Medium |
| `GET` | `/v3/enterprise/members/users/{id}` | Simple |
| `PATCH` | `/v3/enterprise/members/users/{id}` | Simple |
| `DELETE` | `/v3/enterprise/members/users/{id}` | Medium |
| `GET` | `/v3/enterprise/members/idp-users` | Simple |
| `GET/POST/DELETE/PATCH` | `/v3/organizations/{org_id}/members/users/*` | Medium |
| `GET` | `/v3/organizations/{org_id}/members/idp-users` | Simple |

**Resources:** `devin_user`, `devin_organization_membership`. **Hours:** 24h

#### Roles (Enterprise)
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/roles` | Simple |

**Data Source only**. **Hours:** 2h

#### Service Users (Enterprise)
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/members/service-users` | Medium |
| `POST` | `/v3/enterprise/members/service-users` | Medium |
| `GET` | `/v3/enterprise/members/service-users/{id}` | Simple |
| `PATCH` | `/v3/enterprise/members/service-users/{id}` | Simple |
| `DELETE` | `/v3/enterprise/members/service-users/{id}` | Medium |
| `GET/POST/PATCH/DELETE` | `/v3/organizations/{org_id}/members/service-users/*` | Medium |

**Resource:** `devin_service_user`, `devin_organization_service_user`. **Hours:** 20h

#### Service User API Keys (Enterprise, Beta)
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/members/service-users/{id}/api-keys` | Simple |
| `POST` | `/v3/enterprise/members/service-users/{id}/api-keys` | Complex |
| `DELETE` | `/v3/enterprise/members/service-users/{id}/api-keys/{key_id}` | Simple |
| `POST` | `/v3/enterprise/members/service-users/{id}/api-keys/{key_id}/rotate` | Complex |

**Resource:** `devin_service_user_api_key` — Write-only pattern (key value only returned on create/rotate). Rotate is a special lifecycle operation.
**Hours:** 16h

#### IDP Groups (Enterprise)
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/idp-groups` | Simple |
| `POST` | `/v3/enterprise/idp-groups` | Medium |
| `DELETE` | `/v3/enterprise/idp-groups/{id}` | Simple |
| `GET/POST/PATCH/DELETE` | `/v3/enterprise/members/idp-groups/*` | Medium |
| `GET/POST/PATCH/DELETE` | `/v3/organizations/{org_id}/members/idp-groups/*` | Medium |

**Resources:** `devin_idp_group`, `devin_idp_group_assignment`, `devin_organization_idp_group`. **Hours:** 20h

#### IP Access List (Enterprise)
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/ip-access-list` | Simple |
| `PUT` | `/v3/enterprise/ip-access-list` | Simple |
| `DELETE` | `/v3/enterprise/ip-access-list` | Simple |

**Resource:** `devin_ip_access_list` — Set-based (replace entire list). **Hours:** 6h

#### Org Group Limits (Enterprise)
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/enterprise/org-group-limits` | Simple |
| `PUT` | `/v3/enterprise/org-group-limits` | Simple |

**Resource:** `devin_org_group_limits`. **Hours:** 4h

#### Self
| Method | Path | Complexity |
|--------|------|------------|
| `GET` | `/v3/self` | Simple |

**Data Source:** `devin_self` — Returns info about authenticated API key. **Hours:** 2h

---

## 2. Complexity Summary

| Complexity | Count | Description |
|------------|-------|-------------|
| **Simple** | ~85 endpoints | Standard CRUD, single GET, no special lifecycle |
| **Medium** | ~45 endpoints | Pagination, conditional fields, enum validation, write-only patterns |
| **Complex** | ~8 endpoints | Async operations, lifecycle management, file uploads, state machines |

### Complexity Definitions

- **Simple:** Standard CRUD mapping. No special Terraform patterns needed. ~4-8h per resource including tests and docs.
- **Medium:** Requires custom validation, conditional logic, pagination helpers, or write-only field patterns. ~12-20h per resource.
- **Complex:** Lifecycle management (async operations, polling), file uploads (multipart), state machines (session states), or write-only secrets. ~24-32h per resource.

---

## 3. Dependency Graph

```
devin_self (standalone)
  │
  ├── devin_knowledge_note (standalone)
  │     └── used by: devin_session (knowledge_ids)
  │
  ├── devin_playbook (standalone)
  │     ├── used by: devin_schedule (playbook_id)
  │     └── used by: devin_session (playbook_id)
  │
  ├── devin_secret (standalone)
  │     └── used by: devin_session (secret_ids)
  │
  ├── devin_schedule ──depends on──▶ devin_playbook (optional)
  │
  ├── devin_session ──depends on──▶ devin_knowledge_note (optional)
  │                 ──depends on──▶ devin_playbook (optional)
  │                 ──depends on──▶ devin_secret (optional)
  │                 ──depends on──▶ devin_attachment (optional)
  │
  ├── devin_blueprint (standalone, beta)
  │     └── devin_build ──depends on──▶ devin_blueprint
  │
  ├── devin_repository_indexing (standalone, beta)
  │
  ├── devin_pr_review (standalone)
  │
  ├── devin_organization (enterprise, standalone)
  │     ├── devin_user ──depends on──▶ devin_organization (for membership)
  │     ├── devin_service_user ──depends on──▶ devin_organization (for assignment)
  │     └── devin_idp_group ──depends on──▶ devin_organization (for assignment)
  │
  └── devin_organization_tags (standalone per org)
```

### Shared Infrastructure Requirements

| Component | Used By | Priority |
|-----------|---------|----------|
| **HTTP client** (auth, base URL, error handling) | All resources | P0 — build first |
| **Cursor-based pagination helper** | All list/data-source endpoints | P0 — build first |
| **Error response parser** (HTTPValidationError) | All resources | P0 — build first |
| **Write-only field pattern** (sensitive, ForceNew) | Secrets, API Keys | P1 |
| **Async operation poller** (status polling) | Sessions, Builds, PR Reviews | P2 |
| **File upload helper** (multipart/form-data) | Attachments, Blueprint files | P2 |
| **Org ID configuration** (provider-level) | All org-scope resources | P0 |

---

## 4. Release Scope Proposals

### Scope A — Minimal (v0.1.0): Knowledge Notes Only

**What ships:** Complete the existing skeleton — wire the `devin_knowledge_note` resource and `devin_knowledge_notes` data source to the real API.

| Metric | Value |
|--------|-------|
| Resources | 1 (`devin_knowledge_note`) |
| Data Sources | 2 (`devin_knowledge_notes`, `devin_knowledge_folders`) |
| New LoC (est.) | ~800 (client + resource + tests) |
| Implementation time | **0.5 person-weeks** |
| Acceptance tests | 5-8 (CRUD + import + data source) |
| Doc pages | 3 (provider index, 1 resource, 1 data source) |

**Pros:**
- Ship fast, validate the provider architecture end-to-end
- Knowledge notes are the most natural IaC fit (declarative config)
- Already scaffolded, lowest risk

**Cons:**
- Very thin value proposition — users can't do much with just knowledge notes
- Doesn't validate medium-complexity patterns (pagination, conditional fields)

### Scope B — Core (v0.1.0): Knowledge + Playbooks + Schedules

**What ships:** The three resources that form the "automation triad" — define knowledge, create playbooks that reference that knowledge, and schedule sessions that execute those playbooks.

| Metric | Value |
|--------|-------|
| Resources | 3 (`devin_knowledge_note`, `devin_playbook`, `devin_schedule`) |
| Data Sources | 4 (`devin_knowledge_notes`, `devin_knowledge_folders`, `devin_playbooks`, `devin_schedules`) |
| New LoC (est.) | ~2,400 (client + 3 resources + tests) |
| Implementation time | **2 person-weeks** |
| Acceptance tests | 15-20 |
| Doc pages | 8 (provider index, 3 resources, 4 data sources) |

**Pros:**
- Complete workflow: knowledge → playbook → schedule is a meaningful IaC story
- Validates Simple + Medium complexity patterns
- Tests pagination helper, conditional fields, enum validation, optional references
- Enough surface area to prove the provider to early adopters
- Schedule → playbook dependency validates cross-resource references

**Cons:**
- Doesn't include secrets (common request for IaC)
- Doesn't include sessions (the core Devin primitive)
- 2 weeks is a meaningful commitment for v0.1.0

### Scope C — Full Org-Scope (v0.1.0): All Organization Resources

**What ships:** Every resource and data source that operates at the organization level.

| Metric | Value |
|--------|-------|
| Resources | 9+ (`knowledge_note`, `playbook`, `schedule`, `secret`, `session`, `attachment`, `pr_review`, `repository_indexing`, `blueprint`, `build`, `git_permission`, `organization_tags`) |
| Data Sources | 12+ (list endpoints for all above + `self`, `audit_logs`, `consumption`, `metrics`) |
| New LoC (est.) | ~6,500 |
| Implementation time | **6-7 person-weeks** |
| Acceptance tests | 50-60 |
| Doc pages | 25+ |

**Pros:**
- Comprehensive — users can manage their entire Devin org via Terraform
- Validates all complexity patterns (write-only, async, file upload, lifecycle)
- Strong competitive positioning

**Cons:**
- Very long time to first release — 6-7 weeks before any user can try the provider
- Beta APIs (snapshot-setup, repositories) may change, causing breaking changes in v0.1.0
- Sessions resource is Complex and may not fit well in Terraform's declarative model
- Attachments (file upload) and PR Reviews (trigger-only) are unusual Terraform patterns
- Risk of scope creep into enterprise endpoints

---

## 5. Industry Comparison: How Similar Providers Scoped Initial Releases

### terraform-provider-pagerduty

| Version | Date | Resources | Notes |
|---------|------|-----------|-------|
| v0.1.0 | Jun 2017 | ~8 | Provider split from Terraform core. Shipped: `pagerduty_user`, `pagerduty_team`, `pagerduty_service`, `pagerduty_escalation_policy`, `pagerduty_schedule`, `pagerduty_service_integration`, + data sources for `vendor`, `escalation_policy` |
| v0.1.1 | Aug 2017 | +2 | Added `pagerduty_team_membership`, `pagerduty_maintenance_window` |
| v1.0.0 | Mar 2018 | +2 | Added `pagerduty_extension`, `pagerduty_extension_schema` data source. 9 months after v0.1.0 |

**Key insight:** PagerDuty shipped ~8 core resources in v0.1.0, covering the primary workflow (user → team → escalation policy → service → schedule). This maps to our Scope B approach.

### terraform-provider-datadog

| Version | Date | Resources | Notes |
|---------|------|-----------|-------|
| v0.1.0 | Jun 2017 | ~3 | Provider split from Terraform core. Shipped: `datadog_monitor`, `datadog_timeboard`, `datadog_user` |
| v0.1.1 | Sep 2017 | +1 | Added `datadog_metric_metadata` |
| v1.0.0 | N/A | — | Jumped from v0.x directly to v1.4.0 in Oct 2018. Added screenboards, dashboards |

**Key insight:** Datadog started with just 3 resources — the absolute core primitives (monitors, dashboards, users). Even more minimal than our Scope A.

### terraform-provider-github

| Version | Date | Resources | Notes |
|---------|------|-----------|-------|
| v0.1.0 | Jun 2017 | ~6 | Provider split. Shipped: `github_repository`, `github_team`, `github_team_membership`, `github_team_repository`, `github_branch_protection`, `github_membership` |
| v1.0.0 | Feb 2018 | ~10 | Added webhooks, deploy keys, org settings. 8 months after v0.1.0 |

**Key insight:** GitHub shipped the core workflow (repo → team → membership → branch protection). ~6 resources covering the primary use case.

### Pattern Summary

| Provider | v0.1.0 Resources | Strategy | Time to v1.0 |
|----------|------------------|----------|--------------|
| PagerDuty | 8 | Core workflow (incident management chain) | 9 months |
| Datadog | 3 | Absolute minimum (monitoring primitives) | ~16 months |
| GitHub | 6 | Core workflow (repo + team management) | 8 months |
| **Devin (proposed Scope B)** | **3** | **Automation workflow (knowledge → playbook → schedule)** | **TBD** |

All three established providers shipped a **coherent workflow** rather than exhaustive coverage. None shipped analytics/metrics endpoints in their initial release. None included admin/IAM resources in v0.1.0.

---

## 6. Effort Estimate Summary

| Resource Group | Resource | Data Sources | Hours | Complexity |
|---------------|----------|-------------|-------|------------|
| **Knowledge Notes** | 1 | 2 | 8 | Simple |
| **Playbooks** | 1 | 1 | 12 | Simple |
| **Schedules** | 1 | 1 | 20 | Medium |
| **Secrets** | 1 | 1 | 16 | Medium |
| **Sessions** | 1 | 4 | 32 | Complex |
| **Attachments** | 1 | 1 | 12 | Medium |
| **PR Reviews** | 1 | 1 | 12 | Medium |
| **Repositories** | 1 | 2 | 16 | Medium (beta) |
| **Blueprints + Builds** | 2 | 3 | 32 | Medium-Complex (beta) |
| **Git Permissions** | 1 | 1 | 12 | Medium |
| **Tags** | 1 | 1 | 8 | Simple |
| **Self** | 0 | 1 | 2 | Simple |
| **Audit Logs** | 0 | 2 | 8 | Medium |
| **Consumption** | 0 | 5 | 12 | Medium |
| **Metrics** | 0 | 8 | 10 | Simple |
| **Queue** | 0 | 1 | 2 | Simple |
| **Shared Infra** | — | — | 16 | — |
| | | | | |
| **Enterprise resources** | | | | |
| Organizations | 1 | 1 | 16 | Medium |
| Users & Membership | 2 | 2 | 24 | Medium |
| Service Users + API Keys | 2 | 2 | 36 | Complex |
| IDP Groups | 3 | 3 | 20 | Medium |
| IP Access List | 1 | 1 | 6 | Simple |
| Org Group Limits | 1 | 1 | 4 | Simple |
| Roles | 0 | 1 | 2 | Simple |
| Enterprise Notes/Playbooks | 2 | 2 | 12 | Simple |
| Enterprise Sessions | 0 | 4 | 12 | Medium |
| Enterprise Blueprints | 1 | 2 | 16 | Medium |
| Enterprise PR Reviews | 1 | 1 | 6 | Medium |
| Hypervisors | 0 | 1 | 2 | Simple |
| Git Connections | 0 | 2 | 4 | Simple |
| Guardrail Violations | 0 | 2 | 6 | Medium (beta) |
| | | | | |
| **TOTAL (all)** | **~25** | **~58** | **~420h** | |

---

## 7. Recommendation

### Ship Scope B (Core) as v0.1.0

**Recommended scope:** Knowledge Notes + Playbooks + Schedules

**Why Scope B:**

1. **Coherent IaC story.** The knowledge → playbook → schedule workflow is the most natural Terraform use case: "I want to declare my Devin automation as code and version-control it." This mirrors how PagerDuty and GitHub scoped their initial releases around a core workflow.

2. **Right complexity balance.** Scope B validates both Simple (knowledge notes, playbooks) and Medium (schedules with conditional fields, cron validation, playbook references) patterns without hitting the Complex patterns (async lifecycle, write-only secrets) that add risk to a v0.1.0.

3. **Shared infra investment pays off.** Building the HTTP client, pagination helper, and error handling for 3 resources amortizes the infrastructure cost. The same patterns directly extend to secrets, sessions, and enterprise resources in v0.2.0+.

4. **Fast enough to iterate.** At 2 person-weeks, Scope B ships quickly enough to get user feedback before committing to the Complex patterns (sessions, blueprints) that may require design iteration.

5. **Avoids premature enterprise scope.** Enterprise resources add IAM, RBAC, and multi-org complexity that most early adopters don't need. Better to nail org-scope resources first.

### Proposed Timeline

| Milestone | Scope | Timeline |
|-----------|-------|----------|
| **v0.1.0** | Knowledge Notes + Playbooks + Schedules | Week 1-2 |
| **v0.2.0** | + Secrets + Sessions (org-level) | Week 3-5 |
| **v0.3.0** | + Blueprints + Repository Indexing | Week 6-8 |
| **v0.4.0** | + Enterprise resources (orgs, users, service users) | Week 9-12 |
| **v1.0.0** | All org-scope + core enterprise resources, stable API surface | Week 14-16 |

### What to Defer

- **Sessions:** Complex lifecycle (async, state machine) — needs careful Terraform design work. Target v0.2.0.
- **Secrets:** Write-only pattern needs design for `value` field handling. Target v0.2.0.
- **Beta APIs** (snapshot-setup, repositories): Wait for API stability. Target v0.3.0.
- **Enterprise resources:** Lower demand, higher complexity. Target v0.4.0.
- **Analytics data sources** (metrics, consumption): Read-only, low urgency. Sprinkle across releases as needed.

### Risks

| Risk | Mitigation |
|------|------------|
| API v3 changes before v1.0 | Pin to documented OpenAPI specs, version API client |
| Beta endpoints (v3beta1) breaking | Defer to v0.3.0, mark as experimental |
| Session resource design may not fit Terraform model | Research `terraform-provider-kubernetes` Job resource for async patterns |
| Write-only secret value may confuse users | Document clearly, consider `ignore_changes` lifecycle block in examples |
