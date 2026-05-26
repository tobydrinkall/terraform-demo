// Package enterprise_poc demonstrates two architectural approaches for handling
// enterprise-scope endpoints alongside organization-scope endpoints in the
// Devin Terraform provider (Decision 5.1).
//
// This file implements Option A: Single Provider with conditional enterprise support.
package enterprise_poc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ProviderSingle returns a single "devin" provider that conditionally enables
// enterprise resources based on the API key's permissions.
//
// Architecture mirrors how GitHub, PagerDuty, and Datadog handle multi-scope:
//
//	provider "devin" {
//	  api_key = var.devin_api_key          # org-scope OR enterprise-scope key
//	  org_id  = "org-abc123"               # always required
//	  enterprise_id = "enterprise-xyz789"  # optional — unlocks enterprise resources
//	}
func ProviderSingle() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("DEVIN_API_KEY", nil),
				Description: "The API key for authenticating with the Devin API v3. " +
					"If the key is an enterprise-scoped service user key, enterprise " +
					"resources and data sources will be available. Can also be set " +
					"via the `DEVIN_API_KEY` environment variable.",
			},
			"org_id": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("DEVIN_ORG_ID", nil),
				Description: "The organization ID for org-scoped API calls " +
					"(e.g., `org-abc123`). Can also be set via the `DEVIN_ORG_ID` " +
					"environment variable.",
			},
			"enterprise_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("DEVIN_ENTERPRISE_ID", nil),
				Description: "The enterprise ID for enterprise-scoped API calls " +
					"(e.g., `enterprise-xyz789`). When set, enterprise data sources " +
					"become available. Requires an enterprise-scoped service user key. " +
					"Can also be set via the `DEVIN_ENTERPRISE_ID` environment variable.",
			},
			"base_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "https://api.devin.ai/v3",
				Description: "The base URL of the Devin API. Defaults to `https://api.devin.ai/v3`.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			// Org-scope resources (always available)
			"devin_knowledge_note": resourceKnowledgeNotePOC(),
			"devin_session":        resourceSessionPOC(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			// Org-scope data sources (always available)
			"devin_knowledge_notes": dataSourceKnowledgeNotesPOC(),
			// Enterprise-scope data sources (require enterprise key)
			"devin_enterprise_audit_logs": DataSourceEnterpriseAuditLogs(),
			"devin_enterprise_members":    dataSourceEnterpriseMembersPOC(),
		},
		ConfigureContextFunc: configureSingleProvider,
	}
}

// DevinClient holds authenticated client state, including whether enterprise
// features are available.
type DevinClient struct {
	APIKey       string
	OrgID        string
	EnterpriseID string
	BaseURL      string
	HTTPClient   *http.Client

	// enterpriseEnabled is set during provider configuration by probing the
	// enterprise endpoint. Resources check this before making enterprise calls.
	enterpriseEnabled bool
}

// IsEnterpriseEnabled returns whether the provider was configured with a valid
// enterprise key and enterprise_id.
func (c *DevinClient) IsEnterpriseEnabled() bool {
	return c.enterpriseEnabled
}

// RequireEnterprise returns a diagnostic error if enterprise features are not
// available. Resources/data sources call this at the top of their CRUD funcs.
//
// The error message tells the user exactly what they need:
//
//	Error: Enterprise features not available
//
//	This resource requires an enterprise-scoped service user API key and
//	the "enterprise_id" provider attribute. To use enterprise resources:
//
//	1. Create a service user with enterprise scope in the Devin dashboard
//	2. Set the "enterprise_id" attribute in the provider block
//	3. Use the enterprise-scoped API key
func (c *DevinClient) RequireEnterprise() diag.Diagnostics {
	if c.enterpriseEnabled {
		return nil
	}
	return diag.Diagnostics{
		{
			Severity: diag.Error,
			Summary:  "Enterprise features not available",
			Detail: "This resource requires an enterprise-scoped service user API key " +
				"and the \"enterprise_id\" provider attribute.\n\n" +
				"To use enterprise resources:\n" +
				"  1. Create a service user with enterprise scope in the Devin dashboard\n" +
				"     (Settings > Enterprise > Service Users)\n" +
				"  2. Set the \"enterprise_id\" attribute in the provider block:\n\n" +
				"     provider \"devin\" {\n" +
				"       api_key       = var.devin_api_key\n" +
				"       org_id        = var.devin_org_id\n" +
				"       enterprise_id = var.devin_enterprise_id  # <-- add this\n" +
				"     }\n\n" +
				"  3. Use the enterprise-scoped API key as the api_key value\n\n" +
				"For more information, see: https://docs.devin.ai/enterprise/service-users",
		},
	}
}

// OrgURL builds a URL for an organization-scope API call.
// Example: /v3/organizations/org-abc123/knowledge
func (c *DevinClient) OrgURL(path string) string {
	return fmt.Sprintf("%s/organizations/%s%s", c.BaseURL, c.OrgID, path)
}

// EnterpriseURL builds a URL for an enterprise-scope API call.
// Example: /v3/enterprise/enterprise-xyz789/audit-logs
func (c *DevinClient) EnterpriseURL(path string) string {
	return fmt.Sprintf("%s/enterprise/%s%s", c.BaseURL, c.EnterpriseID, path)
}

func configureSingleProvider(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	apiKey := d.Get("api_key").(string)
	orgID := d.Get("org_id").(string)
	baseURL := d.Get("base_url").(string)

	client := &DevinClient{
		APIKey:     apiKey,
		OrgID:      orgID,
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}

	// Check if enterprise_id is configured
	if v, ok := d.GetOk("enterprise_id"); ok {
		client.EnterpriseID = v.(string)
		client.enterpriseEnabled = true
	}

	return client, nil
}

// --- Placeholder org-scope resources (minimal stubs) ---

func resourceKnowledgeNotePOC() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a knowledge note in the Devin platform.",
		CreateContext: func(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics { return nil },
		ReadContext:   func(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics { return nil },
		UpdateContext: func(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics { return nil },
		DeleteContext: func(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics { return nil },
		Schema: map[string]*schema.Schema{
			"name":    {Type: schema.TypeString, Required: true},
			"content": {Type: schema.TypeString, Required: true},
		},
	}
}

func resourceSessionPOC() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates a Devin session.",
		CreateContext: func(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics { return nil },
		ReadContext:   func(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics { return nil },
		UpdateContext: func(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics { return nil },
		DeleteContext: func(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics { return nil },
		Schema: map[string]*schema.Schema{
			"prompt": {Type: schema.TypeString, Required: true},
		},
	}
}

func dataSourceKnowledgeNotesPOC() *schema.Resource {
	return &schema.Resource{
		Description: "Lists knowledge notes.",
		ReadContext: func(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics { return nil },
		Schema: map[string]*schema.Schema{
			"notes": {Type: schema.TypeList, Computed: true, Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"id":   {Type: schema.TypeString, Computed: true},
					"name": {Type: schema.TypeString, Computed: true},
				},
			}},
		},
	}
}
