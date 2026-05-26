package sessions_poc

// Approach D: Resource for create + data sources for query
//
// This approach combines:
//   - ResourceSessionWaitForCompletion (from Approach B) for session creation/lifecycle
//   - DataSourceSession / DataSourceSessions (from Approach C) for querying
//
// The key insight is that Approach D doesn't need new Go code — it's an
// architectural recommendation about which resources and data sources to
// include in the provider together.
//
// The Terraform configuration demonstrates the combined usage pattern:
//
//   # Create a session (resource)
//   resource "devin_session" "deploy_review" {
//     prompt              = "Review the deployment PR #42"
//     wait_for_completion = true
//     timeout             = "1h"
//   }
//
//   # Query existing sessions (data source)
//   data "devin_sessions" "running" {
//     status_filter = "running"
//     limit         = 10
//   }
//
//   # Look up a specific session (data source)
//   data "devin_session" "specific" {
//     session_id = "abc123"
//   }
//
// prevent_destroy consideration:
//
// Terraform supports lifecycle { prevent_destroy = true } on resources.
// For sessions, this is complex because:
//   1. Sessions are ephemeral — they complete and become read-only
//   2. Once in "exit" state, destroy is a no-op (just removes from state)
//   3. prevent_destroy on an active session would block terraform destroy entirely
//
// Recommendation: Do NOT set prevent_destroy by default. Instead, document
// the pattern for users who want it:
//
//   resource "devin_session" "critical" {
//     prompt = "Deploy to production"
//     lifecycle {
//       prevent_destroy = true  # User opt-in
//     }
//   }

// ApproachDRegistration describes what gets registered in the provider for Approach D.
type ApproachDRegistration struct {
	Resources   map[string]string
	DataSources map[string]string
}

// GetApproachDRegistration returns the resource/data-source mapping for Approach D.
func GetApproachDRegistration() ApproachDRegistration {
	return ApproachDRegistration{
		Resources: map[string]string{
			"devin_session": "ResourceSessionWaitForCompletion (Approach B)",
		},
		DataSources: map[string]string{
			"devin_session":  "DataSourceSession (Approach C - single)",
			"devin_sessions": "DataSourceSessions (Approach C - list)",
		},
	}
}
