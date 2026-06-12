package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tobydrinkall/terraform-provider-devin/internal/client"
	"github.com/tobydrinkall/terraform-provider-devin/internal/resources"
	sharedtypes "github.com/tobydrinkall/terraform-provider-devin/internal/types"
)

var _ provider.Provider = &DevinProvider{}

// DevinProvider implements the Devin Terraform provider.
type DevinProvider struct {
	version string
}

// DevinProviderModel describes the provider data model.
type DevinProviderModel struct {
	APIKey         types.String `tfsdk:"api_key"`
	OrganizationID types.String `tfsdk:"organization_id"`
	BaseURL        types.String `tfsdk:"base_url"`
}

// New returns a provider.Provider factory function.
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
		Description: "The Devin provider allows you to manage Devin resources via the Devin API v3.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "The Devin API key (service user token starting with cog_). Can also be set via DEVIN_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The Devin organization ID (e.g. org-abc123). Can also be set via DEVIN_ORG_ID environment variable.",
				Optional:    true,
			},
			"base_url": schema.StringAttribute{
				Description: "The Devin API base URL. Defaults to https://api.devin.ai. Can also be set via DEVIN_BASE_URL environment variable.",
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

	// Resolve API key
	apiKey := os.Getenv("DEVIN_API_KEY")
	if !config.APIKey.IsNull() && !config.APIKey.IsUnknown() {
		apiKey = config.APIKey.ValueString()
	}
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"The Devin API key must be configured via the api_key provider attribute or DEVIN_API_KEY environment variable.",
		)
		return
	}

	// Resolve organization ID
	orgID := os.Getenv("DEVIN_ORG_ID")
	if !config.OrganizationID.IsNull() && !config.OrganizationID.IsUnknown() {
		orgID = config.OrganizationID.ValueString()
	}
	if orgID == "" {
		resp.Diagnostics.AddError(
			"Missing Organization ID",
			"The Devin organization ID must be configured via the organization_id provider attribute or DEVIN_ORG_ID environment variable.",
		)
		return
	}

	// Resolve base URL
	baseURL := os.Getenv("DEVIN_BASE_URL")
	if !config.BaseURL.IsNull() && !config.BaseURL.IsUnknown() {
		baseURL = config.BaseURL.ValueString()
	}

	var clientOpts []client.Option
	if baseURL != "" {
		clientOpts = append(clientOpts, client.WithBaseURL(baseURL))
	}
	clientOpts = append(clientOpts, client.WithUserAgent("terraform-provider-devin/"+p.version))

	c := client.New(apiKey, clientOpts...)

	providerData := &sharedtypes.ProviderData{
		Client:         c,
		OrganizationID: orgID,
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *DevinProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewKnowledgeNoteResource,
	}
}

func (p *DevinProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
