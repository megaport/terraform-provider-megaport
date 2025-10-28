package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/suite"
)

type MCRPrefixFilterListDataSourceProviderTestSuite ProviderTestSuite

func TestMCRPrefixFilterListDataSourceProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRPrefixFilterListDataSourceProviderTestSuite))
}

func (suite *MCRPrefixFilterListDataSourceProviderTestSuite) TestAccMegaportMCRPrefixFilterListDataSource_Basic() {
	mcrName := RandomTestName()
	prefixFilterName := RandomTestName()
	prefixFilterName2 := RandomTestName()
	costCentreName := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				
				resource "megaport_mcr" "mcr" {
					product_name         = "%s"
					port_speed          = 1000
					location_id         = data.megaport_location.test_location.id
					contract_term_months = 12
					cost_centre         = "%s"
					
					# Explicitly set empty prefix filter lists to avoid conflicts
					prefix_filter_lists = []

					lifecycle {
						ignore_changes = [prefix_filter_lists]
					}
				}

				resource "megaport_mcr_prefix_filter_list" "prefix_list_1" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv4"
					entries = [
						{
							action = "permit"
							prefix = "10.0.1.0/24"
							ge     = 25
							le     = 32
						},
						{
							action = "deny"
							prefix = "10.0.2.0/24"
							ge     = 25
							le     = 27
						}
					]
				}

				resource "megaport_mcr_prefix_filter_list" "prefix_list_2" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv6"
					entries = [
						{
							action = "permit"
							prefix = "2001:db8::/32"
							ge     = 48
							le     = 64
						}
					]
				}

				data "megaport_mcr_prefix_filter_lists" "all_lists" {
					mcr_id = megaport_mcr.mcr.product_uid
					depends_on = [
						megaport_mcr_prefix_filter_list.prefix_list_1,
						megaport_mcr_prefix_filter_list.prefix_list_2
					]
				}
				`, MCRTestLocationIDNum, mcrName, costCentreName, prefixFilterName, prefixFilterName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Data source checks
					resource.TestCheckResourceAttr("data.megaport_mcr_prefix_filter_lists.all_lists", "prefix_filter_lists.#", "2"),

					// Check that we can find our created prefix filter lists
					resource.TestCheckTypeSetElemNestedAttrs("data.megaport_mcr_prefix_filter_lists.all_lists", "prefix_filter_lists.*", map[string]string{
						"description":    prefixFilterName,
						"address_family": "IPv4",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.megaport_mcr_prefix_filter_lists.all_lists", "prefix_filter_lists.*", map[string]string{
						"description":    prefixFilterName2,
						"address_family": "IPv6",
					}),
				),
			},
		},
	})
}
