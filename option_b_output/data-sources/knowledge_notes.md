---
page_title: "devin_knowledge_notes Data Source - devin"
subcategory: "Knowledge"
description: |-
  Retrieves a list of knowledge notes from the Devin platform. Use this data source to look up existing knowledge notes by scope, name pattern, or repository.
---

# devin_knowledge_notes (Data Source)

Retrieves a list of knowledge notes from the Devin platform.

Use this data source to look up existing knowledge notes by scope, name pattern, or repository. This is useful for:

- **Auditing** — enumerate all knowledge notes in your organization.
- **Cross-referencing** — feed note IDs into other resources or modules.
- **Conditional logic** — check whether a note exists before creating it.

## Example Usage

### List All Organization Notes

```terraform
data "devin_knowledge_notes" "all_org_notes" {
  scope = "organization"
}

output "note_count" {
  value = length(data.devin_knowledge_notes.all_org_notes.notes)
}
```

### Filter by Name

```terraform
data "devin_knowledge_notes" "deployment_notes" {
  scope       = "organization"
  name_filter = "deploy"
}
```

### List Repository-Scoped Notes

```terraform
data "devin_knowledge_notes" "repo_notes" {
  scope     = "repo"
  repo_name = "myorg/payments-service"
}

output "note_names" {
  value = [for note in data.devin_knowledge_notes.repo_notes.notes : note.name]
}
```

### Check If a Note Exists

```terraform
data "devin_knowledge_notes" "existing" {
  scope       = "organization"
  name_filter = "Deployment Process"
}

resource "devin_knowledge_note" "deployment_process" {
  count = length(data.devin_knowledge_notes.existing.notes) == 0 ? 1 : 0

  name    = "Deployment Process"
  content = "..."
  trigger = "When deploying infrastructure"
}
```

## Argument Reference

The following arguments are supported:

- `scope` - (Optional) Filter notes by visibility scope. Valid values are `"organization"`, `"user"`, or `"repo"`. Defaults to `"organization"`.
- `name_filter` - (Optional) A substring filter applied to note names. Only notes whose name contains this string (case-insensitive) will be returned.
- `repo_name` - (Optional) Filter notes by repository. Only applicable when `scope` is `"repo"`. Format: `owner/repo`.

## Attribute Reference

The following attributes are exported:

- `id` - The ID of this data source.
- `notes` - A list of knowledge notes matching the filter criteria. Each note contains the following attributes:
  - `id` - The unique identifier of the knowledge note.
  - `name` - The display name of the knowledge note.
  - `content` - The body content of the knowledge note.
  - `trigger` - The trigger description for the knowledge note.
  - `scope` - The visibility scope of the knowledge note.
  - `author` - The author of the knowledge note (`"user"` or `"system"`).
  - `created_at` - The ISO 8601 timestamp when the note was created.
  - `updated_at` - The ISO 8601 timestamp when the note was last updated.
  - `size` - The size of the note content in characters.

## API Reference

This data source corresponds to the following Devin API v3 endpoint:

| Operation | Method | Endpoint |
|-----------|--------|----------|
| List      | `GET`  | `/v3/knowledge` |

Query parameters are mapped from the data source arguments. See the [Devin API documentation](https://docs.devin.ai/api-reference/knowledge) for full details.

## Notes

- The data source returns **all matching notes** — there is no built-in pagination. If your organization has a very large number of notes, consider using `name_filter` or `scope` to reduce the result set.
- Results are sorted by `created_at` in descending order (newest first).
- The `name_filter` performs case-insensitive substring matching, not regex or glob matching.
