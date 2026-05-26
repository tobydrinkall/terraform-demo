package schedule_poc

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func testProviderOptionB() *schema.Provider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"devin_schedule_b": ResourceScheduleOptionB(),
		},
	}
}

// TestOptionB_RecurringValid — happy path.
func TestOptionB_RecurringValid(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionB(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_schedule_b" "daily" {
  title         = "Daily standup"
  prompt        = "Summarise open PRs"
  schedule_type = "recurring"
  frequency     = "0 9 * * 1-5"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("devin_schedule_b.daily", "frequency", "0 9 * * 1-5"),
				),
			},
		},
	})
}

// TestOptionB_OneTimeValid — happy path.
func TestOptionB_OneTimeValid(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionB(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_schedule_b" "once" {
  title         = "One-off deploy"
  prompt        = "Deploy v2.1"
  schedule_type = "one_time"
  scheduled_at  = "2025-06-01T09:00:00Z"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("devin_schedule_b.once", "scheduled_at", "2025-06-01T09:00:00Z"),
				),
			},
		},
	})
}

// TestOptionB_ConflictsBothSet — ConflictsWith should error at plan time
// when both frequency and scheduled_at are specified.
func TestOptionB_ConflictsBothSet(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionB(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_schedule_b" "bad" {
  title         = "Both set"
  prompt        = "Will fail"
  schedule_type = "recurring"
  frequency     = "0 9 * * 1-5"
  scheduled_at  = "2025-06-01T09:00:00Z"
}`,
				ExpectError: MustCompileRegex(`Conflicting configuration arguments|"frequency": conflicts with scheduled_at|"scheduled_at": conflicts with frequency`),
			},
		},
	})
}

// TestOptionB_RecurringMissingFrequency — CustomizeDiff catches at plan.
func TestOptionB_RecurringMissingFrequency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionB(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_schedule_b" "bad" {
  title         = "Missing freq"
  prompt        = "Will fail"
  schedule_type = "recurring"
}`,
				ExpectError: MustCompileRegex(`"frequency" is required when schedule_type is "recurring"`),
			},
		},
	})
}

// TestOptionB_OneTimeMissingScheduledAt — CustomizeDiff catches at plan.
func TestOptionB_OneTimeMissingScheduledAt(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionB(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_schedule_b" "bad" {
  title         = "Missing scheduled_at"
  prompt        = "Will fail"
  schedule_type = "one_time"
}`,
				ExpectError: MustCompileRegex(`"scheduled_at" is required when schedule_type is "one_time"`),
			},
		},
	})
}

// TestOptionB_InvalidCron — ValidateFunc catches bad cron at plan time.
func TestOptionB_InvalidCron(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionB(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_schedule_b" "bad" {
  title         = "Bad cron"
  prompt        = "Will fail"
  schedule_type = "recurring"
  frequency     = "not-a-cron"
}`,
				ExpectError: MustCompileRegex(`must be a valid 5-field cron expression`),
			},
		},
	})
}

// TestOptionB_InvalidDatetime — ValidateFunc catches bad datetime at plan.
func TestOptionB_InvalidDatetime(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionB(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_schedule_b" "bad" {
  title         = "Bad datetime"
  prompt        = "Will fail"
  schedule_type = "one_time"
  scheduled_at  = "next tuesday"
}`,
				ExpectError: MustCompileRegex(`must be a valid ISO 8601`),
			},
		},
	})
}
