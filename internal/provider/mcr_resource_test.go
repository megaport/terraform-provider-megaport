package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMegaportMCR_Basic(t *testing.T) {
	mcrName := RandomTestName()
	prefixFilterName := RandomTestName()
	prefixFilterName2 := RandomTestName()
	prefixFilterNameNew := RandomTestName()
	costCentreName := RandomTestName()
	mcrNameNew := RandomTestName()
	costCentreNameNew := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "bne_nxt1" {
					name = "NextDC B1"
				}
				  resource "megaport_mcr" "mcr" {
					product_name             = "%s"
					port_speed               = 1000
					location_id              = data.megaport_location.bne_nxt1.id
					contract_term_months     = 12
					cost_centre              = "%s"

					prefix_filter_lists = [
					{
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 25
							le      = 32
						  },
						  {
							action  = "deny"
							prefix  = "10.0.2.0/24"
							ge      = 25
							le      = 27
						  }
						]
					  }, 
					  {
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 26
							le      = 32
						  },
						  {
							action  = "deny"
							prefix  = "10.0.2.0/24"
							ge      = 24
							le      = 25
						  }
						]
					  }]
				  }
				  `, mcrName, costCentreName, prefixFilterName, prefixFilterName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.description", prefixFilterName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.description", prefixFilterName2),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.le", "27"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.ge", "26"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.le", "25"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_mcr.mcr",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_mcr.mcr"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "provisioning_status"},
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "bne_nxt1" {
					name = "NextDC B1"
				}
				  resource "megaport_mcr" "mcr" {
					product_name             = "%s"
					port_speed               = 1000
					location_id              = data.megaport_location.bne_nxt1.id
					contract_term_months     = 12
					cost_centre              = "%s"

					prefix_filter_lists = [{
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 28
							le      = 32
						  }
						]
					  }]
				  }
				  `, mcrNameNew, costCentreNameNew, prefixFilterNameNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.description", prefixFilterNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.#", "1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.#", "1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.ge", "28"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.le", "32"),
				),
			},
		},
	})
}
