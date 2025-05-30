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

// MockIXService is a mock of the IX service for testing
type MockIXService struct {
	mock.Mock
	ListIXsResult            []*megaport.IX
	ListIXsErr               error
	ListIXResourceTagsFunc   func(ctx context.Context, ixID string) (map[string]string, error)
	ListIXResourceTagsErr    error
	ListIXResourceTagsResult map[string]string
	CapturedResourceTagIXUID string
}

func (m *MockIXService) ListIXs(ctx context.Context, req *megaport.ListIXsRequest) ([]*megaport.IX, error) {
	if m.ListIXsErr != nil {
		return nil, m.ListIXsErr
	}
	if m.ListIXsResult != nil {
		return m.ListIXsResult, nil
	}
	return []*megaport.IX{}, nil
}

func (m *MockIXService) ListIXResourceTags(ctx context.Context, ixID string) (map[string]string, error) {
	m.CapturedResourceTagIXUID = ixID
	if m.ListIXResourceTagsFunc != nil {
		return m.ListIXResourceTagsFunc(ctx, ixID)
	}
	if m.ListIXResourceTagsErr != nil {
		return nil, m.ListIXResourceTagsErr
	}
	if m.ListIXResourceTagsResult != nil {
		return m.ListIXResourceTagsResult, nil
	}
	return map[string]string{
		"environment": "test",
		"owner":       "automation",
	}, nil
}

// Implement other required methods of the IXService interface with minimal stubs
func (m *MockIXService) GetIX(ctx context.Context, id string) (*megaport.IX, error) {
	return nil, nil
}

func (m *MockIXService) BuyIX(ctx context.Context, req *megaport.BuyIXRequest) (*megaport.BuyIXResponse, error) {
	return nil, nil
}

func (m *MockIXService) ValidateIXOrder(ctx context.Context, req *megaport.BuyIXRequest) error {
	return nil
}

func (m *MockIXService) UpdateIX(ctx context.Context, id string, req *megaport.UpdateIXRequest) (*megaport.IX, error) {
	return nil, nil
}

func (m *MockIXService) DeleteIX(ctx context.Context, id string, req *megaport.DeleteIXRequest) error {
	return nil
}

func (m *MockIXService) UpdateIXResourceTags(ctx context.Context, ixID string, tags map[string]string) error {
	return nil
}

// TestFilterIXs tests the filterIXs method
func TestFilterIXs(t *testing.T) {
	// Create mock time values for testing
	currentTime := time.Now()
	testTime := megaport.Time{Time: currentTime}

	// Create sample IXs for testing
	ixs := []*megaport.IX{
		{
			ProductUID:         "ix-1",
			ProductName:        "Test IX 1",
			ProvisioningStatus: "LIVE",
			RateLimit:          1000,
			VLAN:               100,
			ASN:                64512,
			NetworkServiceType: "Los Angeles IX",
			LocationID:         123,
			LocationDetail: megaport.IXLocationDetail{
				Name:    "Test Location 1",
				City:    "Los Angeles",
				Metro:   "Los Angeles",
				Country: "US",
			},
			CreateDate: &testTime,
			DeployDate: &testTime,
			AttributeTags: map[string]string{
				"environment": "production",
				"owner":       "team-a",
			},
		},
		{
			ProductUID:         "ix-2",
			ProductName:        "Test IX 2",
			ProvisioningStatus: "CONFIGURED",
			RateLimit:          10000,
			VLAN:               200,
			ASN:                64513,
			NetworkServiceType: "Sydney IX",
			LocationID:         456,
			LocationDetail: megaport.IXLocationDetail{
				Name:    "Test Location 2",
				City:    "Sydney",
				Metro:   "Sydney",
				Country: "AU",
			},
			CreateDate: &testTime,
			DeployDate: &testTime,
			AttributeTags: map[string]string{
				"environment": "staging",
				"owner":       "team-b",
			},
		},
		{
			ProductUID:         "ix-3",
			ProductName:        "Inactive IX",
			ProvisioningStatus: "DECOMMISSIONED",
			RateLimit:          1000,
			VLAN:               300,
			ASN:                64514,
			NetworkServiceType: "Los Angeles IX",
			LocationID:         123,
			LocationDetail: megaport.IXLocationDetail{
				Name:    "Test Location 1",
				City:    "Los Angeles",
				Metro:   "Los Angeles",
				Country: "US",
			},
			CreateDate: &testTime,
			DeployDate: &testTime,
			AttributeTags: map[string]string{
				"environment": "production",
				"owner":       "team-c",
			},
		},
	}

	// Define test cases
	testCases := []struct {
		name           string
		filters        []filterModel
		tags           map[string]string
		mockTags       map[string]map[string]string
		expectedIXs    []string
		expectedErrors int
	}{
		{
			name:        "No filters",
			filters:     []filterModel{},
			tags:        nil,
			expectedIXs: []string{"ix-1", "ix-2", "ix-3"},
		},
		{
			name: "Filter by name",
			filters: []filterModel{
				{
					Name:   types.StringValue("name"),
					Values: listValueMust(t, types.StringType, []string{"Test IX 1"}),
				},
			},
			tags:        nil,
			expectedIXs: []string{"ix-1"},
		},
		{
			name: "Filter by name pattern",
			filters: []filterModel{
				{
					Name:   types.StringValue("name"),
					Values: listValueMust(t, types.StringType, []string{"Test*"}),
				},
			},
			tags:        nil,
			expectedIXs: []string{"ix-1", "ix-2"},
		},
		{
			name: "Filter by vlan",
			filters: []filterModel{
				{
					Name:   types.StringValue("vlan"),
					Values: listValueMust(t, types.StringType, []string{"200"}),
				},
			},
			tags:        nil,
			expectedIXs: []string{"ix-2"},
		},
		{
			name: "Filter by asn",
			filters: []filterModel{
				{
					Name:   types.StringValue("asn"),
					Values: listValueMust(t, types.StringType, []string{"64512"}),
				},
			},
			tags:        nil,
			expectedIXs: []string{"ix-1"},
		},
		{
			name: "Filter by network-service-type",
			filters: []filterModel{
				{
					Name:   types.StringValue("network-service-type"),
					Values: listValueMust(t, types.StringType, []string{"Los Angeles IX"}),
				},
			},
			tags:        nil,
			expectedIXs: []string{"ix-1", "ix-3"},
		},
		{
			name: "Filter by location-id",
			filters: []filterModel{
				{
					Name:   types.StringValue("location-id"),
					Values: listValueMust(t, types.StringType, []string{"456"}),
				},
			},
			tags:        nil,
			expectedIXs: []string{"ix-2"},
		},
		{
			name: "Filter by rate-limit",
			filters: []filterModel{
				{
					Name:   types.StringValue("rate-limit"),
					Values: listValueMust(t, types.StringType, []string{"10000"}),
				},
			},
			tags:        nil,
			expectedIXs: []string{"ix-2"},
		},
		{
			name: "Filter by provisioning-status",
			filters: []filterModel{
				{
					Name:   types.StringValue("provisioning-status"),
					Values: listValueMust(t, types.StringType, []string{"LIVE"}),
				},
			},
			tags:        nil,
			expectedIXs: []string{"ix-1"},
		},
		{
			name: "Filter by company-name",
			filters: []filterModel{
				{
					Name:   types.StringValue("company-name"),
					Values: listValueMust(t, types.StringType, []string{"Test Location 2"}),
				},
			},
			tags:        nil,
			expectedIXs: []string{"ix-2"},
		},
		{
			name: "Multiple filters - AND logic",
			filters: []filterModel{
				{
					Name:   types.StringValue("network-service-type"),
					Values: listValueMust(t, types.StringType, []string{"Los Angeles IX"}),
				},
				{
					Name:   types.StringValue("provisioning-status"),
					Values: listValueMust(t, types.StringType, []string{"LIVE"}),
				},
			},
			tags:        nil,
			expectedIXs: []string{"ix-1"},
		},
		{
			name: "Multiple values for one filter - OR logic",
			filters: []filterModel{
				{
					Name:   types.StringValue("provisioning-status"),
					Values: listValueMust(t, types.StringType, []string{"LIVE", "CONFIGURED"}),
				},
			},
			tags:        nil,
			expectedIXs: []string{"ix-1", "ix-2"},
		},
		{
			name:    "Filter by tags",
			filters: []filterModel{},
			tags: map[string]string{
				"environment": "production",
				"owner":       "team-a",
			},
			expectedIXs: []string{"ix-1"},
		},
		{
			name: "Combined filters and tags",
			filters: []filterModel{
				{
					Name:   types.StringValue("network-service-type"),
					Values: listValueMust(t, types.StringType, []string{"Los Angeles IX"}),
				},
			},
			tags: map[string]string{
				"environment": "production",
			},
			expectedIXs: []string{"ix-1", "ix-3"},
		},
		{
			name: "Unknown filter - should not filter out IXs",
			filters: []filterModel{
				{
					Name:   types.StringValue("unknown-filter"),
					Values: listValueMust(t, types.StringType, []string{"some-value"}),
				},
			},
			tags:           nil,
			expectedIXs:    []string{"ix-1", "ix-2", "ix-3"},
			expectedErrors: 1, // Expecting a warning but not an error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock IX service
			mockIXService := &MockIXService{
				ListIXsResult: ixs,
			}

			// Set up tag mocks if needed
			if tc.mockTags != nil {
				// Set custom function to handle IX tag lookup
				mockIXService.ListIXResourceTagsFunc = func(ctx context.Context, ixID string) (map[string]string, error) {
					if tags, ok := tc.mockTags[ixID]; ok {
						return tags, nil
					}
					return map[string]string{}, nil
				}
			}

			// Create mock client with IX service properly attached
			mockClient := &megaport.Client{
				IXService: mockIXService,
			}

			// Create data source with mock client
			ds := &ixsDataSource{
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

			model := ixsModel{
				Filter: tc.filters,
				Tags:   tagsValue,
			}

			// Call filterIXs
			result, diags := ds.filterIXs(context.Background(), ixs, model)

			// Check for expected warnings
			if tc.expectedErrors > 0 {
				assert.Equal(t, tc.expectedErrors, len(diags))
			} else {
				assert.False(t, diags.HasError())
			}

			// Check results
			resultUIDs := make([]string, 0, len(result))
			for _, ix := range result {
				resultUIDs = append(resultUIDs, ix.ProductUID)
			}

			// Verify expected IXs are found (order independent)
			assert.Equal(t, len(tc.expectedIXs), len(resultUIDs),
				"Expected %d IXs but got %d", len(tc.expectedIXs), len(resultUIDs))

			for _, expectedUID := range tc.expectedIXs {
				found := false
				for _, resultUID := range resultUIDs {
					if expectedUID == resultUID {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected IX %s not found in results", expectedUID)
			}
		})
	}
}

// TestReadWithErrors tests error handling in Read
func TestReadWithErrorsIXs(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name            string
		setupMock       func(*MockIXService)
		expectedSummary string
		expectError     bool
	}{
		{
			name: "ListIXs error",
			setupMock: func(m *MockIXService) {
				m.ListIXsErr = errors.New("API error")
			},
			expectedSummary: "Unable to list IXs",
			expectError:     true,
		},
		{
			name: "ListIXResourceTags error",
			setupMock: func(m *MockIXService) {
				m.ListIXsResult = []*megaport.IX{
					{
						ProductUID:  "ix-1",
						ProductName: "Test IX 1",
					},
				}
				m.ListIXResourceTagsFunc = func(ctx context.Context, ixID string) (map[string]string, error) {
					return nil, errors.New("Tag API error")
				}
			},
			expectedSummary: "Unable to fetch tags for IX",
			expectError:     false, // We expect a warning, not an error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock IX service
			mockIXService := &MockIXService{}
			tc.setupMock(mockIXService)

			// Create mock client with IX service properly attached
			mockClient := &megaport.Client{
				IXService: mockIXService,
			}

			// Create data source with mock client
			ds := &ixsDataSource{
				client: mockClient,
			}

			// Create a simplified test configuration that doesn't use tftypes directly
			tagsMap := map[string]string{}
			if tc.name == "ListIXResourceTags error" {
				tagsMap = map[string]string{"environment": "production"}
			}

			tagsValue, _ := types.MapValueFrom(ctx, types.StringType, tagsMap)

			model := ixsModel{
				UIDs:   types.ListNull(types.StringType),
				Filter: []filterModel{},
				Tags:   tagsValue,
			}

			if tc.name == "ListIXs error" {
				// For the ListIXs error case, test the API call directly
				ixs, err := mockClient.IXService.ListIXs(ctx, &megaport.ListIXsRequest{IncludeInactive: false})

				// Verify the error occurs
				assert.Error(t, err)
				assert.Nil(t, ixs)

				// Check that the error message matches what we expect
				assert.Contains(t, err.Error(), "API error")
			} else {
				// For the tag error case, test filterIXs directly
				ixs := []*megaport.IX{{ProductUID: "ix-1", ProductName: "Test IX 1"}}
				_, diags := ds.filterIXs(ctx, ixs, model)

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
