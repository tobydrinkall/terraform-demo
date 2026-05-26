# Decision 0.2: Terraform Provider Namespace — Personal vs Organization

| Field        | Value                                                    |
|-------------|----------------------------------------------------------|
| **Decision** | Which GitHub/Registry namespace to publish the Devin Terraform provider under |
| **Status**   | Proposed                                                 |
| **Date**     | 2026-05-26                                               |

---

## Table of Contents

1. [Context](#context)
2. [How the Terraform Registry Namespace System Works](#how-the-terraform-registry-namespace-system-works)
3. [Option A — Personal Namespace](#option-a--personal-namespace-eg-tobydrinkalldevin)
4. [Option B — Organization Namespace](#option-b--organization-namespace-eg-cognition-aidevin)
5. [How Other API-Product Companies Handle This](#how-other-api-product-companies-handle-this)
6. [Partner & Partner Premier Tier Deep-Dive](#partner--partner-premier-tier-deep-dive)
7. [Comparison Table](#comparison-table)
8. [Migration & Transfer Risks](#migration--transfer-risks)
9. [Recommendation](#recommendation)
10. [Next Steps](#next-steps)

---

## Context

We are building a Terraform provider for the Devin API v3. Before publishing, we need to decide which namespace to use on the Terraform Registry. The namespace directly maps to the GitHub user or organization that owns the `terraform-provider-devin` repository. This decision is **effectively permanent** — changing it later breaks every user's `required_providers` block.

---

## How the Terraform Registry Namespace System Works

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Terraform Registry                              │
│                                                                     │
│  registry.terraform.io / {NAMESPACE} / {PROVIDER_NAME}              │
│                          │                  │                        │
│                          │                  └─ Derived from repo     │
│                          │                     name after            │
│                          │                     "terraform-provider-" │
│                          │                                           │
│                          └─ MUST match the GitHub username           │
│                             or GitHub org name exactly               │
│                                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐  ┌───────────────┐   │
│  │ Official │  │ Partner  │  │   Partner     │  │  Community    │   │
│  │          │  │ Premier  │  │              │  │               │   │
│  │hashicorp/│  │mongodb/  │  │PagerDuty/    │  │jianyuan/      │   │
│  │aws       │  │mongodb-  │  │pagerduty     │  │sentry         │   │
│  │          │  │atlas     │  │              │  │               │   │
│  └──────────┘  └──────────┘  └──────────────┘  └───────────────┘   │
│                                                                     │
│  Tier badges:                                                       │
│  Official  = HashiCorp-owned                                        │
│  Partner Premier = Verified partner + advanced features             │
│  Partner   = HashiCorp Technology Partner Program member             │
│  Community = Anyone with a GitHub account                           │
└─────────────────────────────────────────────────────────────────────┘
```

**Key rules** (from [Terraform Registry Publishing docs](https://developer.hashicorp.com/terraform/registry/providers/publishing)):

1. Repository **must** be named `terraform-provider-{NAME}` (lowercase only)
2. Repository **must** be public
3. Namespace = GitHub username or org name (no override possible)
4. Releases **must** be GPG-signed with a key registered in the Registry
5. Release tags **must** follow semver preceded by `v` (e.g., `v1.0.0`)

---

## Option A — Personal Namespace (e.g. `tobydrinkall/devin`)

### What Users Would Write

```hcl
terraform {
  required_providers {
    devin = {
      source  = "tobydrinkall/devin"
      version = "~> 1.0"
    }
  }
}
```

### Setup Steps

1. **GitHub repo**: Create `tobydrinkall/terraform-provider-devin` (public)
2. **GPG key**: Generate a keypair (RSA or DSA — NOT ECC)
   ```bash
   gpg --full-generate-key        # Choose RSA, 4096-bit
   gpg --armor --export "your@email.com"
   ```
3. **Registry sign-in**: Go to [registry.terraform.io](https://registry.terraform.io), sign in with the `tobydrinkall` GitHub account
4. **Upload GPG public key**: Settings → Signing Keys → Add GPG key (for personal namespace)
5. **Authorize OAuth**: Ensure the Terraform Registry GitHub OAuth app has `repo` and `admin:org_hook` permissions
6. **Publish**: Registry → Publish → Provider → select `tobydrinkall/terraform-provider-devin`
7. **Release workflow**: Set up GitHub Actions with GoReleaser:
   - Store `GPG_PRIVATE_KEY` and `PASSPHRASE` in repo secrets
   - Copy `.goreleaser.yml` from [hashicorp/terraform-provider-scaffolding-framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework)
   - Copy release workflow to `.github/workflows/release.yml`
   - Tag with `git tag v1.0.0 && git push --tags` to trigger release
8. **Registry webhook**: Automatically created on publish — listens for `release` events

### Registry Tier

**Community** — no badge. Cannot upgrade to Partner without moving to an org namespace that participates in the HashiCorp Technology Partner Program.

### Pros

- Fastest path to publishing (~30 min setup)
- No org creation overhead
- Good for prototyping or internal validation

### Cons

- **Permanently tied to one person's GitHub account**
- Community tier only — no Partner badge possible
- Less professional appearance for an enterprise product
- Bus factor of 1 — if the account is compromised/deleted, the provider is orphaned
- Logo = personal GitHub avatar, not company branding

---

## Option B — Organization Namespace (e.g. `cognition-ai/devin`)

### What Users Would Write

```hcl
terraform {
  required_providers {
    devin = {
      source  = "cognition-ai/devin"
      version = "~> 1.0"
    }
  }
}
```

### Setup Steps

1. **GitHub org**: Create or use existing org (e.g., `cognition-ai`, `devinai`)
   - Ensure org has a profile picture (this becomes the Registry logo)
   - Any GitHub org plan works (Free, Team, Enterprise)
2. **GitHub repo**: Create `{org}/terraform-provider-devin` (public)
3. **GPG key**: Generate a keypair (same process as personal)
   ```bash
   gpg --full-generate-key        # Choose RSA, 4096-bit
   gpg --armor --export "your@email.com"
   ```
4. **Registry sign-in**: Sign in with a GitHub account that is an **admin** of the org
5. **Upload GPG public key**: Settings → Signing Keys → select the **organization** → Add GPG key
6. **Authorize OAuth**: Ensure the Terraform Registry GitHub OAuth app has permissions on the org
   - Go to GitHub Settings → Applications → Authorized OAuth Apps → Terraform Registry
   - Grant access to the organization
7. **Publish**: Registry → Publish → Provider → select `{org}/terraform-provider-devin`
8. **Release workflow**: Same GitHub Actions + GoReleaser setup as personal namespace
9. **(Optional) Apply for Partner tier**: See [Partner Tier Deep-Dive](#partner--partner-premier-tier-deep-dive) below

### Registry Tier

**Community** initially → can upgrade to **Partner** → can further upgrade to **Partner Premier**

### Org Membership Requirements

- The GitHub user who publishes the provider must be an **owner/admin** of the GitHub org
- Any org member can push releases via GitHub Actions
- Multiple admins can manage Registry settings
- No requirement for a specific GitHub plan (Free org works)

### Pros

- **Professional credibility** — matches industry standard
- **Eligible for Partner badge** via HashiCorp Technology Partner Program
- Multi-person admin access — no single point of failure
- Company logo and branding on Registry page
- Can add/remove team members without ownership transfer
- Aligns with how 100% of first-party API-product providers are published

### Cons

- Requires GitHub org to exist (trivial to create)
- Partner badge requires applying to HashiCorp Technology Partner Program
- Slightly more initial coordination

---

## How Other API-Product Companies Handle This

### Verified Registry Data (API responses from `registry.terraform.io/v1/providers/`)

```
┌─────────────────┬──────────────────────┬───────────────┬──────────────┬───────────────┐
│ Company         │ Registry Namespace   │ Tier          │ Downloads    │ GitHub Org     │
├─────────────────┼──────────────────────┼───────────────┼──────────────┼───────────────┤
│ AWS             │ hashicorp/aws        │ Official      │ 6.28B        │ hashicorp     │
│ Datadog         │ DataDog/datadog      │ Partner       │ 429M         │ DataDog       │
│ GitHub          │ integrations/github  │ Partner       │ 301M         │ integrations  │
│ Cloudflare      │ cloudflare/cloudflare│ Partner       │ 256M         │ cloudflare    │
│ Grafana         │ grafana/grafana      │ Partner       │ 167M         │ grafana       │
│ PagerDuty       │ PagerDuty/pagerduty  │ Partner       │ 132M         │ PagerDuty     │
│ MongoDB Atlas   │ mongodb/mongodbatlas │ Partner Prem. │ 72M          │ mongodb       │
│ Sentry (*)      │ jianyuan/sentry      │ Community     │ 21M          │ jianyuan (!)  │
│ LaunchDarkly    │ launchdarkly/ld      │ Partner       │ 17M          │ launchdarkly  │
└─────────────────┴──────────────────────┴───────────────┴──────────────┴───────────────┘

(*) Sentry is the ONLY major API-product company without a first-party provider.
    The community provider is maintained by an individual (jianyuan).
    This is widely considered a gap — not a model to follow.
```

### Visual: How Namespace Affects User Perception

```
     PERSONAL NAMESPACE                    ORG NAMESPACE
     ──────────────────                    ─────────────

  ┌─────────────────────────┐        ┌──────────────────────────┐
  │  🔷 jianyuan/sentry     │        │  ✦ DataDog/datadog       │
  │                         │        │                          │
  │  Community              │        │  Partner                 │
  │  No verified badge      │        │  ✓ Verified by HashiCorp │
  │  Personal avatar        │        │  Company logo            │
  │  21M downloads          │        │  429M downloads          │
  │                         │        │                          │
  │  "Who maintains this?"  │        │  "This is the official   │
  │  "Is this abandoned?"   │        │   Datadog provider"      │
  └─────────────────────────┘        └──────────────────────────┘

  Users TRUST org namespaces         Partner badge = HashiCorp
  less by default, especially        has verified the company
  for infrastructure tooling.        and the integration.
```

### Key Pattern: Company Name = Namespace

Every successful first-party provider follows this pattern:

```
Company Name    →  GitHub Org      →  Registry Namespace   →  Provider Source
────────────    ─  ──────────      ─  ──────────────────   ─  ───────────────
Datadog         →  DataDog         →  DataDog/datadog      →  DataDog/terraform-provider-datadog
PagerDuty       →  PagerDuty       →  PagerDuty/pagerduty  →  PagerDuty/terraform-provider-pagerduty
LaunchDarkly    →  launchdarkly    →  launchdarkly/ld      →  launchdarkly/terraform-provider-launchdarkly
Cloudflare      →  cloudflare      →  cloudflare/cloudflare→  cloudflare/terraform-provider-cloudflare
Grafana         →  grafana         →  grafana/grafana      →  grafana/terraform-provider-grafana
MongoDB         →  mongodb         →  mongodb/mongodbatlas →  mongodb/terraform-provider-mongodbatlas
```

**Implication for Cognition/Devin:**
```
Cognition AI    →  cognition-ai    →  cognition-ai/devin   →  cognition-ai/terraform-provider-devin
     OR
Devin AI        →  devinai         →  devinai/devin        →  devinai/terraform-provider-devin
```

---

## Partner & Partner Premier Tier Deep-Dive

### How to Get the Partner Badge

The Partner badge is granted by HashiCorp when you participate in the [HashiCorp Technology Partner Program](https://www.hashicorp.com/ecosystem/become-a-partner/). The process:

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│ 1.PREPARE│────>│2.PUBLISH │────>│ 3.APPLY  │────>│4.VERIFY  │────>│5.SUPPORT │
│          │     │          │     │          │     │          │     │          │
│Build     │     │Publish to│     │Apply to  │     │HashiCorp │     │Maintain  │
│provider  │     │Registry  │     │Tech      │     │Alliances │     │provider, │
│using     │     │under org │     │Partner   │     │team      │     │fix bugs  │
│Plugin    │     │namespace │     │Program   │     │verifies  │     │within    │
│Framework │     │          │     │          │     │          │     │SLAs      │
└──────────┘     └──────────┘     └──────────┘     └──────────┘     └──────────┘
```

### Partner Requirements (from HashiCorp docs)

| # | Requirement | Notes |
|---|-------------|-------|
| 1 | Join [HashiCorp Technology Partner Program](https://www.hashicorp.com/ecosystem/become-a-partner/) | Free to apply |
| 2 | Built on Terraform Plugin Framework (not legacy SDK) | Our provider already uses Plugin Framework |
| 3 | Terraform Plugin testing integrated | `go test` with acceptance tests |
| 4 | Commit to updating and fixing issues | Ongoing maintenance commitment |
| 5 | Collaborate on joint customer escalations | Support SLA partnership |
| 6 | Nightly CI testing via GitHub Actions | Automated test pipeline |

### Partner Premier Additional Requirements (1 technical + 1 feature)

**Technical (pick 1):**
- Include SBOM (Software Bill of Materials) using GitHub SBOM Generator, Syft, or CycloneDX

**Feature (pick 1):**
- Ephemeral Resources (if applicable)
- Terraform Search (requires resource identities + list resources)
- Terraform Actions

### Support SLAs for Partner Tier

| Severity | Response Time |
|----------|--------------|
| Critical | 48 hours |
| All other | 5 business days |

> HashiCorp reserves the right to remove partner status from any integration that does not meet maintenance requirements.

### License Requirements

Partner providers must use an approved open-source license. Recommended: **MPL 2.0** (same as Terraform itself) or **Apache 2.0**.

---

## Comparison Table

| Dimension | Personal Namespace | Organization Namespace |
|-----------|-------------------|----------------------|
| **Example source** | `tobydrinkall/devin` | `cognition-ai/devin` |
| **Setup effort** | ~30 min | ~1 hour |
| **Initial tier** | Community | Community |
| **Partner eligible** | No | Yes |
| **Partner Premier eligible** | No | Yes |
| **Credibility** | Low — personal project perception | High — first-party product perception |
| **Bus factor** | 1 person | Org with multiple admins |
| **Logo/branding** | Personal avatar | Company logo |
| **GPG key management** | Personal only | Org-level, multiple admins |
| **GitHub Actions** | Same | Same |
| **GoReleaser** | Same | Same |
| **Can transfer later?** | No — breaks `required_providers` | No — but less likely to need to |
| **Migration path** | Must publish new provider under new namespace, old one is abandoned | Stable long-term |
| **Industry precedent** | Sentry (community, not first-party) | Datadog, PagerDuty, LaunchDarkly, Cloudflare, Grafana, MongoDB |
| **Maintenance** | Individual responsibility | Team responsibility |
| **HCP Terraform management** | Limited | Full org features |

---

## Migration & Transfer Risks

### Can You Transfer from Personal to Org Later?

**Short answer: No, not without breaking users.**

From the [Terraform Registry FAQ](https://developer.hashicorp.com/terraform/registry/faq):

> We strongly recommend *against* moving or renaming an existing artifact. Instead, it's often a better choice to create a new repository, representing the new name of your artifact while leaving the old content in place.

**What would happen if you started personal and moved to org:**

```
BEFORE (personal):
  source = "tobydrinkall/devin"
  ↓
  terraform init → downloads from registry.terraform.io/tobydrinkall/devin ✓

AFTER (moved to org):
  source = "tobydrinkall/devin"     ← All existing users' configs
  ↓
  terraform init → BROKEN ✗         ← Old namespace is abandoned

  source = "cognition-ai/devin"     ← Must update every config
  ↓
  terraform init → works ✓           ← But requires state migration
```

**Migration would require every user to:**
1. Update `required_providers` source
2. Run `terraform state replace-provider tobydrinkall/devin cognition-ai/devin`
3. Re-run `terraform init`

This is **disruptive** and erodes user trust. It's the #1 reason to start with the right namespace.

### GitHub Org Rename

If you rename the GitHub org, GitHub's automatic redirect means the Registry continues to work. However, HashiCorp recommends reclaiming old org names to prevent namespace hijacking.

---

## Recommendation

### Use an Organization Namespace

**Recommended namespace:** `cognition-ai/devin` or `devinai/devin`

**Rationale:**

1. **Industry standard**: 100% of successful first-party API-product providers (Datadog, PagerDuty, LaunchDarkly, Cloudflare, Grafana, MongoDB) use their company's GitHub org namespace. Zero use personal namespaces.

2. **Credibility**: The namespace is the first thing users see. `cognition-ai/devin` immediately communicates "this is the official provider from Cognition AI." `tobydrinkall/devin` communicates "someone made a community provider."

3. **Partner badge path**: Only org namespaces can earn the Partner badge through the HashiCorp Technology Partner Program. This badge significantly increases adoption trust.

4. **No migration tax**: Starting with the org namespace avoids the painful and disruptive migration that would be required if starting personal and later needing to move.

5. **Team scalability**: Multiple org admins can manage GPG keys, Registry settings, and releases. The provider isn't tied to one person's GitHub account.

6. **Minimal extra effort**: Creating a GitHub org is free and takes 5 minutes. The total setup difference is ~30 minutes.

### Suggested Namespace Choice

| Option | Terraform Source | Notes |
|--------|-----------------|-------|
| `cognition-ai/devin` | `cognition-ai/devin` | Matches company name, professional |
| `devinai/devin` | `devinai/devin` | Product-focused, shorter |

Either works. Choose based on whether the long-term brand emphasis should be on the company (Cognition AI) or the product (Devin).

### Recommended Tier Strategy

```
Phase 1 (Now):          Community tier under org namespace
                        ↓
Phase 2 (Post-launch):  Apply to HashiCorp Technology Partner Program
                        ↓
Phase 3 (Mature):       Partner badge → increased trust/adoption
                        ↓
Phase 4 (Optional):     Partner Premier (SBOM + advanced features)
```

---

## Next Steps

1. **Create/confirm GitHub org** (e.g., `cognition-ai` or `devinai`)
2. **Create public repo** `{org}/terraform-provider-devin`
3. **Generate GPG signing key** (RSA 4096-bit)
4. **Register the provider** on the Terraform Registry under the org namespace
5. **Set up GitHub Actions** with GoReleaser for automated releases
6. **Publish v0.1.0** to validate the full pipeline
7. **(Post-launch)** Apply to HashiCorp Technology Partner Program for Partner badge

---

## References

- [Terraform Registry: Publishing Providers](https://developer.hashicorp.com/terraform/registry/providers/publishing)
- [Terraform Registry: Provider Tiers & Namespaces](https://developer.hashicorp.com/terraform/registry/providers)
- [Terraform Integration Program](https://developer.hashicorp.com/terraform/docs/partnerships)
- [Terraform Registry FAQ](https://developer.hashicorp.com/terraform/registry/faq)
- [HashiCorp Technology Partner Program](https://www.hashicorp.com/ecosystem/become-a-partner/)
- [GoReleaser Config (scaffolding)](https://github.com/hashicorp/terraform-provider-scaffolding-framework)
