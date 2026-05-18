package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// New returns a new Devin provider.
func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("DEVIN_API_KEY", nil),
				Description: "The API key for authenticating with the Devin API v3. Can also be set via the `DEVIN_API_KEY` environment variable.",
			},
			"base_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "https://api.devin.ai/v3",
				Description: "The base URL of the Devin API. Defaults to `https://api.devin.ai/v3`.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"devin_knowledge_note": resourceKnowledgeNote(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"devin_knowledge_notes": dataSourceKnowledgeNotes(),
		},
		ConfigureContextFunc: configureProvider,
	}
}

func configureProvider(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	apiKey := d.Get("api_key").(string)
	baseURL := d.Get("base_url").(string)

	return &apiClient{
		apiKey:  apiKey,
		baseURL: baseURL,
	}, nil
}

type apiClient struct {
	apiKey  string
	baseURL string
}
