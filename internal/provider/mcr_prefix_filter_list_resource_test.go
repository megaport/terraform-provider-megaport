package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

type MCRPrefixFilterListProviderTestSuite ProviderTestSuite

func TestMCRPrefixFilterListProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRPrefixFilterListProviderTestSuite))
}

func (suite *MCRPrefixFilterListProviderTestSuite) TestAccMegaportMCRPrefixFilterList_Basic() {
	mcrName := RandomTestName()
	prefixFilterName := RandomTestName()
	prefixFilterName2 := RandomTestName()
	prefixFilterNameNew := RandomTestName()
	prefixFilterNameNew2 := RandomTestName()
	prefixFilterNameNew3 := RandomTestName()
	prefixFilterNameNew4 := RandomTestName()
	costCentreName := RandomTestName()
	mcrNameNew := RandomTestName()
	costCentreNameNew := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create MCR and prefix filter lists
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

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

					# Explicitly set empty prefix filter lists since we're using standalone resources
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
					address_family = "IPv4"
					entries = [
						{
							action = "permit"
							prefix = "10.0.1.0/24"
							ge     = 26
							le     = 32
						},
						{
							action = "deny"
							prefix = "10.0.2.0/24"
							ge     = 24
							le     = 25
						}
					]
				}
				`, MCRTestLocationIDNum, mcrName, costCentreName, prefixFilterName, prefixFilterName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					// MCR checks
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),

					// Prefix filter list 1 checks
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "description", prefixFilterName),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "address_family", "IPv4"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.0.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.1.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.1.le", "27"),
					resource.TestCheckResourceAttrSet("megaport_mcr_prefix_filter_list.prefix_list_1", "id"),
					resource.TestCheckResourceAttrSet("megaport_mcr_prefix_filter_list.prefix_list_1", "last_updated"),

					// Prefix filter list 2 checks
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "description", prefixFilterName2),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "address_family", "IPv4"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.0.ge", "26"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.1.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.1.le", "25"),
					resource.TestCheckResourceAttrSet("megaport_mcr_prefix_filter_list.prefix_list_2", "id"),
					resource.TestCheckResourceAttrSet("megaport_mcr_prefix_filter_list.prefix_list_2", "last_updated"),
				),
			},
			// Test ImportState for prefix filter list 1
			{
				ResourceName:      "megaport_mcr_prefix_filter_list.prefix_list_1",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName1 := "megaport_mcr.mcr"
					resourceName2 := "megaport_mcr_prefix_filter_list.prefix_list_1"
					var mcrUID, prefixListID string

					for _, m := range state.Modules {
						if len(m.Resources) > 0 {
							if v, ok := m.Resources[resourceName1]; ok {
								mcrUID = v.Primary.Attributes["product_uid"]
							}
							if v, ok := m.Resources[resourceName2]; ok {
								prefixListID = v.Primary.Attributes["id"]
							}
						}
					}
					return fmt.Sprintf("%s:%s", mcrUID, prefixListID), nil
				},
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			// Update Test 1: Modify existing prefix filter lists and add a new one
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

					resource_tags = {
						"key1updated" = "value1updated"
						"key2updated" = "value2updated"
					}

					# Explicitly set empty prefix filter lists since we're using standalone resources
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
							ge     = 24
							le     = 32
						},
						{
							action = "deny"
							prefix = "10.0.2.0/24"
							ge     = 25
							le     = 29
						}
					]
				}

				resource "megaport_mcr_prefix_filter_list" "prefix_list_2" {
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
							ge     = 24
							le     = 26
						}
					]
				}

				resource "megaport_mcr_prefix_filter_list" "prefix_list_3" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv4"
					entries = [
						{
							action = "permit"
							prefix = "10.0.1.0/24"
							ge     = 24
							le     = 30
						},
						{
							action = "deny"
							prefix = "10.0.2.0/24"
							ge     = 27
							le     = 32
						}
					]
				}
				`, MCRTestLocationIDNum, mcrName, costCentreName, prefixFilterNameNew, prefixFilterNameNew2, prefixFilterNameNew3),
				Check: resource.ComposeAggregateTestCheckFunc(
					// MCR checks
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2updated", "value2updated"),

					// Updated prefix filter list 1 checks
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "description", prefixFilterNameNew),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.0.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.1.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.1.le", "29"),

					// Updated prefix filter list 2 checks
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "description", prefixFilterNameNew2),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.0.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.1.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.1.le", "26"),

					// New prefix filter list 3 checks
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_3", "description", prefixFilterNameNew3),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_3", "entries.0.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_3", "entries.0.le", "30"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_3", "entries.1.ge", "27"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_3", "entries.1.le", "32"),
					resource.TestCheckResourceAttrSet("megaport_mcr_prefix_filter_list.prefix_list_3", "id"),
				),
			},
			// Update Test 2: Reduce to single prefix filter list
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

					resource_tags = {
						"key1updated" = "value1updated"
						"key2updated" = "value2updated"
					}

					# Explicitly set empty prefix filter lists since we're using standalone resources
					prefix_filter_lists = []

					lifecycle {
						ignore_changes = [prefix_filter_lists]
					}
				}

				resource "megaport_mcr_prefix_filter_list" "prefix_list_single" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv4"
					entries = [
						{
							action = "permit"
							prefix = "10.0.1.0/24"
							ge     = 28
							le     = 32
						}
					]
				}
				`, MCRTestLocationIDNum, mcrNameNew, costCentreNameNew, prefixFilterNameNew4),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_single", "description", prefixFilterNameNew4),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_single", "entries.#", "1"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_single", "entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_single", "entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_single", "entries.0.ge", "28"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_single", "entries.0.le", "32"),
				),
			},
		},
	})
}

func (suite *MCRPrefixFilterListProviderTestSuite) TestAccMegaportMCRPrefixFilterList_IPv6() {
	mcrName := RandomTestName()
	prefixFilterName := RandomTestName()
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

					# Explicitly set empty prefix filter lists since we're using standalone resources
					prefix_filter_lists = []

					lifecycle {
						ignore_changes = [prefix_filter_lists]
					}
				}

				resource "megaport_mcr_prefix_filter_list" "ipv6_list" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv6"
					entries = [
						{
							action = "permit"
							prefix = "2001:db8::/32"
							ge     = 48
							le     = 64
						},
						{
							action = "deny"
							prefix = "2001:db8:1::/48"
							ge     = 56
							le     = 128
						}
					]
				}
				`, MCRTestLocationIDNum, mcrName, costCentreName, prefixFilterName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "description", prefixFilterName),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "address_family", "IPv6"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.0.prefix", "2001:db8::/32"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.0.ge", "48"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.0.le", "64"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.1.prefix", "2001:db8:1::/48"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.1.ge", "56"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.1.le", "128"),
					resource.TestCheckResourceAttrSet("megaport_mcr_prefix_filter_list.ipv6_list", "id"),
				),
			},
		},
	})
}
