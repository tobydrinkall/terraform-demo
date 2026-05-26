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

// ResourceScheduleOptionB returns a schedule resource that uses SDK-level
// validators: ConflictsWith to prevent setting both frequency and
// scheduled_at, and custom ValidateFunc / ValidateDiagFunc to enforce
// format rules. A CustomizeDiff enforces the conditional requirement
// (the right field must be present for the chosen schedule_type) so the
// error appears at **plan time**, not apply time.
func ResourceScheduleOptionB() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Devin schedule (Option B: SDK validators). " +
			"Uses ConflictsWith and CustomizeDiff for plan-time validation.",

		CreateContext: optionBCreate,
		ReadContext:   optionBRead,
		UpdateContext: optionBUpdate,
		DeleteContext: optionBDelete,

		CustomizeDiff: optionBCustomizeDiff,

		Schema: map[string]*schema.Schema{
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
			"schedule_type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"recurring", "one_time"}, false),
				Description:  "The type of schedule. Valid values: `recurring`, `one_time`.",
			},
			"frequency": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringMatch(
					regexp.MustCompile(`^(\S+\s+){4}\S+$`),
					"must be a valid 5-field cron expression (e.g. '0 9 * * 1-5')",
				),
				ConflictsWith: []string{"scheduled_at"},
				Description:   "Cron expression for recurring schedules. Conflicts with `scheduled_at`.",
			},
			"scheduled_at": {
				Type:     schema.TypeString,
				Optional: true,
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
				ConflictsWith: []string{"frequency"},
				Description:   "ISO 8601 datetime for one-time schedules. Conflicts with `frequency`.",
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
		},
	}
}

// optionBCustomizeDiff runs at plan time and ensures the conditional
// requirement: recurring → frequency required, one_time → scheduled_at
// required. This gives users immediate feedback during `terraform plan`.
func optionBCustomizeDiff(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	scheduleType := diff.Get("schedule_type").(string)

	switch scheduleType {
	case "recurring":
		freq, ok := diff.GetOk("frequency")
		if !ok || freq.(string) == "" {
			return fmt.Errorf(
				"\"frequency\" is required when schedule_type is \"recurring\"")
		}
	case "one_time":
		sa, ok := diff.GetOk("scheduled_at")
		if !ok || sa.(string) == "" {
			return fmt.Errorf(
				"\"scheduled_at\" is required when schedule_type is \"one_time\"")
		}
	}
	return nil
}

func optionBCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId("sched-b-placeholder")
	d.Set("created_at", time.Now().UTC().Format(time.RFC3339))
	return nil
}

func optionBRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func optionBUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func optionBDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}
