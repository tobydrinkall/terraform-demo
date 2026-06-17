package framework

import (
	"context"
	"fmt"

	"github.com/COG-GTM/terraform-provider-devin/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &KnowledgeNoteResource{}
	_ resource.ResourceWithImportState = &KnowledgeNoteResource{}
)

type KnowledgeNoteResource struct {
	client *client.Client
}

// KnowledgeNoteResourceModel is the typed model for the resource state.
// Framework uses strongly-typed struct fields mapped via `tfsdk` tags.
type KnowledgeNoteResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Trigger    types.String `tfsdk:"trigger"`
	Body       types.String `tfsdk:"body"`
	FolderID   types.String `tfsdk:"folder_id"`
	FolderPath types.String `tfsdk:"folder_path"`
	IsEnabled  types.Bool   `tfsdk:"is_enabled"`
	Macro      types.String `tfsdk:"macro"`
	OrgID      types.String `tfsdk:"org_id"`
	PinnedRepo types.String `tfsdk:"pinned_repo"`
	AccessType types.String `tfsdk:"access_type"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

func NewKnowledgeNoteResource() resource.Resource {
	return &KnowledgeNoteResource{}
}

func (r *KnowledgeNoteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_knowledge_note"
}

func (r *KnowledgeNoteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Devin knowledge note.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the knowledge note.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name/title of the knowledge note.",
				Required:    true,
			},
			"trigger": schema.StringAttribute{
				Description: "When this knowledge should be retrieved (scope description).",
				Required:    true,
			},
			"body": schema.StringAttribute{
				Description: "The content of the knowledge note.",
				Required:    true,
			},
			"folder_id": schema.StringAttribute{
				Description: "Optional folder ID to organize the note.",
				Optional:    true,
			},
			"folder_path": schema.StringAttribute{
				Description: "The computed folder path.",
				Computed:    true,
			},
			"is_enabled": schema.BoolAttribute{
				Description: "Whether the note is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"macro": schema.StringAttribute{
				Description: "The macro identifier for the note.",
				Computed:    true,
			},
			"org_id": schema.StringAttribute{
				Description: "The organization ID that owns this note.",
				Computed:    true,
			},
			"pinned_repo": schema.StringAttribute{
				Description: "Optional repository to pin this note to.",
				Optional:    true,
			},
			"access_type": schema.StringAttribute{
				Description: "The access type for the note.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the note was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the note was last updated.",
				Computed:    true,
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
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
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

	apiReq := client.KnowledgeNoteRequest{
		Name:    plan.Name.ValueString(),
		Trigger: plan.Trigger.ValueString(),
		Body:    plan.Body.ValueString(),
	}
	if !plan.FolderID.IsNull() {
		v := plan.FolderID.ValueString()
		apiReq.FolderID = &v
	}
	if !plan.IsEnabled.IsNull() {
		v := plan.IsEnabled.ValueBool()
		apiReq.IsEnabled = &v
	}
	if !plan.PinnedRepo.IsNull() {
		v := plan.PinnedRepo.ValueString()
		apiReq.PinnedRepo = &v
	}

	note, err := r.client.CreateKnowledgeNote(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating knowledge note", err.Error())
		return
	}

	mapNoteToModel(note, &plan)
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
		resp.Diagnostics.AddError("Error reading knowledge note", err.Error())
		return
	}

	mapNoteToModel(note, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *KnowledgeNoteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan KnowledgeNoteResourceModel
	var state KnowledgeNoteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := client.KnowledgeNoteRequest{
		Name:    plan.Name.ValueString(),
		Trigger: plan.Trigger.ValueString(),
		Body:    plan.Body.ValueString(),
	}
	if !plan.FolderID.IsNull() {
		v := plan.FolderID.ValueString()
		apiReq.FolderID = &v
	}
	if !plan.IsEnabled.IsNull() {
		v := plan.IsEnabled.ValueBool()
		apiReq.IsEnabled = &v
	}
	if !plan.PinnedRepo.IsNull() {
		v := plan.PinnedRepo.ValueString()
		apiReq.PinnedRepo = &v
	}

	note, err := r.client.UpdateKnowledgeNote(ctx, state.ID.ValueString(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating knowledge note", err.Error())
		return
	}

	mapNoteToModel(note, &plan)
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

func mapNoteToModel(note *client.KnowledgeNote, model *KnowledgeNoteResourceModel) {
	model.ID = types.StringValue(note.NoteID)
	model.Name = types.StringValue(note.Name)
	model.Trigger = types.StringValue(note.Trigger)
	model.Body = types.StringValue(note.Body)
	model.IsEnabled = types.BoolValue(note.IsEnabled)
	model.OrgID = types.StringValue(note.OrgID)
	model.CreatedAt = types.StringValue(note.CreatedAt)
	model.UpdatedAt = types.StringValue(note.UpdatedAt)

	if note.FolderID != "" {
		model.FolderID = types.StringValue(note.FolderID)
	} else {
		model.FolderID = types.StringNull()
	}
	if note.FolderPath != "" {
		model.FolderPath = types.StringValue(note.FolderPath)
	} else {
		model.FolderPath = types.StringValue("")
	}
	if note.Macro != "" {
		model.Macro = types.StringValue(note.Macro)
	} else {
		model.Macro = types.StringValue("")
	}
	if note.PinnedRepo != "" {
		model.PinnedRepo = types.StringValue(note.PinnedRepo)
	} else {
		model.PinnedRepo = types.StringNull()
	}
	if note.AccessType != "" {
		model.AccessType = types.StringValue(note.AccessType)
	} else {
		model.AccessType = types.StringValue("")
	}
}
