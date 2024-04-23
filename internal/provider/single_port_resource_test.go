package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
		},
	})
}
