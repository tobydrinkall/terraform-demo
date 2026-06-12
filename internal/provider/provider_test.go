package provider_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/tobydrinkall/terraform-provider-devin/internal/provider"
)

func TestProviderSchema(t *testing.T) {
	t.Parallel()

	resp, err := providerserver.NewProtocol6WithError(provider.New("test")())()
	if err != nil {
		t.Fatalf("failed to create provider server: %s", err)
	}

	ctx := context.Background()
	schemaResp, err := resp.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("failed to get provider schema: %s", err)
	}

	if schemaResp.Provider == nil {
		t.Fatal("expected provider schema, got nil")
	}

	expectedAttrs := []string{"api_key", "organization_id", "base_url"}
	for _, attr := range expectedAttrs {
		found := false
		for _, a := range schemaResp.Provider.Block.Attributes {
			if a.Name == attr {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected attribute %q in provider schema", attr)
		}
	}

	if schemaResp.ResourceSchemas == nil {
		t.Fatal("expected resource schemas, got nil")
	}
	if _, ok := schemaResp.ResourceSchemas["devin_knowledge_note"]; !ok {
		t.Error("expected devin_knowledge_note resource in schema")
	}
}
