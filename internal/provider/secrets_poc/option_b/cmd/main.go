package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	option_b "github.com/tobydrinkall/terraform-demo/internal/provider/secrets_poc/option_b"
)

func main() {
	err := providerserver.Serve(context.Background(), option_b.NewProvider, providerserver.ServeOpts{
		Address: "registry.terraform.io/tobydrinkall/devinsecret",
	})
	if err != nil {
		log.Fatal(err)
	}
}
