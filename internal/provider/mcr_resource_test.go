package provider

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

const (
	MCRTestLocation      = "Digital Realty Silicon Valley SJC34 (SCL2)"
	MCRTestLocationIDNum = 65 // "Digital Realty Silicon Valley SJC34 (SCL2)"
)

type MCRProviderTestSuite ProviderTestSuite

func TestMCRProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRProviderTestSuite))
}

func (suite *MCRProviderTestSuite) TestAccMegaportMCR_Basic() {
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
				  resource "megaport_mcr" "mcr" {
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
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.description", prefixFilterName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.description", prefixFilterName2),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.le", "27"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.ge", "26"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.le", "25"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_mcr.mcr",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_mcr.mcr"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "provisioning_status"},
			},
			// Update Test 1
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr" "mcr" {
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
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.#", "3"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.description", prefixFilterNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.description", prefixFilterNameNew2),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.description", prefixFilterNameNew3),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.le", "29"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.le", "26"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.0.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.0.le", "24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.1.ge", "27"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.1.le", "32"),
				),
			},
			// Update Test 2
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr" "mcr" {
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
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.description", prefixFilterNameNew4),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.#", "1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.#", "1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.ge", "28"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.le", "32"),
				),
			},
			// Update Test 3
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr" "mcr" {
					product_name             = "%s"
					port_speed               = 1000
					location_id              = data.megaport_location.test_location.id
					contract_term_months     = 12
					cost_centre              = "%s"

					prefix_filter_lists = []
				  }
				  `, MCRTestLocationIDNum, mcrNameNew2, costCentreNameNew2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrNameNew2),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreNameNew2),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.#", "0"),
				),
			},
		},
	})
}

func (suite *MCRProviderTestSuite) TestAccMegaportMCR_CostCentreRemoval() {
	mcrName := RandomTestName()
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
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 1
					cost_centre = "%s"
				}`, MCRTestLocationIDNum, mcrName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_mcr" "mcr" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 1
					cost_centre = ""
				}`, MCRTestLocationIDNum, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", ""),
				),
			},
		},
	})
}

func (suite *MCRProviderTestSuite) TestAccMegaportMCR_ContractTermUpdate() {
	mcrName := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_mcr" "mcr" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 1
				}`, MCRTestLocationIDNum, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "1"),
					waitForProvisioningStatus("megaport_mcr.mcr"),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_mcr" "mcr" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 12
				}`, MCRTestLocationIDNum, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
				),
			},
		},
	})
}

func (suite *MCRProviderTestSuite) TestAccMegaportMCRCustomASN_Basic() {
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
				  resource "megaport_mcr" "mcr" {
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
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "asn", "65000"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_mcr.mcr",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_mcr.mcr"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "provisioning_status"},
			},
			// Update Test 1
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr" "mcr" {
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
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "asn", "65000"),
				),
			},
		},
	})
}

func TestRateLimiter_SingleToken(t *testing.T) {
	rl := NewRateLimiter(5, 100*time.Millisecond)

	select {
	case <-rl.rateLimitCh:
		// Successfully got token
	default:
		t.Error("Failed to get first token")
	}
}

func TestRateLimiter_BurstLimit(t *testing.T) {
	rl := NewRateLimiter(5, 100*time.Millisecond)

	// Should get 5 tokens
	for i := 0; i < 5; i++ {
		select {
		case <-rl.rateLimitCh:
			// Successfully got token
		default:
			t.Errorf("Failed to get token %d within burst limit", i+1)
		}
	}

	// Should fail to get 6th token
	select {
	case <-rl.rateLimitCh:
		t.Error("Got token beyond burst limit")
	default:
		// Expected failure to get token
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	rl := NewRateLimiter(5, 100*time.Millisecond)

	// Use all tokens
	for i := 0; i < 5; i++ {
		select {
		case <-rl.rateLimitCh:
			// Token consumed
		default:
			t.Errorf("Failed to get token %d from initial burst", i)
		}
	}

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should get a token after refill
	select {
	case <-rl.rateLimitCh:
		// Successfully got token after refill
	default:
		t.Error("Failed to get token after refill")
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	rl := NewRateLimiter(10, 100*time.Millisecond)
	var wg sync.WaitGroup
	successCount := int32(0)

	// Launch 20 goroutines
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-rl.rateLimitCh:
				atomic.AddInt32(&successCount, 1)
			default:
				// Failed to get token
			}
		}()
	}

	wg.Wait()

	// Should have exactly 10 successes (burst limit)
	if atomic.LoadInt32(&successCount) != 10 {
		t.Errorf("Expected 10 successful token acquisitions, got %d", successCount)
	}
}

func TestRateLimiter_RateOverTime(t *testing.T) {
	rl := NewRateLimiter(5, 100*time.Millisecond)
	start := time.Now()
	count := 0

	for time.Since(start) < 450*time.Millisecond {
		select {
		case <-rl.rateLimitCh:
			count++
		default:
			// No token available
		}
		time.Sleep(10 * time.Millisecond)
	}

	expected := 9
	if count != expected {
		t.Errorf("Expected %d tokens over time period, got %d", expected, count)
	}
}
