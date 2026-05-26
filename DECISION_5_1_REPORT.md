# Decision 5.1: Single Provider vs Dual Provider for Enterprise-Scope Endpoints

## Executive Summary

The Devin API v3 has two authentication scopes: **organization-scope** (`POST /v3/organizations/{org_id}/...`) and **enterprise-scope** (`GET /v3/enterprise/...`). This decision evaluates whether to handle both in a single `devin` provider or split enterprise into a separate `devin-enterprise` provider.

**Recommendation: Option A — Single Provider** with conditional enterprise enablement. This matches the overwhelming industry consensus (5/5 major providers studied use this pattern) and produces a simpler, more maintainable user experience.

---

## Industry Survey: How 6 Major Providers Handle Multi-Scope Auth

### 1. GitHub Provider (`integrations/terraform-provider-github`)

**Pattern: Single provider, enterprise resources coexist with org/user resources**

```
provider "github" {
  token = var.github_token    # PAT or GitHub App — same key, different scopes
  owner = "my-org"
}

# Org-scope resource
resource "github_repository" "app" { ... }

# Enterprise-scope resource — same provider, requires enterprise permissions
resource "github_enterprise_actions_permissions" "ent" {
  enterprise_slug = "my-enterprise"
}
```

The GitHub provider registers `github_enterprise_*` resources alongside `github_repository`, `github_membership`, etc. — all in one provider. Enterprise resources take an `enterprise_slug` attribute. If the token lacks enterprise permissions, the API returns 403 and Terraform surfaces the error.

**Key insight**: GitHub chose NOT to create a separate `github-enterprise` provider despite having distinct API scopes. Enterprise resources are prefixed `github_enterprise_` to signal scope.

```
┌─────────────────────────────────────────────────┐
│            provider "github"                    │
│                                                 │
│  ┌──────────────┐  ┌─────────────────────────┐  │
│  │  Org/User    │  │  Enterprise             │  │
│  │  Resources   │  │  Resources              │  │
│  │              │  │                          │  │
│  │  repository  │  │  enterprise_actions_     │  │
│  │  membership  │  │    permissions           │  │
│  │  team        │  │  enterprise_            │  │
│  │  branch_     │  │    organization          │  │
│  │   protection │  │                          │  │
│  └──────────────┘  └─────────────────────────┘  │
│                                                 │
│  Single token — scope determined by permissions │
└─────────────────────────────────────────────────┘
```

### 2. Datadog Provider (`DataDog/terraform-provider-datadog`)

**Pattern: Single provider, single API key covers all scopes**

```
provider "datadog" {
  api_key = var.dd_api_key    # Same key for team + org resources
  app_key = var.dd_app_key
  api_url = "https://api.datadoghq.com"
}

# Team-scope
resource "datadog_monitor" "cpu" { ... }

# Org-scope
resource "datadog_organization_settings" "org" { ... }
```

Datadog manages team-level resources (monitors, dashboards) and org-level resources (organization settings, API keys, child organizations) in a single provider. There is no separate "datadog-enterprise" provider. The API key's permissions determine which operations succeed.

### 3. AWS Provider (`hashicorp/terraform-provider-aws`)

**Pattern: Single provider, account-level + org-level resources coexist**

```
provider "aws" {
  region = "us-east-1"
  # Single set of credentials — IAM permissions determine scope
}

# Account-level resource
resource "aws_s3_bucket" "data" { ... }

# Organization-level resource — same provider, requires org permissions
resource "aws_organizations_policy" "scp" { ... }
resource "aws_organizations_account" "dev" { ... }
```

AWS has `aws_organizations_*` resources alongside hundreds of account-scoped resources. The provider does NOT split into a separate "aws-organizations" provider. IAM permissions on the credentials control access — if the role lacks `organizations:*` permissions, those resources fail with a clear permissions error.

```
┌──────────────────────────────────────────────────┐
│             provider "aws"                       │
│                                                  │
│  ┌──────────────┐  ┌──────────────────────────┐  │
│  │  Account     │  │  Organization            │  │
│  │  Resources   │  │  Resources               │  │
│  │  (~1000+)    │  │  (~15 resources)          │  │
│  │              │  │                           │  │
│  │  s3_bucket   │  │  organizations_policy     │  │
│  │  ec2_inst    │  │  organizations_account    │  │
│  │  lambda_fn   │  │  organizations_ou         │  │
│  └──────────────┘  └──────────────────────────┘  │
│                                                  │
│  Single IAM role — permissions = scope           │
└──────────────────────────────────────────────────┘
```

### 4. PagerDuty Provider (`PagerDuty/terraform-provider-pagerduty`)

**Pattern: Single provider, multiple auth methods (token + OAuth + user token)**

```
provider "pagerduty" {
  token      = var.pd_token       # Account-level API token
  user_token = var.pd_user_token  # Optional user-scoped token

  # OR OAuth scoped token
  use_app_oauth_scoped_token {
    pd_client_id     = var.client_id
    pd_client_secret = var.client_secret
    pd_subdomain     = "mycompany"
  }
}
```

PagerDuty supports account-level resources (services, escalation policies) and user-level operations in a single provider. The provider accepts multiple authentication methods — the token scope determines which resources work. There is no separate PagerDuty Enterprise provider.

### 5. Okta Provider (`okta/okta`)

**Pattern: Single provider for all tiers (Developer, Production, Enterprise)**

```
provider "okta" {
  org_name  = "my-company"
  base_url  = "okta.com"        # or "oktapreview.com"
  api_token = var.okta_token
}

# Works on all tiers
resource "okta_user" "jane" { ... }

# Enterprise-only features — same provider
resource "okta_admin_role_targets" "admin" { ... }
resource "okta_group_schema_property" "custom" { ... }
```

Okta does not create separate providers for Developer vs Enterprise tiers. All resources live in one provider. Enterprise-only features (custom schemas, admin role targets) simply fail with API errors when used on non-enterprise tenants.

### 6. Auth0 Provider (`auth0/terraform-provider-auth0`)

**Pattern: Single provider, tenant features unlocked by plan**

```
provider "auth0" {
  domain    = "my-tenant.auth0.com"
  api_token = var.auth0_token
}

# Free tier
resource "auth0_client" "spa" { ... }

# Enterprise tier features — same provider
resource "auth0_custom_domain" "brand" { ... }
resource "auth0_organization" "corp" { ... }
```

Auth0 bundles free-tier and enterprise-tier resources in one provider. Enterprise features like custom domains and organizations return API errors if the tenant plan doesn't support them.

### Industry Summary

| Provider   | Approach       | Enterprise Resources          | Separate Provider? |
|------------|----------------|-------------------------------|--------------------|
| GitHub     | Single         | `github_enterprise_*` prefix  | No                 |
| Datadog    | Single         | `datadog_organization_*`      | No                 |
| AWS        | Single         | `aws_organizations_*`         | No                 |
| PagerDuty  | Single         | Multiple auth methods          | No                 |
| Okta       | Single         | Plan-gated features           | No                 |
| Auth0      | Single         | Plan-gated features           | No                 |

**Result: 6/6 providers use a single provider. None split enterprise into a separate provider.**

---

## Architecture Comparison

### Option A: Single Provider (Recommended)

```
┌──────────────────────────────────────────────────────────────────────┐
│                      provider "devin"                                │
│                                                                      │
│  api_key       = var.devin_api_key                                   │
│  org_id        = "org-abc123"                                        │
│  enterprise_id = "enterprise-xyz789"  ← optional                     │
│                                                                      │
│  ┌─────────────────────┐     ┌────────────────────────────────────┐  │
│  │   Org Resources     │     │   Enterprise Data Sources         │  │
│  │                     │     │                                    │  │
│  │  devin_knowledge_   │     │  devin_enterprise_audit_logs      │  │
│  │    note             │     │  devin_enterprise_members          │  │
│  │  devin_session      │     │  devin_enterprise_usage            │  │
│  │  devin_secret       │     │                                    │  │
│  │  devin_playbook     │     │  if enterprise_id not set:         │  │
│  │                     │     │    → actionable error message      │  │
│  └─────────────────────┘     └────────────────────────────────────┘  │
│                                                                      │
│  DevinClient {                                                       │
│    APIKey, OrgID, EnterpriseID                                       │
│    IsEnterpriseEnabled() bool                                        │
│    RequireEnterprise() diag.Diagnostics                              │
│    OrgURL(path) string                                               │
│    EnterpriseURL(path) string                                        │
│  }                                                                   │
└──────────────────────────────────────────────────────────────────────┘
```

**How enterprise gating works:**

```go
func dataSourceEnterpriseAuditLogsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
    client := meta.(*DevinClient)

    // Gate: one line, consistent across all enterprise resources
    if diags := client.RequireEnterprise(); diags != nil {
        return diags
    }

    // Proceed with enterprise API call...
}
```

**Error message when enterprise_id is missing:**

```
Error: Enterprise features not available

  This resource requires an enterprise-scoped service user API key
  and the "enterprise_id" provider attribute.

  To use enterprise resources:
    1. Create a service user with enterprise scope in the Devin dashboard
       (Settings > Enterprise > Service Users)
    2. Set the "enterprise_id" attribute in the provider block
    3. Use the enterprise-scoped API key

  For more information, see: https://docs.devin.ai/enterprise/service-users
```

### Option B: Dual Provider

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        User's Terraform Config                          │
│                                                                          │
│  terraform {                                                             │
│    required_providers {                                                   │
│      devin = {                                                           │
│        source  = "devin-ai/devin"                                        │
│        version = "~> 1.0"                                                │
│      }                                                                   │
│      devin-enterprise = {                        ← 2nd provider          │
│        source  = "devin-ai/devin-enterprise"                             │
│        version = "~> 1.0"                                                │
│      }                                                                   │
│    }                                                                     │
│  }                                                                       │
│                                                                          │
│  ┌────────────────────────┐    ┌──────────────────────────────────────┐  │
│  │  provider "devin" {    │    │  provider "devin-enterprise" {      │  │
│  │    api_key = org_key   │    │    api_key       = ent_key          │  │
│  │    org_id  = "org-123" │    │    enterprise_id = "ent-789"        │  │
│  │  }                     │    │  }                                   │  │
│  │                        │    │                                      │  │
│  │  devin_knowledge_note  │    │  devin-enterprise_audit_logs        │  │
│  │  devin_session         │    │  devin-enterprise_members           │  │
│  │  devin_knowledge_notes │    │  devin-enterprise_knowledge_notes ← │  │
│  └────────────────────────┘    └───────────── DUPLICATE? ────────────┘  │
│                                                                          │
│  PROBLEM: knowledge_notes exists in BOTH providers.                      │
│  Which one do you use? They hit the same API.                            │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Code Complexity Analysis

### Option A: Single Provider

| Component                           | Lines | Notes                                     |
|-------------------------------------|-------|-------------------------------------------|
| Provider schema (with enterprise_id)| ~80   | One provider block with optional field    |
| DevinClient struct + helpers        | ~50   | IsEnterpriseEnabled(), RequireEnterprise()|
| Enterprise data source (audit logs) | ~140  | Full schema + gated Read function         |
| Enterprise data source (members)    | ~80   | Same pattern, less schema                 |
| Tests                               | ~200  | Covers gate logic, schemas, URL builders  |
| **Total PoC**                       | **~550**| Single package, single binary           |

Adding a new enterprise resource requires:
1. Create the data source file (~80-140 lines)
2. Register it in the provider's `DataSourcesMap` (1 line)
3. Call `client.RequireEnterprise()` at the top of Read (2 lines)

**Incremental cost per enterprise resource: ~85-145 lines**

### Option B: Dual Provider

| Component                           | Lines | Notes                                     |
|-------------------------------------|-------|-------------------------------------------|
| Org provider (existing)             | ~50   | Unchanged                                 |
| Enterprise provider schema          | ~60   | Separate provider block                   |
| EnterpriseClient struct             | ~30   | Separate client type                      |
| Enterprise data sources             | ~220  | Same schemas, different client type        |
| Shared knowledge_notes (duplicate)  | ~50   | Read-only version with cross-org filter   |
| Tests (per provider)                | ~300  | Need tests for both providers             |
| Build/release infrastructure        | ~100  | Separate go.mod, Makefile, CI pipeline    |
| **Total PoC**                       | **~810**| Two packages, two binaries              |

Adding a new enterprise resource requires:
1. Create the data source file (~80-140 lines) — same as Option A
2. Register it in the enterprise provider (1 line)
3. Decide: does this resource also exist in the org provider? If yes, duplicate.

**Incremental cost per enterprise resource: ~85-145 lines + potential duplication**

### Ongoing Maintenance

| Concern                    | Single Provider       | Dual Provider              |
|----------------------------|-----------------------|----------------------------|
| Go modules                 | 1 go.mod              | 2 go.mods                  |
| Binary artifacts           | 1                     | 2                          |
| Registry listings          | 1                     | 2                          |
| CI/CD pipelines            | 1                     | 2                          |
| Dependency upgrades        | 1 pass                | 2 passes (coordinated)     |
| SDK version alignment      | Automatic             | Manual coordination needed |
| API client code            | Shared in-process     | Duplicated or extracted    |
| Shared resource schemas    | N/A                   | Must be kept in sync       |
| Documentation              | 1 docs site           | 2 docs sites               |

---

## User Experience Comparison

### Getting Started (Single Provider)

```hcl
# Org-only user — simple, no enterprise config needed
provider "devin" {
  api_key = var.devin_api_key
  org_id  = var.devin_org_id
}

resource "devin_knowledge_note" "standards" {
  name    = "Coding Standards"
  content = "..."
}
```

Later, when the user gets enterprise access:

```hcl
# Just add enterprise_id — no new provider, no new required_providers block
provider "devin" {
  api_key       = var.devin_api_key
  org_id        = var.devin_org_id
  enterprise_id = var.devin_enterprise_id  # ← add this one line
}

# Now enterprise data sources work
data "devin_enterprise_audit_logs" "recent" { ... }
```

### Getting Started (Dual Provider)

```hcl
# Org user needs just the "devin" provider
terraform {
  required_providers {
    devin = {
      source = "devin-ai/devin"
      version = "~> 1.0"
    }
  }
}

provider "devin" {
  api_key = var.devin_api_key
  org_id  = var.devin_org_id
}
```

Later, when they get enterprise access:

```hcl
# Must add a SECOND provider + required_providers entry
terraform {
  required_providers {
    devin = {
      source  = "devin-ai/devin"
      version = "~> 1.0"
    }
    devin-enterprise = {                     # ← new block
      source  = "devin-ai/devin-enterprise"
      version = "~> 1.0"
    }
  }
}

provider "devin" {
  api_key = var.org_api_key
  org_id  = var.devin_org_id
}

provider "devin-enterprise" {               # ← new provider block
  api_key       = var.enterprise_api_key     # ← new variable
  enterprise_id = var.devin_enterprise_id
}
```

### UX Comparison Summary

| Scenario                          | Single Provider     | Dual Provider             |
|-----------------------------------|---------------------|---------------------------|
| First-time org-only setup         | 4 lines             | 10 lines                  |
| Adding enterprise later           | +1 line             | +12 lines, new provider   |
| Provider version management       | 1 version           | 2 versions to coordinate  |
| `terraform init` downloads        | 1 binary            | 2 binaries                |
| Credential management             | 1 API key           | 2 API keys                |
| Resource name discoverability     | `devin_*` prefix    | `devin_*` + `devin-enterprise_*` |
| Cross-scope data references       | Direct              | Requires explicit passing |
| Registry search experience        | One listing         | "Which provider do I need?" |

---

## Risk Analysis

### Single Provider Risks

| Risk                                   | Severity | Mitigation                                   |
|----------------------------------------|----------|----------------------------------------------|
| Provider binary grows large            | Low      | Enterprise adds data sources, not heavy deps |
| Users accidentally use enterprise resources | Low  | Clear error message with instructions        |
| Enterprise API changes break org users | Low      | Enterprise resources are isolated code paths |

### Dual Provider Risks

| Risk                                   | Severity | Mitigation                                   |
|----------------------------------------|----------|----------------------------------------------|
| Version skew between providers         | Medium   | Coordinated releases (operational burden)    |
| Shared resource confusion              | High     | Documentation — but users will still be confused |
| Double the maintenance burden          | Medium   | Shared library extraction (adds complexity)  |
| Users don't know which provider to use | Medium   | Documentation — poor discoverability         |

---

## Recommendation: Option A — Single Provider

**Recommended approach: Single `devin` provider with optional `enterprise_id` attribute.**

### Rationale

1. **Industry consensus is unanimous.** All 6 major providers studied (GitHub, Datadog, AWS, PagerDuty, Okta, Auth0) use a single provider for multi-scope auth. None split enterprise into a separate provider. This is a solved design problem.

2. **Simpler user experience.** Adding enterprise is +1 line (`enterprise_id = ...`) vs. +12 lines (new required_providers, new provider block, new credentials). The upgrade path from org-only to enterprise is trivial.

3. **Lower maintenance cost.** One binary, one CI pipeline, one go.mod, one registry listing, one documentation site. The dual approach roughly doubles operational overhead with no user benefit.

4. **No shared resource problem.** Knowledge notes, sessions, and other org-scope resources exist once. With dual providers, users face "which provider manages knowledge notes?" confusion.

5. **Error messages solve the permission gap.** The `RequireEnterprise()` gate produces actionable errors that tell users exactly what they need (enterprise service user key, enterprise_id attribute, link to docs). This is better UX than "you installed the wrong provider."

6. **Future-proof.** If enterprise write endpoints are added later (e.g., managing enterprise-wide policies), they slot into the single provider with the same `RequireEnterprise()` gate. No architecture change needed.

### Implementation Plan

```
internal/provider/
├── provider.go                          # Add enterprise_id to schema
├── client.go                            # Add EnterpriseID, RequireEnterprise()
├── resource_knowledge_note.go           # Unchanged (org-scope)
├── data_source_knowledge_notes.go       # Unchanged (org-scope)
├── data_source_enterprise_audit_logs.go # NEW — calls RequireEnterprise()
├── data_source_enterprise_members.go    # NEW — calls RequireEnterprise()
└── data_source_enterprise_usage.go      # NEW — calls RequireEnterprise()
```

### Naming Convention

Following GitHub's pattern, enterprise resources use the `devin_enterprise_` prefix:

```
devin_knowledge_note              → org-scope resource
devin_knowledge_notes             → org-scope data source
devin_enterprise_audit_logs       → enterprise-scope data source
devin_enterprise_members          → enterprise-scope data source
```

This makes scope immediately visible in resource names without requiring a separate provider.

---

## PoC Artifacts

All implementation files are in `internal/provider/enterprise_poc/`:

| File                                    | Description                                      |
|-----------------------------------------|--------------------------------------------------|
| `provider_single.go`                    | Single provider with conditional enterprise      |
| `data_source_enterprise_audit_logs.go`  | Enterprise audit logs with permission gating      |
| `data_source_enterprise_members.go`     | Enterprise members with permission gating         |
| `provider_dual.go`                      | Dual provider sketch (for comparison)            |
| `provider_single_test.go`              | 9 tests covering schemas, gates, URL builders    |
| `examples/single_provider/main.tf`      | Example config for single provider approach       |
| `examples/dual_provider/main.tf`        | Example config for dual provider approach         |

### Test Results

```
=== RUN   TestProviderSingle_Schema                    --- PASS
=== RUN   TestProviderSingle_EnterpriseGate_NoEntID    --- PASS
=== RUN   TestProviderSingle_EnterpriseGate_WithEntID  --- PASS
=== RUN   TestProviderSingle_URLBuilders               --- PASS
=== RUN   TestProviderEnterprise_Schema                --- PASS
=== RUN   TestProviderSingle_IsConfigurable            --- PASS
=== RUN   TestDataSourceEnterpriseAuditLogs_Schema     --- PASS
=== RUN   TestProviderComparison_ResourceCounts        --- PASS
=== RUN   TestDataSourceEnterpriseAuditLogs_SchemaRes  --- PASS
PASS — 9/9 tests passing
```
