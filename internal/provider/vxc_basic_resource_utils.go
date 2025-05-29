package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	megaport "github.com/megaport/megaportgo"
)

// createVXCBasicEndConfiguration creates an endpoint configuration for a VXC (Virtual Cross Connect).
//
// This function processes the endpoint configuration data and constructs a proper VXCOrderEndpointConfiguration
// that can be used when creating or updating VXC connections. It handles special cases for different product
// types (MCR, MVE) and processes various partner configurations for cloud service providers.
//
// Parameters:
//   - ctx: The context for the operation
//   - name: Name of the VXC being created (for error messaging)
//   - rateLimit: The speed limit for the connection in Mbps
//   - c: The endpoint configuration model from Terraform
//   - partnerConfig: Partner configuration object (if connecting to a partner service like AWS, Azure, etc.)
//
// Returns:
//   - diag.Diagnostics: Any validation or processing errors
//   - megaport.VXCOrderEndpointConfiguration: The configured endpoint ready for the API
//   - basetypes.ObjectValue: The processed partner configuration
//
// The function performs several key operations:
//  1. Validates product-specific requirements (e.g., MVE requires NetworkInterfaceIndex)
//  2. Sets VLAN configuration based on product type
//  3. Processes inner VLAN and network interface settings
//  4. Handles partner-specific configurations (AWS, Azure, Google, Oracle, IBM, etc.)
func (r *vxcBasicResource) createVXCBasicEndConfiguration(ctx context.Context, name string, rateLimit int, c vxcBasicEndConfigurationModel, partnerConfig basetypes.ObjectValue) (diag.Diagnostics, megaport.VXCOrderEndpointConfiguration, basetypes.ObjectValue) {
	diags := diag.Diagnostics{}
	endConfig := megaport.VXCOrderEndpointConfiguration{
		ProductUID: c.RequestedProductUID.ValueString(),
	}

	partnerObj := basetypes.NewObjectNull(vxcBasicPartnerConfigAttrs)

	productType, _ := r.client.ProductService.GetProductType(ctx, c.RequestedProductUID.ValueString())
	if strings.EqualFold(productType, megaport.PRODUCT_MCR) || strings.EqualFold(productType, megaport.PRODUCT_MVE) {

		if strings.EqualFold(productType, megaport.PRODUCT_MVE) {
			if c.NetworkInterfaceIndex.IsNull() || c.NetworkInterfaceIndex.IsUnknown() {
				diags.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+name+": Network Interface Index is required for MVE products",
				)
			}
		}
		if !c.VLAN.IsUnknown() {
			diags.AddError(
				"Error creating VXC",
				fmt.Sprintf("Cannot specify VLAN for product type %s (UID: %s). MCR and MVE products don't support VLAN specification.",
					productType, c.RequestedProductUID.ValueString()),
			)
		}

		if strings.EqualFold(productType, megaport.PRODUCT_MCR) {
			endConfig.VLAN = 0
		} else if strings.EqualFold(productType, megaport.PRODUCT_MVE) {
			if c.NetworkInterfaceIndex.IsNull() && c.NetworkInterfaceIndex.IsUnknown() {
				diags.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+name+": Network Interface Index is required for MVE products",
				)
			}
			endConfig.VLAN = 0
		}
	} else {
		if c.VLAN.IsNull() {
			endConfig.VLAN = -1
		} else {
			endConfig.VLAN = int(c.VLAN.ValueInt64())
		}
	}

	if !c.InnerVLAN.IsNull() || !c.NetworkInterfaceIndex.IsNull() {
		vxcOrderMVEConfig := &megaport.VXCOrderMVEConfig{}
		if c.InnerVLAN.IsNull() {
			// If inner_vlan is explicitly set to null, convert to -1 for the API
			// This will cause the API to return null, effectively removing the inner_vlan
			vxcOrderMVEConfig.InnerVLAN = -1
		} else if !c.InnerVLAN.IsUnknown() {
			innerVLANValue := c.InnerVLAN.ValueInt64()
			if innerVLANValue <= 0 {
				diags.AddError(
					"Error creating VXC",
					fmt.Sprintf("Invalid inner_vlan value: %d. inner_vlan must be between 2 and 4093. Auto-assignment (0) and untagged (-1) are not supported in Basic VXC.", innerVLANValue),
				)
			} else {
				// Only set inner_vlan for valid values
				vxcOrderMVEConfig.InnerVLAN = int(innerVLANValue)
			}
		}
		if !c.NetworkInterfaceIndex.IsNull() {
			vxcOrderMVEConfig.NetworkInterfaceIndex = int(c.NetworkInterfaceIndex.ValueInt64())
		}
		endConfig.VXCOrderMVEConfig = vxcOrderMVEConfig
	}

	if !partnerConfig.IsNull() {
		var partnerConfigModel vxcBasicPartnerConfigurationModel
		aPartnerDiags := partnerConfig.As(ctx, &partnerConfigModel, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})
		diags.Append(aPartnerDiags...)
		switch partnerConfigModel.Partner.ValueString() {
		case "aws":
			if partnerConfigModel.AWSPartnerConfig.IsNull() {
				diags.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+name+": AWS Partner configuration is required",
				)
			}
			var awsConfig vxcPartnerConfigAWSModel
			awsDiags := partnerConfigModel.AWSPartnerConfig.As(ctx, &awsConfig, basetypes.ObjectAsOptions{})
			if awsDiags.HasError() {
				diags.Append(awsDiags...)
			}
			awsDiags, partnerConfig, partnerConfigObj := createBasicVXCAWSPartnerConfig(ctx, awsConfig)
			if awsDiags.HasError() {
				diags.Append(awsDiags...)
			}
			partnerObj = partnerConfigObj
			endConfig.PartnerConfig = partnerConfig
		case "azure":
			if partnerConfigModel.AzurePartnerConfig.IsNull() {
				diags.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+name+": Azure Partner configuration is required",
				)
			}
			var azureConfig vxcPartnerConfigAzureModel
			azureDiags := partnerConfigModel.AzurePartnerConfig.As(ctx, &azureConfig, basetypes.ObjectAsOptions{})
			if azureDiags.HasError() {
				diags.Append(azureDiags...)
			}
			azureDiags, partnerConfig, partnerConfigObj := createBasicVXCAzurePartnerConfig(ctx, azureConfig)
			diags.Append(azureDiags...)
			endConfig.PartnerConfig = partnerConfig
			partnerPortReq := &megaport.ListPartnerPortsRequest{
				Key:     azureConfig.ServiceKey.ValueString(),
				Partner: "AZURE",
			}
			partnerPortRes, err := r.client.VXCService.ListPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				diags.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", name, err.Error()),
				)
			}
			// find primary or secondary port
			for _, port := range partnerPortRes.Data.Megaports {
				p := &port
				if p.Type == azureConfig.PortChoice.ValueString() {
					endConfig.ProductUID = p.ProductUID
				}
			}
			if endConfig.ProductUID == "" {
				diags.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not find azure port with type: %s", azureConfig.PortChoice.ValueString()),
				)
			}

			megaportPartnerConfig := megaport.VXCPartnerConfigAzure{
				ConnectType: "AZURE",
				ServiceKey:  azureConfig.ServiceKey.ValueString(),
			}

			azurePeerModels := []partnerOrderAzurePeeringConfigModel{}
			azurePeerDiags := azureConfig.Peers.ElementsAs(ctx, &azurePeerModels, false)
			diags.Append(azurePeerDiags...)
			if len(azurePeerModels) > 0 {
				megaportPartnerConfig.Peers = []megaport.PartnerOrderAzurePeeringConfig{}
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
					megaportPartnerConfig.Peers = append(megaportPartnerConfig.Peers, peeringConfig)
				}
			}

			partnerObj = partnerConfigObj
			endConfig.PartnerConfig = megaportPartnerConfig
		case "google":
			if partnerConfigModel.GooglePartnerConfig.IsNull() {
				diags.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+name+": Google Partner configuration is required",
				)
			}
			var googleConfig vxcPartnerConfigGoogleModel
			googleDiags := partnerConfigModel.GooglePartnerConfig.As(ctx, &googleConfig, basetypes.ObjectAsOptions{})
			diags.Append(googleDiags...)
			googleDiags, googlePartnerConfig, partnerConfigObj := createBasicVXCGooglePartnerConfig(ctx, googleConfig)
			diags.Append(googleDiags...)
			partnerPortReq := &megaport.LookupPartnerPortsRequest{
				Key:       googleConfig.PairingKey.ValueString(),
				PortSpeed: rateLimit,
				Partner:   "GOOGLE",
			}
			partnerPortReq.ProductID = c.RequestedProductUID.ValueString()
			partnerPortRes, err := r.client.VXCService.LookupPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				diags.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", name, err.Error()),
				)
			}
			endConfig.ProductUID = partnerPortRes.ProductUID

			partnerObj = partnerConfigObj
			endConfig.PartnerConfig = googlePartnerConfig
		case "oracle":
			if partnerConfigModel.OraclePartnerConfig.IsNull() {
				diags.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+name+": Oracle Partner configuration is required",
				)
			}
			var oracleConfig vxcPartnerConfigOracleModel
			oracleDiags := partnerConfigModel.OraclePartnerConfig.As(ctx, &oracleConfig, basetypes.ObjectAsOptions{})
			diags.Append(oracleDiags...)
			oracleDiags, oraclePartnerConfig, partnerConfigObj := createBasicVXCOraclePartnerConfig(ctx, oracleConfig)
			diags.Append(oracleDiags...)

			partnerPortReq := &megaport.LookupPartnerPortsRequest{
				Key:       oracleConfig.VirtualCircuitId.ValueString(),
				PortSpeed: rateLimit,
				Partner:   "ORACLE",
			}
			partnerPortReq.ProductID = c.RequestedProductUID.ValueString()

			partnerPortRes, err := r.client.VXCService.LookupPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				diags.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", name, err.Error()),
				)
			}
			endConfig.ProductUID = partnerPortRes.ProductUID
			partnerObj = partnerConfigObj
			endConfig.PartnerConfig = oraclePartnerConfig
		case "ibm":
			if partnerConfigModel.IBMPartnerConfig.IsNull() {
				diags.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+name+": IBM Partner configuration is required",
				)
			}
			var ibmConfig vxcPartnerConfigIbmModel
			ibmDiags := partnerConfigModel.IBMPartnerConfig.As(ctx, &ibmConfig, basetypes.ObjectAsOptions{})
			diags.Append(ibmDiags...)

			ibmDiags, ibmPartnerConfig, partnerConfigObj := createBasicVXCIBMPartnerConfig(ctx, ibmConfig)
			diags.Append(ibmDiags...)
			partnerObj = partnerConfigObj
			endConfig.PartnerConfig = ibmPartnerConfig
		case "mcr":
			if partnerConfigModel.MCRPartnerConfig.IsNull() {
				diags.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+name+": Virtual router configuration is required",
				)
			}
			var partnerConfigMCR vxcPartnerConfigVrouterModel
			endDiags := partnerConfigModel.MCRPartnerConfig.As(ctx, &partnerConfigMCR, basetypes.ObjectAsOptions{})
			diags.Append(endDiags...)
			prefixFilterList, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, c.RequestedProductUID.ValueString())
			if err != nil {
				diags.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+name+": "+err.Error(),
				)
			}

			vrouterDiags, vrouterMegaportConfig, partnerConfigObj := createBasicVXCMCRPartnerConfig(ctx, partnerConfigMCR, prefixFilterList)
			if vrouterDiags.HasError() {
				diags.Append(vrouterDiags...)
			}
			partnerObj = partnerConfigObj
			endConfig.PartnerConfig = vrouterMegaportConfig
		case "transit":
			transitDiags, transitPartnerConfig, partnerConfigObj := createBasicVXCTransitPartnerConfig(ctx)
			diags.Append(transitDiags...)
			partnerObj = partnerConfigObj
			endConfig.PartnerConfig = transitPartnerConfig
		default:
			diags.AddError(
				"Error creating VXC",
				"Could not create VXC with name "+name+": Partner configuration not supported",
			)
		}
	}

	return diags, endConfig, partnerObj
}

// modifyPlanBasicEndConfig modifies the Terraform plan for an endpoint configuration to handle
// special cases, particularly for cloud service provider endpoints where Megaport automatically
// manages port assignments.
//
// Parameters:
//   - ctx: The context for the operation
//   - endPlanObj: The plan object for the endpoint configuration
//   - endStateObj: The current state object for the endpoint configuration
//   - planPartner: The plan object for the partner configuration
//   - statePartner: The current state object for the partner configuration
//
// Returns:
//   - diag.Diagnostics: Any validation or processing errors
//   - basetypes.ObjectValue: The modified plan object for the endpoint
//   - basetypes.ObjectValue: The modified state object for the endpoint
//   - basetypes.ObjectValue: The modified state object for the partner
//   - path.Paths: Any attributes that require resource replacement if changed
//
// The function handles several important scenarios:
//  1. Detects cloud service provider (CSP) endpoints and manages port UIDs appropriately
//  2. Issues warnings when cloud provider port mappings are detected
//  3. Determines which changes require resource replacement
//  4. Synchronizes requested and current product UIDs for consistency
func modifyPlanBasicEndConfig(ctx context.Context, endPlanObj basetypes.ObjectValue, endStateObj basetypes.ObjectValue, planPartner basetypes.ObjectValue, statePartner basetypes.ObjectValue) (diag.Diagnostics, basetypes.ObjectValue, basetypes.ObjectValue, basetypes.ObjectValue, path.Paths) {
	diags := diag.Diagnostics{}
	requiresReplace := path.Paths{}

	var endCSP bool
	endStateConfig := &vxcBasicEndConfigurationModel{}
	endDiags := endStateObj.As(ctx, endStateConfig, basetypes.ObjectAsOptions{})
	diags = append(diags, endDiags...)
	endPlanConfig := &vxcBasicEndConfigurationModel{}
	partnerConfigModel := &vxcBasicPartnerConfigurationModel{}
	endDiags = endPlanObj.As(ctx, endPlanConfig, basetypes.ObjectAsOptions{})
	diags = append(diags, endDiags...)
	partnerConfigDiags := planPartner.As(ctx, &partnerConfigModel, basetypes.ObjectAsOptions{})
	diags = append(diags, partnerConfigDiags...)
	if !planPartner.IsNull() {
		if !partnerConfigModel.Partner.IsNull() {
			if partnerConfigModel.Partner.ValueString() != "transit" && partnerConfigModel.Partner.ValueString() != "mcr" && partnerConfigModel.Partner.ValueString() != "a-end" {
				endCSP = true
			}
		}
	}
	if statePartner.IsNull() {
		if !planPartner.IsNull() {
			statePartner = planPartner
		} else {
			statePartner = types.ObjectNull(vxcBasicPartnerConfigAttrs)
		}
	} else {
		if !planPartner.Equal(statePartner) && endCSP {
			requiresReplace = append(requiresReplace, path.Root("a_end_partner_config"))
		}
	}

	if endStateConfig.RequestedProductUID.IsNull() {
		if endPlanConfig.RequestedProductUID.IsNull() {
			endStateConfig.RequestedProductUID = endStateConfig.CurrentProductUID
			endPlanConfig.RequestedProductUID = endStateConfig.CurrentProductUID
		} else {
			endStateConfig.RequestedProductUID = endPlanConfig.RequestedProductUID
		}
	} else if endCSP {
		if !endPlanConfig.RequestedProductUID.IsNull() && !endPlanConfig.RequestedProductUID.Equal(endStateConfig.RequestedProductUID) {
			diags.AddWarning(
				"Cloud provider port mapping detected",
				fmt.Sprintf("Different A-End Product UIDs detected for cloud provider endpoint: requested=%s, actual=%s. This is normal - Megaport automatically manages cloud connection port assignments. Your configuration remains unchanged while the connection uses the provider-assigned Product UID. No action needed.",
					endPlanConfig.RequestedProductUID.ValueString(),
					endStateConfig.CurrentProductUID.ValueString()),
			)
		}
		endPlanConfig.RequestedProductUID = endStateConfig.RequestedProductUID
	}

	newPlanEndObj, endDiags := types.ObjectValueFrom(ctx, vxcBasicEndConfigurationAttrs, endPlanConfig)
	diags.Append(endDiags...)
	newStateEndObj, endDiags := types.ObjectValueFrom(ctx, vxcBasicEndConfigurationAttrs, endStateConfig)
	diags.Append(endDiags...)

	return diags, newPlanEndObj, newStateEndObj, statePartner, requiresReplace
}

// makeUpdateEndConfig prepares configurations for updating an endpoint, determining which
// attributes need to be updated and handling partner configuration changes.
//
// Parameters:
//   - ctx: The context for the operation
//   - name: Name of the VXC (for error messaging)
//   - planEndConfig: The planned endpoint configuration
//   - stateEndConfig: The current state endpoint configuration
//   - planPartnerConfig: The planned partner configuration
//   - statePartnerConfig: The current state partner configuration
//
// Returns:
//   - diag.Diagnostics: Any validation or processing errors
//   - basetypes.ObjectValue: The updated state object for the endpoint
//   - basetypes.ObjectValue: The updated partner configuration object
//   - megaport.VXCPartnerConfiguration: The partner configuration for the API
//   - *string: The requested product UID if it needs to be updated (nil if no change)
//   - *int: The VLAN if it needs to be updated (nil if no change)
//   - *int: The inner VLAN if it needs to be updated (nil if no change)
//   - *int: The vNIC index if it needs to be updated (nil if no change)
//   - bool: Whether the endpoint is a cloud service provider endpoint
//
// This function:
//  1. Detects changes between plan and state configurations
//  2. Determines which attributes need updates
//  3. Handles special cases for cloud service provider endpoints
//  4. Prepares partner configurations for update operations
//  5. Returns the necessary parameters for the VXC update operation
func (r *vxcBasicResource) makeUpdateEndConfig(ctx context.Context, name string, planEndConfig, stateEndConfig, planPartnerConfig, statePartnerConfig basetypes.ObjectValue) (diag.Diagnostics, basetypes.ObjectValue, basetypes.ObjectValue, megaport.VXCPartnerConfiguration, *string, *int, *int, *int, bool) {
	diags := diag.Diagnostics{}

	var partnerChange bool

	partnerObj := basetypes.NewObjectNull(vxcBasicPartnerConfigAttrs)

	var megaportPartnerConfig megaport.VXCPartnerConfiguration

	// If Imported, AEndPartnerConfig will be null. Set the partner config to the existing one in the plan.
	if !planPartnerConfig.Equal(statePartnerConfig) {
		partnerChange = true
	}
	if statePartnerConfig.IsNull() {
		statePartnerConfig = planPartnerConfig
	}

	var endPlanModel, endStateModel *vxcBasicEndConfigurationModel
	var partnerPlan, partnerState *vxcBasicPartnerConfigurationModel

	// Check if partner config is a CSP partner
	var endCSP, isMCRPartnerConfig bool

	endPlanDiags := planEndConfig.As(ctx, &endPlanModel, basetypes.ObjectAsOptions{})
	diags.Append(endPlanDiags...)

	stateDiags := stateEndConfig.As(ctx, &endStateModel, basetypes.ObjectAsOptions{})
	diags.Append(stateDiags...)

	partnerPlanDiags := planPartnerConfig.As(ctx, &partnerPlan, basetypes.ObjectAsOptions{})
	diags.Append(partnerPlanDiags...)

	partnerStateDiags := statePartnerConfig.As(ctx, &partnerState, basetypes.ObjectAsOptions{})
	diags.Append(partnerStateDiags...)

	if !planPartnerConfig.IsNull() {
		if !partnerPlan.Partner.IsNull() {
			if partnerPlan.Partner.ValueString() != "a-end" && partnerPlan.Partner.ValueString() != "mcr" && partnerPlan.Partner.ValueString() != "transit" {
				endCSP = true
			} else if partnerPlan.Partner.ValueString() == "mcr" {
				isMCRPartnerConfig = true
			}
		}
	}
	var endVLAN, endInnerVLAN, vNicIndex *int

	var requestedProductUID *string

	if !endPlanModel.RequestedProductUID.IsNull() && !endPlanModel.RequestedProductUID.Equal(endStateModel.RequestedProductUID) {
		// Do not update the product UID if the partner is a CSP
		if !endCSP {
			requestedProductUID = megaport.PtrTo(endPlanModel.RequestedProductUID.ValueString())
			endStateModel.RequestedProductUID = endPlanModel.RequestedProductUID
		}
	}

	if !endPlanModel.NetworkInterfaceIndex.IsUnknown() && !endPlanModel.NetworkInterfaceIndex.Equal(endStateModel.NetworkInterfaceIndex) && !endStateModel.NetworkInterfaceIndex.IsNull() {
		if endPlanModel.NetworkInterfaceIndex.IsNull() {
			// Result in an error because it must not be null for MVE VXCs
			diags.AddError(
				"Error updating VXC",
				fmt.Sprintf("Network Interface Index cannot be null for VXC with name %s. It must be specified for MVE products.", name),
			)
		} else {
			vNicIndex = megaport.PtrTo(int(endPlanModel.NetworkInterfaceIndex.ValueInt64()))
			endStateModel.NetworkInterfaceIndex = endPlanModel.NetworkInterfaceIndex
		}
	}

	// Check for attempt to update VLAN for MCR or MVE products
	if !endPlanModel.VLAN.IsUnknown() && !endPlanModel.VLAN.Equal(endStateModel.VLAN) {
		if endPlanModel.VLAN.IsNull() {
			// Only convert to -1 if the prior state value was actually set (not null or unknown)
			// This prevents sending untag command when the field was simply omitted
			if !endStateModel.VLAN.IsNull() && !endStateModel.VLAN.IsUnknown() {
				// If VLAN is explicitly set to null and had a prior value, convert to -1 for the API
				// This will cause the API to return null, effectively removing the VLAN
				endVLAN = megaport.PtrTo(-1)
			}
		} else {
			// Check if End Product is MCR of MVE
			productType, err := r.client.ProductService.GetProductType(ctx, endPlanModel.RequestedProductUID.ValueString())
			if err == nil && (strings.EqualFold(productType, megaport.PRODUCT_MCR) || strings.EqualFold(productType, megaport.PRODUCT_MVE)) {
				diags.AddError(
					"Error updating VXC",
					fmt.Sprintf("Cannot update VLAN for product type %s (UID: %s). MCR and MVE products don't support VLAN specification.",
						productType, endPlanModel.RequestedProductUID.ValueString()),
				)
			}

			endVLAN = megaport.PtrTo(int(endPlanModel.VLAN.ValueInt64()))
		}
	}
	endStateModel.VLAN = endPlanModel.VLAN

	if !endPlanModel.InnerVLAN.IsUnknown() && !endPlanModel.InnerVLAN.Equal(endStateModel.InnerVLAN) {
		if endPlanModel.InnerVLAN.IsNull() {
			// Only convert to -1 if the prior state value was actually set (not null or unknown)
			// This prevents sending untag command when the field was simply omitted
			if !endStateModel.InnerVLAN.IsNull() && !endStateModel.InnerVLAN.IsUnknown() {
				// If inner_vlan is explicitly set to null and had a prior value, convert to -1 for the API
				// This will cause the API to return null, effectively removing the inner_vlan
				if !isMCRPartnerConfig {
					endInnerVLAN = megaport.PtrTo(-1)
				}
			}
			// Otherwise, don't send any update for innerVLAN (keep endInnerVLAN as nil)
		} else {
			// Check for invalid values
			innerVLANValue := endPlanModel.InnerVLAN.ValueInt64()
			if innerVLANValue <= 0 {
				diags.AddError(
					"Error updating VXC",
					fmt.Sprintf("Invalid inner_vlan value: %d. inner_vlan must be between 2 and 4093. Auto-assignment (0) and untagged (-1) are not supported in Basic VXC.", innerVLANValue),
				)
			}
			endInnerVLAN = megaport.PtrTo(int(innerVLANValue))
		}
	}
	endStateModel.InnerVLAN = endPlanModel.InnerVLAN

	if !planPartnerConfig.IsNull() && partnerChange {
		if endCSP {
			// For CSP partners (AWS, Azure, Google, etc.), preserve the partner config from the plan
			partnerObj = planPartnerConfig
		} else {
			switch partnerPlan.Partner.ValueString() {
			case "transit":
				transitDiags, transitPartnerConfig, partnerConfigObj := createBasicVXCTransitPartnerConfig(ctx)
				diags.Append(transitDiags...)
				partnerObj = partnerConfigObj
				megaportPartnerConfig = transitPartnerConfig
			case "mcr":
				if partnerPlan.MCRPartnerConfig.IsNull() {
					diags.AddError(
						"Error updating VXC",
						"Could not update VXC with name "+name+": Virtual router configuration is required",
					)
				}
				var partnerConfigAEnd vxcPartnerConfigVrouterModel
				endDiags := partnerPlan.MCRPartnerConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
				diags.Append(endDiags...)
				prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, endStateModel.RequestedProductUID.ValueString())
				if err != nil {
					diags.AddError(
						"Error updating VXC",
						"Could not update VXC with name "+name+": "+err.Error(),
					)
				}
				mcrDiags, mcrPartnerConfig, partnerConfigObj := createBasicVXCMCRPartnerConfig(ctx, partnerConfigAEnd, prefixFilterListRes)
				diags.Append(mcrDiags...)
				partnerObj = partnerConfigObj
				megaportPartnerConfig = mcrPartnerConfig
			default:
				diags.AddError(
					"Error Updating VXC",
					"Could not update VXC with name "+name+": Partner configuration not supported",
				)
			}
		}
	}
	stateObj, stateDiags := types.ObjectValueFrom(ctx, vxcBasicEndConfigurationAttrs, endStateModel)
	diags.Append(stateDiags...)

	return diags, stateObj, partnerObj, megaportPartnerConfig, requestedProductUID, endVLAN, endInnerVLAN, vNicIndex, endCSP
}

func createBasicVXCAWSPartnerConfig(ctx context.Context, awsConfig vxcPartnerConfigAWSModel) (diag.Diagnostics, *megaport.VXCPartnerConfigAWS, basetypes.ObjectValue) {
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
	mcr := types.ObjectNull(vxcPartnerConfigMCRAttrs)
	ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
	partnerConfigModel := &vxcBasicPartnerConfigurationModel{
		Partner:             types.StringValue("aws"),
		AWSPartnerConfig:    awsConfigObj,
		AzurePartnerConfig:  azure,
		GooglePartnerConfig: google,
		OraclePartnerConfig: oracle,
		MCRPartnerConfig:    mcr,
		IBMPartnerConfig:    ibmPartner,
	}

	partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcBasicPartnerConfigAttrs, partnerConfigModel)
	diags.Append(partnerDiags...)

	return diags, partnerConfig, partnerConfigObj
}

func createBasicVXCAzurePartnerConfig(ctx context.Context, azureConfig vxcPartnerConfigAzureModel) (diag.Diagnostics, megaport.VXCPartnerConfigAzure, basetypes.ObjectValue) {
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
	mcr := types.ObjectNull(vxcPartnerConfigMCRAttrs)
	ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
	partnerConfigModel := &vxcBasicPartnerConfigurationModel{
		Partner:             types.StringValue("azure"),
		AWSPartnerConfig:    aws,
		AzurePartnerConfig:  azureConfigObj,
		GooglePartnerConfig: google,
		OraclePartnerConfig: oracle,
		MCRPartnerConfig:    mcr,
		IBMPartnerConfig:    ibmPartner,
	}

	partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcBasicPartnerConfigAttrs, partnerConfigModel)
	diags.Append(partnerDiags...)
	return diags, partnerConfig, partnerConfigObj
}

func createBasicVXCGooglePartnerConfig(ctx context.Context, googleConfig vxcPartnerConfigGoogleModel) (diag.Diagnostics, megaport.VXCPartnerConfigGoogle, basetypes.ObjectValue) {
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
	mcr := types.ObjectNull(vxcPartnerConfigMCRAttrs)
	ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
	aEndPartnerConfigModel := &vxcBasicPartnerConfigurationModel{
		Partner:             types.StringValue("google"),
		AWSPartnerConfig:    aws,
		AzurePartnerConfig:  azure,
		GooglePartnerConfig: googleConfigObj,
		OraclePartnerConfig: oracle,
		MCRPartnerConfig:    mcr,
		IBMPartnerConfig:    ibmPartner,
	}

	partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcBasicPartnerConfigAttrs, aEndPartnerConfigModel)
	diags.Append(partnerDiags...)
	return diags, googlePartnerConfig, partnerConfigObj
}

func createBasicVXCOraclePartnerConfig(ctx context.Context, oracleConfig vxcPartnerConfigOracleModel) (diag.Diagnostics, megaport.VXCPartnerConfigOracle, basetypes.ObjectValue) {
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
	mcr := types.ObjectNull(vxcPartnerConfigMCRAttrs)
	ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
	bEndPartnerConfigModel := &vxcBasicPartnerConfigurationModel{
		Partner:             types.StringValue("oracle"),
		AWSPartnerConfig:    aws,
		AzurePartnerConfig:  azure,
		GooglePartnerConfig: google,
		OraclePartnerConfig: oracleConfigObj,
		IBMPartnerConfig:    ibmPartner,
		MCRPartnerConfig:    mcr,
	}

	partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcBasicPartnerConfigAttrs, bEndPartnerConfigModel)
	diags.Append(partnerDiags...)
	return diags, oraclePartnerConfig, partnerConfigObj
}

func createBasicVXCIBMPartnerConfig(ctx context.Context, ibmConfig vxcPartnerConfigIbmModel) (diag.Diagnostics, megaport.VXCPartnerConfigIBM, basetypes.ObjectValue) {
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
	mcr := types.ObjectNull(vxcPartnerConfigMCRAttrs)
	ibmParnterConfigObj, ibmDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigIbmAttrs, ibmConfig)
	diags.Append(ibmDiags...)
	aEndPartnerConfigModel := &vxcBasicPartnerConfigurationModel{
		Partner:             types.StringValue("ibm"),
		AWSPartnerConfig:    aws,
		AzurePartnerConfig:  azure,
		GooglePartnerConfig: google,
		OraclePartnerConfig: oracle,
		MCRPartnerConfig:    mcr,
		IBMPartnerConfig:    ibmParnterConfigObj,
	}
	partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcBasicPartnerConfigAttrs, aEndPartnerConfigModel)
	diags.Append(partnerDiags...)
	return diags, ibmPartnerConfig, partnerConfigObj
}

func createBasicVXCMCRPartnerConfig(ctx context.Context, vrouterConfig vxcPartnerConfigVrouterModel, prefixFilterList []*megaport.PrefixFilterList) (diag.Diagnostics, *megaport.VXCOrderVrouterPartnerConfig, basetypes.ObjectValue) {
	diags := diag.Diagnostics{}
	mcrPartnerConfig := &megaport.VXCOrderVrouterPartnerConfig{}
	ifaceModels := []*vxcPartnerConfigInterfaceModel{}
	ifaceDiags := vrouterConfig.Interfaces.ElementsAs(ctx, &ifaceModels, false)
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
		mcrPartnerConfig.Interfaces = append(mcrPartnerConfig.Interfaces, toAppend)
	}
	mcrConfigObj, bEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigMCRAttrs, vrouterConfig)
	diags.Append(bEndDiags...)
	aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
	azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
	google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
	oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
	ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
	partnerConfigModel := &vxcBasicPartnerConfigurationModel{
		Partner:             types.StringValue("mcr"),
		AWSPartnerConfig:    aws,
		AzurePartnerConfig:  azure,
		GooglePartnerConfig: google,
		OraclePartnerConfig: oracle,
		IBMPartnerConfig:    ibmPartner,
		MCRPartnerConfig:    mcrConfigObj,
	}
	partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcBasicPartnerConfigAttrs, partnerConfigModel)
	diags.Append(partnerDiags...)
	return diags, mcrPartnerConfig, partnerConfigObj
}

func createBasicVXCTransitPartnerConfig(ctx context.Context) (diag.Diagnostics, megaport.VXCPartnerConfigTransit, basetypes.ObjectValue) {
	diags := diag.Diagnostics{}
	transitPartnerConfig := megaport.VXCPartnerConfigTransit{
		ConnectType: "TRANSIT",
	}

	aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
	azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
	google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
	oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
	mcr := types.ObjectNull(vxcPartnerConfigMCRAttrs)
	ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)

	transitPartnerConfigModel := &vxcBasicPartnerConfigurationModel{
		Partner:             types.StringValue("transit"),
		AWSPartnerConfig:    aws,
		AzurePartnerConfig:  azure,
		GooglePartnerConfig: google,
		OraclePartnerConfig: oracle,
		MCRPartnerConfig:    mcr,
		IBMPartnerConfig:    ibmPartner,
	}

	transitConfigObj, transitDiags := types.ObjectValueFrom(ctx, vxcBasicPartnerConfigAttrs, transitPartnerConfigModel)
	diags.Append(transitDiags...)

	return diags, transitPartnerConfig, transitConfigObj
}

// detectNullModifier creates a plan modifier that forces a change when value is explicitly set to null
func detectNullModifier() planmodifier.Int64 {
	return &detectNullInt64PlanModifier{}
}

// detectNullInt64PlanModifier implements the plan modifier
type detectNullInt64PlanModifier struct{}

// Description returns the description for the plan modifier
func (m *detectNullInt64PlanModifier) Description(ctx context.Context) string {
	return "Detects when a value is explicitly set to null"
}

// MarkdownDescription returns the markdown description for the plan modifier
func (m *detectNullInt64PlanModifier) MarkdownDescription(ctx context.Context) string {
	return "Detects when a value is explicitly set to null"
}

// PlanModifyInt64 implements the actual modification logic
func (m *detectNullInt64PlanModifier) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	// Only continue if there's a plan update
	if req.ConfigValue.IsNull() && !req.StateValue.IsNull() {
		// Force modified to true when config is null and state is not null
		resp.PlanValue = types.Int64Null()
	}
}
