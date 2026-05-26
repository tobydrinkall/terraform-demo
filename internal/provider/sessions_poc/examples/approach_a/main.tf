# Approach A: Fire-and-Forget Session Resource
#
# terraform apply  → creates the session → returns immediately
# terraform destroy → terminates the session (if still active)

terraform {
  required_providers {
    devin = {
      source = "registry.terraform.io/tobydrinkall/devin"
    }
  }
}

provider "devin" {
  # api_key via DEVIN_API_KEY env var
}

# Simple fire-and-forget session
resource "devin_session" "code_review" {
  prompt = "Review the latest PR on tobydrinkall/terraform-demo and provide feedback"
}

output "session_url" {
  value       = devin_session.code_review.url
  description = "View the session in the Devin web app"
}

output "session_status" {
  value       = devin_session.code_review.status
  description = "Initial status (will be 'new' or 'claimed')"
}

# Multiple concurrent sessions
resource "devin_session" "batch" {
  for_each = {
    lint   = "Run linting on the terraform-demo repo"
    test   = "Run tests on the terraform-demo repo"
    docs   = "Update README for the terraform-demo repo"
  }

  prompt = each.value
}

output "batch_urls" {
  value = { for k, v in devin_session.batch : k => v.url }
}
