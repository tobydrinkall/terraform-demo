# Approach B: Wait-for-Completion Session Resource
#
# terraform apply → creates session → optionally waits for completion
# Great for CI/CD pipelines where you need to know the result

terraform {
  required_providers {
    devin = {
      source = "registry.terraform.io/tobydrinkall/devin"
    }
  }
}

provider "devin" {}

# Fire-and-forget mode (same as Approach A)
resource "devin_session" "quick_task" {
  prompt              = "Say hello and stop"
  wait_for_completion = false
}

# Blocking mode — terraform apply waits for Devin to finish
resource "devin_session" "deploy_review" {
  prompt              = "Review deployment PR #42 and approve if tests pass"
  wait_for_completion = true
  timeout             = "1h"
  poll_interval       = "30s"
}

output "review_completed" {
  value = !devin_session.deploy_review.timed_out
}

output "review_status" {
  value = devin_session.deploy_review.status
}

output "acus_used" {
  value = devin_session.deploy_review.acus_consumed
}

# CI/CD pipeline pattern: sequential dependent sessions
resource "devin_session" "build" {
  prompt              = "Build and test the application"
  wait_for_completion = true
  timeout             = "30m"
}

resource "devin_session" "deploy" {
  # Only runs after build completes
  depends_on = [devin_session.build]

  prompt              = "Deploy build ${devin_session.build.session_id} to staging"
  wait_for_completion = true
  timeout             = "15m"
}
