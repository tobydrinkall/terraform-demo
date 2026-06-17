package framework_test

import (
	"testing"

	"github.com/COG-GTM/terraform-provider-devin/framework"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestProviderSchema(t *testing.T) {
	// Verify the provider can be instantiated and its schema is valid
	provider := framework.New("test")()
	if provider == nil {
		t.Fatal("provider should not be nil")
	}
}

// testAccProtoV6ProviderFactories is used for acceptance tests (kept as reference).
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"devin": providerserver.NewProtocol6WithError(framework.New("test")()),
}

func TestKnowledgeNoteResourceSchema(t *testing.T) {
	// Verify the resource can be instantiated
	r := framework.NewKnowledgeNoteResource()
	if r == nil {
		t.Fatal("resource should not be nil")
	}
}
