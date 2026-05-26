package testing_poc

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tobydrinkall/terraform-demo/internal/client"
	"github.com/tobydrinkall/terraform-demo/internal/provider"
)

// testAccProtoV6ProviderFactories returns provider factories for acceptance tests.
func testAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"devin": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

// testAccPreCheck validates required environment variables are set.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if v := os.Getenv("DEVIN_API_KEY"); v == "" {
		if v = os.Getenv("DEVIN_API_KEY_TERRAFORM"); v == "" {
			t.Fatal("DEVIN_API_KEY or DEVIN_API_KEY_TERRAFORM must be set for acceptance tests")
		}
		os.Setenv("DEVIN_API_KEY", v)
	}
	if v := os.Getenv("DEVIN_ORG_ID"); v == "" {
		t.Fatal("DEVIN_ORG_ID must be set for acceptance tests")
	}
}

// testAccCheckKnowledgeNoteDestroy verifies the note was deleted from the API.
func testAccCheckKnowledgeNoteDestroy(s *terraform.State) error {
	apiKey := os.Getenv("DEVIN_API_KEY")
	orgID := os.Getenv("DEVIN_ORG_ID")
	c := client.NewClient("", orgID, apiKey)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "devin_knowledge_note" {
			continue
		}
		_, err := c.GetKnowledgeNote(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("knowledge note %s still exists after destroy", rs.Primary.ID)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking note %s: %s", rs.Primary.ID, err)
		}
	}
	return nil
}

// testAccCheckKnowledgeNoteExists verifies a note exists in the API.
func testAccCheckKnowledgeNoteExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("resource ID is empty")
		}

		apiKey := os.Getenv("DEVIN_API_KEY")
		orgID := os.Getenv("DEVIN_ORG_ID")
		c := client.NewClient("", orgID, apiKey)

		note, err := c.GetKnowledgeNote(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error fetching note %s: %s", rs.Primary.ID, err)
		}
		if !strings.HasPrefix(note.NoteID, "note-") {
			return fmt.Errorf("expected note ID starting with 'note-', got %s", note.NoteID)
		}
		return nil
	}
}

// --- Test: Basic Create/Read/Delete ---

func TestAccKnowledgeNote_basic(t *testing.T) {
	resourceName := "devin_knowledge_note.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckKnowledgeNoteDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_knowledge_note" "test" {
  name    = "TF Acc Test - Basic"
  body    = "This note was created by acceptance tests."
  trigger = "Never - acceptance test only"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKnowledgeNoteExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "TF Acc Test - Basic"),
					resource.TestCheckResourceAttr(resourceName, "body", "This note was created by acceptance tests."),
					resource.TestCheckResourceAttr(resourceName, "trigger", "Never - acceptance test only"),
					resource.TestCheckResourceAttr(resourceName, "is_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "access_type", "org"),
					resource.TestCheckResourceAttr(resourceName, "folder_path", "/"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
				),
			},
		},
	})
}

// --- Test: Update In-Place ---

func TestAccKnowledgeNote_update(t *testing.T) {
	resourceName := "devin_knowledge_note.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckKnowledgeNoteDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_knowledge_note" "test" {
  name    = "TF Acc Test - Before Update"
  body    = "Original body content."
  trigger = "Never - acceptance test only"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKnowledgeNoteExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "TF Acc Test - Before Update"),
					resource.TestCheckResourceAttr(resourceName, "body", "Original body content."),
				),
			},
			{
				Config: `
resource "devin_knowledge_note" "test" {
  name    = "TF Acc Test - After Update"
  body    = "Updated body content via acceptance test."
  trigger = "Never - updated acceptance test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKnowledgeNoteExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "TF Acc Test - After Update"),
					resource.TestCheckResourceAttr(resourceName, "body", "Updated body content via acceptance test."),
					resource.TestCheckResourceAttr(resourceName, "trigger", "Never - updated acceptance test"),
				),
			},
		},
	})
}

// --- Test: Import ---

func TestAccKnowledgeNote_import(t *testing.T) {
	resourceName := "devin_knowledge_note.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckKnowledgeNoteDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_knowledge_note" "test" {
  name    = "TF Acc Test - Import"
  body    = "This note tests terraform import."
  trigger = "Never - import acceptance test"
}
`,
				Check: testAccCheckKnowledgeNoteExists(resourceName),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
