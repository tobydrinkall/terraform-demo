# List all organization-level knowledge notes
data "devin_knowledge_notes" "all_org_notes" {
  scope = "organization"
}

# Filter notes by name
data "devin_knowledge_notes" "deployment_notes" {
  scope       = "organization"
  name_filter = "deploy"
}

# List repo-scoped notes
data "devin_knowledge_notes" "repo_notes" {
  scope     = "repo"
  repo_name = "myorg/payments-service"
}

# Use the data source to output note names
output "note_names" {
  value = [for note in data.devin_knowledge_notes.all_org_notes.notes : note.name]
}
