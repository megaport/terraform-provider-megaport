// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/suite"
)

var providerConfig = fmt.Sprintf(`
provider "megaport" {
  environment = "staging"
  access_key = "%s"
  secret_key     = "%s"
  accept_purchase_terms = true
}
`, os.Getenv("MEGAPORT_ACCESS_KEY"), os.Getenv("MEGAPORT_SECRET_KEY"))

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"scaffolding": providerserver.NewProtocol6WithError(New("test")()),
	"megaport":    providerserver.NewProtocol6WithError(New("test")()),
}

type ProviderTestSuite struct {
	suite.Suite
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
	default:
		return "", fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// waitForProvisioningStatus returns a TestCheckFunc that waits for a resource to reach the expected provisioning status.
// This function polls the Megaport API directly to check the actual provisioning_status until it matches expectedStatus.
// This is necessary before updating contract terms, as the API requires the resource to be fully provisioned (LIVE status).
// Based on testing in the Staging API, resources typically take 40-60 seconds to reach LIVE status.
func waitForProvisioningStatus(resourceName string, expectedStatus string, timeout time.Duration) func(*terraform.State) error {
	return func(s *terraform.State) error {
		startTime := time.Now()
		pollInterval := 2 * time.Second

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
