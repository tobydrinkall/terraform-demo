resource "devin_playbook" "code_review" {
  title   = "Code Review Playbook"
  content = <<-EOT
    ## Code Review Steps
    1. Check for correctness
    2. Run the test suite
    3. Verify no regressions
    4. Leave constructive feedback
  EOT

  # Optional automation macro
  # macro = "!code_review"
}
