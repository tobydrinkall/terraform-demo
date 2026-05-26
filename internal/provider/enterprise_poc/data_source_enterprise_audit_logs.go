package enterprise_poc

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// DataSourceEnterpriseAuditLogs returns a data source for reading enterprise
// audit logs. This demonstrates the key pattern: enterprise resources that
// gracefully error when the API key lacks enterprise permissions.
//
// Usage:
//
//	data "devin_enterprise_audit_logs" "recent" {
//	  start_date = "2024-01-01T00:00:00Z"
//	  end_date   = "2024-01-31T23:59:59Z"
//	  event_type = "session.created"
//	}
func DataSourceEnterpriseAuditLogs() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves audit logs from the Devin enterprise API. " +
			"**Requires an enterprise-scoped service user API key** and the " +
			"`enterprise_id` provider attribute. If the API key does not have " +
			"enterprise permissions, this data source will return an actionable " +
			"error explaining how to configure enterprise access.",

		ReadContext: dataSourceEnterpriseAuditLogsRead,

		Schema: map[string]*schema.Schema{
			"start_date": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsRFC3339Time,
				Description:  "The start of the time range for audit log entries (RFC 3339 format).",
			},
			"end_date": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsRFC3339Time,
				Description:  "The end of the time range for audit log entries (RFC 3339 format).",
			},
			"event_type": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					"session.created",
					"session.completed",
					"session.failed",
					"knowledge.created",
					"knowledge.updated",
					"knowledge.deleted",
					"member.added",
					"member.removed",
					"settings.updated",
				}, false),
				Description: "Filter audit logs by event type. Valid values: " +
					"`session.created`, `session.completed`, `session.failed`, " +
					"`knowledge.created`, `knowledge.updated`, `knowledge.deleted`, " +
					"`member.added`, `member.removed`, `settings.updated`.",
			},
			"org_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter audit logs to a specific organization within the enterprise.",
			},
			"entries": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of audit log entries matching the filter criteria.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the audit log entry.",
						},
						"timestamp": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ISO 8601 timestamp of the event.",
						},
						"event_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of event that occurred.",
						},
						"actor_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the user or service that performed the action.",
						},
						"actor_email": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The email of the user who performed the action.",
						},
						"org_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The organization ID where the event occurred.",
						},
						"resource_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of resource affected (e.g., session, knowledge_note).",
						},
						"resource_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the resource affected.",
						},
						"details": {
							Type:        schema.TypeMap,
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "Additional details about the event.",
						},
					},
				},
			},
			"total_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The total number of audit log entries matching the filter.",
			},
		},
	}
}

func dataSourceEnterpriseAuditLogsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*DevinClient)

	// Gate: check enterprise permissions before making any API call
	if diags := client.RequireEnterprise(); diags != nil {
		return diags
	}

	// In a real implementation, this would call:
	//   GET /v3/enterprise/{enterprise_id}/audit-logs?start=...&end=...&event_type=...
	//
	// The API returns 403 if the key lacks enterprise scope, which we'd translate
	// to the same RequireEnterprise() diagnostic for consistency.

	_ = d.Get("start_date").(string)
	_ = d.Get("end_date").(string)

	// Placeholder: set an ID and empty results
	d.SetId("audit-logs-" + time.Now().Format("20060102150405"))
	_ = d.Set("entries", []interface{}{})
	_ = d.Set("total_count", 0)

	return nil
}
