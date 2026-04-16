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
)

func TestFromAPIPort_Full(t *testing.T) {
	ctx := context.Background()
	apiPort := &megaport.Port{
		UID:                   "port-uid-123",
		Name:                  "Test Port",
		PortSpeed:             10000,
		LocationID:            42,
		MarketplaceVisibility: true,
		CompanyUID:            "company-uid-456",
		CostCentre:            "cost-centre-1",
		ContractTermMonths:    12,
		DiversityZone:         "red",
		VXCResources: megaport.PortResources{
			Interface: megaport.PortInterface{
				Demarcation: "Test Demarcation",
				Up:          1,
			},
		},
	}
	tags := map[string]string{"env": "test", "team": "platform"}

	model := &singlePortResourceModel{}
	diags := model.fromAPIPort(ctx, apiPort, tags)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	assert.Equal(t, "port-uid-123", model.UID.ValueString())
	assert.Equal(t, "Test Port", model.Name.ValueString())
	assert.Equal(t, int64(10000), model.PortSpeed.ValueInt64())
	assert.Equal(t, int64(42), model.LocationID.ValueInt64())
	assert.True(t, model.MarketplaceVisibility.ValueBool())
	assert.Equal(t, "company-uid-456", model.CompanyUID.ValueString())
	assert.Equal(t, "cost-centre-1", model.CostCentre.ValueString())
	assert.Equal(t, int64(12), model.ContractTermMonths.ValueInt64())
	assert.Equal(t, "red", model.DiversityZone.ValueString())

	// Verify resources object is set (not null)
	assert.False(t, model.Resources.IsNull())
	assert.False(t, model.Resources.IsUnknown())

	// Verify resource tags
	assert.False(t, model.ResourceTags.IsNull())
	tagElements := model.ResourceTags.Elements()
	require.Len(t, tagElements, 2)
	assert.Equal(t, "test", tagElements["env"].(types.String).ValueString())
	assert.Equal(t, "platform", tagElements["team"].(types.String).ValueString())
}

func TestFromAPIPort_MinimalFields(t *testing.T) {
	ctx := context.Background()
	apiPort := &megaport.Port{
		UID:  "port-minimal",
		Name: "Minimal Port",
		VXCResources: megaport.PortResources{
			Interface: megaport.PortInterface{},
		},
	}

	model := &singlePortResourceModel{}
	diags := model.fromAPIPort(ctx, apiPort, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	assert.Equal(t, "port-minimal", model.UID.ValueString())
	assert.Equal(t, "Minimal Port", model.Name.ValueString())
	assert.Equal(t, int64(0), model.PortSpeed.ValueInt64())
	assert.Equal(t, int64(0), model.LocationID.ValueInt64())
	assert.False(t, model.MarketplaceVisibility.ValueBool())
	assert.Equal(t, "", model.CompanyUID.ValueString())
	assert.Equal(t, "", model.CostCentre.ValueString())
	assert.Equal(t, int64(0), model.ContractTermMonths.ValueInt64())
	assert.Equal(t, "", model.DiversityZone.ValueString())

	// Resources should still be populated (zero-value interface)
	assert.False(t, model.Resources.IsNull())

	// Tags should be null for nil input
	assert.True(t, model.ResourceTags.IsNull())
}

func TestFromAPIPort_NilTags(t *testing.T) {
	ctx := context.Background()
	apiPort := &megaport.Port{
		UID:  "port-nil-tags",
		Name: "Port Nil Tags",
		VXCResources: megaport.PortResources{
			Interface: megaport.PortInterface{},
		},
	}

	model := &singlePortResourceModel{}
	diags := model.fromAPIPort(ctx, apiPort, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	assert.True(t, model.ResourceTags.IsNull(), "expected null tags for nil input")
}

func TestFromAPIPort_EmptyTags(t *testing.T) {
	ctx := context.Background()
	apiPort := &megaport.Port{
		UID:  "port-empty-tags",
		Name: "Port Empty Tags",
		VXCResources: megaport.PortResources{
			Interface: megaport.PortInterface{},
		},
	}

	model := &singlePortResourceModel{}
	diags := model.fromAPIPort(ctx, apiPort, map[string]string{})
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	assert.True(t, model.ResourceTags.IsNull(), "expected null tags for empty map")
}

func TestAccMegaportSinglePort_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findPortTestLocation(t, 1000)
	portName := RandomTestName()
	portNameNew := RandomTestName()
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
					resource "megaport_port" "port" {
			        product_name  = "%s"
			        port_speed  = 1000
					cost_centre = "%s"
			        location_id = data.megaport_location.test_location.id
			        contract_term_months        = 12
					marketplace_visibility = true
					diversity_zone = "red"

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}
			      }`, locationID, portName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "product_name", portName),
					resource.TestCheckResourceAttr("megaport_port.port", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port", "marketplace_visibility", "true"),
					resource.TestCheckResourceAttr("megaport_port.port", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_port.port", "diversity_zone", "red"),
					resource.TestCheckResourceAttr("megaport_port.port", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_port.port", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "company_uid"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_port.port",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_port.port"
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
				ImportStateVerifyIgnore: []string{"resources"},
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
					resource "megaport_port" "port" {
			        product_name  = "%s"
			        port_speed  = 1000
					cost_centre = "%s"
			        location_id = data.megaport_location.test_location.id
			        contract_term_months        = 12
					marketplace_visibility = false
					diversity_zone = "red"
					resource_tags = {
						"key1-updated" = "value1-updated"
						"key2-updated" = "value2-updated"
					}
			      }`, locationID, portNameNew, costCentreNameNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "product_name", portNameNew),
					resource.TestCheckResourceAttr("megaport_port.port", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_port.port", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_port.port", "diversity_zone", "red"),
					resource.TestCheckResourceAttr("megaport_port.port", "resource_tags.key1-updated", "value1-updated"),
					resource.TestCheckResourceAttr("megaport_port.port", "resource_tags.key2-updated", "value2-updated"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_port.port", "company_uid"),
				),
			},
		},
	})
}

func TestAccMegaportSinglePort_CostCentreRemoval(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findPortTestLocation(t, 1000)
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
				resource "megaport_port" "port" {
					product_name  = "%s"
					port_speed  = 1000
					cost_centre = "%s"
					location_id = data.megaport_location.test_location.id
					contract_term_months = 1
					marketplace_visibility = false
				}`, locationID, portName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "cost_centre", costCentreName),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_port" "port" {
					product_name  = "%s"
					port_speed  = 1000
					cost_centre = ""
					location_id = data.megaport_location.test_location.id
					contract_term_months = 1
					marketplace_visibility = false
				}`, locationID, portName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "cost_centre", ""),
				),
			},
		},
	})
}

func TestAccMegaportSinglePort_ContractTermUpdate(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findPortTestLocation(t, 1000)
	portName := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_port" "port" {
					product_name  = "%s"
					port_speed  = 1000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 12
					marketplace_visibility = false
				}`, locationID, portName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "contract_term_months", "12"),
					waitForProvisioningStatus("megaport_port.port"),
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_port" "port" {
					product_name  = "%s"
					port_speed  = 1000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 24
					marketplace_visibility = false
				}`, locationID, portName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port", "contract_term_months", "24"),
				),
			},
		},
	})
}
