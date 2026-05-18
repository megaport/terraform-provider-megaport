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

// --- Acceptance tests ---

func TestAccMegaportMCR_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	prefixFilterName := RandomTestName()
	prefixFilterName2 := RandomTestName()
	prefixFilterNameNew := RandomTestName()
	prefixFilterNameNew2 := RandomTestName()
	prefixFilterNameNew3 := RandomTestName()
	prefixFilterNameNew4 := RandomTestName()
	costCentreName := RandomTestName()
	mcrNameNew := RandomTestName()
	mcrNameNew2 := RandomTestName()
	costCentreNameNew := RandomTestName()
	costCentreNameNew2 := RandomTestName()
	resource.Test(t, resource.TestCase{
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

					prefix_filter_lists = [
					{
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 25
							le      = 32
						  },
						  {
							action  = "deny"
							prefix  = "10.0.2.0/24"
							ge      = 25
							le      = 27
						  }
						]
					  },
					  {
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 26
							le      = 32
						  },
						  {
							action  = "deny"
							prefix  = "10.0.2.0/24"
							ge      = 24
							le      = 25
						  }
						]
					  }]
				  }
				  `, locationID, mcrName, costCentreName, prefixFilterName, prefixFilterName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.description", prefixFilterName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.description", prefixFilterName2),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.le", "27"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.ge", "26"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.le", "25"),
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "provisioning_status"},
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
					resource_tags = {
						"key1updated" = "value1updated"
						"key2updated" = "value2updated"
					}

					prefix_filter_lists = [
					{
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 24
							le      = 32
						  },
						  {
							action  = "deny"
							prefix  = "10.0.2.0/24"
							ge      = 25
							le      = 29
						  }
						]
					  },
					  {
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 25
							le      = 32
						  },
						  {
							action  = "deny"
							prefix  = "10.0.2.0/24"
							ge      = 24
							le      = 26
						  }
						]
					  },
					  {
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 24
							le      = 24
						  },
						  {
							action  = "deny"
							prefix  = "10.0.2.0/24"
							ge      = 27
							le      = 32
						  }
						]
					  }]
				  }
				  `, locationID, mcrName, costCentreName, prefixFilterNameNew, prefixFilterNameNew2, prefixFilterNameNew3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.#", "3"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.description", prefixFilterNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.description", prefixFilterNameNew2),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.description", prefixFilterNameNew3),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.1.le", "29"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.1.entries.1.le", "26"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.0.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.0.le", "24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.1.ge", "27"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.2.entries.1.le", "32"),
				),
			},
			// Update Test 2
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

					prefix_filter_lists = [{
						description     = "%s"
						address_family  = "IPv4"
						entries = [
						  {
							action  = "permit"
							prefix  = "10.0.1.0/24"
							ge      = 28
							le      = 32
						  }
						]
					  }]
				  }
				  `, locationID, mcrNameNew, costCentreNameNew, prefixFilterNameNew4),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.description", prefixFilterNameNew4),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.#", "1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.#", "1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.ge", "28"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.0.entries.0.le", "32"),
				),
			},
			// Update Test 3
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

					prefix_filter_lists = []
				  }
				  `, locationID, mcrNameNew2, costCentreNameNew2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrNameNew2),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreNameNew2),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "prefix_filter_lists.#", "0"),
				),
			},
		},
	})
}

func TestAccMegaportMCR_CostCentreRemoval(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	costCentreName := RandomTestName()
	resource.Test(t, resource.TestCase{
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
				}`, locationID, mcrName, costCentreName),
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
				}`, locationID, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", ""),
				),
			},
		},
	})
}

func TestAccMegaportMCR_ContractTermUpdate(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	resource.Test(t, resource.TestCase{
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
					contract_term_months = 12
				}`, locationID, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					waitForProvisioningStatus("megaport_mcr.mcr"),
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
					contract_term_months = 24
				}`, locationID, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "24"),
				),
			},
		},
	})
}

func TestAccMegaportMCRCustomASN_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	mcrNameNew := RandomTestName()
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
				  `, locationID, mcrName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
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
				ImportStateVerifyIgnore: []string{"last_updated", "contract_start_date", "contract_end_date", "live_date", "provisioning_status"},
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
				  `, locationID, mcrNameNew, costCentreNameNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2updated", "value2updated"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "asn", "65000"),
				),
			},
		},
	})
}

