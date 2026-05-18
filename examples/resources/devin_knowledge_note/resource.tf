# Manage an organization-wide knowledge note
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

# Manage a repo-scoped knowledge note
resource "devin_knowledge_note" "api_conventions" {
  name      = "API Conventions"
  content   = "All API endpoints must use snake_case for JSON fields and return pagination metadata."
  trigger   = "When adding or modifying API endpoints in the payments service"
  scope     = "repo"
  repo_name = "myorg/payments-service"
}
