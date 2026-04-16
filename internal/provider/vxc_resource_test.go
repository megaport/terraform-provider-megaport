package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	MVEArubaImageID = 152
)

func TestAccMegaportVXC_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocations(t, 1)
	portName1 := RandomTestName()
	portName2 := RandomTestName()
	portName3 := RandomTestName()
	portName4 := RandomTestName()
	vxcName := RandomTestName()
	vxcNameNew := RandomTestName()
	costCentreName := RandomTestName()
	costCentreNew := RandomTestName()

	resource.Test(t, resource.TestCase{
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
                  `, locs[0], portName1, portName2, portName3, portName4, vxcName, costCentreName),
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
			      `, locs[0], portName1, portName2, portName3, portName4, vxcName, costCentreName),
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
			      `, locs[0], portName1, portName2, portName3, portName4, vxcNameNew, costCentreNew),
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

func TestAccMegaportVXC_CostCentreRemoval(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocations(t, 1)
	portName1 := RandomTestName()
	portName2 := RandomTestName()
	vxcName := RandomTestName()
	costCentreName := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc" {
					id = %d
				}
				resource "megaport_port" "port_1" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.loc.id
					contract_term_months = 1
					marketplace_visibility = false
				}
				resource "megaport_port" "port_2" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.loc.id
					contract_term_months = 1
					marketplace_visibility = false
				}
				resource "megaport_vxc" "vxc" {
					product_name = "%s"
					rate_limit = 200
					contract_term_months = 1
					cost_centre = "%s"
					resource_tags = {
						"key1" = "value1"
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
				}`, locs[0], portName1, portName2, vxcName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "cost_centre", costCentreName),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc" {
					id = %d
				}
				resource "megaport_port" "port_1" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.loc.id
					contract_term_months = 1
					marketplace_visibility = false
				}
				resource "megaport_port" "port_2" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.loc.id
					contract_term_months = 1
					marketplace_visibility = false
				}
				resource "megaport_vxc" "vxc" {
					product_name = "%s"
					rate_limit = 200
					contract_term_months = 1
					cost_centre = ""
					resource_tags = {
						"key1" = "value1"
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
				}`, locs[0], portName1, portName2, vxcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "cost_centre", ""),
				),
			},
		},
	})
}

func TestAccMegaportVXC_ContractTermUpdate(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocations(t, 1)
	portName1 := RandomTestName()
	portName2 := RandomTestName()
	vxcName := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc" {
					id = %d
				}
				resource "megaport_port" "port_1" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.loc.id
					contract_term_months = 1
					marketplace_visibility = false
				}
				resource "megaport_port" "port_2" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.loc.id
					contract_term_months = 1
					marketplace_visibility = false
				}
				resource "megaport_vxc" "vxc" {
					product_name = "%s"
					rate_limit = 200
					contract_term_months = 1
					a_end = {
						requested_product_uid = megaport_port.port_1.product_uid
						ordered_vlan = 100
					}
					b_end = {
						requested_product_uid = megaport_port.port_2.product_uid
						ordered_vlan = 101
					}
				}`, locs[0], portName1, portName2, vxcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "contract_term_months", "1"),
					waitForProvisioningStatus("megaport_vxc.vxc"),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc" {
					id = %d
				}
				resource "megaport_port" "port_1" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.loc.id
					contract_term_months = 1
					marketplace_visibility = false
				}
				resource "megaport_port" "port_2" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.loc.id
					contract_term_months = 1
					marketplace_visibility = false
				}
				resource "megaport_vxc" "vxc" {
					product_name = "%s"
					rate_limit = 200
					contract_term_months = 12
					a_end = {
						requested_product_uid = megaport_port.port_1.product_uid
						ordered_vlan = 100
					}
					b_end = {
						requested_product_uid = megaport_port.port_2.product_uid
						ordered_vlan = 101
					}
				}`, locs[0], portName1, portName2, vxcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "contract_term_months", "12"),
				),
			},
		},
	})
}

func TestAccMegaportVXC_BasicUntagVLAN(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocations(t, 1)
	portName1 := RandomTestName()
	portName2 := RandomTestName()
	vxcName := RandomTestName()
	vxcNameNew := RandomTestName()
	costCentreName := RandomTestName()
	costCentreNew := RandomTestName()

	resource.Test(t, resource.TestCase{
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
                  `, locs[0], portName1, portName2, vxcName, costCentreName),
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
                  `, locs[0], portName1, portName2, vxcName, costCentreName),
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
			      `, locs[0], portName1, portName2, vxcNameNew, costCentreNew),
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

// func TestUpdateVLAN(t *testing.T) {
// 	t.Parallel()
// 	defer acquireAccTestSlot(t)()
// 	portName := RandomTestName()
// 	costCentreName := RandomTestName()
// 	awsVXCName := RandomTestName()

// 	resource.Test(t, resource.TestCase{
// 		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: providerConfig + fmt.Sprintf(`
// 				data "megaport_location" "loc1" {
// 					id = %d
// 				  }

// 				  data "megaport_location" "loc2" {
// 					id = %d
// 				  }

// 				  data "megaport_partner" "aws_port" {
// 					connect_type = "AWS"
// 					company_name = "AWS"
// 					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
// 					location_id  = data.megaport_location.loc1.id
// 				  }

// 				  resource "megaport_port" "port" {
// 					product_name            = "%s"
// 					port_speed              = 1000
// 					location_id             = data.megaport_location.loc2.id
// 					contract_term_months    = 12
// 					marketplace_visibility  = true
// 					cost_centre = "%s"
// 				  }

// 				  resource "megaport_vxc" "aws_vxc" {
// 					product_name            = "%s"
// 					rate_limit              = 1000
// 					contract_term_months    = 1

// 					a_end = {
// 					  requested_product_uid = megaport_port.port.product_uid
// 					  ordered_vlan = 191
// 					}

// 					b_end = {
// 					  requested_product_uid = data.megaport_partner.aws_port.product_uid
// 					}

// 					b_end_partner_config = {
// 					  partner = "aws"
// 					  aws_config = {
// 						name          = "%s"
// 						asn           = 64550
// 						type          = "private"
// 						connect_type  = "AWSHC"
// 						amazon_asn    = 64551
// 						owner_account = "123456789012"
// 					  }
// 					}
// 				  }
//                   `, VXCLocationID1, VXCLocationID2, portName, costCentreName, awsVXCName, awsVXCName),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
// 					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "b_end_partner_config.aws_config.name", awsVXCName),
// 					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.ordered_vlan", "191"),
// 					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.vlan", "191"),
// 				),
// 			},
// 			// ImportState testing
// 			{
// 				ResourceName:                         "megaport_vxc.aws_vxc",
// 				ImportState:                          true,
// 				ImportStateVerify:                    true,
// 				ImportStateVerifyIdentifierAttribute: "product_uid",
// 				ImportStateIdFunc: func(state *terraform.State) (string, error) {
// 					resourceName := "megaport_vxc.aws_vxc"
// 					var rawState map[string]string
// 					for _, m := range state.Modules {
// 						if len(m.Resources) > 0 {
// 							if v, ok := m.Resources[resourceName]; ok {
// 								rawState = v.Primary.Attributes
// 							}
// 						}
// 					}
// 					return rawState["product_uid"], nil
// 				},
// 				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end_partner_config", "b_end_partner_config", "a_end.ordered_vlan", "b_end.ordered_vlan", "a_end.requested_product_uid", "b_end.requested_product_uid"},
// 			},
// 			// Update Test - Change A-End VLAN
// 			{
// 				Config: providerConfig + fmt.Sprintf(`
// 				data "megaport_location" "loc1" {
// 					id = %d
// 				  }

// 				  data "megaport_location" "loc2" {
// 					id = %d
// 				  }
// 				  data "megaport_partner" "aws_port" {
// 					connect_type = "AWS"
// 					company_name = "AWS"
// 					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
// 					location_id  = data.megaport_location.loc1.id
// 				  }

// 				  resource "megaport_port" "port" {
// 					product_name            = "%s"
// 					port_speed              = 1000
// 					location_id             = data.megaport_location.loc2.id
// 					contract_term_months    = 12
// 					marketplace_visibility  = true
// 					cost_centre = "%s"
// 				  }

// 				  resource "megaport_vxc" "aws_vxc" {
// 					product_name            = "%s"
// 					rate_limit              = 1000
// 					contract_term_months    = 1

// 					a_end = {
// 					  requested_product_uid = megaport_port.port.product_uid
// 					  ordered_vlan = 195
// 					}

// 					b_end = {
// 					  requested_product_uid = data.megaport_partner.aws_port.product_uid
// 					}

// 					b_end_partner_config = {
// 					  partner = "aws"
// 					  aws_config = {
// 						name          = "%s"
// 						asn           = 64550
// 						type          = "private"
// 						connect_type  = "AWSHC"
// 						amazon_asn    = 64551
// 						owner_account = "123456789012"
// 					  }
// 					}
// 				  }
//                   `, VXCLocationID1, VXCLocationID2, portName, costCentreName, awsVXCName, awsVXCName),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
// 					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "b_end_partner_config.aws_config.name", awsVXCName),
// 					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.ordered_vlan", "195"),
// 					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.vlan", "195"),
// 				),
// 			},
// 			// Update Test - Untag VLAN
// 			{
// 				Config: providerConfig + fmt.Sprintf(`
// 				data "megaport_location" "loc1" {
// 					id = %d
// 				  }

// 				  data "megaport_location" "loc2" {
// 					id = %d
// 				  }
// 				  data "megaport_partner" "aws_port" {
// 					connect_type = "AWS"
// 					company_name = "AWS"
// 					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
// 					location_id  = data.megaport_location.loc1.id
// 				  }

// 				  resource "megaport_port" "port" {
// 					product_name            = "%s"
// 					port_speed              = 1000
// 					location_id             = data.megaport_location.loc2.id
// 					contract_term_months    = 12
// 					marketplace_visibility  = true
// 					cost_centre = "%s"
// 				  }

// 				  resource "megaport_vxc" "aws_vxc" {
// 					product_name            = "%s"
// 					rate_limit              = 1000
// 					contract_term_months    = 1

// 					a_end = {
// 					  requested_product_uid = megaport_port.port.product_uid
// 					  ordered_vlan = -1
// 					}

// 					b_end = {
// 					  requested_product_uid = data.megaport_partner.aws_port.product_uid
// 					}

// 					b_end_partner_config = {
// 					  partner = "aws"
// 					  aws_config = {
// 						name          = "%s"
// 						asn           = 64550
// 						type          = "private"
// 						connect_type  = "AWSHC"
// 						amazon_asn    = 64551
// 						owner_account = "123456789012"
// 					  }
// 					}
// 				  }
//                   `, VXCLocationID1, VXCLocationID2, portName, costCentreName, awsVXCName, awsVXCName),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
// 					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "b_end_partner_config.aws_config.name", awsVXCName),
// 					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "a_end.ordered_vlan", "-1"),
// 					resource.TestCheckNoResourceAttr("megaport_vxc.aws_vxc", "a_end.vlan"),
// 				),
// 			},
// 		},
// 	})
// }

func TestAccMegaportMCRVXCWithCSPs_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocationsWithPartner(t, 1, "AWS")
	azure := pickAzureServiceKey(t)
	gcp := pickGCPPairingKey(t)
	mcrLocationID, _ := findMCRTestLocation(t, 5000)
	mcrName := RandomTestName()
	vxcName1 := RandomTestName()
	vxcName2 := RandomTestName()
	vxcName3 := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
                    id    = %d
                  }

                  data "megaport_location" "loc2" {
                    id = %d
                  }

                  data "megaport_partner" "aws_port" {
                    connect_type = "AWS"
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
                            pairing_key = "%s"
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

                    b_end = {
                      requested_product_uid = "%s"
                    }

                    b_end_partner_config = {
                        partner = "azure"
                        azure_config = {
							port_choice = "primary"
                            service_key = "%s"
                        }
                    }
                  }
                  `, mcrLocationID, locs[0], mcrName, vxcName1, vxcName1, vxcName2, gcp.Key, vxcName3, azure.PartnerPortUID, azure.Key),
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

func TestAccMegaportMCRVXCWithBGP_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	mcrLocID, _ := findMCRTestLocation(t, 5000)
	awsLocs := findVXCPortTestLocationsWithPartner(t, 1, "AWS")
	locs := []int{mcrLocID, awsLocs[0]}
	mcrName := RandomTestName()
	vxcName1 := RandomTestName()
	prefixFilterListName := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					id = %d
				  }

				  data "megaport_location" "loc2" {
					id = %d
				  }

				  data "megaport_partner" "aws_port" {
					connect_type = "AWS"
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
                  `, locs[0], locs[1], mcrName, prefixFilterListName, vxcName1, prefixFilterListName, vxcName1),
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
					id = %d
				  }

				  data "megaport_location" "loc2" {
					id = %d
				  }

				  data "megaport_partner" "aws_port" {
					connect_type = "AWS"
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
                  `, locs[0], locs[1], mcrName, prefixFilterListName, vxcName1, prefixFilterListName, vxcName1),
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

func TestGCPVXCWithProductUID(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	gcp := pickGCPPairingKey(t)
	mcrName := RandomTestName()
	mcrCostCentreName := RandomTestName()
	gcpCostCentreName := RandomTestName()
	gcpVXCName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				  resource "megaport_mcr" "mcr" {
					product_name            = "%s"
					port_speed              = 2500
					location_id             = %d
					contract_term_months    = 1
					asn                      = 64555
					cost_centre = "%s"
				  }

				  resource "megaport_vxc" "gcp_vxc" {
					product_name            = "%s"
					rate_limit              = 1000
					contract_term_months    = 12
					cost_centre             = "%s"

					a_end = {
					  requested_product_uid = megaport_mcr.mcr.product_uid
					  ordered_vlan = 182
					}

					b_end = {
					  requested_product_uid = "%s"
					}

					b_end_partner_config = {
					  partner = "google"
					  google_config = {
						pairing_key = "%s"
					  }
					}
				  }
                  `, mcrName, gcp.LocationID, mcrCostCentreName, gcpVXCName, gcpCostCentreName, gcp.PartnerPortUID, gcp.Key),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_vxc.gcp_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.gcp_vxc", "cost_centre", gcpCostCentreName),
					resource.TestCheckResourceAttrSet("megaport_vxc.gcp_vxc", "b_end.product_name"),
				),
			},
		},
	})
}

func TestOracleVXCWithProductUID(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocationsWithPartner(t, 1, "ORACLE")
	oracleVCID := pickOracleVirtualCircuitID(t)
	mcrName := RandomTestName()
	mcrCostCentreName := RandomTestName()
	oracleCostCentreName := RandomTestName()
	oracleVXCName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					id = %d
				  }

				data "megaport_partner" "oracle_port" {
  					connect_type = "ORACLE"
  					location_id  = 147
				  }

				  resource "megaport_mcr" "mcr" {
					product_name            = "%s"
					port_speed              = 2500
					location_id             = data.megaport_location.loc1.id
					contract_term_months    = 1
					asn                      = 64555
					cost_centre = "%s"
				  }

				  resource "megaport_vxc" "oracle_vxc" {
					product_name            = "%s"
					rate_limit              = 1000
					contract_term_months    = 12
					cost_centre             = "%s"

					a_end = {
					  requested_product_uid = megaport_mcr.mcr.product_uid
					  ordered_vlan = 182
					}

					b_end = {
					  requested_product_uid = data.megaport_partner.oracle_port.product_uid
					}

					b_end_partner_config = {
                        partner = "oracle"
                        oracle_config = {
                            virtual_circuit_id = "%s"
                        }
                    }
				  }
                  `, locs[0], mcrName, mcrCostCentreName, oracleVXCName, oracleCostCentreName, oracleVCID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_vxc.oracle_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.oracle_vxc", "cost_centre", oracleCostCentreName),
					resource.TestCheckResourceAttrSet("megaport_vxc.oracle_vxc", "b_end.product_name"),
				),
			},
		},
	})
}

func TestAzureVXCWithProductUID(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	azure := pickAzureServiceKey(t)
	mcrLocationID, _ := findMCRTestLocation(t, 2500)
	mcrName := RandomTestName()
	mcrCostCentreName := RandomTestName()
	azureCostCentreName := RandomTestName()
	azureVXCName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					id = %d
				  }

				  resource "megaport_mcr" "mcr" {
					product_name            = "%s"
					port_speed              = 2500
					location_id             = data.megaport_location.loc1.id
					contract_term_months    = 1
					asn                      = 64555
					cost_centre = "%s"
				  }

				  resource "megaport_vxc" "azure_vxc" {
					product_name            = "%s"
					rate_limit              = 1000
					contract_term_months    = 12
					cost_centre             = "%s"

					a_end = {
					  requested_product_uid = megaport_mcr.mcr.product_uid
					  ordered_vlan = 182
					}

					b_end = {
					  requested_product_uid = "%s"
					}

					b_end_partner_config = {
					  partner = "azure"
					  azure_config = {
						service_key = "%s"
						port_choice = "secondary"
					  }
					}
				  }
                  `, mcrLocationID, mcrName, mcrCostCentreName, azureVXCName, azureCostCentreName, azure.PartnerPortUID, azure.Key),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_vxc.azure_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.azure_vxc", "cost_centre", azureCostCentreName),
					resource.TestCheckResourceAttrSet("megaport_vxc.azure_vxc", "b_end.product_name"),
				),
			},
		},
	})
}

// TestAccMegaportMCRVXC_BEndIpMtu tests that ip_mtu is correctly applied to
// the B-End vrouter partner config of an MCR-to-MCR VXC (GitHub issue #319).
func TestAccMegaportMCRVXC_BEndIpMtu(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortAndMCRTestLocations(t, 1, 1000)
	mcrNameA := RandomTestName()
	mcrNameB := RandomTestName()
	vxcName := RandomTestName()

	vxcConfig := func(ipMtu int) string {
		return providerConfig + fmt.Sprintf(`
			data "megaport_location" "loc" {
				id = %d
			}

			resource "megaport_mcr" "mcr_a" {
				product_name         = "%s"
				location_id          = data.megaport_location.loc.id
				contract_term_months = 1
				port_speed           = 1000
				asn                  = 64555
			}

			resource "megaport_mcr" "mcr_b" {
				product_name         = "%s"
				location_id          = data.megaport_location.loc.id
				contract_term_months = 1
				port_speed           = 1000
				asn                  = 64556
			}

			resource "megaport_vxc" "vxc" {
				product_name         = "%s"
				rate_limit           = 500
				contract_term_months = 1

				a_end = {
					requested_product_uid = megaport_mcr.mcr_a.product_uid
					ordered_vlan          = 100
				}

				a_end_partner_config = {
					partner = "vrouter"
					vrouter_config = {
						interfaces = [{
							ip_addresses = ["10.0.0.1/30"]
							ip_mtu       = %d
							bgp_connections = [{
								peer_asn         = 64556
								local_ip_address = "10.0.0.1"
								peer_ip_address  = "10.0.0.2"
								shutdown         = false
								description      = "A-End BGP"
								med_in           = 100
								med_out          = 100
								bfd_enabled      = false
								export_policy    = "permit"
							}]
						}]
					}
				}

				b_end = {
					requested_product_uid = megaport_mcr.mcr_b.product_uid
					ordered_vlan          = 200
				}

				b_end_partner_config = {
					partner = "vrouter"
					vrouter_config = {
						interfaces = [{
							ip_addresses = ["10.0.0.2/30"]
							ip_mtu       = %d
							bgp_connections = [{
								peer_asn         = 64555
								local_ip_address = "10.0.0.2"
								peer_ip_address  = "10.0.0.1"
								shutdown         = false
								description      = "B-End BGP"
								med_in           = 100
								med_out          = 100
								bfd_enabled      = false
								export_policy    = "permit"
							}]
						}]
					}
				}
			}
		`, locs[0], mcrNameA, mcrNameB, vxcName, ipMtu, ipMtu)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create MCR-to-MCR VXC with ip_mtu 9000 on both ends
			{
				Config: vxcConfig(9000),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end_partner_config.partner", "vrouter"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end_partner_config.partner", "vrouter"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end_partner_config.vrouter_config.interfaces.0.ip_mtu", "9000"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end_partner_config.vrouter_config.interfaces.0.ip_mtu", "9000"),
				),
			},
			// Step 2: Update ip_mtu to 1500 on both ends
			{
				Config: vxcConfig(1500),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end_partner_config.vrouter_config.interfaces.0.ip_mtu", "1500"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end_partner_config.vrouter_config.interfaces.0.ip_mtu", "1500"),
				),
			},
		},
	})
}

func TestFullEcosystem(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocationsWithPartner(t, 3, "AWS")
	azure := pickAzureServiceKey(t)
	gcp := pickGCPPairingKey(t)
	portName := RandomTestName()
	lagPortName := RandomTestName()
	mcrName := RandomTestName()
	portVXCName := RandomTestName()
	costCentreName := RandomTestName()
	mcrVXCName := RandomTestName()
	awsVXCName := RandomTestName()
	gcpVXCName := RandomTestName()
	azureVXCName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					id = %d
				  }

				  data "megaport_location" "loc2" {
					id = %d
				  }

				  data "megaport_location" "loc3" {
					id = %d
				  }

				  data "megaport_partner" "aws_port" {
					connect_type = "AWS"
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
						pairing_key = "%s"
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

					b_end = {
					  requested_product_uid = "%s"
					}

					b_end_partner_config = {
					  partner = "azure"
					  azure_config = {
						service_key = "%s"
					  port_choice = "primary"
					  }
					}
				  }
                  `, locs[0], locs[1], locs[2], lagPortName, costCentreName, portName, costCentreName, mcrName, portVXCName, mcrVXCName, awsVXCName, awsVXCName, gcpVXCName, gcp.Key, azureVXCName, azure.PartnerPortUID, azure.Key),
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

func TestAccMegaportOracleVXC_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocationsWithPartner(t, 1, "ORACLE")
	oracleVCID := pickOracleVirtualCircuitID(t)
	portName := RandomTestName()
	oracleVXCName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
                data "megaport_location" "loc1" {
                    id = %d
                }

                resource "megaport_port" "port" {
                    product_name           = "%s"
                    port_speed             = 1000
                    location_id            = data.megaport_location.loc1.id
                    contract_term_months   = 12
                    marketplace_visibility = false
                }

                resource "megaport_vxc" "oracle_vxc" {
                    product_name            = "%s"
                    rate_limit              = 100
                    contract_term_months    = 1

                    a_end = {
                        requested_product_uid = megaport_port.port.product_uid
                        ordered_vlan          = 0
                    }

                    b_end = {}

                    b_end_partner_config = {
                        partner = "oracle"
                        oracle_config = {
                            virtual_circuit_id = "%s"
                        }
                    }
                }
                `, locs[0], portName, oracleVXCName, oracleVCID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "product_name", portName),
					resource.TestCheckResourceAttr("megaport_port.port", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "product_uid"),

					resource.TestCheckResourceAttr("megaport_vxc.oracle_vxc", "product_name", oracleVXCName),
					resource.TestCheckResourceAttr("megaport_vxc.oracle_vxc", "rate_limit", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.oracle_vxc", "contract_term_months", "1"),
					resource.TestCheckResourceAttrSet("megaport_vxc.oracle_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.oracle_vxc", "b_end_partner_config.oracle_config.virtual_circuit_id", oracleVCID),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.oracle_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.oracle_vxc"
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

func TestMVE_TransitVXC(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocationsWithPartner(t, 2, "TRANSIT")
	portName := RandomTestName()
	costCentreName := RandomTestName()
	mveName := RandomTestName()
	transitVXCName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					id = %d
				  }

				  data "megaport_location" "loc2" {
					id = %d
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
					  product_size  = "SMALL"
					  mve_label     = "MVE 2/8"
					  image_id      = %d
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
                  `, locs[0], locs[1], portName, costCentreName, mveName, MVEArubaImageID, mveName, mveName, transitVXCName),
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

func TestMVE_TransitVXCAWS(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	// loc2 needs both AWS and TRANSIT partner ports; loc1 only needs port capacity.
	partnerLocs := findVXCPortTestLocationsWithPartners(t, 1, "AWS", "TRANSIT")
	portLocs := findVXCPortTestLocations(t, 1)
	locs := []int{portLocs[0], partnerLocs[0]}
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

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					id = %d
				  }

				  data "megaport_location" "loc2" {
					id = %d
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
					location_id   = data.megaport_location.loc2.id
				  }

				   data "megaport_partner" "aws_port" {
					connect_type = "AWS"
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
					  product_size  = "SMALL"
					  mve_label     = "MVE 2/8"
					  image_id      = %d
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
                  `, locs[0], locs[1], portName, portCostCentreName, mveName, MVEArubaImageID, mveName, mveName, transitVXCName, transitVXCCostCentreName, portVXCName, portVXCCostCentreName, portVXCAEndInnerVLAN, portVXCBEndInnerVLAN, awsVXCName, awsVXCCostCentreName, awsVXCAEndInnerVLAN, awsVXCName),
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
					id = %d
				  }

				  data "megaport_location" "loc2" {
					id = %d
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
					location_id   = data.megaport_location.loc2.id
				  }

				   data "megaport_partner" "aws_port" {
					connect_type = "AWS"
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
					  product_size  = "SMALL"
					  mve_label     = "MVE 2/8"
					  image_id      = %d
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
                  `, locs[0], locs[1], portName, portCostCentreNameNew, mveName, MVEArubaImageID, mveName, mveName, transitVXCName, transitVXCCostCentreNameNew, portVXCName, portVXCCostCentreNameNew, portVXCAEndInnerVLANNew, portVXCBEndInnerVLANNew, awsVXCName, awsVXCCostCentreNameNew, awsVXCAEndInnerVLANNew, awsVXCName),
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

func TestMVE_AWS_VXC(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	mveLocID, _ := findMVETestLocation(t, 0)
	awsLocs := findVXCPortTestLocationsWithPartner(t, 1, "AWS")
	portName := RandomTestName()
	costCentreName := RandomTestName()
	mveName := RandomTestName()
	awsVXCName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc1" {
					id = %d
				  }

				  data "megaport_location" "syd_gs" {
					id = %d
				  }

				  data "megaport_partner" "aws_port" {
					connect_type = "AWS"
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
                        product_size = "SMALL"
						mve_label     = "MVE 2/8"
                        image_id = %d
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

                  `, mveLocID, awsLocs[0], portName, costCentreName, mveName, MVEArubaImageID, mveName, mveName, awsVXCName, awsVXCName),
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
					id = %d
				  }

				  data "megaport_location" "loc2" {
					id = %d
				  }

				  data "megaport_partner" "aws_port" {
					connect_type = "AWS"
					location_id  = data.megaport_location.loc2.id
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
                        product_size = "SMALL"
						mve_label     = "MVE 2/8"
                        image_id = %d
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

                  `, mveLocID, awsLocs[0], portName, costCentreName, mveName, MVEArubaImageID, mveName, mveName, awsVXCName, awsVXCName),
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

func TestAccMegaportVXC_InnerVLANUntagged(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocations(t, 1)
	portName1 := RandomTestName()
	portName2 := RandomTestName()
	vxcName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create VXC with inner_vlan = -1 (untagged)
			{
				Config: providerConfig + fmt.Sprintf(`
                data "megaport_location" "loc" {
                    id = %d
                }
                resource "megaport_port" "port_1" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months = 1
                    marketplace_visibility = false
                }
                resource "megaport_port" "port_2" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months = 1
                    marketplace_visibility = false
                }
                resource "megaport_vxc" "vxc_test" {
                    product_name = "%s"
                    rate_limit = 100
                    contract_term_months = 1
                    
                    a_end = {
                        requested_product_uid = megaport_port.port_1.product_uid
                        ordered_vlan = 310
                        inner_vlan = -1
                    }
                    
                    b_end = {
                        requested_product_uid = megaport_port.port_2.product_uid
                        ordered_vlan = 311
                        inner_vlan = -1
                    }
                }
                `, locs[0], portName1, portName2, vxcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc_test", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc_test", "a_end.inner_vlan", "-1"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc_test", "b_end.inner_vlan", "-1"),
				),
			},
		},
	})
}

func TestAccMegaportVXC_InnerVLANNull(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocations(t, 1)
	portName1 := RandomTestName()
	portName2 := RandomTestName()
	vxcName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create VXC without specifying inner_vlan (null)
			{
				Config: providerConfig + fmt.Sprintf(`
                data "megaport_location" "loc" {
                    id = %d
                }
                resource "megaport_port" "port_1" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months = 1
                    marketplace_visibility = false
                }
                resource "megaport_port" "port_2" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months = 1
                    marketplace_visibility = false
                }
                resource "megaport_vxc" "vxc_test" {
                    product_name = "%s"
                    rate_limit = 100
                    contract_term_months = 1
                    
                    a_end = {
                        requested_product_uid = megaport_port.port_1.product_uid
                        ordered_vlan = 310
                        // inner_vlan not specified (null)
                    }
                    
                    b_end = {
                        requested_product_uid = megaport_port.port_2.product_uid
                        ordered_vlan = 311
                        // inner_vlan not specified (null)
                    }
                }
                `, locs[0], portName1, portName2, vxcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc_test", "product_uid"),
					resource.TestCheckNoResourceAttr("megaport_vxc.vxc_test", "a_end.inner_vlan"),
					resource.TestCheckNoResourceAttr("megaport_vxc.vxc_test", "b_end.inner_vlan"),
				),
			},
		},
	})
}

func TestAccMegaportVXC_InnerVLANToUntagged(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocations(t, 1)
	portName1 := RandomTestName()
	portName2 := RandomTestName()
	vxcName := RandomTestName()

	initialInnerVLAN := 456

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create VXC with specific inner_vlan values
			{
				Config: providerConfig + fmt.Sprintf(`
                data "megaport_location" "loc" {
                    id = %d
                }
                resource "megaport_port" "port_1" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months = 1
                    marketplace_visibility = false
                }
                resource "megaport_port" "port_2" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months = 1
                    marketplace_visibility = false
                }
                resource "megaport_vxc" "vxc_test" {
                    product_name = "%s"
                    rate_limit = 100
                    contract_term_months = 1
                    
                    a_end = {
                        requested_product_uid = megaport_port.port_1.product_uid
                        ordered_vlan = 310
                        inner_vlan = %d
                    }
                    
                    b_end = {
                        requested_product_uid = megaport_port.port_2.product_uid
                        ordered_vlan = 311
                        inner_vlan = %d
                    }
                }
                `, locs[0], portName1, portName2, vxcName, initialInnerVLAN, initialInnerVLAN),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc_test", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc_test", "a_end.inner_vlan", fmt.Sprintf("%d", initialInnerVLAN)),
					resource.TestCheckResourceAttr("megaport_vxc.vxc_test", "b_end.inner_vlan", fmt.Sprintf("%d", initialInnerVLAN)),
				),
			},
			// Step 2: Update inner_vlan values to -1 (untagged)
			{
				Config: providerConfig + fmt.Sprintf(`
                data "megaport_location" "loc" {
                    id = %d
                }
                resource "megaport_port" "port_1" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months = 1
                    marketplace_visibility = false
                }
                resource "megaport_port" "port_2" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months = 1
                    marketplace_visibility = false
                }
                resource "megaport_vxc" "vxc_test" {
                    product_name = "%s"
                    rate_limit = 100
                    contract_term_months = 1
                    
                    a_end = {
                        requested_product_uid = megaport_port.port_1.product_uid
                        ordered_vlan = 310
                        inner_vlan = -1
                    }
                    
                    b_end = {
                        requested_product_uid = megaport_port.port_2.product_uid
                        ordered_vlan = 311
                        inner_vlan = -1
                    }
                }
                `, locs[0], portName1, portName2, vxcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc_test", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc_test", "a_end.inner_vlan", "-1"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc_test", "b_end.inner_vlan", "-1"),
				),
			},
		},
	})
}

func TestAccMegaportSafeDelete(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocations(t, 1)
	mveLocationID, _ := findMVETestLocationBlueZone(t)
	mcrLocationID, _ := findMCRTestLocation(t, 2500)
	portName := RandomTestName()
	mcrName := RandomTestName()
	mveName := RandomTestName()
	vxcPortToMCRName := RandomTestName()
	vxcMCRToMVEName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create port, MCR, MVE and connect them with VXCs
			{
				Config: providerConfig + fmt.Sprintf(`
                // Create a port
                resource "megaport_port" "test_port" {
                    product_name         = "%s"
                    port_speed           = 1000
                    location_id          = %d
                    contract_term_months = 1
                    marketplace_visibility = false
                }

                // Create an MCR
                resource "megaport_mcr" "test_mcr" {
                    product_name         = "%s"
                    port_speed           = 1000
                    location_id          = %d
                    contract_term_months = 1
                }

                // Create an MVE
                resource "megaport_mve" "test_mve" {
                    product_name         = "%s"
                    location_id          = %d
                    contract_term_months = 1
                    diversity_zone       = "blue"

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
                        vendor       = "aruba"
                        product_size = "SMALL"
						mve_label     = "MVE 2/8"
                        image_id     = %d
                        account_name = "%s-account"
                        account_key  = "%s-key"
                        system_tag   = "Preconfiguration-test-1"
                    }
                }

                // Connect port to MCR with VXC
                resource "megaport_vxc" "port_to_mcr" {
                    product_name         = "%s"
                    rate_limit           = 100
                    contract_term_months = 1

                    a_end = {
                        requested_product_uid = megaport_port.test_port.product_uid
                        ordered_vlan          = 100
                    }

                    b_end = {
                        requested_product_uid = megaport_mcr.test_mcr.product_uid
                        ordered_vlan          = 101
                    }
                }

                // Connect MCR to MVE with VXC
                resource "megaport_vxc" "mcr_to_mve" {
                    product_name         = "%s"
                    rate_limit           = 100
                    contract_term_months = 1

                    a_end = {
                        requested_product_uid = megaport_mcr.test_mcr.product_uid
                        ordered_vlan          = 200
                    }

                    b_end = {
                        requested_product_uid = megaport_mve.test_mve.product_uid
                        vnic_index            = 0
                        ordered_vlan          = 201
                    }
                }
                `,
					portName, locs[0],
					mcrName, mcrLocationID,
					mveName, mveLocationID, MVEArubaImageID,
					mveName, mveName,
					vxcPortToMCRName, vxcMCRToMVEName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.test_port", "product_name", portName),
					resource.TestCheckResourceAttr("megaport_mcr.test_mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mve.test_mve", "product_name", mveName),
					resource.TestCheckResourceAttrSet("megaport_vxc.port_to_mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_vxc.mcr_to_mve", "product_uid"),
				),
			},
			// Step 2: Try to delete the port while keeping the VXC - this should fail
			{
				Config: providerConfig + fmt.Sprintf(`
                // Keep only the VXCs - this should fail because we can't delete 
                // resources with VXCs still connected

                // Connect port to MCR with VXC - now referring to resources
                // that we're trying to delete
                resource "megaport_vxc" "port_to_mcr" {
                    product_name         = "%s"
                    rate_limit           = 100
                    contract_term_months = 1

                    a_end = {
                        requested_product_uid = "%s" // Direct product_uid instead of reference
                        ordered_vlan          = 100
                    }

                    b_end = {
                        requested_product_uid = "%s" // Direct product_uid instead of reference
                        ordered_vlan          = 101
                    }
                }

                // Connect MCR to MVE with VXC
                resource "megaport_vxc" "mcr_to_mve" {
                    product_name         = "%s"
                    rate_limit           = 100
                    contract_term_months = 1

                    a_end = {
                        requested_product_uid = "%s" // Direct product_uid instead of reference
                        ordered_vlan          = 200
                    }

                    b_end = {
                        requested_product_uid = "%s" // Direct product_uid instead of reference
                        vnic_index            = 0
                        ordered_vlan          = 201
                    }
                }
                `,
					vxcPortToMCRName, "PORT_UID_PLACEHOLDER", "MCR_UID_PLACEHOLDER",
					vxcMCRToMVEName, "MCR_UID_PLACEHOLDER", "MVE_UID_PLACEHOLDER"),
				ExpectError: regexp.MustCompile(`has active VXCs associated with it.`),
			},
			// Step 3: Delete the VXCs first, then delete the resources
			{
				Config: providerConfig + fmt.Sprintf(`
                // Create a port
                resource "megaport_port" "test_port" {
                    product_name         = "%s"
                    port_speed           = 1000
                    location_id          = %d
                    contract_term_months = 1
                    marketplace_visibility = false
                }

                // Create an MCR
                resource "megaport_mcr" "test_mcr" {
                    product_name         = "%s"
                    port_speed           = 1000
                    location_id          = %d
                    contract_term_months = 1
                }

                // Create an MVE
                resource "megaport_mve" "test_mve" {
                    product_name         = "%s"
                    location_id          = %d
                    contract_term_months = 1
                    diversity_zone       = "blue"

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
                        vendor       = "aruba"
                        product_size = "SMALL"
						mve_label     = "MVE 2/8"
                        image_id     = %d
                        account_name = "%s-account"
                        account_key  = "%s-key"
                        system_tag   = "Preconfiguration-test-1"
                    }
                }
                `,
					portName, locs[0],
					mcrName, mcrLocationID,
					mveName, mveLocationID, MVEArubaImageID,
					mveName, mveName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.test_port", "product_name", portName),
					resource.TestCheckResourceAttr("megaport_mcr.test_mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mve.test_mve", "product_name", mveName),
				),
			},
			// Step 4: Now we can delete the resources safely
			{
				Config: providerConfig,
			},
		},
	})
}

func TestAccMegaportMVE_to_MVE_VXC(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	mveLocationID, _ := findMVETestLocationHighCapacity(t, 4)
	mveName1 := RandomTestName()
	mveName2 := RandomTestName()
	mveName3 := RandomTestName()
	mveName4 := RandomTestName()
	vxcName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
                data "megaport_mve_images" "aruba" {
                    vendor_filter = "Aruba"
                    id_filter = %d
                }
                resource "megaport_mve" "mve_1" {
                    product_name         = "%s"
                    location_id          = %d
                    contract_term_months = 1
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
                      product_size  = "SMALL"
					  mve_label     = "MVE 2/8"
                      image_id      = data.megaport_mve_images.aruba.mve_images.0.id
                      account_name  = "%s-1"
                      account_key   = "%s-1"
                      system_tag    = "Preconfiguration-aruba-test-1"
                    }
                }
                resource "megaport_mve" "mve_2" {
                    product_name         = "%s"
                    location_id          = %d
                    contract_term_months = 1
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
                      product_size  = "SMALL"
					  mve_label     = "MVE 2/8"
                      image_id      = data.megaport_mve_images.aruba.mve_images.0.id
                      account_name  = "%s-2"
                      account_key   = "%s-2"
                      system_tag    = "Preconfiguration-aruba-test-2"
                    }
                }
                resource "megaport_mve" "mve_3" {
                    product_name         = "%s"
                    location_id          = %d
                    contract_term_months = 1
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
                      product_size  = "SMALL"
					  mve_label     = "MVE 2/8"
                      image_id      = data.megaport_mve_images.aruba.mve_images.0.id
                      account_name  = "%s-3"
                      account_key   = "%s-3"
                      system_tag    = "Preconfiguration-aruba-test-3"
                    }
                }
                resource "megaport_mve" "mve_4" {
                    product_name         = "%s"
                    location_id          = %d
                    contract_term_months = 1
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
                      product_size  = "SMALL"
					  mve_label     = "MVE 2/8"
                      image_id      = data.megaport_mve_images.aruba.mve_images.0.id
                      account_name  = "%s-4"
                      account_key   = "%s-4"
                      system_tag    = "Preconfiguration-aruba-test-4"
                    }
                }
                resource "megaport_vxc" "mve_vxc" {
                    product_name         = "%s"
                    rate_limit           = 100
                    contract_term_months = 1
                    a_end = {
                      requested_product_uid = megaport_mve.mve_1.product_uid
                      vnic_index            = 0
                    }
                    b_end = {
                      requested_product_uid = megaport_mve.mve_2.product_uid
                      vnic_index            = 0
                    }
                }
                `,
					MVEArubaImageID,
					mveName1, mveLocationID, mveName1, mveName1,
					mveName2, mveLocationID, mveName2, mveName2,
					mveName3, mveLocationID, mveName3, mveName3,
					mveName4, mveLocationID, mveName4, mveName4,
					vxcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check MVEs
					resource.TestCheckResourceAttr("megaport_mve.mve_1", "product_name", mveName1),
					resource.TestCheckResourceAttr("megaport_mve.mve_2", "product_name", mveName2),
					resource.TestCheckResourceAttr("megaport_mve.mve_3", "product_name", mveName3),
					resource.TestCheckResourceAttr("megaport_mve.mve_4", "product_name", mveName4),
					resource.TestCheckResourceAttrSet("megaport_mve.mve_1", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve_2", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve_3", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve_4", "product_uid"),

					// Check VXC connecting MVE 1 and MVE 2 with VNIC index 0
					resource.TestCheckResourceAttr("megaport_vxc.mve_vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.mve_vxc", "a_end.vnic_index", "0"),
					resource.TestCheckResourceAttr("megaport_vxc.mve_vxc", "b_end.vnic_index", "0"),
					resource.TestCheckResourceAttrSet("megaport_vxc.mve_vxc", "product_uid"),
				),
			},
			// Update test - Move VXC to MVE 3 and MVE 4, change VNIC index to 1
			{
				Config: providerConfig + fmt.Sprintf(`
                data "megaport_mve_images" "aruba" {
                    vendor_filter = "Aruba"
                    id_filter = %d
                }
                resource "megaport_mve" "mve_1" {
                    product_name         = "%s"
                    location_id          = %d
                    contract_term_months = 1
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
                      product_size  = "SMALL"
					  mve_label     = "MVE 2/8"
                      image_id      = data.megaport_mve_images.aruba.mve_images.0.id
                      account_name  = "%s-1"
                      account_key   = "%s-1"
                      system_tag    = "Preconfiguration-aruba-test-1"
                    }
                }
                resource "megaport_mve" "mve_2" {
                    product_name         = "%s"
                    location_id          = %d
                    contract_term_months = 1
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
                      product_size  = "SMALL"
					  mve_label     = "MVE 2/8"
                      image_id      = data.megaport_mve_images.aruba.mve_images.0.id
                      account_name  = "%s-2"
                      account_key   = "%s-2"
                      system_tag    = "Preconfiguration-aruba-test-2"
                    }
                }
                resource "megaport_mve" "mve_3" {
                    product_name         = "%s"
                    location_id          = %d
                    contract_term_months = 1
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
                      product_size  = "SMALL"
					  mve_label     = "MVE 2/8"
                      image_id      = data.megaport_mve_images.aruba.mve_images.0.id
                      account_name  = "%s-3"
                      account_key   = "%s-3"
                      system_tag    = "Preconfiguration-aruba-test-3"
                    }
                }
                resource "megaport_mve" "mve_4" {
                    product_name         = "%s"
                    location_id          = %d
                    contract_term_months = 1
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
                      product_size  = "SMALL"
					  mve_label     = "MVE 2/8"
                      image_id      = data.megaport_mve_images.aruba.mve_images.0.id
                      account_name  = "%s-4"
                      account_key   = "%s-4"
                      system_tag    = "Preconfiguration-aruba-test-4"
                    }
                }
                resource "megaport_vxc" "mve_vxc" {
                    product_name         = "%s"
                    rate_limit           = 100
                    contract_term_months = 1
                    a_end = {
                      requested_product_uid = megaport_mve.mve_3.product_uid
                      vnic_index            = 1
                    }
                    b_end = {
                      requested_product_uid = megaport_mve.mve_4.product_uid
                      vnic_index            = 1
                    }
                }
                `,
					MVEArubaImageID,
					mveName1, mveLocationID, mveName1, mveName1,
					mveName2, mveLocationID, mveName2, mveName2,
					mveName3, mveLocationID, mveName3, mveName3,
					mveName4, mveLocationID, mveName4, mveName4,
					vxcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check MVEs still exist
					resource.TestCheckResourceAttr("megaport_mve.mve_1", "product_name", mveName1),
					resource.TestCheckResourceAttr("megaport_mve.mve_2", "product_name", mveName2),
					resource.TestCheckResourceAttr("megaport_mve.mve_3", "product_name", mveName3),
					resource.TestCheckResourceAttr("megaport_mve.mve_4", "product_name", mveName4),

					// Check VXC has been updated to connect MVE 3 and MVE 4 with VNIC index 1
					resource.TestCheckResourceAttr("megaport_vxc.mve_vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.mve_vxc", "a_end.vnic_index", "1"),
					resource.TestCheckResourceAttr("megaport_vxc.mve_vxc", "b_end.vnic_index", "1"),

					// Verify VXC is now connected to the new MVEs
					resource.TestCheckResourceAttrPair(
						"megaport_vxc.mve_vxc", "a_end.requested_product_uid",
						"megaport_mve.mve_3", "product_uid",
					),
					resource.TestCheckResourceAttrPair(
						"megaport_vxc.mve_vxc", "b_end.requested_product_uid",
						"megaport_mve.mve_4", "product_uid",
					),
				),
			},
		},
	})
}

func TestAccMegaportVXC_MVEVnicIndexUpdate(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocations(t, 1)
	mveLocationID, _ := findMVETestLocationBlueZone(t)
	// Test names
	portName := RandomTestName()
	mveName := RandomTestName()
	vxcName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create a Port, MVE, and VXC connecting them with VNIC index 0
			{
				Config: providerConfig + fmt.Sprintf(`
                // Create a port
                resource "megaport_port" "test_port" {
                    product_name         = "%s"
                    port_speed           = 1000
                    location_id          = %d
                    contract_term_months = 1
                    marketplace_visibility = false
                }
                
                // Create an MVE
                resource "megaport_mve" "test_mve" {
                    product_name         = "%s"
                    location_id          = %d
                    contract_term_months = 1
                    diversity_zone       = "blue"

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
                        vendor       = "aruba"
                        product_size = "SMALL"
                        mve_label    = "MVE 2/8"
                        image_id     = %d
                        account_name = "%s-account"
                        account_key  = "%s-key"
                        system_tag   = "Preconfiguration-test-1"
                    }
                }

                // Connect port to MVE with VXC
                resource "megaport_vxc" "port_to_mve" {
                    product_name         = "%s"
                    rate_limit           = 100
                    contract_term_months = 1
                    
                    a_end = {
                        requested_product_uid = megaport_port.test_port.product_uid
                        ordered_vlan          = 100
                    }
                    
                    b_end = {
                        requested_product_uid = megaport_mve.test_mve.product_uid
                        vnic_index            = 0
                        ordered_vlan          = 101
                    }
                }
                `,
					portName, locs[0],
					mveName, mveLocationID, MVEArubaImageID,
					mveName, mveName,
					vxcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check port and MVE were created
					resource.TestCheckResourceAttrSet("megaport_port.test_port", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.test_mve", "product_uid"),

					// Check VXC was created with VNIC index 0
					resource.TestCheckResourceAttrSet("megaport_vxc.port_to_mve", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.port_to_mve", "b_end.vnic_index", "0"),
				),
			},
			// Step 2: Update the VNIC index to 1 - this should pass only if the VNIC index is properly sent in the update
			{
				Config: providerConfig + fmt.Sprintf(`
                // Create a port
                resource "megaport_port" "test_port" {
                    product_name         = "%s"
                    port_speed           = 1000
                    location_id          = %d
                    contract_term_months = 1
                    marketplace_visibility = false
                }

                // Create an MVE
                resource "megaport_mve" "test_mve" {
                    product_name         = "%s"
                    location_id          = %d
                    contract_term_months = 1
                    diversity_zone       = "blue"

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
                        vendor       = "aruba"
                        product_size = "SMALL"
                        mve_label    = "MVE 2/8"
                        image_id     = %d
                        account_name = "%s-account"
                        account_key  = "%s-key"
                        system_tag   = "Preconfiguration-test-1"
                    }
                }

                // Connect port to MVE with VXC - updated VNIC index
                resource "megaport_vxc" "port_to_mve" {
                    product_name         = "%s"
                    rate_limit           = 100
                    contract_term_months = 1

                    a_end = {
                        requested_product_uid = megaport_port.test_port.product_uid
                        ordered_vlan          = 100
                    }

                    b_end = {
                        requested_product_uid = megaport_mve.test_mve.product_uid
                        vnic_index            = 1  // Changed from 0 to 1
                        ordered_vlan          = 101
                    }
                }
                `,
					portName, locs[0],
					mveName, mveLocationID, MVEArubaImageID,
					mveName, mveName,
					vxcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check VXC was updated with new VNIC index
					resource.TestCheckResourceAttr("megaport_vxc.port_to_mve", "b_end.vnic_index", "1"),
				),
			},
			// Step 3: Plan-only to verify NO drift after VNIC index update.
			// The API may return stale VNIC index values after an update; the
			// Update handler must copy the plan value into state so the next
			// refresh does not detect a spurious change.
			{
				Config: providerConfig + fmt.Sprintf(`
                resource "megaport_port" "test_port" {
                    product_name         = "%s"
                    port_speed           = 1000
                    location_id          = %d
                    contract_term_months = 1
                    marketplace_visibility = false
                }
                resource "megaport_mve" "test_mve" {
                    product_name         = "%s"
                    location_id          = %d
                    contract_term_months = 1
                    vnics = [
                        { description = "Data Plane" },
                        { description = "Management Plane" },
                        { description = "Control Plane" }
                    ]
                    vendor_config = {
                        vendor       = "aruba"
                        product_size = "SMALL"
                        mve_label    = "MVE 2/8"
                        image_id     = %d
                        account_name = "%s-account"
                        account_key  = "%s-key"
                        system_tag   = "Preconfiguration-test-1"
                    }
                }
                resource "megaport_vxc" "port_to_mve" {
                    product_name         = "%s"
                    rate_limit           = 100
                    contract_term_months = 1
                    a_end = {
                        requested_product_uid = megaport_port.test_port.product_uid
                        ordered_vlan          = 100
                    }
                    b_end = {
                        requested_product_uid = megaport_mve.test_mve.product_uid
                        vnic_index            = 1
                        ordered_vlan          = 101
                    }
                }
                `,
					portName, locs[0],
					mveName, mveLocationID, MVEArubaImageID,
					mveName, mveName,
					vxcName),
				PlanOnly: true,
			},
		},
	})
}

// TestAccMegaportVXC_ImportDrift_Basic tests that a basic VXC import does not cause
// drift on subsequent plans. This validates the fix for GitHub Issue #263.
func TestAccMegaportVXC_ImportDrift_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocations(t, 1)
	portName1 := RandomTestName()
	portName2 := RandomTestName()
	vxcName := RandomTestName()

	vxcConfig := func() string {
		return providerConfig + fmt.Sprintf(`
			data "megaport_location" "loc" {
				id = %d
			}
			resource "megaport_port" "port_1" {
				product_name           = "%s"
				port_speed             = 1000
				location_id            = data.megaport_location.loc.id
				contract_term_months   = 1
				marketplace_visibility = false
			}
			resource "megaport_port" "port_2" {
				product_name           = "%s"
				port_speed             = 1000
				location_id            = data.megaport_location.loc.id
				contract_term_months   = 1
				marketplace_visibility = false
			}
			resource "megaport_vxc" "vxc" {
				product_name         = "%s"
				rate_limit           = 500
				contract_term_months = 1

				a_end = {
					requested_product_uid = megaport_port.port_1.product_uid
					ordered_vlan          = 100
				}

				b_end = {
					requested_product_uid = megaport_port.port_2.product_uid
					ordered_vlan          = 200
				}
			}
		`, locs[0], portName1, portName2, vxcName)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create VXC with user-only fields
			{
				Config: vxcConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.ordered_vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.ordered_vlan", "200"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "a_end.requested_product_uid"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "b_end.requested_product_uid"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
				),
			},
			// Step 2: Import the VXC
			{
				ResourceName:                         "megaport_vxc.vxc",
				ImportState:                          true,
				ImportStateVerify:                    false, // We expect differences initially
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
			},
			// Step 3: Apply the same config - this reconciles state after import
			{
				Config: vxcConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.ordered_vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.ordered_vlan", "200"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "a_end.requested_product_uid"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "b_end.requested_product_uid"),
				),
			},
			// Step 4: Plan-only to verify NO drift - this is the critical test for the fix
			{
				Config:   vxcConfig(),
				PlanOnly: true,
			},
		},
	})
}

// TestAccMegaportVXC_ImportDrift_WithPartnerConfig tests that a VXC with partner configs
// does not cause drift after import. This is the scenario from the original bug report
// where MCR VXCs with vrouter partner configs would continuously show changes.
func TestAccMegaportVXC_ImportDrift_WithPartnerConfig(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortAndMCRTestLocations(t, 1, 1000)
	mcrName := RandomTestName()
	portName := RandomTestName()
	vxcName := RandomTestName()

	vxcConfig := func() string {
		return providerConfig + fmt.Sprintf(`
			data "megaport_location" "loc" {
				id = %d
			}
			resource "megaport_mcr" "mcr" {
				product_name         = "%s"
				location_id          = data.megaport_location.loc.id
				contract_term_months = 1
				port_speed           = 1000
				asn                  = 64555
			}
			resource "megaport_port" "port" {
				product_name           = "%s"
				port_speed             = 1000
				location_id            = data.megaport_location.loc.id
				contract_term_months   = 1
				marketplace_visibility = false
			}
			resource "megaport_vxc" "vxc" {
				product_name         = "%s"
				rate_limit           = 500
				contract_term_months = 1

				a_end = {
					requested_product_uid = megaport_mcr.mcr.product_uid
					ordered_vlan          = 100
				}

				a_end_partner_config = {
					partner = "vrouter"
					vrouter_config = {
						interfaces = [{
							ip_addresses = ["10.0.0.1/30"]
							bgp_connections = [{
								peer_asn         = 64512
								local_ip_address = "10.0.0.1"
								peer_ip_address  = "10.0.0.2"
								password         = "testPassword123"
								shutdown         = false
								description      = "Test BGP Connection"
								med_in           = 100
								med_out          = 100
								bfd_enabled      = false
								export_policy    = "permit"
							}]
						}]
					}
				}

				b_end = {
					requested_product_uid = megaport_port.port.product_uid
					ordered_vlan          = 200
				}
			}
		`, locs[0], mcrName, portName, vxcName)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create VXC with MCR and vrouter partner config
			{
				Config: vxcConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.ordered_vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.ordered_vlan", "200"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end_partner_config.partner", "vrouter"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
				),
			},
			// Step 2: Import the VXC
			{
				ResourceName:                         "megaport_vxc.vxc",
				ImportState:                          true,
				ImportStateVerify:                    false, // We expect differences initially
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
			},
			// Step 3: Apply the same config - this reconciles state after import
			{
				Config: vxcConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.ordered_vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.ordered_vlan", "200"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end_partner_config.partner", "vrouter"),
				),
			},
			// Step 4: Plan-only to verify NO drift - this validates the fix
			{
				Config:   vxcConfig(),
				PlanOnly: true,
			},
		},
	})
}

// TestAccMegaportVXC_ImportDrift_WithInnerVLAN tests that a VXC with inner_vlan
// does not cause drift after import.
func TestAccMegaportVXC_ImportDrift_WithInnerVLAN(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocations(t, 1)
	portName1 := RandomTestName()
	portName2 := RandomTestName()
	vxcName := RandomTestName()

	vxcConfig := func() string {
		return providerConfig + fmt.Sprintf(`
			data "megaport_location" "loc" {
				id = %d
			}
			resource "megaport_port" "port_1" {
				product_name           = "%s"
				port_speed             = 1000
				location_id            = data.megaport_location.loc.id
				contract_term_months   = 1
				marketplace_visibility = false
			}
			resource "megaport_port" "port_2" {
				product_name           = "%s"
				port_speed             = 1000
				location_id            = data.megaport_location.loc.id
				contract_term_months   = 1
				marketplace_visibility = false
			}
			resource "megaport_vxc" "vxc" {
				product_name         = "%s"
				rate_limit           = 500
				contract_term_months = 1

				a_end = {
					requested_product_uid = megaport_port.port_1.product_uid
					ordered_vlan          = 100
					inner_vlan            = 300
				}

				b_end = {
					requested_product_uid = megaport_port.port_2.product_uid
					ordered_vlan          = 200
					inner_vlan            = 400
				}
			}
		`, locs[0], portName1, portName2, vxcName)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create VXC with inner_vlan
			{
				Config: vxcConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.ordered_vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.inner_vlan", "300"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.ordered_vlan", "200"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.inner_vlan", "400"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
				),
			},
			// Step 2: Import the VXC
			{
				ResourceName:                         "megaport_vxc.vxc",
				ImportState:                          true,
				ImportStateVerify:                    false, // We expect differences initially
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
			},
			// Step 3: Apply the same config - this reconciles state after import
			{
				Config: vxcConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.ordered_vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.inner_vlan", "300"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.ordered_vlan", "200"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.inner_vlan", "400"),
				),
			},
			// Step 4: Plan-only to verify NO drift - this validates the fix
			{
				Config:   vxcConfig(),
				PlanOnly: true,
			},
		},
	})
}

// TestAccMegaportVXC_ImportDrift_AWSHostedConnection tests the exact scenario from
// GitHub Issue #263: an AWS Hosted Connection VXC with b_end_partner_config continuously
// shows changes after import. This is the primary bug that was reported.
func TestAccMegaportVXC_ImportDrift_AWSHostedConnection(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocationsWithPartner(t, 1, "AWSHC")
	portName := RandomTestName()
	vxcName := RandomTestName()

	vxcConfig := func() string {
		return providerConfig + fmt.Sprintf(`
			data "megaport_location" "loc" {
				id = %d
			}

			data "megaport_partner" "aws_port" {
				connect_type = "AWSHC"
				location_id  = data.megaport_location.loc.id
			}

			resource "megaport_port" "port" {
				product_name           = "%s"
				port_speed             = 1000
				location_id            = data.megaport_location.loc.id
				contract_term_months   = 1
				marketplace_visibility = false
			}

			resource "megaport_vxc" "vxc" {
				product_name         = "%s"
				rate_limit           = 500
				contract_term_months = 1

				a_end = {
					requested_product_uid = megaport_port.port.product_uid
					ordered_vlan          = 200
				}

				b_end = {
					requested_product_uid = data.megaport_partner.aws_port.product_uid
				}

				b_end_partner_config = {
					partner = "aws"
					aws_config = {
						name          = "%s"
						type          = "private"
						connect_type  = "AWSHC"
						owner_account = "123456789012"
					}
				}
			}
		`, locs[0], portName, vxcName, vxcName)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create AWS Hosted Connection VXC
			{
				Config: vxcConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.ordered_vlan", "200"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end_partner_config.partner", "aws"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end_partner_config.aws_config.name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end_partner_config.aws_config.connect_type", "AWSHC"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
				),
			},
			// Step 2: Import the VXC (simulates the bug scenario)
			{
				ResourceName:                         "megaport_vxc.vxc",
				ImportState:                          true,
				ImportStateVerify:                    false, // We expect differences initially
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
			},
			// Step 3: Apply the same config - first apply after import
			{
				Config: vxcConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.ordered_vlan", "200"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end_partner_config.partner", "aws"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end_partner_config.aws_config.name", vxcName),
				),
			},
			// Step 4: Plan-only to verify NO drift - THIS IS THE BUG FIX VALIDATION
			// Before the fix, this would fail because the plan would show changes
			// for b_end_partner_config even though nothing changed.
			{
				Config:   vxcConfig(),
				PlanOnly: true,
			},
		},
	})
}

// TestAccMegaportVXC_ImportDrift_WithVnicIndex tests that a VXC connected to an
// MVE with a vnic_index does not cause drift after import. The API does not
// return the user-configured vnic_index on read, so the provider must preserve
// it from state/plan to avoid an infinite update loop.
func TestAccMegaportVXC_ImportDrift_WithVnicIndex(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocations(t, 1)
	mveLocationID, _ := findMVETestLocation(t, 2)
	portName := RandomTestName()
	mveName := RandomTestName()
	vxcName := RandomTestName()

	vxcConfig := func() string {
		return providerConfig + fmt.Sprintf(`
			resource "megaport_port" "port" {
				product_name           = "%s"
				port_speed             = 1000
				location_id            = %d
				contract_term_months   = 1
				marketplace_visibility = false
			}

			resource "megaport_mve" "mve" {
				product_name         = "%s"
				location_id          = %d
				contract_term_months = 1

				vnics = [
					{ description = "Data Plane" },
					{ description = "Management Plane" },
					{ description = "Control Plane" }
				]

				vendor_config = {
					vendor       = "aruba"
					product_size = "SMALL"
					mve_label    = "MVE 2/8"
					image_id     = %d
					account_name = "%s-account"
					account_key  = "%s-key"
					system_tag   = "Preconfiguration-drift-test"
				}
			}

			resource "megaport_vxc" "vxc" {
				product_name         = "%s"
				rate_limit           = 100
				contract_term_months = 1

				a_end = {
					requested_product_uid = megaport_port.port.product_uid
					ordered_vlan          = 100
				}

				b_end = {
					requested_product_uid = megaport_mve.mve.product_uid
					vnic_index            = 1
					ordered_vlan          = 101
				}
			}
		`, portName, locs[0],
			mveName, mveLocationID, MVEArubaImageID,
			mveName, mveName,
			vxcName)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create Port, MVE, and VXC with vnic_index
			{
				Config: vxcConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.vnic_index", "1"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.ordered_vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.ordered_vlan", "101"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
				),
			},
			// Step 2: Import the VXC (vnic_index will be lost from state)
			{
				ResourceName:                         "megaport_vxc.vxc",
				ImportState:                          true,
				ImportStateVerify:                    false,
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
			},
			// Step 3: Apply the same config - reconciles state after import
			{
				Config: vxcConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.vnic_index", "1"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end.ordered_vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "b_end.ordered_vlan", "101"),
				),
			},
			// Step 4: Plan-only to verify NO drift on vnic_index
			{
				Config:   vxcConfig(),
				PlanOnly: true,
			},
		},
	})
}
