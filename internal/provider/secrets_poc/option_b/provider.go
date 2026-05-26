package option_b

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	fwschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &secretProvider{}

type secretProvider struct{}

func NewProvider() provider.Provider {
	return &secretProvider{}
}

func (p *secretProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "devinsecret"
}

func (p *secretProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = fwschema.Schema{
		Description: "PoC provider for Option B (WriteOnly secret).",
		Attributes: map[string]fwschema.Attribute{
			"api_key": fwschema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The API key (mock for PoC).",
			},
		},
	}
}

func (p *secretProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}

func (p *secretProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSecretResource,
	}
}

func (p *secretProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
