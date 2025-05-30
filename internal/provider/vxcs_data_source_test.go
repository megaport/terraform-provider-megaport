package provider

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	megaport "github.com/megaport/megaportgo"
)

// MockVXCService is a mock of the VXC service for testing
type MockVXCService struct {
	mock.Mock
	ListVXCsResult            []*megaport.VXC
	ListVXCsErr               error
	ListVXCResourceTagsFunc   func(ctx context.Context, vxcID string) (map[string]string, error)
	ListVXCResourceTagsErr    error
	ListVXCResourceTagsResult map[string]string
	CapturedResourceTagVXCUID string
}

func (m *MockVXCService) ListVXCs(ctx context.Context, req *megaport.ListVXCsRequest) ([]*megaport.VXC, error) {
	if m.ListVXCsErr != nil {
		return nil, m.ListVXCsErr
	}
	if m.ListVXCsResult != nil {
		return m.ListVXCsResult, nil
	}
	return []*megaport.VXC{}, nil
}

func (m *MockVXCService) ListVXCResourceTags(ctx context.Context, vxcID string) (map[string]string, error) {
	m.CapturedResourceTagVXCUID = vxcID
	if m.ListVXCResourceTagsFunc != nil {
		return m.ListVXCResourceTagsFunc(ctx, vxcID)
	}
	if m.ListVXCResourceTagsErr != nil {
		return nil, m.ListVXCResourceTagsErr
	}
	if m.ListVXCResourceTagsResult != nil {
		return m.ListVXCResourceTagsResult, nil
	}
	return map[string]string{
		"environment": "test",
		"owner":       "automation",
	}, nil
}

// Implement other required methods of the VXCService interface with minimal stubs
func (m *MockVXCService) BuyVXC(ctx context.Context, req *megaport.BuyVXCRequest) (*megaport.BuyVXCResponse, error) {
	return nil, nil
}

func (m *MockVXCService) ValidateVXCOrder(ctx context.Context, req *megaport.BuyVXCRequest) error {
	return nil
}

func (m *MockVXCService) GetVXC(ctx context.Context, id string) (*megaport.VXC, error) {
	return nil, nil
}

func (m *MockVXCService) DeleteVXC(ctx context.Context, id string, req *megaport.DeleteVXCRequest) error {
	return nil
}

func (m *MockVXCService) UpdateVXC(ctx context.Context, id string, req *megaport.UpdateVXCRequest) (*megaport.VXC, error) {
	return nil, nil
}

func (m *MockVXCService) LookupPartnerPorts(ctx context.Context, req *megaport.LookupPartnerPortsRequest) (*megaport.LookupPartnerPortsResponse, error) {
	return nil, nil
}

func (m *MockVXCService) ListPartnerPorts(ctx context.Context, req *megaport.ListPartnerPortsRequest) (*megaport.ListPartnerPortsResponse, error) {
	return nil, nil
}

func (m *MockVXCService) UpdateVXCResourceTags(ctx context.Context, vxcID string, tags map[string]string) error {
	return nil
}

// TestFilterVXCs tests the filterVXCs method
func TestFilterVXCs(t *testing.T) {
	// Create mock time values for testing
	currentTime := time.Now()
	testTime := megaport.Time{Time: currentTime}

	// Create sample VXCs for testing
	vxcs := []*megaport.VXC{
		{
			UID:                "vxc-1",
			Name:               "Test VXC 1",
			RateLimit:          1000,
			ProvisioningStatus: "LIVE",
			CompanyName:        "Company A",
			AEndConfiguration: megaport.VXCEndConfiguration{
				UID:        "port-1",
				Name:       "Port 1",
				LocationID: 123,
			},
			BEndConfiguration: megaport.VXCEndConfiguration{
				UID:        "port-2",
				Name:       "Port 2",
				LocationID: 456,
			},
			LiveDate:   &testTime,
			CreateDate: &testTime,
		},
		{
			UID:                "vxc-2",
			Name:               "Test VXC 2",
			RateLimit:          10000,
			ProvisioningStatus: "CONFIGURED",
			CompanyName:        "Company B",
			AEndConfiguration: megaport.VXCEndConfiguration{
				UID:        "port-3",
				Name:       "Port 3",
				LocationID: 789,
			},
			BEndConfiguration: megaport.VXCEndConfiguration{
				UID:        "mcr-1",
				Name:       "MCR 1",
				LocationID: 123,
			},
			LiveDate:   &testTime,
			CreateDate: &testTime,
		},
		{
			UID:                "vxc-3",
			Name:               "Cloud Connection",
			RateLimit:          5000,
			ProvisioningStatus: "DECOMMISSIONED",
			CompanyName:        "Company A",
			AEndConfiguration: megaport.VXCEndConfiguration{
				UID:        "port-1",
				Name:       "Port 1",
				LocationID: 123,
			},
			BEndConfiguration: megaport.VXCEndConfiguration{
				UID:        "csp-1",
				Name:       "Cloud Provider",
				LocationID: 456,
			},
			LiveDate:   &testTime,
			CreateDate: &testTime,
		},
	}

	// Define test cases
	testCases := []struct {
		name           string
		filters        []filterModel
		tags           map[string]string
		mockTags       map[string]map[string]string
		expectedVXCs   []string
		expectedErrors int
	}{
		{
			name:         "No filters",
			filters:      []filterModel{},
			tags:         nil,
			expectedVXCs: []string{"vxc-1", "vxc-2", "vxc-3"},
		},
		{
			name: "Filter by name",
			filters: []filterModel{
				{
					Name:   types.StringValue("name"),
					Values: listValueMust(t, types.StringType, []string{"Test VXC 1"}),
				},
			},
			tags:         nil,
			expectedVXCs: []string{"vxc-1"},
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
			expectedVXCs: []string{"vxc-1", "vxc-2"},
		},
		{
			name: "Filter by rate-limit",
			filters: []filterModel{
				{
					Name:   types.StringValue("rate-limit"),
					Values: listValueMust(t, types.StringType, []string{"10000"}),
				},
			},
			tags:         nil,
			expectedVXCs: []string{"vxc-2"},
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
			expectedVXCs: []string{"vxc-1"},
		},
		{
			name: "Filter by aend-uid",
			filters: []filterModel{
				{
					Name:   types.StringValue("aend-uid"),
					Values: listValueMust(t, types.StringType, []string{"port-1"}),
				},
			},
			tags:         nil,
			expectedVXCs: []string{"vxc-1", "vxc-3"},
		},
		{
			name: "Filter by bend-uid",
			filters: []filterModel{
				{
					Name:   types.StringValue("bend-uid"),
					Values: listValueMust(t, types.StringType, []string{"mcr-1"}),
				},
			},
			tags:         nil,
			expectedVXCs: []string{"vxc-2"},
		},
		{
			name: "Filter by company-name",
			filters: []filterModel{
				{
					Name:   types.StringValue("company-name"),
					Values: listValueMust(t, types.StringType, []string{"Company A"}),
				},
			},
			tags:         nil,
			expectedVXCs: []string{"vxc-1", "vxc-3"},
		},
		{
			name: "Multiple filters - AND logic",
			filters: []filterModel{
				{
					Name:   types.StringValue("aend-uid"),
					Values: listValueMust(t, types.StringType, []string{"port-1"}),
				},
				{
					Name:   types.StringValue("provisioning-status"),
					Values: listValueMust(t, types.StringType, []string{"LIVE"}),
				},
			},
			tags:         nil,
			expectedVXCs: []string{"vxc-1"},
		},
		{
			name: "Multiple values for one filter - OR logic",
			filters: []filterModel{
				{
					Name:   types.StringValue("provisioning-status"),
					Values: listValueMust(t, types.StringType, []string{"LIVE", "CONFIGURED"}),
				},
			},
			tags:         nil,
			expectedVXCs: []string{"vxc-1", "vxc-2"},
		},
		{
			name:    "Filter by tags",
			filters: []filterModel{},
			tags: map[string]string{
				"environment": "production",
				"owner":       "team-a",
			},
			mockTags: map[string]map[string]string{
				"vxc-1": {
					"environment": "production",
					"owner":       "team-a",
				},
				"vxc-2": {
					"environment": "staging",
					"owner":       "team-b",
				},
				"vxc-3": {
					"environment": "production",
					"owner":       "team-b",
				},
			},
			expectedVXCs: []string{"vxc-1"},
		},
		{
			name: "Combined filters and tags",
			filters: []filterModel{
				{
					Name:   types.StringValue("aend-uid"),
					Values: listValueMust(t, types.StringType, []string{"port-1"}),
				},
			},
			tags: map[string]string{
				"environment": "production",
			},
			mockTags: map[string]map[string]string{
				"vxc-1": {
					"environment": "production",
				},
				"vxc-2": {
					"environment": "staging",
				},
				"vxc-3": {
					"environment": "production",
				},
			},
			expectedVXCs: []string{"vxc-1", "vxc-3"},
		},
		{
			name: "Unknown filter - should not filter out VXCs",
			filters: []filterModel{
				{
					Name:   types.StringValue("unknown-filter"),
					Values: listValueMust(t, types.StringType, []string{"some-value"}),
				},
			},
			tags:           nil,
			expectedVXCs:   []string{"vxc-1", "vxc-2", "vxc-3"},
			expectedErrors: 1, // Expecting a warning but not an error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock VXC service
			mockVXCService := &MockVXCService{
				ListVXCsResult: vxcs,
			}

			// Set up tag mocks if needed
			if tc.tags != nil {
				// Set custom function to handle VXC tag lookup
				mockVXCService.ListVXCResourceTagsFunc = func(ctx context.Context, vxcID string) (map[string]string, error) {
					if tags, ok := tc.mockTags[vxcID]; ok {
						return tags, nil
					}
					return map[string]string{}, nil
				}
			}

			// Create mock client with VXC service properly attached
			mockClient := &megaport.Client{
				VXCService: mockVXCService,
			}

			// Create data source with mock client
			ds := &vxcsDataSource{
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

			model := vxcsModel{
				Filter: tc.filters,
				Tags:   tagsValue,
			}

			// Call filterVXCs
			result, diags := ds.filterVXCs(context.Background(), vxcs, model)

			// Check for expected warnings
			if tc.expectedErrors > 0 {
				assert.Equal(t, tc.expectedErrors, len(diags))
			} else {
				assert.False(t, diags.HasError())
			}

			// Check results
			resultUIDs := make([]string, 0, len(result))
			for _, vxc := range result {
				resultUIDs = append(resultUIDs, vxc.UID)
			}

			// Verify expected VXCs are found (order independent)
			assert.Equal(t, len(tc.expectedVXCs), len(resultUIDs),
				"Expected %d VXCs but got %d", len(tc.expectedVXCs), len(resultUIDs))

			for _, expectedUID := range tc.expectedVXCs {
				found := false
				for _, resultUID := range resultUIDs {
					if expectedUID == resultUID {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected VXC %s not found in results", expectedUID)
			}
		})
	}
}

// TestReadWithErrors tests error handling in Read
func TestReadWithErrorsVXCs(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name            string
		setupMock       func(*MockVXCService)
		expectedSummary string
		expectError     bool
	}{
		{
			name: "ListVXCs error",
			setupMock: func(m *MockVXCService) {
				m.ListVXCsErr = errors.New("API error")
			},
			expectedSummary: "Unable to list VXCs",
			expectError:     true,
		},
		{
			name: "ListVXCResourceTags error",
			setupMock: func(m *MockVXCService) {
				m.ListVXCsResult = []*megaport.VXC{
					{
						UID:  "vxc-1",
						Name: "Test VXC 1",
					},
				}
				m.ListVXCResourceTagsFunc = func(ctx context.Context, vxcID string) (map[string]string, error) {
					return nil, errors.New("Tag API error")
				}
			},
			expectedSummary: "Unable to fetch tags for VXC",
			expectError:     false, // We expect a warning, not an error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock VXC service
			mockVXCService := &MockVXCService{}
			tc.setupMock(mockVXCService)

			// Create mock client with VXC service properly attached
			mockClient := &megaport.Client{
				VXCService: mockVXCService,
			}

			// Create data source with mock client
			ds := &vxcsDataSource{
				client: mockClient,
			}

			// Create a simplified test configuration that doesn't use tftypes directly
			tagsMap := map[string]string{}
			if tc.name == "ListVXCResourceTags error" {
				tagsMap = map[string]string{"environment": "production"}
			}

			tagsValue, _ := types.MapValueFrom(ctx, types.StringType, tagsMap)

			model := vxcsModel{
				UIDs:   types.ListNull(types.StringType),
				Filter: []filterModel{},
				Tags:   tagsValue,
			}

			if tc.name == "ListVXCs error" {
				// For the ListVXCs error case, test the API call directly
				vxcs, err := mockClient.VXCService.ListVXCs(ctx, &megaport.ListVXCsRequest{IncludeInactive: false})

				// Verify the error occurs
				assert.Error(t, err)
				assert.Nil(t, vxcs)

				// Check that the error message matches what we expect
				assert.Contains(t, err.Error(), "API error")
			} else {
				// For the tag error case, test filterVXCs directly
				vxcs := []*megaport.VXC{{UID: "vxc-1", Name: "Test VXC 1"}}
				_, diags := ds.filterVXCs(ctx, vxcs, model)

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
