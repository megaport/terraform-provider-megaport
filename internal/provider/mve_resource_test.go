package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMegaportMVEAruba_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMVETestLocation(t, 4)
	mveName := RandomTestName()
	mveKey := RandomTestName()
	mveNameNew := RandomTestName()
	costCentre := RandomTestName()
	costCentreNew := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				data "megaport_mve_images" "aruba" {
  					vendor_filter = "Aruba"
  					id_filter = %d
				}

				resource "megaport_mve" "mve" {
                    product_name  = "%s"
                    location_id = data.megaport_location.test_location.id
                    contract_term_months        = 1
					cost_centre = "%s"
					diversity_zone = "red"

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "SMALL"
						mve_label = "MVE 2/8"
                        image_id = data.megaport_mve_images.aruba.mve_images.0.id
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
                    }

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}
					]
                  }`, locationID, MVEArubaImageID, mveName, costCentre, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveName),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentre),
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_type", "MVE"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor", "ARUBA"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "mve_size", "SMALL"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "market"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_name"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "diversity_zone", "red"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_mve.mve",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_mve.mve"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "vendor_config", "resources", "provisioning_status"},
			},
			// Update Testing
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				data "megaport_mve_images" "aruba" {
  					vendor_filter = "Aruba"
  					id_filter = %d
				}
				// Use mixed casing on vendor/size to verify the provider treats them
				// case-insensitively and does NOT force a destroy+recreate.
				resource "megaport_mve" "mve" {
                    product_name  = "%s"
					cost_centre = "%s"
                    location_id = data.megaport_location.test_location.id
                    contract_term_months        = 1
					diversity_zone = "red"

					resource_tags = {
						"key1updated" = "value1updated"
						"key2updated" = "value2updated"
					}

                    vendor_config = {
                        vendor = "aRuBa"
                        product_size = "SmAlL"
						mve_label = "MVE 2/8"
                        image_id = data.megaport_mve_images.aruba.mve_images.0.id
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
                    }

					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}
					]
                  }`, locationID, MVEArubaImageID, mveNameNew, costCentreNew, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveNameNew),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentreNew),
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_type", "MVE"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor", "ARUBA"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "mve_size", "SMALL"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "diversity_zone", "red"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "market"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_name"),
				),
			},
			// Make sure resource has not been destroyed by change of casing in Vendor Config
			{
				ResourceName:                         "megaport_mve.mve",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_mve.mve"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "vendor_config", "resources", "provisioning_status"},
			},
		},
	})
}

func TestAccMegaportMVEAruba_CostCentreRemoval(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMVETestLocation(t, 4)
	mveName := RandomTestName()
	mveKey := RandomTestName()
	costCentreName := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				data "megaport_mve_images" "aruba" {
					vendor_filter = "Aruba"
					id_filter = %d
				}
				resource "megaport_mve" "mve" {
					product_name = "%s"
					location_id = data.megaport_location.test_location.id
					contract_term_months = 1
					cost_centre = "%s"
					diversity_zone = "red"
					vendor_config = {
						vendor = "aruba"
						product_size = "SMALL"
						mve_label = "MVE 2/8"
						image_id = data.megaport_mve_images.aruba.mve_images.0.id
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
					}
					resource_tags = {
						"key1" = "value1"
					}
					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}]
				}`, locationID, MVEArubaImageID, mveName, costCentreName, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentreName),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				data "megaport_mve_images" "aruba" {
					vendor_filter = "Aruba"
					id_filter = %d
				}
				resource "megaport_mve" "mve" {
					product_name = "%s"
					location_id = data.megaport_location.test_location.id
					contract_term_months = 1
					cost_centre = ""
					diversity_zone = "red"
					vendor_config = {
						vendor = "aruba"
						product_size = "SMALL"
						mve_label = "MVE 2/8"
						image_id = data.megaport_mve_images.aruba.mve_images.0.id
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
					}
					resource_tags = {
						"key1" = "value1"
					}
					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}]
				}`, locationID, MVEArubaImageID, mveName, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", ""),
				),
			},
		},
	})
}

// TestAccMegaportMVEAruba_PromoCode exercises promo_code on megaport_mve
// against the v1.8.0 ordering endpoint. State tracks the config-supplied
// value.
func TestAccMegaportMVEAruba_PromoCode(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMVETestLocation(t, 4)
	mveName := RandomTestName()
	mveKey := RandomTestName()
	initialPromo := testPromoCode()
	const otherPromo = "tf-acc-test-promo-other"

	configFor := func(promoLine string) string {
		return providerConfig + fmt.Sprintf(`
		data "megaport_location" "test_location" {
			id = %d
		}
		data "megaport_mve_images" "aruba" {
			vendor_filter = "Aruba"
			id_filter = %d
		}
		resource "megaport_mve" "mve" {
			product_name         = "%s"
			location_id          = data.megaport_location.test_location.id
			contract_term_months = 1
			%s
			vendor_config = {
				vendor       = "aruba"
				product_size = "SMALL"
				mve_label    = "MVE 2/8"
				image_id     = data.megaport_mve_images.aruba.mve_images.0.id
				account_name = "%s"
				account_key  = "%s"
				system_tag   = "Preconfiguration-aruba-test-1"
			}
			vnics = [{
				description = "Data Plane"
			},
			{
				description = "Control Plane"
			},
			{
				description = "Management Plane"
			},
			{
				description = "Extra Plane"
			}]
		}`, locationID, MVEArubaImageID, mveName, promoLine, mveName, mveKey)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configFor(fmt.Sprintf(`promo_code = "%s"`, initialPromo)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "promo_code", initialPromo),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_uid"),
				),
			},
			{
				Config: configFor(fmt.Sprintf(`promo_code = "%s"`, otherPromo)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "promo_code", otherPromo),
				),
			},
			{
				Config: configFor(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("megaport_mve.mve", "promo_code"),
				),
			},
		},
	})
}

func TestAccMegaportMVEAruba_ContractTermUpdate(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMVETestLocation(t, 4)
	mveName := RandomTestName()
	mveKey := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				data "megaport_mve_images" "aruba" {
					vendor_filter = "Aruba"
					id_filter = %d
				}
				resource "megaport_mve" "mve" {
					product_name = "%s"
					location_id = data.megaport_location.test_location.id
					contract_term_months = 12
					diversity_zone = "red"
					vendor_config = {
						vendor = "aruba"
						product_size = "SMALL"
						mve_label = "MVE 2/8"
						image_id = data.megaport_mve_images.aruba.mve_images.0.id
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
					}
					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}]
				}`, locationID, MVEArubaImageID, mveName, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "12"),
					waitForProvisioningStatus("megaport_mve.mve"),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				data "megaport_mve_images" "aruba" {
					vendor_filter = "Aruba"
					id_filter = %d
				}
				resource "megaport_mve" "mve" {
					product_name = "%s"
					location_id = data.megaport_location.test_location.id
					contract_term_months = 24
					diversity_zone = "red"
					vendor_config = {
						vendor = "aruba"
						product_size = "SMALL"
						mve_label = "MVE 2/8"
						image_id = data.megaport_mve_images.aruba.mve_images.0.id
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
					}
					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}]
				}`, locationID, MVEArubaImageID, mveName, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "24"),
				),
			},
		},
	})
}

// TestAccMegaportMVEAruba_VnicDescriptionUpdate exercises ESD-1201:
// editing only vnics[*].description must update the MVE in place
// (product_uid preserved), while changing the vNIC count still forces
// replacement (product_uid changes).
func TestAccMegaportMVEAruba_VnicDescriptionUpdate(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	// Probe with 3 vNICs — the final step replaces the MVE with three vNICs,
	// so the location must validate for the maximum count the test exercises.
	locationID, _ := findMVETestLocation(t, 3)
	mveName := RandomTestName()
	mveKey := RandomTestName()

	configWithVnics := func(vnics string) string {
		return providerConfig + fmt.Sprintf(`
			data "megaport_location" "test_location" {
				id = %d
			}
			data "megaport_mve_images" "aruba" {
				vendor_filter = "Aruba"
				id_filter = %d
			}
			resource "megaport_mve" "mve" {
				product_name = "%s"
				location_id = data.megaport_location.test_location.id
				contract_term_months = 12
				diversity_zone = "red"
				vendor_config = {
					vendor = "aruba"
					product_size = "SMALL"
					mve_label = "MVE 2/8"
					image_id = data.megaport_mve_images.aruba.mve_images.0.id
					account_name = "%s"
					account_key = "%s"
					system_tag = "Preconfiguration-aruba-test-1"
				}
				vnics = %s
			}`, locationID, MVEArubaImageID, mveName, mveName, mveKey, vnics)
	}

	twoVnics := `[
		{ description = "Data Plane" },
		{ description = "Control Plane" }
	]`
	twoVnicsRenamed := `[
		{ description = "Data Plane Renamed" },
		{ description = "Control Plane Renamed" }
	]`
	threeVnics := `[
		{ description = "Data Plane Renamed" },
		{ description = "Control Plane Renamed" },
		{ description = "Management Plane" }
	]`

	var originalUID, postRenameUID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configWithVnics(twoVnics),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.0.description", "Data Plane"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.1.description", "Control Plane"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_uid"),
					waitForProvisioningStatus("megaport_mve.mve"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["megaport_mve.mve"]
						if !ok {
							return fmt.Errorf("megaport_mve.mve not found in state")
						}
						originalUID = rs.Primary.Attributes["product_uid"]
						return nil
					},
				),
			},
			// Rename both vNIC descriptions — must NOT replace.
			{
				Config: configWithVnics(twoVnicsRenamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.0.description", "Data Plane Renamed"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.1.description", "Control Plane Renamed"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["megaport_mve.mve"]
						if !ok {
							return fmt.Errorf("megaport_mve.mve not found in state")
						}
						if originalUID == "" {
							return fmt.Errorf("originalUID was not captured in step 1")
						}
						postRenameUID = rs.Primary.Attributes["product_uid"]
						if postRenameUID != originalUID {
							return fmt.Errorf("MVE was replaced (product_uid changed) when only vNIC descriptions changed; want in-place update. before=%s after=%s", originalUID, postRenameUID)
						}
						return nil
					},
				),
			},
			// Plan-only to confirm no drift after the in-place update.
			{
				Config:   configWithVnics(twoVnicsRenamed),
				PlanOnly: true,
			},
			// Add a vNIC — must force replacement (product_uid changes).
			{
				Config: configWithVnics(threeVnics),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.#", "3"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["megaport_mve.mve"]
						if !ok {
							return fmt.Errorf("megaport_mve.mve not found in state")
						}
						if postRenameUID == "" {
							return fmt.Errorf("postRenameUID was not captured in step 2")
						}
						if rs.Primary.Attributes["product_uid"] == postRenameUID {
							return fmt.Errorf("MVE was updated in place when vNIC count changed; want replacement. product_uid unchanged=%s", postRenameUID)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccMegaportMVEVersa_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMVEVersaTestLocation(t)
	mveName := RandomTestName()
	mveNameNew := RandomTestName()
	costCentre := RandomTestName()
	costCentreNew := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				data "megaport_mve_images" "versa" {
  					vendor_filter = "Versa"
  					id_filter = 20
				}

				resource "megaport_mve" "mve" {
                    product_name  = "%s"
                    location_id = data.megaport_location.test_location.id
                    contract_term_months        = 1
					cost_centre = "%s"
					diversity_zone = "red"

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

                    vendor_config = {
                        vendor = "versa"
                        product_size = "SMALL"
						mve_label = "MVE 2/8"
                        image_id = data.megaport_mve_images.versa.mve_images.0.id
						director_address = "director1.versa.com"
						controller_address = "controller1.versa.com"
						local_auth = "SDWAN-Branch@Versa.com"
						remote_auth = "Controller-1-staging@Versa.com"
						serial_number = "Megaport-Hub1"
                    }

					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}
					]
                  }`, locationID, mveName, costCentre),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveName),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentre),
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_type", "MVE"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor", "VERSA"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "mve_size", "SMALL"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "market"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_name"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "diversity_zone", "red"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_mve.mve",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_mve.mve"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "vendor_config", "provisioning_status"},
			},
			// Update Testing
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				data "megaport_mve_images" "versa" {
  					vendor_filter = "Versa"
  					id_filter = 20
				}

				resource "megaport_mve" "mve" {
                    product_name  = "%s"
                    location_id = data.megaport_location.test_location.id
                    contract_term_months        = 1
					cost_centre = "%s"
					diversity_zone = "red"

					resource_tags = {
						"key1updated" = "value1updated"
						"key2updated" = "value2updated"
					}

                    vendor_config = {
                        vendor = "versa"
                        product_size = "SMALL"
						mve_label = "MVE 2/8"
                        image_id = data.megaport_mve_images.versa.mve_images.0.id
						director_address = "director1.versa.com"
						controller_address = "controller1.versa.com"
						local_auth = "SDWAN-Branch@Versa.com"
						remote_auth = "Controller-1-staging@Versa.com"
						serial_number = "Megaport-Hub1"
                    }

					vnics = [{
						description = "Data Plane"
					},
					{
						description = "Control Plane"
					},
					{
						description = "Management Plane"
					},
					{
						description = "Extra Plane"
					}
					]
                  }`, locationID, mveNameNew, costCentreNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveNameNew),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentreNew),
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_type", "MVE"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor", "VERSA"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "mve_size", "SMALL"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "market"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_name"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "diversity_zone", "red"),
				),
			},
			// Make sure resource has not been destroyed by change of casing in Vendor Config
			{
				ResourceName:                         "megaport_mve.mve",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_mve.mve"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "vendor_config", "provisioning_status"},
			},
		},
	})
}

func TestAccMegaportMVEImport_WithLifecycleIgnoreChanges(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMVETestLocation(t, 2)
	mveName := RandomTestName()
	mveKey := RandomTestName()
	costCentre := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// First create a standard MVE
			{
				Config: providerConfig + fmt.Sprintf(`
                data "megaport_location" "test_location" {
                    id = %d
                }

                data "megaport_mve_images" "aruba" {
                      vendor_filter = "Aruba"
                      id_filter = %d
                }

                resource "megaport_mve" "import_test" {
                    product_name  = "%s"
                    location_id = data.megaport_location.test_location.id
                    contract_term_months = 1
                    cost_centre = "%s"
                    diversity_zone = "red"

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "SMALL"
                        mve_label = "MVE Import Test"
                        image_id = data.megaport_mve_images.aruba.mve_images.0.id
                        account_name = "%s"
                        account_key = "%s"
                        system_tag = "Preconfiguration-aruba-test-1"
                    }

                    vnics = [{
                        description = "Data Plane"
                    },
                    {
                        description = "Control Plane"
                    }]
                }`, locationID, MVEArubaImageID, mveName, costCentre, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.import_test", "product_name", mveName),
				),
			},
			// ImportState testing using the standard pattern
			{
				ResourceName:                         "megaport_mve.import_test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_mve.import_test"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "vendor_config", "resources", "provisioning_status"},
			},
			// Test a modification with lifecycle.ignore_changes
			{
				Config: providerConfig + fmt.Sprintf(`
                data "megaport_location" "test_location" {
                    id = %d
                }

                data "megaport_mve_images" "aruba" {
                      vendor_filter = "Aruba"
                      id_filter = %d
                }

                resource "megaport_mve" "import_test" {
                    product_name  = "%s-updated"
                    location_id = data.megaport_location.test_location.id
                    contract_term_months = 1
                    cost_centre = "%s-updated"
                    diversity_zone = "red"

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "SMALL"
                        mve_label = "MVE Import Test"
                        image_id = data.megaport_mve_images.aruba.mve_images.0.id
                        account_name = "%s"
                        account_key = "%s"
                        system_tag = "Preconfiguration-aruba-test-1"
                    }

                    vnics = [{
                        description = "Data Plane"
                    },
                    {
                        description = "Control Plane"
                    }]

                    lifecycle {
                        ignore_changes = [vendor_config]
                    }
                }`, locationID, MVEArubaImageID, mveName, costCentre, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.import_test", "product_name", mveName+"-updated"),
					resource.TestCheckResourceAttr("megaport_mve.import_test", "cost_centre", costCentre+"-updated"),
					resource.TestCheckResourceAttrSet("megaport_mve.import_test", "product_uid"),
				),
			},
		},
	})
}

// TestAccMegaportMVECisco_Basic exercises Cisco FTDv MVE provisioning end-to-end.
// The test uses manage_locally=true so no FMC details are required.
// Required fields per the Megaport API: adminPassword (≥9 chars) and manageLocally.
// Cisco Firewall does not support the "MVE 2/8" size; "MVE 4/16" (MEDIUM) is used.
// The Cisco Firewall requires at least 4 vNICs.
func TestAccMegaportMVECisco_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, imageID, _ := findMVECiscoTestLocation(t)
	mveName := RandomTestName()
	mveNameNew := RandomTestName()
	costCentre := RandomTestName()
	costCentreNew := RandomTestName()
	// Generate once and reuse across create + update steps so the second
	// apply doesn't show a vendor_config diff (the value is write-only and
	// not stored in state, but reusing it keeps the test config readable).
	adminPassword := mveTestCiscoAdminPassword(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				data "megaport_mve_images" "cisco" {
					vendor_filter = "Cisco"
					id_filter     = %d
				}

				resource "megaport_mve" "mve" {
					product_name         = "%s"
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
					cost_centre          = "%s"
					diversity_zone       = "red"

					vendor_config = {
						vendor         = "cisco"
						product_size   = "MEDIUM"
						mve_label      = "MVE 4/16"
						image_id       = data.megaport_mve_images.cisco.mve_images.0.id
						manage_locally = true
						admin_password = "%s"
					}

					vnics = [
						{ description = "Data Plane" },
						{ description = "Control Plane" },
						{ description = "Management Plane" },
						{ description = "HA Plane" },
					]
				}`, locationID, imageID, mveName, costCentre, adminPassword),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveName),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentre),
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_type", "MVE"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor", "CISCO"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "diversity_zone", "red"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "mve_size"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "market"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_name"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_mve.mve",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_mve.mve"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "vendor_config", "resources", "provisioning_status"},
			},
			// Update: rename and change cost centre.
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				data "megaport_mve_images" "cisco" {
					vendor_filter = "Cisco"
					id_filter     = %d
				}

				resource "megaport_mve" "mve" {
					product_name         = "%s"
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
					cost_centre          = "%s"
					diversity_zone       = "red"

					vendor_config = {
						vendor         = "cisco"
						product_size   = "MEDIUM"
						mve_label      = "MVE 4/16"
						image_id       = data.megaport_mve_images.cisco.mve_images.0.id
						manage_locally = true
						admin_password = "%s"
					}

					vnics = [
						{ description = "Data Plane" },
						{ description = "Control Plane" },
						{ description = "Management Plane" },
						{ description = "HA Plane" },
					]
				}`, locationID, imageID, mveNameNew, costCentreNew, adminPassword),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveNameNew),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentreNew),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_uid"),
				),
			},
		},
	})
}

// TestAccMegaportMVEPaloAlto_Basic exercises Palo Alto VM-Series MVE provisioning
// end-to-end. Required fields per the Megaport API: adminPasswordHash (sha256crypt
// format) and sshPublicKey (RSA 2048 bit). The SSH key is generated freshly per
// test run via mveTestSSHPublicKey — no external setup needed.
func TestAccMegaportMVEPaloAlto_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	sshPublicKey := mveTestSSHPublicKey(t)
	locationID, imageID, _ := findMVEPaloAltoTestLocation(t)
	mveName := RandomTestName()
	mveNameNew := RandomTestName()
	costCentre := RandomTestName()
	costCentreNew := RandomTestName()
	// Generate once and reuse across create + update steps. admin_password_hash
	// is stored in state (not write-only), so using different values would
	// trigger an unintended vendor_config replace on the update step.
	adminPasswordHash := mveTestPaloAltoAdminPasswordHash(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				data "megaport_mve_images" "palo_alto" {
					vendor_filter = "Palo Alto"
					id_filter     = %d
				}

				resource "megaport_mve" "mve" {
					product_name         = "%s"
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
					cost_centre          = "%s"
					diversity_zone       = "red"

					vendor_config = {
						vendor              = "palo_alto"
						product_size        = "MEDIUM"
						mve_label           = "MVE 4/16"
						image_id            = data.megaport_mve_images.palo_alto.mve_images.0.id
						admin_password_hash = "%s"
						ssh_public_key      = "%s"
					}

					vnics = [
						{ description = "Management" },
						{ description = "Data" },
					]
				}`, locationID, imageID, mveName, costCentre, adminPasswordHash, sshPublicKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveName),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentre),
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_type", "MVE"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor", "PALO_ALTO"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "diversity_zone", "red"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "mve_size"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "market"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_name"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_mve.mve",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_mve.mve"
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "vendor_config", "resources", "provisioning_status"},
			},
			// Update: rename and change cost centre.
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				data "megaport_mve_images" "palo_alto" {
					vendor_filter = "Palo Alto"
					id_filter     = %d
				}

				resource "megaport_mve" "mve" {
					product_name         = "%s"
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
					cost_centre          = "%s"
					diversity_zone       = "red"

					vendor_config = {
						vendor              = "palo_alto"
						product_size        = "MEDIUM"
						mve_label           = "MVE 4/16"
						image_id            = data.megaport_mve_images.palo_alto.mve_images.0.id
						admin_password_hash = "%s"
						ssh_public_key      = "%s"
					}

					vnics = [
						{ description = "Management" },
						{ description = "Data" },
					]
				}`, locationID, imageID, mveNameNew, costCentreNew, adminPasswordHash, sshPublicKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveNameNew),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentreNew),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_uid"),
				),
			},
		},
	})
}
