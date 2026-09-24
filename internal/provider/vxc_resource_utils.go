package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	megaport "github.com/megaport/megaportgo"
)

// serviceKeyImportedPrivateKey marks that the null service_key in state came
// from the read after an import, not from an ordinary VXC that was never
// given a key. Read sets it; Update clears it once the key is recorded.
const serviceKeyImportedPrivateKey = "service_key_imported"

// requiresReplaceServiceKey replaces the VXC on any service key change,
// except recording a key for the first time on a VXC imported without one.
// The API never returns the key, so an imported VXC has it null in state
// regardless of whether the live VXC has one; the private flag tells that
// case apart from an ordinary VXC that was simply created with no key, which
// must still replace so the key actually reaches the API.
func requiresReplaceServiceKey(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	if !req.StateValue.IsNull() {
		resp.RequiresReplace = true
		return
	}

	imported, diags := req.Private.GetKey(ctx, serviceKeyImportedPrivateKey)
	resp.Diagnostics.Append(diags...)
	resp.RequiresReplace = len(imported) == 0
}

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
func (orm *vxcResourceModel) fromAPIVXC(ctx context.Context, v *megaport.VXC, tags map[string]string, plan *vxcResourceModel) diag.Diagnostics {
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
	// megalith only sets up billing once the order is fully approved, so a VXC
	// still awaiting approval reports the default 1-month term. Keep the
	// configured value until the order is approved, or Terraform rejects the
	// apply as an inconsistent result.
	if !vxcOrderPendingApproval(v.VXCApproval) || orm.ContractTermMonths.IsNull() {
		orm.ContractTermMonths = types.Int64Value(int64(v.ContractTermMonths))
	}
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
	// Neither end has a requested_product_uid until state or plan actually
	// supplies one. An import supplies neither, so it stays null: recording
	// the port the order landed on would pin a cloud end to it, because
	// ModifyPlan holds a cloud end at the value state already carries. A
	// managed refresh must not collapse this with a cloud end that legitimately
	// never requested a port, which state already holds as an empty string.
	aEndRequestedProductUIDNull, bEndRequestedProductUIDNull := true, true

	// First, try to get values from existing state
	if !orm.AEndConfiguration.IsNull() {
		existingAEnd := &vxcEndConfigurationModel{}
		aEndDiags := orm.AEndConfiguration.As(ctx, existingAEnd, basetypes.ObjectAsOptions{})
		apiDiags = append(apiDiags, aEndDiags...)
		if !existingAEnd.RequestedProductUID.IsNull() {
			aEndRequestedProductUID = existingAEnd.RequestedProductUID.ValueString()
			aEndRequestedProductUIDNull = false
		}
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

		if aEndRequestedProductUIDNull && !planAEnd.RequestedProductUID.IsNull() {
			aEndRequestedProductUID = planAEnd.RequestedProductUID.ValueString()
			aEndRequestedProductUIDNull = false
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

	aEndRequestedProductUIDValue := types.StringNull()
	if !aEndRequestedProductUIDNull {
		aEndRequestedProductUIDValue = types.StringValue(aEndRequestedProductUID)
	}

	aEndModel := &vxcEndConfigurationModel{
		OwnerUID:              types.StringValue(v.AEndConfiguration.OwnerUID),
		RequestedProductUID:   aEndRequestedProductUIDValue,
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
		if !existingBEnd.RequestedProductUID.IsNull() {
			bEndRequestedProductUID = existingBEnd.RequestedProductUID.ValueString()
			bEndRequestedProductUIDNull = false
		}
	}

	// If plan is provided and state values are empty, use plan values for B-End.
	if plan != nil && !plan.BEndConfiguration.IsNull() {
		planBEnd := &vxcEndConfigurationModel{}
		planDiags := plan.BEndConfiguration.As(ctx, planBEnd, basetypes.ObjectAsOptions{})
		apiDiags = append(apiDiags, planDiags...)

		if bEndRequestedProductUIDNull && !planBEnd.RequestedProductUID.IsNull() {
			bEndRequestedProductUID = planBEnd.RequestedProductUID.ValueString()
			bEndRequestedProductUIDNull = false
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

	bEndRequestedProductUIDValue := types.StringNull()
	if !bEndRequestedProductUIDNull {
		bEndRequestedProductUIDValue = types.StringValue(bEndRequestedProductUID)
	}

	bEndModel := &vxcEndConfigurationModel{
		OwnerUID:              types.StringValue(v.BEndConfiguration.OwnerUID),
		RequestedProductUID:   bEndRequestedProductUIDValue,
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

	// Preserve partner configs from plan if provided.
	// Partner configs are user-only values not returned by the API.
	if plan != nil {
		if !plan.AEndPartnerConfig.IsNull() {
			orm.AEndPartnerConfig = plan.AEndPartnerConfig
		}
		if !plan.BEndPartnerConfig.IsNull() {
			orm.BEndPartnerConfig = plan.BEndPartnerConfig
		}
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
		if !iface.DhcpPools.IsNull() && !iface.DhcpPools.IsUnknown() {
			pools := []*dhcpPoolModel{}
			poolDiags := iface.DhcpPools.ElementsAs(ctx, &pools, false)
			diags.Append(poolDiags...)
			for _, pool := range pools {
				poolToAppend := megaport.DhcpPoolConfig{
					Network:        pool.Network.ValueString(),
					StartIpAddress: pool.StartIPAddress.ValueString(),
					EndIpAddress:   pool.EndIPAddress.ValueString(),
					DefaultGateway: pool.DefaultGateway.ValueString(),
					Description:    pool.Description.ValueString(),
				}
				if !pool.DNSServers.IsNull() && !pool.DNSServers.IsUnknown() {
					dnsServers := []string{}
					dnsDiags := pool.DNSServers.ElementsAs(ctx, &dnsServers, true)
					diags.Append(dnsDiags...)
					poolToAppend.DnsServers = dnsServers
				}
				toAppend.DhcpPools = append(toAppend.DhcpPools, poolToAppend)
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
	ifaceModels := []*vxcPartnerConfigAEndInterfaceModel{}
	ifaceDiags := partnerConfigAEndModel.Interfaces.ElementsAs(ctx, &ifaceModels, true)
	diags.Append(ifaceDiags...)
	for _, iface := range ifaceModels {
		toAppend := megaport.PartnerConfigInterface{}
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
			bgpConnections := []*aEndBgpConnectionConfigModel{}
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

// fillBEndPartnerConfigOnImport records b_end_partner_config from the B-End CSP
// connection: transit, AWS, AWS hosted connection, Azure, Google, or Oracle. It
// records the settings a configuration has to carry, and leaves the ones the
// cloud assigns null: recording those would clash with a configuration that
// omits them, and the update check treats that as a change it cannot send. One
// "b_csp_connection" the import can read fills the block, more than one is left
// for the user.
func (orm *vxcResourceModel) fillBEndPartnerConfigOnImport(ctx context.Context, v *megaport.VXC) diag.Diagnostics {
	var diags diag.Diagnostics
	if !orm.BEndPartnerConfig.IsNull() || v.Resources == nil || v.Resources.CSPConnection == nil {
		return diags
	}
	var matches []megaport.CSPConnectionConfig
	for _, c := range v.Resources.CSPConnection.CSPConnection {
		if bEndImportCSPConnection(c) {
			matches = append(matches, c)
		}
	}
	const notRecorded = "b_end_partner_config not recorded on import"
	switch len(matches) {
	case 0:
		if bEndCSPConnectionPresent(v) {
			diags.AddWarning(
				notRecorded,
				"The provider cannot rebuild b_end_partner_config from the B-End connection this VXC uses. Add the block to the configuration by hand.",
			)
		}
		return diags
	case 1:
	default:
		diags.AddWarning(
			notRecorded,
			fmt.Sprintf("The VXC has %d B-End connections the import can read, so the provider cannot tell which one to record. Add b_end_partner_config to the configuration by hand.", len(matches)),
		)
		return diags
	}

	// A caller without permission on the B-End gets the connection stripped back
	// to its resource name and connect type, which would rebuild a block holding
	// nothing but the partner name.
	if !bEndImportCarriesSettings(matches[0]) {
		diags.AddWarning(
			notRecorded,
			"The read of the B-End connection carries none of the settings b_end_partner_config needs. Add the block to the configuration by hand.",
		)
		return diags
	}

	const summary = "Import complete, some settings need adding by hand"
	const recordsOnNextApply = "Setting one records the value in Terraform state on the next apply, which does not change the live VXC."
	var partnerDiags diag.Diagnostics
	var partnerObj basetypes.ObjectValue
	switch conn := matches[0].(type) {
	case megaport.CSPConnectionTransit:
		partnerDiags, _, partnerObj = createTransitPartnerConfig(ctx)
	case megaport.CSPConnectionAWS:
		partnerDiags, _, partnerObj = createAWSPartnerConfig(ctx, vxcPartnerConfigAWSModel{
			ConnectType:    stringOrNull(conn.ConnectType),
			Type:           stringOrNull(conn.Type),
			OwnerAccount:   stringOrNull(conn.OwnerAccount),
			Prefixes:       stringOrNull(string(conn.Prefixes)),
			ConnectionName: stringOrNull(conn.Name),
		})
		diags.AddWarning(
			summary,
			"The import leaves aws_config.asn, aws_config.amazon_asn, aws_config.auth_key, aws_config.customer_ip_address, and aws_config.amazon_ip_address null in b_end_partner_config. AWS assigns those values when the order leaves them out, and the read cannot tell an assigned value from one the configuration set. "+recordsOnNextApply+" The import records aws_config.prefixes as the API reports it. A configuration that leaves it out fails the next apply, so copy the recorded value into the configuration.",
		)
	case megaport.CSPConnectionAWSHC:
		partnerDiags, _, partnerObj = createAWSPartnerConfig(ctx, vxcPartnerConfigAWSModel{
			ConnectType:    stringOrNull(conn.ConnectType),
			OwnerAccount:   stringOrNull(conn.OwnerAccount),
			ConnectionName: stringOrNull(conn.Name),
		})
		diags.AddWarning(
			summary,
			"The import leaves aws_config.type, aws_config.asn, aws_config.amazon_asn, aws_config.auth_key, aws_config.customer_ip_address, aws_config.amazon_ip_address, and aws_config.prefixes null in b_end_partner_config. The read of an AWS hosted connection carries none of them. "+recordsOnNextApply,
		)
	case megaport.CSPConnectionAzure:
		partnerDiags, _, partnerObj = createAzurePartnerConfig(ctx, vxcPartnerConfigAzureModel{
			ServiceKey: stringOrNull(conn.ServiceKey),
			Peers:      types.ListNull(types.ObjectType{AttrTypes: partnerOrderAzurePeeringConfigAttrs}),
		})
		diags.AddWarning(
			summary,
			"The import leaves azure_config.port_choice and azure_config.peers null in b_end_partner_config. Set azure_config.port_choice to the port the live service uses, primary or secondary. "+recordsOnNextApply,
		)
	case megaport.CSPConnectionGoogle:
		partnerDiags, _, partnerObj = createGooglePartnerConfig(ctx, vxcPartnerConfigGoogleModel{
			PairingKey: stringOrNull(conn.PairingKey),
		})
	case megaport.CSPConnectionOracle:
		partnerDiags, _, partnerObj = createOraclePartnerConfig(ctx, vxcPartnerConfigOracleModel{
			VirtualCircuitId: stringOrNull(conn.VirtualCircuitId),
		})
	default:
		return diags
	}
	diags.Append(partnerDiags...)
	if !diags.HasError() {
		orm.BEndPartnerConfig = partnerObj
	}
	return diags
}

// bEndImportCSPConnection reports whether c is a B-End connection the import
// rebuilds a partner config from.
func bEndImportCSPConnection(c megaport.CSPConnectionConfig) bool {
	switch conn := c.(type) {
	case megaport.CSPConnectionTransit:
		return conn.ResourceName == "b_csp_connection"
	case megaport.CSPConnectionAWS:
		return conn.ResourceName == "b_csp_connection"
	case megaport.CSPConnectionAWSHC:
		return conn.ResourceName == "b_csp_connection"
	case megaport.CSPConnectionAzure:
		return conn.ResourceName == "b_csp_connection"
	case megaport.CSPConnectionGoogle:
		return conn.ResourceName == "b_csp_connection"
	case megaport.CSPConnectionOracle:
		return conn.ResourceName == "b_csp_connection"
	}
	return false
}

// bEndCSPConnectionPresent reports whether the VXC has a B-End CSP connection at
// all, whatever its partner. It separates a VXC whose B-End the import cannot
// rebuild, such as IBM, from one that has no B-End CSP connection to rebuild.
func bEndCSPConnectionPresent(v *megaport.VXC) bool {
	for _, c := range v.Resources.CSPConnection.CSPConnection {
		if ibm, ok := c.(megaport.CSPConnectionIBM); ok && ibm.ResourceName == "b_csp_connection" {
			return true
		}
		if bEndImportCSPConnection(c) {
			return true
		}
	}
	return false
}

// bEndImportCarriesSettings reports whether c holds any setting the import
// records. A transit config carries none by design, so it always qualifies.
func bEndImportCarriesSettings(c megaport.CSPConnectionConfig) bool {
	switch conn := c.(type) {
	case megaport.CSPConnectionTransit:
		return true
	case megaport.CSPConnectionAWS:
		return conn.Type != "" || conn.OwnerAccount != "" || conn.Name != ""
	case megaport.CSPConnectionAWSHC:
		return conn.OwnerAccount != "" || conn.Name != ""
	case megaport.CSPConnectionAzure:
		return conn.ServiceKey != ""
	case megaport.CSPConnectionGoogle:
		return conn.PairingKey != ""
	case megaport.CSPConnectionOracle:
		return conn.VirtualCircuitId != ""
	}
	return false
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

// movesPort reports whether Update sends this end to a different port. A
// partner port that rotated under us (the planned UID is already the current
// one) is not a move, and a CSP end is never moved.
func movesPort(plan, state *vxcEndConfigurationModel, isCSP bool) bool {
	return !plan.RequestedProductUID.IsNull() &&
		!plan.RequestedProductUID.Equal(state.RequestedProductUID) &&
		!isCSP &&
		!plan.RequestedProductUID.Equal(state.CurrentProductUID)
}

// bEndCSPConnectType returns the connect type of the cloud or transit service
// on the B-End, or "" when the B-End is a Megaport product. NetAuto refuses a
// B-End VLAN change on every such connect type. State's b_csp_connection
// decides, and the planned partner config stands in when state has none.
func bEndCSPConnectType(ctx context.Context, stateCSPConnections types.List, planPartnerConfig types.Object, diags *diag.Diagnostics) string {
	var conns []cspConnectionModel
	if !stateCSPConnections.IsNull() && !stateCSPConnections.IsUnknown() {
		*diags = append(*diags, stateCSPConnections.ElementsAs(ctx, &conns, false)...)
	}
	for _, c := range conns {
		if c.ResourceName.ValueString() != "b_csp_connection" {
			continue
		}
		if c.ConnectType.ValueString() == "VROUTER" {
			return ""
		}
		return c.ConnectType.ValueString()
	}
	partner, csp := classifyPartner(ctx, planPartnerConfig, diags)
	if partner.IsUnknown() || !csp && partner.ValueString() != "transit" {
		return ""
	}
	return strings.ToUpper(partner.ValueString())
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

type vlanPreflightInput struct {
	svc         megaport.PortService
	end         string // "A-End" or "B-End", used in the error message
	productUID  string
	productType string
	orderedVLAN types.Int64
	// currentVLAN is what this end already holds, so pinning an API-allocated
	// VLAN is not mistaken for requesting a taken one. Null on create.
	currentVLAN types.Int64
	// hasPartnerConfig marks an end whose port Megaport picks, so the requested
	// UID may not be the port the order lands on.
	hasPartnerConfig bool
}

// vlanAvailabilityPreflight turns a taken VLAN into a clear error naming the end
// and the port, instead of the backend's "VLAN N not available on service <id>",
// an internal id the user cannot map to their config. Only an explicit "taken"
// answer is acted on: the API answers per port but VLANs are unique across a CSP
// capacity group, so "available" does not mean the order will be accepted.
func vlanAvailabilityPreflight(ctx context.Context, in vlanPreflightInput) diag.Diagnostics {
	var diags diag.Diagnostics

	if in.orderedVLAN.IsNull() || in.orderedVLAN.IsUnknown() {
		return diags
	}
	vlan := int(in.orderedVLAN.ValueInt64())
	// 0 auto-assigns and -1 is untagged, so neither pins a VLAN to check.
	if vlan <= 0 {
		return diags
	}
	// Asking to keep the VLAN this end already holds always reads as unavailable.
	if !in.currentVLAN.IsNull() && !in.currentVLAN.IsUnknown() && int(in.currentVLAN.ValueInt64()) == vlan {
		return diags
	}
	// The API may rotate a Partner Port to a sibling in the same location and
	// diversity zone, so the requested port's answer can be the wrong port's.
	if in.hasPartnerConfig {
		return diags
	}
	// MVE VLANs are scoped per vNIC and MCR/VRouter ends can dictate their own,
	// so a per-service answer there could wrongly block a valid order.
	if in.productUID == "" || !strings.EqualFold(in.productType, megaport.PRODUCT_MEGAPORT) {
		return diags
	}

	available, err := in.svc.CheckPortVLANAvailability(ctx, in.productUID, vlan)
	if err != nil {
		tflog.Debug(ctx, "VLAN availability preflight skipped", map[string]any{
			"end": in.end, "product_uid": in.productUID, "vlan": vlan, "error": err.Error(),
		})
		return diags
	}
	if !available {
		diags.AddError(
			fmt.Sprintf("VLAN %d is not available on the %s port", vlan, in.end),
			fmt.Sprintf("VLAN %d is already in use on %s port %s. Pick a different %s ordered_vlan, or set it to 0 to let Megaport allocate one.", vlan, in.end, in.productUID, in.end),
		)
	}
	return diags
}

// prefixFilterIDToName resolves a prefix filter list ID to its description.
// The second return is false when a set ID is missing from the map, which means
// the caller cannot rebuild the config faithfully.
func prefixFilterIDToName(id int, pflMap map[int]string) (basetypes.StringValue, bool) {
	if id == 0 {
		return types.StringNull(), true
	}
	if name, ok := pflMap[id]; ok {
		return types.StringValue(name), true
	}
	return types.StringNull(), false
}

// buildVrouterPartnerConfigFromAPI builds a vrouter partner config object from
// the CSP VirtualRouter data on a VXC read. It returns a null object when the
// API data cannot produce a faithful config, so the caller leaves state alone.
//
// Some attributes always stay null. The BGP password is deliberate: the API
// does return it, and writing it would persist a live MD5 key in plain text in
// state. megaportgo does not model the interface-level ip_mtu, vlan,
// description, interface_type, packet filters, IPsec tunnel options or DHCP
// pools, and the
// read never echoes permit_export_to or deny_export_to. The caller warns about
// those, because they are missing whether or not this rebuild runs. The
// interface bfd block is the one exception. megalith does not re-serialize it
// and NetAuto discards it. A warning would name a setting the API cannot
// accept.
func buildVrouterPartnerConfigFromAPI(ctx context.Context, vrConn megaport.CSPConnectionVirtualRouter, pflMap map[int]string) (basetypes.ObjectValue, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	if len(vrConn.Interfaces) == 0 {
		return types.ObjectNull(vxcPartnerConfigAttrs), diags
	}

	interfaceModels := make([]vxcPartnerConfigInterfaceModel, 0, len(vrConn.Interfaces))
	for _, apiIface := range vrConn.Interfaces {
		ifaceModel := vxcPartnerConfigInterfaceModel{
			IpMtu:              types.Int64Null(),
			VLAN:               types.Int64Null(),
			Description:        types.StringNull(),
			InterfaceType:      types.StringNull(),
			PacketFilterIn:     types.Int64Null(),
			PacketFilterOut:    types.Int64Null(),
			IpSecTunnelOptions: types.ObjectNull(ipSecTunnelOptionsAttrs),
			DhcpPools:          types.ListNull(types.ObjectType{}.WithAttributeTypes(dhcpPoolAttrs)),
			IPAddresses:        types.ListNull(types.StringType),
			NatIPAddresses:     types.ListNull(types.StringType),
			IPRoutes:           types.ListNull(types.ObjectType{}.WithAttributeTypes(ipRouteAttrs)),
			Bfd:                types.ObjectNull(bfdConfigAttrs),
			BgpConnections:     types.ListNull(types.ObjectType{}.WithAttributeTypes(bgpVrouterConnectionConfig)),
		}

		if len(apiIface.IPAddresses) > 0 {
			ipList, ipDiags := types.ListValueFrom(ctx, types.StringType, apiIface.IPAddresses)
			diags.Append(ipDiags...)
			ifaceModel.IPAddresses = ipList
		}

		if len(apiIface.NatIPAddresses) > 0 {
			natList, natDiags := types.ListValueFrom(ctx, types.StringType, apiIface.NatIPAddresses)
			diags.Append(natDiags...)
			ifaceModel.NatIPAddresses = natList
		}

		if len(apiIface.IPRoutes) > 0 {
			routeModels := make([]ipRouteModel, 0, len(apiIface.IPRoutes))
			for _, r := range apiIface.IPRoutes {
				routeModels = append(routeModels, ipRouteModel{
					Prefix:      types.StringValue(r.Prefix),
					Description: stringOrNull(r.Description),
					NextHop:     types.StringValue(r.NextHop),
				})
			}
			routeList, routeDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(ipRouteAttrs), routeModels)
			diags.Append(routeDiags...)
			ifaceModel.IPRoutes = routeList
		}

		if len(apiIface.BGPConnections) > 0 {
			bgpModels := make([]bgpConnectionConfigModel, 0, len(apiIface.BGPConnections))
			for _, apiBgp := range apiIface.BGPConnections {
				// The API omits every optional BGP attribute it has no value
				// for, and megaportgo decodes those to zero values. Writing a
				// zero value would make the user restate it in the
				// configuration, and peer_type "" fails its own validator.
				bgpModel := bgpConnectionConfigModel{
					PeerAsn:            types.Int64Value(int64(apiBgp.PeerAsn)),
					PeerType:           stringOrNull(apiBgp.PeerType),
					LocalIPAddress:     types.StringValue(apiBgp.LocalIpAddress),
					PeerIPAddress:      types.StringValue(apiBgp.PeerIpAddress),
					Password:           types.StringNull(),
					Shutdown:           boolOrNull(apiBgp.Shutdown),
					Description:        stringOrNull(apiBgp.Description),
					MedIn:              int64OrNull(apiBgp.MedIn),
					MedOut:             int64OrNull(apiBgp.MedOut),
					BfdEnabled:         boolOrNull(apiBgp.BfdEnabled),
					ExportPolicy:       stringOrNull(apiBgp.ExportPolicy),
					AsPathPrependCount: int64OrNull(apiBgp.AsPathPrependCount),
					LocalAsn:           types.Int64Null(),
					AsOverride:         types.BoolNull(),
					PermitExportTo:     types.ListNull(types.StringType),
					DenyExportTo:       types.ListNull(types.StringType),
				}
				if apiBgp.LocalAsn != nil {
					bgpModel.LocalAsn = types.Int64Value(int64(*apiBgp.LocalAsn))
				}
				if apiBgp.AsOverride != nil {
					bgpModel.AsOverride = types.BoolValue(*apiBgp.AsOverride)
				}
				if len(apiBgp.PermitExportTo) > 0 {
					permitList, permitDiags := types.ListValueFrom(ctx, types.StringType, apiBgp.PermitExportTo)
					diags.Append(permitDiags...)
					bgpModel.PermitExportTo = permitList
				}
				if len(apiBgp.DenyExportTo) > 0 {
					denyList, denyDiags := types.ListValueFrom(ctx, types.StringType, apiBgp.DenyExportTo)
					diags.Append(denyDiags...)
					bgpModel.DenyExportTo = denyList
				}

				// An unresolved ID would write a null filter over a live one, so
				// abandon the whole config rather than report a partial one.
				for _, pfl := range []struct {
					id   int
					name string
					dst  *basetypes.StringValue
				}{
					{apiBgp.ImportWhitelist, "import_whitelist", &bgpModel.ImportWhitelist},
					{apiBgp.ImportBlacklist, "import_blacklist", &bgpModel.ImportBlacklist},
					{apiBgp.ExportWhitelist, "export_whitelist", &bgpModel.ExportWhitelist},
					{apiBgp.ExportBlacklist, "export_blacklist", &bgpModel.ExportBlacklist},
				} {
					name, ok := prefixFilterIDToName(pfl.id, pflMap)
					if !ok {
						diags.AddWarning(
							"Could not resolve a BGP prefix filter list",
							fmt.Sprintf("The BGP connection to peer %s uses prefix filter list ID %d for %s, but no list with that ID exists on the endpoint. Terraform left the partner configuration out of state. Add it to the configuration by hand.", apiBgp.PeerIpAddress, pfl.id, pfl.name),
						)
						return types.ObjectNull(vxcPartnerConfigAttrs), diags
					}
					*pfl.dst = name
				}

				bgpModels = append(bgpModels, bgpModel)
			}
			bgpList, bgpDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(bgpVrouterConnectionConfig), bgpModels)
			diags.Append(bgpDiags...)
			ifaceModel.BgpConnections = bgpList
		}

		interfaceModels = append(interfaceModels, ifaceModel)
	}

	ifaceList, ifaceDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(vxcVrouterInterfaceAttrs), interfaceModels)
	diags.Append(ifaceDiags...)
	vrouterObj, vrouterDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, vxcPartnerConfigVrouterModel{Interfaces: ifaceList})
	diags.Append(vrouterDiags...)

	obj, objDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, &vxcPartnerConfigurationModel{
		Partner:              types.StringValue("vrouter"),
		VrouterPartnerConfig: vrouterObj,
		AWSPartnerConfig:     types.ObjectNull(vxcPartnerConfigAWSAttrs),
		AzurePartnerConfig:   types.ObjectNull(vxcPartnerConfigAzureAttrs),
		GooglePartnerConfig:  types.ObjectNull(vxcPartnerConfigGoogleAttrs),
		OraclePartnerConfig:  types.ObjectNull(vxcPartnerConfigOracleAttrs),
		IBMPartnerConfig:     types.ObjectNull(vxcPartnerConfigIbmAttrs),
		PartnerAEndConfig:    types.ObjectNull(vxcPartnerConfigAEndAttrs),
	})
	diags.Append(objDiags...)
	if diags.HasError() {
		return types.ObjectNull(vxcPartnerConfigAttrs), diags
	}
	return obj, diags
}

// boolOrNull and int64OrNull are stringOrNull for the other two scalar kinds: a
// zero value maps back to null, so an attribute the API omitted stays absent
// instead of arriving as a default.
func boolOrNull(v bool) basetypes.BoolValue {
	if !v {
		return types.BoolNull()
	}
	return types.BoolValue(v)
}

func int64OrNull(v int) basetypes.Int64Value {
	if v == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(int64(v))
}
