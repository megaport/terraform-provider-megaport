package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"

	megaport "github.com/megaport/megaportgo"
)

// The data source decodes entries in its own loop, so resolveGeLe has to be
// exercised through Read rather than through the resource model.
func TestReadMCRPrefixFilterListDataSource_ResolvesAbsentBounds(t *testing.T) {
	ctx := context.Background()
	mockMCRService := &MockMCRService{
		ListMCRPrefixFilterListsResult: []*megaport.PrefixFilterList{
			{Id: 1, Description: "absent bounds", AddressFamily: "IPv4"},
		},
		GetMCRPrefixFilterListResult: map[int]*megaport.MCRPrefixFilterList{
			1: {
				ID:            1,
				Description:   "absent bounds",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					// Exact match: the API omits both bounds.
					{Action: "permit", Prefix: "10.0.0.0/24"},
					// ge above the prefix length: the API omits le only.
					{Action: "deny", Prefix: "192.168.0.0/16", Ge: 24},
				},
			},
		},
	}
	ds := &mcrPrefixFilterListDataSource{client: &megaport.Client{MCRService: mockMCRService}}

	schemaResp := datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	attrValues := make(map[string]tftypes.Value, len(schemaResp.Schema.Attributes))
	for name, attr := range schemaResp.Schema.Attributes {
		attrValues[name] = tftypes.NewValue(attr.GetType().TerraformType(ctx), nil)
	}
	attrValues["mcr_id"] = tftypes.NewValue(tftypes.String, "mcr-1")
	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), attrValues),
		},
	}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	ds.Read(ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)

	var state mcrPrefixFilterListDataSourceModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	var lists []mcrPrefixFilterListDataModel
	require.False(t, state.PrefixFilterLists.ElementsAs(ctx, &lists, false).HasError())
	require.Len(t, lists, 1)

	var entries []mcrPrefixFilterListEntryResourceModel
	require.False(t, lists[0].Entries.ElementsAs(ctx, &entries, false).HasError())
	require.Len(t, entries, 2)

	// Both absent resolves to the prefix length, not the family maximum.
	require.Equal(t, int64(24), entries[0].Ge.ValueInt64())
	require.Equal(t, int64(24), entries[0].Le.ValueInt64())
	// An absent le with ge set resolves to the family maximum.
	require.Equal(t, int64(24), entries[1].Ge.ValueInt64())
	require.Equal(t, int64(32), entries[1].Le.ValueInt64())
}

func TestAccMegaportMCRPrefixFilterListDataSource_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	prefixFilterName := RandomTestName()
	prefixFilterName2 := RandomTestName()
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
					product_name         = "%s"
					port_speed          = 1000
					location_id         = data.megaport_location.test_location.id
					contract_term_months = 12
					cost_centre         = "%s"

					# Explicitly set empty prefix filter lists to avoid conflicts
					prefix_filter_lists = []

					lifecycle {
						ignore_changes = [prefix_filter_lists]
					}
				}

				resource "megaport_mcr_prefix_filter_list" "prefix_list_1" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv4"
					entries = [
						{
							action = "permit"
							prefix = "10.0.1.0/24"
							ge     = 25
							le     = 32
						},
						{
							action = "deny"
							prefix = "10.0.2.0/24"
							ge     = 25
							le     = 27
						},
						{
							action = "permit"
							prefix = "10.0.3.0/24"
							ge     = 24
							le     = 24
						}
					]
				}

				resource "megaport_mcr_prefix_filter_list" "prefix_list_2" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv6"
					entries = [
						{
							action = "permit"
							prefix = "2001:db8::/32"
							ge     = 48
							le     = 64
						}
					]
				}

				data "megaport_mcr_prefix_filter_lists" "all_lists" {
					mcr_id = megaport_mcr.mcr.product_uid
					depends_on = [
						megaport_mcr_prefix_filter_list.prefix_list_1,
						megaport_mcr_prefix_filter_list.prefix_list_2
					]
				}
				`, locationID, mcrName, costCentreName, prefixFilterName, prefixFilterName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Data source checks
					resource.TestCheckResourceAttr("data.megaport_mcr_prefix_filter_lists.all_lists", "prefix_filter_lists.#", "2"),

					// Check that we can find our created prefix filter lists
					// The exact match entry comes back with ge and le absent, so this is
					// what covers the data source read resolving them.
					resource.TestCheckTypeSetElemNestedAttrs("data.megaport_mcr_prefix_filter_lists.all_lists", "prefix_filter_lists.*", map[string]string{
						"description":      prefixFilterName,
						"address_family":   "IPv4",
						"entries.2.prefix": "10.0.3.0/24",
						"entries.2.ge":     "24",
						"entries.2.le":     "24",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.megaport_mcr_prefix_filter_lists.all_lists", "prefix_filter_lists.*", map[string]string{
						"description":    prefixFilterName2,
						"address_family": "IPv6",
					}),
				),
			},
		},
	})
}
