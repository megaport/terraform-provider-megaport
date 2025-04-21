package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

type BasicVXCBasicProviderTestSuite ProviderTestSuite

func TestBasicVXCBasicProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(BasicVXCBasicProviderTestSuite))
}

func (suite *BasicVXCBasicProviderTestSuite) TestAccMegaportBasicVXC_Basic() {
	portName1 := RandomTestName()
	portName2 := RandomTestName()
	portName3 := RandomTestName()
	portName4 := RandomTestName()
	vxcName := RandomTestName()
	vxcNameNew := RandomTestName()
	costCentreName := RandomTestName()
	costCentreNew := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc" {
					id = %d
				}
					resource "megaport_port" "port_1" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
                  resource "megaport_port" "port_2" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
				  resource "megaport_port" "port_3" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
                  resource "megaport_port" "port_4" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
                  resource "megaport_vxc_basic" "vxc" {
                    product_name   = "%s"
                    rate_limit = 500
                    contract_term_months = 12
					cost_centre = "%s"

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

                    a_end = {
                        requested_product_uid = megaport_port.port_1.product_uid
						ordered_vlan = 100
						inner_vlan = 300
                    }

                    b_end = {
                        requested_product_uid = megaport_port.port_2.product_uid
						ordered_vlan = 101
						inner_vlan = 301
                    }
                  }
                  `, VXCLocationID1, portName1, portName2, portName3, portName4, vxcName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port_1", "product_name", portName1),
					resource.TestCheckResourceAttr("megaport_port.port_1", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_1", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_1", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_1", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "product_name", portName2),
					resource.TestCheckResourceAttr("megaport_port.port_2", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_2", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "product_name", portName3),
					resource.TestCheckResourceAttr("megaport_port.port_3", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_3", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "product_name", portName4),
					resource.TestCheckResourceAttr("megaport_port.port_4", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_4", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "rate_limit", "500"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.ordered_vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "b_end.ordered_vlan", "101"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "b_end.vlan", "101"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.inner_vlan", "300"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "b_end.inner_vlan", "301"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_vxc_basic.vxc", "product_uid"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc_basic.vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc_basic.vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config", "provisioning_status"},
			},
			// Update Test - Move VXC
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc" {
					id = %d
				}
					resource "megaport_port" "port_1" {
			        product_name  = "%s"
			        port_speed  = 1000
			        location_id = data.megaport_location.loc.id
			        contract_term_months        = 12
					cost_centre = "test"
					marketplace_visibility = false
			      }
			      resource "megaport_port" "port_2" {
			        product_name  = "%s"
			        port_speed  = 1000
			        location_id = data.megaport_location.loc.id
			        contract_term_months        = 12
					cost_centre = "test"
					marketplace_visibility = false
			      }
				  resource "megaport_port" "port_3" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
                  resource "megaport_port" "port_4" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
			      resource "megaport_vxc_basic" "vxc" {
			        product_name   = "%s"
			        rate_limit = 500
					contract_term_months = 12
					cost_centre = "%s"

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

			        a_end = {
			            requested_product_uid = megaport_port.port_3.product_uid
						ordered_vlan = 100
						inner_vlan = 300
			        }

			        b_end = {
			            requested_product_uid = megaport_port.port_4.product_uid
						ordered_vlan = 101
						inner_vlan = 301
			        }
			      }
			      `, VXCLocationID1, portName1, portName2, portName3, portName4, vxcName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port_1", "product_name", portName1),
					resource.TestCheckResourceAttr("megaport_port.port_1", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_1", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_1", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_1", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "product_name", portName2),
					resource.TestCheckResourceAttr("megaport_port.port_2", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_2", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "product_name", portName3),
					resource.TestCheckResourceAttr("megaport_port.port_3", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_3", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "product_name", portName4),
					resource.TestCheckResourceAttr("megaport_port.port_4", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_4", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "rate_limit", "500"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "contract_term_months", "12"),
					resource.TestCheckResourceAttrSet("megaport_vxc_basic.vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.ordered_vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "b_end.ordered_vlan", "101"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "b_end.vlan", "101"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.inner_vlan", "300"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "b_end.inner_vlan", "301"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "resource_tags.key2", "value2"),
				),
			},
			// Update Test 2 - Change Name/Cost Centre/Rate Limit/Contract Term/VLAN/Inner VLAN/Resource Tags
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc" {
					id = %d
				}
					resource "megaport_port" "port_1" {
			        product_name  = "%s"
			        port_speed  = 1000
			        location_id = data.megaport_location.loc.id
			        contract_term_months        = 12
					cost_centre = "test"
					marketplace_visibility = false
			      }
			      resource "megaport_port" "port_2" {
			        product_name  = "%s"
			        port_speed  = 1000
			        location_id = data.megaport_location.loc.id
			        contract_term_months        = 12
					cost_centre = "test"
					marketplace_visibility = false
			      }
				  resource "megaport_port" "port_3" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
                  resource "megaport_port" "port_4" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
			      resource "megaport_vxc_basic" "vxc" {
			        product_name   = "%s"
			        rate_limit = 600
					contract_term_months = 24
					cost_centre = "%s"

					resource_tags = {
						"key1updated" = "value1updated"
						"key2updated" = "value2updated"
					}

			        a_end = {
			            requested_product_uid = megaport_port.port_3.product_uid
						ordered_vlan = 200
						inner_vlan = 400
			        }

			        b_end = {
			            requested_product_uid = megaport_port.port_4.product_uid
						ordered_vlan = 201
						inner_vlan = 401
			        }
			      }
			      `, VXCLocationID1, portName1, portName2, portName3, portName4, vxcNameNew, costCentreNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port_1", "product_name", portName1),
					resource.TestCheckResourceAttr("megaport_port.port_1", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_1", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_1", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_1", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "product_name", portName2),
					resource.TestCheckResourceAttr("megaport_port.port_2", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_2", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "product_name", portName3),
					resource.TestCheckResourceAttr("megaport_port.port_3", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_3", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "product_name", portName4),
					resource.TestCheckResourceAttr("megaport_port.port_4", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_4", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "product_name", vxcNameNew),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "cost_centre", costCentreNew),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "rate_limit", "600"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "contract_term_months", "24"),
					resource.TestCheckResourceAttrSet("megaport_vxc_basic.vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.ordered_vlan", "200"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.vlan", "200"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "b_end.ordered_vlan", "201"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "b_end.vlan", "201"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.inner_vlan", "400"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "b_end.inner_vlan", "401"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "resource_tags.key2updated", "value2updated"),
				),
			},
		},
	})
}
