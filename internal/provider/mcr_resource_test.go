package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMegaportMCR_Basic(t *testing.T) {
	mcrName := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`resource "megaport_mcr" "mcr" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = 5
                    contract_term_months        = 1
					market = "AU"
					marketplace_visibility = false
                  }`, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "location_id", "5"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "market", "AU"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
				),
			},
		},
	})
}
