package testing_poc

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

const vcrReplayOrgID = "org-vcr-replay"

const cassettesDir = "testdata/cassettes"

// vcrMode returns whether to record or replay cassettes.
// Set VCR_RECORD=1 to record new cassettes against the real API.
func vcrMode() recorder.Mode {
	if os.Getenv("VCR_RECORD") == "1" {
		return recorder.ModeRecordOnly
	}
	return recorder.ModeReplayOnly
}

// newVCRRecorder creates a go-vcr recorder for the given cassette name.
func newVCRRecorder(t *testing.T, cassetteName string) *recorder.Recorder {
	t.Helper()

	cassettePath := filepath.Join(cassettesDir, cassetteName)

	rec, err := recorder.New(
		cassettePath,
		recorder.WithMode(vcrMode()),
		recorder.WithRealTransport(http.DefaultTransport),
		recorder.WithSkipRequestLatency(true),
		recorder.WithHook(func(i *cassette.Interaction) error {
			// Sanitize secrets: redact auth header and scrub real org_id to stable placeholder
			i.Request.Headers.Set("Authorization", "Bearer REDACTED")
			realOrgID := os.Getenv("DEVIN_ORG_ID")
			if realOrgID != "" && realOrgID != vcrReplayOrgID {
				i.Request.URL = strings.ReplaceAll(i.Request.URL, realOrgID, vcrReplayOrgID)
				i.Request.Body = strings.ReplaceAll(i.Request.Body, realOrgID, vcrReplayOrgID)
				i.Response.Body = strings.ReplaceAll(i.Response.Body, realOrgID, vcrReplayOrgID)
			}
			return nil
		}, recorder.BeforeSaveHook),
		recorder.WithMatcher(func(r *http.Request, cr cassette.Request) bool {
			return r.Method == cr.Method && r.URL.String() == cr.URL
		}),
	)
	if err != nil {
		t.Fatalf("failed to create VCR recorder: %v", err)
	}

	return rec
}

// vcrPreCheck ensures env vars are set when recording cassettes.
func vcrPreCheck(t *testing.T) {
	t.Helper()
	if vcrMode() == recorder.ModeRecordOnly {
		if v := os.Getenv("DEVIN_API_KEY"); v == "" {
			if v = os.Getenv("DEVIN_API_KEY_TERRAFORM"); v == "" {
				t.Fatal("DEVIN_API_KEY or DEVIN_API_KEY_TERRAFORM must be set when recording cassettes (VCR_RECORD=1)")
			}
			os.Setenv("DEVIN_API_KEY", v)
		}
		if os.Getenv("DEVIN_ORG_ID") == "" {
			t.Fatal("DEVIN_ORG_ID must be set when recording cassettes (VCR_RECORD=1)")
		}
	} else {
		// Replay mode: set dummy credentials so the provider initializes
		if os.Getenv("DEVIN_API_KEY") == "" {
			os.Setenv("DEVIN_API_KEY", "cog_vcr_replay_dummy")
		}
		if os.Getenv("DEVIN_ORG_ID") == "" {
			os.Setenv("DEVIN_ORG_ID", "org-vcr-replay")
		}
	}
}

// vcrCheckDestroy is a no-op for VCR tests — Terraform handles destroy via the
// recorded/replayed interactions; we don't independently verify API state.
func vcrCheckDestroy(_ *terraform.State) error {
	return nil
}

// --- Test: Basic CRUD via VCR ---

func TestVCR_KnowledgeNote_basic(t *testing.T) {
	rec := newVCRRecorder(t, "knowledge_note_basic")
	defer rec.Stop()

	resourceName := "devin_knowledge_note.test"

	origTransport := http.DefaultTransport
	http.DefaultTransport = rec
	t.Cleanup(func() { http.DefaultTransport = origTransport })

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { vcrPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             vcrCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_knowledge_note" "test" {
  name    = "TF VCR Test - Basic"
  body    = "Created via go-vcr recorded cassette."
  trigger = "Never - VCR test only"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "TF VCR Test - Basic"),
					resource.TestCheckResourceAttr(resourceName, "body", "Created via go-vcr recorded cassette."),
					resource.TestCheckResourceAttr(resourceName, "trigger", "Never - VCR test only"),
					resource.TestCheckResourceAttr(resourceName, "is_enabled", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
				),
			},
		},
	})
}

// --- Test: Update via VCR ---

func TestVCR_KnowledgeNote_update(t *testing.T) {
	rec := newVCRRecorder(t, "knowledge_note_update")
	defer rec.Stop()

	resourceName := "devin_knowledge_note.test"

	origTransport := http.DefaultTransport
	http.DefaultTransport = rec
	t.Cleanup(func() { http.DefaultTransport = origTransport })

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { vcrPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             vcrCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_knowledge_note" "test" {
  name    = "TF VCR Test - Before Update"
  body    = "Original VCR body."
  trigger = "Never - VCR test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "TF VCR Test - Before Update"),
					resource.TestCheckResourceAttr(resourceName, "body", "Original VCR body."),
				),
			},
			{
				Config: `
resource "devin_knowledge_note" "test" {
  name    = "TF VCR Test - After Update"
  body    = "Updated VCR body content."
  trigger = "Never - VCR updated test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "TF VCR Test - After Update"),
					resource.TestCheckResourceAttr(resourceName, "body", "Updated VCR body content."),
				),
			},
		},
	})
}

// --- Test: Import via VCR ---

func TestVCR_KnowledgeNote_import(t *testing.T) {
	rec := newVCRRecorder(t, "knowledge_note_import")
	defer rec.Stop()

	resourceName := "devin_knowledge_note.test"

	origTransport := http.DefaultTransport
	http.DefaultTransport = rec
	t.Cleanup(func() { http.DefaultTransport = origTransport })

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { vcrPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             vcrCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "devin_knowledge_note" "test" {
  name    = "TF VCR Test - Import"
  body    = "Testing import via VCR."
  trigger = "Never - VCR import test"
}
`),
				Check: resource.TestCheckResourceAttrSet(resourceName, "id"),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
