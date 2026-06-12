package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tobydrinkall/terraform-provider-devin/internal/client"
	sharedtypes "github.com/tobydrinkall/terraform-provider-devin/internal/types"
)

var _ datasource.DataSource = &PlaybooksDataSource{}

type PlaybooksDataSource struct {
	client *client.Client
	orgID  string
}

type PlaybooksDataSourceModel struct {
	Playbooks []PlaybookModel `tfsdk:"playbooks"`
}

type PlaybookModel struct {
	ID        types.String `tfsdk:"id"`
	Title     types.String `tfsdk:"title"`
	Content   types.String `tfsdk:"content"`
	Macro     types.String `tfsdk:"macro"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func NewPlaybooksDataSource() datasource.DataSource {
	return &PlaybooksDataSource{}
}

func (d *PlaybooksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbooks"
}

func (d *PlaybooksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all Devin playbooks in the organization.",
		Attributes: map[string]schema.Attribute{
			"playbooks": schema.ListNestedAttribute{
				Description: "List of playbooks.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.StringAttribute{Computed: true},
						"title":      schema.StringAttribute{Computed: true},
						"content":    schema.StringAttribute{Computed: true},
						"macro":      schema.StringAttribute{Computed: true},
						"created_at": schema.Int64Attribute{Computed: true},
						"updated_at": schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *PlaybooksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*sharedtypes.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *types.ProviderData, got: %T", req.ProviderData),
		)
		return
	}
	d.client = pd.Client
	d.orgID = pd.OrganizationID
}

func (d *PlaybooksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PlaybooksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var allPlaybooks []client.PlaybookResponse
	var cursor string
	for {
		result, err := d.client.ListPlaybooks(ctx, d.orgID, cursor, 200)
		if err != nil {
			resp.Diagnostics.AddError("Error listing playbooks", err.Error())
			return
		}
		allPlaybooks = append(allPlaybooks, result.Playbooks...)
		if !result.HasNextPage || result.EndCursor == nil {
			break
		}
		cursor = *result.EndCursor
	}

	playbooks := make([]PlaybookModel, len(allPlaybooks))
	for i, pb := range allPlaybooks {
		playbooks[i] = PlaybookModel{
			ID:        types.StringValue(pb.PlaybookID),
			Title:     types.StringValue(pb.Title),
			Content:   types.StringValue(pb.Content),
			CreatedAt: types.Int64Value(pb.CreatedAt),
			UpdatedAt: types.Int64Value(pb.UpdatedAt),
		}
		if pb.Macro != nil {
			playbooks[i].Macro = types.StringValue(*pb.Macro)
		} else {
			playbooks[i].Macro = types.StringNull()
		}
	}

	config.Playbooks = playbooks
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
