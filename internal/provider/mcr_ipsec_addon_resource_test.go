package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/suite"
)

type MCRIpsecAddonProviderTestSuite ProviderTestSuite

func TestMCRIpsecAddonProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRIpsecAddonProviderTestSuite))
}

func (suite *MCRIpsecAddonProviderTestSuite) TestAccMegaportMCRIpsecAddon_Basic() {
	mcrName := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create MCR and attach IPSec add-on with 10 tunnels
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_mcr" "mcr" {
					product_name         = "%s"
					port_speed           = 1000
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
				}
				resource "megaport_mcr_ipsec_addon" "ipsec" {
					mcr_id       = megaport_mcr.mcr.product_uid
					tunnel_count = 10
				}
				`, MCRTestLocationIDNum, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_ipsec_addon.ipsec", "tunnel_count", "10"),
					resource.TestCheckResourceAttrSet("megaport_mcr_ipsec_addon.ipsec", "add_on_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr_ipsec_addon.ipsec", "mcr_id"),
				),
			},
			// Verify step 1 produces a clean plan (no drift)
			{
				PlanOnly: true,
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_mcr" "mcr" {
					product_name         = "%s"
					port_speed           = 1000
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
				}
				resource "megaport_mcr_ipsec_addon" "ipsec" {
					mcr_id       = megaport_mcr.mcr.product_uid
					tunnel_count = 10
				}
				`, MCRTestLocationIDNum, mcrName),
			},
			// Update tunnel count to 20
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_mcr" "mcr" {
					product_name         = "%s"
					port_speed           = 1000
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
				}
				resource "megaport_mcr_ipsec_addon" "ipsec" {
					mcr_id       = megaport_mcr.mcr.product_uid
					tunnel_count = 20
				}
				`, MCRTestLocationIDNum, mcrName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("megaport_mcr_ipsec_addon.ipsec", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_ipsec_addon.ipsec", "tunnel_count", "20"),
					resource.TestCheckResourceAttrSet("megaport_mcr_ipsec_addon.ipsec", "add_on_uid"),
				),
			},
		},
	})
}
