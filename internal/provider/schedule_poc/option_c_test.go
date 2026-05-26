package schedule_poc

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func testProviderOptionC() *schema.Provider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"devin_recurring_schedule": ResourceRecurringSchedule(),
			"devin_one_time_schedule":  ResourceOneTimeSchedule(),
		},
	}
}

// TestOptionC_RecurringValid — happy path.
func TestOptionC_RecurringValid(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionC(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_recurring_schedule" "daily" {
  title     = "Daily standup"
  prompt    = "Summarise open PRs"
  frequency = "0 9 * * 1-5"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("devin_recurring_schedule.daily", "frequency", "0 9 * * 1-5"),
				),
			},
		},
	})
}

// TestOptionC_OneTimeValid — happy path.
func TestOptionC_OneTimeValid(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionC(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_one_time_schedule" "once" {
  title        = "One-off deploy"
  prompt       = "Deploy v2.1"
  scheduled_at = "2025-06-01T09:00:00Z"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("devin_one_time_schedule.once", "scheduled_at", "2025-06-01T09:00:00Z"),
				),
			},
		},
	})
}

// TestOptionC_RecurringMissingFrequency — schema-level Required catches this.
func TestOptionC_RecurringMissingFrequency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionC(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_recurring_schedule" "bad" {
  title  = "Missing freq"
  prompt = "Will fail"
}`,
				ExpectError: MustCompileRegex(`The argument "frequency" is required`),
			},
		},
	})
}

// TestOptionC_OneTimeMissingScheduledAt — schema-level Required catches this.
func TestOptionC_OneTimeMissingScheduledAt(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionC(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_one_time_schedule" "bad" {
  title  = "Missing scheduled_at"
  prompt = "Will fail"
}`,
				ExpectError: MustCompileRegex(`The argument "scheduled_at" is required`),
			},
		},
	})
}

// TestOptionC_InvalidCron — ValidateFunc catches at plan time.
func TestOptionC_InvalidCron(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionC(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_recurring_schedule" "bad" {
  title     = "Bad cron"
  prompt    = "Will fail"
  frequency = "not-a-cron"
}`,
				ExpectError: MustCompileRegex(`must be a valid 5-field cron expression`),
			},
		},
	})
}

// TestOptionC_InvalidDatetime — ValidateFunc catches at plan time.
func TestOptionC_InvalidDatetime(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionC(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_one_time_schedule" "bad" {
  title        = "Bad datetime"
  prompt       = "Will fail"
  scheduled_at = "next tuesday"
}`,
				ExpectError: MustCompileRegex(`must be a valid ISO 8601`),
			},
		},
	})
}
