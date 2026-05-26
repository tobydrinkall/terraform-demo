# Decision 0.3 — Org ID Resolution Strategy

## Context

The Devin API v3 scopes every endpoint under an organization:

```
https://api.devin.ai/v3/organizations/{org_id}/...
```

The Terraform provider must resolve `org_id` before making any API call. This
decision evaluates three strategies.

---

## GET /v3/self — Live API Response

Called with the service-user API key (`DEVIN_API_KEY_TERRAFORM`):

```bash
curl -H "Authorization: Bearer $DEVIN_API_KEY" https://api.devin.ai/v3/self
```

**Response (HTTP 200):**

```json
{
  "principal_type": "service_user",
  "service_user_id": "service-user-<redacted-hex>",
  "service_user_name": "terraform-demo",
  "org_id": "org-<redacted-hex>"
}
```

### Key observations

| Field               | Value                     | Notes |
|---------------------|---------------------------|-------|
| `principal_type`    | `"service_user"`          | Service-user keys are org-scoped by design |
| `service_user_id`   | `"service-user-..."`      | Stable identifier for the key |
| `service_user_name` | `"terraform-demo"`        | Human-readable label |
| `org_id`            | `"org-..."`               | **Single** org — no array, no ambiguity |

A service-user key always belongs to exactly one organization.
Enterprise keys with multi-org access have not been observed via this endpoint,
but the provider should handle that edge case defensively.

---

## Options Evaluated

All three implementations live in `internal/provider/poc/` and use
`terraform-plugin-framework` (matching the existing provider scaffolding).
A shared `common.go` (84 lines) provides the `/v3/self` HTTP call and
response parsing.

### Option A — Required `organization_id`

**File:** `internal/provider/poc/provider_option_a.go` (110 lines)

```hcl
provider "devin" {
  api_key         = var.devin_api_key   # or DEVIN_API_KEY env var
  organization_id = "org-abc123"        # REQUIRED — or DEVIN_ORG_ID env var
}
```

| Pros | Cons |
|------|------|
| Zero network calls during `terraform init` | User must know their org_id upfront |
| Fully deterministic — no surprises | Extra onboarding friction for new users |
| Works offline / in air-gapped envs | Requires documenting "how to find your org_id" |
| Simple to implement and test | Does not leverage the API key's implicit org binding |

**Error handling:** Minimal — the framework enforces `Required` at plan time.
If the value is empty string, a custom diagnostic is returned.

---

### Option B — Auto-resolve via GET /v3/self

**File:** `internal/provider/poc/provider_option_b.go` (110 lines)

```hcl
provider "devin" {
  api_key = var.devin_api_key   # or DEVIN_API_KEY env var
  # That's it — org_id is auto-resolved
}
```

| Pros | Cons |
|------|------|
| Minimal config — best onboarding UX | Network call on every `terraform init/plan/apply` |
| User never needs to look up org_id | Fails if API is unreachable (e.g., air-gapped) |
| Impossible to misconfigure org_id | No override path for multi-org enterprise keys |
| Matches AWS provider pattern (STS `GetCallerIdentity`) | Adds ~200ms latency to provider startup |

**Error handling:** If `/v3/self` fails (401, 403, timeout, missing org_id),
a detailed diagnostic is returned explaining the error and suggesting the user
check their API key.

---

### Option C — Optional with fallback (Recommended)

**File:** `internal/provider/poc/provider_option_c.go` (139 lines)

```hcl
# Minimal — auto-resolve (same UX as Option B)
provider "devin" {
  api_key = var.devin_api_key
}

# Explicit — skip the API call (same UX as Option A)
provider "devin" {
  api_key         = var.devin_api_key
  organization_id = "org-abc123"
}
```

| Pros | Cons |
|------|------|
| Best of both worlds — simple AND explicit | Most code (~139 lines vs ~110) |
| Zero-config for single-org service users | Two code paths to test |
| Explicit override for multi-org / air-gapped | Slightly more complex docs |
| Skips API call when org_id is provided | — |
| Matches common Terraform provider patterns | — |

**Error handling:** Three-tier resolution: config value → `DEVIN_ORG_ID` env
var → `/v3/self` auto-resolve. If all three fail, the error message shows
the `/v3/self` error AND provides a copy-paste provider block with the
`organization_id` attribute filled in.

---

## Comparison Matrix

| Dimension | Option A (Required) | Option B (Auto) | Option C (Optional+Fallback) |
|-----------|--------------------|-----------------|-----------------------------|
| **Lines of code** | 110 + 84 shared | 110 + 84 shared | 139 + 84 shared |
| **Network calls at init** | 0 | 1 | 0 or 1 |
| **Onboarding friction** | Medium | None | None |
| **Air-gapped support** | Yes | No | Yes (with explicit org_id) |
| **Multi-org support** | Yes | No | Yes |
| **Misconfiguration risk** | Low (explicit) | None (auto) | Low (fallback catches it) |
| **Terraform plan determinism** | Full | Full | Full |
| **Closest analogy** | GCP provider (`project`) | AWS provider (STS) | Azure provider (`subscription_id`) |

---

## UX Comparison — What the user's `.tf` file looks like

### Getting started (new user, single org)

**Option A** — user must find org_id first:
```hcl
# User: "Where do I find my org ID?" → Docs → Settings → Copy
provider "devin" {
  organization_id = "org-8d158f07ee0f4678a467078b323880a4"
}
```

**Option B** — just works:
```hcl
provider "devin" {}
# DEVIN_API_KEY env var is enough
```

**Option C** — just works (same as B):
```hcl
provider "devin" {}
```

### Multi-org / CI pipeline

**Option A** — works naturally:
```hcl
provider "devin" {
  alias           = "prod"
  organization_id = var.prod_org_id
}
```

**Option B** — impossible without switching API keys:
```hcl
# Can't target a specific org — stuck with whatever the key resolves to
provider "devin" {}
```

**Option C** — works naturally:
```hcl
provider "devin" {
  alias           = "prod"
  organization_id = var.prod_org_id
}
```

---

## Error Handling Comparison

| Scenario | Option A | Option B | Option C |
|----------|----------|----------|----------|
| Invalid API key | Error on first resource call | Error at configure (`/v3/self` 401) | Error at configure if org_id not set; otherwise deferred |
| Wrong org_id | Silent wrong-org operations | N/A | Silent wrong-org operations (if set explicitly) |
| API unreachable | No impact at configure | Configure fails | Falls back to env var; fails only if no org_id anywhere |
| Multi-org key | Works (user picks) | Ambiguous (uses whatever `/v3/self` returns) | Works (user picks explicitly) |
| Missing org_id + missing env | Framework error: "required" | N/A | Auto-resolves via `/v3/self` |

---

## Recommendation

**Option C — Optional with fallback** is recommended.

### Rationale

1. **Best onboarding UX:** New users with a single-org service-user key can
   `terraform init` with zero org configuration — the provider "just works"
   like the AWS provider does with `GetCallerIdentity`.

2. **Escape hatch for advanced use cases:** Multi-org enterprises, CI pipelines
   targeting specific orgs, and air-gapped environments can set
   `organization_id` explicitly to skip the network call entirely.

3. **Precedent:** This is the pattern used by the Azure provider
   (`subscription_id` is optional with auto-resolution) and the Datadog
   provider. It's familiar to Terraform users.

4. **Minimal downside:** The extra ~29 lines of code and one additional code
   path are a small price for the flexibility gained. Both paths are
   straightforward to unit test.

5. **The `/v3/self` response is simple:** Since service-user keys return
   exactly one `org_id` (not an array), there is no disambiguation logic
   needed — the auto-resolve path is deterministic.

### Migration note

The existing provider (on `devin/1779135731-initial-scaffolding`) already has
`organization_id` as `Optional` but errors if it's empty. Adopting Option C
would change the `Configure` function to call `/v3/self` as a fallback instead
of returning an error, which is a non-breaking change.

---

## Artifacts

| File | Lines | Description |
|------|-------|-------------|
| `internal/provider/poc/common.go` | 84 | Shared `/v3/self` HTTP client and response types |
| `internal/provider/poc/provider_option_a.go` | 110 | Required org_id provider |
| `internal/provider/poc/provider_option_b.go` | 110 | Auto-resolve provider |
| `internal/provider/poc/provider_option_c.go` | 139 | Optional + fallback provider |
| `DECISION_0_3_REPORT.md` | — | This report |
