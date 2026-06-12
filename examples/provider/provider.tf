terraform {
  required_providers {
    devin = {
      source = "tobydrinkall/devin"
    }
  }
}

provider "devin" {
  # api_key         = "cog_..."       # Or set DEVIN_API_KEY env var
  # organization_id = "org-abc123"    # Or set DEVIN_ORG_ID env var
}
