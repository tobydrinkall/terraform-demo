package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tobydrinkall/terraform-demo/internal/client"
)

var _ datasource.DataSource = &KnowledgeNoteDataSource{}

type KnowledgeNoteDataSource struct {
	client *client.Client
}

type KnowledgeNoteDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	NoteID     types.String `tfsdk:"note_id"`
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

func NewKnowledgeNoteDataSource() datasource.DataSource {
	return &KnowledgeNoteDataSource{}
}

func (d *KnowledgeNoteDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_knowledge_note"
}

func (d *KnowledgeNoteDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a single Devin Knowledge Note by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The note ID.",
			},
			"note_id": schema.StringAttribute{
				Required:    true,
				Description: "The note ID to look up (e.g., note-abc123def456).",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The note name.",
			},
			"body": schema.StringAttribute{
				Computed:    true,
				Description: "The note content.",
			},
			"trigger": schema.StringAttribute{
				Computed:    true,
				Description: "The note trigger.",
			},
			"pinned_repo": schema.StringAttribute{
				Computed:    true,
				Description: "The pinned repo (if any).",
			},
			"folder_path": schema.StringAttribute{
				Computed:    true,
				Description: "The folder path.",
			},
			"is_enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the note is enabled.",
			},
			"access_type": schema.StringAttribute{
				Computed:    true,
				Description: "Access type.",
			},
			"created_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Unix timestamp of creation.",
			},
			"updated_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Unix timestamp of last update.",
			},
		},
	}
}

func (d *KnowledgeNoteDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *KnowledgeNoteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data KnowledgeNoteDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	note, err := d.client.GetKnowledgeNote(ctx, data.NoteID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading knowledge note", err.Error())
		return
	}

	data.ID = types.StringValue(note.NoteID)
	data.Name = types.StringValue(note.Name)
	data.Body = types.StringValue(note.Body)
	data.Trigger = types.StringValue(note.Trigger)
	data.FolderPath = types.StringValue(note.FolderPath)
	data.IsEnabled = types.BoolValue(note.IsEnabled)
	data.AccessType = types.StringValue(note.AccessType)
	data.CreatedAt = types.Int64Value(note.CreatedAt)
	data.UpdatedAt = types.Int64Value(note.UpdatedAt)

	if note.PinnedRepo != nil {
		data.PinnedRepo = types.StringValue(*note.PinnedRepo)
	} else {
		data.PinnedRepo = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
