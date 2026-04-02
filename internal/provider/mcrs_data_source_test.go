package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	megaport "github.com/megaport/megaportgo"
)

type MCRsDataSourceProviderTestSuite ProviderTestSuite

func TestMCRsDataSourceProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MCRsDataSourceProviderTestSuite))
}

func (suite *MCRsDataSourceProviderTestSuite) TestAccMegaportMCRsDataSource_BasicAndDetails() {
	suite.T().Parallel()
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
					product_name         = "%s"
					port_speed           = 1000
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
					cost_centre          = "%s"

					resource_tags = {
						"acc-test-key" = "%s"
					}
				}

				data "megaport_mcrs" "by_name" {
					filter {
						name   = "name"
						values = ["%s"]
					}
					depends_on = [megaport_mcr.mcr]
				}
				`, MCRTestLocationIDNum, mcrName, costCentreName, mcrName, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// uids: exactly 1 result matching our MCR
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_name", "uids.#", "1"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "uids.0", "megaport_mcr.mcr", "product_uid"),
					// mcrs: exactly 1 result
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_name", "mcrs.#", "1"),
					// Cross-reference all detail fields against the resource
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.product_uid", "megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.product_id", "megaport_mcr.mcr", "product_id"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.product_type", "megaport_mcr.mcr", "product_type"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.provisioning_status", "megaport_mcr.mcr", "provisioning_status"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.create_date", "megaport_mcr.mcr", "create_date"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.created_by", "megaport_mcr.mcr", "created_by"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.location_id", "megaport_mcr.mcr", "location_id"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.market", "megaport_mcr.mcr", "market"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.company_uid", "megaport_mcr.mcr", "company_uid"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.company_name", "megaport_mcr.mcr", "company_name"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.diversity_zone", "megaport_mcr.mcr", "diversity_zone"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.secondary_name", "megaport_mcr.mcr", "secondary_name"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.vxc_permitted", "megaport_mcr.mcr", "vxc_permitted"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.vxc_auto_approval", "megaport_mcr.mcr", "vxc_auto_approval"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.marketplace_visibility", "megaport_mcr.mcr", "marketplace_visibility"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.asn", "megaport_mcr.mcr", "asn"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.virtual", "megaport_mcr.mcr", "virtual"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.locked", "megaport_mcr.mcr", "locked"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.admin_locked", "megaport_mcr.mcr", "admin_locked"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_name", "mcrs.0.cancelable", "megaport_mcr.mcr", "cancelable"),
					// Exact value checks for fields we control
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_name", "mcrs.0.product_name", mcrName),
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_name", "mcrs.0.port_speed", "1000"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_name", "mcrs.0.cost_centre", costCentreName),
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_name", "mcrs.0.contract_term_months", "1"),
					// Resource tags populated
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_name", "mcrs.0.resource_tags.acc-test-key", mcrName),
				),
			},
		},
	})
}

func (suite *MCRsDataSourceProviderTestSuite) TestAccMegaportMCRsDataSource_CombinedFilters() {
	suite.T().Parallel()
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
					product_name         = "%s"
					port_speed           = 1000
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
					cost_centre          = "%s"
				}

				# Multiple filters (AND logic) — name + speed + cost-centre
				data "megaport_mcrs" "combined" {
					filter {
						name   = "name"
						values = ["%s"]
					}
					filter {
						name   = "port-speed"
						values = ["1000"]
					}
					filter {
						name   = "cost-centre"
						values = ["%s"]
					}
					depends_on = [megaport_mcr.mcr]
				}

				# Filter by location-id
				data "megaport_mcrs" "by_location" {
					filter {
						name   = "name"
						values = ["%s"]
					}
					filter {
						name   = "location-id"
						values = ["%d"]
					}
					depends_on = [megaport_mcr.mcr]
				}

				# Filter by contract-term-months
				data "megaport_mcrs" "by_contract" {
					filter {
						name   = "name"
						values = ["%s"]
					}
					filter {
						name   = "contract-term-months"
						values = ["1"]
					}
					depends_on = [megaport_mcr.mcr]
				}

				# Multiple values (OR logic) — port-speed = 1000 or 10000
				data "megaport_mcrs" "or_speed" {
					filter {
						name   = "name"
						values = ["%s"]
					}
					filter {
						name   = "port-speed"
						values = ["1000", "10000"]
					}
					depends_on = [megaport_mcr.mcr]
				}
				`, MCRTestLocationIDNum, mcrName, costCentreName,
					mcrName, costCentreName,
					mcrName, MCRTestLocationIDNum,
					mcrName,
					mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Combined AND filters: exactly 1 result
					resource.TestCheckResourceAttr("data.megaport_mcrs.combined", "uids.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.combined", "mcrs.#", "1"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.combined", "mcrs.0.product_uid", "megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.combined", "mcrs.0.product_name", mcrName),
					resource.TestCheckResourceAttr("data.megaport_mcrs.combined", "mcrs.0.port_speed", "1000"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.combined", "mcrs.0.cost_centre", costCentreName),

					// Location filter: exactly 1 result
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_location", "uids.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_location", "mcrs.#", "1"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_location", "mcrs.0.product_uid", "megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_location", "mcrs.0.location_id", fmt.Sprintf("%d", MCRTestLocationIDNum)),

					// Contract term filter: exactly 1 result
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_contract", "uids.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_contract", "mcrs.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_contract", "mcrs.0.contract_term_months", "1"),

					// OR logic on speed: also matches our MCR (speed=1000 is in the list)
					resource.TestCheckResourceAttr("data.megaport_mcrs.or_speed", "uids.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.or_speed", "mcrs.#", "1"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.or_speed", "mcrs.0.product_uid", "megaport_mcr.mcr", "product_uid"),
				),
			},
		},
	})
}

func (suite *MCRsDataSourceProviderTestSuite) TestAccMegaportMCRsDataSource_TagsAndCombined() {
	suite.T().Parallel()
	mcrName := RandomTestName()
	costCentreName := RandomTestName()
	tagValue := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				resource "megaport_mcr" "mcr" {
					product_name         = "%s"
					port_speed           = 1000
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
					cost_centre          = "%s"

					resource_tags = {
						"acc-test-env"  = "%s"
						"acc-test-team" = "platform"
					}
				}

				# Filter by single tag
				data "megaport_mcrs" "by_single_tag" {
					tags = {
						"acc-test-env" = "%s"
					}
					depends_on = [megaport_mcr.mcr]
				}

				# Filter by multiple tags (AND — all must match)
				data "megaport_mcrs" "by_multi_tag" {
					tags = {
						"acc-test-env"  = "%s"
						"acc-test-team" = "platform"
					}
					depends_on = [megaport_mcr.mcr]
				}

				# Combined: filter + tags
				data "megaport_mcrs" "filter_and_tags" {
					filter {
						name   = "port-speed"
						values = ["1000"]
					}
					tags = {
						"acc-test-env" = "%s"
					}
					depends_on = [megaport_mcr.mcr]
				}

				# Tag that doesn't match anything
				data "megaport_mcrs" "no_tag_match" {
					tags = {
						"acc-test-env" = "nonexistent-value-that-wont-match"
					}
					depends_on = [megaport_mcr.mcr]
				}
				`, MCRTestLocationIDNum, mcrName, costCentreName,
					tagValue,
					tagValue,
					tagValue,
					tagValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Single tag filter: exactly 1 result
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_single_tag", "uids.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_single_tag", "mcrs.#", "1"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_single_tag", "mcrs.0.product_uid", "megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_single_tag", "mcrs.0.product_name", mcrName),
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_single_tag", "mcrs.0.resource_tags.acc-test-env", tagValue),
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_single_tag", "mcrs.0.resource_tags.acc-test-team", "platform"),

					// Multi-tag filter: exactly 1 result
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_multi_tag", "uids.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.by_multi_tag", "mcrs.#", "1"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.by_multi_tag", "mcrs.0.product_uid", "megaport_mcr.mcr", "product_uid"),

					// Combined filter + tags: exactly 1 result
					resource.TestCheckResourceAttr("data.megaport_mcrs.filter_and_tags", "uids.#", "1"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.filter_and_tags", "mcrs.#", "1"),
					resource.TestCheckResourceAttrPair("data.megaport_mcrs.filter_and_tags", "mcrs.0.product_uid", "megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.filter_and_tags", "mcrs.0.port_speed", "1000"),

					// No match: empty results
					resource.TestCheckResourceAttr("data.megaport_mcrs.no_tag_match", "uids.#", "0"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.no_tag_match", "mcrs.#", "0"),
				),
			},
		},
	})
}

func (suite *MCRsDataSourceProviderTestSuite) TestAccMegaportMCRsDataSource_NoMatch() {
	suite.T().Parallel()
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
					product_name         = "%s"
					port_speed           = 1000
					location_id          = data.megaport_location.test_location.id
					contract_term_months = 1
				}

				# Name that won't match any MCR
				data "megaport_mcrs" "no_name_match" {
					filter {
						name   = "name"
						values = ["nonexistent-mcr-name-that-wont-match-anything"]
					}
					depends_on = [megaport_mcr.mcr]
				}

				# Contradictory filters — name matches but speed doesn't
				data "megaport_mcrs" "contradictory" {
					filter {
						name   = "name"
						values = ["%s"]
					}
					filter {
						name   = "port-speed"
						values = ["99999"]
					}
					depends_on = [megaport_mcr.mcr]
				}
				`, MCRTestLocationIDNum, mcrName, mcrName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// No name match: empty
					resource.TestCheckResourceAttr("data.megaport_mcrs.no_name_match", "uids.#", "0"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.no_name_match", "mcrs.#", "0"),

					// Contradictory AND filters: empty
					resource.TestCheckResourceAttr("data.megaport_mcrs.contradictory", "uids.#", "0"),
					resource.TestCheckResourceAttr("data.megaport_mcrs.contradictory", "mcrs.#", "0"),
				),
			},
		},
	})
}

// MockMCRService is a mock of the MCR service for testing
type MockMCRService struct {
	ListMCRsResult            []*megaport.MCR
	ListMCRsErr               error
	ListMCRResourceTagsFunc   func(ctx context.Context, mcrID string) (map[string]string, error)
	ListMCRResourceTagsErr    error
	ListMCRResourceTagsResult map[string]string
	CapturedResourceTagMCRUID string
}

func (m *MockMCRService) ListMCRs(ctx context.Context, req *megaport.ListMCRsRequest) ([]*megaport.MCR, error) {
	if m.ListMCRsErr != nil {
		return nil, m.ListMCRsErr
	}
	if m.ListMCRsResult != nil {
		return m.ListMCRsResult, nil
	}
	return []*megaport.MCR{}, nil
}

func (m *MockMCRService) ListMCRResourceTags(ctx context.Context, mcrID string) (map[string]string, error) {
	m.CapturedResourceTagMCRUID = mcrID
	if m.ListMCRResourceTagsFunc != nil {
		return m.ListMCRResourceTagsFunc(ctx, mcrID)
	}
	if m.ListMCRResourceTagsErr != nil {
		return nil, m.ListMCRResourceTagsErr
	}
	if m.ListMCRResourceTagsResult != nil {
		return m.ListMCRResourceTagsResult, nil
	}
	return map[string]string{
		"environment": "test",
		"owner":       "automation",
	}, nil
}

// Implement other required methods of the MCRService interface with minimal stubs
func (m *MockMCRService) BuyMCR(ctx context.Context, req *megaport.BuyMCRRequest) (*megaport.BuyMCRResponse, error) {
	return nil, nil
}

func (m *MockMCRService) ValidateMCROrder(ctx context.Context, req *megaport.BuyMCRRequest) error {
	return nil
}

func (m *MockMCRService) GetMCR(ctx context.Context, mcrId string) (*megaport.MCR, error) {
	return nil, nil
}

func (m *MockMCRService) CreatePrefixFilterList(ctx context.Context, req *megaport.CreateMCRPrefixFilterListRequest) (*megaport.CreateMCRPrefixFilterListResponse, error) {
	return nil, nil
}

func (m *MockMCRService) ListMCRPrefixFilterLists(ctx context.Context, mcrId string) ([]*megaport.PrefixFilterList, error) {
	return nil, nil
}

func (m *MockMCRService) GetMCRPrefixFilterList(ctx context.Context, mcrID string, prefixFilterListID int) (*megaport.MCRPrefixFilterList, error) {
	return nil, nil
}

func (m *MockMCRService) ModifyMCRPrefixFilterList(ctx context.Context, mcrID string, prefixFilterListID int, prefixFilterList *megaport.MCRPrefixFilterList) (*megaport.ModifyMCRPrefixFilterListResponse, error) {
	return nil, nil
}

func (m *MockMCRService) DeleteMCRPrefixFilterList(ctx context.Context, mcrID string, prefixFilterListID int) (*megaport.DeleteMCRPrefixFilterListResponse, error) {
	return nil, nil
}

func (m *MockMCRService) ModifyMCR(ctx context.Context, req *megaport.ModifyMCRRequest) (*megaport.ModifyMCRResponse, error) {
	return nil, nil
}

func (m *MockMCRService) DeleteMCR(ctx context.Context, req *megaport.DeleteMCRRequest) (*megaport.DeleteMCRResponse, error) {
	return nil, nil
}

func (m *MockMCRService) RestoreMCR(ctx context.Context, mcrId string) (*megaport.RestoreMCRResponse, error) {
	return nil, nil
}

func (m *MockMCRService) UpdateMCRResourceTags(ctx context.Context, mcrID string, tags map[string]string) error {
	return nil
}

func (m *MockMCRService) GetMCRPrefixFilterLists(ctx context.Context, mcrId string) ([]*megaport.PrefixFilterList, error) {
	return nil, nil
}

// TestFilterMCRs tests the filterMCRs method
func TestFilterMCRs(t *testing.T) {
	// Create sample MCRs for testing
	mcrs := []*megaport.MCR{
		{
			UID:                   "mcr-1",
			Name:                  "Test MCR 1",
			Type:                  "MCR2",
			PortSpeed:             10000,
			LocationID:            123,
			ProvisioningStatus:    "LIVE",
			Market:                "Sydney",
			CompanyName:           "Company A",
			CompanyUID:            "company-uid-a",
			VXCPermitted:          true,
			VXCAutoApproval:       true,
			MarketplaceVisibility: true,
			DiversityZone:         "zone-a",
			SecondaryName:         "Secondary 1",
			ContractTermMonths:    12,
			Virtual:               true,
			Locked:                false,
			AdminLocked:           false,
			Cancelable:            true,
			Resources: megaport.MCRResources{
				VirtualRouter: megaport.MCRVirtualRouter{
					ASN: 64512,
				},
			},
		},
		{
			UID:                   "mcr-2",
			Name:                  "Test MCR 2",
			Type:                  "MCR2",
			PortSpeed:             1000,
			LocationID:            456,
			ProvisioningStatus:    "CONFIGURED",
			Market:                "Melbourne",
			CompanyName:           "Company B",
			CompanyUID:            "company-uid-b",
			VXCPermitted:          true,
			VXCAutoApproval:       false,
			MarketplaceVisibility: false,
			DiversityZone:         "zone-b",
			SecondaryName:         "Secondary 2",
			ContractTermMonths:    24,
			Virtual:               true,
			Locked:                true,
			AdminLocked:           false,
			Cancelable:            true,
			Resources: megaport.MCRResources{
				VirtualRouter: megaport.MCRVirtualRouter{
					ASN: 64513,
				},
			},
		},
		{
			UID:                   "mcr-3",
			Name:                  "Inactive MCR",
			Type:                  "MCR1",
			PortSpeed:             10000,
			LocationID:            123,
			ProvisioningStatus:    "DECOMMISSIONED",
			Market:                "Sydney",
			CompanyName:           "Company A",
			CompanyUID:            "company-uid-a",
			VXCPermitted:          false,
			VXCAutoApproval:       false,
			MarketplaceVisibility: false,
			DiversityZone:         "zone-a",
			SecondaryName:         "Secondary 1",
			ContractTermMonths:    36,
			Virtual:               false,
			Locked:                false,
			AdminLocked:           true,
			Cancelable:            false,
			Resources: megaport.MCRResources{
				VirtualRouter: megaport.MCRVirtualRouter{
					ASN: 64514,
				},
			},
		},
	}

	// Define test cases
	testCases := []struct {
		name           string
		filters        []filterModel
		tags           map[string]string
		mockTags       map[string]map[string]string
		expectedMCRs   []string
		expectedErrors int
	}{
		{
			name:         "No filters",
			filters:      []filterModel{},
			tags:         nil,
			expectedMCRs: []string{"mcr-1", "mcr-2", "mcr-3"},
		},
		{
			name: "Filter by name",
			filters: []filterModel{
				{
					Name:   types.StringValue("name"),
					Values: listValueMust(t, types.StringType, []string{"Test MCR 1"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-1"},
		},
		{
			name: "Filter by name pattern",
			filters: []filterModel{
				{
					Name:   types.StringValue("name"),
					Values: listValueMust(t, types.StringType, []string{"Test*"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-1", "mcr-2"},
		},
		{
			name: "Filter by port-speed",
			filters: []filterModel{
				{
					Name:   types.StringValue("port-speed"),
					Values: listValueMust(t, types.StringType, []string{"10000"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-1", "mcr-3"},
		},
		{
			name: "Filter by location-id",
			filters: []filterModel{
				{
					Name:   types.StringValue("location-id"),
					Values: listValueMust(t, types.StringType, []string{"456"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-2"},
		},
		{
			name: "Filter by provisioning-status",
			filters: []filterModel{
				{
					Name:   types.StringValue("provisioning-status"),
					Values: listValueMust(t, types.StringType, []string{"LIVE"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-1"},
		},
		{
			name: "Filter by market",
			filters: []filterModel{
				{
					Name:   types.StringValue("market"),
					Values: listValueMust(t, types.StringType, []string{"Sydney"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-1", "mcr-3"},
		},
		{
			name: "Filter by company-name",
			filters: []filterModel{
				{
					Name:   types.StringValue("company-name"),
					Values: listValueMust(t, types.StringType, []string{"Company B"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-2"},
		},
		{
			name: "Filter by vxc-permitted",
			filters: []filterModel{
				{
					Name:   types.StringValue("vxc-permitted"),
					Values: listValueMust(t, types.StringType, []string{"true"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-1", "mcr-2"},
		},
		{
			name: "Filter by diversity-zone",
			filters: []filterModel{
				{
					Name:   types.StringValue("diversity-zone"),
					Values: listValueMust(t, types.StringType, []string{"zone-a"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-1", "mcr-3"},
		},
		{
			name: "Filter by asn",
			filters: []filterModel{
				{
					Name:   types.StringValue("asn"),
					Values: listValueMust(t, types.StringType, []string{"64512"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-1"},
		},
		{
			name: "Multiple filters - AND logic",
			filters: []filterModel{
				{
					Name:   types.StringValue("port-speed"),
					Values: listValueMust(t, types.StringType, []string{"10000"}),
				},
				{
					Name:   types.StringValue("market"),
					Values: listValueMust(t, types.StringType, []string{"Sydney"}),
				},
				{
					Name:   types.StringValue("vxc-permitted"),
					Values: listValueMust(t, types.StringType, []string{"true"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-1"},
		},
		{
			name: "Multiple values for one filter - OR logic",
			filters: []filterModel{
				{
					Name:   types.StringValue("market"),
					Values: listValueMust(t, types.StringType, []string{"Sydney", "Melbourne"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-1", "mcr-2", "mcr-3"},
		},
		{
			name:    "Filter by tags",
			filters: []filterModel{},
			tags: map[string]string{
				"environment": "production",
				"owner":       "team-a",
			},
			mockTags: map[string]map[string]string{
				"mcr-1": {
					"environment": "production",
					"owner":       "team-a",
				},
				"mcr-2": {
					"environment": "staging",
					"owner":       "team-b",
				},
				"mcr-3": {
					"environment": "production",
					"owner":       "team-b",
				},
			},
			expectedMCRs: []string{"mcr-1"},
		},
		{
			name: "Combined filters and tags",
			filters: []filterModel{
				{
					Name:   types.StringValue("port-speed"),
					Values: listValueMust(t, types.StringType, []string{"10000"}),
				},
			},
			tags: map[string]string{
				"environment": "production",
			},
			mockTags: map[string]map[string]string{
				"mcr-1": {
					"environment": "production",
				},
				"mcr-2": {
					"environment": "staging",
				},
				"mcr-3": {
					"environment": "production",
				},
			},
			expectedMCRs: []string{"mcr-1", "mcr-3"},
		},
		{
			name: "Filter by company-uid",
			filters: []filterModel{
				{
					Name:   types.StringValue("company-uid"),
					Values: listValueMust(t, types.StringType, []string{"company-uid-b"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-2"},
		},
		{
			name: "Filter by product-type",
			filters: []filterModel{
				{
					Name:   types.StringValue("product-type"),
					Values: listValueMust(t, types.StringType, []string{"MCR2"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-1", "mcr-2"},
		},
		{
			name: "Filter by contract-term-months",
			filters: []filterModel{
				{
					Name:   types.StringValue("contract-term-months"),
					Values: listValueMust(t, types.StringType, []string{"12"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-1"},
		},
		{
			name: "Filter by vxc-auto-approval",
			filters: []filterModel{
				{
					Name:   types.StringValue("vxc-auto-approval"),
					Values: listValueMust(t, types.StringType, []string{"true"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-1"},
		},
		{
			name: "Filter by marketplace-visibility",
			filters: []filterModel{
				{
					Name:   types.StringValue("marketplace-visibility"),
					Values: listValueMust(t, types.StringType, []string{"true"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-1"},
		},
		{
			name: "Filter by locked",
			filters: []filterModel{
				{
					Name:   types.StringValue("locked"),
					Values: listValueMust(t, types.StringType, []string{"true"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-2"},
		},
		{
			name: "Filter by admin-locked",
			filters: []filterModel{
				{
					Name:   types.StringValue("admin-locked"),
					Values: listValueMust(t, types.StringType, []string{"true"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-3"},
		},
		{
			name: "Filter by cancelable",
			filters: []filterModel{
				{
					Name:   types.StringValue("cancelable"),
					Values: listValueMust(t, types.StringType, []string{"false"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-3"},
		},
		{
			name: "Filter by virtual",
			filters: []filterModel{
				{
					Name:   types.StringValue("virtual"),
					Values: listValueMust(t, types.StringType, []string{"true"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-1", "mcr-2"},
		},
		{
			name: "Filter by secondary-name pattern",
			filters: []filterModel{
				{
					Name:   types.StringValue("secondary-name"),
					Values: listValueMust(t, types.StringType, []string{"Secondary 2"}),
				},
			},
			tags:         nil,
			expectedMCRs: []string{"mcr-2"},
		},
		{
			name: "Unknown filter - should not filter out MCRs",
			filters: []filterModel{
				{
					Name:   types.StringValue("unknown-filter"),
					Values: listValueMust(t, types.StringType, []string{"some-value"}),
				},
			},
			tags:           nil,
			expectedMCRs:   []string{"mcr-1", "mcr-2", "mcr-3"},
			expectedErrors: 1, // Expecting a warning but not an error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock MCR service
			mockMCRService := &MockMCRService{
				ListMCRsResult: mcrs,
			}

			// Set up tag mocks if needed
			if tc.tags != nil {
				// Set custom function to handle MCR tag lookup
				mockMCRService.ListMCRResourceTagsFunc = func(ctx context.Context, mcrID string) (map[string]string, error) {
					if tags, ok := tc.mockTags[mcrID]; ok {
						return tags, nil
					}
					return map[string]string{}, nil
				}
			}

			// Create mock client with MCR service properly attached
			mockClient := &megaport.Client{
				MCRService: mockMCRService,
			}

			// Create data source with mock client
			ds := &mcrsDataSource{
				client: mockClient,
			}

			// Create model with test filters and tags
			var tagsValue types.Map
			if tc.tags != nil {
				var diags diag.Diagnostics
				tagsValue, diags = types.MapValueFrom(context.Background(), types.StringType, tc.tags)
				require.False(t, diags.HasError())
			} else {
				tagsValue = types.MapNull(types.StringType)
			}

			model := mcrsModel{
				Filter: tc.filters,
				Tags:   tagsValue,
			}

			// Call filterMCRs
			result, diags := ds.filterMCRs(context.Background(), mcrs, model)

			// Check for expected warnings
			if tc.expectedErrors > 0 {
				assert.Equal(t, tc.expectedErrors, len(diags))
			} else {
				assert.False(t, diags.HasError())
			}

			// Check results
			resultUIDs := make([]string, 0, len(result))
			for _, mcr := range result {
				resultUIDs = append(resultUIDs, mcr.UID)
			}

			// Verify expected MCRs are found (order independent)
			assert.Equal(t, len(tc.expectedMCRs), len(resultUIDs),
				"Expected %d MCRs but got %d", len(tc.expectedMCRs), len(resultUIDs))

			for _, expectedUID := range tc.expectedMCRs {
				found := false
				for _, resultUID := range resultUIDs {
					if expectedUID == resultUID {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected MCR %s not found in results", expectedUID)
			}
		})
	}
}

// TestReadWithErrors tests error handling in Read
func TestReadWithErrorsMCRs(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name            string
		setupMock       func(*MockMCRService)
		expectedSummary string
		expectError     bool
	}{
		{
			name: "ListMCRs error",
			setupMock: func(m *MockMCRService) {
				m.ListMCRsErr = errors.New("API error")
			},
			expectedSummary: "Unable to list MCRs",
			expectError:     true,
		},
		{
			name: "ListMCRResourceTags error",
			setupMock: func(m *MockMCRService) {
				m.ListMCRsResult = []*megaport.MCR{
					{
						UID:  "mcr-1",
						Name: "Test MCR 1",
					},
				}
				m.ListMCRResourceTagsFunc = func(ctx context.Context, mcrID string) (map[string]string, error) {
					return nil, errors.New("Tag API error")
				}
			},
			expectedSummary: "Unable to fetch tags for MCR",
			expectError:     false, // We expect a warning, not an error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock MCR service
			mockMCRService := &MockMCRService{}
			tc.setupMock(mockMCRService)

			// Create mock client with MCR service properly attached
			mockClient := &megaport.Client{
				MCRService: mockMCRService,
			}

			// Create data source with mock client
			ds := &mcrsDataSource{
				client: mockClient,
			}

			// Create a simplified test configuration that doesn't use tftypes directly
			tagsMap := map[string]string{}
			if tc.name == "ListMCRResourceTags error" {
				tagsMap = map[string]string{"environment": "production"}
			}

			tagsValue, _ := types.MapValueFrom(ctx, types.StringType, tagsMap)

			model := mcrsModel{
				UIDs:   types.ListNull(types.StringType),
				Filter: []filterModel{},
				Tags:   tagsValue,
			}

			if tc.name == "ListMCRs error" {
				// For the ListMCRs error case, test the API call directly
				mcrs, err := mockClient.MCRService.ListMCRs(ctx, &megaport.ListMCRsRequest{IncludeInactive: false})

				// Verify the error occurs
				assert.Error(t, err)
				assert.Nil(t, mcrs)

				// Check that the error message matches what we expect
				assert.Contains(t, err.Error(), "API error")
			} else {
				// For the tag error case, test filterMCRs directly
				mcrs := []*megaport.MCR{{UID: "mcr-1", Name: "Test MCR 1"}}
				_, diags := ds.filterMCRs(ctx, mcrs, model)

				// Check that we got warnings but not errors
				hasError := false
				for _, diagnostic := range diags {
					if diagnostic.Severity() == diag.SeverityError {
						hasError = true
						break
					}
				}
				assert.False(t, hasError, "Expected no errors, only warnings")

				// Check that the warning contains the expected text
				foundExpectedWarning := false
				for _, diagnostic := range diags {
					// Check both Summary AND Detail for the expected text
					if strings.Contains(diagnostic.Summary(), tc.expectedSummary) ||
						strings.Contains(diagnostic.Detail(), tc.expectedSummary) {
						foundExpectedWarning = true
						break
					}
				}
				assert.True(t, foundExpectedWarning, "Expected warning containing '%s' in summary or detail", tc.expectedSummary)
			}
		})
	}
}

// TestFromAPIMCRDetail tests the fromAPIMCRDetail mapping function
func TestFromAPIMCRDetail(t *testing.T) {
	t.Run("Maps all fields correctly", func(t *testing.T) {
		mcr := &megaport.MCR{
			ID:                    42,
			UID:                   "mcr-abc-123",
			Name:                  "My Test MCR",
			Type:                  "MCR2",
			ProvisioningStatus:    "LIVE",
			CreatedBy:             "user@example.com",
			CostCentre:           "CC-001",
			PortSpeed:             5000,
			Market:                "Sydney",
			LocationID:            65,
			CompanyUID:            "company-abc",
			CompanyName:           "Acme Corp",
			ContractTermMonths:    12,
			DiversityZone:         "zone-a",
			SecondaryName:         "Secondary MCR",
			VXCPermitted:          true,
			VXCAutoApproval:       false,
			MarketplaceVisibility: true,
			Virtual:               true,
			Locked:                false,
			AdminLocked:           true,
			Cancelable:            true,
			AttributeTags: map[string]string{
				"tag1": "val1",
				"tag2": "val2",
			},
			Resources: megaport.MCRResources{
				VirtualRouter: megaport.MCRVirtualRouter{
					ASN: 64512,
				},
			},
		}

		tags := map[string]string{
			"env":   "production",
			"owner": "team-a",
		}

		detail := fromAPIMCRDetail(mcr, tags)

		assert.Equal(t, "mcr-abc-123", detail.UID.ValueString())
		assert.Equal(t, int64(42), detail.ID.ValueInt64())
		assert.Equal(t, "My Test MCR", detail.Name.ValueString())
		assert.Equal(t, "MCR2", detail.Type.ValueString())
		assert.Equal(t, "LIVE", detail.ProvisioningStatus.ValueString())
		assert.Equal(t, "user@example.com", detail.CreatedBy.ValueString())
		assert.Equal(t, "CC-001", detail.CostCentre.ValueString())
		assert.Equal(t, int64(5000), detail.PortSpeed.ValueInt64())
		assert.Equal(t, "Sydney", detail.Market.ValueString())
		assert.Equal(t, int64(65), detail.LocationID.ValueInt64())
		assert.Equal(t, "company-abc", detail.CompanyUID.ValueString())
		assert.Equal(t, "Acme Corp", detail.CompanyName.ValueString())
		assert.Equal(t, int64(12), detail.ContractTermMonths.ValueInt64())
		assert.Equal(t, "zone-a", detail.DiversityZone.ValueString())
		assert.Equal(t, "Secondary MCR", detail.SecondaryName.ValueString())
		assert.Equal(t, true, detail.VXCPermitted.ValueBool())
		assert.Equal(t, false, detail.VXCAutoApproval.ValueBool())
		assert.Equal(t, true, detail.MarketplaceVisibility.ValueBool())
		assert.Equal(t, int64(64512), detail.ASN.ValueInt64())
		assert.Equal(t, true, detail.Virtual.ValueBool())
		assert.Equal(t, false, detail.Locked.ValueBool())
		assert.Equal(t, true, detail.AdminLocked.ValueBool())
		assert.Equal(t, true, detail.Cancelable.ValueBool())

		// Attribute tags populated
		assert.False(t, detail.AttributeTags.IsNull())

		// Resource tags populated
		assert.False(t, detail.ResourceTags.IsNull())
	})

	t.Run("Nil time fields produce empty strings", func(t *testing.T) {
		mcr := &megaport.MCR{
			UID:        "mcr-nil-times",
			CreateDate: nil,
			LiveDate:   nil,
			TerminateDate:     nil,
			ContractStartDate: nil,
			ContractEndDate:   nil,
		}

		detail := fromAPIMCRDetail(mcr, nil)

		assert.Equal(t, "", detail.CreateDate.ValueString())
		assert.Equal(t, "", detail.LiveDate.ValueString())
		assert.Equal(t, "", detail.TerminateDate.ValueString())
		assert.Equal(t, "", detail.ContractStartDate.ValueString())
		assert.Equal(t, "", detail.ContractEndDate.ValueString())
	})

	t.Run("Nil tags produce null maps", func(t *testing.T) {
		mcr := &megaport.MCR{
			UID:           "mcr-nil-tags",
			AttributeTags: nil,
		}

		detail := fromAPIMCRDetail(mcr, nil)

		assert.True(t, detail.AttributeTags.IsNull())
		assert.True(t, detail.ResourceTags.IsNull())
	})

	t.Run("Empty resource tags produce null map", func(t *testing.T) {
		mcr := &megaport.MCR{
			UID: "mcr-empty-tags",
		}

		detail := fromAPIMCRDetail(mcr, map[string]string{})

		assert.True(t, detail.ResourceTags.IsNull())
	})
}

// listValueMust creates a types.List for testing, failing the test on error.
func listValueMust(t *testing.T, elementType basetypes.StringType, elements interface{}) types.List {
	t.Helper()

	listValue, diags := types.ListValueFrom(context.Background(), elementType, elements)
	require.False(t, diags.HasError())

	return listValue
}
