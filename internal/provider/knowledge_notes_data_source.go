package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tobydrinkall/terraform-demo/internal/client"
)

var _ datasource.DataSource = &KnowledgeNotesDataSource{}

type KnowledgeNotesDataSource struct {
	client *client.Client
}

type KnowledgeNotesDataSourceModel struct {
	ID         types.String                   `tfsdk:"id"`
	Search     types.String                   `tfsdk:"search"`
	FolderPath types.String                   `tfsdk:"folder_path"`
	PinnedRepo types.String                   `tfsdk:"pinned_repo"`
	Notes      []KnowledgeNoteDataSourceModel `tfsdk:"notes"`
}

func NewKnowledgeNotesDataSource() datasource.DataSource {
	return &KnowledgeNotesDataSource{}
}

func (d *KnowledgeNotesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_knowledge_notes"
}

func (d *KnowledgeNotesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List Devin Knowledge Notes with optional filters.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Placeholder ID for the data source.",
			},
			"search": schema.StringAttribute{
				Optional:    true,
				Description: "Case-insensitive substring search across note name, trigger, and body.",
			},
			"folder_path": schema.StringAttribute{
				Optional:    true,
				Description: "Filter by folder path.",
			},
			"pinned_repo": schema.StringAttribute{
				Optional:    true,
				Description: "Filter by pinned repository.",
			},
			"notes": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of matching knowledge notes.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"note_id": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"body": schema.StringAttribute{
							Computed: true,
						},
						"trigger": schema.StringAttribute{
							Computed: true,
						},
						"pinned_repo": schema.StringAttribute{
							Computed: true,
						},
						"folder_path": schema.StringAttribute{
							Computed: true,
						},
						"is_enabled": schema.BoolAttribute{
							Computed: true,
						},
						"access_type": schema.StringAttribute{
							Computed: true,
						},
						"created_at": schema.Int64Attribute{
							Computed: true,
						},
						"updated_at": schema.Int64Attribute{
							Computed: true,
						},
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
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData))
		return
	}
	d.client = c
}

func (d *KnowledgeNotesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KnowledgeNotesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := client.ListKnowledgeNotesOptions{}
	if !data.Search.IsNull() {
		v := data.Search.ValueString()
		opts.Search = &v
	}
	if !data.FolderPath.IsNull() {
		v := data.FolderPath.ValueString()
		opts.FolderPath = &v
	}
	if !data.PinnedRepo.IsNull() {
		v := data.PinnedRepo.ValueString()
		opts.PinnedRepo = &v
	}

	notes, err := d.client.ListKnowledgeNotes(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Error listing knowledge notes", err.Error())
		return
	}

	data.ID = types.StringValue("knowledge_notes")
	data.Notes = make([]KnowledgeNoteDataSourceModel, len(notes))
	for i, note := range notes {
		data.Notes[i] = KnowledgeNoteDataSourceModel{
			ID:         types.StringValue(note.NoteID),
			NoteID:     types.StringValue(note.NoteID),
			Name:       types.StringValue(note.Name),
			Body:       types.StringValue(note.Body),
			Trigger:    types.StringValue(note.Trigger),
			FolderPath: types.StringValue(note.FolderPath),
			IsEnabled:  types.BoolValue(note.IsEnabled),
			AccessType: types.StringValue(note.AccessType),
			CreatedAt:  types.Int64Value(note.CreatedAt),
			UpdatedAt:  types.Int64Value(note.UpdatedAt),
		}
		if note.PinnedRepo != nil {
			data.Notes[i].PinnedRepo = types.StringValue(*note.PinnedRepo)
		} else {
			data.Notes[i].PinnedRepo = types.StringNull()
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
