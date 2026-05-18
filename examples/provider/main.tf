terraform {
  required_providers {
    devin = {
      source  = "cognition-ai/devin"
      version = "~> 0.1"
    }
  }
}

provider "devin" {
  # api_key         = var.devin_api_key   # or set DEVIN_API_KEY env var
  # organization_id = var.devin_org_id    # or set DEVIN_ORG_ID env var
}
