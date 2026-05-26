package client_stdlib

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestIntegrationCRUD(t *testing.T) {
	apiKey := os.Getenv("DEVIN_API_KEY_TERRAFORM")
	if apiKey == "" {
		t.Skip("DEVIN_API_KEY_TERRAFORM not set, skipping integration test")
	}

	orgID := "org-8d158f07ee0f4678a467078b323880a4"
	client := NewClient(apiKey, orgID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create
	t.Log("Creating knowledge note...")
	created, err := client.CreateKnowledgeNote(ctx, CreateKnowledgeNoteRequest{
		Name:    "stdlib-integration-test-note",
		Body:    "This is a test note created by the stdlib client integration test.",
		Trigger: "When running stdlib integration tests",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	t.Logf("Created note: %s (ID: %s)", created.Name, created.NoteID)

	if created.Name != "stdlib-integration-test-note" {
		t.Errorf("expected name 'stdlib-integration-test-note', got '%s'", created.Name)
	}
	if created.NoteID == "" {
		t.Fatal("expected non-empty note ID")
	}

	noteID := created.NoteID

	// Read
	t.Log("Reading knowledge note...")
	read, err := client.GetKnowledgeNote(ctx, noteID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	t.Logf("Read note: %s (Body: %s)", read.Name, read.Body)

	if read.NoteID != noteID {
		t.Errorf("expected note ID '%s', got '%s'", noteID, read.NoteID)
	}
	if read.Name != "stdlib-integration-test-note" {
		t.Errorf("expected name 'stdlib-integration-test-note', got '%s'", read.Name)
	}

	// Update
	t.Log("Updating knowledge note...")
	updated, err := client.UpdateKnowledgeNote(ctx, noteID, UpdateKnowledgeNoteRequest{
		Name:    "stdlib-integration-test-note-UPDATED",
		Body:    "This note has been updated by the stdlib client.",
		Trigger: "When running stdlib integration tests (updated)",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	t.Logf("Updated note: %s", updated.Name)

	if updated.Name != "stdlib-integration-test-note-UPDATED" {
		t.Errorf("expected updated name, got '%s'", updated.Name)
	}
	if updated.UpdatedAt <= created.UpdatedAt {
		t.Log("Warning: UpdatedAt did not advance (may be same-second update)")
	}

	// List
	t.Log("Listing knowledge notes...")
	list, err := client.ListKnowledgeNotes(ctx, 100, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	t.Logf("Listed %d notes (total: %d)", len(list.Items), list.Total)

	found := false
	for _, item := range list.Items {
		if item.NoteID == noteID {
			found = true
			break
		}
	}
	if !found {
		t.Error("created note not found in list response")
	}

	// Delete
	t.Log("Deleting knowledge note...")
	err = client.DeleteKnowledgeNote(ctx, noteID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	t.Log("Deleted successfully")

	// Verify deletion
	_, err = client.GetKnowledgeNote(ctx, noteID)
	if err == nil {
		t.Error("expected error getting deleted note, got nil")
	} else {
		t.Logf("Verified deletion: %v", err)
	}
}
