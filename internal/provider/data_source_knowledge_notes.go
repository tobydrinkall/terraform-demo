package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceKnowledgeNotes() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves a list of knowledge notes from the Devin platform. Use this data source to look up existing knowledge notes by scope, name pattern, or repository.",

		ReadContext: dataSourceKnowledgeNotesRead,

		Schema: map[string]*schema.Schema{
			"scope": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "organization",
				ValidateFunc: validation.StringInSlice([]string{"organization", "user", "repo"}, false),
				Description:  "Filter notes by visibility scope. Valid values are `organization`, `user`, or `repo`. Defaults to `organization`.",
			},
			"name_filter": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A substring filter applied to note names. Only notes whose name contains this string (case-insensitive) will be returned.",
			},
			"repo_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter notes by repository. Only applicable when `scope` is `repo`. Format: `owner/repo`.",
			},
			"notes": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of knowledge notes matching the filter criteria.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the knowledge note.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The display name of the knowledge note.",
						},
						"content": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The body content of the knowledge note.",
						},
						"trigger": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The trigger description for the knowledge note.",
						},
						"scope": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The visibility scope of the knowledge note.",
						},
						"author": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The author of the knowledge note.",
						},
						"created_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ISO 8601 timestamp when the note was created.",
						},
						"updated_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ISO 8601 timestamp when the note was last updated.",
						},
						"size": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The size of the note content in characters.",
						},
					},
				},
			},
		},
	}
}

func dataSourceKnowledgeNotesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId("knowledge-notes-list")
	return nil
}
