package sessions_poc

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceSessionFireAndForget implements Approach A:
// A managed Terraform resource that creates a session via the API and returns
// immediately without waiting for the session to complete.
//
// terraform apply  → POST /sessions → returns session_id immediately
// terraform destroy → DELETE /sessions/{id} → terminates the session
func ResourceSessionFireAndForget() *schema.Resource {
	return &schema.Resource{
		Description: "Creates and manages a Devin session. The session is created immediately " +
			"and Terraform does not wait for it to complete (fire-and-forget). " +
			"Destroying this resource terminates the session.",

		CreateContext: resourceSessionFFCreate,
		ReadContext:   resourceSessionFFRead,
		DeleteContext: resourceSessionFFDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"prompt": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The task prompt for the Devin session.",
			},
			"playbook_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Optional playbook ID to use for the session.",
			},
			"idempotency_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Idempotency key for the session creation request. Auto-generated if not specified.",
			},

			// Computed attributes
			"session_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique session identifier.",
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The URL to view the session in the Devin web app.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current status of the session (new, claimed, running, exit).",
			},
			"status_detail": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Detailed status (working, waiting_for_user, finished).",
			},
			"title": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The auto-generated title of the session.",
			},
			"created_at": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Unix timestamp when the session was created.",
			},
		},
	}
}

func resourceSessionFFCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*SessionClient)

	idempotencyKey := d.Get("idempotency_key").(string)
	if idempotencyKey == "" {
		idempotencyKey = uuid.New().String()
	}

	req := CreateSessionRequest{
		Prompt:         d.Get("prompt").(string),
		IdempotencyKey: idempotencyKey,
	}
	if v, ok := d.GetOk("playbook_id"); ok {
		req.PlaybookID = v.(string)
	}

	session, err := client.CreateSession(ctx, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("creating session: %w", err))
	}

	d.SetId(session.SessionID)
	d.Set("session_id", session.SessionID)
	d.Set("url", session.URL)
	d.Set("status", session.Status)
	d.Set("idempotency_key", idempotencyKey)
	d.Set("created_at", session.CreatedAt)

	if session.StatusDetail != nil {
		d.Set("status_detail", *session.StatusDetail)
	}
	if session.Title != nil {
		d.Set("title", *session.Title)
	}

	log.Printf("[INFO] Created Devin session %s (fire-and-forget)", session.SessionID)
	return nil
}

func resourceSessionFFRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*SessionClient)

	session, err := client.GetSession(ctx, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("reading session %s: %w", d.Id(), err))
	}

	if session == nil {
		d.SetId("")
		return nil
	}

	d.Set("session_id", session.SessionID)
	d.Set("url", session.URL)
	d.Set("status", session.Status)
	d.Set("created_at", session.CreatedAt)

	if session.StatusDetail != nil {
		d.Set("status_detail", *session.StatusDetail)
	} else {
		d.Set("status_detail", "")
	}
	if session.Title != nil {
		d.Set("title", *session.Title)
	}

	return nil
}

func resourceSessionFFDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*SessionClient)

	session, err := client.GetSession(ctx, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("reading session before delete: %w", err))
	}

	// If session is already in a terminal state, just remove from state
	if session == nil || IsTerminalStatus(session.Status) {
		log.Printf("[INFO] Session %s already in terminal state, removing from state", d.Id())
		return nil
	}

	// Terminate the active session
	_, err = client.TerminateSession(ctx, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("terminating session %s: %w", d.Id(), err))
	}

	// Wait for actual terminal state, respecting the resource's delete timeout
	_, waitErr := client.WaitForTermination(ctx, d.Id(), d.Timeout(schema.TimeoutDelete), 3*time.Second)
	if waitErr != nil {
		log.Printf("[WARN] Session %s may still be shutting down: %v", d.Id(), waitErr)
	}

	log.Printf("[INFO] Session %s terminated", d.Id())
	return nil
}
