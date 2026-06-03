package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	MVEArubaImageIDMVE = 152 // Aruba MVE image ID
)

func TestAccMegaportMVEAruba_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMVETestLocation(t, 2)
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
                  }`, locationID, MVEArubaImageIDMVE, mveName, costCentre, mveName, mveKey),
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
                  }`, locationID, MVEArubaImageIDMVE, mveNameNew, costCentreNew, mveName, mveKey),
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
	locationID, _ := findMVETestLocation(t, 2)
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
				}`, locationID, MVEArubaImageIDMVE, mveName, costCentreName, mveName, mveKey),
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
				}`, locationID, MVEArubaImageIDMVE, mveName, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", ""),
				),
			},
		},
	})
}

func TestAccMegaportMVEAruba_ContractTermUpdate(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMVETestLocation(t, 2)
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
				}`, locationID, MVEArubaImageIDMVE, mveName, mveName, mveKey),
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
				}`, locationID, MVEArubaImageIDMVE, mveName, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "24"),
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
                }`, locationID, MVEArubaImageIDMVE, mveName, costCentre, mveName, mveKey),
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
                }`, locationID, MVEArubaImageIDMVE, mveName, costCentre, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.import_test", "product_name", mveName+"-updated"),
					resource.TestCheckResourceAttr("megaport_mve.import_test", "cost_centre", costCentre+"-updated"),
					resource.TestCheckResourceAttrSet("megaport_mve.import_test", "product_uid"),
				),
			},
		},
	})
}
