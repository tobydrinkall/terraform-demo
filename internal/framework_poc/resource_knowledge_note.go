package framework_poc

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &KnowledgeNoteResource{}
	_ resource.ResourceWithConfigure   = &KnowledgeNoteResource{}
	_ resource.ResourceWithImportState = &KnowledgeNoteResource{}
)

type KnowledgeNoteResource struct {
	client *apiClient
}

type KnowledgeNoteResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Content   types.String `tfsdk:"content"`
	Trigger   types.String `tfsdk:"trigger"`
	Scope     types.String `tfsdk:"scope"`
	RepoName  types.String `tfsdk:"repo_name"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	Author    types.String `tfsdk:"author"`
	Size      types.Int64  `tfsdk:"size"`
}

func NewKnowledgeNoteResource() resource.Resource {
	return &KnowledgeNoteResource{}
}

func (r *KnowledgeNoteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_knowledge_note"
}

func (r *KnowledgeNoteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a knowledge note in the Devin platform. Knowledge notes store reusable context that Devin retrieves automatically during sessions when the trigger conditions match.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the knowledge note (e.g., `note-abc123`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The display name of the knowledge note. Must be between 1 and 200 characters.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"content": schema.StringAttribute{
				Required:    true,
				Description: "The body content of the knowledge note in markdown format. This is the information Devin will retrieve during sessions.",
			},
			"trigger": schema.StringAttribute{
				Required:    true,
				Description: "A description of when this knowledge note should be retrieved. Defines the scope or conditions under which Devin will surface this note.",
			},
			"scope": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("organization"),
				Description: "The visibility scope of the knowledge note. Valid values are `organization` (visible to all org members), `user` (visible only to the creator), or `repo` (scoped to a specific repository). Defaults to `organization`.",
				Validators: []validator.String{
					stringvalidator.OneOf("organization", "user", "repo"),
				},
			},
			"repo_name": schema.StringAttribute{
				Optional:    true,
				Description: "The repository to scope the note to, required when `scope` is `repo`. Format: `owner/repo` (e.g., `myorg/myrepo`).",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The ISO 8601 timestamp when the knowledge note was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The ISO 8601 timestamp when the knowledge note was last updated.",
			},
			"author": schema.StringAttribute{
				Computed:    true,
				Description: "The author of the knowledge note. Either `user` or `system`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"size": schema.Int64Attribute{
				Computed:    true,
				Description: "The size of the knowledge note content in characters.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *KnowledgeNoteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*apiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *apiClient, got unexpected type.",
		)
		return
	}
	r.client = client
}

func (r *KnowledgeNoteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan KnowledgeNoteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Stub: set placeholder values
	plan.ID = types.StringValue("note-placeholder")
	plan.CreatedAt = types.StringValue("2025-01-01T00:00:00Z")
	plan.UpdatedAt = types.StringValue("2025-01-01T00:00:00Z")
	plan.Author = types.StringValue("user")
	plan.Size = types.Int64Value(int64(len(plan.Content.ValueString())))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KnowledgeNoteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state KnowledgeNoteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Stub: no-op, state unchanged
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *KnowledgeNoteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan KnowledgeNoteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Stub: update timestamp and size
	plan.UpdatedAt = types.StringValue("2025-01-01T00:00:01Z")
	plan.Size = types.Int64Value(int64(len(plan.Content.ValueString())))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KnowledgeNoteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Stub: no-op, state is automatically removed
}

func (r *KnowledgeNoteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
