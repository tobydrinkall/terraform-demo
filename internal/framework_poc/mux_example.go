package framework_poc

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
	sdkprovider "github.com/tobydrinkall/terraform-demo/internal/provider"
)

// ServeMuxedProvider demonstrates how to serve both an SDK-based provider
// and a Framework-based provider from a single binary using terraform-plugin-mux.
//
// Architecture:
//
//	                    ┌──────────────────────┐
//	                    │   Terraform CLI       │
//	                    └──────────┬───────────┘
//	                               │ gRPC (tfprotov6)
//	                    ┌──────────▼───────────┐
//	                    │   tf6muxserver        │
//	                    │   (Protocol v6 Mux)   │
//	                    └──┬───────────────┬───┘
//	           ┌───────────▼──┐      ┌─────▼───────────┐
//	           │ tf5to6server │      │ providerserver   │
//	           │ (SDK v2      │      │ (Framework       │
//	           │  upgraded)   │      │  native v6)      │
//	           └───────┬──────┘      └─────┬───────────┘
//	        ┌──────────▼──────┐    ┌───────▼──────────┐
//	        │ SDK Resources   │    │ Framework        │
//	        │ e.g. devin_     │    │ Resources e.g.   │
//	        │ session         │    │ devin_knowledge  │
//	        └─────────────────┘    │ _note            │
//	                               └──────────────────┘
//
// This allows incremental migration: new resources use Framework,
// existing SDK resources continue working without changes.
func ServeMuxedProvider(ctx context.Context, version string) error {
	// Step 1: Upgrade the SDK provider from protocol v5 to v6
	upgradedSDKProvider, err := tf5to6server.UpgradeServer(
		ctx,
		sdkprovider.New().GRPCProvider,
	)
	if err != nil {
		return err
	}

	// Step 2: Create the Framework provider server (already protocol v6)
	frameworkProvider := providerserver.NewProtocol6(New(version)())

	// Step 3: Combine both into a single muxed server
	providers := []func() tfprotov6.ProviderServer{
		func() tfprotov6.ProviderServer { return upgradedSDKProvider },
		frameworkProvider,
	}

	muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		return err
	}

	// Step 4: Serve the muxed provider
	return tf6server.Serve(
		"registry.terraform.io/tobydrinkall/devin",
		muxServer.ProviderServer,
	)
}
