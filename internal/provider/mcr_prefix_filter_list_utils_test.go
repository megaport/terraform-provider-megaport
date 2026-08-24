package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

func TestNormalizeCIDR(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "IPv4 with host bits set",
			input: "162.43.146.93/31",
			want:  "162.43.146.92/31",
		},
		{
			name:  "IPv4 already canonical",
			input: "10.0.0.0/8",
			want:  "10.0.0.0/8",
		},
		{
			name:  "IPv4 /32 host address",
			input: "10.0.0.1/32",
			want:  "10.0.0.1/32",
		},
		{
			name:  "IPv4 host bits in /24",
			input: "192.168.1.100/24",
			want:  "192.168.1.0/24",
		},
		{
			name:  "IPv6 with host bits set",
			input: "2001:db8::1/32",
			want:  "2001:db8::/32",
		},
		{
			name:  "IPv6 already canonical",
			input: "2001:db8::/32",
			want:  "2001:db8::/32",
		},
		{
			name:  "IPv6 /128 host address",
			input: "2001:db8::1/128",
			want:  "2001:db8::1/128",
		},
		{
			name:  "invalid input returns unchanged",
			input: "invalid-prefix",
			want:  "invalid-prefix",
		},
		{
			name:  "empty string returns unchanged",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeCIDR(tt.input)
			if got != tt.want {
				t.Errorf("normalizeCIDR(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCanonicalCIDRValidator(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "canonical IPv4 - no error",
			input:     "10.0.0.0/8",
			wantError: false,
		},
		{
			name:      "canonical IPv4 /24 - no error",
			input:     "192.168.1.0/24",
			wantError: false,
		},
		{
			name:      "canonical IPv4 /32 host - no error",
			input:     "10.0.0.1/32",
			wantError: false,
		},
		{
			name:      "canonical IPv6 - no error",
			input:     "2001:db8::/32",
			wantError: false,
		},
		{
			name:      "IPv4 with host bits set",
			input:     "192.168.1.100/24",
			wantError: true,
			errorMsg:  `Use the network address "192.168.1.0/24" instead`,
		},
		{
			name:      "IPv4 /31 with host bits set",
			input:     "162.43.146.93/31",
			wantError: true,
			errorMsg:  `Use the network address "162.43.146.92/31" instead`,
		},
		{
			name:      "IPv6 with host bits set",
			input:     "2001:db8::1/32",
			wantError: true,
			errorMsg:  `Use the network address "2001:db8::/32" instead`,
		},
		{
			name:      "invalid CIDR - no error (let other validators handle it)",
			input:     "invalid-prefix",
			wantError: false,
		},
	}

	v := canonicalCIDRValidator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.input),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			if tt.wantError && !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q but got none", tt.input)
			}
			if !tt.wantError && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error for %q: %v", tt.input, resp.Diagnostics)
			}
			if tt.wantError && tt.errorMsg != "" {
				found := false
				for _, d := range resp.Diagnostics {
					if contains(d.Detail(), tt.errorMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("error should contain %q, got: %v", tt.errorMsg, resp.Diagnostics)
				}
			}
		})
	}
}

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
			// Create a resource instance to call the validation method
			resource := &mcrPrefixFilterListResource{}
			diags := resource.validatePrefixListEntry(&tt.entry, tt.addressFamily, tt.index)
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

func TestResolveGeLe(t *testing.T) {
	tests := []struct {
		name          string
		entry         *megaport.MCRPrefixListEntry
		wantGe        int
		wantLe        int
		wantError     bool
		errorContains string
	}{
		{
			name:   "both absent on an IPv4 /8",
			entry:  &megaport.MCRPrefixListEntry{Prefix: "10.0.0.0/8"},
			wantGe: 8,
			wantLe: 8,
		},
		{
			name:   "both absent on an IPv4 /24",
			entry:  &megaport.MCRPrefixListEntry{Prefix: "192.168.1.0/24"},
			wantGe: 24,
			wantLe: 24,
		},
		{
			name:   "both absent on an IPv6 /32",
			entry:  &megaport.MCRPrefixListEntry{Prefix: "2001:db8::/32"},
			wantGe: 32,
			wantLe: 32,
		},
		{
			name:   "both absent on an IPv6 /64",
			entry:  &megaport.MCRPrefixListEntry{Prefix: "fd00:1234:5678:9abc::/64"},
			wantGe: 64,
			wantLe: 64,
		},
		{
			name:   "le returned, ge absent",
			entry:  &megaport.MCRPrefixListEntry{Prefix: "10.0.0.0/8", Le: 24},
			wantGe: 8,
			wantLe: 24,
		},
		{
			name:   "ge returned above the prefix length, le absent",
			entry:  &megaport.MCRPrefixListEntry{Prefix: "10.0.0.0/24", Ge: 26},
			wantGe: 26,
			wantLe: 32,
		},
		{
			name:   "IPv6 ge returned above the prefix length, le absent",
			entry:  &megaport.MCRPrefixListEntry{Prefix: "2001:db8::/32", Ge: 48},
			wantGe: 48,
			wantLe: 128,
		},
		{
			name:   "default route with an explicit le",
			entry:  &megaport.MCRPrefixListEntry{Prefix: "0.0.0.0/0", Le: 32},
			wantGe: 0,
			wantLe: 32,
		},
		{
			name:   "both returned are passed through",
			entry:  &megaport.MCRPrefixListEntry{Prefix: "10.0.0.0/8", Ge: 16, Le: 32},
			wantGe: 16,
			wantLe: 32,
		},
		{
			name:          "invalid prefix format",
			entry:         &megaport.MCRPrefixListEntry{Prefix: "invalid-prefix"},
			wantError:     true,
			errorContains: "Invalid prefix format",
		},
		{
			name:          "empty prefix",
			entry:         &megaport.MCRPrefixListEntry{Prefix: ""},
			wantError:     true,
			errorContains: "Invalid prefix format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ge, le, diags := resolveGeLe(tt.entry)
			hasError := diags.HasError()

			if hasError != tt.wantError {
				t.Errorf("resolveGeLe() hasError = %v, want %v", hasError, tt.wantError)
				if hasError {
					t.Errorf("Diagnostics: %v", diags)
				}
				return
			}

			if !tt.wantError && (ge != tt.wantGe || le != tt.wantLe) {
				t.Errorf("resolveGeLe() = (%v, %v), want (%v, %v)", ge, le, tt.wantGe, tt.wantLe)
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
					t.Errorf("resolveGeLe() error should contain %v, got diagnostics: %v", tt.errorContains, diags)
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
			name: "canonical prefix passes through unchanged",
			entry: mcrPrefixFilterListEntryResourceModel{
				Action: types.StringValue("permit"),
				Prefix: types.StringValue("162.43.146.92/31"),
				Ge:     types.Int64Value(31),
				Le:     types.Int64Value(31),
			},
			addressFamily: "IPv4",
			wantEntry: &megaport.MCRPrefixListEntry{
				Action: "permit",
				Prefix: "162.43.146.92/31",
				Ge:     31,
				Le:     31,
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
						Ge:     0, // Absent, resolves to 8
						Le:     0, // Absent, resolves to 8
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
		// fromAPI keeps every ge and le the API returns.
		{
			name: "IPv4 with le=32 (max) - returns raw API values",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            1001,
				Description:   "IPv4 exact match test",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{
						Action: "permit",
						Prefix: "10.0.0.0/24",
						Ge:     24, // Explicit ge, kept as sent
						Le:     32, // Explicit le, kept as sent
					},
				},
			},
			wantError: false,
		},
		{
			name: "IPv4 with le=32 (max) for /16 prefix - returns raw API values",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            1002,
				Description:   "IPv4 exact match test /16",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{
						Action: "deny",
						Prefix: "172.16.0.0/16",
						Ge:     16, // Explicit ge, kept as sent
						Le:     32, // Explicit le, kept as sent
					},
				},
			},
			wantError: false,
		},
		{
			name: "IPv6 with le=128 (max) - returns raw API values",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            1003,
				Description:   "IPv6 exact match test",
				AddressFamily: "IPv6",
				Entries: []*megaport.MCRPrefixListEntry{
					{
						Action: "permit",
						Prefix: "2001:db8::/64",
						Ge:     64,  // Explicit ge, kept as sent
						Le:     128, // Explicit le, kept as sent
					},
				},
			},
			wantError: false,
		},
		{
			name: "IPv6 with le=128 (max) for /48 prefix - returns raw API values",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            1004,
				Description:   "IPv6 exact match test /48",
				AddressFamily: "IPv6",
				Entries: []*megaport.MCRPrefixListEntry{
					{
						Action: "deny",
						Prefix: "2001:db8::/48",
						Ge:     48,  // Explicit ge, kept as sent
						Le:     128, // Explicit le, kept as sent
					},
				},
			},
			wantError: false,
		},
		// Range tests - values returned as-is (no normalization in fromAPI)
		{
			name: "IPv4 range with le=32 - returns raw API values",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            1005,
				Description:   "IPv4 range to max",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{
						Action: "permit",
						Prefix: "10.0.0.0/8",
						Ge:     8,  // Range from /8 to /32
						Le:     32, // Returns as-is
					},
				},
			},
			wantError: false,
		},
		{
			name: "IPv4 range with intermediate le - returns raw API values",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            1006,
				Description:   "IPv4 range",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{
						Action: "permit",
						Prefix: "10.0.0.0/24",
						Ge:     24,
						Le:     28, // Not max, should remain 28
					},
				},
			},
			wantError: false,
		},
		{
			name: "IPv6 range with le=128 - returns raw API values",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            1007,
				Description:   "IPv6 range to max",
				AddressFamily: "IPv6",
				Entries: []*megaport.MCRPrefixListEntry{
					{
						Action: "permit",
						Prefix: "2001:db8::/32",
						Ge:     32,  // Range from /32 to /128
						Le:     128, // Intentionally set to max - NOT an exact match
					},
				},
			},
			wantError: false,
		},
		// Edge case: ge == le == max (true exact match at max length)
		{
			name: "IPv4 exact match at /32 - returns raw API values",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            1008,
				Description:   "IPv4 exact /32",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{
						Action: "permit",
						Prefix: "10.0.0.1/32",
						Ge:     32,
						Le:     32, // ge == le == max, this is a true exact match
					},
				},
			},
			wantError: false,
		},
		{
			name: "IPv6 exact match at /128 - returns raw API values",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            1009,
				Description:   "IPv6 exact /128",
				AddressFamily: "IPv6",
				Entries: []*megaport.MCRPrefixListEntry{
					{
						Action: "permit",
						Prefix: "2001:db8::1/128",
						Ge:     128,
						Le:     128, // ge == le == max, this is a true exact match
					},
				},
			},
			wantError: false,
		},
		// Multiple entries - all returned as-is from API (no normalization in fromAPI)
		{
			name: "mixed entries - returns raw API values",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            1010,
				Description:   "Mixed entries test",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{
						Action: "permit",
						Prefix: "10.0.0.0/24",
						Ge:     24,
						Le:     32, // Explicit le, kept as sent
					},
					{
						Action: "deny",
						Prefix: "192.168.0.0/16",
						Ge:     16,
						Le:     24, // Returned as-is
					},
					{
						Action: "permit",
						Prefix: "172.16.0.0/12",
						Ge:     12,
						Le:     32, // Returned as-is
					},
				},
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

func TestFromAPIAbsentGeLeDecode(t *testing.T) {
	tests := []struct {
		name         string
		apiList      *megaport.MCRPrefixFilterList
		expectedGeLe []struct{ ge, le int }
	}{
		{
			name: "IPv4 exact match: both fields absent on a /24",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            1,
				Description:   "Test",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "10.0.0.0/24"},
				},
			},
			expectedGeLe: []struct{ ge, le int }{{24, 24}},
		},
		{
			name: "IPv6 exact match: both fields absent on a /32",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            2,
				Description:   "Test",
				AddressFamily: "IPv6",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "2001:db8::/32"},
				},
			},
			expectedGeLe: []struct{ ge, le int }{{32, 32}},
		},
		{
			name: "IPv4 exact match on a /31",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            3,
				Description:   "Test",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "162.43.146.92/31"},
				},
			},
			expectedGeLe: []struct{ ge, le int }{{31, 31}},
		},
		{
			name: "ge absent, le returned: ge resolves to the prefix length",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            4,
				Description:   "Test",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "10.0.0.0/24", Le: 30},
				},
			},
			expectedGeLe: []struct{ ge, le int }{{24, 30}},
		},
		{
			name: "explicit le at the IPv4 maximum is kept",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            5,
				Description:   "Test",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "10.0.0.0/24", Ge: 25, Le: 32},
				},
			},
			expectedGeLe: []struct{ ge, le int }{{25, 32}},
		},
		{
			name: "explicit le at the IPv6 maximum is kept",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            6,
				Description:   "Test",
				AddressFamily: "IPv6",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "2001:db8::/32", Ge: 48, Le: 128},
				},
			},
			expectedGeLe: []struct{ ge, le int }{{48, 128}},
		},
		{
			name: "default route: an absent ge resolves to a prefix length of zero",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            7,
				Description:   "Test",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "0.0.0.0/0", Le: 32},
				},
			},
			expectedGeLe: []struct{ ge, le int }{{0, 32}},
		},
		{
			name: "ge returned, le absent: le resolves to the family maximum",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            8,
				Description:   "Test",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "10.0.0.0/24", Ge: 26},
				},
			},
			expectedGeLe: []struct{ ge, le int }{{26, 32}},
		},
		{
			name: "mixed entries: exact matches and ranges in one list",
			apiList: &megaport.MCRPrefixFilterList{
				ID:            8,
				Description:   "Test",
				AddressFamily: "IPv4",
				Entries: []*megaport.MCRPrefixListEntry{
					{Action: "permit", Prefix: "10.0.0.0/24"},
					{Action: "deny", Prefix: "192.168.0.0/16", Le: 24},
					{Action: "permit", Prefix: "172.16.0.0/12", Ge: 16, Le: 32},
					{Action: "permit", Prefix: "203.0.113.0/24", Ge: 28},
				},
			},
			expectedGeLe: []struct{ ge, le int }{
				{24, 24},
				{16, 24},
				{16, 32},
				{28, 32},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &mcrPrefixFilterListResourceModel{}
			diags := model.fromAPI(context.Background(), tt.apiList)

			if diags.HasError() {
				t.Fatalf("fromAPI() returned unexpected error: %v", diags)
			}

			var entries []*mcrPrefixFilterListEntryResourceModel
			entriesDiags := model.Entries.ElementsAs(context.Background(), &entries, false)
			if entriesDiags.HasError() {
				t.Fatalf("Failed to extract entries: %v", entriesDiags)
			}

			if len(entries) != len(tt.expectedGeLe) {
				t.Fatalf("Expected %d entries, got %d", len(tt.expectedGeLe), len(entries))
			}

			for i, expected := range tt.expectedGeLe {
				actualGe := int(entries[i].Ge.ValueInt64())
				actualLe := int(entries[i].Le.ValueInt64())

				if actualGe != expected.ge {
					t.Errorf("Entry[%d] ge: expected %d, got %d", i, expected.ge, actualGe)
				}
				if actualLe != expected.le {
					t.Errorf("Entry[%d] le: expected %d, got %d", i, expected.le, actualLe)
				}
			}
		})
	}
}
