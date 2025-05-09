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
type MCRVXCBGPCSPTestSuite ProviderTestSuite
type FullEcosystemTestSuite ProviderTestSuite

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

func TestVXCBGPCSPTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRVXCBGPCSPTestSuite))
}

func TestFullEcosystemTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(FullEcosystemTestSuite))
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

func (suite *MCRVXCBGPCSPTestSuite) TestAccMegaportMCRVXCBasicWithBGP_Basic() {
	mcrName := RandomTestName()
	vxcName1 := RandomTestName()
	prefixFilterListName := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
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
					company_name = "AWS"
					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
					location_id  = data.megaport_location.loc2.id
				  }

				  resource "megaport_mcr" "mcr" {
					product_name            = "%s"
					location_id             = data.megaport_location.loc1.id
					contract_term_months    = 1
					port_speed              = 1000
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

				  resource "megaport_vxc_basic" "aws_vxc" {
					product_name           = "%s"
					rate_limit             = 100
					contract_term_months   = 1

					a_end = {
                      requested_product_uid = megaport_mcr.mcr.product_uid
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
                  `, VXCLocationID1, VXCLocationID2, mcrName, prefixFilterListName, vxcName1, vxcName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc_basic.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.aws_vxc", "b_end_partner_config.aws_config.name", vxcName1),
					resource.TestCheckResourceAttr("megaport_vxc_basic.aws_vxc", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.aws_vxc", "resource_tags.key2", "value2"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc_basic.aws_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc_basic.aws_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config"},
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
					company_name = "AWS"
					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
					location_id  = data.megaport_location.loc2.id
				  }

				  resource "megaport_mcr" "mcr" {
					product_name            = "%s"
					location_id             = data.megaport_location.loc1.id
					contract_term_months    = 1
					port_speed              = 1000
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

				  resource "megaport_vxc_basic" "aws_vxc" {
					product_name           = "%s"
					rate_limit             = 100
					contract_term_months   = 1

					a_end = {
                      requested_product_uid = megaport_mcr.mcr.product_uid
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
                  `, VXCLocationID1, VXCLocationID2, mcrName, vxcName1, prefixFilterListName, vxcName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc_basic.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.aws_vxc", "b_end_partner_config.aws_config.name", vxcName1),
					resource.TestCheckResourceAttr("megaport_vxc_basic.aws_vxc", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.aws_vxc", "resource_tags.key2updated", "value2updated"),
				),
			},
		},
	})
}

func (suite *FullEcosystemTestSuite) TestFullEcosystem() {
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
					port_speed              = 1000
					location_id             = data.megaport_location.loc1.id
					contract_term_months    = 1
					asn                      = 64555
				  }

				  resource "megaport_vxc_basic" "port_vxc" {
					product_name           = "%s"
					rate_limit             = 100
					contract_term_months   = 12

					a_end = {
					  requested_product_uid = megaport_port.port.product_uid
					  vlan = 101
					}

					b_end = {
					  requested_product_uid = megaport_lag_port.lag_port.product_uid
					  vlan = 102
					}

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}
				  }

				  resource "megaport_vxc_basic" "mcr_vxc" {
					product_name           = "%s"
					rate_limit             = 100
					contract_term_months   = 12

					a_end = {
					  requested_product_uid = megaport_port.port.product_uid
					  vlan = 181
					}

					b_end = {
					  requested_product_uid = megaport_mcr.mcr.product_uid
					}

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}
				  }

				  resource "megaport_vxc_basic" "aws_vxc" {
					product_name            = "%s"
					rate_limit              = 100
					contract_term_months    = 1

					a_end = {
					  requested_product_uid = megaport_mcr.mcr.product_uid
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

				  resource "megaport_vxc_basic" "gcp_vxc" {
					product_name            = "%s"
					rate_limit              = 100
					contract_term_months    = 12

					a_end = {
					  requested_product_uid = megaport_mcr.mcr.product_uid
					}

					b_end = {}

					b_end_partner_config = {
					  partner = "google"
					  google_config = {
						pairing_key = "%s"
					  }
					}
				  }

				  resource "megaport_vxc_basic" "azure_vxc" {
					product_name            = "%s"
					rate_limit              = 100
					contract_term_months    = 12

					a_end = {
					  requested_product_uid = megaport_mcr.mcr.product_uid
					}

					b_end = {}

					b_end_partner_config = {
					  partner = "azure"
					  azure_config = {
					    port_choice = "primary"
						service_key = "%s"
					  }
					}
				  }
                  `, VXCLocationID1, VXCLocationID2, VXCLocationID3, lagPortName, costCentreName, portName, costCentreName, mcrName, portVXCName, mcrVXCName, awsVXCName, awsVXCName, gcpVXCName, GooglePairingKey, azureVXCName, AzureServiceKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc_basic.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.aws_vxc", "b_end_partner_config.aws_config.name", awsVXCName),
					resource.TestCheckResourceAttr("megaport_vxc_basic.port_vxc", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.port_vxc", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.mcr_vxc", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_vxc_basic.mcr_vxc", "resource_tags.key2", "value2"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc_basic.aws_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc_basic.aws_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config"},
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc_basic.gcp_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc_basic.gcp_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config"},
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc_basic.azure_vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc_basic.azure_vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status", "a_end.requested_product_uid", "b_end.requested_product_uid", "a_end_partner_config", "b_end_partner_config"},
			},
		},
	})
}
