package option_a

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func NewProvider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("DEVIN_API_KEY", "mock-key"),
				Description: "The API key (mock for PoC).",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"devinsecret_secret": ResourceSecret(),
		},
		ConfigureContextFunc: func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
			return nil, nil
		},
	}
}
