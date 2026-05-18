---
page_title: "devin_knowledge_note Resource - devin"
subcategory: "Knowledge"
description: |-
  Manages a knowledge note in the Devin platform. Knowledge notes store reusable context that Devin retrieves automatically during sessions when the trigger conditions match.
---

# devin_knowledge_note (Resource)

Manages a knowledge note in the Devin platform.

Knowledge notes store reusable context — coding standards, deployment runbooks, architectural decisions — that Devin retrieves **automatically** during sessions when the trigger conditions match. They are the primary mechanism for teaching Devin about your organization's practices without repeating yourself in every prompt.

## When to Use This Resource

- **Codify tribal knowledge** — capture standards that would otherwise live in a wiki or Slack thread.
- **Enforce per-repo context** — scope notes to specific repositories so Devin only sees what's relevant.
- **Automate knowledge rollouts** — use `for_each` or modules to provision notes across many repos.

## Example Usage

### Organization-Wide Note

```terraform
resource "devin_knowledge_note" "deployment_process" {
  name    = "Deployment Process"
  content = <<-EOT
    ## Deployment Steps
    1. Run `terraform plan` to preview changes
    2. Get approval from the #platform-eng channel
    3. Run `terraform apply` during the maintenance window
    4. Verify the deployment via the status dashboard
  EOT
  trigger = "When working on infrastructure deployment tasks"
  scope   = "organization"
}
```

### Repository-Scoped Note

```terraform
resource "devin_knowledge_note" "api_conventions" {
  name      = "API Conventions"
  content   = "All API endpoints must use snake_case for JSON fields and return pagination metadata."
  trigger   = "When adding or modifying API endpoints in the payments service"
  scope     = "repo"
  repo_name = "myorg/payments-service"
}
```

### Bulk Provisioning with `for_each`

```terraform
variable "repo_notes" {
  type = map(object({
    name    = string
    content = string
    trigger = string
    repo    = string
  }))
}

resource "devin_knowledge_note" "repo_notes" {
  for_each = var.repo_notes

  name      = each.value.name
  content   = each.value.content
  trigger   = each.value.trigger
  scope     = "repo"
  repo_name = each.value.repo
}
```

## Argument Reference

The following arguments are supported:

- `name` - (Required) The display name of the knowledge note. Must be between 1 and 200 characters.
- `content` - (Required) The body content of the knowledge note in markdown format. This is the information Devin will retrieve during sessions.
- `trigger` - (Required) A description of when this knowledge note should be retrieved. Defines the scope or conditions under which Devin will surface this note (e.g., `"When working on the payments service"`).
- `scope` - (Optional) The visibility scope of the knowledge note. Valid values:
  - `"organization"` — visible to all org members (default).
  - `"user"` — visible only to the creator.
  - `"repo"` — scoped to a specific repository (requires `repo_name`).
- `repo_name` - (Optional) The repository to scope the note to, required when `scope` is `"repo"`. Format: `owner/repo` (e.g., `"myorg/myrepo"`).

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The unique identifier of the knowledge note (e.g., `note-abc123`).
- `author` - The author of the knowledge note. Either `"user"` or `"system"`.
- `created_at` - The ISO 8601 timestamp when the knowledge note was created.
- `updated_at` - The ISO 8601 timestamp when the knowledge note was last updated.
- `size` - The size of the knowledge note content in characters.

## Import

Knowledge notes can be imported using their ID:

```shell
terraform import devin_knowledge_note.example note-abc123
```

## API Reference

This resource corresponds to the following Devin API v3 endpoints:

| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create    | `POST`   | `/v3/knowledge` |
| Read      | `GET`    | `/v3/knowledge/{id}` |
| Update    | `PATCH`  | `/v3/knowledge/{id}` |
| Delete    | `DELETE` | `/v3/knowledge/{id}` |

See the [Devin API documentation](https://docs.devin.ai/api-reference/knowledge) for full details.

## Notes

- Knowledge notes are **eventually consistent** — after creation or update, it may take up to 30 seconds for Devin to surface the note in new sessions.
- The `trigger` field uses semantic matching, not exact string comparison. Devin's retrieval system interprets the trigger contextually.
- Notes with `scope = "repo"` are only surfaced when Devin is working in the specified repository.
- The `content` field supports full GitHub-flavored markdown, including code blocks, tables, and links.
