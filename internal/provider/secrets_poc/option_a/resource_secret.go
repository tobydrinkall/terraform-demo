package option_a

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceSecret() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a secret in the Devin platform. The value is stored in Terraform state (encrypted at rest). Any change to the value triggers destroy+create (ForceNew).",

		CreateContext: resourceSecretCreate,
		ReadContext:   resourceSecretRead,
		DeleteContext: resourceSecretDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the secret.",
			},
			"value": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				ForceNew:    true,
				Description: "The secret value. Stored in state as sensitive. Changes trigger replacement.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ISO 8601 timestamp when the secret was created.",
			},
		},
	}
}

func resourceSecretCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Get("name").(string)

	// Mock: In real provider, POST /secrets with name+value
	d.SetId(fmt.Sprintf("secret-%s", name))
	d.Set("created_at", "2025-01-15T10:30:00Z")

	return nil
}

func resourceSecretRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Mock: In real provider, GET /secrets/{id} — API returns name but NOT value.
	// The value remains in state from the Create call. This is the SDK v2 default
	// behavior: if we don't call d.Set("value", ...) here, the state retains the
	// value from creation.
	return nil
}

func resourceSecretDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Mock: In real provider, DELETE /secrets/{id}
	d.SetId("")
	return nil
}
