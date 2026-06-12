resource "devin_knowledge_note" "example" {
  name    = "CI Pipeline Tips"
  body    = "Always run linting before pushing. Use `make lint` to check locally."
  trigger = "When working on CI pipelines or GitHub Actions workflows"

  # Optional: pin to a specific repository
  # pinned_repo = "myorg/myrepo"
}
