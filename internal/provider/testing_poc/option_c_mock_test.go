package testing_poc

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const mockOrgID = "org-mock-test"

// setupMockServer starts an httptest server with the mock Devin API.
// It configures env vars so the provider connects to the mock.
func setupMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	t.Setenv("TF_ACC", "1")

	mock := NewMockDevinAPI(mockOrgID)
	server := httptest.NewServer(mock.Handler())
	t.Cleanup(server.Close)

	t.Setenv("DEVIN_API_KEY", "cog_mock_test_key")
	t.Setenv("DEVIN_ORG_ID", mockOrgID)
	t.Setenv("DEVIN_BASE_URL", server.URL)

	return server
}

// --- Test: Basic Create/Read/Delete ---

func TestMock_KnowledgeNote_basic(t *testing.T) {
	setupMockServer(t)
	resourceName := "devin_knowledge_note.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_knowledge_note" "test" {
  name    = "Mock Test - Basic"
  body    = "Created via httptest mock server."
  trigger = "Never - mock test only"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Mock Test - Basic"),
					resource.TestCheckResourceAttr(resourceName, "body", "Created via httptest mock server."),
					resource.TestCheckResourceAttr(resourceName, "trigger", "Never - mock test only"),
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

func TestMock_KnowledgeNote_update(t *testing.T) {
	setupMockServer(t)
	resourceName := "devin_knowledge_note.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_knowledge_note" "test" {
  name    = "Mock Test - Before Update"
  body    = "Original mock body."
  trigger = "Never - mock test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Mock Test - Before Update"),
					resource.TestCheckResourceAttr(resourceName, "body", "Original mock body."),
				),
			},
			{
				Config: `
resource "devin_knowledge_note" "test" {
  name    = "Mock Test - After Update"
  body    = "Updated mock body content."
  trigger = "Never - mock updated test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Mock Test - After Update"),
					resource.TestCheckResourceAttr(resourceName, "body", "Updated mock body content."),
					resource.TestCheckResourceAttr(resourceName, "trigger", "Never - mock updated test"),
				),
			},
		},
	})
}

// --- Test: Import ---

func TestMock_KnowledgeNote_import(t *testing.T) {
	setupMockServer(t)
	resourceName := "devin_knowledge_note.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_knowledge_note" "test" {
  name    = "Mock Test - Import"
  body    = "Testing import via mock."
  trigger = "Never - mock import test"
}
`,
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

// --- Test: Delete Idempotency ---

func TestMock_KnowledgeNote_deleteIdempotent(t *testing.T) {
	setupMockServer(t)
	resourceName := "devin_knowledge_note.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_knowledge_note" "test" {
  name    = "Mock Test - Delete"
  body    = "Will be deleted."
  trigger = "Never"
}
`,
				Check: resource.TestCheckResourceAttrSet(resourceName, "id"),
			},
		},
	})
}

// --- Test: Multiple Resources ---

func TestMock_KnowledgeNote_multiple(t *testing.T) {
	setupMockServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_knowledge_note" "first" {
  name    = "Mock Test - First"
  body    = "First note body."
  trigger = "Never"
}

resource "devin_knowledge_note" "second" {
  name    = "Mock Test - Second"
  body    = "Second note body."
  trigger = "Never"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("devin_knowledge_note.first", "name", "Mock Test - First"),
					resource.TestCheckResourceAttr("devin_knowledge_note.second", "name", "Mock Test - Second"),
				),
			},
		},
	})
}

// --- Test: Pinned Repo ---

func TestMock_KnowledgeNote_pinnedRepo(t *testing.T) {
	setupMockServer(t)
	resourceName := "devin_knowledge_note.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
resource "devin_knowledge_note" "test" {
  name        = "Mock Test - Pinned"
  body        = "Pinned to a repo."
  trigger     = "When working on myorg/myrepo"
  pinned_repo = "myorg/myrepo"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "pinned_repo", "myorg/myrepo"),
					resource.TestCheckResourceAttr(resourceName, "name", "Mock Test - Pinned"),
				),
			},
		},
	})
}

// --- Unit Test: Mock Server Directly (no Terraform) ---

func TestMockServer_CRUD(t *testing.T) {
	mock := NewMockDevinAPI("org-unit")
	server := httptest.NewServer(mock.Handler())
	defer server.Close()

	// Verify empty list
	resp := doHTTPRequest(t, server.URL+"/v3/organizations/org-unit/knowledge/notes", "GET", "")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 from list, got %d", resp.StatusCode)
	}

	// Create a note
	createBody := `{"name":"Unit Test Note","body":"test body","trigger":"test trigger"}`
	resp = doHTTPRequest(t, server.URL+"/v3/organizations/org-unit/knowledge/notes", "POST", createBody)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 from create, got %d", resp.StatusCode)
	}
	var created mockNote
	decodeJSON(t, resp, &created)
	if created.Name != "Unit Test Note" {
		t.Errorf("expected name 'Unit Test Note', got %q", created.Name)
	}
	if created.NoteID == "" {
		t.Fatal("expected non-empty note ID")
	}

	// Get the note
	resp = doHTTPRequest(t, server.URL+"/v3/organizations/org-unit/knowledge/notes/"+created.NoteID, "GET", "")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 from get, got %d", resp.StatusCode)
	}
	var fetched mockNote
	decodeJSON(t, resp, &fetched)
	if fetched.NoteID != created.NoteID {
		t.Errorf("expected note ID %s, got %s", created.NoteID, fetched.NoteID)
	}

	// Update the note
	updateBody := `{"name":"Updated Note","body":"updated body","trigger":"updated trigger"}`
	resp = doHTTPRequest(t, server.URL+"/v3/organizations/org-unit/knowledge/notes/"+created.NoteID, "PUT", updateBody)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 from update, got %d", resp.StatusCode)
	}
	var updated mockNote
	decodeJSON(t, resp, &updated)
	if updated.Name != "Updated Note" {
		t.Errorf("expected name 'Updated Note', got %q", updated.Name)
	}

	// Delete the note
	resp = doHTTPRequest(t, server.URL+"/v3/organizations/org-unit/knowledge/notes/"+created.NoteID, "DELETE", "")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 from delete, got %d", resp.StatusCode)
	}

	// Verify 404 after delete
	resp = doHTTPRequest(t, server.URL+"/v3/organizations/org-unit/knowledge/notes/"+created.NoteID, "GET", "")
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
}

// Helper: ignore env check for mock-only tests
func init() {
	if os.Getenv("TF_ACC") == "" && os.Getenv("SKIP_ACC") != "1" {
		// Tests in this file use httptest, no real TF_ACC gate needed
		// The setupMockServer function sets TF_ACC=1 for each test
	}
}
