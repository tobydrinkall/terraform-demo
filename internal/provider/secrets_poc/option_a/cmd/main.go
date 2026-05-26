package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	option_a "github.com/tobydrinkall/terraform-demo/internal/provider/secrets_poc/option_a"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: option_a.NewProvider,
	})
}
