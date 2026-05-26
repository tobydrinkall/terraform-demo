package sessions_poc

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourceSessionWaitForCompletion implements Approach B:
// A managed Terraform resource that creates a session and optionally waits
// for it to reach a terminal state before returning.
//
// When wait_for_completion = false: behaves identically to Approach A.
// When wait_for_completion = true:  polls session status until terminal or timeout.
func ResourceSessionWaitForCompletion() *schema.Resource {
	return &schema.Resource{
		Description: "Creates and manages a Devin session with optional wait-for-completion behavior. " +
			"When `wait_for_completion` is true, Terraform will poll the session status until " +
			"it reaches a terminal state or the timeout is reached.",

		CreateContext: resourceSessionWFCCreate,
		ReadContext:   resourceSessionWFCRead,
		UpdateContext: resourceSessionWFCUpdate,
		DeleteContext: resourceSessionWFCDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(4 * time.Hour),
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
				Description: "Idempotency key for the session creation request.",
			},
			"wait_for_completion": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, terraform apply will block until the session reaches a terminal state.",
			},
			"timeout": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "4h",
				ValidateFunc:     validation.StringIsNotEmpty,
				Description:      "Maximum time to wait for session completion (Go duration format, e.g. '30m', '2h'). Only used when wait_for_completion is true.",
				DiffSuppressFunc: suppressWhenNotWaiting,
			},
			"poll_interval": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "30s",
				ValidateFunc:     validation.StringIsNotEmpty,
				Description:      "How frequently to poll the session status (Go duration format). Only used when wait_for_completion is true.",
				DiffSuppressFunc: suppressWhenNotWaiting,
			},

			// Computed
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
			"created_at": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Unix timestamp when the session was created.",
			},
			"acus_consumed": {
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "ACUs consumed by the session (only populated when wait_for_completion is true).",
			},
			"timed_out": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the wait timed out before the session completed.",
			},
		},
	}
}

func suppressWhenNotWaiting(k, old, new string, d *schema.ResourceData) bool {
	waitForCompletion := d.Get("wait_for_completion").(bool)
	return !waitForCompletion
}

func resourceSessionWFCCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
	d.Set("timed_out", false)

	if session.Title != nil {
		d.Set("title", *session.Title)
	}

	log.Printf("[INFO] Created Devin session %s", session.SessionID)

	// If wait_for_completion is enabled, poll until terminal state
	waitForCompletion := d.Get("wait_for_completion").(bool)
	if waitForCompletion {
		timeoutStr := d.Get("timeout").(string)
		timeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return diag.FromErr(fmt.Errorf("parsing timeout %q: %w", timeoutStr, err))
		}

		pollIntervalStr := d.Get("poll_interval").(string)
		pollInterval, err := time.ParseDuration(pollIntervalStr)
		if err != nil {
			return diag.FromErr(fmt.Errorf("parsing poll_interval %q: %w", pollIntervalStr, err))
		}

		log.Printf("[INFO] Waiting for session %s to complete (timeout: %s, poll: %s)",
			session.SessionID, timeout, pollInterval)

		finalSession, err := client.WaitForCompletion(ctx, session.SessionID, timeout, pollInterval)
		if err != nil {
			if finalSession != nil {
				// Timeout — session is still running, not a hard error
				d.Set("timed_out", true)
				d.Set("status", finalSession.Status)
				d.Set("acus_consumed", finalSession.ACUsConsumed)
				if finalSession.StatusDetail != nil {
					d.Set("status_detail", *finalSession.StatusDetail)
				}
				if finalSession.Title != nil {
					d.Set("title", *finalSession.Title)
				}
				log.Printf("[WARN] %v", err)

				return diag.Diagnostics{
					{
						Severity: diag.Warning,
						Summary:  "Session did not complete within timeout",
						Detail:   fmt.Sprintf("Session %s is still in status %q after %s. It will continue running.", session.SessionID, finalSession.Status, timeout),
					},
				}
			}

			// Real error (network failure, session vanished) — return as error
			return diag.FromErr(fmt.Errorf("waiting for session %s: %w", session.SessionID, err))
		}

		d.Set("status", finalSession.Status)
		d.Set("acus_consumed", finalSession.ACUsConsumed)
		if finalSession.StatusDetail != nil {
			d.Set("status_detail", *finalSession.StatusDetail)
		}
		if finalSession.Title != nil {
			d.Set("title", *finalSession.Title)
		}
		log.Printf("[INFO] Session %s completed with status: %s", session.SessionID, finalSession.Status)
	}

	return nil
}

func resourceSessionWFCRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
	d.Set("acus_consumed", session.ACUsConsumed)

	if session.StatusDetail != nil {
		d.Set("status_detail", *session.StatusDetail)
	} else {
		d.Set("status_detail", "")
	}
	if session.Title != nil {
		d.Set("title", *session.Title)
	} else {
		d.Set("title", "")
	}

	return nil
}

func resourceSessionWFCUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Only wait_for_completion, timeout, and poll_interval can change (non-ForceNew).
	// No API call needed — just re-read.
	return resourceSessionWFCRead(ctx, d, meta)
}

func resourceSessionWFCDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*SessionClient)

	session, err := client.GetSession(ctx, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("reading session before delete: %w", err))
	}

	if session == nil || IsTerminalStatus(session.Status) {
		log.Printf("[INFO] Session %s already terminated, removing from state", d.Id())
		return nil
	}

	_, err = client.TerminateSession(ctx, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("terminating session %s: %w", d.Id(), err))
	}

	// Wait for actual terminal state (not running/finished which is transient after DELETE)
	_, waitErr := client.WaitForTermination(ctx, d.Id(), d.Timeout(schema.TimeoutDelete), 3*time.Second)
	if waitErr != nil {
		log.Printf("[WARN] Session %s may still be shutting down: %v", d.Id(), waitErr)
	}

	log.Printf("[INFO] Session %s terminated", d.Id())
	return nil
}
