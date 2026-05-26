# ──────────────────────────────────────────────────────────────────
# Option A — CRUD-time validation
#
# Both `frequency` and `scheduled_at` are Optional in the schema.
# Invalid combinations are caught during `terraform apply` (not plan).
# ──────────────────────────────────────────────────────────────────

terraform {
  required_providers {
    devin = {
      source = "tobydrinkall/devin"
    }
  }
}

provider "devin" {
  api_key = "test-key"
}

# ── Happy path: recurring ────────────────────────────────────────

resource "devin_schedule_a" "daily_standup" {
  title         = "Daily standup summary"
  prompt        = "Summarise all open PRs and blockers"
  schedule_type = "recurring"
  frequency     = "0 9 * * 1-5" # weekdays at 09:00 UTC
}

# ── Happy path: one-time ─────────────────────────────────────────

resource "devin_schedule_a" "launch_day" {
  title         = "Launch-day deploy"
  prompt        = "Deploy release v2.1 to production"
  schedule_type = "one_time"
  scheduled_at  = "2025-06-01T09:00:00Z"
}

# ── Error case: recurring without frequency ──────────────────────
# Uncomment to see the error (appears at APPLY time):
#
#   Error: Missing required attribute for recurring schedule
#
#   When schedule_type is "recurring", the "frequency" attribute
#   must be set to a valid cron expression.
#
# resource "devin_schedule_a" "bad_recurring" {
#   title         = "Will fail"
#   prompt        = "No frequency"
#   schedule_type = "recurring"
# }

# ── Error case: one-time without scheduled_at ────────────────────
# Uncomment to see the error (appears at APPLY time):
#
#   Error: Missing required attribute for one-time schedule
#
#   When schedule_type is "one_time", the "scheduled_at" attribute
#   must be set to a valid ISO 8601 datetime.
#
# resource "devin_schedule_a" "bad_one_time" {
#   title         = "Will fail"
#   prompt        = "No datetime"
#   schedule_type = "one_time"
# }

# ── Error case: recurring with scheduled_at ──────────────────────
# Uncomment to see the error (appears at APPLY time):
#
#   Error: Invalid attribute for recurring schedule
#
#   The "scheduled_at" attribute must not be set when
#   schedule_type is "recurring".
#
# resource "devin_schedule_a" "wrong_combo" {
#   title         = "Will fail"
#   prompt        = "Wrong attribute"
#   schedule_type = "recurring"
#   scheduled_at  = "2025-06-01T09:00:00Z"
# }
