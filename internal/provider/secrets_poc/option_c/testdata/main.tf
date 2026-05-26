terraform {
  required_providers {
    devinsecret = {
      source = "registry.terraform.io/tobydrinkall/devinsecret"
    }
  }
}

provider "devinsecret" {}

resource "devinsecret_secret" "db_password" {
  name  = "DB_PASSWORD"
  value = "super-secret-p@ssw0rd-123"
}

output "secret_id" {
  value = devinsecret_secret.db_password.id
}
