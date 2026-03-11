package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

const (
	SinglePortTestLocation      = "NextDC B1"
	SinglePortTestLocationIDNum = 5 // "NextDC B1"
)

type SinglePortProviderTestSuite ProviderTestSuite

func TestSinglePortProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SinglePortProviderTestSuite))
}
func (suite *SinglePortProviderTestSuite) TestAccMegaportSinglePort_Basic() {
	portName := RandomTestName()
	portNameNew := RandomTestName()
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
                resource "megaport_port" "port" {
                    product_name  = "%s"
                    port_speed  = 1000
                    cost_centre = "%s"
                    location_id = data.megaport_location.test_location.id
                    contract_term_months        = 12
                    marketplace_visibility = true
                    diversity_zone = "red"

                    resource_tags = {
                        "key1" = "value1"
                        "key2" = "value2"
                      }
                }
                
                # Test port data source with name filter
                data "megaport_ports" "test_name_filter" {
                    filter {
                        name = "name"
                        values = ["%s"]
                    }
                    depends_on = [megaport_port.port]
                }
                
                # Test port data source with port-speed filter
                data "megaport_ports" "test_speed_filter" {
                    filter {
                        name = "port-speed"
                        values = ["1000"]
                    }
                    depends_on = [megaport_port.port]
                }
                
                # Test port data source with location-id filter
                data "megaport_ports" "test_location_filter" {
                    filter {
                        name = "location-id"
                        values = ["%d"]
                    }
                    depends_on = [megaport_port.port]
                }
                
                # Test port data source with multiple filters (speed and cost-centre)
                data "megaport_ports" "test_multi_filter" {
                    filter {
                        name = "port-speed"
                        values = ["1000"]
                    }
                    filter {
                        name = "cost-centre"
                        values = ["%s"]
                    }
                    depends_on = [megaport_port.port]
                }
                
                `, SinglePortTestLocationIDNum, portName, costCentreName, portName, SinglePortTestLocationIDNum, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "product_name", portName),
					resource.TestCheckResourceAttr("megaport_port.port", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port", "marketplace_visibility", "true"),
					resource.TestCheckResourceAttr("megaport_port.port", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_port.port", "diversity_zone", "red"),
					resource.TestCheckResourceAttr("megaport_port.port", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_port.port", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "company_uid"),

					// Check data source results
					resource.TestCheckResourceAttr("data.megaport_ports.test_name_filter", "uids.#", "1"),
					resource.TestCheckResourceAttrSet("data.megaport_ports.test_speed_filter", "uids.#"),
					resource.TestCheckResourceAttrSet("data.megaport_ports.test_location_filter", "uids.#"),
					resource.TestCheckResourceAttrSet("data.megaport_ports.test_multi_filter", "uids.#"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_port.port",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_port.port"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status"},
			},
			{
				Config: providerConfig + fmt.Sprintf(`
                data "megaport_location" "test_location" {
                    id = %d
                }
                resource "megaport_port" "port" {
                    product_name  = "%s"
                    port_speed  = 1000
                    cost_centre = "%s"
                    location_id = data.megaport_location.test_location.id
                    contract_term_months        = 12
                    marketplace_visibility = false
                    diversity_zone = "red"
                    resource_tags = {
                        "key1-updated" = "value1-updated"
                        "key2-updated" = "value2-updated"
                    }
                }
                
                # Test port data source with updated name filter
                data "megaport_ports" "test_name_filter" {
                    filter {
                        name = "name"
                        values = ["%s"]
                    }
                    depends_on = [megaport_port.port]
                }
                
                # Test port data source with port-speed filter (still the same)
                data "megaport_ports" "test_speed_filter" {
                    filter {
                        name = "port-speed"
                        values = ["1000"]
                    }
                    depends_on = [megaport_port.port]
                }
                
                # Test port data source with multiple filters (speed and cost-centre)
                data "megaport_ports" "test_multi_filter" {
                    filter {
                        name = "port-speed"
                        values = ["1000"]
                    }
                    filter {
                        name = "cost-centre"
                        values = ["%s"]
                    }
                    depends_on = [megaport_port.port]
                }
                `, SinglePortTestLocationIDNum, portNameNew, costCentreNameNew, portNameNew, costCentreNameNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "product_name", portNameNew),
					resource.TestCheckResourceAttr("megaport_port.port", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_port.port", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_port.port", "diversity_zone", "red"),
					resource.TestCheckResourceAttr("megaport_port.port", "resource_tags.key1-updated", "value1-updated"),
					resource.TestCheckResourceAttr("megaport_port.port", "resource_tags.key2-updated", "value2-updated"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "company_uid"),

					// Check updated data source results
					resource.TestCheckResourceAttr("data.megaport_ports.test_name_filter", "uids.#", "1"),
					resource.TestCheckResourceAttrSet("data.megaport_ports.test_speed_filter", "uids.#"),
					resource.TestCheckResourceAttrSet("data.megaport_ports.test_multi_filter", "uids.#"),
				),
			},
		},
	})
}

func (suite *SinglePortProviderTestSuite) TestAccMegaportSinglePort_CostCentreRemoval() {
	portName := RandomTestName()
	costCentreName := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_port" "port" {
					product_name  = "%s"
					port_speed  = 1000
					cost_centre = "%s"
					location_id = data.megaport_location.test_location.id
					contract_term_months = 1
					marketplace_visibility = false
				}`, SinglePortTestLocationIDNum, portName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "cost_centre", costCentreName),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_port" "port" {
					product_name  = "%s"
					port_speed  = 1000
					cost_centre = ""
					location_id = data.megaport_location.test_location.id
					contract_term_months = 1
					marketplace_visibility = false
				}`, SinglePortTestLocationIDNum, portName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "cost_centre", ""),
				),
			},
		},
	})
}

func (suite *SinglePortProviderTestSuite) TestAccMegaportSinglePort_ContractTermUpdate() {
	portName := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_port" "port" {
					product_name  = "%s"
					port_speed  = 1000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 1
					marketplace_visibility = false
				}`, SinglePortTestLocationIDNum, portName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "contract_term_months", "1"),
					waitForProvisioningStatus("megaport_port.port"),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_port" "port" {
					product_name  = "%s"
					port_speed  = 1000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 12
					marketplace_visibility = false
				}`, SinglePortTestLocationIDNum, portName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "contract_term_months", "12"),
				),
			},
		},
	})
}
