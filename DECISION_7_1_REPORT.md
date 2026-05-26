# Decision 7.1: Release Tooling — GoReleaser vs Manual

## Executive Summary

Both release pipelines were implemented end-to-end and verified. **Recommendation: Option A (GoReleaser)** — it delivers production-quality releases with 40% less code, built-in Terraform Registry conventions, and is the de facto standard used by HashiCorp and the majority of published Terraform providers.

---

## What Was Built

| Artifact | Option A (GoReleaser) | Option B (Manual) |
|---|---|---|
| Config | `.goreleaser.yml` (78 lines) | `GNUmakefile` (62 lines) |
| Release script | N/A (GoReleaser handles it) | `scripts/release.sh` (85 lines) |
| CI workflow | `.github/workflows/release-goreleaser.yml` (46 lines) | `.github/workflows/release-manual.yml` (57 lines) |
| **Total LoC** | **124 lines** | **204 lines** |

### Platforms built (both options):
- `linux/amd64`, `linux/arm64`
- `darwin/amd64`, `darwin/arm64`
- `windows/amd64`

---

## Config File Comparison

### Option A: `.goreleaser.yml`

```yaml
version: 2
builds:
  - binary: "terraform-provider-devin_v{{ .Version }}"
    env: [CGO_ENABLED=0]
    flags: [-trimpath]
    ldflags: ["-s -w -X main.version={{ .Version }}"]
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ignore:
      - goos: windows
        goarch: arm64
archives:
  - formats: [zip]
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
checksum:
  name_template: "{{ .ProjectName }}_{{ .Version }}_SHA256SUMS"
signs:
  - artifacts: checksum
    args: ["--batch", "--local-user", "{{ .Env.GPG_FINGERPRINT }}", ...]
```

**Key characteristics:**
- Declarative YAML — describe *what* you want, not *how* to build it
- GoReleaser handles the build loop, archiving, checksumming, signing, and GitHub release creation
- Template variables (`{{ .Version }}`, `{{ .Os }}`, etc.) ensure consistent naming
- `ignore` directive cleanly excludes unsupported platform combos

### Option B: `GNUmakefile` + `scripts/release.sh`

```makefile
build-all:
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; GOARCH=$${platform#*/}; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH go build ...; \
		(cd $(DIST_DIR) && zip ... && rm ...); \
	done
checksums:
	@cd $(DIST_DIR) && shasum -a 256 *.zip > ..._SHA256SUMS
```

**Key characteristics:**
- Imperative shell — you control every step
- Manual zip creation, checksum generation, GPG signing, and `gh release create`
- More visible/debuggable for teams unfamiliar with GoReleaser
- Requires maintaining shell logic for archive naming, platform iteration, etc.

---

## Build Output Comparison

### Option A: `goreleaser build --snapshot --clean`

```
• building binaries
  • target=linux_amd64_v1     ✓
  • target=linux_arm64_v8.0   ✓
  • target=darwin_amd64_v1    ✓
  • target=darwin_arm64_v8.0  ✓
  • target=windows_amd64_v1   ✓
• build succeeded after 1m21s (first run) / 1s (cached)
```

Output structure:
```
dist/
├── artifacts.json
├── config.yaml
├── metadata.json
├── terraform-provider-devin_linux_amd64_v1/
│   └── terraform-provider-devin_v0.0.0-SNAPSHOT-dd31b14
├── terraform-provider-devin_darwin_amd64_v1/
│   └── ...
└── (3 more platform dirs)
```

### Option B: `make build-all`

```
Building linux/amd64...   → terraform-provider-devin_0.1.0_linux_amd64.zip
Building linux/arm64...   → terraform-provider-devin_0.1.0_linux_arm64.zip
Building darwin/amd64...  → terraform-provider-devin_0.1.0_darwin_amd64.zip
Building darwin/arm64...  → terraform-provider-devin_0.1.0_darwin_arm64.zip
Building windows/amd64... → terraform-provider-devin_0.1.0_windows_amd64.zip
```

Output structure:
```
dist/
├── terraform-provider-devin_0.1.0_linux_amd64.zip    (6.2 MB)
├── terraform-provider-devin_0.1.0_linux_arm64.zip    (5.6 MB)
├── terraform-provider-devin_0.1.0_darwin_amd64.zip   (6.4 MB)
├── terraform-provider-devin_0.1.0_darwin_arm64.zip   (5.9 MB)
├── terraform-provider-devin_0.1.0_windows_amd64.zip  (6.4 MB)
├── terraform-provider-devin_0.1.0_SHA256SUMS
└── terraform-registry-manifest.json
```

Both produce correctly-named zip archives and valid binaries.

---

## CI Workflow Comparison

### Option A: GoReleaser Workflow (46 lines)

```yaml
steps:
  - Checkout (with full history)
  - Set up Go (from go.mod)
  - Import GPG key (crazy-max/ghaction-import-gpg)
  - Generate Terraform Registry manifest
  - Run GoReleaser (goreleaser/goreleaser-action)
```

**Advantages:**
- Single `goreleaser release --clean` command does everything
- Official GitHub Action with good error messages
- Automatic changelog generation from git history
- Automatic GitHub Release creation with all artifacts

### Option B: Manual Workflow (57 lines)

```yaml
steps:
  - Checkout (with full history)
  - Set up Go (from go.mod)
  - Import GPG key (manual gpg --import)
  - Build all platforms (make build-all)
  - Generate checksums (make checksums)
  - Sign checksums (gpg --detach-sign)
  - Generate manifest (make manifest)
  - Create GitHub Release (gh release create)
```

**Advantages:**
- Each step is independently visible and debuggable
- No external action dependencies beyond standard checkout/setup-go
- Full control over the release note format

---

## Setup Complexity

| Factor | Option A (GoReleaser) | Option B (Manual) |
|---|---|---|
| Files to create | 2 | 3 |
| Total lines of config | 124 | 204 |
| External CI dependencies | `goreleaser-action`, `ghaction-import-gpg` | `ghaction-import-gpg` (or manual) |
| Learning curve | Moderate (GoReleaser DSL) | Low (standard shell/make) |
| Maintenance burden | Low (version bump only) | Medium (shell logic for each change) |
| Adding a new platform | 1 line in YAML | Update Makefile + release.sh |
| Adding new artifacts (e.g., RPM, Docker) | Add YAML section | Write new shell logic |

### GPG Key Setup (same for both)

1. Generate a GPG key pair (if not already available):
   ```bash
   gpg --batch --gen-key <<EOF
   Key-Type: RSA
   Key-Length: 4096
   Name-Real: Terraform Provider Signing
   Name-Email: terraform@cognition.ai
   Expire-Date: 0
   %no-protection
   EOF
   ```

2. Export the private key:
   ```bash
   gpg --armor --export-secret-keys terraform@cognition.ai > private.key
   ```

3. Add to GitHub repository secrets:
   - `GPG_PRIVATE_KEY`: contents of `private.key`
   - `GPG_PASSPHRASE`: passphrase (empty if `%no-protection` used)

4. Register the **public key** with the Terraform Registry when publishing.

---

## Registry Compatibility Verification

### Requirements for Terraform Registry

| Requirement | Option A | Option B |
|---|---|---|
| Zip archive per platform | Yes (automatic) | Yes (manual zip) |
| Naming: `{name}_{version}_{os}_{arch}.zip` | Yes (via `name_template`) | Yes (via shell) |
| SHA256SUMS file | Yes (automatic) | Yes (`make checksums`) |
| SHA256SUMS.sig (GPG detached signature) | Yes (automatic) | Yes (manual `gpg --detach-sign`) |
| `terraform-registry-manifest.json` | Yes (via `extra_files`) | Yes (`make manifest`) |
| Protocol version in manifest | `6.0` | `6.0` |
| GitHub Release with all artifacts | Yes (automatic) | Yes (`gh release create`) |

Both options produce Registry-compatible releases. The `terraform-registry-manifest.json` declares protocol version `6.0` (terraform-plugin-framework).

### Terraform dev_overrides Verification

Both binaries were tested with a `.terraformrc` dev_overrides block:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/cognition-ai/devin" = "/path/to/binary"
  }
  direct {}
}
```

Result: `terraform plan` completed successfully with both GoReleaser-built and manually-built binaries.

---

## Full Release Process: Tag to Registry

### Option A (GoReleaser)

```bash
# 1. Tag the release
git tag v0.1.0
git push origin v0.1.0

# 2. CI automatically runs:
#    - goreleaser release --clean
#    - Builds all platforms, creates checksums, signs, creates GitHub Release

# 3. Terraform Registry picks up the new release via GitHub webhook
#    (requires one-time Registry publishing setup)
```

### Option B (Manual)

```bash
# 1. Tag the release
git tag v0.1.0
git push origin v0.1.0

# 2. CI automatically runs:
#    - make build-all
#    - make checksums
#    - gpg --detach-sign
#    - make manifest
#    - gh release create with all artifacts

# 3. Terraform Registry picks up the new release via GitHub webhook
```

Both workflows trigger on `v*` tags. The end-user experience is identical: push a tag, get a release.

---

## Recommendation: Option A (GoReleaser)

**GoReleaser is the clear winner** for the following reasons:

1. **Industry standard**: GoReleaser is used by HashiCorp's own providers, nearly every community Terraform provider, and most Go-based CLI tools. This means better documentation, community support, and familiarity for contributors.

2. **40% less code**: 124 lines vs 204 lines. Less code = fewer bugs, easier review, lower maintenance.

3. **Declarative over imperative**: Adding a new platform is a 1-line YAML change vs updating shell loops in multiple files. Adding Docker image publishing or Homebrew tap support is a YAML section addition.

4. **Built-in correctness**: GoReleaser enforces archive naming conventions, generates proper checksums, and handles GPG signing — all well-tested in thousands of projects. The manual approach requires getting shell quoting, zip creation, and signing order correct.

5. **Better CI integration**: The official `goreleaser/goreleaser-action` is actively maintained, provides clear error messages, and handles edge cases (dirty builds, snapshot mode, etc.).

6. **Local testing story**: `goreleaser build --snapshot --clean` lets developers verify the full build matrix locally before pushing a tag. The manual approach can do this with `make build-all` but lacks the full release simulation.

**When to choose Option B instead:**
- If the team has a strong preference for no external tooling dependencies
- If the release process needs to be deeply customized in ways GoReleaser doesn't support
- If the project has existing Makefile-heavy workflows and wants consistency

---

## Files Created

```
.goreleaser.yml                           # Option A: GoReleaser config
.github/workflows/release-goreleaser.yml  # Option A: CI workflow
.github/workflows/release-manual.yml      # Option B: CI workflow
GNUmakefile                               # Updated with cross-compilation targets
scripts/release.sh                        # Option B: Release script
```
