provider "devin" {
  api_key = var.devin_api_key # Or set DEVIN_API_KEY env var

  # Optional: override the API base URL (defaults to https://api.devin.ai/v3)
  # base_url = "https://api.devin.ai/v3"
}
