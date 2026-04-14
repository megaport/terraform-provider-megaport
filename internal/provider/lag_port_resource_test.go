package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestFromAPILagPort_Full(t *testing.T) {
	ctx := context.Background()
	apiPort := &megaport.Port{
		UID:                   "lag-uid-123",
		Name:                  "Test LAG Port",
		PortSpeed:             10000,
		LocationID:            42,
		MarketplaceVisibility: true,
		CompanyUID:            "company-uid-456",
		CostCentre:            "cost-centre-1",
		ContractTermMonths:    24,
		DiversityZone:         "blue",
		LagCount:              4,
		VXCResources: megaport.PortResources{
			Interface: megaport.PortInterface{
				Demarcation: "LAG Demarcation",
				Up:          1,
			},
		},
	}
	tags := map[string]string{"env": "prod"}

	model := &lagPortResourceModel{}
	diags := model.fromAPIPort(ctx, apiPort, tags)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	assert.Equal(t, "lag-uid-123", model.UID.ValueString())
	assert.Equal(t, "Test LAG Port", model.Name.ValueString())
	assert.Equal(t, int64(10000), model.PortSpeed.ValueInt64())
	assert.Equal(t, int64(42), model.LocationID.ValueInt64())
	assert.True(t, model.MarketplaceVisibility.ValueBool())
	assert.Equal(t, "company-uid-456", model.CompanyUID.ValueString())
	assert.Equal(t, "cost-centre-1", model.CostCentre.ValueString())
	assert.Equal(t, int64(24), model.ContractTermMonths.ValueInt64())
	assert.Equal(t, "blue", model.DiversityZone.ValueString())
	assert.Equal(t, int64(4), model.LagCount.ValueInt64())

	// Verify resources object is set
	assert.False(t, model.Resources.IsNull())
	assert.False(t, model.Resources.IsUnknown())

	// Verify resource tags
	assert.False(t, model.ResourceTags.IsNull())
	tagElements := model.ResourceTags.Elements()
	require.Len(t, tagElements, 1)
	assert.Equal(t, "prod", tagElements["env"].(types.String).ValueString())
}

func TestFromAPILagPort_MinimalFields(t *testing.T) {
	ctx := context.Background()
	apiPort := &megaport.Port{
		UID:  "lag-minimal",
		Name: "Minimal LAG",
		VXCResources: megaport.PortResources{
			Interface: megaport.PortInterface{},
		},
	}

	model := &lagPortResourceModel{}
	diags := model.fromAPIPort(ctx, apiPort, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	assert.Equal(t, "lag-minimal", model.UID.ValueString())
	assert.Equal(t, "Minimal LAG", model.Name.ValueString())
	assert.Equal(t, int64(0), model.PortSpeed.ValueInt64())
	assert.Equal(t, int64(0), model.LocationID.ValueInt64())
	assert.False(t, model.MarketplaceVisibility.ValueBool())
	assert.Equal(t, "", model.CompanyUID.ValueString())
	assert.Equal(t, "", model.CostCentre.ValueString())
	assert.Equal(t, int64(0), model.ContractTermMonths.ValueInt64())
	assert.Equal(t, "", model.DiversityZone.ValueString())
	assert.Equal(t, int64(0), model.LagCount.ValueInt64())

	// Resources should still be populated
	assert.False(t, model.Resources.IsNull())

	// Tags should be null
	assert.True(t, model.ResourceTags.IsNull())
}

const (
	LagPortTestLocation      = "NextDC B1"
	LagPortTestLocationIDNum = 5 // "NextDC B1"
)

type LagPortProviderTestSuite ProviderTestSuite

func TestLagPortProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(LagPortProviderTestSuite))
}

func (suite *LagPortProviderTestSuite) TestAccMegaportLAGPort_Basic() {
	portName := RandomTestName()
	costCentreName := RandomTestName()
	portNameNew := RandomTestName()
	costCentreNameNew := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
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
			      }`, LagPortTestLocationIDNum, portName, costCentreName),
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
				ImportStateVerifyIgnore: []string{"lag_count", "lag_port_uids", "resources"},
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
			      }`, LagPortTestLocationIDNum, portNameNew, costCentreNameNew),
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
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_lag_port.lag_port", "company_uid"),
				),
			},
		},
	})
}

func (suite *LagPortProviderTestSuite) TestAccMegaportLAGPort_CostCentreRemoval() {
	portName := RandomTestName()
	costCentreName := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
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
				}`, LagPortTestLocationIDNum, portName, costCentreName),
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
				}`, LagPortTestLocationIDNum, portName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "cost_centre", ""),
				),
			},
		},
	})
}

func (suite *LagPortProviderTestSuite) TestAccMegaportLAGPort_ContractTermUpdate() {
	portName := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
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
					contract_term_months = 1
					marketplace_visibility = false
					lag_count = 1
				}`, LagPortTestLocationIDNum, portName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "contract_term_months", "1"),
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
					contract_term_months = 12
					marketplace_visibility = false
					lag_count = 1
				}`, LagPortTestLocationIDNum, portName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_lag_port.lag_port", "contract_term_months", "12"),
				),
			},
		},
	})
}
