package sdkv2_test

import (
	"testing"

	"github.com/COG-GTM/terraform-provider-devin/sdkv2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestProviderSchema(t *testing.T) {
	p := sdkv2.New("test")
	if p == nil {
		t.Fatal("provider should not be nil")
	}

	if err := p.InternalValidate(); err != nil {
		t.Fatalf("provider schema validation failed: %s", err)
	}
}

func TestProviderHasResources(t *testing.T) {
	p := sdkv2.New("test")
	resources := p.ResourcesMap

	if _, ok := resources["devin_knowledge_note"]; !ok {
		t.Fatal("expected devin_knowledge_note resource to be registered")
	}
}

func TestKnowledgeNoteSchema(t *testing.T) {
	p := sdkv2.New("test")
	r := p.ResourcesMap["devin_knowledge_note"]

	requiredFields := []string{"name", "trigger", "body"}
	for _, field := range requiredFields {
		s, ok := r.Schema[field]
		if !ok {
			t.Errorf("expected schema to contain field %q", field)
			continue
		}
		if !s.Required {
			t.Errorf("expected field %q to be required", field)
		}
	}

	computedFields := []string{"folder_path", "macro", "org_id", "access_type", "created_at", "updated_at"}
	for _, field := range computedFields {
		s, ok := r.Schema[field]
		if !ok {
			t.Errorf("expected schema to contain computed field %q", field)
			continue
		}
		if !s.Computed {
			t.Errorf("expected field %q to be computed", field)
		}
	}

	// Verify is_enabled default
	isEnabled := r.Schema["is_enabled"]
	if isEnabled.Default != true {
		t.Error("expected is_enabled to default to true")
	}
}

// providerFactories for acceptance tests (reference).
var providerFactories = map[string]func() (*schema.Provider, error){
	"devin": func() (*schema.Provider, error) {
		return sdkv2.New("test"), nil
	},
}

// Ensure providerFactories is used to avoid lint error.
var _ = providerFactories
