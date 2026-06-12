package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tobydrinkall/terraform-provider-devin/internal/client"
	sharedtypes "github.com/tobydrinkall/terraform-provider-devin/internal/types"
)

var (
	_ resource.Resource                = &SecretResource{}
	_ resource.ResourceWithImportState = &SecretResource{}
)

type SecretResource struct {
	client *client.Client
	orgID  string
}

type SecretResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Key         types.String `tfsdk:"key"`
	Value       types.String `tfsdk:"value"`
	SecretType  types.String `tfsdk:"type"`
	IsSensitive types.Bool   `tfsdk:"is_sensitive"`
	Note        types.String `tfsdk:"note"`
	AccessType  types.String `tfsdk:"access_type"`
	CreatedBy   types.String `tfsdk:"created_by"`
	CreatedAt   types.Int64  `tfsdk:"created_at"`
	UpdatedAt   types.Int64  `tfsdk:"updated_at"`
}

func NewSecretResource() resource.Resource {
	return &SecretResource{}
}

func (r *SecretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *SecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Devin secret. Secrets store sensitive values (API keys, tokens, TOTP seeds) used by Devin sessions. Note: the secret value is write-only and cannot be read back from the API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the secret (secret_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				Description: "The key/name of the secret (e.g. 'GITHUB_TOKEN').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Description: "The secret value. Write-only — cannot be read back from the API. Changing this forces recreation.",
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The type of secret: 'key-value', 'cookie', or 'totp'.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"is_sensitive": schema.BoolAttribute{
				Description: "Whether the secret value should be masked in the UI.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"note": schema.StringAttribute{
				Description: "A human-readable note describing what this secret is for.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_type": schema.StringAttribute{
				Description: "The access type (org or personal).",
				Computed:    true,
			},
			"created_by": schema.StringAttribute{
				Description: "Who created this secret.",
				Computed:    true,
			},
			"created_at": schema.Int64Attribute{
				Description: "Unix timestamp when the secret was created.",
				Computed:    true,
			},
			"updated_at": schema.Int64Attribute{
				Description: "Unix timestamp when the secret was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *SecretResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.SecretCreateRequest{
		Key:         plan.Key.ValueString(),
		Value:       plan.Value.ValueString(),
		SecretType:  plan.SecretType.ValueString(),
		IsSensitive: plan.IsSensitive.ValueBool(),
	}
	if !plan.Note.IsNull() && !plan.Note.IsUnknown() {
		v := plan.Note.ValueString()
		createReq.Note = &v
	}

	secret, err := r.client.CreateSecret(ctx, r.orgID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating secret", err.Error())
		return
	}

	mapSecretToModel(secret, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, err := r.client.FindSecretByID(ctx, r.orgID, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading secret", err.Error())
		return
	}

	mapSecretToModel(secret, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SecretResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"Secrets cannot be updated in-place. Changes to key, value, or type force recreation.",
	)
}

func (r *SecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSecret(ctx, r.orgID, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting secret", err.Error())
	}
}

func (r *SecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: <org_id>/<secret_id>, got: %s", req.ID),
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

func mapSecretToModel(s *client.SecretResponse, model *SecretResourceModel) {
	model.ID = types.StringValue(s.SecretID)
	model.SecretType = types.StringValue(s.SecretType)
	model.IsSensitive = types.BoolValue(s.IsSensitive)
	model.AccessType = types.StringValue(s.AccessType)
	model.CreatedBy = types.StringValue(s.CreatedBy)
	model.CreatedAt = types.Int64Value(s.CreatedAt)

	if s.Key != nil {
		model.Key = types.StringValue(*s.Key)
	} else {
		model.Key = types.StringNull()
	}
	if s.Note != nil {
		model.Note = types.StringValue(*s.Note)
	} else {
		model.Note = types.StringNull()
	}
	if s.UpdatedAt != nil {
		model.UpdatedAt = types.Int64Value(*s.UpdatedAt)
	} else {
		model.UpdatedAt = types.Int64Null()
	}
}
