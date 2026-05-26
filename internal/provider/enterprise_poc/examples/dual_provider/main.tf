# ============================================================================
# Example: Dual Provider — Separate providers for org and enterprise scope
# ============================================================================
#
# This is the ALTERNATIVE approach (Option B).
# Two providers, two API keys, potentially separate state.

terraform {
  required_providers {
    devin = {
      source  = "devin-ai/devin"
      version = "~> 1.0"
    }
    devin-enterprise = {
      source  = "devin-ai/devin-enterprise"
      version = "~> 1.0"
    }
  }
}

# ─────────────────────────────────────────────────────────────────────
# Provider 1: Organization scope (regular API key)
# ─────────────────────────────────────────────────────────────────────
provider "devin" {
  api_key = var.devin_org_api_key
  org_id  = var.devin_org_id
}

# ─────────────────────────────────────────────────────────────────────
# Provider 2: Enterprise scope (enterprise service user key)
# ─────────────────────────────────────────────────────────────────────
provider "devin-enterprise" {
  api_key       = var.devin_enterprise_api_key
  enterprise_id = var.devin_enterprise_id
}

# ─────────────────────────────────────────────────────────────────────
# Org-scope resources — use "devin" provider
# ─────────────────────────────────────────────────────────────────────
resource "devin_knowledge_note" "coding_standards" {
  name    = "Coding Standards"
  content = "Follow the team's coding standards..."
}

# ─────────────────────────────────────────────────────────────────────
# Enterprise data sources — use "devin-enterprise" provider
# Note the resource name prefix: "devin-enterprise_" not "devin_"
# ─────────────────────────────────────────────────────────────────────
data "devin-enterprise_audit_logs" "last_30_days" {
  start_date = "2024-12-01T00:00:00Z"
  end_date   = "2024-12-31T23:59:59Z"
}

data "devin-enterprise_members" "all" {}

# ─────────────────────────────────────────────────────────────────────
# PROBLEM: Knowledge notes exist in both scopes!
#
# The org provider can read/write notes for one org:
#   data "devin_knowledge_notes" "my_org" { ... }
#
# The enterprise provider can read notes across ALL orgs:
#   data "devin-enterprise_knowledge_notes" "all_orgs" { ... }
#
# Users must understand which to use, and the schemas differ.
# Cross-references between providers require explicit data passing.
# ─────────────────────────────────────────────────────────────────────
data "devin-enterprise_knowledge_notes" "cross_org" {
  org_id = var.devin_org_id
}

# ─────────────────────────────────────────────────────────────────────
# Variables — now the user needs TWO sets of credentials
# ─────────────────────────────────────────────────────────────────────
variable "devin_org_api_key" {
  type      = string
  sensitive = true
}

variable "devin_enterprise_api_key" {
  type      = string
  sensitive = true
}

variable "devin_org_id" {
  type = string
}

variable "devin_enterprise_id" {
  type = string
}
