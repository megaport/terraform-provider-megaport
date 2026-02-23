package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

type ServiceKeyProviderTestSuite ProviderTestSuite

func TestServiceKeyProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ServiceKeyProviderTestSuite))
}

func (suite *ServiceKeyProviderTestSuite) TestAccMegaportServiceKey_MultiUse() {
	portName := RandomTestName()
	costCentreName := RandomTestName()
	keyDescription := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create port + multi-use service key
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				resource "megaport_port" "port" {
					product_name         = "%s"
					port_speed           = 1000
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
					marketplace_visibility = false
					cost_centre          = "%s"
				}

				resource "megaport_service_key" "test" {
					product_uid = megaport_port.port.product_uid
					description = "%s"
					max_speed   = 500
					single_use  = false
					active      = true
				}
				`, SinglePortTestLocationIDNum, portName, costCentreName, keyDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_service_key.test", "description", keyDescription),
					resource.TestCheckResourceAttr("megaport_service_key.test", "max_speed", "500"),
					resource.TestCheckResourceAttr("megaport_service_key.test", "single_use", "false"),
					resource.TestCheckResourceAttr("megaport_service_key.test", "active", "true"),
					resource.TestCheckResourceAttrSet("megaport_service_key.test", "key"),
					resource.TestCheckResourceAttrSet("megaport_service_key.test", "product_name"),
					resource.TestCheckResourceAttrSet("megaport_service_key.test", "company_id"),
					resource.TestCheckResourceAttrSet("megaport_service_key.test", "company_uid"),
					resource.TestCheckResourceAttrSet("megaport_service_key.test", "company_name"),
					resource.TestCheckResourceAttrSet("megaport_service_key.test", "create_date"),
				),
			},
			// Step 2: Import by key string
			{
				ResourceName:                         "megaport_service_key.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "key",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["megaport_service_key.test"]
					if !ok {
						return "", fmt.Errorf("megaport_service_key.test not found in state")
					}
					return rs.Primary.Attributes["key"], nil
				},
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			// Step 3: Update active to false (deactivate)
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				resource "megaport_port" "port" {
					product_name         = "%s"
					port_speed           = 1000
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
					marketplace_visibility = false
					cost_centre          = "%s"
				}

				resource "megaport_service_key" "test" {
					product_uid = megaport_port.port.product_uid
					description = "%s"
					max_speed   = 500
					single_use  = false
					active      = false
				}
				`, SinglePortTestLocationIDNum, portName, costCentreName, keyDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_service_key.test", "active", "false"),
					resource.TestCheckResourceAttr("megaport_service_key.test", "description", keyDescription),
				),
			},
			// Step 4: Update active back to true (reactivate)
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				resource "megaport_port" "port" {
					product_name         = "%s"
					port_speed           = 1000
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
					marketplace_visibility = false
					cost_centre          = "%s"
				}

				resource "megaport_service_key" "test" {
					product_uid = megaport_port.port.product_uid
					description = "%s"
					max_speed   = 500
					single_use  = false
					active      = true
				}
				`, SinglePortTestLocationIDNum, portName, costCentreName, keyDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_service_key.test", "active", "true"),
				),
			},
		},
	})
}

func (suite *ServiceKeyProviderTestSuite) TestAccMegaportServiceKey_SingleUseWithVLAN() {
	portName := RandomTestName()
	costCentreName := RandomTestName()
	keyDescription := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				resource "megaport_port" "port" {
					product_name         = "%s"
					port_speed           = 1000
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
					marketplace_visibility = false
					cost_centre          = "%s"
				}

				resource "megaport_service_key" "test_single" {
					product_uid = megaport_port.port.product_uid
					description = "%s"
					max_speed   = 100
					single_use  = true
					active      = true
					vlan        = 100
				}
				`, SinglePortTestLocationIDNum, portName, costCentreName, keyDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_service_key.test_single", "single_use", "true"),
					resource.TestCheckResourceAttr("megaport_service_key.test_single", "vlan", "100"),
					resource.TestCheckResourceAttr("megaport_service_key.test_single", "max_speed", "100"),
					resource.TestCheckResourceAttr("megaport_service_key.test_single", "active", "true"),
					resource.TestCheckResourceAttrSet("megaport_service_key.test_single", "key"),
				),
			},
		},
	})
}

func (suite *ServiceKeyProviderTestSuite) TestAccMegaportServiceKey_ValidFor() {
	portName := RandomTestName()
	costCentreName := RandomTestName()
	keyDescription := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with valid_for
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				resource "megaport_port" "port" {
					product_name         = "%s"
					port_speed           = 1000
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
					marketplace_visibility = false
					cost_centre          = "%s"
				}

				resource "megaport_service_key" "test_valid_for" {
					product_uid = megaport_port.port.product_uid
					description = "%s"
					max_speed   = 500
					single_use  = false
					active      = true

					valid_for = {
						start_time = "2025-01-01T00:00:00Z"
						end_time   = "2026-12-31T23:59:59Z"
					}
				}
				`, SinglePortTestLocationIDNum, portName, costCentreName, keyDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_service_key.test_valid_for", "active", "true"),
					resource.TestCheckResourceAttrSet("megaport_service_key.test_valid_for", "valid_for.start_time"),
					resource.TestCheckResourceAttrSet("megaport_service_key.test_valid_for", "valid_for.end_time"),
				),
			},
			// Step 2: Update valid_for end_time (in-place update, no replace)
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				resource "megaport_port" "port" {
					product_name         = "%s"
					port_speed           = 1000
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
					marketplace_visibility = false
					cost_centre          = "%s"
				}

				resource "megaport_service_key" "test_valid_for" {
					product_uid = megaport_port.port.product_uid
					description = "%s"
					max_speed   = 500
					single_use  = false
					active      = true

					valid_for = {
						start_time = "2025-01-01T00:00:00Z"
						end_time   = "2027-06-30T23:59:59Z"
					}
				}
				`, SinglePortTestLocationIDNum, portName, costCentreName, keyDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_service_key.test_valid_for", "active", "true"),
					resource.TestCheckResourceAttrSet("megaport_service_key.test_valid_for", "valid_for.start_time"),
					resource.TestCheckResourceAttrSet("megaport_service_key.test_valid_for", "valid_for.end_time"),
				),
			},
		},
	})
}
