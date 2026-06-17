package sdkv2

import (
	"context"
	"errors"
	"fmt"

	"github.com/COG-GTM/terraform-provider-devin/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceKnowledgeNote() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Devin knowledge note.",
		CreateContext: resourceKnowledgeNoteCreate,
		ReadContext:   resourceKnowledgeNoteRead,
		UpdateContext: resourceKnowledgeNoteUpdate,
		DeleteContext: resourceKnowledgeNoteDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		// CustomizeDiff validates that name is non-empty at plan time.
		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
			if v := diff.Get("name"); v.(string) == "" {
				return fmt.Errorf("name must not be empty")
			}
			return nil
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique identifier of the knowledge note.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name/title of the knowledge note.",
			},
			"trigger": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "When this knowledge should be retrieved (scope description).",
			},
			"body": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The content of the knowledge note.",
			},
			"folder_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Optional folder ID to organize the note.",
			},
			"folder_path": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The computed folder path.",
			},
			"is_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the note is enabled.",
			},
			"macro": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The macro identifier for the note.",
			},
			"org_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The organization ID that owns this note.",
			},
			"pinned_repo": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Optional repository to pin this note to.",
			},
			"access_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The access type for the note.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the note was created.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the note was last updated.",
			},
		},
	}
}

func resourceKnowledgeNoteCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.Client)

	req := client.KnowledgeNoteRequest{
		Name:    d.Get("name").(string),
		Trigger: d.Get("trigger").(string),
		Body:    d.Get("body").(string),
	}

	if v, ok := d.GetOk("folder_id"); ok {
		s := v.(string)
		req.FolderID = &s
	}
	// Always read is_enabled directly — GetOk cannot distinguish false from unset for booleans.
	// Since is_enabled has Default: true, it is always present in the config.
	isEnabled := d.Get("is_enabled").(bool)
	req.IsEnabled = &isEnabled
	if v, ok := d.GetOk("pinned_repo"); ok {
		s := v.(string)
		req.PinnedRepo = &s
	}

	note, err := c.CreateKnowledgeNote(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(note.NoteID)
	return setNoteState(d, note)
}

func resourceKnowledgeNoteRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.Client)

	note, err := c.GetKnowledgeNote(ctx, d.Id())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return setNoteState(d, note)
}

func resourceKnowledgeNoteUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.Client)

	req := client.KnowledgeNoteRequest{
		Name:    d.Get("name").(string),
		Trigger: d.Get("trigger").(string),
		Body:    d.Get("body").(string),
	}

	if v, ok := d.GetOk("folder_id"); ok {
		s := v.(string)
		req.FolderID = &s
	}
	isEnabled := d.Get("is_enabled").(bool)
	req.IsEnabled = &isEnabled
	if v, ok := d.GetOk("pinned_repo"); ok {
		s := v.(string)
		req.PinnedRepo = &s
	}

	note, err := c.UpdateKnowledgeNote(ctx, d.Id(), req)
	if err != nil {
		return diag.FromErr(err)
	}

	return setNoteState(d, note)
}

func resourceKnowledgeNoteDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.Client)

	err := c.DeleteKnowledgeNote(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func setNoteState(d *schema.ResourceData, note *client.KnowledgeNote) diag.Diagnostics {
	var diags diag.Diagnostics

	pairs := map[string]interface{}{
		"name":        note.Name,
		"trigger":     note.Trigger,
		"body":        note.Body,
		"folder_id":   note.FolderID,
		"folder_path": note.FolderPath,
		"is_enabled":  note.IsEnabled,
		"macro":       note.Macro,
		"org_id":      note.OrgID,
		"pinned_repo": note.PinnedRepo,
		"access_type": note.AccessType,
		"created_at":  note.CreatedAt,
		"updated_at":  note.UpdatedAt,
	}

	for key, val := range pairs {
		if err := d.Set(key, val); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Error setting " + key,
				Detail:   err.Error(),
			})
		}
	}
	return diags
}
