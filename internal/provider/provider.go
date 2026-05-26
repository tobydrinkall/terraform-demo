package provider

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

var _ provider.Provider = &DevinProvider{}

// DevinProvider implements the Terraform provider for Devin API v3.
type DevinProvider struct {
	version string
}

// DevinProviderModel maps provider schema data.
type DevinProviderModel struct {
	APIKey         types.String `tfsdk:"api_key"`
	OrganizationID types.String `tfsdk:"organization_id"`
	BaseURL        types.String `tfsdk:"base_url"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DevinProvider{
			version: version,
		}
	}
}

func (p *DevinProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "devin"
	resp.Version = p.version
}

func (p *DevinProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Devin resources via the API v3.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Devin API key (service user credential with cog_ prefix). Can also be set via DEVIN_API_KEY environment variable.",
			},
			"organization_id": schema.StringAttribute{
				Optional:    true,
				Description: "Devin organization ID. Can also be set via DEVIN_ORG_ID environment variable.",
			},
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL for the Devin API (default: https://api.devin.ai). Can also be set via DEVIN_BASE_URL environment variable.",
			},
		},
	}
}

func (p *DevinProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config DevinProviderModel
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
			"The Devin API key must be set in the provider configuration or via the DEVIN_API_KEY environment variable.",
		)
	}

	orgID := os.Getenv("DEVIN_ORG_ID")
	if !config.OrganizationID.IsNull() {
		orgID = config.OrganizationID.ValueString()
	}
	if orgID == "" {
		resp.Diagnostics.AddError(
			"Missing Organization ID",
			"The Devin organization ID must be set in the provider configuration or via the DEVIN_ORG_ID environment variable.",
		)
	}

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

func (p *DevinProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewKnowledgeNoteResource,
	}
}

func (p *DevinProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewKnowledgeNoteDataSource,
		NewKnowledgeNotesDataSource,
	}
}
