# ──────────────────────────────────────────────────────────────────
# Option C — Two separate resources
#
# devin_recurring_schedule  → requires `frequency`, no scheduled_at
# devin_one_time_schedule   → requires `scheduled_at`, no frequency
#
# No conditional logic needed — the schema itself enforces the
# contract. All errors appear at PLAN time (schema-level Required).
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

resource "devin_recurring_schedule" "daily_standup" {
  title     = "Daily standup summary"
  prompt    = "Summarise all open PRs and blockers"
  frequency = "0 9 * * 1-5"
}

# ── Happy path: one-time ─────────────────────────────────────────

resource "devin_one_time_schedule" "launch_day" {
  title        = "Launch-day deploy"
  prompt       = "Deploy release v2.1 to production"
  scheduled_at = "2025-06-01T09:00:00Z"
}

# ── Error case: recurring without frequency ──────────────────────
# Uncomment to see (PLAN time — schema Required):
#
#   Error: Missing required argument
#
#   The argument "frequency" is required, but no definition was found.
#
# resource "devin_recurring_schedule" "missing_freq" {
#   title  = "Will fail"
#   prompt = "No frequency"
# }

# ── Error case: one-time without scheduled_at ────────────────────
# Uncomment to see (PLAN time — schema Required):
#
#   Error: Missing required argument
#
#   The argument "scheduled_at" is required, but no definition was found.
#
# resource "devin_one_time_schedule" "missing_datetime" {
#   title  = "Will fail"
#   prompt = "No datetime"
# }

# ── Error case: invalid cron → ValidateFunc (PLAN time) ──────────
# Uncomment to see:
#
#   Error: expected value of frequency to match regular expression
#
# resource "devin_recurring_schedule" "bad_cron" {
#   title     = "Will fail"
#   prompt    = "Bad cron"
#   frequency = "not-a-cron"
# }

# ── Error case: invalid datetime → ValidateFunc (PLAN time) ──────
# Uncomment to see:
#
#   Error: "scheduled_at" must be a valid ISO 8601 / RFC 3339 datetime
#
# resource "devin_one_time_schedule" "bad_datetime" {
#   title        = "Will fail"
#   prompt       = "Bad datetime"
#   scheduled_at = "next tuesday"
# }

# ── Note: users literally cannot mix up the attributes ───────────
# There is no `scheduled_at` field on devin_recurring_schedule, and
# no `frequency` field on devin_one_time_schedule. Terraform will
# reject unknown attributes at parse time with:
#
#   Error: Unsupported argument
#
#   An argument named "scheduled_at" is not expected here.
