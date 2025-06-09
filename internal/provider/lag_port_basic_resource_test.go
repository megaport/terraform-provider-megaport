package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

type LagPortBasicProviderTestSuite ProviderTestSuite

func TestLagPortBasicProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(LagPortBasicProviderTestSuite))
}

func (suite *LagPortBasicProviderTestSuite) TestAccMegaportLAGPort_Basic() {
	portName := RandomTestName()
	costCentreName := RandomTestName()
	portNameNew := RandomTestName()
	costCentreNameNew := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
					resource "megaport_lag_port_basic" "lag_port" {
			        product_name  = "%s"
					cost_centre = "%s"
			        port_speed  = 10000
			        location_id = data.megaport_location.test_location.id
			        contract_term_months        = 12
					marketplace_visibility = true
                    lag_count = 1
					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}
			      }`, LagPortTestLocationIDNum, portName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "product_name", portName),
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "port_speed", "10000"),
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "marketplace_visibility", "true"),
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "lag_count", "1"),
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_lag_port_basic.lag_port", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_lag_port_basic.lag_port", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_lag_port_basic.lag_port", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_lag_port_basic.lag_port", "company_uid"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_lag_port_basic.lag_port",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_lag_port_basic.lag_port"
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
				ImportStateVerifyIgnore: []string{"lag_count", "lag_port_uids"},
			},
			// Update Testing
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
					resource "megaport_lag_port_basic" "lag_port" {
			        product_name  = "%s"
					cost_centre = "%s"
			        port_speed  = 10000
			        location_id = data.megaport_location.test_location.id
			        contract_term_months        = 12
					marketplace_visibility = false
                    lag_count = 1
					resource_tags = {
						"key1updated" = "value1updated"
						"key2updated" = "value2updated"
			 	  	}
			      }`, LagPortTestLocationIDNum, portNameNew, costCentreNameNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "product_name", portNameNew),
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "port_speed", "10000"),
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "lag_count", "1"),
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_lag_port_basic.lag_port", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_lag_port_basic.lag_port", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_lag_port_basic.lag_port", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_lag_port_basic.lag_port", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_lag_port_basic.lag_port", "company_uid"),
				),
			},
		},
	})
}
