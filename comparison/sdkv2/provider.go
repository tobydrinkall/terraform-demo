package sdkv2

import (
	"context"

	"github.com/COG-GTM/terraform-provider-devin/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// New returns the sdkv2 provider instance.
func New(version string) *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "API key for authenticating with Devin API.",
			},
			"org_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Organization ID for scoping API requests.",
			},
			"base_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "https://api.devin.ai/v3",
				Description: "Base URL of the Devin API.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"devin_knowledge_note": resourceKnowledgeNote(),
		},
	}

	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		apiKey := d.Get("api_key").(string)
		orgID := d.Get("org_id").(string)
		baseURL := d.Get("base_url").(string)

		c := client.NewClient(baseURL, orgID, apiKey)
		return c, nil
	}

	return p
}
