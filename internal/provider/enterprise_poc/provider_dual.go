package enterprise_poc

// This file sketches Option B: a SEPARATE "devin-enterprise" provider.
//
// This is NOT meant to compile as a standalone binary — it illustrates the
// architecture, UX, and trade-offs of splitting enterprise into its own provider.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ──────────────────────────────────────────────────────────────────────────────
// Architecture Diagram: Dual Provider
//
//  ┌──────────────────────────────────────────────────────────────────────┐
//  │                     User's Terraform Config                         │
//  │                                                                     │
//  │  provider "devin" {                provider "devin-enterprise" {    │
//  │    api_key = var.org_key             api_key = var.ent_key          │
//  │    org_id  = var.org_id              enterprise_id = var.ent_id     │
//  │  }                                 }                                │
//  │                                                                     │
//  │  resource "devin_knowledge_note"   data "devin-enterprise_audit_    │
//  │    "deploy_guide" { ... }            logs" "recent" { ... }         │
//  │                                                                     │
//  │  resource "devin_session"          data "devin-enterprise_members"  │
//  │    "fix_bug" { ... }                 "all" { ... }                  │
//  │                                                                     │
//  │  # PROBLEM: knowledge notes exist in BOTH providers                 │
//  │  # Which one do you use? They hit the same API.                     │
//  │  data "devin_knowledge_notes" "all" { ... }                         │
//  │  data "devin-enterprise_knowledge_notes" "all" { ... }              │
//  └──────────────────────────────────────────────────────────────────────┘
//
//  The key problem: shared resources like knowledge_notes can be managed at
//  both org and enterprise scope. With dual providers, users must choose
//  which provider to use — or worse, the same resource appears in both.
// ──────────────────────────────────────────────────────────────────────────────

// ProviderEnterprise returns a separate "devin-enterprise" provider.
//
// In practice this would live in a separate Go module and produce a separate
// binary: terraform-provider-devin-enterprise.
//
// User config would look like:
//
//	terraform {
//	  required_providers {
//	    devin = {
//	      source  = "devin-ai/devin"
//	      version = "~> 1.0"
//	    }
//	    devin-enterprise = {
//	      source  = "devin-ai/devin-enterprise"
//	      version = "~> 1.0"
//	    }
//	  }
//	}
func ProviderEnterprise() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("DEVIN_ENTERPRISE_API_KEY", nil),
				Description: "The enterprise-scoped service user API key. " +
					"Can also be set via the `DEVIN_ENTERPRISE_API_KEY` environment variable.",
			},
			"enterprise_id": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("DEVIN_ENTERPRISE_ID", nil),
				Description: "The enterprise ID (e.g., `enterprise-xyz789`). " +
					"Can also be set via the `DEVIN_ENTERPRISE_ID` environment variable.",
			},
			"base_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "https://api.devin.ai/v3",
				Description: "The base URL of the Devin API.",
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			// Enterprise-only data sources
			"devin-enterprise_audit_logs": dualAuditLogsPOC(),
			"devin-enterprise_members":    dualMembersPOC(),

			// PROBLEM: Should knowledge notes be here too?
			// They are org-scoped in the API (POST /v3/organizations/{org_id}/knowledge)
			// but enterprise admins may want to read them across orgs.
			//
			// Option 1: Don't include → user must use the "devin" provider for knowledge
			// Option 2: Include a read-only version → resource duplication
			// Option 3: Include with cross-org filter → different schema than the org version
			"devin-enterprise_knowledge_notes": dualKnowledgeNotesPOC(),
		},
		// Enterprise provider has NO resources — enterprise endpoints are read-only
		// (audit logs, member lists, usage reports). If enterprise write endpoints
		// are added later, they'd go here.
		ResourcesMap: map[string]*schema.Resource{},

		ConfigureContextFunc: configureEnterpriseProvider,
	}
}

// EnterpriseClient is a separate client type for the enterprise provider.
// It cannot make org-scope calls — it only knows about enterprise endpoints.
type EnterpriseClient struct {
	APIKey       string
	EnterpriseID string
	BaseURL      string
	HTTPClient   *http.Client
}

// EnterpriseURL builds a URL for an enterprise API call.
func (c *EnterpriseClient) EnterpriseURL(path string) string {
	return fmt.Sprintf("%s/enterprise/%s%s", c.BaseURL, c.EnterpriseID, path)
}

func configureEnterpriseProvider(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return &EnterpriseClient{
		APIKey:       d.Get("api_key").(string),
		EnterpriseID: d.Get("enterprise_id").(string),
		BaseURL:      d.Get("base_url").(string),
		HTTPClient:   &http.Client{},
	}, nil
}

// --- Stub data sources for the enterprise provider ---

func dualAuditLogsPOC() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves enterprise audit logs.",
		ReadContext: func(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
			d.SetId("audit-logs")
			return nil
		},
		Schema: map[string]*schema.Schema{
			"start_date": {Type: schema.TypeString, Required: true},
			"end_date":   {Type: schema.TypeString, Required: true},
			"entries":    {Type: schema.TypeList, Computed: true, Elem: &schema.Resource{Schema: map[string]*schema.Schema{"id": {Type: schema.TypeString, Computed: true}}}},
		},
	}
}

func dualMembersPOC() *schema.Resource {
	return &schema.Resource{
		Description: "Lists enterprise members.",
		ReadContext: func(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
			d.SetId("members")
			return nil
		},
		Schema: map[string]*schema.Schema{
			"members": {Type: schema.TypeList, Computed: true, Elem: &schema.Resource{Schema: map[string]*schema.Schema{"email": {Type: schema.TypeString, Computed: true}}}},
		},
	}
}

func dualKnowledgeNotesPOC() *schema.Resource {
	return &schema.Resource{
		Description: "Lists knowledge notes across all organizations in the enterprise. " +
			"Note: This is a READ-ONLY cross-org view. To manage individual notes, " +
			"use the `devin_knowledge_note` resource in the `devin` provider.",
		ReadContext: func(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
			d.SetId("enterprise-knowledge-notes")
			return nil
		},
		Schema: map[string]*schema.Schema{
			"org_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter to a specific org within the enterprise.",
			},
			"notes": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":     {Type: schema.TypeString, Computed: true},
						"name":   {Type: schema.TypeString, Computed: true},
						"org_id": {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}
