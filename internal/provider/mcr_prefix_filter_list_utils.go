package provider

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// mcrPrefixFilterListResourceModel represents the Terraform model for the MCR prefix filter list resource
type mcrPrefixFilterListResourceModel struct {
	ID            types.Int64  `tfsdk:"id"`
	MCRID         types.String `tfsdk:"mcr_id"`
	Description   types.String `tfsdk:"description"`
	AddressFamily types.String `tfsdk:"address_family"`
	Entries       types.List   `tfsdk:"entries"`
	LastUpdated   types.String `tfsdk:"last_updated"`
}

// mcrPrefixFilterListEntryResourceModel represents a single entry in a prefix filter list
type mcrPrefixFilterListEntryResourceModel struct {
	Action types.String `tfsdk:"action"`
	Prefix types.String `tfsdk:"prefix"`
	Ge     types.Int64  `tfsdk:"ge"`
	Le     types.Int64  `tfsdk:"le"`
}

// planToAPI converts the Terraform plan to the API model
func (m *mcrPrefixFilterListResourceModel) planToAPI(ctx context.Context) (*megaport.MCRPrefixFilterList, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	apiList := &megaport.MCRPrefixFilterList{
		Description:   m.Description.ValueString(),
		AddressFamily: m.AddressFamily.ValueString(),
	}

	if !m.Entries.IsNull() && !m.Entries.IsUnknown() {
		entries := []*mcrPrefixFilterListEntryResourceModel{}
		entryDiags := m.Entries.ElementsAs(ctx, &entries, false)
		diags.Append(entryDiags...)
		if diags.HasError() {
			return nil, diags
		}

		for i, entry := range entries {
			apiEntry, convertDiags := convertEntryToAPI(entry, m.AddressFamily.ValueString())
			diags.Append(convertDiags...)
			if diags.HasError() {
				continue
			}

			// Validate entry
			if validationDiags := validatePrefixListEntry(entry, m.AddressFamily.ValueString(), i); validationDiags.HasError() {
				diags.Append(validationDiags...)
				continue
			}

			apiList.Entries = append(apiList.Entries, apiEntry)
		}
	}

	return apiList, diags
}

// fromAPI converts the API response to the Terraform model
func (m *mcrPrefixFilterListResourceModel) fromAPI(ctx context.Context, apiList *megaport.MCRPrefixFilterList) diag.Diagnostics {
	diags := diag.Diagnostics{}

	m.ID = types.Int64Value(int64(apiList.ID))
	m.Description = types.StringValue(apiList.Description)
	m.AddressFamily = types.StringValue(apiList.AddressFamily)

	// Convert entries
	entriesList := []types.Object{}
	for _, entry := range apiList.Entries {
		// Only handle cases where API actually returns 0 for ge/le, not when values differ
		ge, le := entry.Ge, entry.Le
		if entry.Ge == 0 || entry.Le == 0 {
			calculatedGe, calculatedLe, calcDiags := calculateGeLeFromPrefix(entry.Prefix, m.AddressFamily.ValueString())
			diags.Append(calcDiags...)
			if calcDiags.HasError() {
				continue
			}

			if entry.Ge == 0 {
				ge = calculatedGe
			}
			if entry.Le == 0 {
				le = calculatedLe
			}
		}

		entryModel := &mcrPrefixFilterListEntryResourceModel{
			Action: types.StringValue(entry.Action),
			Prefix: types.StringValue(entry.Prefix),
			Ge:     types.Int64Value(int64(ge)),
			Le:     types.Int64Value(int64(le)),
		}

		entryObj, entryDiags := types.ObjectValueFrom(ctx, mcrPrefixFilterListEntryAttributes, entryModel)
		diags.Append(entryDiags...)
		if !diags.HasError() {
			entriesList = append(entriesList, entryObj)
		}
	}

	entries, entriesDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListEntryAttributes), entriesList)
	diags.Append(entriesDiags...)
	m.Entries = entries

	return diags
}

// convertEntryToAPI converts a single entry from Terraform model to API model
func convertEntryToAPI(entry *mcrPrefixFilterListEntryResourceModel, addressFamily string) (*megaport.MCRPrefixListEntry, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	apiEntry := &megaport.MCRPrefixListEntry{
		Action: entry.Action.ValueString(),
		Prefix: entry.Prefix.ValueString(),
	}

	// Handle ge/le values with appropriate defaults
	ge, le, convertDiags := calculateGeLe(entry, addressFamily)
	diags.Append(convertDiags...)
	if diags.HasError() {
		return nil, diags
	}

	apiEntry.Ge = ge
	apiEntry.Le = le

	return apiEntry, diags
}

// calculateGeLe calculates appropriate ge/le values based on the prefix and address family
func calculateGeLe(entry *mcrPrefixFilterListEntryResourceModel, addressFamily string) (int, int, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	// Parse the prefix to get the network length
	_, network, err := net.ParseCIDR(entry.Prefix.ValueString())
	if err != nil {
		diags.AddError(
			"Invalid prefix format",
			fmt.Sprintf("Error parsing prefix %s: %s", entry.Prefix.ValueString(), err.Error()),
		)
		return 0, 0, diags
	}

	prefixLength, _ := network.Mask.Size()
	maxLength := 32
	if addressFamily == "IPv6" {
		maxLength = 128
	}

	var ge, le int

	if !entry.Ge.IsNull() {
		ge = int(entry.Ge.ValueInt64())
	} else {
		// Default ge to the prefix length
		ge = prefixLength
	}

	if !entry.Le.IsNull() {
		le = int(entry.Le.ValueInt64())
	} else {
		// Default le to maximum length for address family
		le = maxLength
	}

	// Validate ge/le relationship
	if ge > le {
		diags.AddError(
			"Invalid ge/le values",
			fmt.Sprintf("ge (%d) cannot be greater than le (%d)", ge, le),
		)
		return 0, 0, diags
	}

	if ge < prefixLength {
		diags.AddError(
			"Invalid ge value",
			fmt.Sprintf("ge (%d) cannot be less than the prefix length (%d)", ge, prefixLength),
		)
		return 0, 0, diags
	}

	if le > maxLength {
		diags.AddError(
			"Invalid le value",
			fmt.Sprintf("le (%d) cannot be greater than %d for %s", le, maxLength, addressFamily),
		)
		return 0, 0, diags
	}

	return ge, le, diags
}

// validatePrefixListEntry validates a single prefix list entry
func validatePrefixListEntry(entry *mcrPrefixFilterListEntryResourceModel, addressFamily string, index int) diag.Diagnostics {
	diags := diag.Diagnostics{}

	// Validate prefix format
	_, network, err := net.ParseCIDR(entry.Prefix.ValueString())
	if err != nil {
		diags.AddError(
			fmt.Sprintf("Invalid prefix in entry %d", index),
			fmt.Sprintf("Error parsing prefix %s: %s", entry.Prefix.ValueString(), err.Error()),
		)
		return diags
	}

	// Validate address family matches prefix type
	isIPv4 := network.IP.To4() != nil
	expectedIPv4 := addressFamily == "IPv4"

	if isIPv4 != expectedIPv4 {
		diags.AddError(
			fmt.Sprintf("Address family mismatch in entry %d", index),
			fmt.Sprintf("Prefix %s is %s but address family is set to %s",
				entry.Prefix.ValueString(),
				map[bool]string{true: "IPv4", false: "IPv6"}[isIPv4],
				addressFamily),
		)
	}

	// Validate ge/le ranges based on address family
	maxLength := 32
	if addressFamily == "IPv6" {
		maxLength = 128
	}

	if !entry.Ge.IsNull() {
		ge := entry.Ge.ValueInt64()
		if ge < 0 || ge > int64(maxLength) {
			diags.AddError(
				fmt.Sprintf("Invalid ge value in entry %d", index),
				fmt.Sprintf("ge must be between 0 and %d for %s", maxLength, addressFamily),
			)
		}
	}

	if !entry.Le.IsNull() {
		le := entry.Le.ValueInt64()
		if le < 0 || le > int64(maxLength) {
			diags.AddError(
				fmt.Sprintf("Invalid le value in entry %d", index),
				fmt.Sprintf("le must be between 0 and %d for %s", maxLength, addressFamily),
			)
		}
	}

	return diags
}

// generateImportID creates an import ID in the format "mcr_uid:prefix_list_id"
func generateImportID(mcrUID string, prefixListID int64) string {
	return fmt.Sprintf("%s:%d", mcrUID, prefixListID)
}

// calculateGeLeFromPrefix calculates appropriate ge/le values for a prefix when API returns 0 values
func calculateGeLeFromPrefix(prefix string, addressFamily string) (int, int, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	// Parse the prefix to get the network length
	_, network, err := net.ParseCIDR(prefix)
	if err != nil {
		diags.AddError(
			"Invalid prefix format",
			fmt.Sprintf("Error parsing prefix %s: %s", prefix, err.Error()),
		)
		return 0, 0, diags
	}

	prefixLength, _ := network.Mask.Size()
	maxLength := 32
	if addressFamily == "IPv6" {
		maxLength = 128
	}

	// Default ge to the prefix length and le to maximum length
	return prefixLength, maxLength, diags
}

// parseImportID parses an import ID and returns the MCR UID and prefix list ID
func parseImportID(importID string) (string, int64, error) {
	parts := make([]string, 0, 2)

	// Find the colon separator
	colonIndex := -1
	for i, char := range importID {
		if char == ':' {
			colonIndex = i
			break
		}
	}

	if colonIndex == -1 {
		return "", 0, fmt.Errorf("invalid import ID format, expected 'mcr_uid:prefix_list_id', got '%s'", importID)
	}

	parts = append(parts, importID[:colonIndex], importID[colonIndex+1:])

	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid import ID format, expected 'mcr_uid:prefix_list_id', got '%s'", importID)
	}

	mcrUID := parts[0]
	prefixListIDStr := parts[1]

	if mcrUID == "" || prefixListIDStr == "" {
		return "", 0, fmt.Errorf("invalid import ID format, MCR UID and prefix list ID cannot be empty")
	}

	prefixListID, err := strconv.ParseInt(prefixListIDStr, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid prefix list ID '%s': %w", prefixListIDStr, err)
	}

	if prefixListID <= 0 {
		return "", 0, fmt.Errorf("invalid prefix list ID '%s': must be a positive integer", prefixListIDStr)
	}

	return mcrUID, prefixListID, nil
}
