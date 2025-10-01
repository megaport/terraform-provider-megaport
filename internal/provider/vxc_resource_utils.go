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

func (orm *vxcResourceModel) fromAPIVXC(ctx context.Context, v *megaport.VXC, tags map[string]string) diag.Diagnostics {
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
	var aEndRequestedProductUID, bEndRequestedProductUID string
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
		bEndRequestedProductUID = existingBEnd.RequestedProductUID.ValueString()
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

func (r *vxcResource) waitForVXCUpdate(ctx context.Context, uid string, updateReq *megaport.UpdateVXCRequest, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	backoff := 2 * time.Second
	maxBackoff := 10 * time.Second

	for time.Now().Before(deadline) {
		vxc, err := r.client.VXCService.GetVXC(ctx, uid)
		if err != nil {
			return fmt.Errorf("failed to verify update: %w", err)
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

func (r *vxcResource) verifyUpdateApplied(vxc *megaport.VXC, updateReq *megaport.UpdateVXCRequest) bool {
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
	return true
}
