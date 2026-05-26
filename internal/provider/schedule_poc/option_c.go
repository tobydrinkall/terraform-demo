package schedule_poc

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ---------------------------------------------------------------------------
// Option C: Two separate resources — devin_recurring_schedule and
// devin_one_time_schedule. Each resource has exactly the fields it needs,
// so there is no conditional requirement to enforce at all.
// ---------------------------------------------------------------------------

// ResourceRecurringSchedule requires `frequency` (cron) and forbids
// `scheduled_at`. The schedule_type is implicit: always "recurring".
func ResourceRecurringSchedule() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a recurring Devin schedule (Option C). " +
			"Requires a cron frequency; schedule_type is implicitly 'recurring'.",

		CreateContext: recurringCreate,
		ReadContext:   recurringRead,
		UpdateContext: recurringUpdate,
		DeleteContext: recurringDelete,

		Schema: sharedFields(map[string]*schema.Schema{
			"frequency": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringMatch(
					regexp.MustCompile(`^(\S+\s+){4}\S+$`),
					"must be a valid 5-field cron expression (e.g. '0 9 * * 1-5')",
				),
				Description: "Cron expression defining the recurrence (e.g. `0 9 * * 1-5`).",
			},
		}),
	}
}

// ResourceOneTimeSchedule requires `scheduled_at` (ISO 8601) and forbids
// `frequency`. The schedule_type is implicit: always "one_time".
func ResourceOneTimeSchedule() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a one-time Devin schedule (Option C). " +
			"Requires a scheduled_at datetime; schedule_type is implicitly 'one_time'.",

		CreateContext: oneTimeCreate,
		ReadContext:   oneTimeRead,
		UpdateContext: oneTimeUpdate,
		DeleteContext: oneTimeDelete,

		Schema: sharedFields(map[string]*schema.Schema{
			"scheduled_at": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					if _, err := time.Parse(time.RFC3339, v); err != nil {
						errs = append(errs, fmt.Errorf(
							"%q must be a valid ISO 8601 / RFC 3339 datetime, got %q: %s",
							key, v, err,
						))
					}
					return
				},
				Description: "ISO 8601 datetime for the one-time execution (e.g. `2025-06-01T09:00:00Z`).",
			},
		}),
	}
}

// sharedFields returns the base schema merged with type-specific fields.
func sharedFields(extra map[string]*schema.Schema) map[string]*schema.Schema {
	base := map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The unique identifier of the schedule.",
		},
		"title": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringLenBetween(1, 200),
			Description:  "The display name of the schedule.",
		},
		"prompt": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The prompt that Devin will execute on each trigger.",
		},
		"is_active": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Whether the schedule is active.",
		},
		"created_at": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "ISO 8601 creation timestamp.",
		},
	}
	for k, v := range extra {
		base[k] = v
	}
	return base
}

// ---------- Recurring CRUD ----------

func recurringCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId(fmt.Sprintf("sched-rec-%d", time.Now().UnixMilli()))
	d.Set("created_at", time.Now().UTC().Format(time.RFC3339))
	return nil
}

func recurringRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func recurringUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func recurringDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}

// ---------- One-Time CRUD ----------

func oneTimeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId(fmt.Sprintf("sched-ot-%d", time.Now().UnixMilli()))
	d.Set("created_at", time.Now().UTC().Format(time.RFC3339))
	return nil
}

func oneTimeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func oneTimeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func oneTimeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}
