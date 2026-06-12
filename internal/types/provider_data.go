package types

import "github.com/tobydrinkall/terraform-provider-devin/internal/client"

// ProviderData is passed to resources and data sources via Configure.
type ProviderData struct {
	Client         *client.Client
	OrganizationID string
}
