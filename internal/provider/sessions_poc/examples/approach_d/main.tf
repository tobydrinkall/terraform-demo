# Approach D: Resource + Data Sources Combined
#
# Best of both worlds — create sessions AND query existing ones.
# This is the recommended approach for production use.

terraform {
  required_providers {
    devin = {
      source = "registry.terraform.io/tobydrinkall/devin"
    }
  }
}

provider "devin" {}

# Create a new session (resource, with wait)
resource "devin_session" "code_review" {
  prompt              = "Review PR #42 on tobydrinkall/terraform-demo"
  wait_for_completion = true
  timeout             = "1h"
}

# Query running sessions to check concurrency (data source)
data "devin_sessions" "currently_running" {
  status_filter = "running"
  limit         = 100
}

# Look up a specific historical session (data source)
data "devin_session" "last_deploy" {
  session_id = var.last_deploy_session_id
}

variable "last_deploy_session_id" {
  type    = string
  default = ""
}

# Outputs combining resource and data source info
output "new_session_url" {
  value = devin_session.code_review.url
}

output "concurrent_sessions" {
  value = length(data.devin_sessions.currently_running.sessions)
}

# prevent_destroy example (user opt-in, NOT default)
resource "devin_session" "critical_deploy" {
  prompt              = "Deploy v2.0 to production"
  wait_for_completion = true
  timeout             = "2h"

  lifecycle {
    prevent_destroy = true
  }
}
