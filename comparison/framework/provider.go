package framework

import (
	"context"

	"github.com/COG-GTM/terraform-provider-devin/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &DevinProvider{}

// DevinProvider implements the terraform-plugin-framework provider.
type DevinProvider struct {
	version string
}

// DevinProviderModel maps provider schema to a Go struct with typed fields.
type DevinProviderModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	OrgID   types.String `tfsdk:"org_id"`
	BaseURL types.String `tfsdk:"base_url"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DevinProvider{version: version}
	}
}

func (p *DevinProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "devin"
	resp.Version = p.version
}

func (p *DevinProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Devin API v3 resources.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "API key for authenticating with Devin API.",
				Required:    true,
				Sensitive:   true,
			},
			"org_id": schema.StringAttribute{
				Description: "Organization ID for scoping API requests.",
				Required:    true,
			},
			"base_url": schema.StringAttribute{
				Description: "Base URL of the Devin API.",
				Optional:    true,
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

	baseURL := "https://api.devin.ai/v3"
	if !config.BaseURL.IsNull() && !config.BaseURL.IsUnknown() {
		baseURL = config.BaseURL.ValueString()
	}

	c := client.NewClient(baseURL, config.OrgID.ValueString(), config.APIKey.ValueString())
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *DevinProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewKnowledgeNoteResource,
	}
}

func (p *DevinProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
