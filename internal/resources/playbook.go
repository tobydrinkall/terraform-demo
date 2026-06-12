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
	_ resource.Resource                = &PlaybookResource{}
	_ resource.ResourceWithImportState = &PlaybookResource{}
)

type PlaybookResource struct {
	client *client.Client
	orgID  string
}

type PlaybookResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Title     types.String `tfsdk:"title"`
	Content   types.String `tfsdk:"content"`
	Macro     types.String `tfsdk:"macro"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func NewPlaybookResource() resource.Resource {
	return &PlaybookResource{}
}

func (r *PlaybookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook"
}

func (r *PlaybookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Devin playbook. Playbooks define reusable instruction sets for Devin sessions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the playbook.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Description: "The title of the playbook.",
				Required:    true,
			},
			"content": schema.StringAttribute{
				Description: "The markdown content of the playbook.",
				Required:    true,
			},
			"macro": schema.StringAttribute{
				Description: "Automation macro identifier. Must start with '!' followed by word characters (e.g. '!my_macro').",
				Optional:    true,
			},
			"created_at": schema.Int64Attribute{
				Description: "Unix timestamp when the playbook was created.",
				Computed:    true,
			},
			"updated_at": schema.Int64Attribute{
				Description: "Unix timestamp when the playbook was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *PlaybookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PlaybookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PlaybookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.PlaybookCreateRequest{
		Title:   plan.Title.ValueString(),
		Content: plan.Content.ValueString(),
	}
	if !plan.Macro.IsNull() && !plan.Macro.IsUnknown() {
		v := plan.Macro.ValueString()
		createReq.Macro = &v
	}

	playbook, err := r.client.CreatePlaybook(ctx, r.orgID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating playbook", err.Error())
		return
	}

	mapPlaybookToModel(playbook, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PlaybookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PlaybookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	playbook, err := r.client.GetPlaybook(ctx, r.orgID, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading playbook", err.Error())
		return
	}

	mapPlaybookToModel(playbook, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PlaybookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PlaybookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state PlaybookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &client.PlaybookCreateRequest{
		Title:   plan.Title.ValueString(),
		Content: plan.Content.ValueString(),
	}
	if !plan.Macro.IsNull() && !plan.Macro.IsUnknown() {
		v := plan.Macro.ValueString()
		updateReq.Macro = &v
	}

	playbook, err := r.client.UpdatePlaybook(ctx, r.orgID, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating playbook", err.Error())
		return
	}

	mapPlaybookToModel(playbook, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PlaybookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PlaybookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePlaybook(ctx, r.orgID, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting playbook", err.Error())
	}
}

func (r *PlaybookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: <org_id>/<playbook_id>, got: %s", req.ID),
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
}

func mapPlaybookToModel(pb *client.PlaybookResponse, model *PlaybookResourceModel) {
	model.ID = types.StringValue(pb.PlaybookID)
	model.Title = types.StringValue(pb.Title)
	model.Content = types.StringValue(pb.Content)
	model.CreatedAt = types.Int64Value(pb.CreatedAt)
	model.UpdatedAt = types.Int64Value(pb.UpdatedAt)

	if pb.Macro != nil {
		model.Macro = types.StringValue(*pb.Macro)
	} else {
		model.Macro = types.StringNull()
	}
}
