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

// --- Unit tests for fromAPIMCR ---

func TestFromAPIMCR_Full(t *testing.T) {
	ctx := context.Background()
	apiMCR := &megaport.MCR{
		UID:                   "mcr-uid-123",
		Name:                  "Test MCR",
		CostCentre:            "CC-100",
		PortSpeed:             5000,
		LocationID:            123,
		MarketplaceVisibility: true,
		CompanyUID:            "company-uid-456",
		ContractTermMonths:    12,
		DiversityZone:         "blue",
		Resources: megaport.MCRResources{
			VirtualRouter: megaport.MCRVirtualRouter{
				ASN: 64512,
			},
		},
		AttributeTags: map[string]string{
			"account": "prod",
			"team":    "network",
		},
	}
	tags := map[string]string{"env": "test", "owner": "ci"}

	model := &mcrResourceModel{}
	diags := model.fromAPIMCR(ctx, apiMCR, tags)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	assert.Equal(t, "mcr-uid-123", model.UID.ValueString())
	assert.Equal(t, "Test MCR", model.Name.ValueString())
	assert.Equal(t, "CC-100", model.CostCentre.ValueString())
	assert.Equal(t, int64(5000), model.PortSpeed.ValueInt64())
	assert.Equal(t, int64(123), model.LocationID.ValueInt64())
	assert.True(t, model.MarketplaceVisibility.ValueBool())
	assert.Equal(t, "company-uid-456", model.CompanyUID.ValueString())
	assert.Equal(t, int64(12), model.ContractTermMonths.ValueInt64())
	assert.Equal(t, "blue", model.DiversityZone.ValueString())
	assert.Equal(t, int64(64512), model.ASN.ValueInt64())

	// Verify attribute tags
	assert.False(t, model.AttributeTags.IsNull())
	attrTags := model.AttributeTags.Elements()
	assert.Len(t, attrTags, 2)
	assert.Equal(t, "prod", attrTags["account"].(types.String).ValueString())
	assert.Equal(t, "network", attrTags["team"].(types.String).ValueString())

	// Verify resource tags
	assert.False(t, model.ResourceTags.IsNull())
	resTags := model.ResourceTags.Elements()
	assert.Len(t, resTags, 2)
	assert.Equal(t, "test", resTags["env"].(types.String).ValueString())
	assert.Equal(t, "ci", resTags["owner"].(types.String).ValueString())
}

func TestFromAPIMCR_MinimalFields(t *testing.T) {
	ctx := context.Background()
	apiMCR := &megaport.MCR{
		UID:       "mcr-minimal",
		Name:      "Minimal",
		PortSpeed: 1000,
	}

	model := &mcrResourceModel{}
	diags := model.fromAPIMCR(ctx, apiMCR, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	assert.Equal(t, "mcr-minimal", model.UID.ValueString())
	assert.Equal(t, "Minimal", model.Name.ValueString())
	assert.Equal(t, "", model.CostCentre.ValueString())
	assert.Equal(t, int64(1000), model.PortSpeed.ValueInt64())
	assert.Equal(t, int64(0), model.LocationID.ValueInt64())
	assert.False(t, model.MarketplaceVisibility.ValueBool())
	assert.Equal(t, "", model.CompanyUID.ValueString())
	assert.Equal(t, int64(0), model.ContractTermMonths.ValueInt64())
	assert.Equal(t, "", model.DiversityZone.ValueString())

	// ASN should be null when zero
	assert.True(t, model.ASN.IsNull())
	// Resource tags should be null when nil tags
	assert.True(t, model.ResourceTags.IsNull())
}

func TestFromAPIMCR_ZeroASN(t *testing.T) {
	ctx := context.Background()
	apiMCR := &megaport.MCR{
		UID:  "mcr-zero-asn",
		Name: "Zero ASN",
		Resources: megaport.MCRResources{
			VirtualRouter: megaport.MCRVirtualRouter{
				ASN: 0,
			},
		},
	}

	model := &mcrResourceModel{}
	diags := model.fromAPIMCR(ctx, apiMCR, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	assert.True(t, model.ASN.IsNull(), "ASN should be null when API returns 0")
}

func TestFromAPIMCR_NonZeroASN(t *testing.T) {
	ctx := context.Background()
	apiMCR := &megaport.MCR{
		UID:  "mcr-nonzero-asn",
		Name: "NonZero ASN",
		Resources: megaport.MCRResources{
			VirtualRouter: megaport.MCRVirtualRouter{
				ASN: 64512,
			},
		},
	}

	model := &mcrResourceModel{}
	diags := model.fromAPIMCR(ctx, apiMCR, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	assert.False(t, model.ASN.IsNull(), "ASN should not be null when non-zero")
	assert.Equal(t, int64(64512), model.ASN.ValueInt64())
}

func TestFromAPIMCR_NilTags(t *testing.T) {
	ctx := context.Background()
	apiMCR := &megaport.MCR{
		UID:  "mcr-nil-tags",
		Name: "Nil Tags",
	}

	model := &mcrResourceModel{}
	diags := model.fromAPIMCR(ctx, apiMCR, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	assert.True(t, model.ResourceTags.IsNull(), "resource tags should be null when tags is nil")
}

func TestFromAPIMCR_EmptyTags(t *testing.T) {
	ctx := context.Background()
	apiMCR := &megaport.MCR{
		UID:  "mcr-empty-tags",
		Name: "Empty Tags",
	}
	tags := map[string]string{}

	model := &mcrResourceModel{}
	diags := model.fromAPIMCR(ctx, apiMCR, tags)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	assert.True(t, model.ResourceTags.IsNull(), "resource tags should be null when tags map is empty")
}

func TestFromAPIMCR_AttributeTags(t *testing.T) {
	ctx := context.Background()
	apiMCR := &megaport.MCR{
		UID:  "mcr-attr-tags",
		Name: "Attr Tags",
		AttributeTags: map[string]string{
			"account": "prod",
			"region":  "us-west",
			"tier":    "premium",
		},
	}

	model := &mcrResourceModel{}
	diags := model.fromAPIMCR(ctx, apiMCR, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	assert.False(t, model.AttributeTags.IsNull(), "attribute tags should not be null")
	attrTags := model.AttributeTags.Elements()
	assert.Len(t, attrTags, 3)
	assert.Equal(t, "prod", attrTags["account"].(types.String).ValueString())
	assert.Equal(t, "us-west", attrTags["region"].(types.String).ValueString())
	assert.Equal(t, "premium", attrTags["tier"].(types.String).ValueString())
}

func TestFromAPIMCR_EmptyAttributeTags(t *testing.T) {
	ctx := context.Background()
	apiMCR := &megaport.MCR{
		UID:           "mcr-empty-attr",
		Name:          "Empty Attr Tags",
		AttributeTags: nil,
	}

	model := &mcrResourceModel{}
	diags := model.fromAPIMCR(ctx, apiMCR, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	// When AttributeTags is nil, the field is not set, so it remains the zero value
	assert.True(t, model.AttributeTags.IsNull(), "attribute tags should be null when API returns nil")
}

const (
	MCRTestLocation      = "Digital Realty Silicon Valley SJC34 (SCL2)"
	MCRTestLocationIDNum = 65 // "Digital Realty Silicon Valley SJC34 (SCL2)"
)

type MCRBasicProviderTestSuite ProviderTestSuite

func TestMCRBasicProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRBasicProviderTestSuite))
}

func (suite *MCRBasicProviderTestSuite) TestAccMegaportMCR_Basic() {
	mcrName := RandomTestName()
	costCentreName := RandomTestName()
	mcrNameNew := RandomTestName()
	costCentreNameNew := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
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
				  }
				  `, MCRTestLocationIDNum, mcrName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
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
				ImportStateVerifyIgnore: []string{},
			},
			// Update Test
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
				  }
				  `, MCRTestLocationIDNum, mcrNameNew, costCentreNameNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
				),
			},
		},
	})
}

type MCRCostCentreProviderTestSuite ProviderTestSuite

func TestMCRCostCentreProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRCostCentreProviderTestSuite))
}

func (suite *MCRCostCentreProviderTestSuite) TestAccMegaportMCR_CostCentreRemoval() {
	mcrName := RandomTestName()
	costCentreName := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
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
				}`, MCRTestLocationIDNum, mcrName, costCentreName),
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
				}`, MCRTestLocationIDNum, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", ""),
				),
			},
		},
	})
}

type MCRContractTermProviderTestSuite ProviderTestSuite

func TestMCRContractTermProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRContractTermProviderTestSuite))
}

func (suite *MCRContractTermProviderTestSuite) TestAccMegaportMCR_ContractTermUpdate() {
	mcrName := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
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
				}`, MCRTestLocationIDNum, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "1"),
					waitForProvisioningStatus("megaport_mcr.mcr"),
				),
			},
			{
				// contract_term_months has RequiresReplace — changing it
				// destroys and recreates the MCR. Verify the new resource
				// has the updated value.
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				resource "megaport_mcr" "mcr" {
					product_name = "%s"
					port_speed = 1000
					location_id = data.megaport_location.test_location.id
					contract_term_months = 12
				}`, MCRTestLocationIDNum, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					waitForProvisioningStatus("megaport_mcr.mcr"),
				),
			},
		},
	})
}

type MCRCustomASNProviderTestSuite ProviderTestSuite

func TestMCRCustomASNProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRCustomASNProviderTestSuite))
}

func (suite *MCRCustomASNProviderTestSuite) TestAccMegaportMCRCustomASN_Basic() {
	mcrName := RandomTestName()
	mcrNameNew := RandomTestName()
	costCentreName := RandomTestName()
	costCentreNameNew := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
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
				  `, MCRTestLocationIDNum, mcrName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
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
				ImportStateVerifyIgnore: []string{},
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
				  `, MCRTestLocationIDNum, mcrNameNew, costCentreNameNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "asn", "65000"),
				),
			},
		},
	})
}
