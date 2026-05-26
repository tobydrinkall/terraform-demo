// Option C: Optional org_id with fallback to /v3/self.
//
// If the user sets organization_id (or DEVIN_ORG_ID), it is used directly
// — no network call is made. If omitted, the provider calls GET /v3/self
// to auto-resolve it. This gives power users explicit control while keeping
// the simple case zero-config.
//
// Example HCL (minimal — auto-resolve):
//
//	provider "devin" {
//	  api_key = var.devin_api_key
//	}
//
// Example HCL (explicit):
//
//	provider "devin" {
//	  api_key         = var.devin_api_key
//	  organization_id = "org-abc123"
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

var _ provider.Provider = &OptionCProvider{}

type OptionCProvider struct{ version string }

type OptionCModel struct {
	APIKey         types.String `tfsdk:"api_key"`
	OrganizationID types.String `tfsdk:"organization_id"`
	BaseURL        types.String `tfsdk:"base_url"`
}

func NewOptionC(version string) func() provider.Provider {
	return func() provider.Provider { return &OptionCProvider{version: version} }
}

func (p *OptionCProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "devin"
	resp.Version = p.version
}

func (p *OptionCProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for Devin API v3. Organization ID is optional; auto-resolved from the API key if omitted.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Devin API key. Falls back to DEVIN_API_KEY env var.",
			},
			"organization_id": schema.StringAttribute{
				Optional: true,
				Description: "Devin organization ID. If omitted, the provider resolves it " +
					"automatically via GET /v3/self. Set this explicitly to skip the " +
					"extra API call or if you belong to multiple organizations. " +
					"Falls back to DEVIN_ORG_ID env var.",
			},
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL for the Devin API (default: https://api.devin.ai). Falls back to DEVIN_BASE_URL env var.",
			},
		},
	}
}

func (p *OptionCProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config OptionCModel
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

	// Resolve org_id: config > env > /v3/self auto-resolve
	orgID := os.Getenv("DEVIN_ORG_ID")
	if !config.OrganizationID.IsNull() {
		orgID = config.OrganizationID.ValueString()
	}

	if orgID == "" {
		resolved, err := resolveSelfOrgID(baseURL, apiKey)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to Auto-Resolve Organization",
				fmt.Sprintf(
					"organization_id was not set in the provider config or DEVIN_ORG_ID "+
						"env var, so the provider tried to resolve it via GET /v3/self. "+
						"This failed:\n\n  %s\n\n"+
						"Set organization_id explicitly in the provider block:\n\n"+
						"  provider \"devin\" {\n"+
						"    api_key         = var.devin_api_key\n"+
						"    organization_id = \"org-your-org-id\"\n"+
						"  }",
					err,
				),
			)
			return
		}
		orgID = resolved
	}

	c := client.NewClient(baseURL, orgID, apiKey)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *OptionCProvider) Resources(_ context.Context) []func() resource.Resource   { return nil }
func (p *OptionCProvider) DataSources(_ context.Context) []func() datasource.DataSource { return nil }
