package framework_poc

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &KnowledgeNotesDataSource{}
	_ datasource.DataSourceWithConfigure = &KnowledgeNotesDataSource{}
)

type KnowledgeNotesDataSource struct {
	client *apiClient
}

type KnowledgeNotesDataSourceModel struct {
	Scope      types.String              `tfsdk:"scope"`
	NameFilter types.String              `tfsdk:"name_filter"`
	RepoName   types.String              `tfsdk:"repo_name"`
	ID         types.String              `tfsdk:"id"`
	Notes      []KnowledgeNoteItemModel  `tfsdk:"notes"`
}

type KnowledgeNoteItemModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Content   types.String `tfsdk:"content"`
	Trigger   types.String `tfsdk:"trigger"`
	Scope     types.String `tfsdk:"scope"`
	Author    types.String `tfsdk:"author"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	Size      types.Int64  `tfsdk:"size"`
}

func NewKnowledgeNotesDataSource() datasource.DataSource {
	return &KnowledgeNotesDataSource{}
}

func (d *KnowledgeNotesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_knowledge_notes"
}

func (d *KnowledgeNotesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of knowledge notes from the Devin platform. Use this data source to look up existing knowledge notes by scope, name pattern, or repository.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Placeholder identifier for this data source.",
			},
			"scope": schema.StringAttribute{
				Optional:    true,
				Description: "Filter notes by visibility scope. Valid values are `organization`, `user`, or `repo`. Defaults to `organization`.",
				Validators: []validator.String{
					stringvalidator.OneOf("organization", "user", "repo"),
				},
			},
			"name_filter": schema.StringAttribute{
				Optional:    true,
				Description: "A substring filter applied to note names. Only notes whose name contains this string (case-insensitive) will be returned.",
			},
			"repo_name": schema.StringAttribute{
				Optional:    true,
				Description: "Filter notes by repository. Only applicable when `scope` is `repo`. Format: `owner/repo`.",
			},
		},

		Blocks: map[string]schema.Block{},
	}

	// Notes is a nested list attribute — defined separately for clarity
	resp.Schema.Attributes["notes"] = schema.ListNestedAttribute{
		Computed:    true,
		Description: "The list of knowledge notes matching the filter criteria.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Computed:    true,
					Description: "The unique identifier of the knowledge note.",
				},
				"name": schema.StringAttribute{
					Computed:    true,
					Description: "The display name of the knowledge note.",
				},
				"content": schema.StringAttribute{
					Computed:    true,
					Description: "The body content of the knowledge note.",
				},
				"trigger": schema.StringAttribute{
					Computed:    true,
					Description: "The trigger description for the knowledge note.",
				},
				"scope": schema.StringAttribute{
					Computed:    true,
					Description: "The visibility scope of the knowledge note.",
				},
				"author": schema.StringAttribute{
					Computed:    true,
					Description: "The author of the knowledge note.",
				},
				"created_at": schema.StringAttribute{
					Computed:    true,
					Description: "The ISO 8601 timestamp when the note was created.",
				},
				"updated_at": schema.StringAttribute{
					Computed:    true,
					Description: "The ISO 8601 timestamp when the note was last updated.",
				},
				"size": schema.Int64Attribute{
					Computed:    true,
					Description: "The size of the note content in characters.",
				},
			},
		},
	}
}

func (d *KnowledgeNotesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*apiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			"Expected *apiClient, got unexpected type.",
		)
		return
	}
	d.client = client
}

func (d *KnowledgeNotesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config KnowledgeNotesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Stub: return empty list
	config.ID = types.StringValue("knowledge-notes-list")
	config.Notes = []KnowledgeNoteItemModel{}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
