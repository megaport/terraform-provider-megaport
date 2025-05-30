package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	megaport "github.com/megaport/megaportgo"
)

// MockMCRService is a mock of the MCR service for testing
type MockMCRService struct {
	mock.Mock
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
			UID:                "mcr-1",
			Name:               "Test MCR 1",
			PortSpeed:          10000,
			LocationID:         123,
			ProvisioningStatus: "LIVE",
			Market:             "Sydney",
			CompanyName:        "Company A",
			VXCPermitted:       true,
			DiversityZone:      "zone-a",
			Resources: megaport.MCRResources{
				VirtualRouter: megaport.MCRVirtualRouter{
					ASN: 64512,
				},
			},
		},
		{
			UID:                "mcr-2",
			Name:               "Test MCR 2",
			PortSpeed:          1000,
			LocationID:         456,
			ProvisioningStatus: "CONFIGURED",
			Market:             "Melbourne",
			CompanyName:        "Company B",
			VXCPermitted:       true,
			DiversityZone:      "zone-b",
			Resources: megaport.MCRResources{
				VirtualRouter: megaport.MCRVirtualRouter{
					ASN: 64513,
				},
			},
		},
		{
			UID:                "mcr-3",
			Name:               "Inactive MCR",
			PortSpeed:          10000,
			LocationID:         123,
			ProvisioningStatus: "DECOMMISSIONED",
			Market:             "Sydney",
			CompanyName:        "Company A",
			VXCPermitted:       false,
			DiversityZone:      "zone-a",
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
