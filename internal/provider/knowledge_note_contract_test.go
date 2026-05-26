package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tobydrinkall/terraform-demo/internal/testutil"
)

const (
	testAPIKey = "cog_test_key_contract"
	testOrgID  = "org-test"
)

func testAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"devin": providerserver.NewProtocol6WithError(New("test")()),
	}
}

func setupContractTest(t *testing.T) (context.Context, *testutil.WireMockContainer) {
	t.Helper()
	if os.Getenv("SKIP_DOCKER_TESTS") == "1" {
		t.Skip("Skipping: SKIP_DOCKER_TESTS=1")
	}
	t.Setenv("TF_ACC", "1")

	ctx := context.Background()
	wm := testutil.StartWireMock(ctx, t)
	t.Cleanup(func() { wm.Stop(ctx) })
	return ctx, wm
}

func providerConfigWithBaseURL(baseURL string) string {
	return fmt.Sprintf(`
provider "devin" {
  api_key         = %q
  organization_id = %q
  base_url        = %q
}
`, testAPIKey, testOrgID, baseURL)
}

func TestContract_CreateAndRead(t *testing.T) {
	_, wm := setupContractTest(t)

	noteID := "note-contract-001"
	noteResp := testutil.DefaultNoteResponse(noteID, "Contract Test Note", "Test body content", "When testing contracts")

	testutil.RegisterCreateNoteStub(t, wm, noteResp, 200)
	testutil.RegisterGetNoteStub(t, wm, noteID, noteResp, 200)
	testutil.RegisterDeleteNoteStub(t, wm, noteID, noteResp, 200)

	config := providerConfigWithBaseURL(wm.BaseURL) + fmt.Sprintf(`
resource "devin_knowledge_note" "test" {
  name    = %q
  body    = %q
  trigger = %q
}
`, "Contract Test Note", "Test body content", "When testing contracts")

	resourceName := "devin_knowledge_note.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Contract Test Note"),
					resource.TestCheckResourceAttr(resourceName, "body", "Test body content"),
					resource.TestCheckResourceAttr(resourceName, "trigger", "When testing contracts"),
					resource.TestCheckResourceAttr(resourceName, "id", noteID),
					resource.TestCheckResourceAttr(resourceName, "is_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "access_type", "org"),
					resource.TestCheckResourceAttr(resourceName, "folder_path", "/"),
				),
			},
		},
	})

	// Post-test WireMock verifications
	wm.VerifyCallCount(t, "POST", "/v3/organizations/"+testOrgID+"/knowledge/notes", 1)
	wm.VerifyAuthHeader(t, "POST", "/v3/organizations/"+testOrgID+"/knowledge/notes", testAPIKey)
}

func TestContract_UpdateInPlace(t *testing.T) {
	_, wm := setupContractTest(t)

	noteID := "note-contract-002"
	createResp := testutil.DefaultNoteResponse(noteID, "Original Name", "Original body", "Original trigger")
	updateResp := testutil.DefaultNoteResponse(noteID, "Updated Name", "Updated body", "Updated trigger")
	updateResp.UpdatedAt = 1716000001

	notesPath := "/v3/organizations/" + testOrgID + "/knowledge/notes"
	notePath := notesPath + "/" + noteID
	scenario := "update-lifecycle"

	// POST creates the note and transitions to "Created" state
	testutil.RegisterCreateNoteStub(t, wm, createResp, 200)

	// GET returns original response in "Started" state (initial WireMock state)
	wm.RegisterStub(t, testutil.StubMapping{
		ScenarioName:  scenario,
		RequiredState: "Started",
		Request:       testutil.StubRequest{Method: "GET", URLPath: notePath},
		Response: testutil.StubResponse{
			Status: 200, JSONBody: createResp,
			Headers: map[string]string{"Content-Type": "application/json"},
		},
	})

	// PUT updates the note and transitions scenario to "Updated"
	wm.RegisterStub(t, testutil.StubMapping{
		ScenarioName: scenario,
		NewState:     "Updated",
		Request:      testutil.StubRequest{Method: "PUT", URLPath: notePath},
		Response: testutil.StubResponse{
			Status: 200, JSONBody: updateResp,
			Headers: map[string]string{"Content-Type": "application/json"},
		},
	})

	// GET returns updated response in "Updated" state
	wm.RegisterStub(t, testutil.StubMapping{
		ScenarioName:  scenario,
		RequiredState: "Updated",
		Request:       testutil.StubRequest{Method: "GET", URLPath: notePath},
		Response: testutil.StubResponse{
			Status: 200, JSONBody: updateResp,
			Headers: map[string]string{"Content-Type": "application/json"},
		},
	})

	testutil.RegisterDeleteNoteStub(t, wm, noteID, updateResp, 200)

	createConfig := providerConfigWithBaseURL(wm.BaseURL) + `
resource "devin_knowledge_note" "test" {
  name    = "Original Name"
  body    = "Original body"
  trigger = "Original trigger"
}
`
	updateConfig := providerConfigWithBaseURL(wm.BaseURL) + `
resource "devin_knowledge_note" "test" {
  name    = "Updated Name"
  body    = "Updated body"
  trigger = "Updated trigger"
}
`

	resourceName := "devin_knowledge_note.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Original Name"),
					resource.TestCheckResourceAttr(resourceName, "id", noteID),
				),
			},
			{
				Config: updateConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Updated Name"),
					resource.TestCheckResourceAttr(resourceName, "body", "Updated body"),
					resource.TestCheckResourceAttr(resourceName, "id", noteID),
				),
			},
		},
	})

	// Verify PUT was called
	wm.VerifyCallCount(t, "PUT", notePath, 1)
}

func TestContract_DeleteHandles404(t *testing.T) {
	_, wm := setupContractTest(t)

	noteID := "note-contract-003"
	noteResp := testutil.DefaultNoteResponse(noteID, "To Be Deleted", "body", "trigger")

	testutil.RegisterCreateNoteStub(t, wm, noteResp, 200)
	testutil.RegisterGetNoteStub(t, wm, noteID, noteResp, 200)
	testutil.RegisterDeleteNoteStub(t, wm, noteID, nil, 404)

	config := providerConfigWithBaseURL(wm.BaseURL) + `
resource "devin_knowledge_note" "test" {
  name    = "To Be Deleted"
  body    = "body"
  trigger = "trigger"
}
`
	resourceName := "devin_knowledge_note.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "To Be Deleted"),
					resource.TestCheckResourceAttr(resourceName, "id", noteID),
				),
			},
		},
	})
}

func TestContract_ReadNotFound_RemovesFromState(t *testing.T) {
	_, wm := setupContractTest(t)

	noteID := "note-contract-004"
	noteResp := testutil.DefaultNoteResponse(noteID, "Disappearing Note", "body", "trigger")

	testutil.RegisterCreateNoteStub(t, wm, noteResp, 200)
	testutil.RegisterGetNoteStub(t, wm, noteID, nil, 404)
	testutil.RegisterDeleteNoteStub(t, wm, noteID, nil, 404)

	config := providerConfigWithBaseURL(wm.BaseURL) + `
resource "devin_knowledge_note" "test" {
  name    = "Disappearing Note"
  body    = "body"
  trigger = "trigger"
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:             config,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestContract_WithPinnedRepo(t *testing.T) {
	_, wm := setupContractTest(t)

	noteID := "note-contract-005"
	pinnedRepo := "myorg/myrepo"
	noteResp := testutil.DefaultNoteResponse(noteID, "Pinned Note", "body", "trigger")
	noteResp.PinnedRepo = &pinnedRepo

	testutil.RegisterCreateNoteStub(t, wm, noteResp, 200)
	testutil.RegisterGetNoteStub(t, wm, noteID, noteResp, 200)
	testutil.RegisterDeleteNoteStub(t, wm, noteID, noteResp, 200)

	config := providerConfigWithBaseURL(wm.BaseURL) + `
resource "devin_knowledge_note" "test" {
  name        = "Pinned Note"
  body        = "body"
  trigger     = "trigger"
  pinned_repo = "myorg/myrepo"
}
`
	resourceName := "devin_knowledge_note.test"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "pinned_repo", "myorg/myrepo"),
					resource.TestCheckResourceAttr(resourceName, "id", noteID),
				),
			},
		},
	})

	// Verify pinned_repo was in the request body
	requests := wm.GetRequests(t, "POST", "/v3/organizations/"+testOrgID+"/knowledge/notes")
	if len(requests) == 0 {
		t.Fatal("expected at least one POST request")
	}
	if requests[0].Body == "" {
		t.Fatal("expected non-empty request body")
	}
	if !stringContains(requests[0].Body, "myorg/myrepo") {
		t.Errorf("expected request body to contain 'myorg/myrepo', got: %s", requests[0].Body)
	}
}

func TestContract_ValidationError(t *testing.T) {
	_, wm := setupContractTest(t)

	errorResp := map[string]interface{}{
		"detail": []map[string]interface{}{
			{"loc": []string{"body", "name"}, "msg": "field required", "type": "value_error.missing"},
		},
	}

	testutil.RegisterCreateNoteStub(t, wm, errorResp, 422)

	config := providerConfigWithBaseURL(wm.BaseURL) + `
resource "devin_knowledge_note" "test" {
  name    = ""
  body    = "body"
  trigger = "trigger"
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexpFromString("Error creating knowledge note"),
			},
		},
	})
}

func regexpFromString(s string) *regexp.Regexp {
	return regexp.MustCompile(regexp.QuoteMeta(s))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
