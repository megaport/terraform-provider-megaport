package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

type MCRBasicProviderTestSuite ProviderTestSuite

func TestMCRBasicProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRBasicProviderTestSuite))
}

func (suite *MCRBasicProviderTestSuite) TestAccMegaportMCR_Basic() {
	mcrName := RandomTestName()
	prefixFilterName := RandomTestName()
	prefixFilterName2 := RandomTestName()
	prefixFilterNameNew := RandomTestName()
	prefixFilterNameNew2 := RandomTestName()
	prefixFilterNameNew3 := RandomTestName()
	prefixFilterNameNew4 := RandomTestName()
	costCentreName := RandomTestName()
	mcrNameNew := RandomTestName()
	mcrNameNew2 := RandomTestName()
	costCentreNameNew := RandomTestName()
	costCentreNameNew2 := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr_basic" "mcr" {
					product_name             = "%s"
					port_speed               = 1000
					location_id              = data.megaport_location.test_location.id
					contract_term_months     = 12
					cost_centre              = "%s"

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

					prefix_filter_lists = [
					{
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 25
							le      = 32
						  },
						  {
							action  = "deny"
							prefix  = "10.0.2.0/24"
							ge      = 25
							le      = 27
						  }
						]
					  },
					  {
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 26
							le      = 32
						  },
						  {
							action  = "deny"
							prefix  = "10.0.2.0/24"
							ge      = 24
							le      = 25
						  }
						]
					  }]
				  }
				  `, MCRTestLocationIDNum, mcrName, costCentreName, prefixFilterName, prefixFilterName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.description", prefixFilterName),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.description", prefixFilterName2),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.0.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.1.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.1.le", "27"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.0.ge", "26"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.1.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.1.le", "25"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_mcr_basic.mcr",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_mcr_basic.mcr"
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
			},
			// Update Test 1
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr_basic" "mcr" {
					product_name             = "%s"
					port_speed               = 1000
					location_id              = data.megaport_location.test_location.id
					contract_term_months     = 12
					cost_centre              = "%s"
					resource_tags = {
						"key1updated" = "value1updated"
						"key2updated" = "value2updated"
					}

					prefix_filter_lists = [
					{
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 24
							le      = 32
						  },
						  {
							action  = "deny"
							prefix  = "10.0.2.0/24"
							ge      = 25
							le      = 29
						  }
						]
					  },
					  {
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 25
							le      = 32
						  },
						  {
							action  = "deny"
							prefix  = "10.0.2.0/24"
							ge      = 24
							le      = 26
						  }
						]
					  },
					  {
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 24
							le      = 24
						  },
						  {
							action  = "deny"
							prefix  = "10.0.2.0/24"
							ge      = 27
							le      = 32
						  }
						]
					  }]
				  }
				  `, MCRTestLocationIDNum, mcrName, costCentreName, prefixFilterNameNew, prefixFilterNameNew2, prefixFilterNameNew3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.#", "3"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.description", prefixFilterNameNew),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.description", prefixFilterNameNew2),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.2.description", prefixFilterNameNew3),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.0.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.1.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.1.le", "29"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.0.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.1.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.1.entries.1.le", "26"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.2.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.2.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.2.entries.0.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.2.entries.0.le", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.2.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.2.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.2.entries.1.ge", "27"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.2.entries.1.le", "32"),
				),
			},
			// Update Test 2
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr_basic" "mcr" {
					product_name             = "%s"
					port_speed               = 1000
					location_id              = data.megaport_location.test_location.id
					contract_term_months     = 12
					cost_centre              = "%s"

					resource_tags = {
						"key1updated" = "value1updated"
						"key2updated" = "value2updated"
					}

					prefix_filter_lists = [{
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 28
							le      = 32
						  }
						]
					  }]
				  }
				  `, MCRTestLocationIDNum, mcrNameNew, costCentreNameNew, prefixFilterNameNew4),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "product_name", mcrNameNew),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.description", prefixFilterNameNew4),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.#", "1"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.#", "1"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.0.ge", "28"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.0.entries.0.le", "32"),
				),
			},
			// Update Test 3
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr_basic" "mcr" {
					product_name             = "%s"
					port_speed               = 1000
					location_id              = data.megaport_location.test_location.id
					contract_term_months     = 12
					cost_centre              = "%s"

					prefix_filter_lists = []
				  }
				  `, MCRTestLocationIDNum, mcrNameNew2, costCentreNameNew2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "product_name", mcrNameNew2),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "cost_centre", costCentreNameNew2),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "prefix_filter_lists.#", "0"),
				),
			},
		},
	})
}

func (suite *MCRBasicProviderTestSuite) TestAccMegaportMCRCustomASN_Basic() {
	mcrName := RandomTestName()
	mcrNameNew := RandomTestName()
	costCentreName := RandomTestName()
	costCentreNameNew := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr_basic" "mcr" {
					product_name             = "%s"
					port_speed               = 1000
					location_id              = data.megaport_location.test_location.id
					contract_term_months     = 12
					cost_centre              = "%s"
					asn = 65000

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}
				  }
				  `, MCRTestLocationIDNum, mcrName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "asn", "65000"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_mcr_basic.mcr",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_mcr_basic.mcr"
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
			},
			// Update Test 1
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr_basic" "mcr" {
					product_name             = "%s"
					port_speed               = 1000
					location_id              = data.megaport_location.test_location.id
					contract_term_months     = 12
					cost_centre              = "%s"
					asn = 65000

					resource_tags = {"key1updated" = "value1updated", "key2updated" = "value2updated"}
				  }
				  `, MCRTestLocationIDNum, mcrNameNew, costCentreNameNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "product_name", mcrNameNew),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr_basic.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr_basic.mcr", "asn", "65000"),
				),
			},
		},
	})
}
