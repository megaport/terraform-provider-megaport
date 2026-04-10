package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

type MCRIpsecAddonProviderTestSuite ProviderTestSuite

func TestMCRIpsecAddonProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRIpsecAddonProviderTestSuite))
}

func (suite *MCRIpsecAddonProviderTestSuite) TestAccMegaportMCRIpsecAddon_Basic() {
	mcrName := RandomTestName()
	costCentreName := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create MCR and IPSec add-on with 10 tunnels
			{
				Config: providerConfig + testAccMCRIpsecAddonConfig(mcrName, costCentreName, 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_ipsec_addon.test", "tunnel_count", "10"),
					resource.TestCheckResourceAttrSet("megaport_mcr_ipsec_addon.test", "add_on_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr_ipsec_addon.test", "mcr_id"),
				),
			},
			// Plan-only check — no drift
			{
				Config:             providerConfig + testAccMCRIpsecAddonConfig(mcrName, costCentreName, 10),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update to 20 tunnels
			{
				Config: providerConfig + testAccMCRIpsecAddonConfig(mcrName, costCentreName, 20),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_ipsec_addon.test", "tunnel_count", "20"),
					resource.TestCheckResourceAttrSet("megaport_mcr_ipsec_addon.test", "add_on_uid"),
				),
			},
			// Import
			{
				ResourceName:                         "megaport_mcr_ipsec_addon.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "add_on_uid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["megaport_mcr_ipsec_addon.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: megaport_mcr_ipsec_addon.test")
					}
					return rs.Primary.Attributes["mcr_id"] + ":" + rs.Primary.Attributes["add_on_uid"], nil
				},
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
		},
	})
}

func testAccMCRIpsecAddonConfig(mcrName, costCentreName string, tunnelCount int) string {
	return fmt.Sprintf(`
data "megaport_location" "test_location" {
	id = %d
}

resource "megaport_mcr" "mcr" {
	product_name         = "%s"
	port_speed           = 1000
	location_id          = data.megaport_location.test_location.id
	contract_term_months = 1
	cost_centre          = "%s"

	prefix_filter_lists = []

	lifecycle {
		ignore_changes = [prefix_filter_lists]
	}
}

resource "megaport_mcr_ipsec_addon" "test" {
	mcr_id       = megaport_mcr.mcr.product_uid
	tunnel_count = %d
}
`, MCRTestLocationIDNum, mcrName, costCentreName, tunnelCount)
}
