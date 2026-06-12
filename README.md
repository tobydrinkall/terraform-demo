# Terraform Provider for Devin

A Terraform provider to manage [Devin](https://devin.ai) resources via the Devin API v3.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22

## Usage

```hcl
terraform {
  required_providers {
    devin = {
      source = "tobydrinkall/devin"
    }
  }
}

provider "devin" {
  api_key         = "cog_..."       # Or set DEVIN_API_KEY env var
  organization_id = "org-abc123"    # Or set DEVIN_ORG_ID env var
}

resource "devin_knowledge_note" "example" {
  name    = "CI Pipeline Tips"
  body    = "Always run linting before pushing."
  trigger = "When working on CI pipelines"
}
```

## Authentication

The provider requires a Devin API key (service user token) and organization ID:

| Method | Variable |
|--------|----------|
| Provider attribute | `api_key` / `organization_id` |
| Environment variable | `DEVIN_API_KEY` / `DEVIN_ORG_ID` |

## Resources

- `devin_knowledge_note` — Manage knowledge notes that provide contextual information to Devin.

## Import

Resources can be imported using the composite ID format `<org_id>/<resource_id>`:

```bash
terraform import devin_knowledge_note.example org-abc123/note-def456
```

## Development

```bash
# Build
make build

# Run unit tests
make test

# Run acceptance tests (requires API credentials)
make testacc

# Lint
make lint
```

## License

MPL-2.0
