package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
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
				data "megaport_location" "bne_nxt1" {
					name = "NextDC B1"
				}
					resource "megaport_port" "port" {
			        product_name  = "%s"
			        port_speed  = 1000
					cost_centre = "%s"
			        location_id = data.megaport_location.bne_nxt1.id
			        contract_term_months        = 12
					marketplace_visibility = true
					diversity_zone = "red"
					resource_tags = [{"key" = "test-key-1", "value" = "test-value-1"}]
			      }`, portName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "product_name", portName),
					resource.TestCheckResourceAttr("megaport_port.port", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port", "marketplace_visibility", "true"),
					resource.TestCheckResourceAttr("megaport_port.port", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_port.port", "diversity_zone", "red"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "company_uid"),
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
				data "megaport_location" "bne_nxt1" {
					name = "NextDC B1"
				}
					resource "megaport_port" "port" {
			        product_name  = "%s"
			        port_speed  = 1000
					cost_centre = "%s"
			        location_id = data.megaport_location.bne_nxt1.id
			        contract_term_months        = 12
					marketplace_visibility = false
					diversity_zone = "red"
					resource_tags = [{"key" = "test-key-1", "value" = "test-value-1"}]
			      }`, portNameNew, costCentreNameNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "product_name", portNameNew),
					resource.TestCheckResourceAttr("megaport_port.port", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_port.port", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_port.port", "diversity_zone", "red"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "company_uid"),
				),
			},
		},
	})
}
