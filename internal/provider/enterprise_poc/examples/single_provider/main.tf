# ============================================================================
# Example: Single Provider — Organization + Enterprise in one provider block
# ============================================================================
#
# This is the RECOMMENDED approach (Option A).
# One provider, one API key, one state file.

terraform {
  required_providers {
    devin = {
      source  = "devin-ai/devin"
      version = "~> 1.0"
    }
  }
}

# ─────────────────────────────────────────────────────────────────────
# Provider config: enterprise_id is optional.
# If omitted, enterprise data sources return an actionable error.
# ─────────────────────────────────────────────────────────────────────
provider "devin" {
  api_key       = var.devin_api_key
  org_id        = var.devin_org_id
  enterprise_id = var.devin_enterprise_id # optional — set for enterprise features
}

# ─────────────────────────────────────────────────────────────────────
# Org-scope resources — always work with any API key
# ─────────────────────────────────────────────────────────────────────
resource "devin_knowledge_note" "coding_standards" {
  name    = "Coding Standards"
  content = "Follow the team's coding standards..."
}

# ─────────────────────────────────────────────────────────────────────
# Enterprise data source — only works with enterprise-scoped key
# If the key lacks permissions, Terraform shows:
#
#   Error: Enterprise features not available
#
#   This resource requires an enterprise-scoped service user API key
#   and the "enterprise_id" provider attribute.
#   ...
# ─────────────────────────────────────────────────────────────────────
data "devin_enterprise_audit_logs" "last_30_days" {
  start_date = "2024-12-01T00:00:00Z"
  end_date   = "2024-12-31T23:59:59Z"
  event_type = "session.created"
}

data "devin_enterprise_members" "all" {}

# ─────────────────────────────────────────────────────────────────────
# Variables
# ─────────────────────────────────────────────────────────────────────
variable "devin_api_key" {
  type      = string
  sensitive = true
}

variable "devin_org_id" {
  type = string
}

variable "devin_enterprise_id" {
  type    = string
  default = "" # Empty = enterprise features disabled (no error at plan time)
}

# ─────────────────────────────────────────────────────────────────────
# Outputs
# ─────────────────────────────────────────────────────────────────────
output "audit_log_count" {
  value = data.devin_enterprise_audit_logs.last_30_days.total_count
}
