# Terraform Provider for Devin API v3

A Terraform provider for managing [Devin](https://devin.ai) resources via the [API v3](https://docs.devin.ai/api-reference/overview).

## Status

**Work in progress** — starting with `devin_knowledge_note` as the first resource.

## Resources

| Resource | Status |
|----------|--------|
| `devin_knowledge_note` | In progress |

## Data Sources

| Data Source | Status |
|-------------|--------|
| `devin_knowledge_note` (singular) | In progress |
| `devin_knowledge_notes` (plural) | In progress |

## Quick Start

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
}
```

### Example: Knowledge Note

```hcl
resource "devin_knowledge_note" "coding_standards" {
  name        = "Coding Standards"
  trigger     = "When writing or reviewing code in any repository"
  body        = file("knowledge/coding-standards.md")
  pinned_repo = "myorg/myrepo"
}
```

## Development

### Prerequisites

- Go 1.22+
- Docker (for integration tests)
- Terraform 1.5+ (for manual testing)

### Build

```bash
make build
```

### Test

```bash
# Unit tests (no external deps)
make test

# Integration tests (requires Docker for WireMock)
make testintegration

# Acceptance tests (requires DEVIN_API_KEY + DEVIN_ORG_ID)
make testacc
```

## Architecture

```
internal/
├── client/       # HTTP client for Devin API v3
├── provider/     # Terraform resource and data source implementations
└── testutil/     # WireMock test infrastructure (testcontainers-go)
```

## Testing Layers

| Layer | External Deps | Runs When | Purpose |
|-------|--------------|-----------|---------|
| Unit | None | Every push | Client logic, error handling, serialization |
| Integration (WireMock) | Docker | Every push | Contract verification, lifecycle sequences |
| Acceptance | Devin API | Main only | Real API validation |

## License

MIT
