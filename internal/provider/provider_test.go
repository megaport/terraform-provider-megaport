// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	megaport "github.com/megaport/megaportgo"
)

var providerConfig = fmt.Sprintf(`
provider "megaport" {
  environment = "staging"
  access_key = "%s"
  secret_key     = "%s"
  accept_purchase_terms = true
}
`, os.Getenv("MEGAPORT_ACCESS_KEY"), os.Getenv("MEGAPORT_SECRET_KEY"))

// managedAccountProviderConfig mirrors providerConfig but sets
// managed_account_uid so the client acts on behalf of a managed account.
func managedAccountProviderConfig(managedAccountUID string) string {
	return fmt.Sprintf(`
provider "megaport" {
  environment            = "staging"
  access_key             = "%s"
  secret_key             = "%s"
  accept_purchase_terms  = true
  managed_account_uid    = "%s"
}
`, os.Getenv("MEGAPORT_ACCESS_KEY"), os.Getenv("MEGAPORT_SECRET_KEY"), managedAccountUID)
}

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"scaffolding": providerserver.NewProtocol6WithError(New("test")()),
	"megaport":    providerserver.NewProtocol6WithError(New("test")()),
}

var (
	testClient     *megaport.Client
	testClientOnce sync.Once
	testClientErr  error
)

// getTestClient returns a singleton Megaport API client for use in tests
func getTestClient() (*megaport.Client, error) {
	testClientOnce.Do(func() {
		accessKey := os.Getenv("MEGAPORT_ACCESS_KEY")
		secretKey := os.Getenv("MEGAPORT_SECRET_KEY")

		if accessKey == "" || secretKey == "" {
			testClientErr = fmt.Errorf("MEGAPORT_ACCESS_KEY and MEGAPORT_SECRET_KEY must be set")
			return
		}

		testClient, testClientErr = megaport.New(nil,
			megaport.WithEnvironment(megaport.EnvironmentStaging),
			megaport.WithCredentials(accessKey, secretKey),
		)
		if testClientErr != nil {
			return
		}

		ctx := context.Background()
		_, testClientErr = testClient.Authorize(ctx)
	})

	return testClient, testClientErr
}

// getProductStatus retrieves the provisioning status for a product based on its type
func getProductStatus(ctx context.Context, client *megaport.Client, productUID string, resourceType string) (string, error) {
	switch resourceType {
	case "megaport_port", "megaport_lag_port":
		port, err := client.PortService.GetPort(ctx, productUID)
		if err != nil {
			return "", err
		}
		return port.ProvisioningStatus, nil
	case "megaport_mcr":
		mcr, err := client.MCRService.GetMCR(ctx, productUID)
		if err != nil {
			return "", err
		}
		return mcr.ProvisioningStatus, nil
	case "megaport_mve":
		mve, err := client.MVEService.GetMVE(ctx, productUID)
		if err != nil {
			return "", err
		}
		return mve.ProvisioningStatus, nil
	case "megaport_vxc":
		vxc, err := client.VXCService.GetVXC(ctx, productUID)
		if err != nil {
			return "", err
		}
		return vxc.ProvisioningStatus, nil
	case "megaport_nat_gateway":
		gw, err := client.NATGatewayService.GetNATGateway(ctx, productUID)
		if err != nil {
			return "", err
		}
		return gw.ProvisioningStatus, nil
	default:
		return "", fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// waitForProvisioningStatus returns a TestCheckFunc that waits for a resource to reach LIVE provisioning status.
// This function polls the Megaport API directly to check the actual provisioning_status until it reaches LIVE.
// This is necessary before updating contract terms, as the API requires the resource to be fully provisioned (LIVE status).
// Based on testing in the Staging API, resources typically take 40-60 seconds to reach LIVE status, but we allow
// up to 20 minutes to account for slower provisioning times in production environments.
func waitForProvisioningStatus(resourceName string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		startTime := time.Now()
		pollInterval := 2 * time.Second
		timeout := 20 * time.Minute
		expectedStatus := "LIVE"

		// Get the resource from state to extract product_uid and resource type
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}

		productUID, ok := rs.Primary.Attributes["product_uid"]
		if !ok {
			return fmt.Errorf("product_uid attribute not found for resource: %s", resourceName)
		}

		// Get API client
		client, err := getTestClient()
		if err != nil {
			return fmt.Errorf("failed to create test client: %w", err)
		}

		ctx := context.Background()

		for time.Since(startTime) < timeout {
			// Query the API directly for current status using the appropriate service
			status, err := getProductStatus(ctx, client, productUID, rs.Type)
			if err != nil {
				return fmt.Errorf("failed to get product status from API: %w", err)
			}

			// Check if status matches expected
			if status == expectedStatus {
				// Resource has reached expected status
				return nil
			}

			time.Sleep(pollInterval)
		}

		// Timeout reached - get final status for error message
		finalStatus, _ := getProductStatus(ctx, client, productUID, rs.Type)
		return fmt.Errorf("ERROR: timeout waiting for %s to reach status %s after %v. Current status: %s",
			resourceName, expectedStatus, timeout, finalStatus)
	}
}

// TestAccMegaportProvider_ManagedAccountUID exercises managed_account_uid
// end to end: with it set, the client sends X-Call-Context and resources
// provision inside the managed account rather than the partner's own.
// Requires MEGAPORT_TEST_MANAGED_ACCOUNT_UID, a real managed account UID the
// test credentials can act on behalf of. A fake UID would just fail
// authorization instead of exercising the attribute, so the test skips
// rather than substituting a placeholder.
func TestAccMegaportProvider_ManagedAccountUID(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()

	managedAccountUID := os.Getenv("MEGAPORT_TEST_MANAGED_ACCOUNT_UID")
	if managedAccountUID == "" {
		t.Skip("MEGAPORT_TEST_MANAGED_ACCOUNT_UID not set, skipping managed account provisioning test")
	}

	locationID, _ := findPortTestLocation(t, 1000)
	portName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: managedAccountProviderConfig(managedAccountUID) + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_port" "managed" {
					product_name           = "%s"
					port_speed             = 1000
					location_id            = data.megaport_location.test_location.id
					contract_term_months   = 1
					marketplace_visibility = false
				}`, locationID, portName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_port.managed", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.managed", "company_uid", managedAccountUID),
				),
			},
		},
	})
}
