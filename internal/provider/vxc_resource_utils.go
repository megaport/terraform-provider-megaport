package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	megaport "github.com/megaport/megaportgo"
)

// resolvePrefixListID looks up a prefix filter list by description on the
// supplied slice (typically returned by vrouterPrefixFilterListsForEndpoint).
// It returns an error diagnostic if zero or more than one list matches —
// matching by description is convenient but can otherwise silently drop a
// whitelist/blacklist on typos or when descriptions are reused, leaving the
// BGP session unfiltered without any signal to the user.
func resolvePrefixListID(prefixFilterList []*megaport.PrefixFilterList, description, fieldName string) (int, diag.Diagnostics) {
	var diags diag.Diagnostics
	matches := make([]*megaport.PrefixFilterList, 0, 1)
	for _, pfl := range prefixFilterList {
		if pfl.Description == description {
			matches = append(matches, pfl)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0].Id, diags
	case 0:
		diags.AddError(
			"Prefix filter list not found",
			fmt.Sprintf("%s references prefix filter list %q, but no list with that description exists on the attached MCR or NAT Gateway. Prefix lists are resolved by description.", fieldName, description),
		)
		return 0, diags
	default:
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = strconv.Itoa(m.Id)
		}
		diags.AddError(
			"Ambiguous prefix filter list description",
			fmt.Sprintf("%s references prefix filter list %q, but %d lists on the attached MCR or NAT Gateway share that description (IDs: %s). Prefix list descriptions must be unique for description-based lookup.", fieldName, description, len(matches), strings.Join(ids, ", ")),
		)
		return 0, diags
	}
}

// fromAPIVXC updates the resource model from API response data.
// The optional plan parameter allows preserving user-only fields (like requested_product_uid,
// ordered_vlan, and partner configs) that are not returned by the API. This is particularly
// important after import or during updates where the plan contains user configuration that
// would otherwise be lost.
// The client parameter is used to look up MCR prefix filter lists for BGP drift detection.
func (orm *vxcResourceModel) fromAPIVXC(ctx context.Context, v *megaport.VXC, tags map[string]string, plan *vxcResourceModel, client *megaport.Client) diag.Diagnostics {
	apiDiags := diag.Diagnostics{}

	orm.UID = types.StringValue(v.UID)
	orm.ID = types.Int64Value(int64(v.ID))
	orm.Name = types.StringValue(v.Name)
	orm.ServiceID = types.Int64Value(int64(v.ServiceID))
	orm.Type = types.StringValue(v.Type)
	orm.RateLimit = types.Int64Value(int64(v.RateLimit))
	orm.DistanceBand = types.StringValue(v.DistanceBand)
	orm.ProvisioningStatus = types.StringValue(v.ProvisioningStatus)
	orm.SecondaryName = types.StringValue(v.SecondaryName)
	orm.UsageAlgorithm = types.StringValue(v.UsageAlgorithm)
	orm.CreatedBy = types.StringValue(v.CreatedBy)
	orm.ContractTermMonths = types.Int64Value(int64(v.ContractTermMonths))
	orm.CompanyUID = types.StringValue(v.CompanyUID)
	orm.CompanyName = types.StringValue(v.CompanyName)
	orm.Shutdown = types.BoolValue(v.Shutdown)
	orm.CostCentre = types.StringValue(v.CostCentre)
	orm.Locked = types.BoolValue(v.Locked)
	orm.AdminLocked = types.BoolValue(v.AdminLocked)
	orm.Cancelable = types.BoolValue(v.Cancelable)

	if v.CreateDate != nil {
		orm.CreateDate = types.StringValue(v.CreateDate.Format(time.RFC850))
	} else {
		orm.CreateDate = types.StringNull()
	}
	if v.LiveDate != nil {
		orm.LiveDate = types.StringValue(v.LiveDate.Format(time.RFC850))
	} else {
		orm.LiveDate = types.StringNull()
	}
	if v.ContractStartDate != nil {
		orm.ContractStartDate = types.StringValue(v.ContractStartDate.Format(time.RFC850))
	} else {
		orm.ContractStartDate = types.StringNull()
	}
	if v.ContractEndDate != nil {
		orm.ContractEndDate = types.StringValue(v.ContractEndDate.Format(time.RFC850))
	} else {
		orm.ContractEndDate = types.StringNull()
	}
	var aEndOrderedVLAN, bEndOrderedVLAN *int64
	var aEndInnerVLAN, bEndInnerVLAN *int64
	var aEndVnicIndex, bEndVnicIndex *int64
	var aEndRequestedProductUID, bEndRequestedProductUID string

	// First, try to get values from existing state
	if !orm.AEndConfiguration.IsNull() {
		existingAEnd := &vxcEndConfigurationModel{}
		aEndDiags := orm.AEndConfiguration.As(ctx, existingAEnd, basetypes.ObjectAsOptions{})
		apiDiags = append(apiDiags, aEndDiags...)
		aEndRequestedProductUID = existingAEnd.RequestedProductUID.ValueString()
		if !existingAEnd.OrderedVLAN.IsNull() && !existingAEnd.OrderedVLAN.IsUnknown() {
			vlan := existingAEnd.OrderedVLAN.ValueInt64()
			aEndOrderedVLAN = &vlan
		}
		if !existingAEnd.InnerVLAN.IsNull() && !existingAEnd.InnerVLAN.IsUnknown() {
			vlan := existingAEnd.InnerVLAN.ValueInt64()
			aEndInnerVLAN = &vlan
		}
		// During Read (plan == nil), preserve vnic_index from state because the
		// API does not reliably return it immediately after changes.
		if plan == nil && !existingAEnd.NetworkInterfaceIndex.IsNull() && !existingAEnd.NetworkInterfaceIndex.IsUnknown() {
			idx := existingAEnd.NetworkInterfaceIndex.ValueInt64()
			aEndVnicIndex = &idx
		}
	}

	// If plan is provided and state values are empty, use plan values.
	// This handles the import case where state is initially null, and
	// the Create case where the API may not yet reflect the vnic_index.
	if plan != nil && !plan.AEndConfiguration.IsNull() {
		planAEnd := &vxcEndConfigurationModel{}
		planDiags := plan.AEndConfiguration.As(ctx, planAEnd, basetypes.ObjectAsOptions{})
		apiDiags = append(apiDiags, planDiags...)

		if aEndRequestedProductUID == "" && !planAEnd.RequestedProductUID.IsNull() {
			aEndRequestedProductUID = planAEnd.RequestedProductUID.ValueString()
		}
		if aEndOrderedVLAN == nil && !planAEnd.OrderedVLAN.IsNull() && !planAEnd.OrderedVLAN.IsUnknown() {
			vlan := planAEnd.OrderedVLAN.ValueInt64()
			aEndOrderedVLAN = &vlan
		}
		if aEndInnerVLAN == nil && !planAEnd.InnerVLAN.IsNull() && !planAEnd.InnerVLAN.IsUnknown() {
			vlan := planAEnd.InnerVLAN.ValueInt64()
			aEndInnerVLAN = &vlan
		}
		if aEndVnicIndex == nil && !planAEnd.NetworkInterfaceIndex.IsNull() && !planAEnd.NetworkInterfaceIndex.IsUnknown() {
			idx := planAEnd.NetworkInterfaceIndex.ValueInt64()
			aEndVnicIndex = &idx
		}
	}

	aEndModel := &vxcEndConfigurationModel{
		OwnerUID:              types.StringValue(v.AEndConfiguration.OwnerUID),
		RequestedProductUID:   types.StringValue(aEndRequestedProductUID),
		CurrentProductUID:     types.StringValue(v.AEndConfiguration.UID),
		Name:                  types.StringValue(v.AEndConfiguration.Name),
		LocationID:            types.Int64Value(int64(v.AEndConfiguration.LocationID)),
		Location:              types.StringValue(v.AEndConfiguration.Location),
		NetworkInterfaceIndex: types.Int64Value(int64(v.AEndConfiguration.NetworkInterfaceIndex)),
		SecondaryName:         types.StringValue(v.AEndConfiguration.SecondaryName),
	}
	if aEndVnicIndex != nil {
		aEndModel.NetworkInterfaceIndex = types.Int64Value(*aEndVnicIndex)
	}
	if aEndOrderedVLAN != nil {
		aEndModel.OrderedVLAN = types.Int64Value(*aEndOrderedVLAN)
	}
	if v.AEndConfiguration.InnerVLAN == 0 {
		// Check if existing inner VLAN is null or -1
		if aEndInnerVLAN != nil && *aEndInnerVLAN == -1 {
			// Keep it as -1 (untagged)
			aEndModel.InnerVLAN = types.Int64Value(*aEndInnerVLAN)
		} else {
			// API didn't return a value, keep as null
			aEndModel.InnerVLAN = types.Int64PointerValue(nil)
		}
	} else {
		// API returned a non-zero value - use it
		aEndModel.InnerVLAN = types.Int64Value(int64(v.AEndConfiguration.InnerVLAN))
	}

	if v.AEndConfiguration.VLAN == 0 {
		aEndModel.VLAN = types.Int64PointerValue(nil)
	} else {
		aEndModel.VLAN = types.Int64Value(int64(v.AEndConfiguration.VLAN))
	}
	aEnd, aEndDiags := types.ObjectValueFrom(ctx, vxcEndConfigurationAttrs, aEndModel)
	apiDiags = append(apiDiags, aEndDiags...)
	orm.AEndConfiguration = aEnd

	// First, try to get B-End values from existing state
	if !orm.BEndConfiguration.IsNull() {
		existingBEnd := &vxcEndConfigurationModel{}
		bEndDiags := orm.BEndConfiguration.As(ctx, existingBEnd, basetypes.ObjectAsOptions{})
		apiDiags = append(apiDiags, bEndDiags...)
		if !existingBEnd.OrderedVLAN.IsNull() && !existingBEnd.OrderedVLAN.IsUnknown() {
			vlan := existingBEnd.OrderedVLAN.ValueInt64()
			bEndOrderedVLAN = &vlan
		}
		if !existingBEnd.InnerVLAN.IsNull() && !existingBEnd.InnerVLAN.IsUnknown() {
			vlan := existingBEnd.InnerVLAN.ValueInt64()
			bEndInnerVLAN = &vlan
		}
		// During Read (plan == nil), preserve vnic_index from state because the
		// API does not reliably return it immediately after changes.
		if plan == nil && !existingBEnd.NetworkInterfaceIndex.IsNull() && !existingBEnd.NetworkInterfaceIndex.IsUnknown() {
			idx := existingBEnd.NetworkInterfaceIndex.ValueInt64()
			bEndVnicIndex = &idx
		}
		bEndRequestedProductUID = existingBEnd.RequestedProductUID.ValueString()
	}

	// If plan is provided and state values are empty, use plan values for B-End.
	if plan != nil && !plan.BEndConfiguration.IsNull() {
		planBEnd := &vxcEndConfigurationModel{}
		planDiags := plan.BEndConfiguration.As(ctx, planBEnd, basetypes.ObjectAsOptions{})
		apiDiags = append(apiDiags, planDiags...)

		if bEndRequestedProductUID == "" && !planBEnd.RequestedProductUID.IsNull() {
			bEndRequestedProductUID = planBEnd.RequestedProductUID.ValueString()
		}
		if bEndOrderedVLAN == nil && !planBEnd.OrderedVLAN.IsNull() && !planBEnd.OrderedVLAN.IsUnknown() {
			vlan := planBEnd.OrderedVLAN.ValueInt64()
			bEndOrderedVLAN = &vlan
		}
		if bEndInnerVLAN == nil && !planBEnd.InnerVLAN.IsNull() && !planBEnd.InnerVLAN.IsUnknown() {
			vlan := planBEnd.InnerVLAN.ValueInt64()
			bEndInnerVLAN = &vlan
		}
		if bEndVnicIndex == nil && !planBEnd.NetworkInterfaceIndex.IsNull() && !planBEnd.NetworkInterfaceIndex.IsUnknown() {
			idx := planBEnd.NetworkInterfaceIndex.ValueInt64()
			bEndVnicIndex = &idx
		}
	}

	bEndModel := &vxcEndConfigurationModel{
		OwnerUID:              types.StringValue(v.BEndConfiguration.OwnerUID),
		RequestedProductUID:   types.StringValue(bEndRequestedProductUID),
		CurrentProductUID:     types.StringValue(v.BEndConfiguration.UID),
		Name:                  types.StringValue(v.BEndConfiguration.Name),
		LocationID:            types.Int64Value(int64(v.BEndConfiguration.LocationID)),
		Location:              types.StringValue(v.BEndConfiguration.Location),
		NetworkInterfaceIndex: types.Int64Value(int64(v.BEndConfiguration.NetworkInterfaceIndex)),
		SecondaryName:         types.StringValue(v.BEndConfiguration.SecondaryName),
	}
	if bEndVnicIndex != nil {
		bEndModel.NetworkInterfaceIndex = types.Int64Value(*bEndVnicIndex)
	}
	if bEndOrderedVLAN != nil {
		bEndModel.OrderedVLAN = types.Int64Value(*bEndOrderedVLAN)
	}
	if v.BEndConfiguration.InnerVLAN == 0 {
		// Check if existing inner VLAN is null or -1
		if bEndInnerVLAN != nil && *bEndInnerVLAN == -1 {
			// Keep it as -1 (untagged)
			bEndModel.InnerVLAN = types.Int64Value(*bEndInnerVLAN)
		} else {
			// Keep it as null, which means un-assigned.
			bEndModel.InnerVLAN = types.Int64PointerValue(nil)
		}
	} else {
		bEndModel.InnerVLAN = types.Int64Value(int64(v.BEndConfiguration.InnerVLAN))
	}
	if v.BEndConfiguration.VLAN == 0 {
		bEndModel.VLAN = types.Int64PointerValue(nil)
	} else {
		bEndModel.VLAN = types.Int64Value(int64(v.BEndConfiguration.VLAN))
	}

	bEnd, bEndDiags := types.ObjectValueFrom(ctx, vxcEndConfigurationAttrs, bEndModel)
	apiDiags = append(apiDiags, bEndDiags...)
	orm.BEndConfiguration = bEnd

	if v.Resources != nil && v.Resources.CSPConnection != nil {
		cspConnections := []types.Object{}
		for _, c := range v.Resources.CSPConnection.CSPConnection {
			cspConnection, cspDiags := fromAPICSPConnection(ctx, c)
			apiDiags = append(apiDiags, cspDiags...)
			cspConnections = append(cspConnections, cspConnection)
		}
		cspConnectionsList, cspConnectionDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(cspConnectionFullAttrs), cspConnections)
		apiDiags = append(apiDiags, cspConnectionDiags...)
		orm.CSPConnections = cspConnectionsList
	} else {
		cspConnectionsList := types.ListNull(types.ObjectType{}.WithAttributeTypes(cspConnectionFullAttrs))
		orm.CSPConnections = cspConnectionsList
	}

	if v.AttributeTags != nil {
		attributeTags, attributeDiags := types.MapValueFrom(ctx, types.StringType, v.AttributeTags)
		apiDiags = append(apiDiags, attributeDiags...)
		orm.AttributeTags = attributeTags
	} else {
		orm.AttributeTags = types.MapNull(types.StringType)
	}

	if len(tags) > 0 {
		resourceTags, tagDiags := types.MapValueFrom(ctx, types.StringType, tags)
		apiDiags = append(apiDiags, tagDiags...)
		orm.ResourceTags = resourceTags
	} else {
		orm.ResourceTags = types.MapNull(types.StringType)
	}

	apiDiags = append(apiDiags, orm.reconcilePartnerConfigs(ctx, v, plan, client)...)

	return apiDiags
}

// reconcilePartnerConfigs takes partner configs from the plan during Create and
// Update. During Read it merges API values into the vrouter fields the user set,
// so terraform plan detects drift. Fields the user left unset stay null.
func (orm *vxcResourceModel) reconcilePartnerConfigs(ctx context.Context, v *megaport.VXC, plan *vxcResourceModel, client *megaport.Client) diag.Diagnostics {
	diags := diag.Diagnostics{}

	// Partner configs are user-only values the API does not return, so the plan
	// wins on Create and Update.
	if plan != nil {
		if !plan.AEndPartnerConfig.IsNull() {
			orm.AEndPartnerConfig = plan.AEndPartnerConfig
		}
		if !plan.BEndPartnerConfig.IsNull() {
			orm.BEndPartnerConfig = plan.BEndPartnerConfig
		}
		return diags
	}

	var vrouterCSPConns []megaport.CSPConnectionVirtualRouter
	if v.Resources != nil && v.Resources.CSPConnection != nil {
		for _, c := range v.Resources.CSPConnection.CSPConnection {
			if vr, ok := c.(megaport.CSPConnectionVirtualRouter); ok {
				vrouterCSPConns = append(vrouterCSPConns, vr)
			}
		}
	}

	// Only vrouter merges from the API. The deprecated "a-end" type has a
	// narrower schema that does not map to the shared interface model, and
	// every other partner type keeps state unchanged.
	ends := []struct {
		config       *basetypes.ObjectValue
		resourceName string
		endpointUID  string
	}{
		{&orm.AEndPartnerConfig, "a_csp_connection", v.AEndConfiguration.UID},
		{&orm.BEndPartnerConfig, "b_csp_connection", v.BEndConfiguration.UID},
	}
	for _, end := range ends {
		if end.config.IsNull() || getPartnerType(ctx, *end.config) != "vrouter" {
			continue
		}
		idx := pickVrouterCSPConn(vrouterCSPConns, end.resourceName)
		if idx < 0 {
			continue
		}
		merged, mergeDiags := mergeVrouterPartnerConfigFromAPI(ctx, vrouterCSPConns[idx], *end.config, end.endpointUID, client)
		diags.Append(mergeDiags...)
		if !merged.IsNull() {
			*end.config = merged
		}
	}

	return diags
}

// pickVrouterCSPConn returns the index of the CSP connection the API labels with
// resourceName ("a_csp_connection" or "b_csp_connection"), or -1 when there is
// none.
func pickVrouterCSPConn(conns []megaport.CSPConnectionVirtualRouter, resourceName string) int {
	for i := range conns {
		if conns[i].ResourceName == resourceName {
			return i
		}
	}
	return -1
}

// getPartnerType extracts the "partner" field from a partner config object.
func getPartnerType(ctx context.Context, partnerConfig basetypes.ObjectValue) string {
	partnerConfigModel := &vxcPartnerConfigurationModel{}
	diags := partnerConfig.As(ctx, partnerConfigModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return ""
	}
	return partnerConfigModel.Partner.ValueString()
}

// mergeVrouterPartnerConfigFromAPI writes the API's CSP VirtualRouter values over
// the vrouter fields the user set. A field left null in state stays null, because
// a concrete value in a field the config cannot produce is perpetual drift.
func mergeVrouterPartnerConfigFromAPI(
	ctx context.Context,
	vrConn megaport.CSPConnectionVirtualRouter,
	existingPartnerConfig basetypes.ObjectValue,
	endpointUID string,
	client *megaport.Client,
) (basetypes.ObjectValue, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	if len(vrConn.Interfaces) == 0 {
		return existingPartnerConfig, diags
	}

	// Extract existing interfaces from state
	configModel := &vxcPartnerConfigurationModel{}
	cfgDiags := existingPartnerConfig.As(ctx, configModel, basetypes.ObjectAsOptions{})
	if cfgDiags.HasError() {
		diags.Append(cfgDiags...)
		return existingPartnerConfig, diags
	}

	var interfacesList types.List
	if !configModel.VrouterPartnerConfig.IsNull() {
		vrouterModel := &vxcPartnerConfigVrouterModel{}
		vrDiags := configModel.VrouterPartnerConfig.As(ctx, vrouterModel, basetypes.ObjectAsOptions{})
		if vrDiags.HasError() {
			diags.Append(vrDiags...)
			return existingPartnerConfig, diags
		}
		interfacesList = vrouterModel.Interfaces
	} else {
		return existingPartnerConfig, diags
	}

	if interfacesList.IsNull() || interfacesList.IsUnknown() {
		return existingPartnerConfig, diags
	}

	existingIfaces := []*vxcPartnerConfigInterfaceModel{}
	ifDiags := interfacesList.ElementsAs(ctx, &existingIfaces, false)
	if ifDiags.HasError() {
		diags.Append(ifDiags...)
		return existingPartnerConfig, diags
	}

	// Skip the round-trip when no BGP connection references a prefix filter list.
	var prefixFilterLists []*megaport.PrefixFilterList
	if client != nil && endpointUID != "" && anyPrefixFilterListsConfigured(ctx, existingIfaces) {
		var err error
		prefixFilterLists, err = vrouterPrefixFilterListsForEndpoint(ctx, client, endpointUID)
		if err != nil {
			diags.AddWarning(
				"Could not refresh BGP prefix filter list names",
				fmt.Sprintf("Prefix filter list lookup for %s failed: %s. Prefix filter list names in state may be stale.", endpointUID, err.Error()),
			)
			// Keep merging the other fields with no resolved names.
			prefixFilterLists = nil
		}
	}
	pflMap := make(map[int]string)
	for _, pfl := range prefixFilterLists {
		pflMap[pfl.Id] = pfl.Description
	}

	// Merge API data into existing interfaces
	for ifaceIdx := 0; ifaceIdx < len(existingIfaces) && ifaceIdx < len(vrConn.Interfaces); ifaceIdx++ {
		existing := existingIfaces[ifaceIdx]
		apiIface := vrConn.Interfaces[ifaceIdx]

		// ip_addresses, ip_routes and nat_ip_addresses are ordered lists that the
		// API returns in its own order, so rewriting state from them reorders the
		// list and the plan never converges. The API does not echo interface bfd.

		// Merge BGP connections. Match each state connection to its API
		// counterpart by peer IP address (a stable per-session key), falling
		// back to positional order for any that don't match. Matching by
		// position alone would attach API values to the wrong peer if the API
		// returns connections in a different order than state.
		if !existing.BgpConnections.IsNull() && len(apiIface.BGPConnections) > 0 {
			existingBgps := []*bgpConnectionConfigModel{}
			bgpDiags := existing.BgpConnections.ElementsAs(ctx, &existingBgps, false)
			diags.Append(bgpDiags...)

			apiIdxByPeerIP := make(map[string]int, len(apiIface.BGPConnections))
			for i := range apiIface.BGPConnections {
				if ip := apiIface.BGPConnections[i].PeerIpAddress; ip != "" {
					if _, dup := apiIdxByPeerIP[ip]; !dup {
						apiIdxByPeerIP[ip] = i
					}
				}
			}
			apiUsed := make([]bool, len(apiIface.BGPConnections))
			matchedIdx := make([]int, len(existingBgps))

			// First pass: resolve all peer-IP matches before any positional
			// fallback claims a slot, so an entry without a peer IP can't
			// steal the API slot that a later entry's exact match needs.
			for bgpIdx, existingBgp := range existingBgps {
				matchedIdx[bgpIdx] = -1
				if ip := existingBgp.PeerIPAddress.ValueString(); ip != "" {
					if idx, ok := apiIdxByPeerIP[ip]; ok && !apiUsed[idx] {
						matchedIdx[bgpIdx] = idx
						apiUsed[idx] = true
					}
				}
			}
			// Second pass: fill remaining unmatched entries positionally.
			for bgpIdx := range existingBgps {
				if matchedIdx[bgpIdx] == -1 && bgpIdx < len(apiIface.BGPConnections) && !apiUsed[bgpIdx] {
					matchedIdx[bgpIdx] = bgpIdx
					apiUsed[bgpIdx] = true
				}
			}

			for bgpIdx, existingBgp := range existingBgps {
				apiIdx := matchedIdx[bgpIdx]
				if apiIdx == -1 {
					continue // no API counterpart; preserve state
				}
				apiBgp := apiIface.BGPConnections[apiIdx]

				// Update a field only when state has it and the API echoed a
				// real value: megalith types these nullable, so an omitted key
				// decodes to the Go zero value. shutdown and bfd_enabled are
				// plain bools, so omitted and false are indistinguishable and
				// state alone gates them.
				if !existingBgp.PeerAsn.IsNull() && apiBgp.PeerAsn > 0 {
					existingBgp.PeerAsn = types.Int64Value(int64(apiBgp.PeerAsn))
				}
				if !existingBgp.LocalIPAddress.IsNull() && apiBgp.LocalIpAddress != "" {
					existingBgp.LocalIPAddress = types.StringValue(apiBgp.LocalIpAddress)
				}
				if !existingBgp.PeerIPAddress.IsNull() && apiBgp.PeerIpAddress != "" {
					existingBgp.PeerIPAddress = types.StringValue(apiBgp.PeerIpAddress)
				}
				if !existingBgp.Shutdown.IsNull() {
					existingBgp.Shutdown = types.BoolValue(apiBgp.Shutdown)
				}
				if !existingBgp.BfdEnabled.IsNull() {
					existingBgp.BfdEnabled = types.BoolValue(apiBgp.BfdEnabled)
				}
				if !existingBgp.LocalAsn.IsNull() && apiBgp.LocalAsn != nil {
					existingBgp.LocalAsn = types.Int64Value(int64(*apiBgp.LocalAsn))
				}
				if !existingBgp.AsOverride.IsNull() && apiBgp.AsOverride != nil {
					existingBgp.AsOverride = types.BoolValue(*apiBgp.AsOverride)
				}
				if !existingBgp.PeerType.IsNull() && apiBgp.PeerType != "" {
					existingBgp.PeerType = types.StringValue(apiBgp.PeerType)
				}
				if !existingBgp.Description.IsNull() && apiBgp.Description != "" {
					existingBgp.Description = types.StringValue(apiBgp.Description)
				}
				if !existingBgp.MedIn.IsNull() && apiBgp.MedIn > 0 {
					existingBgp.MedIn = types.Int64Value(int64(apiBgp.MedIn))
				}
				if !existingBgp.MedOut.IsNull() && apiBgp.MedOut > 0 {
					existingBgp.MedOut = types.Int64Value(int64(apiBgp.MedOut))
				}
				if !existingBgp.ExportPolicy.IsNull() && apiBgp.ExportPolicy != "" {
					existingBgp.ExportPolicy = types.StringValue(apiBgp.ExportPolicy)
				}
				if !existingBgp.AsPathPrependCount.IsNull() && apiBgp.AsPathPrependCount > 0 {
					existingBgp.AsPathPrependCount = types.Int64Value(int64(apiBgp.AsPathPrependCount))
				}
				// The API does echo password, in plain text. State still wins:
				// the field is Sensitive but not write-only, so merging the API
				// value would persist a live MD5 key in the state file.

				// The CSP-connection read does not echo these two at all, so an
				// empty API value means unknown, not removed.
				if !existingBgp.PermitExportTo.IsNull() && len(apiBgp.PermitExportTo) > 0 {
					permitList, permitDiags := types.ListValueFrom(ctx, types.StringType, apiBgp.PermitExportTo)
					diags.Append(permitDiags...)
					existingBgp.PermitExportTo = permitList
				}
				if !existingBgp.DenyExportTo.IsNull() && len(apiBgp.DenyExportTo) > 0 {
					denyList, denyDiags := types.ListValueFrom(ctx, types.StringType, apiBgp.DenyExportTo)
					diags.Append(denyDiags...)
					existingBgp.DenyExportTo = denyList
				}

				if !existingBgp.ImportWhitelist.IsNull() {
					existingBgp.ImportWhitelist = prefixFilterIDToName(apiBgp.ImportWhitelist, pflMap, existingBgp.ImportWhitelist)
				}
				if !existingBgp.ImportBlacklist.IsNull() {
					existingBgp.ImportBlacklist = prefixFilterIDToName(apiBgp.ImportBlacklist, pflMap, existingBgp.ImportBlacklist)
				}
				if !existingBgp.ExportWhitelist.IsNull() {
					existingBgp.ExportWhitelist = prefixFilterIDToName(apiBgp.ExportWhitelist, pflMap, existingBgp.ExportWhitelist)
				}
				if !existingBgp.ExportBlacklist.IsNull() {
					existingBgp.ExportBlacklist = prefixFilterIDToName(apiBgp.ExportBlacklist, pflMap, existingBgp.ExportBlacklist)
				}
			}

			bgpValues := make([]bgpConnectionConfigModel, len(existingBgps))
			for i, b := range existingBgps {
				bgpValues[i] = *b
			}
			bgpList, bgpListDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(bgpVrouterConnectionConfig), bgpValues)
			diags.Append(bgpListDiags...)
			existing.BgpConnections = bgpList
		}
	}

	// Re-serialize the updated interfaces back into the vrouter partner config
	ifaceValues := make([]vxcPartnerConfigInterfaceModel, len(existingIfaces))
	for i, iface := range existingIfaces {
		ifaceValues[i] = *iface
	}

	vrouterModel := vxcPartnerConfigVrouterModel{}
	ifaceList, ifaceDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(vxcVrouterInterfaceAttrs), ifaceValues)
	diags.Append(ifaceDiags...)
	vrouterModel.Interfaces = ifaceList

	vrouterObj, vrouterDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, vrouterModel)
	diags.Append(vrouterDiags...)

	configModel.VrouterPartnerConfig = vrouterObj

	resultObj, resultDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, configModel)
	diags.Append(resultDiags...)
	return resultObj, diags
}

// prefixFilterIDToName converts a prefix filter list ID to its description.
// Zero means the API did not echo the field and a non-zero miss means the
// lookup failed. Both keep the state value rather than manufacture drift
// against a filter that is still attached.
func prefixFilterIDToName(id int, pflMap map[int]string, existing basetypes.StringValue) basetypes.StringValue {
	if name, ok := pflMap[id]; ok {
		return types.StringValue(name)
	}
	return existing
}

// anyPrefixFilterListsConfigured reports whether any interface's BGP connections
// have a prefix filter list field set in state, so mergeVrouterPartnerConfigFromAPI
// can skip the prefix filter list round-trip when there is nothing to resolve.
func anyPrefixFilterListsConfigured(ctx context.Context, ifaces []*vxcPartnerConfigInterfaceModel) bool {
	for _, iface := range ifaces {
		if iface.BgpConnections.IsNull() || iface.BgpConnections.IsUnknown() {
			continue
		}
		var bgps []*bgpConnectionConfigModel
		if diags := iface.BgpConnections.ElementsAs(ctx, &bgps, false); diags.HasError() {
			continue
		}
		for _, b := range bgps {
			if !b.ImportWhitelist.IsNull() || !b.ImportBlacklist.IsNull() ||
				!b.ExportWhitelist.IsNull() || !b.ExportBlacklist.IsNull() {
				return true
			}
		}
	}
	return false
}

// These functions are used for partner configurations for ordering VXC Resources through the Megaport API.

func createAWSPartnerConfig(ctx context.Context, awsConfig vxcPartnerConfigAWSModel) (diag.Diagnostics, *megaport.VXCPartnerConfigAWS, basetypes.ObjectValue) {
	diags := diag.Diagnostics{}
	partnerConfig := &megaport.VXCPartnerConfigAWS{
		ConnectType:       awsConfig.ConnectType.ValueString(),
		Type:              awsConfig.Type.ValueString(),
		OwnerAccount:      awsConfig.OwnerAccount.ValueString(),
		ASN:               int(awsConfig.ASN.ValueInt64()),
		AmazonASN:         int(awsConfig.AmazonASN.ValueInt64()),
		AuthKey:           awsConfig.AuthKey.ValueString(),
		Prefixes:          awsConfig.Prefixes.ValueString(),
		CustomerIPAddress: awsConfig.CustomerIPAddress.ValueString(),
		AmazonIPAddress:   awsConfig.AmazonIPAddress.ValueString(),
		ConnectionName:    awsConfig.ConnectionName.ValueString(),
	}
	awsConfigObj, awsDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAWSAttrs, awsConfig)
	diags.Append(awsDiags...)

	azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
	google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
	oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
	vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
	aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
	ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
	partnerConfigModel := &vxcPartnerConfigurationModel{
		Partner:              types.StringValue("aws"),
		AWSPartnerConfig:     awsConfigObj,
		AzurePartnerConfig:   azure,
		GooglePartnerConfig:  google,
		OraclePartnerConfig:  oracle,
		VrouterPartnerConfig: vrouter,
		PartnerAEndConfig:    aEndPartner,
		IBMPartnerConfig:     ibmPartner,
	}

	partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, partnerConfigModel)
	diags.Append(partnerDiags...)

	return diags, partnerConfig, partnerConfigObj
}

func createAzurePartnerConfig(ctx context.Context, azureConfig vxcPartnerConfigAzureModel) (diag.Diagnostics, megaport.VXCPartnerConfigAzure, basetypes.ObjectValue) {
	diags := diag.Diagnostics{}
	partnerConfig := megaport.VXCPartnerConfigAzure{
		ConnectType: "AZURE",
		ServiceKey:  azureConfig.ServiceKey.ValueString(),
	}

	azurePeerModels := []partnerOrderAzurePeeringConfigModel{}
	azurePeerDiags := azureConfig.Peers.ElementsAs(ctx, &azurePeerModels, false)
	diags.Append(azurePeerDiags...)
	if len(azurePeerModels) > 0 {
		partnerConfig.Peers = []megaport.PartnerOrderAzurePeeringConfig{}
		for _, peer := range azurePeerModels {
			peeringConfig := megaport.PartnerOrderAzurePeeringConfig{
				Type:            peer.Type.ValueString(),
				PeerASN:         peer.PeerASN.ValueString(),
				PrimarySubnet:   peer.PrimarySubnet.ValueString(),
				SecondarySubnet: peer.SecondarySubnet.ValueString(),
				VLAN:            int(peer.VLAN.ValueInt64()),
			}
			if !peer.Prefixes.IsNull() {
				peeringConfig.Prefixes = peer.Prefixes.ValueString()
			}
			if !peer.SharedKey.IsNull() {
				peeringConfig.SharedKey = peer.SharedKey.ValueString()
			}
			partnerConfig.Peers = append(partnerConfig.Peers, peeringConfig)
		}
	}

	azureConfigObj, azureDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAzureAttrs, azureConfig)
	diags.Append(azureDiags...)

	aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
	google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
	oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
	vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
	aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
	ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
	partnerConfigModel := &vxcPartnerConfigurationModel{
		Partner:              types.StringValue("azure"),
		AWSPartnerConfig:     aws,
		AzurePartnerConfig:   azureConfigObj,
		GooglePartnerConfig:  google,
		OraclePartnerConfig:  oracle,
		VrouterPartnerConfig: vrouter,
		PartnerAEndConfig:    aEndPartner,
		IBMPartnerConfig:     ibmPartner,
	}

	partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, partnerConfigModel)
	diags.Append(partnerDiags...)
	return diags, partnerConfig, partnerConfigObj
}

func createGooglePartnerConfig(ctx context.Context, googleConfig vxcPartnerConfigGoogleModel) (diag.Diagnostics, megaport.VXCPartnerConfigGoogle, basetypes.ObjectValue) {
	diags := diag.Diagnostics{}
	googlePartnerConfig := megaport.VXCPartnerConfigGoogle{
		ConnectType: "GOOGLE",
		PairingKey:  googleConfig.PairingKey.ValueString(),
	}
	googleConfigObj, googleDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigGoogleAttrs, googleConfig)
	diags.Append(googleDiags...)
	aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
	azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
	oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
	vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
	aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
	ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
	aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
		Partner:              types.StringValue("google"),
		AWSPartnerConfig:     aws,
		AzurePartnerConfig:   azure,
		GooglePartnerConfig:  googleConfigObj,
		OraclePartnerConfig:  oracle,
		VrouterPartnerConfig: vrouter,
		IBMPartnerConfig:     ibmPartner,
		PartnerAEndConfig:    aEndPartner,
	}

	partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
	diags.Append(partnerDiags...)
	return diags, googlePartnerConfig, partnerConfigObj
}

func createOraclePartnerConfig(ctx context.Context, oracleConfig vxcPartnerConfigOracleModel) (diag.Diagnostics, megaport.VXCPartnerConfigOracle, basetypes.ObjectValue) {
	diags := diag.Diagnostics{}
	oraclePartnerConfig := megaport.VXCPartnerConfigOracle{
		ConnectType:      "ORACLE",
		VirtualCircuitId: oracleConfig.VirtualCircuitId.ValueString(),
	}
	oracleConfigObj, oracleDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigOracleAttrs, oracleConfig)
	diags.Append(oracleDiags...)

	aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
	azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
	google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
	vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
	aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
	ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
	bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
		Partner:              types.StringValue("oracle"),
		AWSPartnerConfig:     aws,
		AzurePartnerConfig:   azure,
		GooglePartnerConfig:  google,
		OraclePartnerConfig:  oracleConfigObj,
		IBMPartnerConfig:     ibmPartner,
		VrouterPartnerConfig: vrouter,
		PartnerAEndConfig:    aEndPartner,
	}

	partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
	diags.Append(partnerDiags...)
	return diags, oraclePartnerConfig, partnerConfigObj
}

func createIBMPartnerConfig(ctx context.Context, ibmConfig vxcPartnerConfigIbmModel) (diag.Diagnostics, megaport.VXCPartnerConfigIBM, basetypes.ObjectValue) {
	diags := diag.Diagnostics{}
	ibmPartnerConfig := megaport.VXCPartnerConfigIBM{
		ConnectType:       "IBM",
		AccountID:         ibmConfig.AccountID.ValueString(),
		CustomerASN:       int(ibmConfig.CustomerASN.ValueInt64()),
		Name:              ibmConfig.Name.ValueString(),
		CustomerIPAddress: ibmConfig.CustomerIPAddress.ValueString(),
		ProviderIPAddress: ibmConfig.ProviderIPAddress.ValueString(),
	}
	aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
	azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
	google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
	oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
	vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
	aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
	ibmParnterConfigObj, ibmDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigIbmAttrs, ibmConfig)
	diags.Append(ibmDiags...)
	aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
		Partner:              types.StringValue("ibm"),
		AWSPartnerConfig:     aws,
		AzurePartnerConfig:   azure,
		GooglePartnerConfig:  google,
		OraclePartnerConfig:  oracle,
		VrouterPartnerConfig: vrouter,
		PartnerAEndConfig:    aEndPartner,
		IBMPartnerConfig:     ibmParnterConfigObj,
	}
	partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
	diags.Append(partnerDiags...)
	return diags, ibmPartnerConfig, partnerConfigObj
}

// ipSecPreSharedKeysFromConfig reads the write-only pre_shared_key for each
// vrouter interface from the configuration, keyed by interface index. The PSK is
// a write-only argument, so it is null in the plan and must be sourced from
// config when ordering the tunnel. pathRoot is the partner-config attribute name
// ("a_end_partner_config" or "b_end_partner_config").
func ipSecPreSharedKeysFromConfig(ctx context.Context, config tfsdk.Config, pathRoot string, ifaceCount int) (map[int]string, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	psks := map[int]string{}
	for i := 0; i < ifaceCount; i++ {
		tunnelPath := path.Root(pathRoot).
			AtName("vrouter_config").
			AtName("interfaces").
			AtListIndex(i).
			AtName("ip_sec_tunnel_options")
		// Read the tunnel object first so we never descend into a null object on
		// interfaces that have no tunnel.
		var tunnel types.Object
		diags.Append(config.GetAttribute(ctx, tunnelPath, &tunnel)...)
		if diags.HasError() {
			return psks, diags
		}
		if tunnel.IsNull() || tunnel.IsUnknown() {
			continue
		}
		var psk types.String
		diags.Append(config.GetAttribute(ctx, tunnelPath.AtName("pre_shared_key"), &psk)...)
		if diags.HasError() {
			return psks, diags
		}
		if !psk.IsNull() && !psk.IsUnknown() {
			psks[i] = psk.ValueString()
		}
	}
	return psks, diags
}

func createVrouterPartnerConfig(ctx context.Context, vrouterConfig vxcPartnerConfigVrouterModel, prefixFilterList []*megaport.PrefixFilterList, preSharedKeys map[int]string) (diag.Diagnostics, *megaport.VXCOrderVrouterPartnerConfig, basetypes.ObjectValue) {
	diags := diag.Diagnostics{}
	vrouterPartnerConfig := &megaport.VXCOrderVrouterPartnerConfig{}
	ifaceModels := []*vxcPartnerConfigInterfaceModel{}
	ifaceDiags := vrouterConfig.Interfaces.ElementsAs(ctx, &ifaceModels, false)
	diags.Append(ifaceDiags...)
	for i, iface := range ifaceModels {
		toAppend := megaport.PartnerConfigInterface{}
		if !iface.IpMtu.IsNull() {
			toAppend.IpMtu = int(iface.IpMtu.ValueInt64())
		}
		if !iface.Description.IsNull() {
			toAppend.Description = iface.Description.ValueString()
		}
		if !iface.InterfaceType.IsNull() {
			toAppend.InterfaceType = iface.InterfaceType.ValueString()
		}
		if !iface.PacketFilterIn.IsNull() {
			toAppend.PacketFilterIn = megaport.PtrTo(iface.PacketFilterIn.ValueInt64())
		}
		if !iface.PacketFilterOut.IsNull() {
			toAppend.PacketFilterOut = megaport.PtrTo(iface.PacketFilterOut.ValueInt64())
		}
		if !iface.IPAddresses.IsNull() {
			ipAddresses := []string{}
			ipDiags := iface.IPAddresses.ElementsAs(ctx, &ipAddresses, true)
			diags.Append(ipDiags...)
			toAppend.IpAddresses = ipAddresses
		}
		if !iface.IPRoutes.IsNull() {
			ipRoutes := []*ipRouteModel{}
			ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
			diags.Append(ipRouteDiags...)
			for _, ipRoute := range ipRoutes {
				toAppend.IpRoutes = append(toAppend.IpRoutes, megaport.IpRoute{
					Prefix:      ipRoute.Prefix.ValueString(),
					Description: ipRoute.Description.ValueString(),
					NextHop:     ipRoute.NextHop.ValueString(),
				})
			}
		}
		if !iface.NatIPAddresses.IsNull() {
			natIPAddresses := []string{}
			natDiags := iface.NatIPAddresses.ElementsAs(ctx, &natIPAddresses, true)
			diags.Append(natDiags...)
			toAppend.NatIpAddresses = natIPAddresses
		}
		if !iface.Bfd.IsNull() {
			bfd := &bfdConfigModel{}
			bfdDiags := iface.Bfd.As(ctx, bfd, basetypes.ObjectAsOptions{})
			diags.Append(bfdDiags...)
			toAppend.Bfd = megaport.BfdConfig{
				TxInterval: int(bfd.TxInterval.ValueInt64()),
				RxInterval: int(bfd.RxInterval.ValueInt64()),
				Multiplier: int(bfd.Multiplier.ValueInt64()),
			}
		}
		if !iface.VLAN.IsNull() {
			toAppend.VLAN = int(iface.VLAN.ValueInt64())
		}
		if !iface.BgpConnections.IsNull() {
			bgpConnections := []*bgpConnectionConfigModel{}
			bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConnections, false)
			diags.Append(bgpDiags...)
			for _, bgpConnection := range bgpConnections {
				bgpToAppend := megaport.BgpConnectionConfig{
					PeerAsn:            int(bgpConnection.PeerAsn.ValueInt64()),
					LocalIpAddress:     bgpConnection.LocalIPAddress.ValueString(),
					PeerIpAddress:      bgpConnection.PeerIPAddress.ValueString(),
					Password:           bgpConnection.Password.ValueString(),
					Shutdown:           bgpConnection.Shutdown.ValueBool(),
					Description:        bgpConnection.Description.ValueString(),
					MedIn:              int(bgpConnection.MedIn.ValueInt64()),
					MedOut:             int(bgpConnection.MedOut.ValueInt64()),
					BfdEnabled:         bgpConnection.BfdEnabled.ValueBool(),
					ExportPolicy:       bgpConnection.ExportPolicy.ValueString(),
					AsPathPrependCount: int(bgpConnection.AsPathPrependCount.ValueInt64()),
					PeerType:           bgpConnection.PeerType.ValueString(),
				}
				if !bgpConnection.LocalAsn.IsNull() {
					bgpToAppend.LocalAsn = megaport.PtrTo(int(bgpConnection.LocalAsn.ValueInt64()))
				}
				if !bgpConnection.AsOverride.IsNull() {
					bgpToAppend.AsOverride = megaport.PtrTo(bgpConnection.AsOverride.ValueBool())
				}
				if !bgpConnection.ImportWhitelist.IsNull() {
					id, d := resolvePrefixListID(prefixFilterList, bgpConnection.ImportWhitelist.ValueString(), "import_whitelist")
					diags.Append(d...)
					bgpToAppend.ImportWhitelist = id
				}
				if !bgpConnection.ImportBlacklist.IsNull() {
					id, d := resolvePrefixListID(prefixFilterList, bgpConnection.ImportBlacklist.ValueString(), "import_blacklist")
					diags.Append(d...)
					bgpToAppend.ImportBlacklist = id
				}
				if !bgpConnection.ExportWhitelist.IsNull() {
					id, d := resolvePrefixListID(prefixFilterList, bgpConnection.ExportWhitelist.ValueString(), "export_whitelist")
					diags.Append(d...)
					bgpToAppend.ExportWhitelist = id
				}
				if !bgpConnection.ExportBlacklist.IsNull() {
					id, d := resolvePrefixListID(prefixFilterList, bgpConnection.ExportBlacklist.ValueString(), "export_blacklist")
					diags.Append(d...)
					bgpToAppend.ExportBlacklist = id
				}
				if !bgpConnection.PermitExportTo.IsNull() {
					permitExportTo := []string{}
					permitDiags := bgpConnection.PermitExportTo.ElementsAs(ctx, &permitExportTo, true)
					diags.Append(permitDiags...)
					bgpToAppend.PermitExportTo = permitExportTo
				}
				if !bgpConnection.DenyExportTo.IsNull() {
					denyExportTo := []string{}
					denyDiags := bgpConnection.DenyExportTo.ElementsAs(ctx, &denyExportTo, true)
					diags.Append(denyDiags...)
					bgpToAppend.DenyExportTo = denyExportTo
				}
				toAppend.BgpConnections = append(toAppend.BgpConnections, bgpToAppend)
			}
		}
		if !iface.IpSecTunnelOptions.IsNull() && !iface.IpSecTunnelOptions.IsUnknown() {
			var t ipSecTunnelOptionsModel
			tunnelDiags := iface.IpSecTunnelOptions.As(ctx, &t, basetypes.ObjectAsOptions{})
			diags.Append(tunnelDiags...)
			// pre_shared_key is write-only, so it is null in t (sourced from the
			// plan). Pull it from the configuration, keyed by interface index.
			tunnel := megaport.IPsecTunnelConfig{
				SourceIpAddress:      t.SourceIPAddress.ValueString(),
				DestinationIpAddress: t.DestinationIPAddress.ValueString(),
				PreSharedKey:         preSharedKeys[i],
				LocalId:              t.LocalID.ValueString(),
				RemoteId:             t.RemoteID.ValueString(),
			}
			// Pointer fields: only set when configured so nil keeps the API default.
			if !t.Passive.IsNull() {
				tunnel.Passive = megaport.PtrTo(t.Passive.ValueBool())
			}
			if !t.Phase1Lifetime.IsNull() {
				tunnel.Phase1Lifetime = megaport.PtrTo(int(t.Phase1Lifetime.ValueInt64()))
			}
			if !t.Phase2Lifetime.IsNull() {
				tunnel.Phase2Lifetime = megaport.PtrTo(int(t.Phase2Lifetime.ValueInt64()))
			}
			toAppend.IpSecTunnelOptions = &tunnel
		}
		vrouterPartnerConfig.Interfaces = append(vrouterPartnerConfig.Interfaces, toAppend)
	}
	vrouterConfigObj, bEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, vrouterConfig)
	diags.Append(bEndDiags...)
	aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
	azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
	google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
	oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
	aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
	ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
	partnerConfigModel := &vxcPartnerConfigurationModel{
		Partner:              types.StringValue("vrouter"),
		AWSPartnerConfig:     aws,
		AzurePartnerConfig:   azure,
		GooglePartnerConfig:  google,
		OraclePartnerConfig:  oracle,
		IBMPartnerConfig:     ibmPartner,
		VrouterPartnerConfig: vrouterConfigObj,
		PartnerAEndConfig:    aEndPartner,
	}
	partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, partnerConfigModel)
	diags.Append(partnerDiags...)
	return diags, vrouterPartnerConfig, partnerConfigObj
}

func createAEndPartnerConfig(ctx context.Context, partnerConfigAEndModel vxcPartnerConfigAEndModel, prefixFilterList []*megaport.PrefixFilterList) (diag.Diagnostics, *megaport.VXCOrderVrouterPartnerConfig, basetypes.ObjectValue) {
	diags := diag.Diagnostics{}
	aEndMegaportConfig := &megaport.VXCOrderVrouterPartnerConfig{}
	ifaceModels := []*vxcPartnerConfigInterfaceModel{}
	ifaceDiags := partnerConfigAEndModel.Interfaces.ElementsAs(ctx, &ifaceModels, true)
	diags.Append(ifaceDiags...)
	for _, iface := range ifaceModels {
		toAppend := megaport.PartnerConfigInterface{}
		if !iface.IpMtu.IsNull() {
			toAppend.IpMtu = int(iface.IpMtu.ValueInt64())
		}
		if !iface.IPAddresses.IsNull() {
			ipAddresses := []string{}
			ipDiags := iface.IPAddresses.ElementsAs(ctx, &ipAddresses, true)
			diags.Append(ipDiags...)
			toAppend.IpAddresses = ipAddresses
		}
		if !iface.IPRoutes.IsNull() {
			ipRoutes := []*ipRouteModel{}
			ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
			diags.Append(ipRouteDiags...)
			for _, ipRoute := range ipRoutes {
				toAppend.IpRoutes = append(toAppend.IpRoutes, megaport.IpRoute{
					Prefix:      ipRoute.Prefix.ValueString(),
					Description: ipRoute.Description.ValueString(),
					NextHop:     ipRoute.NextHop.ValueString(),
				})
			}
		}
		if !iface.NatIPAddresses.IsNull() {
			natIPAddresses := []string{}
			natDiags := iface.NatIPAddresses.ElementsAs(ctx, &natIPAddresses, true)
			diags.Append(natDiags...)
			toAppend.NatIpAddresses = natIPAddresses
		}
		if !iface.Bfd.IsNull() {
			bfd := &bfdConfigModel{}
			bfdDiags := iface.Bfd.As(ctx, bfd, basetypes.ObjectAsOptions{})
			diags.Append(bfdDiags...)
			toAppend.Bfd = megaport.BfdConfig{
				TxInterval: int(bfd.TxInterval.ValueInt64()),
				RxInterval: int(bfd.RxInterval.ValueInt64()),
				Multiplier: int(bfd.Multiplier.ValueInt64()),
			}
		}
		if !iface.BgpConnections.IsNull() {
			bgpConnections := []*bgpConnectionConfigModel{}
			bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConnections, false)
			diags.Append(bgpDiags...)
			for _, bgpConnection := range bgpConnections {
				bgpToAppend := megaport.BgpConnectionConfig{
					PeerAsn:            int(bgpConnection.PeerAsn.ValueInt64()),
					LocalIpAddress:     bgpConnection.LocalIPAddress.ValueString(),
					PeerIpAddress:      bgpConnection.PeerIPAddress.ValueString(),
					Password:           bgpConnection.Password.ValueString(),
					Shutdown:           bgpConnection.Shutdown.ValueBool(),
					Description:        bgpConnection.Description.ValueString(),
					MedIn:              int(bgpConnection.MedIn.ValueInt64()),
					MedOut:             int(bgpConnection.MedOut.ValueInt64()),
					BfdEnabled:         bgpConnection.BfdEnabled.ValueBool(),
					ExportPolicy:       bgpConnection.ExportPolicy.ValueString(),
					AsPathPrependCount: int(bgpConnection.AsPathPrependCount.ValueInt64()),
				}
				if !bgpConnection.LocalAsn.IsNull() {
					bgpToAppend.LocalAsn = megaport.PtrTo(int(bgpConnection.LocalAsn.ValueInt64()))
				}
				if !bgpConnection.AsOverride.IsNull() {
					bgpToAppend.AsOverride = megaport.PtrTo(bgpConnection.AsOverride.ValueBool())
				}
				if !bgpConnection.ImportWhitelist.IsNull() {
					id, d := resolvePrefixListID(prefixFilterList, bgpConnection.ImportWhitelist.ValueString(), "import_whitelist")
					diags.Append(d...)
					bgpToAppend.ImportWhitelist = id
				}
				if !bgpConnection.ImportBlacklist.IsNull() {
					id, d := resolvePrefixListID(prefixFilterList, bgpConnection.ImportBlacklist.ValueString(), "import_blacklist")
					diags.Append(d...)
					bgpToAppend.ImportBlacklist = id
				}
				if !bgpConnection.ExportWhitelist.IsNull() {
					id, d := resolvePrefixListID(prefixFilterList, bgpConnection.ExportWhitelist.ValueString(), "export_whitelist")
					diags.Append(d...)
					bgpToAppend.ExportWhitelist = id
				}
				if !bgpConnection.ExportBlacklist.IsNull() {
					id, d := resolvePrefixListID(prefixFilterList, bgpConnection.ExportBlacklist.ValueString(), "export_blacklist")
					diags.Append(d...)
					bgpToAppend.ExportBlacklist = id
				}
				if !bgpConnection.PermitExportTo.IsNull() {
					permitExportTo := []string{}
					permitDiags := bgpConnection.PermitExportTo.ElementsAs(ctx, &permitExportTo, true)
					diags.Append(permitDiags...)
					bgpToAppend.PermitExportTo = permitExportTo
				}
				if !bgpConnection.DenyExportTo.IsNull() {
					denyExportTo := []string{}
					denyDiags := bgpConnection.DenyExportTo.ElementsAs(ctx, &denyExportTo, true)
					diags.Append(denyDiags...)
					bgpToAppend.DenyExportTo = denyExportTo
				}
				toAppend.BgpConnections = append(toAppend.BgpConnections, bgpToAppend)
			}
		}
		aEndMegaportConfig.Interfaces = append(aEndMegaportConfig.Interfaces, toAppend)
	}
	aEndConfigObj, aEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAEndAttrs, partnerConfigAEndModel)
	diags.Append(aEndDiags...)
	aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
	azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
	google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
	oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
	vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
	ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
	aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
		Partner:              types.StringValue("a-end"),
		AWSPartnerConfig:     aws,
		AzurePartnerConfig:   azure,
		GooglePartnerConfig:  google,
		OraclePartnerConfig:  oracle,
		PartnerAEndConfig:    aEndConfigObj,
		VrouterPartnerConfig: vrouter,
		IBMPartnerConfig:     ibmPartner,
	}
	partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
	diags.Append(partnerDiags...)
	return diags, aEndMegaportConfig, partnerConfigObj
}

func createTransitPartnerConfig(ctx context.Context) (diag.Diagnostics, megaport.VXCPartnerConfigTransit, basetypes.ObjectValue) {
	diags := diag.Diagnostics{}
	transitPartnerConfig := megaport.VXCPartnerConfigTransit{
		ConnectType: "TRANSIT",
	}

	aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
	azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
	google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
	oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
	vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
	aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
	ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)

	transitPartnerConfigModel := &vxcPartnerConfigurationModel{
		Partner:              types.StringValue("transit"),
		AWSPartnerConfig:     aws,
		AzurePartnerConfig:   azure,
		GooglePartnerConfig:  google,
		OraclePartnerConfig:  oracle,
		VrouterPartnerConfig: vrouter,
		PartnerAEndConfig:    aEndPartner,
		IBMPartnerConfig:     ibmPartner,
	}

	transitConfigObj, transitDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, transitPartnerConfigModel)
	diags.Append(transitDiags...)

	return diags, transitPartnerConfig, transitConfigObj
}

func supportVLANUpdates(partnerType string) bool {
	// AWS and Transit connections do not support VLAN updates
	if partnerType == "aws" || partnerType == "transit" {
		return false
	}
	return true
}

// waitForVXCUpdate polls the VXC API to verify that an update has propagated successfully.
// It uses exponential backoff with a maximum backoff time to efficiently wait for API propagation.
//
// Parameters:
//   - ctx: Context for the operation (can be used for cancellation)
//   - uid: The unique identifier of the VXC being updated
//   - updateReq: The update request containing the expected values to verify
//   - timeout: Maximum time to wait for the update to propagate
//
// Returns an error if:
//   - The API calls fail
//   - The context is cancelled
//   - The timeout is reached before the update is verified
func (r *vxcResource) waitForVXCUpdate(ctx context.Context, uid string, updateReq *megaport.UpdateVXCRequest, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	backoff := 2 * time.Second
	maxBackoff := 10 * time.Second

	// Add initial delay before first check to allow for quick propagation
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second):
	}

	for time.Now().Before(deadline) {
		vxc, err := r.client.VXCService.GetVXC(ctx, uid)
		if err != nil {
			return fmt.Errorf("failed to retrieve VXC status during update verification for VXC UID %s: %w", uid, err)
		}

		// Verify the expected changes are reflected
		if r.verifyUpdateApplied(vxc, updateReq) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff = time.Duration(float64(backoff) * 1.5)
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	return fmt.Errorf("update verification timed out after %v", timeout)
}

// innerVLANMatches compares a requested inner VLAN to the value returned by the API,
// treating requested -1 (untagged) as satisfied by a returned 0, mirroring the
// normalization fromAPIVXC applies on Read.
func innerVLANMatches(requested, actual int) bool {
	if requested == -1 && actual == 0 {
		return true
	}
	return requested == actual
}

// verifyUpdateApplied checks if the VXC returned from the API matches the expected values
// from the update request. It verifies all fields that can be updated.
//
// Parameters:
//   - vxc: The VXC object retrieved from the API
//   - updateReq: The update request containing the expected values
//
// Returns true if all updated fields match their expected values, false otherwise.
func (r *vxcResource) verifyUpdateApplied(vxc *megaport.VXC, updateReq *megaport.UpdateVXCRequest) bool {
	// Verify VLAN-related fields
	if updateReq.AEndInnerVLAN != nil && !innerVLANMatches(*updateReq.AEndInnerVLAN, vxc.AEndConfiguration.InnerVLAN) {
		return false
	}
	if updateReq.BEndInnerVLAN != nil && !innerVLANMatches(*updateReq.BEndInnerVLAN, vxc.BEndConfiguration.InnerVLAN) {
		return false
	}
	if updateReq.AEndVLAN != nil && vxc.AEndConfiguration.VLAN != *updateReq.AEndVLAN {
		return false
	}
	if updateReq.BEndVLAN != nil && vxc.BEndConfiguration.VLAN != *updateReq.BEndVLAN {
		return false
	}

	// Verify basic VXC properties
	if updateReq.Name != nil && vxc.Name != *updateReq.Name {
		return false
	}
	if updateReq.RateLimit != nil && vxc.RateLimit != *updateReq.RateLimit {
		return false
	}
	if updateReq.CostCentre != nil && vxc.CostCentre != *updateReq.CostCentre {
		return false
	}
	if updateReq.Shutdown != nil && vxc.Shutdown != *updateReq.Shutdown {
		return false
	}
	if updateReq.Term != nil && vxc.ContractTermMonths != *updateReq.Term {
		return false
	}

	// Verify endpoint product UIDs
	if updateReq.AEndProductUID != nil && vxc.AEndConfiguration.UID != *updateReq.AEndProductUID {
		return false
	}
	if updateReq.BEndProductUID != nil && vxc.BEndConfiguration.UID != *updateReq.BEndProductUID {
		return false
	}

	// Verify VNIC indices
	if updateReq.AVnicIndex != nil && vxc.AEndConfiguration.NetworkInterfaceIndex != *updateReq.AVnicIndex {
		return false
	}
	if updateReq.BVnicIndex != nil && vxc.BEndConfiguration.NetworkInterfaceIndex != *updateReq.BVnicIndex {
		return false
	}

	// Note: Partner configs (AEndPartnerConfig, BEndPartnerConfig) are complex objects
	// and their verification would require deep comparison. For now, we focus on the
	// simpler scalar fields that are more prone to propagation delays.

	return true
}

// waitForVnicIndex polls the VXC API until the NetworkInterfaceIndex for the
// A-end and/or B-end matches the expected values. The API updates vnic_index
// asynchronously, so an immediate read after create/update may return a stale
// value. This function returns the VXC from the first successful poll so the
// caller can use it directly without another API call.
func (r *vxcResource) waitForVnicIndex(ctx context.Context, uid string, expectedAEnd *int, expectedBEnd *int, timeout time.Duration) (*megaport.VXC, error) {
	if expectedAEnd == nil && expectedBEnd == nil {
		// Nothing to wait for — just do a normal read.
		return r.client.VXCService.GetVXC(ctx, uid)
	}

	deadline := time.Now().Add(timeout)
	backoff := 2 * time.Second
	maxBackoff := 10 * time.Second

	// Small initial delay to let the API propagate.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(1 * time.Second):
	}

	for time.Now().Before(deadline) {
		vxc, err := r.client.VXCService.GetVXC(ctx, uid)
		if err != nil {
			return nil, fmt.Errorf("failed to read VXC %s while waiting for vnic_index propagation: %w", uid, err)
		}

		match := true
		if expectedAEnd != nil && vxc.AEndConfiguration.NetworkInterfaceIndex != *expectedAEnd {
			match = false
		}
		if expectedBEnd != nil && vxc.BEndConfiguration.NetworkInterfaceIndex != *expectedBEnd {
			match = false
		}
		if match {
			return vxc, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			backoff = time.Duration(float64(backoff) * 1.5)
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	// Timed out — do a final read, patch in the expected values so state
	// stays consistent with the plan, and warn the caller.
	vxc, err := r.client.VXCService.GetVXC(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to read VXC %s after vnic_index wait timeout: %w", uid, err)
	}
	if expectedAEnd != nil {
		vxc.AEndConfiguration.NetworkInterfaceIndex = *expectedAEnd
	}
	if expectedBEnd != nil {
		vxc.BEndConfiguration.NetworkInterfaceIndex = *expectedBEnd
	}
	return vxc, fmt.Errorf("vnic_index propagation timed out after %v for VXC %s — using expected values", timeout, uid)
}
