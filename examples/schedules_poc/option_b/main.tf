# ──────────────────────────────────────────────────────────────────
# Option B — SDK validators (ConflictsWith + CustomizeDiff)
#
# `frequency` and `scheduled_at` use ConflictsWith so they cannot
# both be set. A CustomizeDiff enforces that the correct one IS
# set for the chosen schedule_type. All errors appear at PLAN time.
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

resource "devin_schedule_b" "daily_standup" {
  title         = "Daily standup summary"
  prompt        = "Summarise all open PRs and blockers"
  schedule_type = "recurring"
  frequency     = "0 9 * * 1-5"
}

# ── Happy path: one-time ─────────────────────────────────────────

resource "devin_schedule_b" "launch_day" {
  title         = "Launch-day deploy"
  prompt        = "Deploy release v2.1 to production"
  schedule_type = "one_time"
  scheduled_at  = "2025-06-01T09:00:00Z"
}

# ── Error case: both set → ConflictsWith (PLAN time) ─────────────
# Uncomment to see:
#
#   Error: Conflicting configuration arguments
#
#   "frequency": conflicts with scheduled_at
#
# resource "devin_schedule_b" "both_set" {
#   title         = "Will fail"
#   prompt        = "Both attrs"
#   schedule_type = "recurring"
#   frequency     = "0 9 * * 1-5"
#   scheduled_at  = "2025-06-01T09:00:00Z"
# }

# ── Error case: recurring without frequency → CustomizeDiff (PLAN) ─
# Uncomment to see:
#
#   Error: "frequency" is required when schedule_type is "recurring"
#
# resource "devin_schedule_b" "missing_freq" {
#   title         = "Will fail"
#   prompt        = "No frequency"
#   schedule_type = "recurring"
# }

# ── Error case: one-time without scheduled_at → CustomizeDiff (PLAN)
# Uncomment to see:
#
#   Error: "scheduled_at" is required when schedule_type is "one_time"
#
# resource "devin_schedule_b" "missing_datetime" {
#   title         = "Will fail"
#   prompt        = "No datetime"
#   schedule_type = "one_time"
# }

# ── Error case: invalid cron → ValidateFunc (PLAN time) ──────────
# Uncomment to see:
#
#   Error: expected value of frequency to match regular expression
#   must be a valid 5-field cron expression
#
# resource "devin_schedule_b" "bad_cron" {
#   title         = "Will fail"
#   prompt        = "Bad cron"
#   schedule_type = "recurring"
#   frequency     = "not-a-cron"
# }

# ── Error case: invalid datetime → ValidateFunc (PLAN time) ──────
# Uncomment to see:
#
#   Error: "scheduled_at" must be a valid ISO 8601 / RFC 3339 datetime
#
# resource "devin_schedule_b" "bad_datetime" {
#   title         = "Will fail"
#   prompt        = "Bad datetime"
#   schedule_type = "one_time"
#   scheduled_at  = "next tuesday"
# }
