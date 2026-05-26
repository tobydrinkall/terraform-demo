package framework_poc

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &DevinProvider{}

type DevinProvider struct {
	version string
}

type DevinProviderModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	BaseURL types.String `tfsdk:"base_url"`
}

type apiClient struct {
	apiKey  string
	baseURL string
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
		Description: "Terraform provider for the Devin AI platform API v3.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The API key for authenticating with the Devin API v3. Can also be set via the `DEVIN_API_KEY` environment variable.",
			},
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "The base URL of the Devin API. Defaults to `https://api.devin.ai/v3`.",
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

	apiKey := config.APIKey.ValueString()
	baseURL := config.BaseURL.ValueString()
	if baseURL == "" {
		baseURL = "https://api.devin.ai/v3"
	}

	client := &apiClient{
		apiKey:  apiKey,
		baseURL: baseURL,
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *DevinProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewKnowledgeNoteResource,
	}
}

func (p *DevinProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewKnowledgeNotesDataSource,
	}
}
