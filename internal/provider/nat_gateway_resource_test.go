package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

// NATGatewayProviderTestSuite reuses the provider test suite for Megaport
type NATGatewayProviderTestSuite ProviderTestSuite

func TestNATGatewayProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(NATGatewayProviderTestSuite))
}

// getNATGatewayTestConfig queries the staging NAT Gateway sessions API to get a valid
// speed and session count for acceptance testing.
func getNATGatewayTestConfig() (speed int, sessionCount int, err error) {
	client, err := getTestClient()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create test client: %w", err)
	}

	sessions, err := client.NATGatewayService.ListNATGatewaySessions(context.Background())
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list NAT Gateway sessions: %w", err)
	}
	if len(sessions) == 0 {
		return 0, 0, fmt.Errorf("no NAT Gateway sessions available")
	}
	if len(sessions[0].SessionCount) == 0 {
		return 0, 0, fmt.Errorf("no session counts available for speed %d", sessions[0].SpeedMbps)
	}

	return sessions[0].SpeedMbps, sessions[0].SessionCount[0], nil
}

// TestAccMegaportNATGateway_Basic tests the full lifecycle of a NAT Gateway resource
func (suite *NATGatewayProviderTestSuite) TestAccMegaportNATGateway_Basic() {
	natGWName := RandomTestName()
	natGWNameUpdated := RandomTestName()
	resourceName := "megaport_nat_gateway.test"

	speed, sessionCount, err := getNATGatewayTestConfig()
	if err != nil {
		suite.T().Skipf("Skipping NAT Gateway test: %v", err)
	}

	configInitial := providerConfig + fmt.Sprintf(`
data "megaport_location" "test_location" {
    id = %d
}

resource "megaport_nat_gateway" "test" {
    product_name         = "%s"
    location_id          = data.megaport_location.test_location.id
    speed                = %d
    contract_term_months = 1
    session_count        = %d
    diversity_zone       = "red"

    resource_tags = {
        "key1" = "value1"
        "key2" = "value2"
    }
}
`, MCRTestLocationIDNum, natGWName, speed, sessionCount)

	configUpdated := providerConfig + fmt.Sprintf(`
data "megaport_location" "test_location" {
    id = %d
}

resource "megaport_nat_gateway" "test" {
    product_name         = "%s"
    location_id          = data.megaport_location.test_location.id
    speed                = %d
    contract_term_months = 1
    session_count        = %d
    diversity_zone       = "red"

    resource_tags = {
        "key1" = "value1-updated"
        "key3" = "value3"
    }
}
`, MCRTestLocationIDNum, natGWNameUpdated, speed, sessionCount)

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: configInitial,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "product_name", natGWName),
					resource.TestCheckResourceAttr(resourceName, "speed", fmt.Sprintf("%d", speed)),
					resource.TestCheckResourceAttr(resourceName, "contract_term_months", "1"),
					resource.TestCheckResourceAttr(resourceName, "session_count", fmt.Sprintf("%d", sessionCount)),
					resource.TestCheckResourceAttr(resourceName, "diversity_zone", "red"),
					resource.TestCheckResourceAttr(resourceName, "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr(resourceName, "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet(resourceName, "product_uid"),
					resource.TestCheckResourceAttrSet(resourceName, "provisioning_status"),
					resource.TestCheckResourceAttrSet(resourceName, "create_date"),
					resource.TestCheckResourceAttrSet(resourceName, "created_by"),
					resource.TestCheckResourceAttrSet(resourceName, "location_id"),
					resource.TestCheckResourceAttrSet(resourceName, "asn"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					var rawState map[string]string
					for _, m := range state.Modules {
						if len(m.Resources) > 0 {
							if v, ok := m.Resources[resourceName]; ok {
								rawState = v.Primary.Attributes
							}
						}
					}
					return rawState["product_uid"], nil
				},
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			// Update and verify
			{
				Config: configUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "product_name", natGWNameUpdated),
					resource.TestCheckResourceAttr(resourceName, "resource_tags.key1", "value1-updated"),
					resource.TestCheckResourceAttr(resourceName, "resource_tags.key3", "value3"),
					resource.TestCheckNoResourceAttr(resourceName, "resource_tags.key2"),
				),
			},
		},
	})
}
