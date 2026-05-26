package sessions_poc

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// DataSourceSession implements Approach C (part 1):
// A data source to look up a single existing Devin session by ID.
// No creation or management — sessions are created outside Terraform.
func DataSourceSession() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves details of a single Devin session by its ID. " +
			"Use this to reference sessions created outside of Terraform.",

		ReadContext: dataSourceSessionRead,

		Schema: map[string]*schema.Schema{
			"session_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The session ID to look up.",
			},

			// Computed
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The URL to view the session in the Devin web app.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current status of the session.",
			},
			"status_detail": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Detailed status information.",
			},
			"title": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The auto-generated title of the session.",
			},
			"origin": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "How the session was created (api, webapp, slack, etc).",
			},
			"created_at": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Unix timestamp when the session was created.",
			},
			"updated_at": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Unix timestamp when the session was last updated.",
			},
			"is_archived": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the session is archived.",
			},
			"acus_consumed": {
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "ACUs consumed by the session.",
			},
			"user_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The user who owns the session.",
			},
			"tags": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Tags associated with the session.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceSessionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*SessionClient)

	sessionID := d.Get("session_id").(string)

	session, err := client.GetSession(sessionID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("reading session %s: %w", sessionID, err))
	}
	if session == nil {
		return diag.Errorf("session %s not found", sessionID)
	}

	d.SetId(session.SessionID)
	d.Set("url", session.URL)
	d.Set("status", session.Status)
	d.Set("origin", session.Origin)
	d.Set("created_at", session.CreatedAt)
	d.Set("updated_at", session.UpdatedAt)
	d.Set("is_archived", session.IsArchived)
	d.Set("acus_consumed", session.ACUsConsumed)
	d.Set("user_id", session.UserID)
	d.Set("tags", session.Tags)

	if session.StatusDetail != nil {
		d.Set("status_detail", *session.StatusDetail)
	}
	if session.Title != nil {
		d.Set("title", *session.Title)
	}

	return nil
}

// DataSourceSessions implements Approach C (part 2):
// A data source to list/filter existing Devin sessions.
func DataSourceSessions() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves a list of Devin sessions with optional filtering. " +
			"Use this for audit, monitoring, or referencing sessions created outside Terraform.",

		ReadContext: dataSourceSessionsRead,

		Schema: map[string]*schema.Schema{
			"status_filter": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"running", "exit", "new", "claimed"}, false),
				Description:  "Filter sessions by status.",
			},
			"limit": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      20,
				ValidateFunc: validation.IntBetween(1, 100),
				Description:  "Maximum number of sessions to return.",
			},

			// Computed
			"sessions": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of sessions matching the filter criteria.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"session_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"url": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"status": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"status_detail": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"title": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"origin": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"created_at": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"acus_consumed": {
							Type:     schema.TypeFloat,
							Computed: true,
						},
						"is_archived": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceSessionsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*SessionClient)

	limit := d.Get("limit").(int)
	statusFilter := ""
	if v, ok := d.GetOk("status_filter"); ok {
		statusFilter = v.(string)
	}

	list, err := client.ListSessions(limit, statusFilter, "")
	if err != nil {
		return diag.FromErr(fmt.Errorf("listing sessions: %w", err))
	}

	sessions := make([]map[string]interface{}, 0, len(list.Items))
	for _, s := range list.Items {
		session := map[string]interface{}{
			"session_id":    s.SessionID,
			"url":           s.URL,
			"status":        s.Status,
			"origin":        s.Origin,
			"created_at":    s.CreatedAt,
			"acus_consumed": s.ACUsConsumed,
			"is_archived":   s.IsArchived,
		}
		if s.StatusDetail != nil {
			session["status_detail"] = *s.StatusDetail
		} else {
			session["status_detail"] = ""
		}
		if s.Title != nil {
			session["title"] = *s.Title
		} else {
			session["title"] = ""
		}
		sessions = append(sessions, session)
	}

	d.SetId("devin-sessions-list")
	d.Set("sessions", sessions)

	return nil
}
