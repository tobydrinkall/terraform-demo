resource "devin_secret" "github_token" {
  key          = "GITHUB_TOKEN"
  value        = var.github_token
  type         = "key-value"
  is_sensitive = true
  note         = "GitHub PAT for CI access"
}

resource "devin_secret" "totp_secret" {
  key          = "GITHUB_2FA"
  value        = var.totp_seed
  type         = "totp"
  is_sensitive = true
  note         = "GitHub TOTP seed for 2FA"
}
