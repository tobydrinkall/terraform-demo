package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tobydrinkall/terraform-provider-devin/internal/client"
	sharedtypes "github.com/tobydrinkall/terraform-provider-devin/internal/types"
)

var (
	_ resource.Resource                = &KnowledgeNoteResource{}
	_ resource.ResourceWithImportState = &KnowledgeNoteResource{}
)

// KnowledgeNoteResource manages a Devin knowledge note.
type KnowledgeNoteResource struct {
	client *client.Client
	orgID  string
}

// KnowledgeNoteResourceModel describes the resource data model.
type KnowledgeNoteResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Body       types.String `tfsdk:"body"`
	Trigger    types.String `tfsdk:"trigger"`
	PinnedRepo types.String `tfsdk:"pinned_repo"`
	FolderID   types.String `tfsdk:"folder_id"`
	FolderPath types.String `tfsdk:"folder_path"`
	IsEnabled  types.Bool   `tfsdk:"is_enabled"`
	Macro      types.String `tfsdk:"macro"`
	AccessType types.String `tfsdk:"access_type"`
	OrgID      types.String `tfsdk:"org_id"`
	CreatedAt  types.Int64  `tfsdk:"created_at"`
	UpdatedAt  types.Int64  `tfsdk:"updated_at"`
}

// NewKnowledgeNoteResource returns a new resource.Resource.
func NewKnowledgeNoteResource() resource.Resource {
	return &KnowledgeNoteResource{}
}

func (r *KnowledgeNoteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_knowledge_note"
}

func (r *KnowledgeNoteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Devin knowledge note. Knowledge notes provide contextual information that Devin uses when working on tasks.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the knowledge note (note_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the knowledge note.",
				Required:    true,
			},
			"body": schema.StringAttribute{
				Description: "The content body of the knowledge note.",
				Required:    true,
			},
			"trigger": schema.StringAttribute{
				Description: "A description of when this knowledge should be retrieved (the scope/trigger).",
				Required:    true,
			},
			"pinned_repo": schema.StringAttribute{
				Description: "Pin this note to a specific repository (owner/repo format). Optional.",
				Optional:    true,
			},
			"folder_id": schema.StringAttribute{
				Description: "The folder ID containing this note.",
				Computed:    true,
			},
			"folder_path": schema.StringAttribute{
				Description: "The folder path of this note.",
				Computed:    true,
			},
			"is_enabled": schema.BoolAttribute{
				Description: "Whether the note is enabled.",
				Computed:    true,
			},
			"macro": schema.StringAttribute{
				Description: "The macro associated with this note.",
				Computed:    true,
			},
			"access_type": schema.StringAttribute{
				Description: "The access type (org or enterprise).",
				Computed:    true,
			},
			"org_id": schema.StringAttribute{
				Description: "The organization ID this note belongs to.",
				Computed:    true,
			},
			"created_at": schema.Int64Attribute{
				Description: "Unix timestamp when the note was created.",
				Computed:    true,
			},
			"updated_at": schema.Int64Attribute{
				Description: "Unix timestamp when the note was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *KnowledgeNoteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*sharedtypes.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *types.ProviderData, got: %T", req.ProviderData),
		)
		return
	}
	r.client = pd.Client
	r.orgID = pd.OrganizationID
}

func (r *KnowledgeNoteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan KnowledgeNoteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.KnowledgeNoteCreateRequest{
		Name:    plan.Name.ValueString(),
		Body:    plan.Body.ValueString(),
		Trigger: plan.Trigger.ValueString(),
	}
	if !plan.PinnedRepo.IsNull() && !plan.PinnedRepo.IsUnknown() {
		v := plan.PinnedRepo.ValueString()
		createReq.PinnedRepo = &v
	}

	note, err := r.client.CreateNote(ctx, r.orgID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating knowledge note", err.Error())
		return
	}

	mapResponseToModel(note, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KnowledgeNoteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state KnowledgeNoteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	note, err := r.client.GetNote(ctx, r.orgID, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading knowledge note", err.Error())
		return
	}

	mapResponseToModel(note, &state)
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

	updateReq := &client.KnowledgeNoteCreateRequest{
		Name:    plan.Name.ValueString(),
		Body:    plan.Body.ValueString(),
		Trigger: plan.Trigger.ValueString(),
	}
	if !plan.PinnedRepo.IsNull() && !plan.PinnedRepo.IsUnknown() {
		v := plan.PinnedRepo.ValueString()
		updateReq.PinnedRepo = &v
	}

	note, err := r.client.UpdateNote(ctx, r.orgID, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating knowledge note", err.Error())
		return
	}

	mapResponseToModel(note, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KnowledgeNoteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state KnowledgeNoteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteNote(ctx, r.orgID, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting knowledge note", err.Error())
	}
}

func (r *KnowledgeNoteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: <org_id>/<note_id>, got: %s", req.ID),
		)
		return
	}

	if parts[0] != r.orgID {
		resp.Diagnostics.AddError(
			"Organization ID mismatch",
			fmt.Sprintf("Import org_id %q does not match provider organization_id %q.", parts[0], r.orgID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_id"), parts[0])...)
}

func mapResponseToModel(note *client.KnowledgeNoteResponse, model *KnowledgeNoteResourceModel) {
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
	if note.FolderID != nil {
		model.FolderID = types.StringValue(*note.FolderID)
	} else {
		model.FolderID = types.StringNull()
	}
	if note.Macro != nil {
		model.Macro = types.StringValue(*note.Macro)
	} else {
		model.Macro = types.StringNull()
	}
	if note.OrgID != nil {
		model.OrgID = types.StringValue(*note.OrgID)
	} else {
		model.OrgID = types.StringNull()
	}
}
