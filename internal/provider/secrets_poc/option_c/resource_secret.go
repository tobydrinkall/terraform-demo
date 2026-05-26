package option_c

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceSecret() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a secret in the Devin platform. Only a SHA-256 hash of the value is stored in state — the plaintext value is never persisted.",

		CreateContext: resourceSecretCreate,
		ReadContext:   resourceSecretRead,
		UpdateContext: resourceSecretUpdate,
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
				Description: "The secret value. Not stored in state — only its hash is persisted.",
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
					// oldValue from state is the hash, newValue is the plaintext from config.
					// Compare hash of new plaintext against stored hash.
					newHash := sha256Hex(newValue)
					return oldValue == newHash
				},
				StateFunc: func(val interface{}) string {
					return sha256Hex(val.(string))
				},
			},
			"value_sha256": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The SHA-256 hash of the secret value, stored in state for drift detection.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ISO 8601 timestamp when the secret was created.",
			},
		},
	}
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

func resourceSecretCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	value := d.Get("value").(string)

	// Mock: POST /secrets with name+value
	d.SetId(fmt.Sprintf("secret-%s", name))
	d.Set("value_sha256", sha256Hex(value))
	d.Set("created_at", "2025-01-15T10:30:00Z")

	return nil
}

func resourceSecretRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Mock: GET /secrets/{id} — API returns name, NOT value.
	// We keep the hash in state for comparison.
	return nil
}

func resourceSecretUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChange("value") {
		value := d.Get("value").(string)
		// Mock: PUT /secrets/{id} with new value
		d.Set("value_sha256", sha256Hex(value))
	}
	return nil
}

func resourceSecretDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Mock: DELETE /secrets/{id}
	d.SetId("")
	return nil
}
