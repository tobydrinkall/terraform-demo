# Approach C: Data Sources Only
#
# No session creation — sessions are created via CLI, web app, or other tools.
# Terraform is used only for querying/monitoring.

terraform {
  required_providers {
    devin = {
      source = "registry.terraform.io/tobydrinkall/devin"
    }
  }
}

provider "devin" {}

# Look up a specific session by ID
data "devin_session" "last_deploy" {
  session_id = var.deploy_session_id
}

variable "deploy_session_id" {
  type        = string
  description = "The session ID of the last deployment session"
}

output "deploy_status" {
  value = data.devin_session.last_deploy.status
}

output "deploy_title" {
  value = data.devin_session.last_deploy.title
}

# List all currently running sessions
data "devin_sessions" "active" {
  status_filter = "running"
  limit         = 50
}

output "active_session_count" {
  value = length(data.devin_sessions.active.sessions)
}

output "active_sessions" {
  value = [for s in data.devin_sessions.active.sessions : {
    id     = s.session_id
    title  = s.title
    status = s.status
    acus   = s.acus_consumed
  }]
}

# Use session data in other resources (e.g., monitoring)
# data "devin_sessions" "recent_failures" {
#   status_filter = "exit"
#   limit         = 10
# }
