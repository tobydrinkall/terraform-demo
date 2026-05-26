package enterprise_poc

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// dataSourceEnterpriseMembersPOC shows a second enterprise-scope data source to
// demonstrate that the pattern scales cleanly — every enterprise resource uses
// the same RequireEnterprise() gate.
func dataSourceEnterpriseMembersPOC() *schema.Resource {
	return &schema.Resource{
		Description: "Lists members across all organizations in the enterprise. " +
			"**Requires an enterprise-scoped service user API key.**",

		ReadContext: dataSourceEnterpriseMembersRead,

		Schema: map[string]*schema.Schema{
			"org_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter members to a specific organization within the enterprise.",
			},
			"role": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter by role (e.g., `admin`, `member`).",
			},
			"members": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of enterprise members.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"user_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique user identifier.",
						},
						"email": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The member's email address.",
						},
						"role": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The member's role within the enterprise.",
						},
						"org_ids": {
							Type:        schema.TypeList,
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "The list of organization IDs the member belongs to.",
						},
					},
				},
			},
		},
	}
}

func dataSourceEnterpriseMembersRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*DevinClient)

	// Same gate pattern — consistent error messages across all enterprise resources
	if diags := client.RequireEnterprise(); diags != nil {
		return diags
	}

	// Real implementation: GET /v3/enterprise/{enterprise_id}/members
	d.SetId("enterprise-members-" + time.Now().Format("20060102150405"))
	_ = d.Set("members", []interface{}{})

	return nil
}
