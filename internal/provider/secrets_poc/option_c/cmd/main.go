package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	option_c "github.com/tobydrinkall/terraform-demo/internal/provider/secrets_poc/option_c"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: option_c.NewProvider,
	})
}
