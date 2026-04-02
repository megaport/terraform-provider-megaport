package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	megaport "github.com/megaport/megaportgo"
)

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

	// Reconstruct partner configs from API CSP connection data when available,
	// falling back to plan/state preservation for non-vrouter partner types.
	apiDiags = append(apiDiags, orm.reconcilePartnerConfigs(ctx, v, plan, client)...)

	return apiDiags
}

// reconcilePartnerConfigs reconstructs vrouter partner configs from API CSP connection
// data to enable drift detection. For non-vrouter partner types (AWS, Azure, etc.),
// it preserves configs from the plan since those are order-time-only values.
func (orm *vxcResourceModel) reconcilePartnerConfigs(ctx context.Context, v *megaport.VXC, plan *vxcResourceModel, client *megaport.Client) diag.Diagnostics {
	diags := diag.Diagnostics{}

	// Find VirtualRouter CSP connections from the API response
	var vrouterCSPConns []megaport.CSPConnectionVirtualRouter
	if v.Resources != nil && v.Resources.CSPConnection != nil {
		for _, c := range v.Resources.CSPConnection.CSPConnection {
			if vr, ok := c.(megaport.CSPConnectionVirtualRouter); ok {
				vrouterCSPConns = append(vrouterCSPConns, vr)
			}
		}
	}

	// Determine the source for partner config (plan during Create/Update, state during Read)
	source := plan
	if source == nil {
		source = orm
	}

	// Handle A-End partner config
	aEndHandled := false
	if source != nil && !source.AEndPartnerConfig.IsNull() {
		partnerType := getPartnerType(ctx, source.AEndPartnerConfig)
		if partnerType == "vrouter" || partnerType == "a-end" {
			if len(vrouterCSPConns) > 0 {
				mcrUID := v.AEndConfiguration.UID
				reconstructed, reconDiags := reconstructVrouterPartnerConfig(ctx, vrouterCSPConns[0], source.AEndPartnerConfig, mcrUID, client, partnerType)
				diags.Append(reconDiags...)
				if !reconstructed.IsNull() {
					orm.AEndPartnerConfig = reconstructed
					aEndHandled = true
				}
			}
		}
		// For non-vrouter types, preserve from plan
		if !aEndHandled && plan != nil && !plan.AEndPartnerConfig.IsNull() {
			orm.AEndPartnerConfig = plan.AEndPartnerConfig
			aEndHandled = true
		}
	} else if plan != nil && !plan.AEndPartnerConfig.IsNull() {
		orm.AEndPartnerConfig = plan.AEndPartnerConfig
		aEndHandled = true
	}
	_ = aEndHandled

	// Handle B-End partner config
	bEndHandled := false
	if source != nil && !source.BEndPartnerConfig.IsNull() {
		partnerType := getPartnerType(ctx, source.BEndPartnerConfig)
		if partnerType == "vrouter" || partnerType == "a-end" {
			// For B-end vrouter, use the last CSP connection if multiple exist (MCR-to-MCR case)
			vrIdx := 0
			if len(vrouterCSPConns) > 1 {
				vrIdx = len(vrouterCSPConns) - 1
			}
			if len(vrouterCSPConns) > vrIdx {
				mcrUID := v.BEndConfiguration.UID
				reconstructed, reconDiags := reconstructVrouterPartnerConfig(ctx, vrouterCSPConns[vrIdx], source.BEndPartnerConfig, mcrUID, client, partnerType)
				diags.Append(reconDiags...)
				if !reconstructed.IsNull() {
					orm.BEndPartnerConfig = reconstructed
					bEndHandled = true
				}
			}
		}
		// For non-vrouter types, preserve from plan
		if !bEndHandled && plan != nil && !plan.BEndPartnerConfig.IsNull() {
			orm.BEndPartnerConfig = plan.BEndPartnerConfig
			bEndHandled = true
		}
	} else if plan != nil && !plan.BEndPartnerConfig.IsNull() {
		orm.BEndPartnerConfig = plan.BEndPartnerConfig
		bEndHandled = true
	}
	_ = bEndHandled

	return diags
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

// reconstructVrouterPartnerConfig builds a partner config object from CSP VirtualRouter
// API data, preserving passwords from existing state and resolving prefix filter list IDs to names.
func reconstructVrouterPartnerConfig(
	ctx context.Context,
	vrConn megaport.CSPConnectionVirtualRouter,
	existingPartnerConfig basetypes.ObjectValue,
	mcrUID string,
	client *megaport.Client,
	partnerType string, // "vrouter" or "a-end"
) (basetypes.ObjectValue, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	if len(vrConn.Interfaces) == 0 {
		// No interface data from API, return null to fall back to state preservation
		return types.ObjectNull(vxcPartnerConfigAttrs), diags
	}

	// Look up prefix filter lists for ID→name mapping
	var prefixFilterLists []*megaport.PrefixFilterList
	if client != nil && mcrUID != "" {
		var err error
		prefixFilterLists, err = client.MCRService.ListMCRPrefixFilterLists(ctx, mcrUID)
		if err != nil {
			tflog.Warn(ctx, "Failed to list MCR prefix filter lists for BGP drift detection, falling back to state",
				map[string]interface{}{"mcr_uid": mcrUID, "error": err.Error()})
			return types.ObjectNull(vxcPartnerConfigAttrs), diags
		}
	}

	// Build a map of prefix filter list ID → description (name)
	pflMap := make(map[int]string)
	for _, pfl := range prefixFilterLists {
		pflMap[pfl.Id] = pfl.Description
	}

	// Extract existing BGP passwords from state for preservation
	existingPasswords := extractExistingBGPPasswords(ctx, existingPartnerConfig, partnerType)

	// Build interface models from API data
	interfaceModels := make([]vxcPartnerConfigInterfaceModel, 0, len(vrConn.Interfaces))
	for ifaceIdx, apiIface := range vrConn.Interfaces {
		ifaceModel := vxcPartnerConfigInterfaceModel{}

		// IP Addresses
		if len(apiIface.IPAddresses) > 0 {
			ipList, ipDiags := types.ListValueFrom(ctx, types.StringType, apiIface.IPAddresses)
			diags.Append(ipDiags...)
			ifaceModel.IPAddresses = ipList
		} else {
			ifaceModel.IPAddresses = types.ListNull(types.StringType)
		}

		// IP Routes
		if len(apiIface.IPRoutes) > 0 {
			routeModels := make([]ipRouteModel, 0, len(apiIface.IPRoutes))
			for _, r := range apiIface.IPRoutes {
				routeModels = append(routeModels, ipRouteModel{
					Prefix:      types.StringValue(r.Prefix),
					Description: types.StringValue(r.Description),
					NextHop:     types.StringValue(r.NextHop),
				})
			}
			routeList, routeDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(ipRouteAttrs), routeModels)
			diags.Append(routeDiags...)
			ifaceModel.IPRoutes = routeList
		} else {
			ifaceModel.IPRoutes = types.ListNull(types.ObjectType{}.WithAttributeTypes(ipRouteAttrs))
		}

		// NAT IP Addresses
		if len(apiIface.NatIPAddresses) > 0 {
			natList, natDiags := types.ListValueFrom(ctx, types.StringType, apiIface.NatIPAddresses)
			diags.Append(natDiags...)
			ifaceModel.NatIPAddresses = natList
		} else {
			ifaceModel.NatIPAddresses = types.ListNull(types.StringType)
		}

		// BFD
		if apiIface.BFD.TxInterval > 0 || apiIface.BFD.RxInterval > 0 || apiIface.BFD.Multiplier > 0 {
			bfdModel := bfdConfigModel{
				TxInterval: types.Int64Value(int64(apiIface.BFD.TxInterval)),
				RxInterval: types.Int64Value(int64(apiIface.BFD.RxInterval)),
				Multiplier: types.Int64Value(int64(apiIface.BFD.Multiplier)),
			}
			bfdObj, bfdDiags := types.ObjectValueFrom(ctx, bfdConfigAttrs, bfdModel)
			diags.Append(bfdDiags...)
			ifaceModel.Bfd = bfdObj
		} else {
			ifaceModel.Bfd = types.ObjectNull(bfdConfigAttrs)
		}

		// VLAN - use 0 as null (not set)
		ifaceModel.VLAN = types.Int64Null()

		// IP MTU - default is 1500, always set from API if available
		ifaceModel.IpMtu = types.Int64Null()

		// BGP Connections
		bgpAttrTypes := bgpVrouterConnectionConfig
		if partnerType == "a-end" {
			bgpAttrTypes = bgpConnectionConfig
		}

		if len(apiIface.BGPConnections) > 0 {
			bgpModels := make([]bgpConnectionConfigModel, 0, len(apiIface.BGPConnections))
			for bgpIdx, apiBgp := range apiIface.BGPConnections {
				bgpModel := bgpConnectionConfigModel{
					PeerAsn:            types.Int64Value(int64(apiBgp.PeerAsn)),
					LocalIPAddress:     types.StringValue(apiBgp.LocalIpAddress),
					PeerIPAddress:      types.StringValue(apiBgp.PeerIpAddress),
					Shutdown:           types.BoolValue(apiBgp.Shutdown),
					Description:        types.StringValue(apiBgp.Description),
					MedIn:              types.Int64Value(int64(apiBgp.MedIn)),
					MedOut:             types.Int64Value(int64(apiBgp.MedOut)),
					BfdEnabled:         types.BoolValue(apiBgp.BfdEnabled),
					ExportPolicy:       types.StringValue(apiBgp.ExportPolicy),
					AsPathPrependCount: types.Int64Value(int64(apiBgp.AsPathPrependCount)),
				}

				// PeerType (only for vrouter, not deprecated a-end)
				if partnerType == "vrouter" {
					bgpModel.PeerType = types.StringValue(apiBgp.PeerType)
				}

				// LocalAsn - handle nil pointer
				if apiBgp.LocalAsn != nil {
					bgpModel.LocalAsn = types.Int64Value(int64(*apiBgp.LocalAsn))
				} else {
					bgpModel.LocalAsn = types.Int64Null()
				}

				// Password - preserve from existing state (API doesn't return it)
				key := fmt.Sprintf("%d:%d", ifaceIdx, bgpIdx)
				if pw, ok := existingPasswords[key]; ok {
					bgpModel.Password = pw
				} else {
					bgpModel.Password = types.StringNull()
				}

				// Prefix filter lists - reverse lookup ID → name
				bgpModel.ImportWhitelist = prefixFilterIDToName(apiBgp.ImportWhitelist, pflMap)
				bgpModel.ImportBlacklist = prefixFilterIDToName(apiBgp.ImportBlacklist, pflMap)
				bgpModel.ExportWhitelist = prefixFilterIDToName(apiBgp.ExportWhitelist, pflMap)
				bgpModel.ExportBlacklist = prefixFilterIDToName(apiBgp.ExportBlacklist, pflMap)

				// PermitExportTo
				if len(apiBgp.PermitExportTo) > 0 {
					permitList, permitDiags := types.ListValueFrom(ctx, types.StringType, apiBgp.PermitExportTo)
					diags.Append(permitDiags...)
					bgpModel.PermitExportTo = permitList
				} else {
					bgpModel.PermitExportTo = types.ListNull(types.StringType)
				}

				// DenyExportTo
				if len(apiBgp.DenyExportTo) > 0 {
					denyList, denyDiags := types.ListValueFrom(ctx, types.StringType, apiBgp.DenyExportTo)
					diags.Append(denyDiags...)
					bgpModel.DenyExportTo = denyList
				} else {
					bgpModel.DenyExportTo = types.ListNull(types.StringType)
				}

				bgpModels = append(bgpModels, bgpModel)
			}
			bgpList, bgpDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(bgpAttrTypes), bgpModels)
			diags.Append(bgpDiags...)
			ifaceModel.BgpConnections = bgpList
		} else {
			ifaceModel.BgpConnections = types.ListNull(types.ObjectType{}.WithAttributeTypes(bgpAttrTypes))
		}

		interfaceModels = append(interfaceModels, ifaceModel)
	}

	// Build the partner config object based on partner type
	var partnerConfigObj basetypes.ObjectValue

	if partnerType == "vrouter" {
		vrouterModel := vxcPartnerConfigVrouterModel{}
		ifaceList, ifaceDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(vxcVrouterInterfaceAttrs), interfaceModels)
		diags.Append(ifaceDiags...)
		vrouterModel.Interfaces = ifaceList

		vrouterObj, vrouterDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, vrouterModel)
		diags.Append(vrouterDiags...)

		partnerConfigModel := &vxcPartnerConfigurationModel{
			Partner:              types.StringValue("vrouter"),
			AWSPartnerConfig:     types.ObjectNull(vxcPartnerConfigAWSAttrs),
			AzurePartnerConfig:   types.ObjectNull(vxcPartnerConfigAzureAttrs),
			GooglePartnerConfig:  types.ObjectNull(vxcPartnerConfigGoogleAttrs),
			OraclePartnerConfig:  types.ObjectNull(vxcPartnerConfigOracleAttrs),
			IBMPartnerConfig:     types.ObjectNull(vxcPartnerConfigIbmAttrs),
			VrouterPartnerConfig: vrouterObj,
			PartnerAEndConfig:    types.ObjectNull(vxcPartnerConfigAEndAttrs),
		}
		obj, objDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, partnerConfigModel)
		diags.Append(objDiags...)
		partnerConfigObj = obj
	} else {
		// "a-end" (deprecated)
		aEndModel := vxcPartnerConfigAEndModel{}
		ifaceList, ifaceDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(vxcInterfaceAttrs), interfaceModels)
		diags.Append(ifaceDiags...)
		aEndModel.Interfaces = ifaceList

		aEndObj, aEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAEndAttrs, aEndModel)
		diags.Append(aEndDiags...)

		partnerConfigModel := &vxcPartnerConfigurationModel{
			Partner:              types.StringValue("a-end"),
			AWSPartnerConfig:     types.ObjectNull(vxcPartnerConfigAWSAttrs),
			AzurePartnerConfig:   types.ObjectNull(vxcPartnerConfigAzureAttrs),
			GooglePartnerConfig:  types.ObjectNull(vxcPartnerConfigGoogleAttrs),
			OraclePartnerConfig:  types.ObjectNull(vxcPartnerConfigOracleAttrs),
			IBMPartnerConfig:     types.ObjectNull(vxcPartnerConfigIbmAttrs),
			VrouterPartnerConfig: types.ObjectNull(vxcPartnerConfigVrouterAttrs),
			PartnerAEndConfig:    aEndObj,
		}
		obj, objDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, partnerConfigModel)
		diags.Append(objDiags...)
		partnerConfigObj = obj
	}

	return partnerConfigObj, diags
}

// extractExistingBGPPasswords extracts BGP passwords from existing partner config state.
// Returns a map of "ifaceIdx:bgpIdx" → password value.
func extractExistingBGPPasswords(ctx context.Context, partnerConfig basetypes.ObjectValue, partnerType string) map[string]basetypes.StringValue {
	passwords := make(map[string]basetypes.StringValue)
	if partnerConfig.IsNull() || partnerConfig.IsUnknown() {
		return passwords
	}

	configModel := &vxcPartnerConfigurationModel{}
	diags := partnerConfig.As(ctx, configModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return passwords
	}

	var interfacesList types.List
	if partnerType == "vrouter" && !configModel.VrouterPartnerConfig.IsNull() {
		vrouterModel := &vxcPartnerConfigVrouterModel{}
		vrDiags := configModel.VrouterPartnerConfig.As(ctx, vrouterModel, basetypes.ObjectAsOptions{})
		if vrDiags.HasError() {
			return passwords
		}
		interfacesList = vrouterModel.Interfaces
	} else if partnerType == "a-end" && !configModel.PartnerAEndConfig.IsNull() {
		aEndModel := &vxcPartnerConfigAEndModel{}
		aeDiags := configModel.PartnerAEndConfig.As(ctx, aEndModel, basetypes.ObjectAsOptions{})
		if aeDiags.HasError() {
			return passwords
		}
		interfacesList = aEndModel.Interfaces
	} else {
		return passwords
	}

	if interfacesList.IsNull() || interfacesList.IsUnknown() {
		return passwords
	}

	ifaceModels := []*vxcPartnerConfigInterfaceModel{}
	ifDiags := interfacesList.ElementsAs(ctx, &ifaceModels, false)
	if ifDiags.HasError() {
		return passwords
	}

	for ifaceIdx, iface := range ifaceModels {
		if iface.BgpConnections.IsNull() || iface.BgpConnections.IsUnknown() {
			continue
		}
		bgpModels := []*bgpConnectionConfigModel{}
		bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpModels, false)
		if bgpDiags.HasError() {
			continue
		}
		for bgpIdx, bgp := range bgpModels {
			key := fmt.Sprintf("%d:%d", ifaceIdx, bgpIdx)
			passwords[key] = bgp.Password
		}
	}

	return passwords
}

// prefixFilterIDToName converts a prefix filter list ID to its name (description).
// Returns null if the ID is 0 (unset).
func prefixFilterIDToName(id int, pflMap map[int]string) basetypes.StringValue {
	if id == 0 {
		return types.StringNull()
	}
	if name, ok := pflMap[id]; ok {
		return types.StringValue(name)
	}
	return types.StringNull()
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

func createVrouterPartnerConfig(ctx context.Context, vrouterConfig vxcPartnerConfigVrouterModel, prefixFilterList []*megaport.PrefixFilterList) (diag.Diagnostics, *megaport.VXCOrderVrouterPartnerConfig, basetypes.ObjectValue) {
	diags := diag.Diagnostics{}
	vrouterPartnerConfig := &megaport.VXCOrderVrouterPartnerConfig{}
	ifaceModels := []*vxcPartnerConfigInterfaceModel{}
	ifaceDiags := vrouterConfig.Interfaces.ElementsAs(ctx, &ifaceModels, false)
	diags.Append(ifaceDiags...)
	for _, iface := range ifaceModels {
		toAppend := megaport.PartnerConfigInterface{}
		if !iface.IpMtu.IsNull() && !iface.IpMtu.IsUnknown() {
			toAppend.IpMtu = int(iface.IpMtu.ValueInt64())
		}
		if !iface.IPAddresses.IsNull() && !iface.IPAddresses.IsUnknown() {
			ipAddresses := []string{}
			ipDiags := iface.IPAddresses.ElementsAs(ctx, &ipAddresses, true)
			diags.Append(ipDiags...)
			toAppend.IpAddresses = ipAddresses
		}
		if !iface.IPRoutes.IsNull() && !iface.IPRoutes.IsUnknown() {
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
		if !iface.NatIPAddresses.IsNull() && !iface.NatIPAddresses.IsUnknown() {
			natIPAddresses := []string{}
			natDiags := iface.NatIPAddresses.ElementsAs(ctx, &natIPAddresses, true)
			diags.Append(natDiags...)
			toAppend.NatIpAddresses = natIPAddresses
		}
		if !iface.Bfd.IsNull() && !iface.Bfd.IsUnknown() {
			bfd := &bfdConfigModel{}
			bfdDiags := iface.Bfd.As(ctx, bfd, basetypes.ObjectAsOptions{})
			diags.Append(bfdDiags...)
			toAppend.Bfd = megaport.BfdConfig{
				TxInterval: int(bfd.TxInterval.ValueInt64()),
				RxInterval: int(bfd.RxInterval.ValueInt64()),
				Multiplier: int(bfd.Multiplier.ValueInt64()),
			}
		}
		if !iface.VLAN.IsNull() && !iface.VLAN.IsUnknown() {
			toAppend.VLAN = int(iface.VLAN.ValueInt64())
		}
		if !iface.BgpConnections.IsNull() && !iface.BgpConnections.IsUnknown() {
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
				if !bgpConnection.LocalAsn.IsNull() && !bgpConnection.LocalAsn.IsUnknown() {
					bgpToAppend.LocalAsn = megaport.PtrTo(int(bgpConnection.LocalAsn.ValueInt64()))
				}
				if !bgpConnection.ImportWhitelist.IsNull() && !bgpConnection.ImportWhitelist.IsUnknown() {
					for _, pfl := range prefixFilterList {
						if pfl.Description == bgpConnection.ImportWhitelist.ValueString() {
							bgpToAppend.ImportWhitelist = pfl.Id
						}
					}
				}
				if !bgpConnection.ImportBlacklist.IsNull() && !bgpConnection.ImportBlacklist.IsUnknown() {
					for _, pfl := range prefixFilterList {
						if pfl.Description == bgpConnection.ImportBlacklist.ValueString() {
							bgpToAppend.ImportBlacklist = pfl.Id
						}
					}
				}
				if !bgpConnection.ExportWhitelist.IsNull() && !bgpConnection.ExportWhitelist.IsUnknown() {
					for _, pfl := range prefixFilterList {
						if pfl.Description == bgpConnection.ExportWhitelist.ValueString() {
							bgpToAppend.ExportWhitelist = pfl.Id
						}
					}
				}
				if !bgpConnection.ExportBlacklist.IsNull() && !bgpConnection.ExportBlacklist.IsUnknown() {
					for _, pfl := range prefixFilterList {
						if pfl.Description == bgpConnection.ExportBlacklist.ValueString() {
							bgpToAppend.ExportBlacklist = pfl.Id
						}
					}
				}
				if !bgpConnection.PermitExportTo.IsNull() && !bgpConnection.PermitExportTo.IsUnknown() {
					permitExportTo := []string{}
					permitDiags := bgpConnection.PermitExportTo.ElementsAs(ctx, &permitExportTo, true)
					diags.Append(permitDiags...)
					bgpToAppend.PermitExportTo = permitExportTo
					bgpToAppend.PermitExportTo = permitExportTo
				}
				if !bgpConnection.DenyExportTo.IsNull() && !bgpConnection.DenyExportTo.IsUnknown() {
					denyExportTo := []string{}
					denyDiags := bgpConnection.DenyExportTo.ElementsAs(ctx, &denyExportTo, true)
					diags.Append(denyDiags...)
					bgpToAppend.DenyExportTo = denyExportTo
				}
				toAppend.BgpConnections = append(toAppend.BgpConnections, bgpToAppend)
			}
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
		if !iface.IpMtu.IsNull() && !iface.IpMtu.IsUnknown() {
			toAppend.IpMtu = int(iface.IpMtu.ValueInt64())
		}
		if !iface.IPAddresses.IsNull() && !iface.IPAddresses.IsUnknown() {
			ipAddresses := []string{}
			ipDiags := iface.IPAddresses.ElementsAs(ctx, &ipAddresses, true)
			diags.Append(ipDiags...)
			toAppend.IpAddresses = ipAddresses
		}
		if !iface.IPRoutes.IsNull() && !iface.IPRoutes.IsUnknown() {
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
		if !iface.NatIPAddresses.IsNull() && !iface.NatIPAddresses.IsUnknown() {
			natIPAddresses := []string{}
			natDiags := iface.NatIPAddresses.ElementsAs(ctx, &natIPAddresses, true)
			diags.Append(natDiags...)
			toAppend.NatIpAddresses = natIPAddresses
		}
		if !iface.Bfd.IsNull() && !iface.Bfd.IsUnknown() {
			bfd := &bfdConfigModel{}
			bfdDiags := iface.Bfd.As(ctx, bfd, basetypes.ObjectAsOptions{})
			diags.Append(bfdDiags...)
			toAppend.Bfd = megaport.BfdConfig{
				TxInterval: int(bfd.TxInterval.ValueInt64()),
				RxInterval: int(bfd.RxInterval.ValueInt64()),
				Multiplier: int(bfd.Multiplier.ValueInt64()),
			}
		}
		if !iface.BgpConnections.IsNull() && !iface.BgpConnections.IsUnknown() {
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
				if !bgpConnection.LocalAsn.IsNull() && !bgpConnection.LocalAsn.IsUnknown() {
					bgpToAppend.LocalAsn = megaport.PtrTo(int(bgpConnection.LocalAsn.ValueInt64()))
				}
				if !bgpConnection.ImportWhitelist.IsNull() && !bgpConnection.ImportWhitelist.IsUnknown() {
					for _, pfl := range prefixFilterList {
						if pfl.Description == bgpConnection.ImportWhitelist.ValueString() {
							bgpToAppend.ImportWhitelist = pfl.Id
						}
					}
				}
				if !bgpConnection.ImportBlacklist.IsNull() && !bgpConnection.ImportBlacklist.IsUnknown() {
					for _, pfl := range prefixFilterList {
						if pfl.Description == bgpConnection.ImportBlacklist.ValueString() {
							bgpToAppend.ImportBlacklist = pfl.Id
						}
					}
				}
				if !bgpConnection.ExportWhitelist.IsNull() && !bgpConnection.ExportWhitelist.IsUnknown() {
					for _, pfl := range prefixFilterList {
						if pfl.Description == bgpConnection.ExportWhitelist.ValueString() {
							bgpToAppend.ExportWhitelist = pfl.Id
						}
					}
				}
				if !bgpConnection.ExportBlacklist.IsNull() && !bgpConnection.ExportBlacklist.IsUnknown() {
					for _, pfl := range prefixFilterList {
						if pfl.Description == bgpConnection.ExportBlacklist.ValueString() {
							bgpToAppend.ExportBlacklist = pfl.Id
						}
					}
				}
				if !bgpConnection.PermitExportTo.IsNull() && !bgpConnection.PermitExportTo.IsUnknown() {
					permitExportTo := []string{}
					permitDiags := bgpConnection.PermitExportTo.ElementsAs(ctx, &permitExportTo, true)
					diags.Append(permitDiags...)
					bgpToAppend.PermitExportTo = permitExportTo
					bgpToAppend.PermitExportTo = permitExportTo
				}
				if !bgpConnection.DenyExportTo.IsNull() && !bgpConnection.DenyExportTo.IsUnknown() {
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
	if updateReq.AEndInnerVLAN != nil && vxc.AEndConfiguration.InnerVLAN != *updateReq.AEndInnerVLAN {
		return false
	}
	if updateReq.BEndInnerVLAN != nil && vxc.BEndConfiguration.InnerVLAN != *updateReq.BEndInnerVLAN {
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
