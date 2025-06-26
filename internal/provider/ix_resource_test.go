package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

// IXProviderTestSuite reuses the provider test suite for Megaport
type IXProviderTestSuite ProviderTestSuite

func TestIXProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IXProviderTestSuite))
}

// TestAccMegaportIX_Basic tests the basic lifecycle of an IX resource
func (suite *IXProviderTestSuite) TestAccMegaportIX_Basic() {
	ixName := RandomTestName()
	portName := RandomTestName()
	ixNameUpdated := ixName + "-updated"
	ixRateLimit := 500
	ixRateLimitUpdated := 750
	ixVLAN := 2001
	ixVLANUpdated := 2002

	configInitial := fmt.Sprintf(`
resource "megaport_port" "test_port" {
    product_name           = "%s"
    location_id            = %d
    port_speed             = 1000
    marketplace_visibility = false
    contract_term_months   = 1
}

resource "megaport_ix" "test_ix" {
    product_name        = "%s"
    requested_product_uid = megaport_port.test_port.product_uid
    network_service_type = "Sydney IX"
    asn                 = 12345
    mac_address         = "00:CA:FE:BA:BE:01"
    rate_limit          = %d
    vlan                = %d
    shutdown            = false
}

// Test port data source with name filter
data "megaport_ports" "test_port_filter" {
    filter {
        name = "name"
        values = ["%s"]
    }
    depends_on = [megaport_port.test_port]
}

// Test IX data source with name filter
data "megaport_ixs" "test_ix_filter" {
    filter {
        name = "name"
        values = ["%s"]
    }
    depends_on = [megaport_ix.test_ix]
}

// Test IX data source with multiple filters
data "megaport_ixs" "test_ix_multi_filter" {
    filter {
        name = "rate-limit"
        values = ["%d"]
    }
    filter {
        name = "vlan"
        values = ["%d"]
    }
    depends_on = [megaport_ix.test_ix]
}
`, portName, SinglePortTestLocationIDNum, ixName, ixRateLimit, ixVLAN, portName, ixName, ixRateLimit, ixVLAN)

	// Updated Terraform config
	configUpdated := fmt.Sprintf(`
resource "megaport_port" "test_port" {
    product_name           = "%s"
    location_id            = %d
    port_speed             = 1000
    marketplace_visibility = false
    contract_term_months   = 1
}

resource "megaport_ix" "test_ix" {
    product_name        = "%s"
    requested_product_uid = megaport_port.test_port.product_uid
    network_service_type = "Sydney IX"
    asn                 = 12345
    mac_address         = "00:CA:FE:BA:BE:01"
    rate_limit          = %d
    vlan                = %d
    shutdown            = false
}

// Test port data source with name filter
data "megaport_ports" "test_port_filter" {
    filter {
        name = "name"
        values = ["%s"]
    }
    depends_on = [megaport_port.test_port]
}

// Test IX data source with name filter (updated name)
data "megaport_ixs" "test_ix_filter" {
    filter {
        name = "name"
        values = ["%s"]
    }
    depends_on = [megaport_ix.test_ix]
}

// Test IX data source with multiple filters (updated values)
data "megaport_ixs" "test_ix_multi_filter" {
    filter {
        name = "rate-limit"
        values = ["%d"]
    }
    filter {
        name = "vlan"
        values = ["%d"]
    }
    depends_on = [megaport_ix.test_ix]
}
`, portName, SinglePortTestLocationIDNum, ixNameUpdated, ixRateLimitUpdated, ixVLANUpdated, portName, ixNameUpdated, ixRateLimitUpdated, ixVLANUpdated)

	resourceName := "megaport_ix.test_ix"

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configInitial,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "product_name", ixName),
					resource.TestCheckResourceAttr(resourceName, "rate_limit", fmt.Sprintf("%d", ixRateLimit)),
					resource.TestCheckResourceAttr(resourceName, "vlan", fmt.Sprintf("%d", ixVLAN)),
					// Validate port data source results
					resource.TestCheckResourceAttr("data.megaport_ports.test_port_filter", "uids.#", "1"),
					// Validate IX data source results
					resource.TestCheckResourceAttr("data.megaport_ixs.test_ix_filter", "uids.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_ixs.test_ix_multi_filter", "uids.#", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_ix.test_ix",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_ix.test_ix"
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
				ImportStateVerifyIgnore: []string{"last_updated", "requested_product_uid", "shutdown", "mac_address", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status"},
			},
			{
				Config: configUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "product_name", ixNameUpdated),
					resource.TestCheckResourceAttr(resourceName, "rate_limit", fmt.Sprintf("%d", ixRateLimitUpdated)),
					resource.TestCheckResourceAttr(resourceName, "vlan", fmt.Sprintf("%d", ixVLANUpdated)),
					// Validate port data source results (still finding the same port)
					resource.TestCheckResourceAttr("data.megaport_ports.test_port_filter", "uids.#", "1"),
					// Validate IX data source results with updated name
					resource.TestCheckResourceAttr("data.megaport_ixs.test_ix_filter", "uids.#", "1"),
					// Validate IX data source results with updated filters
					resource.TestCheckResourceAttr("data.megaport_ixs.test_ix_multi_filter", "uids.#", "1"),
				),
			},
		},
	})
}
