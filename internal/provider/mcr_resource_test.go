package provider

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccMegaportMCR_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	prefixFilterName := RandomTestName()
	prefixFilterName2 := RandomTestName()
	prefixFilterNameNew := RandomTestName()
	prefixFilterNameNew2 := RandomTestName()
	prefixFilterNameNew3 := RandomTestName()
	prefixFilterNameNew4 := RandomTestName()
	costCentreName := RandomTestName()
	mcrNameNew := RandomTestName()
	mcrNameNew2 := RandomTestName()
	costCentreNameNew := RandomTestName()
	costCentreNameNew2 := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr" "mcr" {
					product_name             = "%s"
					port_speed               = 1000
					location_id              = data.megaport_location.test_location.id
					contract_term_months     = 12
					cost_centre              = "%s"

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

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
				  `, locationID, mcrName, costCentreName, prefixFilterName, prefixFilterName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2", "value2"),
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
			// Update Test 1
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr" "mcr" {
					product_name             = "%s"
					port_speed               = 1000
					location_id              = data.megaport_location.test_location.id
					contract_term_months     = 12
					cost_centre              = "%s"
					resource_tags = {
						"key1updated" = "value1updated"
						"key2updated" = "value2updated"
					}

					prefix_filter_lists = [
					{
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 24
							le      = 32
						  },
						  {
							action  = "deny"
							prefix  = "10.0.2.0/24"
							ge      = 25
							le      = 29
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
							ge      = 25
							le      = 32
						  },
						  {
							action  = "deny"
							prefix  = "10.0.2.0/24"
							ge      = 24
							le      = 26
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
							ge      = 24
							le      = 24
						  },
						  {
							action  = "deny"
							prefix  = "10.0.2.0/24"
							ge      = 27
							le      = 32
						  }
						]
					  }]
				  }
				  `, locationID, mcrName, costCentreName, prefixFilterNameNew, prefixFilterNameNew2, prefixFilterNameNew3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.#", "3"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.description", prefixFilterNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.description", prefixFilterNameNew2),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.description", prefixFilterNameNew3),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.le", "29"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.le", "26"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.0.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.0.le", "24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.1.ge", "27"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.1.le", "32"),
				),
			},
			// Update Test 2
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr" "mcr" {
					product_name             = "%s"
					port_speed               = 1000
					location_id              = data.megaport_location.test_location.id
					contract_term_months     = 12
					cost_centre              = "%s"

					resource_tags = {
						"key1updated" = "value1updated"
						"key2updated" = "value2updated"
					}

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
				  `, locationID, mcrNameNew, costCentreNameNew, prefixFilterNameNew4),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.description", prefixFilterNameNew4),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.#", "1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.#", "1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.ge", "28"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.le", "32"),
				),
			},
			// Update Test 3
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr" "mcr" {
					product_name             = "%s"
					port_speed               = 1000
					location_id              = data.megaport_location.test_location.id
					contract_term_months     = 12
					cost_centre              = "%s"

					prefix_filter_lists = []
				  }
				  `, locationID, mcrNameNew2, costCentreNameNew2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrNameNew2),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreNameNew2),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.#", "0"),
				),
			},
		},
	})
}

func TestAccMegaportMCR_CostCentreRemoval(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	costCentreName := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_mcr" "mcr" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 1
					cost_centre = "%s"
				}`, locationID, mcrName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_mcr" "mcr" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 1
					cost_centre = ""
				}`, locationID, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", ""),
				),
			},
		},
	})
}

// TestAccMegaportMCR_PromoCode exercises promo_code against the v1.8.0
// ordering endpoint. State tracks the config-supplied value.
func TestAccMegaportMCR_PromoCode(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	initialPromo := testPromoCode()
	const otherPromo = "tf-acc-test-promo-other"

	configFor := func(promoLine string) string {
		return providerConfig + fmt.Sprintf(`
		data "megaport_location" "test_location" {
			id = %d
		}
		resource "megaport_mcr" "mcr" {
			product_name         = "%s"
			port_speed            = 1000
			location_id          = data.megaport_location.test_location.id
			contract_term_months = 1
			%s
		}`, locationID, mcrName, promoLine)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configFor(fmt.Sprintf(`promo_code = "%s"`, initialPromo)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "promo_code", initialPromo),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
				),
			},
			{
				Config: configFor(fmt.Sprintf(`promo_code = "%s"`, otherPromo)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "promo_code", otherPromo),
				),
			},
			{
				Config: configFor(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("megaport_mcr.mcr", "promo_code"),
				),
			},
		},
	})
}

func TestAccMegaportMCR_ContractTermUpdate(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_mcr" "mcr" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 12
				}`, locationID, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					waitForProvisioningStatus("megaport_mcr.mcr"),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_mcr" "mcr" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 24
				}`, locationID, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "24"),
				),
			},
		},
	})
}

func TestAccMegaportMCRCustomASN_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	mcrNameNew := RandomTestName()
	costCentreName := RandomTestName()
	costCentreNameNew := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr" "mcr" {
					product_name             = "%s"
					port_speed               = 1000
					location_id              = data.megaport_location.test_location.id
					contract_term_months     = 12
					cost_centre              = "%s"
					asn = 65000

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}
				  }
				  `, locationID, mcrName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "asn", "65000"),
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
			// Update Test 1
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				  resource "megaport_mcr" "mcr" {
					product_name             = "%s"
					port_speed               = 1000
					location_id              = data.megaport_location.test_location.id
					contract_term_months     = 12
					cost_centre              = "%s"
					asn = 65000

					resource_tags = {"key1updated" = "value1updated", "key2updated" = "value2updated"}
				  }
				  `, locationID, mcrNameNew, costCentreNameNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "asn", "65000"),
				),
			},
		},
	})
}

func TestRateLimiter_SingleToken(t *testing.T) {
	rl := NewRateLimiter(5, 100*time.Millisecond)

	select {
	case <-rl.rateLimitCh:
		// Successfully got token
	default:
		t.Error("Failed to get first token")
	}
}

func TestRateLimiter_BurstLimit(t *testing.T) {
	rl := NewRateLimiter(5, 100*time.Millisecond)

	// Should get 5 tokens
	for i := 0; i < 5; i++ {
		select {
		case <-rl.rateLimitCh:
			// Successfully got token
		default:
			t.Errorf("Failed to get token %d within burst limit", i+1)
		}
	}

	// Should fail to get 6th token
	select {
	case <-rl.rateLimitCh:
		t.Error("Got token beyond burst limit")
	default:
		// Expected failure to get token
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	rl := NewRateLimiter(5, 100*time.Millisecond)

	// Use all tokens
	for i := 0; i < 5; i++ {
		select {
		case <-rl.rateLimitCh:
			// Token consumed
		default:
			t.Errorf("Failed to get token %d from initial burst", i)
		}
	}

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should get a token after refill
	select {
	case <-rl.rateLimitCh:
		// Successfully got token after refill
	default:
		t.Error("Failed to get token after refill")
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	rl := NewRateLimiter(10, 100*time.Millisecond)
	var wg sync.WaitGroup
	successCount := int32(0)

	// Launch 20 goroutines
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-rl.rateLimitCh:
				atomic.AddInt32(&successCount, 1)
			default:
				// Failed to get token
			}
		}()
	}

	wg.Wait()

	// Should have exactly 10 successes (burst limit)
	if atomic.LoadInt32(&successCount) != 10 {
		t.Errorf("Expected 10 successful token acquisitions, got %d", successCount)
	}
}

// TestAccMegaportMCR_UpdateASN exercises ESD-1094: changing the BGP ASN on
// an existing MCR must update the resource in place rather than forcing a
// destroy-and-recreate. The test asserts the product_uid is preserved
// across the ASN change and that the new ASN reads back from the API.
func TestAccMegaportMCR_UpdateASN(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()

	configWithASN := func(asn int) string {
		return providerConfig + fmt.Sprintf(`
			data "megaport_location" "test_location" {
				id = %d
			}
			resource "megaport_mcr" "mcr" {
				product_name         = "%s"
				port_speed           = 1000
				location_id          = data.megaport_location.test_location.id
				contract_term_months = 12
				asn                  = %d
			}
		`, locationID, mcrName, asn)
	}

	configNoASN := providerConfig + fmt.Sprintf(`
		data "megaport_location" "test_location" {
			id = %d
		}
		resource "megaport_mcr" "mcr" {
			product_name         = "%s"
			port_speed           = 1000
			location_id          = data.megaport_location.test_location.id
			contract_term_months = 12
		}
	`, locationID, mcrName)

	var originalUID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configWithASN(64512),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "asn", "64512"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["megaport_mcr.mcr"]
						if !ok {
							return fmt.Errorf("megaport_mcr.mcr not found in state")
						}
						originalUID = rs.Primary.Attributes["product_uid"]
						return nil
					},
				),
			},
			// Change ASN — must NOT replace.
			{
				Config: configWithASN(64513),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "asn", "64513"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["megaport_mcr.mcr"]
						if !ok {
							return fmt.Errorf("megaport_mcr.mcr not found in state")
						}
						if rs.Primary.Attributes["product_uid"] != originalUID {
							return fmt.Errorf("MCR was replaced (product_uid changed) when ASN was updated; want in-place update. before=%s after=%s", originalUID, rs.Primary.Attributes["product_uid"])
						}
						return nil
					},
				),
			},
			// Plan-only to confirm no drift after the in-place update.
			{
				Config:   configWithASN(64513),
				PlanOnly: true,
			},
			// Omit asn from config — must NOT plan a reset to the default 133937.
			{
				Config:             configNoASN,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// mcrComputedFromAPI lists the attributes fromAPIMCR rewrites from the API
// response on every read. The migration plan marks all of them unknown, so a
// converged plan has to leave each one populated. attribute_tags is left out:
// fromAPIMCR leaves that map alone when the API sends no tags, so state can
// hold a null there.
var mcrComputedFromAPI = []string{
	"aggregation_id",
	"cancelable",
	"company_name",
	"contract_end_date",
	"contract_start_date",
	"lag_id",
	"lag_primary",
	"last_updated",
	"live_date",
	"provisioning_status",
	"terminate_date",
}

// checkMCRComputedPresent asserts every attribute in mcrComputedFromAPI is in
// state. An MCR with no live date reads back an empty one, so presence is the
// check here, not a non-empty value. TestCheckResourceAttrSet rejects both.
func checkMCRComputedPresent(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("%s not found in state", name)
		}
		for _, attribute := range mcrComputedFromAPI {
			if _, ok := rs.Primary.Attributes[attribute]; !ok {
				return fmt.Errorf("%s: %s is missing from state", name, attribute)
			}
		}
		return nil
	}
}

// TestAccMegaportMCR_InlinePrefixListMigration covers GitHub issue #450. An MCR
// that drops the deprecated prefix_filter_lists attribute has to plan clean
// afterwards, with no lifecycle block in the configuration.
func TestAccMegaportMCR_InlinePrefixListMigration(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	mcrNameRenamed := RandomTestName()
	prefixFilterName := RandomTestName()
	costCentreName := RandomTestName()

	withInlineList := providerConfig + fmt.Sprintf(`
	data "megaport_location" "test_location" {
		id = %d
	}

	resource "megaport_mcr" "mcr" {
		product_name         = "%s"
		port_speed           = 1000
		location_id          = data.megaport_location.test_location.id
		contract_term_months = 12
		cost_centre          = "%s"

		prefix_filter_lists = [{
			description    = "%s"
			address_family = "IPv4"
			entries = [{
				action = "permit"
				prefix = "10.0.1.0/24"
				ge     = 25
				le     = 32
			}]
		}]
	}
	`, locationID, mcrName, costCentreName, prefixFilterName)

	// The migrated configuration. No prefix_filter_lists, no lifecycle block,
	// and no empty list either.
	migrated := func(name string) string {
		return providerConfig + fmt.Sprintf(`
	data "megaport_location" "test_location" {
		id = %d
	}

	resource "megaport_mcr" "mcr" {
		product_name         = "%s"
		port_speed           = 1000
		location_id          = data.megaport_location.test_location.id
		contract_term_months = 12
		cost_centre          = "%s"
	}
	`, locationID, name, costCentreName)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// The inline attribute still works, and a create populates every
			// computed attribute the migration plan later marks unknown.
			{
				Config: withInlineList,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.#", "1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.description", prefixFilterName),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_name"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "last_updated"),
					checkMCRComputedPresent("megaport_mcr.mcr"),
				),
			},
			// Dropping the attribute converges. The test framework plans after
			// every apply and fails the step on a non-empty plan, so this step
			// is the regression guard for the reported bug.
			{
				Config: migrated(mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.#", "1"),
					checkMCRComputedPresent("megaport_mcr.mcr"),
				),
			},
			// The next two plans stay empty as well.
			{
				Config:   migrated(mcrName),
				PlanOnly: true,
			},
			{
				Config:   migrated(mcrName),
				PlanOnly: true,
			},
			// A real change still refreshes the computed attributes. Pinning
			// them here would fail the apply with an inconsistent result.
			{
				Config: migrated(mcrNameRenamed),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectUnknownValue("megaport_mcr.mcr", tfjsonpath.New("last_updated")),
						plancheck.ExpectUnknownValue("megaport_mcr.mcr", tfjsonpath.New("provisioning_status")),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrNameRenamed),
					checkMCRComputedPresent("megaport_mcr.mcr"),
				),
			},
		},
	})
}

func TestRateLimiter_RateOverTime(t *testing.T) {
	rl := NewRateLimiter(5, 100*time.Millisecond)
	start := time.Now()
	count := 0

	for time.Since(start) < 450*time.Millisecond {
		select {
		case <-rl.rateLimitCh:
			count++
		default:
			// No token available
		}
		time.Sleep(10 * time.Millisecond)
	}

	expected := 9
	if count != expected {
		t.Errorf("Expected %d tokens over time period, got %d", expected, count)
	}
}
