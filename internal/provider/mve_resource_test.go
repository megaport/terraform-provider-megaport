package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

const (
	MVETestLocation      = "Digital Realty Silicon Valley SJC34 (SCL2)"
	MVETestLocationIDNum = 65 // "Digital Realty Silicon Valley SJC34 (SCL2)"
)

type MVEArubaProviderTestSuite ProviderTestSuite
type MVEVersaProviderTestSuite ProviderTestSuite

func TestMVEArubaProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MVEArubaProviderTestSuite))
}

func TestMVEVersaProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MVEVersaProviderTestSuite))
}

func (suite *MVEArubaProviderTestSuite) TestAccMegaportMVEAruba_Basic() {
	mveName := RandomTestName()
	mveKey := RandomTestName()
	mveNameNew := RandomTestName()
	costCentre := RandomTestName()
	costCentreNew := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				
				data "megaport_mve_images" "aruba" {
  					vendor_filter = "Aruba"
  					id_filter = 23
				}

				resource "megaport_mve" "mve" {
                    product_name  = "%s"
                    location_id = data.megaport_location.test_location.id
                    contract_term_months        = 1
					cost_centre = "%s"
					diversity_zone = "red"

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "MEDIUM"
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
                  }`, MVETestLocationIDNum, mveName, costCentre, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveName),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentre),
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_type", "MVE"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor", "ARUBA"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "mve_size", "MEDIUM"),
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
  					id_filter = 23
				}
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
                        vendor = "ArUbA"
                        product_size = "mEdIuM"
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
                  }`, MVETestLocationIDNum, mveNameNew, costCentreNew, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveNameNew),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentreNew),
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_type", "MVE"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor", "ARUBA"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "mve_size", "MEDIUM"),
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

func (suite *MVEVersaProviderTestSuite) TestAccMegaportMVEVersa_Basic() {
	mveName := RandomTestName()
	mveNameNew := RandomTestName()
	costCentre := RandomTestName()
	costCentreNew := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
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
                        vendor             = "versa"
                        product_size       = "SMALL"
                        image_id           = 20
                        mve_label          = "MVE 2/8"
                        director_address   = "0.0.0.0"
                        controller_address = "0.0.0.0"
                        local_auth         = "test"
                        remote_auth        = "test2"
                        serial_number      = "test-serial-number"
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
                  }`, MVETestLocationIDNum, mveName, costCentre),
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
                        vendor             = "versa"
                        product_size       = "SMALL"
                        image_id           = 20
                        mve_label          = "MVE 2/8"
                        director_address   = "0.0.0.0"
                        controller_address = "0.0.0.0"
                        local_auth         = "test"
                        remote_auth        = "test2"
                        serial_number      = "test-serial-number"
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
                  }`, MVETestLocationIDNum, mveNameNew, costCentreNew),
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
