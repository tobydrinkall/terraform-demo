package enterprise_poc

import (
	"testing"
)

func TestProviderSingle_Schema(t *testing.T) {
	p := ProviderSingle()

	if err := p.InternalValidate(); err != nil {
		t.Fatalf("provider schema validation failed: %s", err)
	}

	// Verify org-scope resources are registered
	orgResources := []string{"devin_knowledge_note", "devin_session"}
	for _, name := range orgResources {
		if _, ok := p.ResourcesMap[name]; !ok {
			t.Errorf("expected org resource %q to be registered", name)
		}
	}

	// Verify enterprise data sources are registered
	entDataSources := []string{"devin_enterprise_audit_logs", "devin_enterprise_members"}
	for _, name := range entDataSources {
		if _, ok := p.DataSourcesMap[name]; !ok {
			t.Errorf("expected enterprise data source %q to be registered", name)
		}
	}
}

func TestProviderSingle_EnterpriseGate_NoEnterpriseID(t *testing.T) {
	client := &DevinClient{
		APIKey:            "test-key",
		OrgID:             "org-123",
		BaseURL:           "https://api.devin.ai/v3",
		enterpriseEnabled: false,
	}

	diags := client.RequireEnterprise()
	if diags == nil {
		t.Fatal("expected error when enterprise is not enabled")
	}
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Summary != "Enterprise features not available" {
		t.Errorf("unexpected summary: %s", diags[0].Summary)
	}
	// Verify the error message includes actionable instructions
	if diags[0].Detail == "" {
		t.Error("expected detailed instructions in error message")
	}
}

func TestProviderSingle_EnterpriseGate_WithEnterpriseID(t *testing.T) {
	client := &DevinClient{
		APIKey:            "test-enterprise-key",
		OrgID:             "org-123",
		EnterpriseID:      "enterprise-456",
		BaseURL:           "https://api.devin.ai/v3",
		enterpriseEnabled: true,
	}

	diags := client.RequireEnterprise()
	if diags != nil {
		t.Fatalf("expected no error when enterprise is enabled, got: %v", diags)
	}
}

func TestProviderSingle_URLBuilders(t *testing.T) {
	client := &DevinClient{
		OrgID:        "org-abc",
		EnterpriseID: "enterprise-xyz",
		BaseURL:      "https://api.devin.ai/v3",
	}

	orgURL := client.OrgURL("/knowledge")
	expected := "https://api.devin.ai/v3/organizations/org-abc/knowledge"
	if orgURL != expected {
		t.Errorf("OrgURL = %q, want %q", orgURL, expected)
	}

	entURL := client.EnterpriseURL("/audit-logs")
	expected = "https://api.devin.ai/v3/enterprise/enterprise-xyz/audit-logs"
	if entURL != expected {
		t.Errorf("EnterpriseURL = %q, want %q", entURL, expected)
	}
}

func TestProviderEnterprise_Schema(t *testing.T) {
	p := ProviderEnterprise()

	if err := p.InternalValidate(); err != nil {
		t.Fatalf("enterprise provider schema validation failed: %s", err)
	}

	// Enterprise provider should have NO resources (read-only)
	if len(p.ResourcesMap) != 0 {
		t.Errorf("expected 0 resources in enterprise provider, got %d", len(p.ResourcesMap))
	}

	// Should have enterprise data sources
	if _, ok := p.DataSourcesMap["devin-enterprise_audit_logs"]; !ok {
		t.Error("expected devin-enterprise_audit_logs data source")
	}
}

func TestProviderSingle_IsConfigurable(t *testing.T) {
	p := ProviderSingle()

	// Verify all expected provider schema attributes exist
	requiredAttrs := []string{"api_key", "org_id"}
	for _, attr := range requiredAttrs {
		s, ok := p.Schema[attr]
		if !ok {
			t.Errorf("expected provider schema attribute %q", attr)
			continue
		}
		if !s.Required {
			t.Errorf("expected %q to be required", attr)
		}
	}

	optionalAttrs := []string{"enterprise_id", "base_url"}
	for _, attr := range optionalAttrs {
		s, ok := p.Schema[attr]
		if !ok {
			t.Errorf("expected provider schema attribute %q", attr)
			continue
		}
		if s.Required {
			t.Errorf("expected %q to be optional", attr)
		}
	}

	// api_key should be sensitive
	if !p.Schema["api_key"].Sensitive {
		t.Error("expected api_key to be sensitive")
	}
}

func TestDataSourceEnterpriseAuditLogs_Schema(t *testing.T) {
	ds := DataSourceEnterpriseAuditLogs()

	if err := ds.InternalValidate(nil, false); err != nil {
		t.Fatalf("audit logs data source schema validation failed: %s", err)
	}

	// Verify required attributes
	for _, attr := range []string{"start_date", "end_date"} {
		s, ok := ds.Schema[attr]
		if !ok {
			t.Errorf("expected schema attribute %q", attr)
			continue
		}
		if !s.Required {
			t.Errorf("expected %q to be required", attr)
		}
	}

	// Verify computed attributes
	for _, attr := range []string{"entries", "total_count"} {
		s, ok := ds.Schema[attr]
		if !ok {
			t.Errorf("expected schema attribute %q", attr)
			continue
		}
		if !s.Computed {
			t.Errorf("expected %q to be computed", attr)
		}
	}
}

// TestProviderComparison_ResourceCounts compares the two approaches side by side
func TestProviderComparison_ResourceCounts(t *testing.T) {
	single := ProviderSingle()
	dual := ProviderEnterprise()

	t.Logf("Single provider: %d resources, %d data sources",
		len(single.ResourcesMap), len(single.DataSourcesMap))
	t.Logf("Dual provider (enterprise only): %d resources, %d data sources",
		len(dual.ResourcesMap), len(dual.DataSourcesMap))

	// Single provider should have BOTH org and enterprise data sources
	// (it has org-scope ones PLUS enterprise ones)
	if len(single.DataSourcesMap) < 2 {
		t.Error("single provider should have multiple data sources covering both scopes")
	}

	// The "devin" provider in single-provider mode has resources;
	// the enterprise-only dual provider should have none
	if len(single.ResourcesMap) == 0 {
		t.Error("single provider should have org-scope resources")
	}
	if len(dual.ResourcesMap) != 0 {
		t.Error("enterprise-only provider should have no resources")
	}
}

// Verify InternalValidate covers the data source that uses schema.Resource as Elem
func TestDataSourceEnterpriseAuditLogs_SchemaResource(t *testing.T) {
	r := DataSourceEnterpriseAuditLogs()
	if err := r.InternalValidate(nil, false); err != nil {
		t.Fatalf("data source schema invalid: %s", err)
	}
}
