// Package poc contains proof-of-concept implementations for Decision 0.3:
// how should the Terraform provider resolve the org_id needed by the
// Devin API v3 URL scheme (https://api.devin.ai/v3/organizations/{org_id}/...).
//
// Each option_*.go file provides a standalone provider implementation using
// the terraform-plugin-framework, matching the conventions of the main provider.
package poc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SelfResponse is the JSON shape returned by GET /v3/self.
//
// Observed response (2026-05-26):
//
//	{
//	  "principal_type": "service_user",
//	  "service_user_id": "service-user-<hex>",
//	  "service_user_name": "terraform-demo",
//	  "org_id": "org-<hex>"
//	}
//
// A service-user key returns exactly one org_id.
// An enterprise/multi-org key may return additional fields (not yet observed).
type SelfResponse struct {
	PrincipalType   string `json:"principal_type"`
	ServiceUserID   string `json:"service_user_id"`
	ServiceUserName string `json:"service_user_name"`
	OrgID           string `json:"org_id"`
}

// resolveSelfOrgID calls GET {baseURL}/v3/self and extracts the org_id.
func resolveSelfOrgID(baseURL, apiKey string) (string, error) {
	httpClient := &http.Client{Timeout: 10 * time.Second}

	url := baseURL + "/v3/self"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("building GET /v3/self request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling GET /v3/self: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading /v3/self response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf(
			"GET /v3/self returned HTTP %d — the API key appears to be invalid or expired",
			resp.StatusCode,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET /v3/self returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var self SelfResponse
	if err := json.Unmarshal(body, &self); err != nil {
		return "", fmt.Errorf("parsing /v3/self JSON: %w", err)
	}

	if self.OrgID == "" {
		return "", fmt.Errorf(
			"GET /v3/self did not include an org_id — " +
				"the API key may be invalid or the account has no organization membership",
		)
	}

	return self.OrgID, nil
}
