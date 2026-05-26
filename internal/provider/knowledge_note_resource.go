package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tobydrinkall/terraform-demo/internal/client"
)

var (
	_ resource.Resource                = &KnowledgeNoteResource{}
	_ resource.ResourceWithImportState = &KnowledgeNoteResource{}
)

type KnowledgeNoteResource struct {
	client *client.Client
}

type KnowledgeNoteResourceModel struct {
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

func NewKnowledgeNoteResource() resource.Resource {
	return &KnowledgeNoteResource{}
}

func (r *KnowledgeNoteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_knowledge_note"
}

func (r *KnowledgeNoteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Devin Knowledge Note.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The note ID (e.g., note-abc123def456).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name/title of the knowledge note.",
			},
			"body": schema.StringAttribute{
				Required:    true,
				Description: "The content of the knowledge note (supports markdown).",
			},
			"trigger": schema.StringAttribute{
				Required:    true,
				Description: "When this knowledge should be retrieved (the scope/trigger description).",
			},
			"pinned_repo": schema.StringAttribute{
				Optional:    true,
				Description: "Pin this note to a specific repo (owner/repo format).",
			},
			"folder_path": schema.StringAttribute{
				Computed:    true,
				Description: "The folder path of the note.",
			},
			"is_enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the note is enabled.",
			},
			"access_type": schema.StringAttribute{
				Computed:    true,
				Description: "Access type: 'org' or 'enterprise'.",
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

func (r *KnowledgeNoteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *KnowledgeNoteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan KnowledgeNoteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.KnowledgeNoteCreateRequest{
		Name:    plan.Name.ValueString(),
		Body:    plan.Body.ValueString(),
		Trigger: plan.Trigger.ValueString(),
	}
	if !plan.PinnedRepo.IsNull() {
		v := plan.PinnedRepo.ValueString()
		createReq.PinnedRepo = &v
	}

	note, err := r.client.CreateKnowledgeNote(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating knowledge note", err.Error())
		return
	}

	mapNoteToState(note, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KnowledgeNoteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state KnowledgeNoteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	note, err := r.client.GetKnowledgeNote(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading knowledge note", err.Error())
		return
	}

	mapNoteToState(note, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *KnowledgeNoteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan KnowledgeNoteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state KnowledgeNoteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.KnowledgeNoteCreateRequest{
		Name:    plan.Name.ValueString(),
		Body:    plan.Body.ValueString(),
		Trigger: plan.Trigger.ValueString(),
	}
	if !plan.PinnedRepo.IsNull() {
		v := plan.PinnedRepo.ValueString()
		updateReq.PinnedRepo = &v
	}

	note, err := r.client.UpdateKnowledgeNote(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating knowledge note", err.Error())
		return
	}

	mapNoteToState(note, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KnowledgeNoteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state KnowledgeNoteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteKnowledgeNote(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting knowledge note", err.Error())
		return
	}
}

func (r *KnowledgeNoteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapNoteToState(note *client.KnowledgeNote, model *KnowledgeNoteResourceModel) {
	model.ID = types.StringValue(note.NoteID)
	model.Name = types.StringValue(note.Name)
	model.Body = types.StringValue(note.Body)
	model.Trigger = types.StringValue(note.Trigger)
	model.FolderPath = types.StringValue(note.FolderPath)
	model.IsEnabled = types.BoolValue(note.IsEnabled)
	model.AccessType = types.StringValue(note.AccessType)
	model.CreatedAt = types.Int64Value(note.CreatedAt)
	model.UpdatedAt = types.Int64Value(note.UpdatedAt)

	if note.PinnedRepo != nil {
		model.PinnedRepo = types.StringValue(*note.PinnedRepo)
	} else {
		model.PinnedRepo = types.StringNull()
	}
}
