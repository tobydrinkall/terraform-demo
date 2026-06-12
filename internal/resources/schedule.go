package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tobydrinkall/terraform-provider-devin/internal/client"
	sharedtypes "github.com/tobydrinkall/terraform-provider-devin/internal/types"
)

var (
	_ resource.Resource                = &ScheduleResource{}
	_ resource.ResourceWithImportState = &ScheduleResource{}
)

type ScheduleResource struct {
	client *client.Client
	orgID  string
}

type ScheduleResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Prompt         types.String `tfsdk:"prompt"`
	PlaybookID     types.String `tfsdk:"playbook_id"`
	Frequency      types.String `tfsdk:"frequency"`
	ScheduleType   types.String `tfsdk:"schedule_type"`
	ScheduledAt    types.String `tfsdk:"scheduled_at"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	NotifyOn       types.String `tfsdk:"notify_on"`
	Agent          types.String `tfsdk:"agent"`
	BypassApproval types.Bool   `tfsdk:"bypass_approval"`
	CreatedAt      types.Int64  `tfsdk:"created_at"`
	UpdatedAt      types.Int64  `tfsdk:"updated_at"`
}

func NewScheduleResource() resource.Resource {
	return &ScheduleResource{}
}

func (r *ScheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schedule"
}

func (r *ScheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Devin schedule. Schedules trigger Devin sessions on a recurring or one-time basis.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the schedule.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name for the schedule.",
				Required:    true,
			},
			"prompt": schema.StringAttribute{
				Description: "The prompt Devin will run on each trigger.",
				Required:    true,
			},
			"playbook_id": schema.StringAttribute{
				Description: "Optional playbook to attach to the scheduled session.",
				Optional:    true,
			},
			"frequency": schema.StringAttribute{
				Description: "Cron expression for recurring schedules (e.g. '0 9 * * 1-5').",
				Optional:    true,
			},
			"schedule_type": schema.StringAttribute{
				Description: "Type of schedule: 'recurring' or 'one_time'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("recurring"),
			},
			"scheduled_at": schema.StringAttribute{
				Description: "ISO 8601 datetime for one-time schedules.",
				Optional:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the schedule is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"notify_on": schema.StringAttribute{
				Description: "When to send notifications: 'always', 'failure', or 'never'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("always"),
			},
			"agent": schema.StringAttribute{
				Description: "Which agent to run: 'devin', 'data_analyst', or 'advanced'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("devin"),
			},
			"bypass_approval": schema.BoolAttribute{
				Description: "Skip MCP tool permission checks for created sessions.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"created_at": schema.Int64Attribute{
				Description: "Unix timestamp when the schedule was created.",
				Computed:    true,
			},
			"updated_at": schema.Int64Attribute{
				Description: "Unix timestamp when the schedule was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *ScheduleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.ScheduleCreateRequest{
		Name:         plan.Name.ValueString(),
		Prompt:       plan.Prompt.ValueString(),
		ScheduleType: plan.ScheduleType.ValueString(),
	}
	if !plan.PlaybookID.IsNull() && !plan.PlaybookID.IsUnknown() {
		v := plan.PlaybookID.ValueString()
		createReq.PlaybookID = &v
	}
	if !plan.Frequency.IsNull() && !plan.Frequency.IsUnknown() {
		v := plan.Frequency.ValueString()
		createReq.Frequency = &v
	}
	if !plan.ScheduledAt.IsNull() && !plan.ScheduledAt.IsUnknown() {
		v := plan.ScheduledAt.ValueString()
		createReq.ScheduledAt = &v
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		createReq.Enabled = &v
	}
	if !plan.NotifyOn.IsNull() && !plan.NotifyOn.IsUnknown() {
		v := plan.NotifyOn.ValueString()
		createReq.NotifyOn = &v
	}
	if !plan.Agent.IsNull() && !plan.Agent.IsUnknown() {
		v := plan.Agent.ValueString()
		createReq.Agent = &v
	}
	if !plan.BypassApproval.IsNull() && !plan.BypassApproval.IsUnknown() {
		v := plan.BypassApproval.ValueBool()
		createReq.BypassApproval = &v
	}

	schedule, err := r.client.CreateSchedule(ctx, r.orgID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating schedule", err.Error())
		return
	}

	mapScheduleToModel(schedule, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schedule, err := r.client.GetSchedule(ctx, r.orgID, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading schedule", err.Error())
		return
	}

	mapScheduleToModel(schedule, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &client.ScheduleUpdateRequest{}
	name := plan.Name.ValueString()
	updateReq.Name = &name
	prompt := plan.Prompt.ValueString()
	updateReq.Prompt = &prompt

	if !plan.PlaybookID.IsNull() && !plan.PlaybookID.IsUnknown() {
		v := plan.PlaybookID.ValueString()
		updateReq.PlaybookID = &v
	}
	if !plan.Frequency.IsNull() && !plan.Frequency.IsUnknown() {
		v := plan.Frequency.ValueString()
		updateReq.Frequency = &v
	}
	if !plan.ScheduledAt.IsNull() && !plan.ScheduledAt.IsUnknown() {
		v := plan.ScheduledAt.ValueString()
		updateReq.ScheduledAt = &v
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		updateReq.Enabled = &v
	}
	if !plan.NotifyOn.IsNull() && !plan.NotifyOn.IsUnknown() {
		v := plan.NotifyOn.ValueString()
		updateReq.NotifyOn = &v
	}
	if !plan.Agent.IsNull() && !plan.Agent.IsUnknown() {
		v := plan.Agent.ValueString()
		updateReq.Agent = &v
	}
	if !plan.BypassApproval.IsNull() && !plan.BypassApproval.IsUnknown() {
		v := plan.BypassApproval.ValueBool()
		updateReq.BypassApproval = &v
	}

	schedule, err := r.client.UpdateSchedule(ctx, r.orgID, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating schedule", err.Error())
		return
	}

	mapScheduleToModel(schedule, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSchedule(ctx, r.orgID, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting schedule", err.Error())
	}
}

func (r *ScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: <org_id>/<schedule_id>, got: %s", req.ID),
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

func mapScheduleToModel(s *client.ScheduleResponse, model *ScheduleResourceModel) {
	model.ID = types.StringValue(s.ScheduleID)
	model.Name = types.StringValue(s.Name)
	model.Prompt = types.StringValue(s.Prompt)
	model.ScheduleType = types.StringValue(s.ScheduleType)
	model.Enabled = types.BoolValue(s.Enabled)
	model.NotifyOn = types.StringValue(s.NotifyOn)
	model.Agent = types.StringValue(s.Agent)
	model.BypassApproval = types.BoolValue(s.BypassApproval)
	model.CreatedAt = types.Int64Value(s.CreatedAt)
	model.UpdatedAt = types.Int64Value(s.UpdatedAt)

	if s.PlaybookID != nil {
		model.PlaybookID = types.StringValue(*s.PlaybookID)
	} else {
		model.PlaybookID = types.StringNull()
	}
	if s.Frequency != nil {
		model.Frequency = types.StringValue(*s.Frequency)
	} else {
		model.Frequency = types.StringNull()
	}
	if s.ScheduledAt != nil {
		model.ScheduledAt = types.StringValue(*s.ScheduledAt)
	} else {
		model.ScheduledAt = types.StringNull()
	}
}
