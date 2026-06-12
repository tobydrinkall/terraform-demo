resource "devin_schedule" "daily_standup" {
  name          = "Daily Standup Summary"
  prompt        = "Summarize yesterday's PRs and open issues for the team."
  schedule_type = "recurring"
  frequency     = "0 9 * * 1-5"
  agent         = "devin"
  notify_on     = "always"
}

resource "devin_schedule" "one_time_migration" {
  name          = "Database Migration"
  prompt        = "Run the pending database migrations in staging."
  schedule_type = "one_time"
  scheduled_at  = "2026-06-15T10:00:00Z"
  agent         = "devin"
  notify_on     = "failure"
}
