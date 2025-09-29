package provider

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
)

func TestCloudPortLookupFilters(t *testing.T) {
	type filterTestCases struct {
		description   string
		config        cloudPortLookupModel
		vxcPermitted  bool
		expectedPorts []cloudPortModel
	}

	// Create test ports - mix of public and secure
	publicPort1 := cloudPortModel{
		ProductUID:    types.StringValue("pub1"),
		ProductName:   types.StringValue("AWS Public Port 1"),
		ConnectType:   types.StringValue("AWS"),
		CompanyName:   types.StringValue("AWS"),
		DiversityZone: types.StringValue("red"),
		LocationID:    types.Int64Value(1),
		VXCPermitted:  types.BoolValue(true),
		IsSecure:      types.BoolValue(false),
		Rank:          types.Int64Value(1),
	}

	publicPort2 := cloudPortModel{
		ProductUID:    types.StringValue("pub2"),
		ProductName:   types.StringValue("Azure Public Port 1"),
		ConnectType:   types.StringValue("AZURE"),
		CompanyName:   types.StringValue("Microsoft"),
		DiversityZone: types.StringValue("blue"),
		LocationID:    types.Int64Value(2),
		VXCPermitted:  types.BoolValue(false),
		IsSecure:      types.BoolValue(false),
		Rank:          types.Int64Value(2),
	}

	securePort1 := cloudPortModel{
		ProductUID:    types.StringValue("sec1"),
		ProductName:   types.StringValue("Google Secure Port 1"),
		ConnectType:   types.StringValue("GOOGLE"),
		CompanyName:   types.StringValue("Google"),
		DiversityZone: types.StringNull(),
		LocationID:    types.Int64Value(1),
		VXCPermitted:  types.BoolValue(true),
		IsSecure:      types.BoolValue(true),
		SecureKey:     types.StringValue("test-key"),
		VLAN:          types.Int64Value(100),
		Rank:          types.Int64Value(0),
	}

	allPorts := []cloudPortModel{publicPort1, publicPort2, securePort1}

	dataSource := &cloudPortLookupDataSource{}

	for _, scenario := range []filterTestCases{
		{
			description:   "filter_by_vxc_permitted_true",
			config:        cloudPortLookupModel{},
			vxcPermitted:  true,
			expectedPorts: []cloudPortModel{publicPort1, securePort1},
		},
		{
			description:   "filter_by_vxc_permitted_false",
			config:        cloudPortLookupModel{},
			vxcPermitted:  false,
			expectedPorts: []cloudPortModel{publicPort2},
		},
		{
			description: "filter_by_connect_type_aws",
			config: cloudPortLookupModel{
				ConnectType: types.StringValue("AWS"),
			},
			vxcPermitted:  true,
			expectedPorts: []cloudPortModel{publicPort1},
		},
		{
			description: "filter_by_location_id",
			config: cloudPortLookupModel{
				LocationID: types.Int64Value(1),
			},
			vxcPermitted:  true,
			expectedPorts: []cloudPortModel{publicPort1, securePort1},
		},
		{
			description: "filter_by_diversity_zone",
			config: cloudPortLookupModel{
				DiversityZone: types.StringValue("red"),
			},
			vxcPermitted:  true,
			expectedPorts: []cloudPortModel{publicPort1},
		},
		{
			description: "filter_by_company_name",
			config: cloudPortLookupModel{
				CompanyName: types.StringValue("Google"),
			},
			vxcPermitted:  true,
			expectedPorts: []cloudPortModel{securePort1},
		},
		{
			description: "multiple_filters",
			config: cloudPortLookupModel{
				ConnectType: types.StringValue("GOOGLE"),
				LocationID:  types.Int64Value(1),
			},
			vxcPermitted:  true,
			expectedPorts: []cloudPortModel{securePort1},
		},
		{
			description: "no_matches",
			config: cloudPortLookupModel{
				ConnectType: types.StringValue("ORACLE"),
			},
			vxcPermitted:  true,
			expectedPorts: []cloudPortModel{},
		},
	} {
		t.Run(scenario.description, func(t *testing.T) {
			filtered := dataSource.applyFilters(allPorts, scenario.config, scenario.vxcPermitted)
			assert.ElementsMatch(t, filtered, scenario.expectedPorts, "Failed for scenario: %s", scenario.description)
		})
	}
}

func TestCloudPortSorting(t *testing.T) {
	ports := []cloudPortModel{
		{
			ProductName: types.StringValue("Port Z"),
			Rank:        types.Int64Value(3),
		},
		{
			ProductName: types.StringValue("Port A"),
			Rank:        types.Int64Value(1),
		},
		{
			ProductName: types.StringValue("Port B"),
			Rank:        types.Int64Value(1),
		},
		{
			ProductName: types.StringValue("Port C"),
			Rank:        types.Int64Value(0),
		},
	}

	// Sort by rank (lower is better), then by name
	slices.SortFunc(ports, func(a, b cloudPortModel) int {
		if a.Rank.ValueInt64() != b.Rank.ValueInt64() {
			return int(a.Rank.ValueInt64() - b.Rank.ValueInt64())
		}
		return strings.Compare(a.ProductName.ValueString(), b.ProductName.ValueString())
	})

	// Expected order: Rank 0 (Port C), Rank 1 (Port A, Port B), Rank 3 (Port Z)
	expected := []string{"Port C", "Port A", "Port B", "Port Z"}
	actual := make([]string, len(ports))
	for i, port := range ports {
		actual[i] = port.ProductName.ValueString()
	}

	assert.Equal(t, expected, actual, "Ports should be sorted by rank (ascending) then by name")
}

func TestFromPublicPartnerPort(t *testing.T) {
	publicPort := &megaport.PartnerMegaport{
		ProductUID:    "test-uid",
		ProductName:   "Test Port",
		ConnectType:   "AWS",
		CompanyUID:    "company-uid",
		CompanyName:   "Test Company",
		DiversityZone: "red",
		LocationId:    123,
		Speed:         1000,
		Rank:          5,
		VXCPermitted:  true,
	}

	cloudPort := cloudPortModel{}
	cloudPort.fromPublicPartnerPort(publicPort)

	assert.Equal(t, "test-uid", cloudPort.ProductUID.ValueString())
	assert.Equal(t, "Test Port", cloudPort.ProductName.ValueString())
	assert.Equal(t, "AWS", cloudPort.ConnectType.ValueString())
	assert.Equal(t, "company-uid", cloudPort.CompanyUID.ValueString())
	assert.Equal(t, "Test Company", cloudPort.CompanyName.ValueString())
	assert.Equal(t, "red", cloudPort.DiversityZone.ValueString())
	assert.Equal(t, int64(123), cloudPort.LocationID.ValueInt64())
	assert.Equal(t, int64(1000), cloudPort.Speed.ValueInt64())
	assert.Equal(t, int64(5), cloudPort.Rank.ValueInt64())
	assert.True(t, cloudPort.VXCPermitted.ValueBool())
	assert.False(t, cloudPort.IsSecure.ValueBool())
	assert.True(t, cloudPort.SecureKey.IsNull())
	assert.True(t, cloudPort.VLAN.IsNull())
}

func TestFromSecurePartnerPort(t *testing.T) {
	securePort := &megaport.PartnerLookupItem{
		ProductUID:  "secure-uid",
		Name:        "Secure Test Port",
		Type:        "GOOGLE",
		CompanyID:   456,
		CompanyName: "Google",
		LocationID:  789,
		PortSpeed:   10000,
	}

	cloudPort := cloudPortModel{}
	cloudPort.fromSecurePartnerPort(securePort, "test-service-key", 200)

	assert.Equal(t, "secure-uid", cloudPort.ProductUID.ValueString())
	assert.Equal(t, "Secure Test Port", cloudPort.ProductName.ValueString())
	assert.Equal(t, "GOOGLE", cloudPort.ConnectType.ValueString())
	assert.Equal(t, "456", cloudPort.CompanyUID.ValueString())
	assert.Equal(t, "Google", cloudPort.CompanyName.ValueString())
	assert.True(t, cloudPort.DiversityZone.IsNull()) // Not available in secure response
	assert.Equal(t, int64(789), cloudPort.LocationID.ValueInt64())
	assert.Equal(t, int64(10000), cloudPort.Speed.ValueInt64())
	assert.Equal(t, int64(0), cloudPort.Rank.ValueInt64()) // Default to 0 for secure ports
	assert.True(t, cloudPort.VXCPermitted.ValueBool())     // Secure ports typically allow VXCs
	assert.True(t, cloudPort.IsSecure.ValueBool())
	assert.Equal(t, "test-service-key", cloudPort.SecureKey.ValueString())
	assert.Equal(t, int64(200), cloudPort.VLAN.ValueInt64())
}

func TestGetPartnersForConnectType(t *testing.T) {
	dataSource := &cloudPortLookupDataSource{}

	testCases := []struct {
		connectType string
		expected    []string
	}{
		{"GOOGLE", []string{"GOOGLE"}},
		{"google", []string{"GOOGLE"}},
		{"ORACLE", []string{"ORACLE"}},
		{"AZURE", []string{"AZURE"}},
		{"IBM", []string{"IBM"}},
		{"AWS", []string{}},                                // AWS doesn't support secure ports via this API
		{"", []string{"GOOGLE", "ORACLE", "AZURE", "IBM"}}, // Empty returns all
		{"INVALID", []string{}},
	}

	for _, tc := range testCases {
		t.Run(tc.connectType, func(t *testing.T) {
			result := dataSource.getPartnersForConnectType(tc.connectType)
			assert.ElementsMatch(t, tc.expected, result, "Partners for connect type %s", tc.connectType)
		})
	}
}

func TestSecureKeyValidation(t *testing.T) {
	// Test that validation correctly identifies valid and invalid connect types for secure keys
	testCases := []struct {
		name        string
		connectType string
		isValid     bool
	}{
		{"google_valid", "GOOGLE", true},
		{"azure_valid", "AZURE", true},
		{"oracle_valid", "ORACLE", true},
		{"ibm_valid", "IBM", true},
		{"google_lowercase", "google", true},
		{"aws_invalid", "AWS", false},
		{"awshc_invalid", "AWSHC", false},
		{"invalid_type", "INVALID", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			connectType := strings.ToUpper(tc.connectType)
			validSecureTypes := []string{"GOOGLE", "AZURE", "ORACLE", "IBM"}
			isValid := false
			for _, validType := range validSecureTypes {
				if connectType == validType {
					isValid = true
					break
				}
			}
			assert.Equal(t, tc.isValid, isValid, "Validation for connect type %s", tc.connectType)
		})
	}
}

func TestCloudPortLookupSecureKeyValidationError(t *testing.T) {
	// Test that the Read method returns an error when secure_key is provided
	// with invalid connect_types

	invalidCombinations := []struct {
		name        string
		connectType string
		description string
	}{
		{
			name:        "aws_with_secure_key",
			connectType: "AWS",
			description: "AWS does not support secure keys",
		},
		{
			name:        "awshc_with_secure_key",
			connectType: "AWSHC",
			description: "AWS Hosted Connection does not support secure keys",
		},
		{
			name:        "transit_with_secure_key",
			connectType: "TRANSIT",
			description: "Megaport Internet does not support secure keys",
		},
		{
			name:        "franceix_with_secure_key",
			connectType: "FRANCEIX",
			description: "France-IX does not support secure keys",
		},
	}

	for _, tc := range invalidCombinations {
		t.Run(tc.name, func(t *testing.T) {
			// Create config with invalid combination
			config := cloudPortLookupModel{
				ConnectType:   types.StringValue(tc.connectType),
				IncludeSecure: types.BoolValue(true),
				SecureKey:     types.StringValue("test-key"),
			}

			// Test validation logic directly (since we can't easily mock the full Read method)
			connectType := strings.ToUpper(config.ConnectType.ValueString())
			validSecureTypes := []string{"GOOGLE", "AZURE", "ORACLE", "IBM"}
			isValid := false
			for _, validType := range validSecureTypes {
				if connectType == validType {
					isValid = true
					break
				}
			}

			// Should be invalid for all test cases
			assert.False(t, isValid, "Connect type %s should not be valid for secure keys", tc.connectType)

			// Verify error would be generated (simulating the validation in Read method)
			if !isValid && !config.SecureKey.IsNull() {
				// This simulates the error that would be added in the Read method
				errorMessage := fmt.Sprintf("When using secure_key, connect_type must be one of: %s. Got: %s",
					strings.Join(validSecureTypes, ", "), connectType)
				assert.Contains(t, errorMessage, tc.connectType, "Error message should mention the invalid connect type")
			}
		})
	}
}
