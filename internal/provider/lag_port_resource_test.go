package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMegaportLAGPort_Basic(t *testing.T) {
	portName := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "bne_nxt1" {
					name = "NextDC B1"
				}
					resource "megaport_lag_port" "lag_port" {
			        product_name  = "%s"
			        port_speed  = 10000
			        location_id = data.megaport_location.bne_nxt1.id
			        contract_term_months        = 1
					market = "AU"
					marketplace_visibility = false
                    lag_count = 3
			      }`, portName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "product_name", portName),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "port_speed", "10000"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "market", "AU"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "lag_count", "3"),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "company_uid"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_lag_port.lag_port",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_lag_port.lag_port"
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
				ImportStateVerifyIgnore: []string{"last_updated", "lag_count", "lag_port_uids", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status"},
			},
		},
	})
}
