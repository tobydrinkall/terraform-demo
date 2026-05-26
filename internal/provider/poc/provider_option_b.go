// Option B: Auto-resolve org_id via GET /v3/self.
//
// The provider has NO organization_id attribute. During Configure, it calls
// GET /v3/self with the API key and extracts the org_id from the response.
//
// Example HCL:
//
//	provider "devin" {
//	  api_key = var.devin_api_key   # or DEVIN_API_KEY env var
//	  # org_id is auto-resolved — zero config
//	}
package poc

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tobydrinkall/terraform-demo/internal/client"
)

var _ provider.Provider = &OptionBProvider{}

type OptionBProvider struct{ version string }

type OptionBModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	BaseURL types.String `tfsdk:"base_url"`
}

func NewOptionB(version string) func() provider.Provider {
	return func() provider.Provider { return &OptionBProvider{version: version} }
}

func (p *OptionBProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "devin"
	resp.Version = p.version
}

func (p *OptionBProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for Devin API v3. Organization is auto-resolved from the API key.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Devin API key. Falls back to DEVIN_API_KEY env var.",
			},
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL for the Devin API (default: https://api.devin.ai). Falls back to DEVIN_BASE_URL env var.",
			},
		},
	}
}

func (p *OptionBProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config OptionBModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("DEVIN_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"Set api_key in the provider block or the DEVIN_API_KEY environment variable.",
		)
		return
	}

	baseURL := os.Getenv("DEVIN_BASE_URL")
	if !config.BaseURL.IsNull() {
		baseURL = config.BaseURL.ValueString()
	}
	if baseURL == "" {
		baseURL = "https://api.devin.ai"
	}

	// Auto-resolve org_id from the API key
	orgID, err := resolveSelfOrgID(baseURL, apiKey)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Auto-Resolve Organization",
			fmt.Sprintf(
				"The provider called GET /v3/self to determine your organization, "+
					"but received an error:\n\n  %s\n\n"+
					"Ensure your API key is valid and has organization membership.",
				err,
			),
		)
		return
	}

	c := client.NewClient(baseURL, orgID, apiKey)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *OptionBProvider) Resources(_ context.Context) []func() resource.Resource   { return nil }
func (p *OptionBProvider) DataSources(_ context.Context) []func() datasource.DataSource { return nil }
