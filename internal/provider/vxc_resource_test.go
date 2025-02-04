package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

type VXCBasicProviderTestSuite ProviderTestSuite
type VXCCSPProviderTestSuite ProviderTestSuite
type VXCMVEProviderTestSuite ProviderTestSuite

const (
	VXCLocationOne   = "NextDC M1"
	VXCLocationTwo   = "Global Switch Sydney West"
	VXCLocationThree = "5GN Melbourne Data Centre (MDC)"
)

func TestVXCBasicProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(VXCBasicProviderTestSuite))
}

func TestVXCCSPProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(VXCCSPProviderTestSuite))
}

func TestVXCMVEProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(VXCMVEProviderTestSuite))
}

func (suite *VXCBasicProviderTestSuite) TestAccMegaportVXC_Basic() {
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
					name = "%s"
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
                  resource "megaport_vxc" "vxc" {
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
                  `, VXCLocationOne, portName1, portName2, portName3, portName4, vxcName, costCentreName),
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
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "rate_limit", "500"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.ordered_vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.ordered_vlan", "101"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.vlan", "101"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.inner_vlan", "300"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.inner_vlan", "301"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status"},
			},
			// Update Test - Move VXC
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc" {
					name = "%s"
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
			      resource "megaport_vxc" "vxc" {
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
			      `, VXCLocationOne, portName1, portName2, portName3, portName4, vxcName, costCentreName),
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
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "rate_limit", "500"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "contract_term_months", "12"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.ordered_vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.ordered_vlan", "101"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.vlan", "101"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.inner_vlan", "300"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.inner_vlan", "301"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "resource_tags.key2", "value2"),
				),
			},
			// Update Test 2 - Change Name/Cost Centre/Rate Limit/Contract Term/VLAN/Inner VLAN/Resource Tags
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc" {
					name = "%s"
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
			      resource "megaport_vxc" "vxc" {
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
			      `, VXCLocationOne, portName1, portName2, portName3, portName4, vxcNameNew, costCentreNew),
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
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcNameNew),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "cost_centre", costCentreNew),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "rate_limit", "600"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "contract_term_months", "24"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.ordered_vlan", "200"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.vlan", "200"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.ordered_vlan", "201"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.vlan", "201"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.inner_vlan", "400"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.inner_vlan", "401"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "resource_tags.key2updated", "value2updated"),
				),
			},
		},
	})
}

func (suite *VXCBasicProviderTestSuite) TestAccMegaportVXC_BasicUntagVLAN() {
	portName1 := RandomTestName()
	portName2 := RandomTestName()
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
					name = "%s"
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
                  resource "megaport_vxc" "vxc" {
                    product_name   = "%s"
                    rate_limit = 500
                    contract_term_months = 12
					cost_centre = "%s"

                    a_end = {
                        requested_product_uid = megaport_port.port_1.product_uid
						ordered_vlan = 100
                    }

                    b_end = {
                        requested_product_uid = megaport_port.port_2.product_uid
						ordered_vlan = 101
                    }
                  }
                  `, VXCLocationOne, portName1, portName2, vxcName, costCentreName),
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
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "rate_limit", "500"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.ordered_vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.ordered_vlan", "101"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.vlan", "101"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "a_end_partner_config", "b_end_partner_config", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status"},
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc" {
					name = "%s"
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
                  resource "megaport_vxc" "vxc" {
                    product_name   = "%s"
                    rate_limit = 500
                    contract_term_months = 12
					cost_centre = "%s"

                    a_end = {
                        requested_product_uid = megaport_port.port_1.product_uid
                    }

                    b_end = {
                        requested_product_uid = megaport_port.port_2.product_uid
                    }
                  }
                  `, VXCLocationOne, portName1, portName2, vxcName, costCentreName),
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
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "rate_limit", "500"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.vlan", "101"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
				),
			},
			// Update Test - Change VXC Name, Untag A-End and B-End VLAN
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc" {
					name = "%s"
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
			      resource "megaport_vxc" "vxc" {
			        product_name   = "%s"
			        rate_limit = 500
					contract_term_months = 12
					cost_centre = "%s"

			        a_end = {
			            requested_product_uid = megaport_port.port_1.product_uid
						ordered_vlan = -1
			        }

			        b_end = {
			            requested_product_uid = megaport_port.port_2.product_uid
						ordered_vlan = -1
			        }
			      }
			      `, VXCLocationOne, portName1, portName2, vxcNameNew, costCentreNew),
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
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcNameNew),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "cost_centre", costCentreNew),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "rate_limit", "500"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "contract_term_months", "12"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.ordered_vlan", "-1"),
					resource.TestCheckNoResourceAttr("megaport_vxc.vxc", "a_end.vlan"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.ordered_vlan", "-1"),
					resource.TestCheckNoResourceAttr("megaport_vxc.vxc", "b_end.vlan"),
				),
			},
		},
	})
}

func (suite *VXCCSPProviderTestSuite) TestUpdateVLAN() {
	portName := RandomTestName()
	costCentreName := RandomTestName()
	awsVXCName := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					name = "%s"
				  }

				  data "megaport_location" "loc2" {
					name = "%s"
				  }

				  data "megaport_partner" "aws_port" {
					connect_type = "AWS"
					company_name = "AWS"
					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
					location_id  = data.megaport_location.loc1.id
				  }

				  resource "megaport_port" "port" {
					product_name            = "%s"
					port_speed              = 1000
					location_id             = data.megaport_location.loc2.id
					contract_term_months    = 12
					marketplace_visibility  = true
					cost_centre = "%s"
				  }

				  resource "megaport_vxc" "aws_vxc" {
					product_name            = "%s"
					rate_limit              = 1000
					contract_term_months    = 1

					a_end = {
					  requested_product_uid = megaport_port.port.product_uid
					  ordered_vlan = 191
					}

					b_end = {
					  requested_product_uid = data.megaport_partner.aws_port.product_uid
					}

					b_end_partner_config = {
					  partner = "aws"
					  aws_config = {
						name          = "%s"
						asn           = 64550
						type          = "private"
						connect_type  = "AWSHC"
						amazon_asn    = 64551
						owner_account = "123456789012"
					  }
					}
				  }
                  `, VXCLocationOne, VXCLocationTwo, portName, costCentreName, awsVXCName, awsVXCName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "b_end_partner_config.aws_config.name", awsVXCName),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.ordered_vlan", "191"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.vlan", "191"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.aws_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.aws_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end_partner_config", "b_end_partner_config", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid"},
			},
			// Update Test - Change A-End VLAN
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					name = "%s"
				  }

				  data "megaport_location" "loc2" {
					name = "%s"
				  }
				  data "megaport_partner" "aws_port" {
					connect_type = "AWS"
					company_name = "AWS"
					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
					location_id  = data.megaport_location.loc1.id
				  }

				  resource "megaport_port" "port" {
					product_name            = "%s"
					port_speed              = 1000
					location_id             = data.megaport_location.loc2.id
					contract_term_months    = 12
					marketplace_visibility  = true
					cost_centre = "%s"
				  }

				  resource "megaport_vxc" "aws_vxc" {
					product_name            = "%s"
					rate_limit              = 1000
					contract_term_months    = 1

					a_end = {
					  requested_product_uid = megaport_port.port.product_uid
					  ordered_vlan = 195
					}

					b_end = {
					  requested_product_uid = data.megaport_partner.aws_port.product_uid
					}

					b_end_partner_config = {
					  partner = "aws"
					  aws_config = {
						name          = "%s"
						asn           = 64550
						type          = "private"
						connect_type  = "AWSHC"
						amazon_asn    = 64551
						owner_account = "123456789012"
					  }
					}
				  }
                  `, VXCLocationOne, VXCLocationTwo, portName, costCentreName, awsVXCName, awsVXCName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "b_end_partner_config.aws_config.name", awsVXCName),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.ordered_vlan", "195"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.vlan", "195"),
				),
			},
			// Update Test - Untag VLAN
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					name = "%s"
				  }

				  data "megaport_location" "loc2" {
					name = "%s"
				  }
				  data "megaport_partner" "aws_port" {
					connect_type = "AWS"
					company_name = "AWS"
					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
					location_id  = data.megaport_location.loc1.id
				  }

				  resource "megaport_port" "port" {
					product_name            = "%s"
					port_speed              = 1000
					location_id             = data.megaport_location.loc2.id
					contract_term_months    = 12
					marketplace_visibility  = true
					cost_centre = "%s"
				  }

				  resource "megaport_vxc" "aws_vxc" {
					product_name            = "%s"
					rate_limit              = 1000
					contract_term_months    = 1

					a_end = {
					  requested_product_uid = megaport_port.port.product_uid
					  ordered_vlan = -1
					}

					b_end = {
					  requested_product_uid = data.megaport_partner.aws_port.product_uid
					}

					b_end_partner_config = {
					  partner = "aws"
					  aws_config = {
						name          = "%s"
						asn           = 64550
						type          = "private"
						connect_type  = "AWSHC"
						amazon_asn    = 64551
						owner_account = "123456789012"
					  }
					}
				  }
                  `, VXCLocationOne, VXCLocationTwo, portName, costCentreName, awsVXCName, awsVXCName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "b_end_partner_config.aws_config.name", awsVXCName),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.ordered_vlan", "-1"),
					resource.TestCheckNoResourceAttr("megaport_vxc.aws_vxc", "a_end.vlan"),
				),
			},
		},
	})
}

func (suite *VXCCSPProviderTestSuite) TestAccMegaportMCRVXCWithCSPs_Basic() {
	mcrName := RandomTestName()
	vxcName1 := RandomTestName()
	vxcName2 := RandomTestName()
	vxcName3 := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
                    name    = "%s"
                  }

                  data "megaport_location" "loc2" {
                    name = "%s"
                  }

                  data "megaport_partner" "aws_port" {
                    connect_type = "AWS"
                    company_name = "AWS"
                    product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
                    location_id  = data.megaport_location.loc2.id
                  }

                  resource "megaport_mcr" "mcr" {
                    product_name    = "%s"
                    location_id = data.megaport_location.loc1.id
                    contract_term_months = 1
                    port_speed = 5000
                    asn = 64555
                  }

                  resource "megaport_vxc" "aws_vxc" {
                    product_name   = "%s"
                    rate_limit = 1000
                    contract_term_months = 1

                    a_end = {
                      requested_product_uid = megaport_mcr.mcr.product_uid
                      ordered_vlan = 2191
                    }

                    b_end = {
                        requested_product_uid = data.megaport_partner.aws_port.product_uid
                    }

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

                    b_end_partner_config = {
                        partner = "aws"
                        aws_config = {
                            name = "%s"
                            asn = 64550
                            type = "private"
                            connect_type = "AWS"
                            amazon_asn = 64551
                            owner_account = "123456789012"
                        }
                    }
                  }

                  resource "megaport_vxc" "gcp_vxc" {
                    product_name   = "%s"
                    rate_limit = 1000
                    contract_term_months = 1

                    a_end = {
                      requested_product_uid = megaport_mcr.mcr.product_uid
                      ordered_vlan = 182
                    }

                    b_end = {}

                    b_end_partner_config = {
                        partner = "google"
                        google_config = {
                            pairing_key = "7e51371e-72a3-40b5-b844-2e3efefaee59/australia-southeast1/2"
                        }
                    }
                  }

                  resource "megaport_vxc" "azure_vxc" {
                    product_name   = "%s"
                    rate_limit = 200
                    contract_term_months = 1

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

                    a_end = {
                      requested_product_uid = megaport_mcr.mcr.product_uid
                      ordered_vlan = 0
                    }

                    b_end = {}

                    b_end_partner_config = {
                        partner = "azure"
                        azure_config = {
							port_choice = "primary"
                            service_key = "197d927b-90bc-4b1b-bffd-fca17a7ec735"
                        }
                    }
                  }
                  `, VXCLocationOne, VXCLocationTwo, mcrName, vxcName1, vxcName1, vxcName2, vxcName3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "b_end_partner_config.aws_config.name", vxcName1),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.ordered_vlan", "2191"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_vxc.azure_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.azure_vxc", "a_end.ordered_vlan", "0"),
					resource.TestCheckResourceAttr("megaport_vxc.azure_vxc", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_vxc.azure_vxc", "resource_tags.key2", "value2"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.aws_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.aws_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config"},
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.azure_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.azure_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config"},
			},
		},
	})
}

func (suite *VXCCSPProviderTestSuite) TestAccMegaportMCRVXCWithBGP_Basic() {
	mcrName := RandomTestName()
	vxcName1 := RandomTestName()
	prefixFilterListName := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					name = "%s"
				  }

				  data "megaport_location" "loc2" {
					name = "%s"
				  }

				  data "megaport_partner" "aws_port" {
					connect_type = "AWS"
					company_name = "AWS"
					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
					location_id  = data.megaport_location.loc2.id
				  }

				  resource "megaport_mcr" "mcr" {
					product_name            = "%s"
					location_id             = data.megaport_location.loc1.id
					contract_term_months    = 1
					port_speed              = 5000
					asn                     = 64555

					prefix_filter_lists = [{
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
						  ge      = 24
						  le      = 24
						}
					  ]
					}]
				  }

				  resource "megaport_vxc" "aws_vxc" {
					product_name           = "%s"
					rate_limit             = 1000
					contract_term_months   = 1

					a_end = {
                      requested_product_uid = megaport_mcr.mcr.product_uid
					  ordered_vlan = 0
					}

					a_end_partner_config = {
					  partner = "vrouter"
					  vrouter_config = {
						interfaces = [{
							ip_addresses     = ["10.0.0.1/30"]
							nat_ip_addresses = ["10.0.0.1"]
						  bfd = {
							tx_interval   = 500
							rx_interval   = 400
							multiplier    = 5
						  }
						  bgp_connections = [
							{
							  peer_asn          = 64512
							  local_ip_address  = "10.0.0.1"
							  peer_ip_address   = "10.0.0.2"
							  password          = "notARealPassword"
							  shutdown          = false
							  description       = "BGP Connection 1"
							  med_in            = 100
							  med_out           = 100
							  bfd_enabled       = true
							  export_policy     = "deny"
							  permit_export_to = ["10.0.1.2"]
							  import_whitelist = "%s"
							  as_path_prepend_count = 4
							}
						  ]
						}]
					  }
					}

					b_end = {
					  requested_product_uid = data.megaport_partner.aws_port.product_uid
					}

					b_end_partner_config = {
					  partner = "aws"
					  aws_config = {
						name            = "%s"
						asn             = 64550
						type            = "private"
						connect_type    = "AWSHC"
						amazon_asn      = 64551
						owner_account   = "684021030471"
					  }
					}

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}
				  }
                  `, VXCLocationOne, VXCLocationTwo, mcrName, prefixFilterListName, vxcName1, prefixFilterListName, vxcName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "b_end_partner_config.aws_config.name", vxcName1),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.ordered_vlan", "0"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "resource_tags.key2", "value2"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.aws_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.aws_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config"},
			},
			// UPDATE Test - Change BGP Connection in Partner Config
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					name = "%s"
				  }

				  data "megaport_location" "loc2" {
					name = "%s"
				  }

				  data "megaport_partner" "aws_port" {
					connect_type = "AWS"
					company_name = "AWS"
					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
					location_id  = data.megaport_location.loc2.id
				  }

				  resource "megaport_mcr" "mcr" {
					product_name            = "%s"
					location_id             = data.megaport_location.loc1.id
					contract_term_months    = 1
					port_speed              = 5000
					asn                     = 64555

					prefix_filter_lists = [{
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
						  ge      = 24
						  le      = 24
						}
					  ]
					}]
				  }

				  resource "megaport_vxc" "aws_vxc" {
					product_name           = "%s"
					rate_limit             = 1000
					contract_term_months   = 1

					a_end = {
                      requested_product_uid = megaport_mcr.mcr.product_uid
					  ordered_vlan = 0
					}

					a_end_partner_config = {
					  partner = "vrouter"
					  vrouter_config = {
						interfaces = [{
							ip_addresses     = ["10.0.0.1/30"]
							nat_ip_addresses = ["10.0.0.1"]
						  bfd = {
							tx_interval   = 500
							rx_interval   = 400
							multiplier    = 5
						  }
						  bgp_connections = [
							{
							  peer_asn          = 64512
							  local_ip_address  = "10.0.0.1"
							  peer_ip_address   = "10.0.0.2"
							  password          = "notARealPassword"
							  shutdown          = false
							  description       = "BGP Connection 1 updated"
							  med_in            = 100
							  med_out           = 100
							  bfd_enabled       = true
							  export_policy     = "deny"
							  permit_export_to = ["10.0.1.2"]
							  import_whitelist = "%s"
							  as_path_prepend_count = 4
							}
						  ]
						}]
					  }
					}

					b_end = {
					  requested_product_uid = data.megaport_partner.aws_port.product_uid
					}

					b_end_partner_config = {
					  partner = "aws"
					  aws_config = {
						name            = "%s"
						asn             = 64550
						type            = "private"
						connect_type    = "AWSHC"
						amazon_asn      = 64551
						owner_account   = "684021030471"
					  }
					}

					resource_tags = {
						"key1updated" = "value1updated"
						"key2updated" = "value2updated"
					}
				  }
                  `, VXCLocationOne, VXCLocationTwo, mcrName, prefixFilterListName, vxcName1, prefixFilterListName, vxcName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "b_end_partner_config.aws_config.name", vxcName1),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.ordered_vlan", "0"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "resource_tags.key2updated", "value2updated"),
				),
			},
		},
	})
}

func (suite *VXCCSPProviderTestSuite) TestFullEcosystem() {
	portName := RandomTestName()
	lagPortName := RandomTestName()
	mcrName := RandomTestName()
	portVXCName := RandomTestName()
	costCentreName := RandomTestName()
	mcrVXCName := RandomTestName()
	awsVXCName := RandomTestName()
	gcpVXCName := RandomTestName()
	azureVXCName := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					name = "%s"
				  }

				  data "megaport_location" "loc2" {
					name = "%s"
				  }

				  data "megaport_location" "loc3" {
					name = "%s"
				  }

				  data "megaport_partner" "aws_port" {
					connect_type = "AWS"
					company_name = "AWS"
					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
					location_id  = data.megaport_location.loc2.id
				  }

				  resource "megaport_lag_port" "lag_port" {
			        product_name  = "%s"
					cost_centre = "%s"
			        port_speed  = 10000
			        location_id = data.megaport_location.loc1.id
			        contract_term_months        = 12
					marketplace_visibility = false
                    lag_count = 1
			      }

				  resource "megaport_port" "port" {
					product_name            = "%s"
					port_speed              = 1000
					location_id             = data.megaport_location.loc2.id
					contract_term_months    = 12
					marketplace_visibility  = true
					cost_centre = "%s"
				  }

				  resource "megaport_mcr" "mcr" {
					product_name            = "%s"
					port_speed              = 2500
					location_id             = data.megaport_location.loc1.id
					contract_term_months    = 1
					asn                      = 64555
				  }

				  resource "megaport_vxc" "port_vxc" {
					product_name           = "%s"
					rate_limit             = 1000
					contract_term_months   = 12

					a_end = {
					  requested_product_uid = megaport_port.port.product_uid
					}

					b_end = {
					  requested_product_uid = megaport_lag_port.lag_port.product_uid
					}

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}
				  }

				  resource "megaport_vxc" "mcr_vxc" {
					product_name           = "%s"
					rate_limit             = 1000
					contract_term_months   = 12

					a_end = {
					  requested_product_uid = megaport_port.port.product_uid
					  ordered_vlan = 181
					}

					b_end = {
					  requested_product_uid = megaport_mcr.mcr.product_uid
					  ordered_vlan = 181
					}

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}
				  }

				  resource "megaport_vxc" "aws_vxc" {
					product_name            = "%s"
					rate_limit              = 1000
					contract_term_months    = 1

					a_end = {
					  requested_product_uid = megaport_mcr.mcr.product_uid
					  ordered_vlan = 191
					}

					b_end = {
					  requested_product_uid = data.megaport_partner.aws_port.product_uid
					}

					b_end_partner_config = {
					  partner = "aws"
					  aws_config = {
						name          = "%s"
						asn           = 64550
						type          = "private"
						connect_type  = "AWS"
						amazon_asn    = 64551
						owner_account = "123456789012"
					  }
					}
				  }

				  resource "megaport_vxc" "gcp_vxc" {
					product_name            = "%s"
					rate_limit              = 1000
					contract_term_months    = 12

					a_end = {
					  requested_product_uid = megaport_mcr.mcr.product_uid
					  ordered_vlan = 182
					}

					b_end = {}

					b_end_partner_config = {
					  partner = "google"
					  google_config = {
						pairing_key = "7e51371e-72a3-40b5-b844-2e3efefaee59/australia-southeast1/2"
					  }
					}
				  }

				  resource "megaport_vxc" "azure_vxc" {
					product_name            = "%s"
					rate_limit              = 200
					contract_term_months    = 12

					a_end = {
					  requested_product_uid = megaport_mcr.mcr.product_uid
					  ordered_vlan = 0
					}

					b_end = {}

					b_end_partner_config = {
					  partner = "azure"
					  azure_config = {
					    port_choice = "primary"
						service_key = "197d927b-90bc-4b1b-bffd-fca17a7ec735"
					  }
					}
				  }
                  `, VXCLocationOne, VXCLocationTwo, VXCLocationThree, lagPortName, costCentreName, portName, costCentreName, mcrName, portVXCName, mcrVXCName, awsVXCName, awsVXCName, gcpVXCName, azureVXCName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "b_end_partner_config.aws_config.name", awsVXCName),
					resource.TestCheckResourceAttr("megaport_vxc.port_vxc", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_vxc.port_vxc", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttr("megaport_vxc.mcr_vxc", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_vxc.mcr_vxc", "resource_tags.key2", "value2"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.aws_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.aws_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config"},
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.gcp_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.gcp_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config"},
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.azure_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.azure_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config"},
			},
		},
	})
}

func (suite *VXCMVEProviderTestSuite) TestMVE_TransitVXC() {
	portName := RandomTestName()
	costCentreName := RandomTestName()
	mveName := RandomTestName()
	transitVXCName := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					name = "%s"
				  }

				  data "megaport_location" "loc2" {
					name = "%s"
				  }

				  resource "megaport_port" "port" {
					product_name           = "%s"
					port_speed             = 1000
					location_id            = data.megaport_location.loc1.id
					contract_term_months   = 12
					marketplace_visibility = true
					cost_centre            = "%s"
				  }

				  data "megaport_partner" "internet_port" {
					connect_type  = "TRANSIT"
					company_name  = "Networks"
					product_name  = "Megaport Internet"
					location_id   = data.megaport_location.loc2.id
				  }

				  resource "megaport_mve" "mve" {
					product_name           = "%s"
					location_id            = data.megaport_location.loc1.id
					contract_term_months   = 1

					vnics = [
					  {
						description = "Data Plane"
					  },
					  {
						description = "Management Plane"
					  },
					  {
						description = "Control Plane"
					  }
					]

					vendor_config = {
					  vendor        = "aruba"
					  product_size  = "MEDIUM"
					  image_id      = 23
					  account_name  = "%s"
					  account_key   = "%s"
					  system_tag    = "Preconfiguration-aruba-test-1"
					}
				  }

				  resource "megaport_vxc" "transit_vxc" {
					product_name         = "%s"
					rate_limit           = 100
					contract_term_months = 1

					a_end = {
					  requested_product_uid = megaport_mve.mve.product_uid
					  vnic_index            = 2
					}

					b_end = {
					  requested_product_uid = data.megaport_partner.internet_port.product_uid
					}

					b_end_partner_config = {
					  partner = "transit"
					}
				  }
                  `, VXCLocationOne, VXCLocationTwo, portName, costCentreName, mveName, mveName, mveName, transitVXCName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.transit_vxc", "product_uid"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.transit_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.transit_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config"},
			},
		},
	})
}

func (suite *VXCCSPProviderTestSuite) TestMVE_TransitVXCAWS() {
	portName := RandomTestName()
	portCostCentreName := RandomTestName()
	portCostCentreNameNew := RandomTestName()
	mveName := RandomTestName()
	transitVXCName := RandomTestName()
	transitVXCCostCentreName := RandomTestName()
	transitVXCCostCentreNameNew := RandomTestName()
	portVXCName := RandomTestName()
	portVXCAEndInnerVLAN := 95
	portVXCBEndInnerVLAN := 96
	portVXCAEndInnerVLANNew := 97
	portVXCBEndInnerVLANNew := 98
	portVXCCostCentreName := RandomTestName()
	portVXCCostCentreNameNew := RandomTestName()
	awsVXCName := RandomTestName()
	awsVXCAEndInnerVLAN := 90
	awsVXCAEndInnerVLANNew := 92
	awsVXCCostCentreName := RandomTestName()
	awsVXCCostCentreNameNew := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					name = "%s"
				  }

				  data "megaport_location" "loc2" {
					name = "%s"
				  }

				  resource "megaport_port" "port" {
					product_name           = "%s"
					port_speed             = 1000
					location_id            = data.megaport_location.loc1.id
					contract_term_months   = 12
					marketplace_visibility = true
					cost_centre            = "%s"
				  }

				  data "megaport_partner" "internet_port" {
					connect_type  = "TRANSIT"
					company_name  = "Networks"
					product_name  = "Megaport Internet"
					location_id   = data.megaport_location.loc2.id
				  }

				   data "megaport_partner" "aws_port" {
					connect_type = "AWS"
					company_name = "AWS"
					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
					location_id  = data.megaport_location.loc2.id
				  }

				  resource "megaport_mve" "mve" {
					product_name           = "%s"
					location_id            = data.megaport_location.loc1.id
					contract_term_months   = 1

					vnics = [
					  {
						description = "Data Plane"
					  },
					  {
						description = "Management Plane"
					  },
					  {
						description = "Control Plane"
					  }
					]

					vendor_config = {
					  vendor        = "aruba"
					  product_size  = "MEDIUM"
					  image_id      = 23
					  account_name  = "%s"
					  account_key   = "%s"
					  system_tag    = "Preconfiguration-aruba-test-1"
					}
				  }

				  resource "megaport_vxc" "transit_vxc" {
					product_name         = "%s"
					rate_limit           = 100
					contract_term_months = 1
					cost_centre = "%s"

					a_end = {
					  requested_product_uid = megaport_mve.mve.product_uid
					  vnic_index            = 0
					}

					b_end = {
					  requested_product_uid = data.megaport_partner.internet_port.product_uid
					}

					b_end_partner_config = {
					  partner = "transit"
					}
				  }

				  resource "megaport_vxc" "port_vxc" {
					product_name         = "%s"
					rate_limit           = 100
					contract_term_months = 1
					cost_centre = "%s"

					a_end = {
					  requested_product_uid = megaport_mve.mve.product_uid
					  vnic_index            = 0
					  inner_vlan = %d
					}

					b_end = {
					  requested_product_uid = megaport_port.port.product_uid
					  inner_vlan = %d
					}
				  }

				  resource "megaport_vxc" "aws_vxc" {
					product_name            = "%s"
					rate_limit              = 100
					contract_term_months    = 1
					cost_centre = "%s"

					a_end = {
						requested_product_uid = megaport_mve.mve.product_uid
						inner_vlan            = %d
						vnic_index            = 0
					}

					b_end = {
						requested_product_uid = data.megaport_partner.aws_port.product_uid
					}

					b_end_partner_config = {
					  partner = "aws"
					  aws_config = {
						name          = "%s"
						asn           = 65121
						type          = "private"
						connect_type  = "AWSHC"
						amazon_asn    = 64512
						owner_account = "123456789012"
					  }
					}
				  }
                  `, VXCLocationOne, VXCLocationTwo, portName, portCostCentreName, mveName, mveName, mveName, transitVXCName, transitVXCCostCentreName, portVXCName, portVXCCostCentreName, portVXCAEndInnerVLAN, portVXCBEndInnerVLAN, awsVXCName, awsVXCCostCentreName, awsVXCAEndInnerVLAN, awsVXCName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.transit_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.port_vxc", "a_end.inner_vlan", fmt.Sprintf("%d", portVXCAEndInnerVLAN)),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.inner_vlan", fmt.Sprintf("%d", awsVXCAEndInnerVLAN)),
					resource.TestCheckResourceAttr("megaport_vxc.port_vxc", "b_end.inner_vlan", fmt.Sprintf("%d", portVXCBEndInnerVLAN)),
					resource.TestCheckNoResourceAttr("megaport_vxc.aws_vxc", "b_end.inner_vlan"),
					resource.TestCheckResourceAttr("megaport_vxc.port_vxc", "cost_centre", portVXCCostCentreName),
					resource.TestCheckResourceAttr("megaport_vxc.port_vxc", "cost_centre", portVXCCostCentreName),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "cost_centre", awsVXCCostCentreName),
					resource.TestCheckNoResourceAttr("megaport_vxc.transit_vxc", "a_end.inner_vlan"),
					resource.TestCheckNoResourceAttr("megaport_vxc.transit_vxc", "b_end.inner_vlan"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.aws_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.aws_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config"},
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.port_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.port_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config"},
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.transit_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.transit_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config"},
			},
			// UPDATE
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					name = "%s"
				  }

				  data "megaport_location" "loc2" {
					name = "%s"
				  }

				  resource "megaport_port" "port" {
					product_name           = "%s"
					port_speed             = 1000
					location_id            = data.megaport_location.loc1.id
					contract_term_months   = 12
					marketplace_visibility = true
					cost_centre            = "%s"
				  }

				  data "megaport_partner" "internet_port" {
					connect_type  = "TRANSIT"
					company_name  = "Networks"
					product_name  = "Megaport Internet"
					location_id   = data.megaport_location.loc2.id
				  }

				   data "megaport_partner" "aws_port" {
					connect_type = "AWS"
					company_name = "AWS"
					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
					location_id  = data.megaport_location.loc2.id
				  }

				  resource "megaport_mve" "mve" {
					product_name           = "%s"
					location_id            = data.megaport_location.loc1.id
					contract_term_months   = 1

					vnics = [
					  {
						description = "Data Plane"
					  },
					  {
						description = "Management Plane"
					  },
					  {
						description = "Control Plane"
					  }
					]

					vendor_config = {
					  vendor        = "aruba"
					  product_size  = "MEDIUM"
					  image_id      = 23
					  account_name  = "%s"
					  account_key   = "%s"
					  system_tag    = "Preconfiguration-aruba-test-1"
					}
				  }

				  resource "megaport_vxc" "transit_vxc" {
					product_name         = "%s"
					rate_limit           = 100
					contract_term_months = 1
					cost_centre = "%s"

					a_end = {
					  requested_product_uid = megaport_mve.mve.product_uid
					  vnic_index            = 0
					}

					b_end = {
					  requested_product_uid = data.megaport_partner.internet_port.product_uid
					}

					b_end_partner_config = {
					  partner = "transit"
					}
				  }

				  resource "megaport_vxc" "port_vxc" {
					product_name         = "%s"
					rate_limit           = 100
					contract_term_months = 1
					cost_centre = "%s"

					a_end = {
					  requested_product_uid = megaport_mve.mve.product_uid
					  vnic_index            = 0
					  inner_vlan = %d
					}

					b_end = {
					  requested_product_uid = megaport_port.port.product_uid
					  inner_vlan = %d
					}
				  }

				  resource "megaport_vxc" "aws_vxc" {
					product_name            = "%s"
					rate_limit              = 100
					contract_term_months    = 1
					cost_centre = "%s"

					a_end = {
						requested_product_uid = megaport_mve.mve.product_uid
						inner_vlan            = %d
						vnic_index            = 0
					}

					b_end = {
						requested_product_uid = data.megaport_partner.aws_port.product_uid
					}

					b_end_partner_config = {
					  partner = "aws"
					  aws_config = {
						name          = "%s"
						asn           = 65121
						type          = "private"
						connect_type  = "AWSHC"
						amazon_asn    = 64512
						owner_account = "123456789012"
					  }
					}
				  }
                  `, VXCLocationOne, VXCLocationTwo, portName, portCostCentreNameNew, mveName, mveName, mveName, transitVXCName, transitVXCCostCentreNameNew, portVXCName, portVXCCostCentreNameNew, portVXCAEndInnerVLANNew, portVXCBEndInnerVLANNew, awsVXCName, awsVXCCostCentreNameNew, awsVXCAEndInnerVLANNew, awsVXCName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.transit_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.inner_vlan", fmt.Sprintf("%d", awsVXCAEndInnerVLANNew)),
					resource.TestCheckResourceAttr("megaport_vxc.port_vxc", "a_end.inner_vlan", fmt.Sprintf("%d", portVXCAEndInnerVLANNew)),
					resource.TestCheckResourceAttr("megaport_vxc.port_vxc", "b_end.inner_vlan", fmt.Sprintf("%d", portVXCBEndInnerVLANNew)),
					resource.TestCheckNoResourceAttr("megaport_vxc.transit_vxc", "a_end.inner_vlan"),
					resource.TestCheckNoResourceAttr("megaport_vxc.transit_vxc", "b_end.inner_vlan"),
					resource.TestCheckNoResourceAttr("megaport_vxc.aws_vxc", "b_end.inner_vlan"),
					resource.TestCheckResourceAttr("megaport_vxc.port_vxc", "cost_centre", portVXCCostCentreNameNew),
					resource.TestCheckResourceAttr("megaport_vxc.port_vxc", "cost_centre", portVXCCostCentreNameNew),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "cost_centre", awsVXCCostCentreNameNew),
					resource.TestCheckResourceAttr("megaport_vxc.transit_vxc", "cost_centre", transitVXCCostCentreNameNew),
				),
			},
		},
	})
}

func (suite *VXCCSPProviderTestSuite) TestMVE_AWS_VXC() {
	portName := RandomTestName()
	costCentreName := RandomTestName()
	mveName := RandomTestName()
	awsVXCName := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					name = "%s"
				  }

				  data "megaport_location" "syd_gs" {
					name = "Global Switch Sydney West"
				  }

				  data "megaport_partner" "aws_port" {
					connect_type = "AWS"
					company_name = "AWS"
					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
					location_id  = data.megaport_location.syd_gs.id
				  }

				  resource "megaport_port" "port" {
					product_name            = "%s"
					port_speed              = 1000
					location_id             = data.megaport_location.loc1.id
					contract_term_months    = 12
					marketplace_visibility  = true
					cost_centre = "%s"
				  }

				resource "megaport_mve" "mve" {
                    product_name  = "%s"
                    location_id = data.megaport_location.loc1.id
                    contract_term_months        = 1

					vnics = [
						{
							description = "to_aws"
						},
						{
								description = "to_port"
						},
					]

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "MEDIUM"
                        image_id = 23
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
                    }
                  }

				  resource "megaport_vxc" "aws_vxc" {
					product_name            = "%s"
					rate_limit              = 100
					contract_term_months    = 1

					a_end = {
						requested_product_uid = megaport_mve.mve.product_uid
						inner_vlan            = 100
						vnic_index            = 0
					}

					b_end = {
						requested_product_uid = data.megaport_partner.aws_port.product_uid
					}

					b_end_partner_config = {
					  partner = "aws"
					  aws_config = {
						name          = "%s"
						asn           = 65121
						type          = "private"
						connect_type  = "AWSHC"
						amazon_asn    = 64512
						owner_account = "123456789012"
					  }
					}

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}
				  }

                  `, VXCLocationOne, portName, costCentreName, mveName, mveName, mveName, awsVXCName, awsVXCName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "b_end_partner_config.aws_config.name", awsVXCName),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.inner_vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "resource_tags.key2", "value2"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.aws_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.aws_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config"},
			},
			// Update
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					name = "%s"
				  }

				  data "megaport_location" "syd_gs" {
					name = "Global Switch Sydney West"
				  }

				  data "megaport_partner" "aws_port" {
					connect_type = "AWS"
					company_name = "AWS"
					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
					location_id  = data.megaport_location.syd_gs.id
				  }

				  resource "megaport_port" "port" {
					product_name            = "%s"
					port_speed              = 1000
					location_id             = data.megaport_location.loc1.id
					contract_term_months    = 12
					marketplace_visibility  = true
					cost_centre = "%s"
				  }

				resource "megaport_mve" "mve" {
                    product_name  = "%s"
                    location_id = data.megaport_location.loc1.id
                    contract_term_months        = 1

					vnics = [
						{
							description = "to_aws"
						},
						{
								description = "to_port"
						},
					]

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "MEDIUM"
                        image_id = 23
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
                    }
                  }

				  resource "megaport_vxc" "aws_vxc" {
					product_name            = "%s"
					rate_limit              = 100
					contract_term_months    = 1

					a_end = {
						requested_product_uid = megaport_mve.mve.product_uid
						inner_vlan            = 99
						vnic_index            = 0
					}

					b_end = {
						requested_product_uid = data.megaport_partner.aws_port.product_uid
					}

					b_end_partner_config = {
					  partner = "aws"
					  aws_config = {
						name          = "%s"
						asn           = 65121
						type          = "private"
						connect_type  = "AWSHC"
						amazon_asn    = 64512
						owner_account = "123456789012"
					  }
					}
					resource_tags = {
						"key1updated" = "value1updated"
						"key2updated" = "value2updated"
					}
				  }

                  `, VXCLocationOne, portName, costCentreName, mveName, mveName, mveName, awsVXCName, awsVXCName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "b_end_partner_config.aws_config.name", awsVXCName),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.inner_vlan", "99"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "resource_tags.key2updated", "value2updated"),
				),
			},
		},
	})
}
