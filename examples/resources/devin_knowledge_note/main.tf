resource "devin_knowledge_note" "example" {
  name    = "Example Coding Standards"
  trigger = "When writing or reviewing code in any repository"
  body    = <<-EOT
    ## Coding Standards

    - Use meaningful variable names
    - Write tests for all new features
    - Follow existing code conventions
  EOT
}

resource "devin_knowledge_note" "pinned" {
  name        = "Repo-Specific Guide"
  trigger     = "When working in the myorg/myrepo repository"
  body        = "Always run `make lint` before committing."
  pinned_repo = "myorg/myrepo"
}

output "example_note_id" {
  value = devin_knowledge_note.example.id
}

output "pinned_note_id" {
  value = devin_knowledge_note.pinned.id
}
