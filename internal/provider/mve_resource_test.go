package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

type MVEArubaProviderTestSuite ProviderTestSuite
type MVEVersaProviderTestSuite ProviderTestSuite

func TestMVEArubaProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MVEArubaProviderTestSuite))
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
				data "megaport_location" "bne_nxt1" {
					name = "NextDC B1"
				}
				resource "megaport_mve" "mve" {
                    product_name  = "%s"
                    location_id = data.megaport_location.bne_nxt1.id
                    contract_term_months        = 1
					cost_centre = "%s"

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "MEDIUM"
                        image_id = 23
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
                  }`, mveName, costCentre, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveName),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentre),
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_type", "MVE"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor", "ARUBA"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "mve_size", "MEDIUM"),
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
			// Update Testing
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "bne_nxt1" {
					name = "NextDC B1"
				}
				resource "megaport_mve" "mve" {
                    product_name  = "%s"
					cost_centre = "%s"
                    location_id = data.megaport_location.bne_nxt1.id
                    contract_term_months        = 1

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "MEDIUM"
                        image_id = 23
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
                  }`, mveNameNew, costCentreNew, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveNameNew),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentreNew),
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_type", "MVE"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor", "ARUBA"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "mve_size", "MEDIUM"),
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
		},
	})
}

func TestMVEVersaProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MVEVersaProviderTestSuite))
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
				data "megaport_location" "bne_nxt1" {
					name = "NextDC B1"
				}
				resource "megaport_mve" "mve" {
                    product_name  = "%s"
                    location_id = data.megaport_location.bne_nxt1.id
                    contract_term_months        = 1
					cost_centre = "%s"

                    vendor_config = {
                        vendor = "versa"
                        product_size = "LARGE"
                        image_id = 20
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
                  }`, mveName, costCentre),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveName),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentre),
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_type", "MVE"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor", "VERSA"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "mve_size", "LARGE"),
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "resources", "vendor_config", "provisioning_status"},
			},
			// Update Testing
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "bne_nxt1" {
					name = "NextDC B1"
				}
				resource "megaport_mve" "mve" {
                    product_name  = "%s"
                    location_id = data.megaport_location.bne_nxt1.id
                    contract_term_months        = 1
					cost_centre = "%s"

                    vendor_config = {
                        vendor = "versa"
                        product_size = "LARGE"
                        image_id = 20
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
                  }`, mveNameNew, costCentreNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveNameNew),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentreNew),
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_type", "MVE"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor", "VERSA"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "mve_size", "LARGE"),
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
		},
	})
}
