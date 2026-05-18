---
page_title: "Provider: Devin"
description: |-
  The Devin provider enables Terraform to manage resources in the Devin AI platform via the Devin API v3. Use it to codify knowledge notes, sessions, playbooks, and other Devin resources as infrastructure.
---

# Devin Provider

The Devin provider enables Terraform to manage resources in the [Devin AI platform](https://devin.ai) via the [Devin API v3](https://docs.devin.ai/api-reference). Use it to codify knowledge notes, playbooks, and other Devin resources as infrastructure-as-code.

This is particularly useful for teams that want to:

- **Version-control Devin's knowledge base** — manage knowledge notes alongside the code they describe.
- **Enforce consistency** — ensure every repository has the same baseline knowledge notes.
- **Automate onboarding** — provision knowledge for new repos or team members via CI/CD pipelines.

## Authentication

The provider requires an API key issued from the [Devin Settings → API Keys](https://app.devin.ai/settings/api-keys) page. The key must have at least `knowledge:read` and `knowledge:write` scopes.

You can provide the API key in two ways:

1. **Environment variable** (recommended): Set `DEVIN_API_KEY` in your shell or CI environment.
2. **Provider configuration**: Pass it directly (not recommended for production).

~> **Warning:** Hard-coding credentials in Terraform configuration is not recommended. Use environment variables or a secrets manager integration.

## Example Usage

```terraform
terraform {
  required_providers {
    devin = {
      source  = "devin-ai/devin"
      version = "~> 1.0"
    }
  }
}

provider "devin" {
  api_key = var.devin_api_key # Or set DEVIN_API_KEY env var

  # Optional: override the API base URL (defaults to https://api.devin.ai/v3)
  # base_url = "https://api.devin.ai/v3"
}
```

## Argument Reference

The following arguments are supported:

- `api_key` - (Required, Sensitive) The API key for authenticating with the Devin API v3. Can also be set via the `DEVIN_API_KEY` environment variable.
- `base_url` - (Optional) The base URL of the Devin API. Defaults to `https://api.devin.ai/v3`. Override this when using a self-hosted or staging instance.

## Rate Limiting

The Devin API enforces rate limits. The provider automatically retries requests that receive `429 Too Many Requests` responses with exponential backoff (up to 3 retries). If you manage a large number of knowledge notes, consider using `-parallelism=5` with `terraform apply`.

## Resources

- [`devin_knowledge_note`](resources/knowledge_note.md) — Manage individual knowledge notes.

## Data Sources

- [`devin_knowledge_notes`](data-sources/knowledge_notes.md) — Query existing knowledge notes.
