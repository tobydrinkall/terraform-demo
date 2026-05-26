package schedule_poc

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func testProviderOptionA() *schema.Provider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"devin_schedule_a": ResourceScheduleOptionA(),
		},
	}
}

// TestOptionA_RecurringValid — happy path: recurring + frequency.
func TestOptionA_RecurringValid(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionA(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_schedule_a" "daily" {
  title         = "Daily standup"
  prompt        = "Summarise open PRs"
  schedule_type = "recurring"
  frequency     = "0 9 * * 1-5"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("devin_schedule_a.daily", "schedule_type", "recurring"),
					resource.TestCheckResourceAttr("devin_schedule_a.daily", "frequency", "0 9 * * 1-5"),
				),
			},
		},
	})
}

// TestOptionA_OneTimeValid — happy path: one_time + scheduled_at.
func TestOptionA_OneTimeValid(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionA(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_schedule_a" "once" {
  title         = "One-off deploy"
  prompt        = "Deploy v2.1"
  schedule_type = "one_time"
  scheduled_at  = "2025-06-01T09:00:00Z"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("devin_schedule_a.once", "schedule_type", "one_time"),
					resource.TestCheckResourceAttr("devin_schedule_a.once", "scheduled_at", "2025-06-01T09:00:00Z"),
				),
			},
		},
	})
}

// TestOptionA_RecurringMissingFrequency — error case: recurring without
// frequency should produce a diagnostic error at apply time.
func TestOptionA_RecurringMissingFrequency(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionA(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_schedule_a" "bad" {
  title         = "Bad recurring"
  prompt        = "Will fail"
  schedule_type = "recurring"
}`,
				ExpectError: MustCompileRegex(`Missing required attribute for recurring schedule`),
			},
		},
	})
}

// TestOptionA_OneTimeMissingScheduledAt — error case: one_time without
// scheduled_at should produce a diagnostic error at apply time.
func TestOptionA_OneTimeMissingScheduledAt(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionA(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_schedule_a" "bad" {
  title         = "Bad one-time"
  prompt        = "Will fail"
  schedule_type = "one_time"
}`,
				ExpectError: MustCompileRegex(`Missing required attribute for one-time schedule`),
			},
		},
	})
}

// TestOptionA_RecurringWithScheduledAt — error case: providing the wrong
// attribute for the schedule type.
func TestOptionA_RecurringWithScheduledAt(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"devin": func() (*schema.Provider, error) { return testProviderOptionA(), nil },
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_schedule_a" "bad" {
  title         = "Wrong combo"
  prompt        = "Will fail"
  schedule_type = "recurring"
  scheduled_at  = "2025-06-01T09:00:00Z"
}`,
				ExpectError: MustCompileRegex(`Invalid attribute for recurring schedule`),
			},
		},
	})
}
