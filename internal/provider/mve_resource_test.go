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
                    product_name         = "%s"
                    location_id          = data.megaport_location.test_location.id
                    contract_term_months = 1
                    cost_centre          = "%s"

                    resource_tags = {
                      "key1" = "value1"
                      "key2" = "value2"
                    }

                    vnics = [
                      {
                        description = "Data Plane"
                      },
                      {
                        description = "Management Plane"
                      },
                      {
                        description = "Control Plane"
                      }
                    ]

                    vendor_config = {
                      vendor       = "aruba"
                      product_size = "MEDIUM"
                      image_id     = 23
                      account_name = "%s-account"
                      account_key  = "%s-key"
                      system_tag   = "Preconfiguration-test-1"
                    }
                  }
                  
                  # Test MVE data source with name filter
                  data "megaport_mves" "test_name_filter" {
                    filter {
                      name = "name"
                      values = ["%s"]
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  
                  # Test MVE data source with vendor filter
                  data "megaport_mves" "test_vendor_filter" {
                    filter {
                      name = "vendor"
                      values = ["aruba"]
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  
                  # Test MVE data source with size filter
                  data "megaport_mves" "test_size_filter" {
                    filter {
                      name = "size"
                      values = ["MEDIUM"]
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  
                  # Test MVE data source with multiple filters
                  data "megaport_mves" "test_multi_filter" {
                    filter {
                      name = "vendor"
                      values = ["aruba"]
                    }
                    filter {
                      name = "location-id"
                      values = ["%d"]
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  
                  # Test MVE data source with tags
                  data "megaport_mves" "test_tag_filter" {
                    tags = {
                      "key1" = "value1"
                      "key2" = "value2"
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  `, MVETestLocationIDNum, mveName, costCentre, mveName, mveName, mveName, MVETestLocationIDNum),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveName),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentre),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "created_by"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "location_id", fmt.Sprintf("%d", MVETestLocationIDNum)),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.vendor", "aruba"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.product_size", "MEDIUM"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.account_name", fmt.Sprintf("%s-account", mveName)),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.account_key", fmt.Sprintf("%s-key", mveName)),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.system_tag", "Preconfiguration-test-1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.#", "3"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.0.description", "Data Plane"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.1.description", "Management Plane"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.2.description", "Control Plane"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key2", "value2"),

					// Check data source results
					resource.TestCheckResourceAttr("data.megaport_mves.test_name_filter", "uids.#", "1"),
					resource.TestCheckResourceAttrSet("data.megaport_mves.test_vendor_filter", "uids.#"),
					resource.TestCheckResourceAttrSet("data.megaport_mves.test_size_filter", "uids.#"),
					resource.TestCheckResourceAttrSet("data.megaport_mves.test_multi_filter", "uids.#"),
					resource.TestCheckResourceAttr("data.megaport_mves.test_tag_filter", "uids.#", "1"),
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "provisioning_status"},
			},
			// Update Test
			{
				Config: providerConfig + fmt.Sprintf(`
                data "megaport_location" "test_location" {
                    id = %d
                }
                  resource "megaport_mve" "mve" {
                    product_name         = "%s"
                    location_id          = data.megaport_location.test_location.id
                    contract_term_months = 1
                    cost_centre          = "%s"

                    resource_tags = {
                      "key1updated" = "value1updated"
                      "key2updated" = "value2updated"
                    }

                    vnics = [
                      {
                        description = "Data Plane"
                      },
                      {
                        description = "Management Plane"
                      },
                      {
                        description = "Control Plane"
                      }
                    ]

                    vendor_config = {
                      vendor       = "aruba"
                      product_size = "MEDIUM"
                      image_id     = 23
                      account_name = "%s-account"
                      account_key  = "%s-key"
                      system_tag   = "Preconfiguration-test-1"
                    }
                  }
                  
                  # Test MVE data source with updated name filter
                  data "megaport_mves" "test_name_filter" {
                    filter {
                      name = "name"
                      values = ["%s"]
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  
                  # Test MVE data source with updated tags
                  data "megaport_mves" "test_tag_filter" {
                    tags = {
                      "key1updated" = "value1updated"
                      "key2updated" = "value2updated"
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  
                  # Test MVE data source with cost-centre filter
                  data "megaport_mves" "test_cost_centre_filter" {
                    filter {
                      name = "cost-centre"
                      values = ["%s"]
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  `, MVETestLocationIDNum, mveNameNew, costCentreNew, mveName, mveName, mveNameNew, costCentreNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveNameNew),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentreNew),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "created_by"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "location_id", fmt.Sprintf("%d", MVETestLocationIDNum)),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.vendor", "aruba"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.product_size", "MEDIUM"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.#", "3"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.0.description", "Data Plane"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.1.description", "Management Plane"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.2.description", "Control Plane"),

					// Check data source results with updated values
					resource.TestCheckResourceAttr("data.megaport_mves.test_name_filter", "uids.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_mves.test_tag_filter", "uids.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_mves.test_cost_centre_filter", "uids.#", "1"),
				),
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
                    product_name             = "%s"
                    location_id              = data.megaport_location.test_location.id
                    contract_term_months     = 1
                    cost_centre              = "%s"

                    resource_tags = {
                        "key1" = "value1"
                        "key2" = "value2"
                    }

                    vnics = [
                        {
                            description = "Data Plane"
                        },
                        {
                            description = "Management Plane"
                        },
                        {
                            description = "Control Plane"
                        }
                    ]

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
                  }
                  
                  # Test MVE data source with name filter
                  data "megaport_mves" "test_name_filter" {
                    filter {
                      name = "name"
                      values = ["%s"]
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  
                  # Test MVE data source with vendor filter
                  data "megaport_mves" "test_vendor_filter" {
                    filter {
                      name = "vendor"
                      values = ["versa"]
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  
                  # Test MVE data source with location filter
                  data "megaport_mves" "test_location_filter" {
                    filter {
                      name = "location-id"
                      values = ["%d"]
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  
                  # Test MVE data source with multiple filters
                  data "megaport_mves" "test_multi_filter" {
                    filter {
                      name = "vendor"
                      values = ["versa"]
                    }
                    filter {
                      name = "size"
                      values = ["SMALL"]
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  
                  # Test MVE data source with tags
                  data "megaport_mves" "test_tag_filter" {
                    tags = {
                      "key1" = "value1"
                      "key2" = "value2"
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  `, MVETestLocationIDNum, mveName, costCentre, mveName, MVETestLocationIDNum),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveName),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentre),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "created_by"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "location_id", fmt.Sprintf("%d", MVETestLocationIDNum)),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.vendor", "versa"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.product_size", "SMALL"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.director_address", "0.0.0.0"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.controller_address", "0.0.0.0"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.local_auth", "test"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.remote_auth", "test2"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.serial_number", "test-serial-number"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.#", "3"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.0.description", "Data Plane"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.1.description", "Management Plane"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.2.description", "Control Plane"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key2", "value2"),

					// Check data source results
					resource.TestCheckResourceAttr("data.megaport_mves.test_name_filter", "uids.#", "1"),
					resource.TestCheckResourceAttrSet("data.megaport_mves.test_vendor_filter", "uids.#"),
					resource.TestCheckResourceAttrSet("data.megaport_mves.test_location_filter", "uids.#"),
					resource.TestCheckResourceAttrSet("data.megaport_mves.test_multi_filter", "uids.#"),
					resource.TestCheckResourceAttr("data.megaport_mves.test_tag_filter", "uids.#", "1"),
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
				ImportStateVerifyIgnore: []string{"last_updated", "vendor_config", "contract_start_date", "contract_end_date", "live_date", "provisioning_status"},
			},
			// Update Test
			{
				Config: providerConfig + fmt.Sprintf(`
                data "megaport_location" "test_location" {
                    id = %d
                }
                  resource "megaport_mve" "mve" {
                    product_name             = "%s"
                    location_id              = data.megaport_location.test_location.id
                    contract_term_months     = 1
                    cost_centre              = "%s"

                    resource_tags = {
                        "key1updated" = "value1updated"
                        "key2updated" = "value2updated"
                    }

                    vnics = [
                        {
                            description = "Data Plane"
                        },
                        {
                            description = "Management Plane"
                        },
                        {
                            description = "Control Plane"
                        }
                    ]

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
                  }
                  
                  # Test MVE data source with updated name filter
                  data "megaport_mves" "test_name_filter" {
                    filter {
                      name = "name"
                      values = ["%s"]
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  
                  # Test MVE data source with updated tags
                  data "megaport_mves" "test_tag_filter" {
                    tags = {
                      "key1updated" = "value1updated"
                      "key2updated" = "value2updated"
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  
                  # Test MVE data source with cost-centre filter
                  data "megaport_mves" "test_cost_centre_filter" {
                    filter {
                      name = "cost-centre"
                      values = ["%s"]
                    }
                    depends_on = [megaport_mve.mve]
                  }
                  `, MVETestLocationIDNum, mveNameNew, costCentreNew, mveNameNew, costCentreNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveNameNew),
					resource.TestCheckResourceAttr("megaport_mve.mve", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "cost_centre", costCentreNew),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "created_by"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "location_id", fmt.Sprintf("%d", MVETestLocationIDNum)),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.vendor", "versa"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vendor_config.product_size", "SMALL"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.#", "3"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.0.description", "Data Plane"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.1.description", "Management Plane"),
					resource.TestCheckResourceAttr("megaport_mve.mve", "vnics.2.description", "Control Plane"),

					// Check data source results with updated values
					resource.TestCheckResourceAttr("data.megaport_mves.test_name_filter", "uids.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_mves.test_tag_filter", "uids.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_mves.test_cost_centre_filter", "uids.#", "1"),
				),
			},
		},
	})
}
