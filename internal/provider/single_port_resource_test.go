package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMegaportSinglePort_Basic(t *testing.T) {
	portName := RandomTestName()
	resource.Test(t, resource.TestCase{
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
			        location_id = data.megaport_location.bne_nxt1.id
			        contract_term_months        = 1
					market = "AU"
					marketplace_visibility = false
			      }`, portName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "product_name", portName),
					resource.TestCheckResourceAttr("megaport_port.port", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_port.port", "market", "AU"),
					resource.TestCheckResourceAttr("megaport_port.port", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "product_uid"),
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
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
		},
	})
}
