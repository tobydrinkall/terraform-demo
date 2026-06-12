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

var _ datasource.DataSource = &KnowledgeNotesDataSource{}

type KnowledgeNotesDataSource struct {
	client *client.Client
	orgID  string
}

type KnowledgeNotesDataSourceModel struct {
	Search     types.String           `tfsdk:"search"`
	FolderPath types.String           `tfsdk:"folder_path"`
	PinnedRepo types.String           `tfsdk:"pinned_repo"`
	Notes      []KnowledgeNoteModel   `tfsdk:"notes"`
}

type KnowledgeNoteModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Body       types.String `tfsdk:"body"`
	Trigger    types.String `tfsdk:"trigger"`
	PinnedRepo types.String `tfsdk:"pinned_repo"`
	FolderPath types.String `tfsdk:"folder_path"`
	IsEnabled  types.Bool   `tfsdk:"is_enabled"`
	AccessType types.String `tfsdk:"access_type"`
	CreatedAt  types.Int64  `tfsdk:"created_at"`
	UpdatedAt  types.Int64  `tfsdk:"updated_at"`
}

func NewKnowledgeNotesDataSource() datasource.DataSource {
	return &KnowledgeNotesDataSource{}
}

func (d *KnowledgeNotesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_knowledge_notes"
}

func (d *KnowledgeNotesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List Devin knowledge notes with optional filtering.",
		Attributes: map[string]schema.Attribute{
			"search": schema.StringAttribute{
				Description: "Search query (case-insensitive substring match across name, trigger, and content).",
				Optional:    true,
			},
			"folder_path": schema.StringAttribute{
				Description: "Filter to notes in a specific folder (e.g. 'Dana/SubFolder' or '/' for root).",
				Optional:    true,
			},
			"pinned_repo": schema.StringAttribute{
				Description: "Filter to notes pinned to a specific repo (owner/repo format).",
				Optional:    true,
			},
			"notes": schema.ListNestedAttribute{
				Description: "List of knowledge notes matching the filter criteria.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.StringAttribute{Computed: true},
						"name":        schema.StringAttribute{Computed: true},
						"body":        schema.StringAttribute{Computed: true},
						"trigger":     schema.StringAttribute{Computed: true},
						"pinned_repo": schema.StringAttribute{Computed: true},
						"folder_path": schema.StringAttribute{Computed: true},
						"is_enabled":  schema.BoolAttribute{Computed: true},
						"access_type": schema.StringAttribute{Computed: true},
						"created_at":  schema.Int64Attribute{Computed: true},
						"updated_at":  schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *KnowledgeNotesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *KnowledgeNotesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config KnowledgeNotesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := &client.ListNotesOptions{}
	if !config.Search.IsNull() && !config.Search.IsUnknown() {
		opts.Search = config.Search.ValueString()
	}
	if !config.FolderPath.IsNull() && !config.FolderPath.IsUnknown() {
		opts.FolderPath = config.FolderPath.ValueString()
	}
	if !config.PinnedRepo.IsNull() && !config.PinnedRepo.IsUnknown() {
		opts.PinnedRepo = config.PinnedRepo.ValueString()
	}
	opts.First = 200

	var allNotes []client.KnowledgeNoteResponse
	for {
		result, err := d.client.ListNotes(ctx, d.orgID, opts)
		if err != nil {
			resp.Diagnostics.AddError("Error listing knowledge notes", err.Error())
			return
		}
		allNotes = append(allNotes, result.Notes...)
		if !result.HasNextPage || result.EndCursor == nil {
			break
		}
		opts.After = *result.EndCursor
	}

	notes := make([]KnowledgeNoteModel, len(allNotes))
	for i, n := range allNotes {
		notes[i] = KnowledgeNoteModel{
			ID:         types.StringValue(n.NoteID),
			Name:       types.StringValue(n.Name),
			Body:       types.StringValue(n.Body),
			Trigger:    types.StringValue(n.Trigger),
			FolderPath: types.StringValue(n.FolderPath),
			IsEnabled:  types.BoolValue(n.IsEnabled),
			AccessType: types.StringValue(n.AccessType),
			CreatedAt:  types.Int64Value(n.CreatedAt),
			UpdatedAt:  types.Int64Value(n.UpdatedAt),
		}
		if n.PinnedRepo != nil {
			notes[i].PinnedRepo = types.StringValue(*n.PinnedRepo)
		} else {
			notes[i].PinnedRepo = types.StringNull()
		}
	}

	config.Notes = notes
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
