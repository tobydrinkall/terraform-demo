package option_b

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &secretResource{}

type secretResource struct{}

type secretResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Value     types.String `tfsdk:"value"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func NewSecretResource() resource.Resource {
	return &secretResource{}
}

func (r *secretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *secretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a secret in the Devin platform. Uses WriteOnly attribute — the value is NEVER stored in Terraform state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the secret.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret.",
			},
			"value": schema.StringAttribute{
				Required:    true,
				WriteOnly:   true,
				Description: "The secret value. WriteOnly — never stored in state.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The ISO 8601 timestamp when the secret was created.",
			},
		},
	}
}

func (r *secretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan secretResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Mock: POST /secrets with name+value
	// plan.Value contains the plaintext — it's available during Create
	// but will NOT be stored in state due to WriteOnly.
	plan.ID = types.StringValue(fmt.Sprintf("secret-%s", plan.Name.ValueString()))
	plan.CreatedAt = types.StringValue("2025-01-15T10:30:00Z")
	// Value will be automatically excluded from state by the framework

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *secretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state secretResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Mock: GET /secrets/{id} — API never returns value
	// Value is WriteOnly, so it's Null in state — nothing to refresh
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *secretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan secretResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Mock: PUT /secrets/{id} with new name/value
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *secretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Mock: DELETE /secrets/{id}
	// State is automatically removed
}
