package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

func TestParseImportID(t *testing.T) {
	tests := []struct {
		name       string
		importID   string
		wantMCRUID string
		wantListID int64
		wantError  bool
		errorMsg   string
	}{
		{
			name:       "valid import ID",
			importID:   "12345678-1234-1234-1234-123456789012:5678",
			wantMCRUID: "12345678-1234-1234-1234-123456789012",
			wantListID: 5678,
			wantError:  false,
		},
		{
			name:       "valid import ID with short MCR UID",
			importID:   "abc-123:999",
			wantMCRUID: "abc-123",
			wantListID: 999,
			wantError:  false,
		},
		{
			name:      "invalid format - missing colon",
			importID:  "12345678-1234-1234-1234-123456789012",
			wantError: true,
			errorMsg:  "invalid import ID format, expected 'mcr_uid:prefix_list_id'",
		},
		{
			name:      "invalid format - empty string",
			importID:  "",
			wantError: true,
			errorMsg:  "invalid import ID format, expected 'mcr_uid:prefix_list_id'",
		},
		{
			name:      "invalid format - only colon",
			importID:  ":",
			wantError: true,
			errorMsg:  "MCR UID and prefix list ID cannot be empty",
		},
		{
			name:      "invalid format - empty MCR UID",
			importID:  ":123",
			wantError: true,
			errorMsg:  "MCR UID and prefix list ID cannot be empty",
		},
		{
			name:      "invalid format - empty prefix list ID",
			importID:  "mcr-uid:",
			wantError: true,
			errorMsg:  "MCR UID and prefix list ID cannot be empty",
		},
		{
			name:      "invalid list ID - not numeric",
			importID:  "12345678-1234-1234-1234-123456789012:abc",
			wantError: true,
			errorMsg:  "invalid prefix list ID 'abc'",
		},
		{
			name:      "invalid list ID - negative number",
			importID:  "12345678-1234-1234-1234-123456789012:-123",
			wantError: true,
			errorMsg:  "invalid prefix list ID '-123': must be a positive integer",
		},
		{
			name:      "invalid list ID - zero",
			importID:  "12345678-1234-1234-1234-123456789012:0",
			wantError: true,
			errorMsg:  "invalid prefix list ID '0': must be a positive integer",
		},
		{
			name:       "multiple colons - should use first as separator",
			importID:   "mcr:uid:with:colons:123",
			wantMCRUID: "mcr",
			wantListID: 0, // This should fail parsing
			wantError:  true,
			errorMsg:   "invalid prefix list ID 'uid:with:colons:123'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcrUID, listID, err := parseImportID(tt.importID)

			if tt.wantError {
				if err == nil {
					t.Errorf("parseImportID() expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					// Allow partial match for dynamic error messages
					if len(tt.errorMsg) > 10 && !contains(err.Error(), tt.errorMsg[:10]) {
						t.Errorf("parseImportID() error = %v, want to contain %v", err.Error(), tt.errorMsg)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("parseImportID() unexpected error: %v", err)
				return
			}

			if mcrUID != tt.wantMCRUID {
				t.Errorf("parseImportID() mcrUID = %v, want %v", mcrUID, tt.wantMCRUID)
			}

			if listID != tt.wantListID {
				t.Errorf("parseImportID() listID = %v, want %v", listID, tt.wantListID)
			}
		})
	}
}

func TestGenerateImportID(t *testing.T) {
	tests := []struct {
		name         string
		mcrUID       string
		prefixListID int64
		want         string
	}{
		{
			name:         "standard format",
			mcrUID:       "12345678-1234-1234-1234-123456789012",
			prefixListID: 5678,
			want:         "12345678-1234-1234-1234-123456789012:5678",
		},
		{
			name:         "short MCR UID",
			mcrUID:       "abc",
			prefixListID: 1,
			want:         "abc:1",
		},
		{
			name:         "large prefix list ID",
			mcrUID:       "test-mcr",
			prefixListID: 999999,
			want:         "test-mcr:999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateImportID(tt.mcrUID, tt.prefixListID)
			if got != tt.want {
				t.Errorf("generateImportID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatePrefixListEntry(t *testing.T) {
	tests := []struct {
		name          string
		entry         mcrPrefixFilterListEntryResourceModel
		addressFamily string
		index         int
		wantError     bool
		errorContains string
	}{
		{
			name: "valid IPv4 entry",
			entry: mcrPrefixFilterListEntryResourceModel{
				Action: types.StringValue("permit"),
				Prefix: types.StringValue("10.0.0.0/8"),
				Ge:     types.Int64Value(16),
				Le:     types.Int64Value(24),
			},
			addressFamily: "IPv4",
			index:         0,
			wantError:     false,
		},
		{
			name: "valid IPv6 entry",
			entry: mcrPrefixFilterListEntryResourceModel{
				Action: types.StringValue("deny"),
				Prefix: types.StringValue("2001:db8::/32"),
				Ge:     types.Int64Value(48),
				Le:     types.Int64Value(64),
			},
			addressFamily: "IPv6",
			index:         0,
			wantError:     false,
		},
		{
			name: "invalid prefix format",
			entry: mcrPrefixFilterListEntryResourceModel{
				Action: types.StringValue("permit"),
				Prefix: types.StringValue("invalid-prefix"),
				Ge:     types.Int64Value(16),
				Le:     types.Int64Value(24),
			},
			addressFamily: "IPv4",
			index:         0,
			wantError:     true,
			errorContains: "Invalid prefix in entry 0",
		},
		{
			name: "IPv4 prefix with IPv6 family",
			entry: mcrPrefixFilterListEntryResourceModel{
				Action: types.StringValue("permit"),
				Prefix: types.StringValue("10.0.0.0/8"),
				Ge:     types.Int64Value(16),
				Le:     types.Int64Value(24),
			},
			addressFamily: "IPv6",
			index:         1,
			wantError:     true,
			errorContains: "Address family mismatch in entry 1",
		},
		{
			name: "IPv6 prefix with IPv4 family",
			entry: mcrPrefixFilterListEntryResourceModel{
				Action: types.StringValue("permit"),
				Prefix: types.StringValue("2001:db8::/32"),
				Ge:     types.Int64Value(48),
				Le:     types.Int64Value(64),
			},
			addressFamily: "IPv4",
			index:         0,
			wantError:     true,
			errorContains: "Address family mismatch in entry 0",
		},
		{
			name: "invalid ge value for IPv4",
			entry: mcrPrefixFilterListEntryResourceModel{
				Action: types.StringValue("permit"),
				Prefix: types.StringValue("10.0.0.0/8"),
				Ge:     types.Int64Value(40), // Invalid for IPv4 (max is 32)
				Le:     types.Int64Value(24),
			},
			addressFamily: "IPv4",
			index:         0,
			wantError:     true,
			errorContains: "Invalid ge value in entry 0",
		},
		{
			name: "invalid le value for IPv6",
			entry: mcrPrefixFilterListEntryResourceModel{
				Action: types.StringValue("permit"),
				Prefix: types.StringValue("2001:db8::/32"),
				Ge:     types.Int64Value(48),
				Le:     types.Int64Value(200), // Invalid for IPv6 (max is 128)
			},
			addressFamily: "IPv6",
			index:         2,
			wantError:     true,
			errorContains: "Invalid le value in entry 2",
		},
		{
			name: "negative ge value",
			entry: mcrPrefixFilterListEntryResourceModel{
				Action: types.StringValue("permit"),
				Prefix: types.StringValue("10.0.0.0/8"),
				Ge:     types.Int64Value(-1),
				Le:     types.Int64Value(24),
			},
			addressFamily: "IPv4",
			index:         0,
			wantError:     true,
			errorContains: "Invalid ge value in entry 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := validatePrefixListEntry(&tt.entry, tt.addressFamily, tt.index)
			hasError := diags.HasError()

			if hasError != tt.wantError {
				t.Errorf("validatePrefixListEntry() hasError = %v, want %v", hasError, tt.wantError)
				if hasError {
					t.Errorf("Diagnostics: %v", diags)
				}
				return
			}

			if tt.wantError && tt.errorContains != "" {
				found := false
				for _, d := range diags {
					if contains(d.Summary(), tt.errorContains) || contains(d.Detail(), tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("validatePrefixListEntry() error should contain %v, got diagnostics: %v", tt.errorContains, diags)
				}
			}
		})
	}
}

func TestCalculateGeLe(t *testing.T) {
	tests := []struct {
		name          string
		entry         mcrPrefixFilterListEntryResourceModel
		addressFamily string
		wantGe        int
		wantLe        int
		wantError     bool
		errorContains string
	}{
		{
			name: "explicit ge and le values - IPv4",
			entry: mcrPrefixFilterListEntryResourceModel{
				Prefix: types.StringValue("10.0.0.0/8"),
				Ge:     types.Int64Value(16),
				Le:     types.Int64Value(24),
			},
			addressFamily: "IPv4",
			wantGe:        16,
			wantLe:        24,
			wantError:     false,
		},
		{
			name: "default ge and le values - IPv4",
			entry: mcrPrefixFilterListEntryResourceModel{
				Prefix: types.StringValue("192.168.0.0/16"),
				Ge:     types.Int64Null(),
				Le:     types.Int64Null(),
			},
			addressFamily: "IPv4",
			wantGe:        16, // Default to prefix length
			wantLe:        32, // Default to max for IPv4
			wantError:     false,
		},
		{
			name: "explicit ge and le values - IPv6",
			entry: mcrPrefixFilterListEntryResourceModel{
				Prefix: types.StringValue("2001:db8::/32"),
				Ge:     types.Int64Value(48),
				Le:     types.Int64Value(64),
			},
			addressFamily: "IPv6",
			wantGe:        48,
			wantLe:        64,
			wantError:     false,
		},
		{
			name: "default ge and le values - IPv6",
			entry: mcrPrefixFilterListEntryResourceModel{
				Prefix: types.StringValue("fd00::/8"),
				Ge:     types.Int64Null(),
				Le:     types.Int64Null(),
			},
			addressFamily: "IPv6",
			wantGe:        8,   // Default to prefix length
			wantLe:        128, // Default to max for IPv6
			wantError:     false,
		},
		{
			name: "ge greater than le",
			entry: mcrPrefixFilterListEntryResourceModel{
				Prefix: types.StringValue("10.0.0.0/8"),
				Ge:     types.Int64Value(24),
				Le:     types.Int64Value(16),
			},
			addressFamily: "IPv4",
			wantError:     true,
			errorContains: "ge (24) cannot be greater than le (16)",
		},
		{
			name: "ge less than prefix length",
			entry: mcrPrefixFilterListEntryResourceModel{
				Prefix: types.StringValue("192.168.0.0/16"),
				Ge:     types.Int64Value(8),
				Le:     types.Int64Value(24),
			},
			addressFamily: "IPv4",
			wantError:     true,
			errorContains: "ge (8) cannot be less than the prefix length (16)",
		},
		{
			name: "le greater than max length IPv4",
			entry: mcrPrefixFilterListEntryResourceModel{
				Prefix: types.StringValue("10.0.0.0/8"),
				Ge:     types.Int64Value(16),
				Le:     types.Int64Value(40),
			},
			addressFamily: "IPv4",
			wantError:     true,
			errorContains: "le (40) cannot be greater than 32 for IPv4",
		},
		{
			name: "le greater than max length IPv6",
			entry: mcrPrefixFilterListEntryResourceModel{
				Prefix: types.StringValue("2001:db8::/32"),
				Ge:     types.Int64Value(48),
				Le:     types.Int64Value(200),
			},
			addressFamily: "IPv6",
			wantError:     true,
			errorContains: "le (200) cannot be greater than 128 for IPv6",
		},
		{
			name: "invalid prefix format",
			entry: mcrPrefixFilterListEntryResourceModel{
				Prefix: types.StringValue("invalid-prefix"),
				Ge:     types.Int64Value(16),
				Le:     types.Int64Value(24),
			},
			addressFamily: "IPv4",
			wantError:     true,
			errorContains: "Invalid prefix format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ge, le, diags := calculateGeLe(&tt.entry, tt.addressFamily)
			hasError := diags.HasError()

			if hasError != tt.wantError {
				t.Errorf("calculateGeLe() hasError = %v, want %v", hasError, tt.wantError)
				if hasError {
					t.Errorf("Diagnostics: %v", diags)
				}
				return
			}

			if !tt.wantError {
				if ge != tt.wantGe {
					t.Errorf("calculateGeLe() ge = %v, want %v", ge, tt.wantGe)
				}
				if le != tt.wantLe {
					t.Errorf("calculateGeLe() le = %v, want %v", le, tt.wantLe)
				}
			}

			if tt.wantError && tt.errorContains != "" {
				found := false
				for _, d := range diags {
					if contains(d.Summary(), tt.errorContains) || contains(d.Detail(), tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("calculateGeLe() error should contain %v, got diagnostics: %v", tt.errorContains, diags)
				}
			}
		})
	}
}

func TestCalculateGeLeFromPrefix(t *testing.T) {
	tests := []struct {
		name          string
		prefix        string
		addressFamily string
		wantGe        int
		wantLe        int
		wantError     bool
		errorContains string
	}{
		{
			name:          "IPv4 /8 prefix",
			prefix:        "10.0.0.0/8",
			addressFamily: "IPv4",
			wantGe:        8,
			wantLe:        32,
			wantError:     false,
		},
		{
			name:          "IPv4 /24 prefix",
			prefix:        "192.168.1.0/24",
			addressFamily: "IPv4",
			wantGe:        24,
			wantLe:        32,
			wantError:     false,
		},
		{
			name:          "IPv6 /32 prefix",
			prefix:        "2001:db8::/32",
			addressFamily: "IPv6",
			wantGe:        32,
			wantLe:        128,
			wantError:     false,
		},
		{
			name:          "IPv6 /64 prefix",
			prefix:        "fd00:1234:5678:9abc::/64",
			addressFamily: "IPv6",
			wantGe:        64,
			wantLe:        128,
			wantError:     false,
		},
		{
			name:          "invalid prefix format",
			prefix:        "invalid-prefix",
			addressFamily: "IPv4",
			wantError:     true,
			errorContains: "Invalid prefix format",
		},
		{
			name:          "empty prefix",
			prefix:        "",
			addressFamily: "IPv4",
			wantError:     true,
			errorContains: "Invalid prefix format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ge, le, diags := calculateGeLeFromPrefix(tt.prefix, tt.addressFamily)
			hasError := diags.HasError()

			if hasError != tt.wantError {
				t.Errorf("calculateGeLeFromPrefix() hasError = %v, want %v", hasError, tt.wantError)
				if hasError {
					t.Errorf("Diagnostics: %v", diags)
				}
				return
			}

			if !tt.wantError {
				if ge != tt.wantGe {
					t.Errorf("calculateGeLeFromPrefix() ge = %v, want %v", ge, tt.wantGe)
				}
				if le != tt.wantLe {
					t.Errorf("calculateGeLeFromPrefix() le = %v, want %v", le, tt.wantLe)
				}
			}

			if tt.wantError && tt.errorContains != "" {
				found := false
				for _, d := range diags {
					if contains(d.Summary(), tt.errorContains) || contains(d.Detail(), tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("calculateGeLeFromPrefix() error should contain %v, got diagnostics: %v", tt.errorContains, diags)
				}
			}
		})
	}
}

func TestConvertEntryToAPI(t *testing.T) {
	tests := []struct {
		name          string
		entry         mcrPrefixFilterListEntryResourceModel
		addressFamily string
		wantEntry     *megaport.MCRPrefixListEntry
		wantError     bool
		errorContains string
	}{
		{
			name: "valid IPv4 entry",
			entry: mcrPrefixFilterListEntryResourceModel{
				Action: types.StringValue("permit"),
				Prefix: types.StringValue("10.0.0.0/8"),
				Ge:     types.Int64Value(16),
				Le:     types.Int64Value(24),
			},
			addressFamily: "IPv4",
			wantEntry: &megaport.MCRPrefixListEntry{
				Action: "permit",
				Prefix: "10.0.0.0/8",
				Ge:     16,
				Le:     24,
			},
			wantError: false,
		},
		{
			name: "valid IPv6 entry",
			entry: mcrPrefixFilterListEntryResourceModel{
				Action: types.StringValue("deny"),
				Prefix: types.StringValue("2001:db8::/32"),
				Ge:     types.Int64Value(48),
				Le:     types.Int64Value(64),
			},
			addressFamily: "IPv6",
			wantEntry: &megaport.MCRPrefixListEntry{
				Action: "deny",
				Prefix: "2001:db8::/32",
				Ge:     48,
				Le:     64,
			},
			wantError: false,
		},
		{
			name: "entry with null ge/le values",
			entry: mcrPrefixFilterListEntryResourceModel{
				Action: types.StringValue("permit"),
				Prefix: types.StringValue("192.168.0.0/16"),
				Ge:     types.Int64Null(),
				Le:     types.Int64Null(),
			},
			addressFamily: "IPv4",
			wantEntry: &megaport.MCRPrefixListEntry{
				Action: "permit",
				Prefix: "192.168.0.0/16",
				Ge:     16, // Should default to prefix length
				Le:     32, // Should default to max for IPv4
			},
			wantError: false,
		},
		{
			name: "invalid prefix",
			entry: mcrPrefixFilterListEntryResourceModel{
				Action: types.StringValue("permit"),
				Prefix: types.StringValue("invalid-prefix"),
				Ge:     types.Int64Value(16),
				Le:     types.Int64Value(24),
			},
			addressFamily: "IPv4",
			wantError:     true,
			errorContains: "Invalid prefix format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiEntry, diags := convertEntryToAPI(&tt.entry, tt.addressFamily)
			hasError := diags.HasError()

			if hasError != tt.wantError {
				t.Errorf("convertEntryToAPI() hasError = %v, want %v", hasError, tt.wantError)
				if hasError {
					t.Errorf("Diagnostics: %v", diags)
				}
				return
			}

			if !tt.wantError && tt.wantEntry != nil {
				if apiEntry.Action != tt.wantEntry.Action {
					t.Errorf("convertEntryToAPI() Action = %v, want %v", apiEntry.Action, tt.wantEntry.Action)
				}
				if apiEntry.Prefix != tt.wantEntry.Prefix {
					t.Errorf("convertEntryToAPI() Prefix = %v, want %v", apiEntry.Prefix, tt.wantEntry.Prefix)
				}
				if apiEntry.Ge != tt.wantEntry.Ge {
					t.Errorf("convertEntryToAPI() Ge = %v, want %v", apiEntry.Ge, tt.wantEntry.Ge)
				}
				if apiEntry.Le != tt.wantEntry.Le {
					t.Errorf("convertEntryToAPI() Le = %v, want %v", apiEntry.Le, tt.wantEntry.Le)
				}
			}

			if tt.wantError && tt.errorContains != "" {
				found := false
				for _, d := range diags {
					if contains(d.Summary(), tt.errorContains) || contains(d.Detail(), tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("convertEntryToAPI() error should contain %v, got diagnostics: %v", tt.errorContains, diags)
				}
			}
		})
	}
}

func TestFromAPI(t *testing.T) {
	tests := []struct {
		name          string
		apiList       *megaport.MCRPrefixFilterList
		wantModel     *mcrPrefixFilterListResourceModel
		wantError     bool
		errorContains string
	}{
		{
			name: "valid API response with entries",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            123,
				Description:   "Test prefix list",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{
						Action: "permit",
						Prefix: "10.0.0.0/8",
						Ge:     16,
						Le:     24,
					},
					{
						Action: "deny",
						Prefix: "192.168.0.0/16",
						Ge:     24,
						Le:     32,
					},
				},
			},
			wantError: false,
		},
		{
			name: "API response with zero ge/le values (should calculate)",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            456,
				Description:   "Test prefix list with zero values",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{
						Action: "permit",
						Prefix: "10.0.0.0/8",
						Ge:     0, // Should be calculated to 8
						Le:     0, // Should be calculated to 32
					},
				},
			},
			wantError: false,
		},
		{
			name: "empty API response",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            789,
				Description:   "Empty list",
				AddressFamily: "IPv6",
				Entries:       []*megaport.MCRPrefixListEntry{},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &mcrPrefixFilterListResourceModel{}
			diags := model.fromAPI(context.Background(), tt.apiList)
			hasError := diags.HasError()

			if hasError != tt.wantError {
				t.Errorf("fromAPI() hasError = %v, want %v", hasError, tt.wantError)
				if hasError {
					t.Errorf("Diagnostics: %v", diags)
				}
				return
			}

			if !tt.wantError {
				// Verify basic fields are populated
				if model.ID.ValueInt64() != int64(tt.apiList.ID) {
					t.Errorf("fromAPI() ID = %v, want %v", model.ID.ValueInt64(), tt.apiList.ID)
				}
				if model.Description.ValueString() != tt.apiList.Description {
					t.Errorf("fromAPI() Description = %v, want %v", model.Description.ValueString(), tt.apiList.Description)
				}
				if model.AddressFamily.ValueString() != tt.apiList.AddressFamily {
					t.Errorf("fromAPI() AddressFamily = %v, want %v", model.AddressFamily.ValueString(), tt.apiList.AddressFamily)
				}
			}

			if tt.wantError && tt.errorContains != "" {
				found := false
				for _, d := range diags {
					if contains(d.Summary(), tt.errorContains) || contains(d.Detail(), tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("fromAPI() error should contain %v, got diagnostics: %v", tt.errorContains, diags)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
