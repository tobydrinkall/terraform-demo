// Option A: Required org_id in provider config.
//
// The user must explicitly set organization_id in their provider block
// (or via the DEVIN_ORG_ID env var). No network call is made during configure.
//
// Example HCL:
//
//	provider "devin" {
//	  api_key         = var.devin_api_key   # or DEVIN_API_KEY env var
//	  organization_id = "org-abc123"        # or DEVIN_ORG_ID env var
//	}
package poc

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tobydrinkall/terraform-demo/internal/client"
)

var _ provider.Provider = &OptionAProvider{}

type OptionAProvider struct{ version string }

type OptionAModel struct {
	APIKey         types.String `tfsdk:"api_key"`
	OrganizationID types.String `tfsdk:"organization_id"`
	BaseURL        types.String `tfsdk:"base_url"`
}

func NewOptionA(version string) func() provider.Provider {
	return func() provider.Provider { return &OptionAProvider{version: version} }
}

func (p *OptionAProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "devin"
	resp.Version = p.version
}

func (p *OptionAProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for Devin API v3. Organization ID is required.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Devin API key. Falls back to DEVIN_API_KEY env var.",
			},
			"organization_id": schema.StringAttribute{
				Required:    true,
				Description: "Devin organization ID (e.g. org-abc123). Required for all API calls.",
			},
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL for the Devin API (default: https://api.devin.ai). Falls back to DEVIN_BASE_URL env var.",
			},
		},
	}
}

func (p *OptionAProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config OptionAModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve api_key: config > env
	apiKey := os.Getenv("DEVIN_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"Set api_key in the provider block or the DEVIN_API_KEY environment variable.",
		)
	}

	// org_id is Required — framework enforces presence, but validate non-empty
	orgID := config.OrganizationID.ValueString()
	if orgID == "" {
		resp.Diagnostics.AddError(
			"Missing Organization ID",
			"organization_id is required. Set it in the provider block.",
		)
	}

	// Resolve base_url: config > env > default
	baseURL := os.Getenv("DEVIN_BASE_URL")
	if !config.BaseURL.IsNull() {
		baseURL = config.BaseURL.ValueString()
	}

	if resp.Diagnostics.HasError() {
		return
	}

	c := client.NewClient(baseURL, orgID, apiKey)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *OptionAProvider) Resources(_ context.Context) []func() resource.Resource   { return nil }
func (p *OptionAProvider) DataSources(_ context.Context) []func() datasource.DataSource { return nil }
