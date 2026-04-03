package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	megaport "github.com/megaport/megaportgo"
)

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
			mockMCRService := &MockMCRService{
				ListMCRsResult: mcrs,
			}

			if tc.tags != nil {
				mockMCRService.ListMCRResourceTagsFunc = func(ctx context.Context, mcrID string) (map[string]string, error) {
					if tags, ok := tc.mockTags[mcrID]; ok {
						return tags, nil
					}
					return map[string]string{}, nil
				}
			}

			mockClient := &megaport.Client{
				MCRService: mockMCRService,
			}

			ds := &mcrsDataSource{
				client: mockClient,
			}

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

			result, diags := ds.filterMCRs(context.Background(), mcrs, model)

			if tc.expectedErrors > 0 {
				assert.Equal(t, tc.expectedErrors, len(diags))
			} else {
				assert.False(t, diags.HasError())
			}

			resultUIDs := make([]string, 0, len(result))
			for _, mcr := range result {
				resultUIDs = append(resultUIDs, mcr.UID)
			}

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

// TestReadWithErrorsMCRs tests error handling in Read
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
			mockMCRService := &MockMCRService{}
			tc.setupMock(mockMCRService)

			mockClient := &megaport.Client{
				MCRService: mockMCRService,
			}

			ds := &mcrsDataSource{
				client: mockClient,
			}

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
				mcrs, err := mockClient.MCRService.ListMCRs(ctx, &megaport.ListMCRsRequest{IncludeInactive: false})

				assert.Error(t, err)
				assert.Nil(t, mcrs)
				assert.Contains(t, err.Error(), "API error")
			} else {
				mcrs := []*megaport.MCR{{UID: "mcr-1", Name: "Test MCR 1"}}
				_, diags := ds.filterMCRs(ctx, mcrs, model)

				hasError := false
				for _, diagnostic := range diags {
					if diagnostic.Severity() == diag.SeverityError {
						hasError = true
						break
					}
				}
				assert.False(t, hasError, "Expected no errors, only warnings")

				foundExpectedWarning := false
				for _, diagnostic := range diags {
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
			CostCentre:            "CC-001",
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
		assert.False(t, detail.AttributeTags.IsNull())
		assert.False(t, detail.ResourceTags.IsNull())
	})

	t.Run("Nil time fields produce empty strings", func(t *testing.T) {
		mcr := &megaport.MCR{
			UID:               "mcr-nil-times",
			CreateDate:        nil,
			LiveDate:          nil,
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
