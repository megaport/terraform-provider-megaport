package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	megaport "github.com/megaport/megaportgo"
)

// isPresent reports whether a Terraform attribute value has been set (i.e., it is
// neither null nor unknown). Use this instead of the verbose !v.IsNull() && !v.IsUnknown()
// pattern which is scattered throughout the codebase.
func isPresent(v attr.Value) bool {
	return !v.IsNull() && !v.IsUnknown()
}

// fromAPIVXC updates the resource model from API response data.
// The optional plan parameter allows preserving user-only fields (like product_uid,
// vlan, and partner configs) that are not returned by the API. This is particularly
// important after import or during updates where the plan contains user configuration that
// would otherwise be lost.
func (orm *vxcResourceModel) fromAPIVXC(ctx context.Context, v *megaport.VXC, tags map[string]string, plan *vxcResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	orm.UID = types.StringValue(v.UID)
	orm.Name = types.StringValue(v.Name)
	orm.ServiceID = types.Int64Value(int64(v.ServiceID))
	orm.RateLimit = types.Int64Value(int64(v.RateLimit))
	orm.DistanceBand = types.StringValue(v.DistanceBand)
	orm.CreatedBy = types.StringValue(v.CreatedBy)
	orm.ContractTermMonths = types.Int64Value(int64(v.ContractTermMonths))
	orm.CompanyUID = types.StringValue(v.CompanyUID)
	orm.Shutdown = types.BoolValue(v.Shutdown)
	orm.CostCentre = types.StringValue(v.CostCentre)

	diags.Append(orm.buildAEndConfig(ctx, v, plan)...)
	diags.Append(orm.buildBEndConfig(ctx, v, plan)...)
	diags.Append(orm.mapVXCTags(ctx, v, tags)...)

	return diags
}

// buildAEndConfig constructs the A-End configuration model from API data, supplementing
// with plan/state values that the API does not return.
func (orm *vxcResourceModel) buildAEndConfig(ctx context.Context, v *megaport.VXC, plan *vxcResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	aEndModel := &vxcAEndConfigModel{
		AssignedProductUID: types.StringValue(v.AEndConfiguration.UID),
	}

	// Default: preserve product_uid from existing state, then override from plan.
	aEndProductUID := ""
	var aEndVnicIndex *int64
	var aEndVLAN *int64
	var aEndPlanVLAN *int64
	var aEndInnerVLAN *int64

	if !orm.AEndConfiguration.IsNull() {
		existingAEnd := &vxcAEndConfigModel{}
		if ds := orm.AEndConfiguration.As(ctx, existingAEnd, basetypes.ObjectAsOptions{}); !ds.HasError() {
			aEndProductUID = existingAEnd.ProductUID.ValueString()
			// Preserve vnic_index from state during Read (plan == nil).
			if plan == nil && isPresent(existingAEnd.NetworkInterfaceIndex) {
				idx := existingAEnd.NetworkInterfaceIndex.ValueInt64()
				aEndVnicIndex = &idx
			}
			if isPresent(existingAEnd.VLAN) {
				vlan := existingAEnd.VLAN.ValueInt64()
				aEndVLAN = &vlan
			}
			if isPresent(existingAEnd.InnerVLAN) {
				vlan := existingAEnd.InnerVLAN.ValueInt64()
				aEndInnerVLAN = &vlan
			}
			// Preserve vrouter_config from state during Read (plan == nil).
			if plan == nil && isPresent(existingAEnd.VrouterPartnerConfig) {
				aEndModel.VrouterPartnerConfig = existingAEnd.VrouterPartnerConfig
			}
		} else {
			diags.Append(ds...)
		}
	}

	if plan != nil && !plan.AEndConfiguration.IsNull() {
		planAEnd := &vxcAEndConfigModel{}
		if ds := plan.AEndConfiguration.As(ctx, planAEnd, basetypes.ObjectAsOptions{}); !ds.HasError() {
			if isPresent(planAEnd.ProductUID) {
				aEndProductUID = planAEnd.ProductUID.ValueString()
			}
			if aEndVnicIndex == nil && isPresent(planAEnd.NetworkInterfaceIndex) {
				idx := planAEnd.NetworkInterfaceIndex.ValueInt64()
				aEndVnicIndex = &idx
			}
			if isPresent(planAEnd.VLAN) {
				vlan := planAEnd.VLAN.ValueInt64()
				aEndVLAN = &vlan
				aEndPlanVLAN = &vlan
			}
			if isPresent(planAEnd.InnerVLAN) {
				vlan := planAEnd.InnerVLAN.ValueInt64()
				aEndInnerVLAN = &vlan
			}
			// Preserve vrouter_config from plan (not returned by API).
			if isPresent(planAEnd.VrouterPartnerConfig) {
				aEndModel.VrouterPartnerConfig = planAEnd.VrouterPartnerConfig
			}
		} else {
			diags.Append(ds...)
		}
	}

	aEndModel.ProductUID = types.StringValue(aEndProductUID)

	// VLAN priority:
	//   1. Plan value (Create/Update): always authoritative, including vlan=0 (auto-assign).
	//      vlan=0 is stored as 0 in state — not replaced with the API-assigned value.
	//      This avoids "planned value does not match config value" framework errors that
	//      arise when plan modifiers try to convert a known config value to Unknown.
	//   2. State value (Read, plan==nil): preserve whatever the user configured in state.
	//      The API can assign a different VLAN than requested (e.g. assigns 1691 for vlan=200),
	//      so the configured value is authoritative — we never overwrite state from the API.
	//   3. API value (non-zero): only used when state has no VLAN (import or first-time read).
	//   4. Null: no VLAN configured and API returned 0.
	if aEndPlanVLAN != nil {
		aEndModel.VLAN = types.Int64Value(*aEndPlanVLAN)
	} else if aEndVLAN != nil {
		// Preserve configured VLAN from state (including 0 for auto-assign).
		aEndModel.VLAN = types.Int64Value(*aEndVLAN)
	} else if v.AEndConfiguration.VLAN != 0 {
		aEndModel.VLAN = types.Int64Value(int64(v.AEndConfiguration.VLAN))
	} else {
		aEndModel.VLAN = types.Int64Null()
	}

	// InnerVLAN: preserve -1 (untagged) from state.
	if v.AEndConfiguration.InnerVLAN != 0 {
		aEndModel.InnerVLAN = types.Int64Value(int64(v.AEndConfiguration.InnerVLAN))
	} else if aEndInnerVLAN != nil && *aEndInnerVLAN == -1 {
		aEndModel.InnerVLAN = types.Int64Value(-1)
	} else {
		aEndModel.InnerVLAN = types.Int64Null()
	}

	// VnicIndex: use API, fall back to plan/state.
	if aEndVnicIndex != nil {
		aEndModel.NetworkInterfaceIndex = types.Int64Value(*aEndVnicIndex)
	} else {
		aEndModel.NetworkInterfaceIndex = types.Int64Value(int64(v.AEndConfiguration.NetworkInterfaceIndex))
	}

	// Initialise vrouter_config to null if not already set from plan.
	if aEndModel.VrouterPartnerConfig.IsUnknown() || aEndModel.VrouterPartnerConfig.IsNull() || aEndModel.VrouterPartnerConfig.AttributeTypes(ctx) == nil {
		aEndModel.VrouterPartnerConfig = types.ObjectNull(vxcPartnerConfigVrouterAttrs)
	}

	aEnd, aEndDiags := types.ObjectValueFrom(ctx, vxcAEndConfigAttrs, aEndModel)
	diags.Append(aEndDiags...)
	orm.AEndConfiguration = aEnd

	return diags
}

// buildBEndConfig constructs the B-End configuration model from API data, supplementing
// with plan/state values that the API does not return.
func (orm *vxcResourceModel) buildBEndConfig(ctx context.Context, v *megaport.VXC, plan *vxcResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	bEndModel := &vxcBEndConfigModel{
		AssignedProductUID: types.StringValue(v.BEndConfiguration.UID),
	}

	bEndProductUID := ""
	var bEndVnicIndex *int64
	var bEndVLAN *int64
	var bEndPlanVLAN *int64
	var bEndInnerVLAN *int64

	if !orm.BEndConfiguration.IsNull() {
		existingBEnd := &vxcBEndConfigModel{}
		if ds := orm.BEndConfiguration.As(ctx, existingBEnd, basetypes.ObjectAsOptions{}); !ds.HasError() {
			bEndProductUID = existingBEnd.ProductUID.ValueString()
			if plan == nil && isPresent(existingBEnd.NetworkInterfaceIndex) {
				idx := existingBEnd.NetworkInterfaceIndex.ValueInt64()
				bEndVnicIndex = &idx
			}
			if isPresent(existingBEnd.VLAN) {
				vlan := existingBEnd.VLAN.ValueInt64()
				bEndVLAN = &vlan
			}
			if isPresent(existingBEnd.InnerVLAN) {
				vlan := existingBEnd.InnerVLAN.ValueInt64()
				bEndInnerVLAN = &vlan
			}
			// Preserve partner configs from state during Read (plan == nil).
			if plan == nil {
				for _, pair := range []struct {
					src *types.Object
					dst *types.Object
				}{
					{&existingBEnd.AWSPartnerConfig, &bEndModel.AWSPartnerConfig},
					{&existingBEnd.AzurePartnerConfig, &bEndModel.AzurePartnerConfig},
					{&existingBEnd.GooglePartnerConfig, &bEndModel.GooglePartnerConfig},
					{&existingBEnd.OraclePartnerConfig, &bEndModel.OraclePartnerConfig},
					{&existingBEnd.IBMPartnerConfig, &bEndModel.IBMPartnerConfig},
					{&existingBEnd.VrouterPartnerConfig, &bEndModel.VrouterPartnerConfig},
				} {
					if isPresent(*pair.src) {
						*pair.dst = *pair.src
					}
				}
				if isPresent(existingBEnd.Transit) {
					bEndModel.Transit = existingBEnd.Transit
				}
			}
		} else {
			diags.Append(ds...)
		}
	}

	if plan != nil && !plan.BEndConfiguration.IsNull() {
		planBEnd := &vxcBEndConfigModel{}
		if ds := plan.BEndConfiguration.As(ctx, planBEnd, basetypes.ObjectAsOptions{}); !ds.HasError() {
			if isPresent(planBEnd.ProductUID) {
				bEndProductUID = planBEnd.ProductUID.ValueString()
			}
			if bEndVnicIndex == nil && isPresent(planBEnd.NetworkInterfaceIndex) {
				idx := planBEnd.NetworkInterfaceIndex.ValueInt64()
				bEndVnicIndex = &idx
			}
			if isPresent(planBEnd.VLAN) {
				vlan := planBEnd.VLAN.ValueInt64()
				bEndVLAN = &vlan
				bEndPlanVLAN = &vlan
			}
			if isPresent(planBEnd.InnerVLAN) {
				vlan := planBEnd.InnerVLAN.ValueInt64()
				bEndInnerVLAN = &vlan
			}
			// Preserve partner configs from plan — the API does not return them on read.
			for _, pair := range []struct {
				src *types.Object
				dst *types.Object
			}{
				{&planBEnd.AWSPartnerConfig, &bEndModel.AWSPartnerConfig},
				{&planBEnd.AzurePartnerConfig, &bEndModel.AzurePartnerConfig},
				{&planBEnd.GooglePartnerConfig, &bEndModel.GooglePartnerConfig},
				{&planBEnd.OraclePartnerConfig, &bEndModel.OraclePartnerConfig},
				{&planBEnd.IBMPartnerConfig, &bEndModel.IBMPartnerConfig},
				{&planBEnd.VrouterPartnerConfig, &bEndModel.VrouterPartnerConfig},
			} {
				if isPresent(*pair.src) {
					*pair.dst = *pair.src
				}
			}
			if isPresent(planBEnd.Transit) {
				bEndModel.Transit = planBEnd.Transit
			}
		} else {
			diags.Append(ds...)
		}
	}

	bEndModel.ProductUID = types.StringValue(bEndProductUID)

	// VLAN priority: same logic as A-End above.
	if bEndPlanVLAN != nil {
		bEndModel.VLAN = types.Int64Value(*bEndPlanVLAN)
	} else if bEndVLAN != nil {
		bEndModel.VLAN = types.Int64Value(*bEndVLAN)
	} else if v.BEndConfiguration.VLAN != 0 {
		bEndModel.VLAN = types.Int64Value(int64(v.BEndConfiguration.VLAN))
	} else {
		bEndModel.VLAN = types.Int64Null()
	}

	if v.BEndConfiguration.InnerVLAN != 0 {
		bEndModel.InnerVLAN = types.Int64Value(int64(v.BEndConfiguration.InnerVLAN))
	} else if bEndInnerVLAN != nil && *bEndInnerVLAN == -1 {
		bEndModel.InnerVLAN = types.Int64Value(-1)
	} else {
		bEndModel.InnerVLAN = types.Int64Null()
	}

	if bEndVnicIndex != nil {
		bEndModel.NetworkInterfaceIndex = types.Int64Value(*bEndVnicIndex)
	} else {
		bEndModel.NetworkInterfaceIndex = types.Int64Value(int64(v.BEndConfiguration.NetworkInterfaceIndex))
	}

	// Initialise any partner config fields to null if not set from plan.
	for _, pair := range []struct {
		field *types.Object
		attrs map[string]attr.Type
	}{
		{&bEndModel.AWSPartnerConfig, vxcPartnerConfigAWSAttrs},
		{&bEndModel.AzurePartnerConfig, vxcPartnerConfigAzureAttrs},
		{&bEndModel.GooglePartnerConfig, vxcPartnerConfigGoogleAttrs},
		{&bEndModel.OraclePartnerConfig, vxcPartnerConfigOracleAttrs},
		{&bEndModel.IBMPartnerConfig, vxcPartnerConfigIbmAttrs},
		{&bEndModel.VrouterPartnerConfig, vxcPartnerConfigVrouterAttrs},
	} {
		if !isPresent(*pair.field) {
			*pair.field = types.ObjectNull(pair.attrs)
		}
	}
	if bEndModel.Transit.IsUnknown() {
		bEndModel.Transit = types.BoolNull()
	}

	bEnd, bEndDiags := types.ObjectValueFrom(ctx, vxcBEndConfigAttrs, bEndModel)
	diags.Append(bEndDiags...)
	orm.BEndConfiguration = bEnd

	return diags
}

// mapVXCTags maps attribute tags and resource tags from API response data into the model.
func (orm *vxcResourceModel) mapVXCTags(ctx context.Context, v *megaport.VXC, tags map[string]string) diag.Diagnostics {
	var diags diag.Diagnostics

	if v.AttributeTags != nil {
		attributeTags, attributeDiags := types.MapValueFrom(ctx, types.StringType, v.AttributeTags)
		diags.Append(attributeDiags...)
		orm.AttributeTags = attributeTags
	} else {
		orm.AttributeTags = types.MapNull(types.StringType)
	}

	if len(tags) > 0 {
		resourceTags, tagDiags := types.MapValueFrom(ctx, types.StringType, tags)
		diags.Append(tagDiags...)
		orm.ResourceTags = resourceTags
	} else {
		orm.ResourceTags = types.MapNull(types.StringType)
	}

	return diags
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
	return diags, partnerConfig, awsConfigObj
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
	return diags, partnerConfig, azureConfigObj
}

func createGooglePartnerConfig(ctx context.Context, googleConfig vxcPartnerConfigGoogleModel) (diag.Diagnostics, megaport.VXCPartnerConfigGoogle, basetypes.ObjectValue) {
	diags := diag.Diagnostics{}
	googlePartnerConfig := megaport.VXCPartnerConfigGoogle{
		ConnectType: "GOOGLE",
		PairingKey:  googleConfig.PairingKey.ValueString(),
	}
	googleConfigObj, googleDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigGoogleAttrs, googleConfig)
	diags.Append(googleDiags...)
	return diags, googlePartnerConfig, googleConfigObj
}

func createOraclePartnerConfig(ctx context.Context, oracleConfig vxcPartnerConfigOracleModel) (diag.Diagnostics, megaport.VXCPartnerConfigOracle, basetypes.ObjectValue) {
	diags := diag.Diagnostics{}
	oraclePartnerConfig := megaport.VXCPartnerConfigOracle{
		ConnectType:      "ORACLE",
		VirtualCircuitId: oracleConfig.VirtualCircuitId.ValueString(),
	}
	oracleConfigObj, oracleDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigOracleAttrs, oracleConfig)
	diags.Append(oracleDiags...)
	return diags, oraclePartnerConfig, oracleConfigObj
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
	ibmConfigObj, ibmDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigIbmAttrs, ibmConfig)
	diags.Append(ibmDiags...)
	return diags, ibmPartnerConfig, ibmConfigObj
}

func createVrouterPartnerConfig(ctx context.Context, vrouterConfig vxcPartnerConfigVrouterModel, prefixFilterList []*megaport.PrefixFilterList) (diag.Diagnostics, *megaport.VXCOrderVrouterPartnerConfig) {
	diags := diag.Diagnostics{}
	vrouterPartnerConfig := &megaport.VXCOrderVrouterPartnerConfig{}
	ifaceModels := []*vxcPartnerConfigInterfaceModel{}
	ifaceDiags := vrouterConfig.Interfaces.ElementsAs(ctx, &ifaceModels, false)
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
					Shutdown:           bgpConnection.Shutdown.ValueBool(),
					Description:        bgpConnection.Description.ValueString(),
					MedIn:              int(bgpConnection.MedIn.ValueInt64()),
					MedOut:             int(bgpConnection.MedOut.ValueInt64()),
					BfdEnabled:         bgpConnection.BfdEnabled.ValueBool(),
					ExportPolicy:       bgpConnection.ExportPolicy.ValueString(),
					AsPathPrependCount: int(bgpConnection.AsPathPrependCount.ValueInt64()),
					PeerType:           bgpConnection.PeerType.ValueString(),
				}
				// Only send password if provided — avoids clearing BGP auth post-import
				// when the user has not supplied the WriteOnly field in their config.
				if !bgpConnection.Password.IsNull() {
					bgpToAppend.Password = bgpConnection.Password.ValueString()
				}
				if !bgpConnection.LocalAsn.IsNull() {
					bgpToAppend.LocalAsn = megaport.PtrTo(int(bgpConnection.LocalAsn.ValueInt64()))
				}
				if !bgpConnection.ImportWhitelist.IsNull() {
					for _, pfl := range prefixFilterList {
						if pfl.Description == bgpConnection.ImportWhitelist.ValueString() {
							bgpToAppend.ImportWhitelist = pfl.Id
						}
					}
				}
				if !bgpConnection.ImportBlacklist.IsNull() {
					for _, pfl := range prefixFilterList {
						if pfl.Description == bgpConnection.ImportBlacklist.ValueString() {
							bgpToAppend.ImportBlacklist = pfl.Id
						}
					}
				}
				if !bgpConnection.ExportWhitelist.IsNull() {
					for _, pfl := range prefixFilterList {
						if pfl.Description == bgpConnection.ExportWhitelist.ValueString() {
							bgpToAppend.ExportWhitelist = pfl.Id
						}
					}
				}
				if !bgpConnection.ExportBlacklist.IsNull() {
					for _, pfl := range prefixFilterList {
						if pfl.Description == bgpConnection.ExportBlacklist.ValueString() {
							bgpToAppend.ExportBlacklist = pfl.Id
						}
					}
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
		vrouterPartnerConfig.Interfaces = append(vrouterPartnerConfig.Interfaces, toAppend)
	}
	return diags, vrouterPartnerConfig
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
	cfg := RetryConfig{
		InitialBackoff: 2 * time.Second,
		MaxBackoff:     10 * time.Second,
		Multiplier:     1.5,
		Timeout:        timeout,
		InitialDelay:   1 * time.Second,
	}

	_, err := PollWithBackoff(ctx, cfg, func(ctx context.Context) (struct{}, bool, error) {
		vxc, err := r.client.VXCService.GetVXC(ctx, uid)
		if err != nil {
			return struct{}{}, false, fmt.Errorf("failed to retrieve VXC status during update verification for VXC UID %s: %w", uid, err)
		}
		if r.verifyUpdateApplied(vxc, updateReq) {
			return struct{}{}, true, nil
		}
		return struct{}{}, false, nil
	})

	if errors.Is(err, ErrPollTimeout) {
		return fmt.Errorf("update verification timed out after %v", timeout)
	}
	return err
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

	cfg := RetryConfig{
		InitialBackoff: 2 * time.Second,
		MaxBackoff:     10 * time.Second,
		Multiplier:     1.5,
		Timeout:        timeout,
		InitialDelay:   1 * time.Second,
	}

	vxc, err := PollWithBackoff(ctx, cfg, func(ctx context.Context) (*megaport.VXC, bool, error) {
		v, getErr := r.client.VXCService.GetVXC(ctx, uid)
		if getErr != nil {
			return nil, false, fmt.Errorf("failed to read VXC %s while waiting for vnic_index propagation: %w", uid, getErr)
		}
		match := (expectedAEnd == nil || v.AEndConfiguration.NetworkInterfaceIndex == *expectedAEnd) &&
			(expectedBEnd == nil || v.BEndConfiguration.NetworkInterfaceIndex == *expectedBEnd)
		if match {
			return v, true, nil
		}
		return nil, false, nil
	})

	if errors.Is(err, ErrPollTimeout) {
		// Timed out — do a final read, patch in the expected values so state
		// stays consistent with the plan, and warn the caller.
		vxc, readErr := r.client.VXCService.GetVXC(ctx, uid)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read VXC %s after vnic_index wait timeout: %w", uid, readErr)
		}
		if expectedAEnd != nil {
			vxc.AEndConfiguration.NetworkInterfaceIndex = *expectedAEnd
		}
		if expectedBEnd != nil {
			vxc.BEndConfiguration.NetworkInterfaceIndex = *expectedBEnd
		}
		return vxc, fmt.Errorf("vnic_index propagation timed out after %v for VXC %s — using expected values", timeout, uid)
	}
	return vxc, err
}
