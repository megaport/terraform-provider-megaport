package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccMegaportLAGPort_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findPortTestLocation(t, 10000)
	portName := RandomTestName()
	costCentreName := RandomTestName()
	portNameNew := RandomTestName()
	costCentreNameNew := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
					resource "megaport_lag_port" "lag_port" {
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
			      }`, locationID, portName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "product_name", portName),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "port_speed", "10000"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "marketplace_visibility", "true"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "lag_count", "1"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "resource_tags.key2", "value2"),
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
			// Update Testing
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
					resource "megaport_lag_port" "lag_port" {
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
			      }`, locationID, portNameNew, costCentreNameNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "product_name", portNameNew),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "port_speed", "10000"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "lag_count", "1"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "company_uid"),
				),
			},
		},
	})
}

func TestAccMegaportLAGPort_CostCentreRemoval(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findPortTestLocation(t, 10000)
	portName := RandomTestName()
	costCentreName := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_lag_port" "lag_port" {
					product_name  = "%s"
					cost_centre = "%s"
					port_speed  = 10000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 1
					marketplace_visibility = false
					lag_count = 1
				}`, locationID, portName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "cost_centre", costCentreName),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_lag_port" "lag_port" {
					product_name  = "%s"
					cost_centre = ""
					port_speed  = 10000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 1
					marketplace_visibility = false
					lag_count = 1
				}`, locationID, portName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "cost_centre", ""),
				),
			},
		},
	})
}

// TestAccMegaportLAGPort_PromoCode exercises promo_code against the v1.8.0
// ordering endpoint. State tracks the config-supplied value.
func TestAccMegaportLAGPort_PromoCode(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findPortTestLocation(t, 10000)
	portName := RandomTestName()
	initialPromo := testPromoCode()
	const otherPromo = "tf-acc-test-promo-other"

	configFor := func(promoLine string) string {
		return providerConfig + fmt.Sprintf(`
		data "megaport_location" "test_location" {
			id = %d
		}
		resource "megaport_lag_port" "lag_port" {
			product_name           = "%s"
			port_speed             = 10000
			location_id            = data.megaport_location.test_location.id
			contract_term_months   = 1
			marketplace_visibility = false
			lag_count              = 1
			%s
		}`, locationID, portName, promoLine)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configFor(fmt.Sprintf(`promo_code = "%s"`, initialPromo)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "promo_code", initialPromo),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "product_uid"),
				),
			},
			{
				Config: configFor(fmt.Sprintf(`promo_code = "%s"`, otherPromo)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "promo_code", otherPromo),
				),
			},
			{
				Config: configFor(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("megaport_lag_port.lag_port", "promo_code"),
				),
			},
		},
	})
}

func TestAccMegaportLAGPort_ContractTermUpdate(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findPortTestLocation(t, 10000)
	portName := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_lag_port" "lag_port" {
					product_name  = "%s"
					port_speed  = 10000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 12
					marketplace_visibility = false
					lag_count = 1
				}`, locationID, portName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "contract_term_months", "12"),
					waitForProvisioningStatus("megaport_lag_port.lag_port"),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_lag_port" "lag_port" {
					product_name  = "%s"
					port_speed  = 10000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 24
					marketplace_visibility = false
					lag_count = 1
				}`, locationID, portName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "contract_term_months", "24"),
				),
			},
		},
	})
}

// TestAccMegaportLAGPort_GrowLagCount covers both directions of a lag_count
// change. Raising it adds ports to the LAG the customer already has; lowering
// it still replaces the resource, because the API has no call to remove a
// member.
func TestAccMegaportLAGPort_GrowLagCount(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findPortTestLocation(t, 10000)
	portName := RandomTestName()
	var initialUIDs []string

	configFor := func(lagCount int) string {
		return providerConfig + fmt.Sprintf(`
		data "megaport_location" "test_location" {
			id = %d
		}
		resource "megaport_lag_port" "lag_port" {
			product_name           = "%s"
			port_speed             = 10000
			location_id            = data.megaport_location.test_location.id
			contract_term_months   = 1
			marketplace_visibility = false
			lag_count              = %d
		}`, locationID, portName, lagCount)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configFor(2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "lag_count", "2"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "lag_port_uids.#", "2"),
					captureLagPortUIDs("megaport_lag_port.lag_port", &initialUIDs),
					waitForProvisioningStatus("megaport_lag_port.lag_port"),
				),
			},
			{
				Config: configFor(3),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("megaport_lag_port.lag_port", plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue("megaport_lag_port.lag_port", tfjsonpath.New("lag_port_uids")),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "lag_count", "3"),
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "lag_port_uids.#", "3"),
					checkLagPortUIDsKept("megaport_lag_port.lag_port", &initialUIDs),
				),
			},
			// The configured count now matches the API, so a plan is empty.
			{
				Config:   configFor(3),
				PlanOnly: true,
			},
			// A lower count still plans a replacement. Planned only, so the run
			// does not pay for a destroy and a rebuild.
			{
				Config:             configFor(2),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("megaport_lag_port.lag_port", plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})
}

// lagPortUIDsFromState reads the LAG member UIDs a resource holds in state.
func lagPortUIDsFromState(s *terraform.State, resourceName string) ([]string, error) {
	rs, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return nil, fmt.Errorf("resource %s not found in state", resourceName)
	}

	count, err := strconv.Atoi(rs.Primary.Attributes["lag_port_uids.#"])
	if err != nil {
		return nil, fmt.Errorf("could not read lag_port_uids.# for %s: %w", resourceName, err)
	}

	uids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		uids = append(uids, rs.Primary.Attributes[fmt.Sprintf("lag_port_uids.%d", i)])
	}
	return uids, nil
}

// captureLagPortUIDs stores the member UIDs for a later step to compare against.
func captureLagPortUIDs(resourceName string, into *[]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		uids, err := lagPortUIDsFromState(s, resourceName)
		if err != nil {
			return err
		}
		*into = uids
		return nil
	}
}

// checkLagPortUIDsKept fails when a port the LAG already had has left it.
func checkLagPortUIDsKept(resourceName string, prior *[]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		uids, err := lagPortUIDsFromState(s, resourceName)
		if err != nil {
			return err
		}

		current := make(map[string]bool, len(uids))
		for _, uid := range uids {
			current[uid] = true
		}
		for _, uid := range *prior {
			if !current[uid] {
				return fmt.Errorf("port %s is no longer in LAG %s, so the grow replaced it", uid, resourceName)
			}
		}
		return nil
	}
}
