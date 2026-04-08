package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	megaport "github.com/megaport/megaportgo"
)

// fromAPIVXC updates the resource model from API response data.
// The optional plan parameter allows preserving user-only fields (like product_uid,
// vlan, and partner configs) that are not returned by the API. This is particularly
// important after import or during updates where the plan contains user configuration that
// would otherwise be lost.
func (orm *vxcResourceModel) fromAPIVXC(ctx context.Context, v *megaport.VXC, tags map[string]string, plan *vxcResourceModel) diag.Diagnostics {
	apiDiags := diag.Diagnostics{}

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

	// Build A-End config from API, supplementing with plan/state values that
	// the API does not return.
	aEndModel := &vxcAEndConfigModel{
		AssignedProductUID: types.StringValue(v.AEndConfiguration.UID),
	}

	// Default: preserve product_uid from existing state, then override from plan.
	aEndProductUID := ""
	var aEndVnicIndex *int64
	var aEndVLAN *int64
	var aEndInnerVLAN *int64

	if !orm.AEndConfiguration.IsNull() {
		existingAEnd := &vxcAEndConfigModel{}
		if ds := orm.AEndConfiguration.As(ctx, existingAEnd, basetypes.ObjectAsOptions{}); !ds.HasError() {
			aEndProductUID = existingAEnd.ProductUID.ValueString()
			// Preserve vnic_index from state during Read (plan == nil).
			if plan == nil && !existingAEnd.NetworkInterfaceIndex.IsNull() && !existingAEnd.NetworkInterfaceIndex.IsUnknown() {
				idx := existingAEnd.NetworkInterfaceIndex.ValueInt64()
				aEndVnicIndex = &idx
			}
			if !existingAEnd.VLAN.IsNull() && !existingAEnd.VLAN.IsUnknown() {
				vlan := existingAEnd.VLAN.ValueInt64()
				aEndVLAN = &vlan
			}
			if !existingAEnd.InnerVLAN.IsNull() && !existingAEnd.InnerVLAN.IsUnknown() {
				vlan := existingAEnd.InnerVLAN.ValueInt64()
				aEndInnerVLAN = &vlan
			}
		} else {
			apiDiags.Append(ds...)
		}
	}

	if plan != nil && !plan.AEndConfiguration.IsNull() {
		planAEnd := &vxcAEndConfigModel{}
		if ds := plan.AEndConfiguration.As(ctx, planAEnd, basetypes.ObjectAsOptions{}); !ds.HasError() {
			if !planAEnd.ProductUID.IsNull() && !planAEnd.ProductUID.IsUnknown() {
				aEndProductUID = planAEnd.ProductUID.ValueString()
			}
			if aEndVnicIndex == nil && !planAEnd.NetworkInterfaceIndex.IsNull() && !planAEnd.NetworkInterfaceIndex.IsUnknown() {
				idx := planAEnd.NetworkInterfaceIndex.ValueInt64()
				aEndVnicIndex = &idx
			}
			if !planAEnd.VLAN.IsNull() && !planAEnd.VLAN.IsUnknown() {
				vlan := planAEnd.VLAN.ValueInt64()
				aEndVLAN = &vlan
			}
			if !planAEnd.InnerVLAN.IsNull() && !planAEnd.InnerVLAN.IsUnknown() {
				vlan := planAEnd.InnerVLAN.ValueInt64()
				aEndInnerVLAN = &vlan
			}
			// Preserve vrouter_config from plan (not returned by API).
			if !planAEnd.VrouterPartnerConfig.IsNull() && !planAEnd.VrouterPartnerConfig.IsUnknown() {
				aEndModel.VrouterPartnerConfig = planAEnd.VrouterPartnerConfig
			}
		} else {
			apiDiags.Append(ds...)
		}
	}

	aEndModel.ProductUID = types.StringValue(aEndProductUID)

	// VLAN: prefer API value; fall back to plan/state if API returns 0.
	if v.AEndConfiguration.VLAN != 0 {
		aEndModel.VLAN = types.Int64Value(int64(v.AEndConfiguration.VLAN))
	} else if aEndVLAN != nil {
		aEndModel.VLAN = types.Int64Value(*aEndVLAN)
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
	apiDiags = append(apiDiags, aEndDiags...)
	orm.AEndConfiguration = aEnd

	// Build B-End config.
	bEndModel := &vxcBEndConfigModel{
		AssignedProductUID: types.StringValue(v.BEndConfiguration.UID),
	}

	bEndProductUID := ""
	var bEndVnicIndex *int64
	var bEndVLAN *int64
	var bEndInnerVLAN *int64

	if !orm.BEndConfiguration.IsNull() {
		existingBEnd := &vxcBEndConfigModel{}
		if ds := orm.BEndConfiguration.As(ctx, existingBEnd, basetypes.ObjectAsOptions{}); !ds.HasError() {
			bEndProductUID = existingBEnd.ProductUID.ValueString()
			if plan == nil && !existingBEnd.NetworkInterfaceIndex.IsNull() && !existingBEnd.NetworkInterfaceIndex.IsUnknown() {
				idx := existingBEnd.NetworkInterfaceIndex.ValueInt64()
				bEndVnicIndex = &idx
			}
			if !existingBEnd.VLAN.IsNull() && !existingBEnd.VLAN.IsUnknown() {
				vlan := existingBEnd.VLAN.ValueInt64()
				bEndVLAN = &vlan
			}
			if !existingBEnd.InnerVLAN.IsNull() && !existingBEnd.InnerVLAN.IsUnknown() {
				vlan := existingBEnd.InnerVLAN.ValueInt64()
				bEndInnerVLAN = &vlan
			}
		} else {
			apiDiags.Append(ds...)
		}
	}

	if plan != nil && !plan.BEndConfiguration.IsNull() {
		planBEnd := &vxcBEndConfigModel{}
		if ds := plan.BEndConfiguration.As(ctx, planBEnd, basetypes.ObjectAsOptions{}); !ds.HasError() {
			if !planBEnd.ProductUID.IsNull() && !planBEnd.ProductUID.IsUnknown() {
				bEndProductUID = planBEnd.ProductUID.ValueString()
			}
			if bEndVnicIndex == nil && !planBEnd.NetworkInterfaceIndex.IsNull() && !planBEnd.NetworkInterfaceIndex.IsUnknown() {
				idx := planBEnd.NetworkInterfaceIndex.ValueInt64()
				bEndVnicIndex = &idx
			}
			if !planBEnd.VLAN.IsNull() && !planBEnd.VLAN.IsUnknown() {
				vlan := planBEnd.VLAN.ValueInt64()
				bEndVLAN = &vlan
			}
			if !planBEnd.InnerVLAN.IsNull() && !planBEnd.InnerVLAN.IsUnknown() {
				vlan := planBEnd.InnerVLAN.ValueInt64()
				bEndInnerVLAN = &vlan
			}
			// Preserve partner configs from plan (not returned by API).
			if !planBEnd.AWSPartnerConfig.IsNull() && !planBEnd.AWSPartnerConfig.IsUnknown() {
				bEndModel.AWSPartnerConfig = planBEnd.AWSPartnerConfig
			}
			if !planBEnd.AzurePartnerConfig.IsNull() && !planBEnd.AzurePartnerConfig.IsUnknown() {
				bEndModel.AzurePartnerConfig = planBEnd.AzurePartnerConfig
			}
			if !planBEnd.GooglePartnerConfig.IsNull() && !planBEnd.GooglePartnerConfig.IsUnknown() {
				bEndModel.GooglePartnerConfig = planBEnd.GooglePartnerConfig
			}
			if !planBEnd.OraclePartnerConfig.IsNull() && !planBEnd.OraclePartnerConfig.IsUnknown() {
				bEndModel.OraclePartnerConfig = planBEnd.OraclePartnerConfig
			}
			if !planBEnd.IBMPartnerConfig.IsNull() && !planBEnd.IBMPartnerConfig.IsUnknown() {
				bEndModel.IBMPartnerConfig = planBEnd.IBMPartnerConfig
			}
			if !planBEnd.VrouterPartnerConfig.IsNull() && !planBEnd.VrouterPartnerConfig.IsUnknown() {
				bEndModel.VrouterPartnerConfig = planBEnd.VrouterPartnerConfig
			}
			if !planBEnd.Transit.IsNull() && !planBEnd.Transit.IsUnknown() {
				bEndModel.Transit = planBEnd.Transit
			}
		} else {
			apiDiags.Append(ds...)
		}
	}

	bEndModel.ProductUID = types.StringValue(bEndProductUID)

	if v.BEndConfiguration.VLAN != 0 {
		bEndModel.VLAN = types.Int64Value(int64(v.BEndConfiguration.VLAN))
	} else if bEndVLAN != nil {
		bEndModel.VLAN = types.Int64Value(*bEndVLAN)
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
	if bEndModel.AWSPartnerConfig.IsUnknown() || bEndModel.AWSPartnerConfig.AttributeTypes(ctx) == nil {
		bEndModel.AWSPartnerConfig = types.ObjectNull(vxcPartnerConfigAWSAttrs)
	}
	if bEndModel.AzurePartnerConfig.IsUnknown() || bEndModel.AzurePartnerConfig.AttributeTypes(ctx) == nil {
		bEndModel.AzurePartnerConfig = types.ObjectNull(vxcPartnerConfigAzureAttrs)
	}
	if bEndModel.GooglePartnerConfig.IsUnknown() || bEndModel.GooglePartnerConfig.AttributeTypes(ctx) == nil {
		bEndModel.GooglePartnerConfig = types.ObjectNull(vxcPartnerConfigGoogleAttrs)
	}
	if bEndModel.OraclePartnerConfig.IsUnknown() || bEndModel.OraclePartnerConfig.AttributeTypes(ctx) == nil {
		bEndModel.OraclePartnerConfig = types.ObjectNull(vxcPartnerConfigOracleAttrs)
	}
	if bEndModel.IBMPartnerConfig.IsUnknown() || bEndModel.IBMPartnerConfig.AttributeTypes(ctx) == nil {
		bEndModel.IBMPartnerConfig = types.ObjectNull(vxcPartnerConfigIbmAttrs)
	}
	if bEndModel.VrouterPartnerConfig.IsUnknown() || bEndModel.VrouterPartnerConfig.AttributeTypes(ctx) == nil {
		bEndModel.VrouterPartnerConfig = types.ObjectNull(vxcPartnerConfigVrouterAttrs)
	}
	if bEndModel.Transit.IsUnknown() {
		bEndModel.Transit = types.BoolNull()
	}

	bEnd, bEndDiags := types.ObjectValueFrom(ctx, vxcBEndConfigAttrs, bEndModel)
	apiDiags = append(apiDiags, bEndDiags...)
	orm.BEndConfiguration = bEnd

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

	return apiDiags
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
