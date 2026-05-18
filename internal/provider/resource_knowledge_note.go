package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceKnowledgeNote() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a knowledge note in the Devin platform. Knowledge notes store reusable context that Devin retrieves automatically during sessions when the trigger conditions match.",

		CreateContext: resourceKnowledgeNoteCreate,
		ReadContext:   resourceKnowledgeNoteRead,
		UpdateContext: resourceKnowledgeNoteUpdate,
		DeleteContext: resourceKnowledgeNoteDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique identifier of the knowledge note (e.g., `note-abc123`).",
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(1, 200),
				Description:  "The display name of the knowledge note. Must be between 1 and 200 characters.",
			},
			"content": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The body content of the knowledge note in markdown format. This is the information Devin will retrieve during sessions.",
			},
			"trigger": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A description of when this knowledge note should be retrieved. Defines the scope or conditions under which Devin will surface this note (e.g., \"When working on the payments service\").",
			},
			"scope": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "organization",
				ValidateFunc: validation.StringInSlice([]string{"organization", "user", "repo"}, false),
				Description:  "The visibility scope of the knowledge note. Valid values are `organization` (visible to all org members), `user` (visible only to the creator), or `repo` (scoped to a specific repository). Defaults to `organization`.",
			},
			"repo_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The repository to scope the note to, required when `scope` is `repo`. Format: `owner/repo` (e.g., `myorg/myrepo`).",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ISO 8601 timestamp when the knowledge note was created.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ISO 8601 timestamp when the knowledge note was last updated.",
			},
			"author": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The author of the knowledge note. Either `user` or `system`.",
			},
			"size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The size of the knowledge note content in characters.",
			},
		},
	}
}

func resourceKnowledgeNoteCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId("note-placeholder")
	return nil
}

func resourceKnowledgeNoteRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceKnowledgeNoteUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceKnowledgeNoteDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}
