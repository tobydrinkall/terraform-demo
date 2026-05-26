# Decision 3.2: Write-Only Secret Resource — State Management Strategy

## Executive Summary

The `devin_secret` resource is write-only: the API accepts a secret value on creation but never returns it. This creates a fundamental tension between **security** (don't store the value) and **drift detection** (know when the value changes). Three approaches were implemented, tested, and compared.

**Recommendation: Option C (Hash in State)** — it provides the best balance of security, drift detection, and framework compatibility.

---

## State File Contents

### Option A — ForceNew on Value Change (SDK v2)

**Plaintext value is stored in state:**

```json
{
  "attributes": {
    "created_at": "2025-01-15T10:30:00Z",
    "id": "secret-DB_PASSWORD",
    "name": "DB_PASSWORD",
    "value": "super-secret-p@ssw0rd-123"
  },
  "sensitive_attributes": [
    [{ "type": "get_attr", "value": "value" }]
  ]
}
```

The value `"super-secret-p@ssw0rd-123"` is **fully visible** in the state file. The `sensitive_attributes` marker only prevents display in CLI output — the plaintext is still in the JSON.

### Option B — WriteOnly Attribute (Plugin Framework)

**Value is null in state — never stored:**

```json
{
  "attributes": {
    "created_at": "2025-01-15T10:30:00Z",
    "id": "secret-DB_PASSWORD",
    "name": "DB_PASSWORD",
    "value": null
  },
  "sensitive_attributes": []
}
```

The value field is `null`. The framework excludes WriteOnly attributes from state serialization entirely.

### Option C — Hash in State (SDK v2)

**Only the SHA-256 hash is stored:**

```json
{
  "attributes": {
    "created_at": "2025-01-15T10:30:00Z",
    "id": "secret-DB_PASSWORD",
    "name": "DB_PASSWORD",
    "value": "c96ea6d00a9d90c493d970b3b261566bd918c4b6ff412329f8214c2da9d8c6dd",
    "value_sha256": "c96ea6d00a9d90c493d970b3b261566bd918c4b6ff412329f8214c2da9d8c6dd"
  },
  "sensitive_attributes": [
    [{ "type": "get_attr", "value": "value" }]
  ]
}
```

`StateFunc` transforms the plaintext to `sha256(value)` before it reaches state. The `value` field contains the hash, not the plaintext. The `value_sha256` computed attribute mirrors it.

---

## Security Analysis

| Threat Vector | Option A (ForceNew) | Option B (WriteOnly) | Option C (Hash) |
|---|---|---|---|
| **State file at rest** | Plaintext exposed | Not stored | Hash only (irreversible) |
| **State in remote backend (S3, TFC)** | Plaintext in transit + storage | Not stored | Hash only |
| **`terraform show` output** | Masked as `(sensitive)` | Shows `(write-only)` | Masked as `(sensitive)` |
| **`terraform state pull` (raw JSON)** | Plaintext fully visible | `null` | Hash visible, safe |
| **Plan file (`.tfplan`)** | Contains plaintext | Not included | Contains hash |
| **VCS-committed state** | Critical secret leak | No risk | No risk (hash is safe) |
| **Log output** | Redacted by Sensitive flag | Not present | Redacted by Sensitive flag |

### Verdict

- **Option A** is the **least secure**. Any user with state file access (S3 bucket, Terraform Cloud workspace, CI artifacts) can read the plaintext secret. The `Sensitive` flag is cosmetic — it only affects CLI rendering.
- **Option B** is the **most secure**. The value never touches state storage at all.
- **Option C** is **secure in practice**. SHA-256 is irreversible for high-entropy secrets. For low-entropy secrets (short passwords), dictionary attacks on the hash are theoretically possible but impractical in most threat models.

---

## Drift Detection Comparison

| Scenario | Option A (ForceNew) | Option B (WriteOnly) | Option C (Hash) |
|---|---|---|---|
| **Value unchanged, re-run plan** | No changes | No changes | No changes |
| **Value changed in config** | Detects change, **destroy+create** | **Cannot detect** — shows "No changes" | Detects change, **update in-place** |
| **Value changed via API (external drift)** | Cannot detect (API doesn't return value) | Cannot detect | Cannot detect (API doesn't return value) |
| **Name changed in config** | Detects, forces replacement | Detects, in-place update | Detects, forces replacement |

### Key Finding: Option B Cannot Detect Value Changes

When the value was changed from `super-secret-p@ssw0rd-123` to `new-changed-password` in the Terraform config:

```
# Option A plan output:
-/+ resource "devinsecret_secret" "db_password" must be replaced
      ~ value = (sensitive value) # forces replacement

# Option B plan output:
No changes. Your infrastructure matches the configuration.

# Option C plan output:
  ~ resource "devinsecret_secret" "db_password" will be updated in-place
      ~ value = (sensitive value)
```

**Option B's WriteOnly attribute is fire-and-forget.** Since the value is not in state, Terraform has nothing to compare against. The user could change the value in their `.tf` file and `terraform plan` would report no changes — the secret would silently not be updated.

This is the fundamental trade-off: **perfect security at the cost of zero drift detection**.

### Drift Detection Verdict

- **Option C** is the best: detects config-side value changes via hash comparison, performs in-place update (no downtime from destroy+create).
- **Option A** detects changes but forces replacement, which means the secret ID changes — any downstream references break.
- **Option B** is a no-op after creation — value changes are invisible to Terraform.

---

## Framework Compatibility (Decision 0.1 Implications)

| Factor | Option A (SDK v2) | Option B (Framework) | Option C (SDK v2) |
|---|---|---|---|
| **Framework** | terraform-plugin-sdk/v2 | terraform-plugin-framework | terraform-plugin-sdk/v2 |
| **Terraform version** | >= 0.12 | **>= 1.11 only** | >= 0.12 |
| **Matches current provider** | Yes — same SDK as existing resources | No — different framework, requires mux | Yes — same SDK as existing resources |
| **Muxing required** | No | Yes — `terraform-plugin-mux` to combine SDK v2 + Framework providers | No |
| **Migration path** | Stays in SDK v2 | Forces partial Framework adoption or full migration | Stays in SDK v2 |
| **WriteOnly support** | N/A | Native | N/A (simulated via StateFunc) |

### Key Finding: Option B Requires Terraform 1.11+

During testing, Option B failed on Terraform 1.8.5:

```
│ Error: WriteOnly Attribute Not Allowed
│ Write-only attributes are only supported in Terraform 1.11 and later.
```

It only succeeded after installing Terraform 1.11.4. This is a **hard version gate** — users on Terraform < 1.11 cannot use this provider at all if any resource uses WriteOnly.

### Key Finding: Option B Requires Provider Muxing

The current provider (`provider.go`, `main.go`) uses SDK v2. Option B uses the Framework. To ship both in the same binary, you need `terraform-plugin-mux` to combine them:

```go
// Hypothetical main.go with muxing
muxServer, _ := tf5muxserver.NewMuxServer(ctx,
    sdkProvider.GRPCProvider,          // SDK v2 resources
    providerserver.NewProtocol5(fwProvider), // Framework resources
)
```

This adds complexity, a new dependency, and a new error surface. Options A and C slot directly into the existing provider.

---

## Implementation Summary

| Metric | Option A | Option B | Option C |
|---|---|---|---|
| **Lines of code (resource)** | 60 | 107 | 95 |
| **Lines of code (provider)** | 28 | 49 | 28 |
| **New dependencies** | None | terraform-plugin-framework | None |
| **Update behavior** | Destroy + create | Update in-place (but blind) | Update in-place (hash-aware) |
| **State cleanup** | ForceNew handles it | Framework handles it | Manual hash management |

---

## Recommendation: Option C (Hash in State)

**Option C is the recommended approach** for the following reasons:

1. **Security**: The plaintext never reaches state. SHA-256 hashes are irreversible for secrets with reasonable entropy. This eliminates the primary security concern of Option A.

2. **Drift Detection**: Unlike Option B, Option C detects when a user changes the secret value in their config. The `DiffSuppressFunc` compares `sha256(new_config_value)` against the stored hash. This prevents silent failures where a rotated secret never gets applied.

3. **Framework Compatibility**: Option C uses SDK v2, identical to the existing provider. No muxing, no new dependencies, no Terraform version gates. It works with Terraform 0.12+.

4. **Update Behavior**: Changes trigger an in-place update (not destroy+create), so the secret ID stays stable. Downstream references are not broken.

5. **Future-Proofing**: If the provider is later migrated to the Framework (Decision 0.1), Option C's pattern can be preserved or upgraded to WriteOnly. Option C is not a dead end.

### When to Reconsider

- If the **minimum supported Terraform version** is raised to 1.11+, Option B becomes viable and should be reconsidered for its superior security guarantees (no hash in state at all).
- If drift detection is truly **not needed** (secrets are only ever set once and never rotated), Option B is simpler.
- Option A should only be used if the team explicitly decides that state-level encryption (e.g., Terraform Cloud managed state, encrypted S3 backend) is sufficient and the simplicity of ForceNew outweighs the security risk.

---

## Artifacts

| File | Description |
|---|---|
| `internal/provider/secrets_poc/option_a/resource_secret.go` | Option A resource implementation |
| `internal/provider/secrets_poc/option_a/provider.go` | Option A provider wrapper |
| `internal/provider/secrets_poc/option_a/cmd/main.go` | Option A entry point |
| `internal/provider/secrets_poc/option_a/testdata/main.tf` | Option A test config |
| `internal/provider/secrets_poc/option_b/resource_secret.go` | Option B resource implementation |
| `internal/provider/secrets_poc/option_b/provider.go` | Option B provider wrapper |
| `internal/provider/secrets_poc/option_b/cmd/main.go` | Option B entry point |
| `internal/provider/secrets_poc/option_b/testdata/main.tf` | Option B test config |
| `internal/provider/secrets_poc/option_c/resource_secret.go` | Option C resource implementation |
| `internal/provider/secrets_poc/option_c/provider.go` | Option C provider wrapper |
| `internal/provider/secrets_poc/option_c/cmd/main.go` | Option C entry point |
| `internal/provider/secrets_poc/option_c/testdata/main.tf` | Option C test config |
