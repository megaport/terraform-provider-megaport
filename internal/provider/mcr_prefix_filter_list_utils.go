package provider

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

		for _, entry := range entries {
			apiEntry, convertDiags := convertEntryToAPI(entry, m.AddressFamily.ValueString())
			diags.Append(convertDiags...)
			if diags.HasError() {
				continue
			}

			apiList.Entries = append(apiList.Entries, apiEntry)
		}
	}

	return apiList, diags
}

// fromAPI converts the API response to the Terraform model
// This version does NOT apply exact match normalization - returns raw API values
// Use this for import scenarios where we don't have prior configuration to compare
func (m *mcrPrefixFilterListResourceModel) fromAPI(ctx context.Context, apiList *megaport.MCRPrefixFilterList) diag.Diagnostics {
	return m.fromAPIWithPlan(ctx, apiList, nil)
}

// fromAPIWithPlan converts the API response to the Terraform model with optional plan/state comparison
// When plannedEntries is provided, it uses the plan/state to determine if exact match normalization should occur
// When plannedEntries is nil (e.g., during import), NO normalization is applied - raw API values are returned
//
// Normalization logic: When the Megaport GUI "Exact" checkbox is used, the API stores and returns
// le=32 (IPv4) or le=128 (IPv6) instead of the exact match value. This causes Terraform to detect
// drift when users configure exact matches (ge=le). We normalize by checking if the user's config
// had an exact match and the API returned le=max, in which case we set le=ge.
func (m *mcrPrefixFilterListResourceModel) fromAPIWithPlan(ctx context.Context, apiList *megaport.MCRPrefixFilterList, plannedEntries []*mcrPrefixFilterListEntryResourceModel) diag.Diagnostics {
	diags := diag.Diagnostics{}

	m.ID = types.Int64Value(int64(apiList.ID))
	m.Description = types.StringValue(apiList.Description)
	m.AddressFamily = types.StringValue(apiList.AddressFamily)

	maxPrefixLength := 32
	if m.AddressFamily.ValueString() == "IPv6" {
		maxPrefixLength = 128
	}

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

		// Normalize exact matches ONLY when we have plan/state to compare against.
		// This fixes the "inconsistent state after apply" error when users configure exact matches.
		// During import (plannedEntries is nil), we return raw API values - the user's HCL will
		// define what they expect, and Terraform will handle any drift detection normally.
		// Search for matching planned entry by prefix (not by position) to handle
		// cases where the API returns entries in a different order than planned.
		for _, plannedEntry := range plannedEntries {
			if plannedEntry.Prefix.ValueString() == entry.Prefix {
				// Only treat as exact match when both ge and le are explicitly set
				if le == maxPrefixLength && le > ge &&
					!plannedEntry.Ge.IsNull() && !plannedEntry.Ge.IsUnknown() &&
					!plannedEntry.Le.IsNull() && !plannedEntry.Le.IsUnknown() {
					plannedGe := int(plannedEntry.Ge.ValueInt64())
					plannedLe := int(plannedEntry.Le.ValueInt64())
					if plannedGe == plannedLe {
						// Plan had exact match, but API returned le=max, normalize it
						le = ge
					}
					// If plan had le=max explicitly, don't normalize - keep the API value
				}
				break // Found matching prefix, no need to continue searching
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

// normalizeCIDR returns the canonical network address for a CIDR prefix.
// For example, "162.43.146.93/31" becomes "162.43.146.92/31" because the
// network address for a /31 is the even address. Returns the input unchanged
// if parsing fails (validators catch invalid CIDRs later).
func normalizeCIDR(prefix string) string {
	_, network, err := net.ParseCIDR(prefix)
	if err != nil {
		return prefix
	}
	return network.String()
}

// canonicalCIDRValidator validates that a CIDR prefix has no host bits set.
// If host bits are set, it returns an error with the correct network address.
type canonicalCIDRValidator struct{}

func (v canonicalCIDRValidator) Description(_ context.Context) string {
	return "CIDR prefix must not have host bits set"
}

func (v canonicalCIDRValidator) MarkdownDescription(_ context.Context) string {
	return "CIDR prefix must not have host bits set"
}

func (v canonicalCIDRValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	prefix := req.ConfigValue.ValueString()
	canonical := normalizeCIDR(prefix)
	if prefix != canonical {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid CIDR prefix",
			fmt.Sprintf("Prefix %q has host bits set. Use the network address %q instead.", prefix, canonical),
		)
	}
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
