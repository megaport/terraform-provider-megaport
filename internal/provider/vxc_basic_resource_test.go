package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

const (
	VXCBasicPortTestLocation   = "NextDC M1"
	VXCBasicPortTestLocationID = 4 // "NextDC M1"
)

type BasicVXCProviderTestSuite ProviderTestSuite
type MCRVLANValidationAEndTestSuite ProviderTestSuite
type MCRVLANValidationBEndTestSuite ProviderTestSuite
type MCRVLANValidationNullTestSuite ProviderTestSuite
type MVEVLANValidationAEndTestSuite ProviderTestSuite
type MVEVLANValidationBEndTestSuite ProviderTestSuite
type MVEVLANValidationWithVNICTestSuite ProviderTestSuite
type MVEVLANValidationNullTestSuite ProviderTestSuite
type MVEVNICIndexValidationAEndTestSuite ProviderTestSuite
type MVEVNICIndexValidationBEndTestSuite ProviderTestSuite
type MCRVLANModificationBEndTestSuite ProviderTestSuite
type MVEVLANModificationAEndTestSuite ProviderTestSuite

func TestBasicVXCProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(BasicVXCProviderTestSuite))
}

func TestMCRVLANValidationAEndTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRVLANValidationAEndTestSuite))
}

func TestMCRVLANValidationBEndTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRVLANValidationBEndTestSuite))
}

func TestMCRVLANValidationNullTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRVLANValidationNullTestSuite))
}

func TestMVEVLANValidationAEndTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MVEVLANValidationAEndTestSuite))
}

func TestMVEVLANValidationBEndTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MVEVLANValidationBEndTestSuite))
}

func TestMVEVLANValidationWithVNICTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MVEVLANValidationWithVNICTestSuite))
}

func TestMVEVLANValidationNullTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MVEVLANValidationNullTestSuite))
}

func TestAccMVEVNICIndexValidationAEndTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MVEVNICIndexValidationAEndTestSuite))
}

func TestAccMVEVNICIndexValidationBEndTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MVEVNICIndexValidationBEndTestSuite))
}

func TestAccVXCResourceWithMCRBEndVLANModification(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRVLANModificationBEndTestSuite))
}

func TestAccVXCResourceWithMVEAEndVLANModification(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MVEVLANModificationAEndTestSuite))
}

func (suite *BasicVXCProviderTestSuite) TestAccMegaportBasicVXC_Basic() {
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
						vlan = 100
						inner_vlan = 300
                    }

                    b_end = {
                        requested_product_uid = megaport_port.port_2.product_uid
						vlan = 101
						inner_vlan = 301
                    }
                  }
                  `, VXCBasicPortTestLocationID, portName1, portName2, portName3, portName4, vxcName, costCentreName),
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
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "b_end.vlan", "101"),
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
				ImportStateVerifyIgnore: []string{"last_updated", "a_end.vlan", "b_end.vlan", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config", "provisioning_status"},
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
						vlan = 100
						inner_vlan = 300
			        }

			        b_end = {
			            requested_product_uid = megaport_port.port_4.product_uid
						vlan = 101
						inner_vlan = 301
			        }
			      }
			      `, VXCBasicPortTestLocationID, portName1, portName2, portName3, portName4, vxcName, costCentreName),
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
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "b_end.vlan", "101"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "b_end.vlan", "101"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.inner_vlan", "300"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "b_end.inner_vlan", "301"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "resource_tags.key2", "value2"),
				),
			},
			// Update Test 2 - Change Name/Cost Centre/Rate Limit/Contract contract_term_months   /VLAN/Inner VLAN/Resource Tags
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
						vlan = 200
						inner_vlan = 400
			        }

			        b_end = {
			            requested_product_uid = megaport_port.port_4.product_uid
						vlan = 201
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
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.vlan", "200"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.vlan", "200"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "b_end.vlan", "201"),
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

func (suite *MCRVLANValidationAEndTestSuite) TestAccMegaportBasicVXC_MCRVLANValidation_AEnd() {
	portName2 := RandomTestName()
	mcrName := RandomTestName()
	vxcName := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "port_loc" {
					id = %d
				}
				data "megaport_location" "mcr_loc" {
					id = %d
				}

				resource "megaport_mcr" "mcr_1" {
					product_name         = "%s"
					location_id          = data.megaport_location.mcr_loc.id
					contract_term_months          = 12
					port_speed           = 1000
					asn                  = 64513
				}
				resource "megaport_port" "port_2" {
					product_name  = "%s"
					port_speed    = 1000
					location_id   = data.megaport_location.port_loc.id
					contract_term_months = 12
					marketplace_visibility = false
				}
				resource "megaport_vxc_basic" "vxc" {
					product_name   = "%s"
					rate_limit = 500
					contract_term_months = 12

					a_end = {
						requested_product_uid = megaport_mcr.mcr_1.product_uid
						vlan = 100
					}

					b_end = {
						requested_product_uid = megaport_port.port_2.product_uid
						vlan = 101
					}
				}
				`, VXCBasicPortTestLocationID, MCRTestLocationIDNum, mcrName, portName2, vxcName),
				ExpectError: regexp.MustCompile(`Error running apply`),
			},
		},
	})
}

func (suite *MCRVLANModificationBEndTestSuite) TestAccVXCResourceWithMCRBEndVLANModification() {
	portName1 := RandomTestName()
	mcrName := RandomTestName()
	vxcName := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "megaport_port" "test1" {
  product_name           = "%s"
  port_speed             = 1000
  location_id            = %d
  contract_term_months   = 1
  marketplace_visibility = false
}

resource "megaport_mcr" "test" {
  product_name           = "%s"
  location_id            = %d
  contract_term_months   = 1
  port_speed             = 1000
}

resource "megaport_vxc_basic" "test" {
  product_name           = "%s"
  rate_limit             = 1000
  contract_term_months   = 1

  a_end = {
    requested_product_uid = megaport_port.test1.product_uid
    vlan                  = 200
  }

  b_end = {
    requested_product_uid = megaport_mcr.test.product_uid
  }
}
`, portName1, VXCBasicPortTestLocationID, mcrName, MCRTestLocationID, vxcName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc_basic.test", "product_name", vxcName),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "megaport_port" "test1" {
  product_name           = "%s"
  port_speed             = 1000
  location_id            = %d
  contract_term_months   = 1
  marketplace_visibility = false
}

resource "megaport_mcr" "test" {
  product_name           = "%s"
  location_id            = %d
  contract_term_months   = 1
  port_speed             = 1000
}

resource "megaport_vxc_basic" "test" {
  product_name           = "%s"
  rate_limit             = 1000
  contract_term_months   = 1

  a_end = {
    requested_product_uid = megaport_port.test1.product_uid
    vlan                  = 200
  }

  b_end = {
    requested_product_uid = megaport_mcr.test.product_uid
    vlan                  = 100
  }
}
`, portName1, VXCBasicPortTestLocationID, mcrName, MCRTestLocationID, vxcName),
				ExpectError: regexp.MustCompile("Error running apply"),
			},
		},
	})
}

func (suite *MCRVLANValidationBEndTestSuite) TestAccMegaportBasicVXC_MCRVLANValidation_BEnd() {
	portName1 := RandomTestName()
	mcrName := RandomTestName()
	vxcName := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "port_loc" {
					id = %d
				}
				data "megaport_location" "mcr_loc" {
					id = %d
				}
				resource "megaport_port" "port_1" {
					product_name  = "%s"
					port_speed    = 1000
					location_id   = data.megaport_location.port_loc.id
					contract_term_months = 12
					marketplace_visibility = false
				}
				resource "megaport_mcr" "mcr_2" {
					product_name         = "%s"
					location_id          = data.megaport_location.mcr_loc.id
					contract_term_months          = 12
					port_speed           = 1000
					asn                  = 64513
				}
				resource "megaport_vxc_basic" "vxc" {
					product_name   = "%s"
					rate_limit = 500
					contract_term_months = 12

					a_end = {
						requested_product_uid = megaport_port.port_1.product_uid
						vlan = 100
					}

					b_end = {
						requested_product_uid = megaport_mcr.mcr_2.product_uid
						vlan = 101
					}
				}
				`, VXCBasicPortTestLocationID, MCRTestLocationIDNum, portName1, mcrName, vxcName),
				ExpectError: regexp.MustCompile(`Error running apply`),
			},
		},
	})
}

func (suite *MCRVLANValidationNullTestSuite) TestAccMegaportBasicVXC_MCRVLANValidation_Null() {
	portName1 := RandomTestName()
	mcrName := RandomTestName()
	vxcName := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "port_loc" {
					id = %d
				}
				data "megaport_location" "mcr_loc" {
					id = %d
				}
				resource "megaport_port" "port_1" {
					product_name  = "%s"
					port_speed    = 1000
					location_id   = data.megaport_location.port_loc.id
					contract_term_months = 12
					marketplace_visibility = false
				}
				resource "megaport_mcr" "mcr_2" {
					product_name         = "%s"
					location_id          = data.megaport_location.mcr_loc.id
					contract_term_months          = 12
					port_speed           = 1000
					asn                  = 64513
				}
				resource "megaport_vxc_basic" "vxc" {
					product_name   = "%s"
					rate_limit = 500
					contract_term_months = 12

					a_end = {
						requested_product_uid = megaport_port.port_1.product_uid
						vlan = 100
					}

					b_end = {
						requested_product_uid = megaport_mcr.mcr_2.product_uid
					}
				}
				`, VXCBasicPortTestLocationID, MCRTestLocationIDNum, portName1, mcrName, vxcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.vlan", "100"),
				),
			},
		},
	})
}

func (suite *MVEVLANValidationAEndTestSuite) TestAccMegaportBasicVXC_MVEVLANValidation_AEnd() {
	portName2 := RandomTestName()
	mveName := RandomTestName()
	vxcName := RandomTestName()
	mveKey := "notARealKey"

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "port_loc" {
					id = %d
				}
				data "megaport_location" "mve_loc" {
					id = %d
				}

				data "megaport_mve_images" "aruba" {
  					vendor_filter = "Aruba"
  					id_filter = 23
				}
                
				resource "megaport_mve" "mve_1" {
                    product_name  = "%s"
                    location_id = data.megaport_location.mve_loc.id
                    contract_term_months        = 1
					diversity_zone = "red"

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "MEDIUM"
                        image_id = data.megaport_mve_images.aruba.mve_images.0.id
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
                    }

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}
					]
                  }

				resource "megaport_port" "port_2" {
					product_name  = "%s"
					port_speed    = 1000
					location_id   = data.megaport_location.port_loc.id
					contract_term_months = 12
					marketplace_visibility = false
				}
				resource "megaport_vxc_basic" "vxc" {
					product_name   = "%s"
					rate_limit = 500
					contract_term_months = 12

					a_end = {
						requested_product_uid = megaport_mve.mve_1.product_uid
						vlan = 100
					}

					b_end = {
						requested_product_uid = megaport_port.port_2.product_uid
						vlan = 101
					}
				}
				`, VXCBasicPortTestLocationID, MVETestLocationIDNum, mveName, mveName, mveKey, portName2, vxcName),
				ExpectError: regexp.MustCompile(`Error running apply`),
			},
		},
	})
}

func (suite *MVEVLANValidationBEndTestSuite) TestAccMegaportBasicVXC_MVEVLANValidation_BEnd() {
	portName1 := RandomTestName()
	mveName := RandomTestName()
	vxcName := RandomTestName()
	mveKey := "notARealKey"

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "port_loc" {
					id = %d
				}
				data "megaport_location" "mve_loc" {
					id = %d
				}
				resource "megaport_port" "port_1" {
					product_name  = "%s"
					port_speed    = 1000
					location_id   = data.megaport_location.port_loc.id
					contract_term_months = 12
					marketplace_visibility = false
				}

				data "megaport_mve_images" "aruba" {
  					vendor_filter = "Aruba"
  					id_filter = 23
				}
                
				resource "megaport_mve" "mve_2" {
                    product_name  = "%s"
                    location_id = data.megaport_location.mve_loc.id
                    contract_term_months        = 1
					diversity_zone = "red"

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "MEDIUM"
                        image_id = data.megaport_mve_images.aruba.mve_images.0.id
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
                    }

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}
					]
                  }
				
				resource "megaport_vxc_basic" "vxc" {
					product_name   = "%s"
					rate_limit = 500
					contract_term_months = 12

					a_end = {
						requested_product_uid = megaport_port.port_1.product_uid
						vlan = 100
					}

					b_end = {
						requested_product_uid = megaport_mve.mve_2.product_uid
						vlan = 101
					}
				}
				`, VXCBasicPortTestLocationID, MVETestLocationIDNum, portName1, mveName, mveName, mveKey, vxcName),
				ExpectError: regexp.MustCompile(`Error running apply`),
			},
		},
	})
}

func (suite *MVEVLANValidationWithVNICTestSuite) TestAccMegaportBasicVXC_MVEVLANValidation_WithVNIC() {
	portName1 := RandomTestName()
	mveName := RandomTestName()
	vxcName := RandomTestName()
	mveKey := "notARealKey"

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "port_loc" {
					id = %d
				}
				data "megaport_location" "mve_loc" {
					id = %d
				}
				resource "megaport_port" "port_1" {
					product_name  = "%s"
					port_speed    = 1000
					location_id   = data.megaport_location.port_loc.id
					contract_term_months = 12
					marketplace_visibility = false
				}

				data "megaport_mve_images" "aruba" {
  					vendor_filter = "Aruba"
  					id_filter = 23
				}

                resource "megaport_mve" "mve_2" {
                    product_name  = "%s"
                    location_id = data.megaport_location.mve_loc.id
                    contract_term_months        = 1
					diversity_zone = "red"

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "MEDIUM"
                        image_id = data.megaport_mve_images.aruba.mve_images.0.id
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
                    }

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}
					]
                  }
				
				resource "megaport_vxc_basic" "vxc" {
					product_name   = "%s"
					rate_limit = 500
					contract_term_months = 12

					a_end = {
						requested_product_uid = megaport_port.port_1.product_uid
						vlan = 100
					}

					b_end = {
						requested_product_uid = megaport_mve.mve_2.product_uid
						vnic_index = 1
					}
				}
				`, VXCBasicPortTestLocationID, MVETestLocationIDNum, portName1, mveName, mveName, mveKey, vxcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.vlan", "100"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "b_end.vnic_index", "1"),
				),
			},
		},
	})
}

func (suite *MVEVLANValidationNullTestSuite) TestAccMegaportBasicVXC_MVEVLANValidation_Null() {
	portName1 := RandomTestName()
	mveName := RandomTestName()
	vxcName := RandomTestName()
	mveKey := "notARealKey"

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "port_loc" {
					id = %d
				}
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_port" "port_1" {
					product_name  = "%s"
					port_speed    = 1000
					location_id   = data.megaport_location.port_loc.id
					contract_term_months = 12
					marketplace_visibility = false
				}

				data "megaport_mve_images" "aruba" {
  					vendor_filter = "Aruba"
  					id_filter = 23
				}
                
				resource "megaport_mve" "mve_2" {
                    product_name  = "%s"
                    location_id = data.megaport_location.test_location.id
                    contract_term_months        = 1
					diversity_zone = "red"

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "MEDIUM"
                        image_id = data.megaport_mve_images.aruba.mve_images.0.id
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
                    }

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}
					]
                  }
				
				resource "megaport_vxc_basic" "vxc" {
					product_name   = "%s"
					rate_limit = 500
					contract_term_months = 12

					a_end = {
						requested_product_uid = megaport_port.port_1.product_uid
						vlan = 100
					}

					b_end = {
						requested_product_uid = megaport_mve.mve_2.product_uid
						vnic_index = 0
					}
				}
				`, VXCBasicPortTestLocationID, MVETestLocationIDNum, portName1, mveName, mveName, mveKey, vxcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc_basic.vxc", "a_end.vlan", "100"),
				),
			},
		},
	})
}

func (suite *MVEVNICIndexValidationAEndTestSuite) TestAccMegaportBasicVXC_MVEVNICIndexValidation_AEnd() {
	portName2 := RandomTestName()
	mveName := RandomTestName()
	vxcName := RandomTestName()
	mveKey := "notARealKey"

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "port_loc" {
					id = %d
				}
				data "megaport_location" "mve_loc" {
					id = %d
				}

				data "megaport_mve_images" "aruba" {
  					vendor_filter = "Aruba"
  					id_filter = 23
				}
                
				resource "megaport_mve" "mve_1" {
                    product_name  = "%s"
                    location_id = data.megaport_location.mve_loc.id
                    contract_term_months = 1
					diversity_zone = "red"

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "MEDIUM"
                        image_id = data.megaport_mve_images.aruba.mve_images.0.id
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
                    }

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}
					]
                  }

				resource "megaport_port" "port_2" {
					product_name  = "%s"
					port_speed    = 1000
					location_id   = data.megaport_location.port_loc.id
					contract_term_months = 12
					marketplace_visibility = false
				}
				resource "megaport_vxc_basic" "vxc" {
					product_name   = "%s"
					rate_limit = 500
					contract_term_months = 12

					a_end = {
						requested_product_uid = megaport_mve.mve_1.product_uid
						# Missing vnic_index which should cause error
					}

					b_end = {
						requested_product_uid = megaport_port.port_2.product_uid
						vlan = 101
					}
				}
				`, VXCBasicPortTestLocationID, MVETestLocationIDNum, mveName, mveName, mveKey, portName2, vxcName),
				ExpectError: regexp.MustCompile(`Error running apply`),
			},
		},
	})
}

func (suite *MVEVNICIndexValidationBEndTestSuite) TestAccMegaportBasicVXC_MVEVNICIndexValidation_BEnd() {
	portName1 := RandomTestName()
	mveName := RandomTestName()
	vxcName := RandomTestName()
	mveKey := "notARealKey"

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "port_loc" {
					id = %d
				}
				data "megaport_location" "mve_loc" {
					id = %d
				}
				resource "megaport_port" "port_1" {
					product_name  = "%s"
					port_speed    = 1000
					location_id   = data.megaport_location.port_loc.id
					contract_term_months = 12
					marketplace_visibility = false
				}

				data "megaport_mve_images" "aruba" {
  					vendor_filter = "Aruba"
  					id_filter = 23
				}

                resource "megaport_mve" "mve_2" {
                    product_name  = "%s"
                    location_id = data.megaport_location.mve_loc.id
                    contract_term_months = 1
					diversity_zone = "red"

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "MEDIUM"
                        image_id = data.megaport_mve_images.aruba.mve_images.0.id
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
                    }

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}
					]
                  }
				
				resource "megaport_vxc_basic" "vxc" {
					product_name   = "%s"
					rate_limit = 500
					contract_term_months = 12

					a_end = {
						requested_product_uid = megaport_port.port_1.product_uid
						vlan = 100
					}

					b_end = {
						requested_product_uid = megaport_mve.mve_2.product_uid
						# Missing vnic_index which should cause error
					}
				}
				`, VXCBasicPortTestLocationID, MVETestLocationIDNum, portName1, mveName, mveName, mveKey, vxcName),
				ExpectError: regexp.MustCompile(`Error running apply`),
			},
		},
	})
}

func (suite *MVEVLANModificationAEndTestSuite) TestAccVXCResourceWithMVEAEndVLANModification() {
	portName2 := RandomTestName()
	mveName := RandomTestName()
	vxcName := RandomTestName()
	mveKey := "notARealKey"

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "port_loc" {
					id = %d
				}
				data "megaport_location" "mve_loc" {
					id = %d
				}
				resource "megaport_port" "port_2" {
					product_name  = "%s"
					port_speed    = 1000
					location_id   = data.megaport_location.port_loc.id
					contract_term_months = 12
					marketplace_visibility = false
				}

				data "megaport_mve_images" "aruba" {
					vendor_filter = "Aruba"
					id_filter = 23
				}

				resource "megaport_mve" "mve_1" {
					product_name  = "%s"
					location_id = data.megaport_location.mve_loc.id
					contract_term_months = 1
					diversity_zone = "red"

					vendor_config = {
						vendor = "aruba"
						product_size = "MEDIUM"
						image_id = data.megaport_mve_images.aruba.mve_images.0.id
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
					}

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}
					]
				}

				resource "megaport_vxc_basic" "test" {
					product_name         = "%s"
					rate_limit           = 1000
					contract_term_months = 1

					a_end = {
						requested_product_uid = megaport_mve.mve_1.product_uid
						vnic_index            = 0
					}

					b_end = {
						requested_product_uid = megaport_port.port_2.product_uid
						vlan                  = 200
					}
				}
				`, VXCBasicPortTestLocationID, MVETestLocationIDNum, portName2, mveName, mveName, mveKey, vxcName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_vxc_basic.test", "product_name", vxcName),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "port_loc" {
					id = %d
				}
				data "megaport_location" "mve_loc" {
					id = %d
				}
				resource "megaport_port" "port_2" {
					product_name  = "%s"
					port_speed    = 1000
					location_id   = data.megaport_location.port_loc.id
					contract_term_months = 12
					marketplace_visibility = false
				}

				data "megaport_mve_images" "aruba" {
					vendor_filter = "Aruba"
					id_filter = 23
				}

				resource "megaport_mve" "mve_1" {
					product_name  = "%s"
					location_id = data.megaport_location.mve_loc.id
					contract_term_months = 1
					diversity_zone = "red"

					vendor_config = {
						vendor = "aruba"
						product_size = "MEDIUM"
						image_id = data.megaport_mve_images.aruba.mve_images.0.id
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
					}

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}
					]
				}

				resource "megaport_vxc_basic" "test" {
					product_name         = "%s"
					rate_limit           = 1000
					contract_term_months = 1

					a_end = {
						requested_product_uid = megaport_mve.mve_1.product_uid
						vnic_index            = 0
						vlan                  = 100
					}

					b_end = {
						requested_product_uid = megaport_port.port_2.product_uid
						vlan                  = 200
					}
				}
				`, VXCBasicPortTestLocationID, MVETestLocationIDNum, portName2, mveName, mveName, mveKey, vxcName),
				ExpectError: regexp.MustCompile("Error running apply"),
			},
		},
	})
}
