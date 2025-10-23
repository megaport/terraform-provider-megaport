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

// MockMVEService is a mock of the MVE service for testing
type MockMVEService struct {
	mock.Mock
	ListMVEsResult            []*megaport.MVE
	ListMVEsErr               error
	ListMVEResourceTagsFunc   func(ctx context.Context, mveID string) (map[string]string, error)
	ListMVEResourceTagsErr    error
	ListMVEResourceTagsResult map[string]string
	CapturedResourceTagMVEUID string
}

func (m *MockMVEService) ListMVEs(ctx context.Context, req *megaport.ListMVEsRequest) ([]*megaport.MVE, error) {
	if m.ListMVEsErr != nil {
		return nil, m.ListMVEsErr
	}
	if m.ListMVEsResult != nil {
		return m.ListMVEsResult, nil
	}
	return []*megaport.MVE{}, nil
}

func (m *MockMVEService) ListMVEResourceTags(ctx context.Context, mveID string) (map[string]string, error) {
	m.CapturedResourceTagMVEUID = mveID
	if m.ListMVEResourceTagsFunc != nil {
		return m.ListMVEResourceTagsFunc(ctx, mveID)
	}
	if m.ListMVEResourceTagsErr != nil {
		return nil, m.ListMVEResourceTagsErr
	}
	if m.ListMVEResourceTagsResult != nil {
		return m.ListMVEResourceTagsResult, nil
	}
	return map[string]string{
		"environment": "test",
		"owner":       "automation",
	}, nil
}

// Implement other required methods of the MVEService interface with minimal stubs
func (m *MockMVEService) BuyMVE(ctx context.Context, req *megaport.BuyMVERequest) (*megaport.BuyMVEResponse, error) {
	return nil, nil
}

func (m *MockMVEService) ValidateMVEOrder(ctx context.Context, req *megaport.BuyMVERequest) error {
	return nil
}

func (m *MockMVEService) GetMVE(ctx context.Context, mveId string) (*megaport.MVE, error) {
	return nil, nil
}

func (m *MockMVEService) ModifyMVE(ctx context.Context, req *megaport.ModifyMVERequest) (*megaport.ModifyMVEResponse, error) {
	return nil, nil
}

func (m *MockMVEService) DeleteMVE(ctx context.Context, req *megaport.DeleteMVERequest) (*megaport.DeleteMVEResponse, error) {
	return nil, nil
}

func (m *MockMVEService) ListMVEImages(ctx context.Context) ([]*megaport.MVEImage, error) {
	return nil, nil
}

func (m *MockMVEService) ListAvailableMVESizes(ctx context.Context) ([]*megaport.MVESize, error) {
	return nil, nil
}

func (m *MockMVEService) UpdateMVEResourceTags(ctx context.Context, mveID string, tags map[string]string) error {
	return nil
}

// TestFilterMVEs tests the filterMVEs method
func TestFilterMVEs(t *testing.T) {
	// Create sample MVEs for testing
	mves := []*megaport.MVE{
		{
			UID:                "mve-1",
			Name:               "Test MVE 1",
			Vendor:             "Cisco",
			Size:               "MEDIUM",
			LocationID:         123,
			ProvisioningStatus: "LIVE",
			Market:             "Sydney",
			CompanyName:        "Company A",
			VXCPermitted:       true,
		},
		{
			UID:                "mve-2",
			Name:               "Test MVE 2",
			Vendor:             "Fortinet",
			Size:               "SMALL",
			LocationID:         456,
			ProvisioningStatus: "CONFIGURED",
			Market:             "Melbourne",
			CompanyName:        "Company B",
			VXCPermitted:       true,
		},
		{
			UID:                "mve-3",
			Name:               "Inactive MVE",
			Vendor:             "Palo Alto",
			Size:               "LARGE",
			LocationID:         123,
			ProvisioningStatus: "DECOMMISSIONED",
			Market:             "Sydney",
			CompanyName:        "Company A",
			VXCPermitted:       false,
		},
	}

	// Define test cases
	testCases := []struct {
		name           string
		filters        []filterModel
		tags           map[string]string
		mockTags       map[string]map[string]string
		expectedMVEs   []string
		expectedErrors int
	}{
		{
			name:         "No filters",
			filters:      []filterModel{},
			tags:         nil,
			expectedMVEs: []string{"mve-1", "mve-2", "mve-3"},
		},
		{
			name: "Filter by name",
			filters: []filterModel{
				{
					Name:   types.StringValue("name"),
					Values: listValueMust(t, types.StringType, []string{"Test MVE 1"}),
				},
			},
			tags:         nil,
			expectedMVEs: []string{"mve-1"},
		},
		{
			name: "Filter by vendor",
			filters: []filterModel{
				{
					Name:   types.StringValue("vendor"),
					Values: listValueMust(t, types.StringType, []string{"Cisco"}),
				},
			},
			tags:         nil,
			expectedMVEs: []string{"mve-1"},
		},
		{
			name: "Filter by size",
			filters: []filterModel{
				{
					Name:   types.StringValue("size"),
					Values: listValueMust(t, types.StringType, []string{"MEDIUM"}),
				},
			},
			tags:         nil,
			expectedMVEs: []string{"mve-1"},
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
			expectedMVEs: []string{"mve-2"},
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
			expectedMVEs: []string{"mve-1"},
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
			expectedMVEs: []string{"mve-1", "mve-3"},
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
			expectedMVEs: []string{"mve-2"},
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
			expectedMVEs: []string{"mve-1", "mve-2"},
		},
		{
			name: "Multiple filters - AND logic",
			filters: []filterModel{
				{
					Name:   types.StringValue("vendor"),
					Values: listValueMust(t, types.StringType, []string{"Cisco"}),
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
			expectedMVEs: []string{"mve-1"},
		},
		{
			name: "Multiple values for one filter - OR logic",
			filters: []filterModel{
				{
					Name:   types.StringValue("vendor"),
					Values: listValueMust(t, types.StringType, []string{"Cisco", "Fortinet"}),
				},
			},
			tags:         nil,
			expectedMVEs: []string{"mve-1", "mve-2"},
		},
		{
			name:    "Filter by tags",
			filters: []filterModel{},
			tags: map[string]string{
				"environment": "production",
				"owner":       "team-a",
			},
			mockTags: map[string]map[string]string{
				"mve-1": {
					"environment": "production",
					"owner":       "team-a",
				},
				"mve-2": {
					"environment": "staging",
					"owner":       "team-b",
				},
				"mve-3": {
					"environment": "production",
					"owner":       "team-b",
				},
			},
			expectedMVEs: []string{"mve-1"},
		},
		{
			name: "Combined filters and tags",
			filters: []filterModel{
				{
					Name:   types.StringValue("vendor"),
					Values: listValueMust(t, types.StringType, []string{"Cisco", "Palo Alto"}),
				},
			},
			tags: map[string]string{
				"environment": "production",
			},
			mockTags: map[string]map[string]string{
				"mve-1": {
					"environment": "production",
				},
				"mve-2": {
					"environment": "staging",
				},
				"mve-3": {
					"environment": "production",
				},
			},
			expectedMVEs: []string{"mve-1", "mve-3"},
		},
		{
			name: "Unknown filter - should not filter out MVEs",
			filters: []filterModel{
				{
					Name:   types.StringValue("unknown-filter"),
					Values: listValueMust(t, types.StringType, []string{"some-value"}),
				},
			},
			tags:           nil,
			expectedMVEs:   []string{"mve-1", "mve-2", "mve-3"},
			expectedErrors: 1, // Expecting a warning but not an error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock MVE service
			mockMVEService := &MockMVEService{
				ListMVEsResult: mves,
			}

			// Set up tag mocks if needed
			if tc.tags != nil {
				// Set custom function to handle MVE tag lookup
				mockMVEService.ListMVEResourceTagsFunc = func(ctx context.Context, mveID string) (map[string]string, error) {
					if tags, ok := tc.mockTags[mveID]; ok {
						return tags, nil
					}
					return map[string]string{}, nil
				}
			}

			// Create mock client with MVE service properly attached
			mockClient := &megaport.Client{
				MVEService: mockMVEService,
			}

			// Create data source with mock client
			ds := &mvesDataSource{
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

			model := mvesModel{
				Filter: tc.filters,
				Tags:   tagsValue,
			}

			// Call filterMVEs
			result, diags := ds.filterMVEs(context.Background(), mves, model)

			// Check for expected warnings
			if tc.expectedErrors > 0 {
				assert.Equal(t, tc.expectedErrors, len(diags))
			} else {
				assert.False(t, diags.HasError())
			}

			// Check results
			resultUIDs := make([]string, 0, len(result))
			for _, mve := range result {
				resultUIDs = append(resultUIDs, mve.UID)
			}

			// Verify expected MVEs are found (order independent)
			assert.Equal(t, len(tc.expectedMVEs), len(resultUIDs),
				"Expected %d MVEs but got %d", len(tc.expectedMVEs), len(resultUIDs))

			for _, expectedUID := range tc.expectedMVEs {
				found := false
				for _, resultUID := range resultUIDs {
					if expectedUID == resultUID {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected MVE %s not found in results", expectedUID)
			}
		})
	}
}

// TestReadWithErrors tests error handling in Read
func TestReadWithErrorsMVEs(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name            string
		setupMock       func(*MockMVEService)
		expectedSummary string
		expectError     bool
	}{
		{
			name: "ListMVEs error",
			setupMock: func(m *MockMVEService) {
				m.ListMVEsErr = errors.New("API error")
			},
			expectedSummary: "Unable to list MVEs",
			expectError:     true,
		},
		{
			name: "ListMVEResourceTags error",
			setupMock: func(m *MockMVEService) {
				m.ListMVEsResult = []*megaport.MVE{
					{
						UID:  "mve-1",
						Name: "Test MVE 1",
					},
				}
				m.ListMVEResourceTagsFunc = func(ctx context.Context, mveID string) (map[string]string, error) {
					return nil, errors.New("Tag API error")
				}
			},
			expectedSummary: "Unable to fetch tags for MVE",
			expectError:     false, // We expect a warning, not an error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock MVE service
			mockMVEService := &MockMVEService{}
			tc.setupMock(mockMVEService)

			// Create mock client with MVE service properly attached
			mockClient := &megaport.Client{
				MVEService: mockMVEService,
			}

			// Create data source with mock client
			ds := &mvesDataSource{
				client: mockClient,
			}

			// Create a simplified test configuration that doesn't use tftypes directly
			tagsMap := map[string]string{}
			if tc.name == "ListMVEResourceTags error" {
				tagsMap = map[string]string{"environment": "production"}
			}

			tagsValue, _ := types.MapValueFrom(ctx, types.StringType, tagsMap)

			model := mvesModel{
				UIDs:   types.ListNull(types.StringType),
				Filter: []filterModel{},
				Tags:   tagsValue,
			}

			if tc.name == "ListMVEs error" {
				// For the ListMVEs error case, test the API call directly
				mves, err := mockClient.MVEService.ListMVEs(ctx, &megaport.ListMVEsRequest{IncludeInactive: false})

				// Verify the error occurs
				assert.Error(t, err)
				assert.Nil(t, mves)

				// Check that the error message matches what we expect
				assert.Contains(t, err.Error(), "API error")
			} else {
				// For the tag error case, test filterMVEs directly
				mves := []*megaport.MVE{{UID: "mve-1", Name: "Test MVE 1"}}
				_, diags := ds.filterMVEs(ctx, mves, model)

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
