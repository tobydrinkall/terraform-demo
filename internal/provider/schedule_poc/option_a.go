package schedule_poc

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourceScheduleOptionA returns a schedule resource that validates
// conditional requirements in the Create and Update CRUD functions.
// Both frequency and scheduled_at are marked Optional; the combination
// is checked imperatively at apply time.
func ResourceScheduleOptionA() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Devin schedule (Option A: CRUD-time validation). " +
			"When schedule_type is 'recurring', frequency (cron) is required. " +
			"When schedule_type is 'one_time', scheduled_at (ISO 8601) is required.",

		CreateContext: optionACreate,
		ReadContext:   optionARead,
		UpdateContext: optionAUpdate,
		DeleteContext: optionADelete,

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
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Cron expression for recurring schedules (e.g. `0 9 * * 1-5`). Required when schedule_type is `recurring`.",
			},
			"scheduled_at": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "ISO 8601 datetime for one-time schedules (e.g. `2025-06-01T09:00:00Z`). Required when schedule_type is `one_time`.",
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

// validateScheduleCombination checks that the correct field is set for the
// chosen schedule_type. Returns diagnostics (errors) if the combination is
// invalid. This is called from both Create and Update.
func validateScheduleCombination(d *schema.ResourceData) diag.Diagnostics {
	scheduleType := d.Get("schedule_type").(string)
	frequency := d.Get("frequency").(string)
	scheduledAt := d.Get("scheduled_at").(string)

	var diags diag.Diagnostics

	switch scheduleType {
	case "recurring":
		if frequency == "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Missing required attribute for recurring schedule",
				Detail: fmt.Sprintf(
					"When schedule_type is \"recurring\", the \"frequency\" attribute "+
						"must be set to a valid cron expression. Got schedule_type=%q "+
						"with no frequency value.", scheduleType),
			})
		}
		if scheduledAt != "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Invalid attribute for recurring schedule",
				Detail: fmt.Sprintf(
					"The \"scheduled_at\" attribute must not be set when schedule_type "+
						"is \"recurring\". Got scheduled_at=%q.", scheduledAt),
			})
		}
	case "one_time":
		if scheduledAt == "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Missing required attribute for one-time schedule",
				Detail: fmt.Sprintf(
					"When schedule_type is \"one_time\", the \"scheduled_at\" attribute "+
						"must be set to a valid ISO 8601 datetime. Got schedule_type=%q "+
						"with no scheduled_at value.", scheduleType),
			})
		} else if _, err := time.Parse(time.RFC3339, scheduledAt); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Invalid scheduled_at format",
				Detail: fmt.Sprintf(
					"The \"scheduled_at\" attribute must be a valid ISO 8601 / RFC 3339 "+
						"datetime. Got %q: %s", scheduledAt, err),
			})
		}
		if frequency != "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Invalid attribute for one-time schedule",
				Detail: fmt.Sprintf(
					"The \"frequency\" attribute must not be set when schedule_type "+
						"is \"one_time\". Got frequency=%q.", frequency),
			})
		}
	}

	return diags
}

func optionACreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if diags := validateScheduleCombination(d); diags.HasError() {
		return diags
	}

	// Simulate API call — set a placeholder ID and timestamp.
	d.SetId("sched-a-placeholder")
	d.Set("created_at", time.Now().UTC().Format(time.RFC3339))
	return nil
}

func optionARead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func optionAUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if diags := validateScheduleCombination(d); diags.HasError() {
		return diags
	}
	return nil
}

func optionADelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}
