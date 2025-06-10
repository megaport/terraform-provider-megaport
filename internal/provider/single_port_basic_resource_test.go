package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

type SinglePortBasicProviderTestSuite ProviderTestSuite

func TestSingleBasicPortProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SinglePortBasicProviderTestSuite))
}

func (suite *SinglePortBasicProviderTestSuite) TestAccMegaportSinglePort_Basic() {
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
					resource "megaport_port_basic" "port" {
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
			      }`, SinglePortTestLocationIDNum, portName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port_basic.port", "product_name", portName),
					resource.TestCheckResourceAttr("megaport_port_basic.port", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port_basic.port", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port_basic.port", "marketplace_visibility", "true"),
					resource.TestCheckResourceAttr("megaport_port_basic.port", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_port_basic.port", "diversity_zone", "red"),
					resource.TestCheckResourceAttr("megaport_port_basic.port", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_port_basic.port", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_port_basic.port", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_port_basic.port", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_port_basic.port", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_port_basic.port", "company_uid"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_port_basic.port",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_port_basic.port"
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
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
					resource "megaport_port_basic" "port" {
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
			      }`, SinglePortTestLocationIDNum, portNameNew, costCentreNameNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port_basic.port", "product_name", portNameNew),
					resource.TestCheckResourceAttr("megaport_port_basic.port", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port_basic.port", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port_basic.port", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_port_basic.port", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_port_basic.port", "diversity_zone", "red"),
					resource.TestCheckResourceAttr("megaport_port_basic.port", "resource_tags.key1-updated", "value1-updated"),
					resource.TestCheckResourceAttr("megaport_port_basic.port", "resource_tags.key2-updated", "value2-updated"),
					resource.TestCheckResourceAttrSet("megaport_port_basic.port", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_port_basic.port", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_port_basic.port", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_port_basic.port", "company_uid"),
				),
			},
		},
	})
}
