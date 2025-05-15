package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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

	partnerObj := basetypes.NewObjectNull(vxcPartnerConfigAttrs)

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
		if !c.InnerVLAN.IsNull() {
			vxcOrderMVEConfig.InnerVLAN = int(c.InnerVLAN.ValueInt64())
		}
		if !c.NetworkInterfaceIndex.IsNull() {
			vxcOrderMVEConfig.NetworkInterfaceIndex = int(c.NetworkInterfaceIndex.ValueInt64())
		}
		endConfig.VXCOrderMVEConfig = vxcOrderMVEConfig
	}

	if !partnerConfig.IsNull() {
		var partnerConfigModel vxcPartnerConfigurationModel
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
			awsDiags, partnerConfig, partnerConfigObj := createAWSPartnerConfig(ctx, awsConfig)
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
			azureDiags, partnerConfig, partnerConfigObj := createAzurePartnerConfig(ctx, azureConfig)
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
			googleDiags, googlePartnerConfig, partnerConfigObj := createGooglePartnerConfig(ctx, googleConfig)
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
			oracleDiags, oraclePartnerConfig, partnerConfigObj := createOraclePartnerConfig(ctx, oracleConfig)
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

			ibmDiags, ibmPartnerConfig, partnerConfigObj := createIBMPartnerConfig(ctx, ibmConfig)
			diags.Append(ibmDiags...)
			partnerObj = partnerConfigObj
			endConfig.PartnerConfig = ibmPartnerConfig
		case "vrouter":
			if partnerConfigModel.VrouterPartnerConfig.IsNull() {
				diags.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+name+": Virtual router configuration is required",
				)
			}
			var partnerConfigAEnd vxcPartnerConfigVrouterModel
			endDiags := partnerConfigModel.VrouterPartnerConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
			diags.Append(endDiags...)
			prefixFilterList, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, c.RequestedProductUID.ValueString())
			if err != nil {
				diags.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+name+": "+err.Error(),
				)
			}

			vrouterDiags, vrouterMegaportConfig, partnerConfigObj := createVrouterPartnerConfig(ctx, partnerConfigAEnd, prefixFilterList)
			if vrouterDiags.HasError() {
				diags.Append(vrouterDiags...)
			}
			partnerObj = partnerConfigObj
			endConfig.PartnerConfig = vrouterMegaportConfig
		case "a-end":
			if partnerConfigModel.PartnerAEndConfig.IsNull() {
				diags.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+name+": A-End Partner configuration is required",
				)
			}
			var partnerConfigAEnd vxcPartnerConfigAEndModel
			endDiags := partnerConfigModel.PartnerAEndConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
			diags.Append(endDiags...)
			prefixFilterList, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, c.RequestedProductUID.ValueString())
			if err != nil {
				diags.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+name+": "+err.Error(),
				)
			}
			endDiags, aEndMegaportConfig, partnerConfigObj := createAEndPartnerConfig(ctx, partnerConfigAEnd, prefixFilterList)
			diags.Append(endDiags...)
			partnerObj = partnerConfigObj
			endConfig.PartnerConfig = aEndMegaportConfig
		case "transit":
			transitDiags, transitPartnerConfig, partnerConfigObj := createTransitPartnerConfig(ctx)
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
	partnerConfigModel := &vxcPartnerConfigurationModel{}
	endDiags = endPlanObj.As(ctx, endPlanConfig, basetypes.ObjectAsOptions{})
	diags = append(diags, endDiags...)
	partnerConfigDiags := planPartner.As(ctx, &partnerConfigModel, basetypes.ObjectAsOptions{})
	diags = append(diags, partnerConfigDiags...)
	if !planPartner.IsNull() {
		if !partnerConfigModel.Partner.IsNull() {
			if partnerConfigModel.Partner.ValueString() != "transit" && partnerConfigModel.Partner.ValueString() != "vrouter" && partnerConfigModel.Partner.ValueString() != "a-end" {
				endCSP = true
			}
		}
	}
	if statePartner.IsNull() {
		if !planPartner.IsNull() {
			statePartner = planPartner
		} else {
			statePartner = types.ObjectNull(vxcPartnerConfigAttrs)
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
//   - bool: Whether the endpoint is a cloud service provider endpoint
//
// This function:
//  1. Detects changes between plan and state configurations
//  2. Determines which attributes need updates
//  3. Handles special cases for cloud service provider endpoints
//  4. Prepares partner configurations for update operations
//  5. Returns the necessary parameters for the VXC update operation
func (r *vxcBasicResource) makeUpdateEndConfig(ctx context.Context, name string, planEndConfig, stateEndConfig, planPartnerConfig, statePartnerConfig basetypes.ObjectValue) (diag.Diagnostics, basetypes.ObjectValue, basetypes.ObjectValue, megaport.VXCPartnerConfiguration, *string, *int, *int, bool) {
	diags := diag.Diagnostics{}

	var partnerChange bool

	partnerObj := basetypes.NewObjectNull(vxcPartnerConfigAttrs)

	var megaportPartnerConfig megaport.VXCPartnerConfiguration

	// If Imported, AEndPartnerConfig will be null. Set the partner config to the existing one in the plan.
	if !planPartnerConfig.Equal(statePartnerConfig) {
		partnerChange = true
	}
	if statePartnerConfig.IsNull() {
		statePartnerConfig = planPartnerConfig
	}

	var endPlanModel, endStateModel *vxcBasicEndConfigurationModel
	var partnerPlan, partnerState *vxcPartnerConfigurationModel

	// Check if partner config is a CSP partner
	var endCSP bool

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
			if partnerPlan.Partner.ValueString() != "a-end" && partnerPlan.Partner.ValueString() != "vrouter" && partnerPlan.Partner.ValueString() != "transit" {
				endCSP = true
			}
		}
	}
	var endVLAN, endInnerVLAN *int

	var requestedProductUID *string

	if !endPlanModel.RequestedProductUID.IsNull() && !endPlanModel.RequestedProductUID.Equal(endStateModel.RequestedProductUID) {
		// Do not update the product UID if the partner is a CSP
		if !endCSP {
			requestedProductUID = megaport.PtrTo(endPlanModel.RequestedProductUID.ValueString())
			endStateModel.RequestedProductUID = endPlanModel.RequestedProductUID
		}
	}

	// Check for attempt to update VLAN for MCR or MVE products
	if !endPlanModel.VLAN.IsUnknown() && !endPlanModel.VLAN.IsNull() && !endPlanModel.VLAN.Equal(endStateModel.VLAN) {
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
	endStateModel.VLAN = endPlanModel.VLAN

	if !endPlanModel.InnerVLAN.IsUnknown() && !endPlanModel.InnerVLAN.IsNull() && !endPlanModel.InnerVLAN.Equal(endStateModel.InnerVLAN) {
		endInnerVLAN = megaport.PtrTo(int(endPlanModel.InnerVLAN.ValueInt64()))
	}
	endStateModel.InnerVLAN = endPlanModel.InnerVLAN

	if !planPartnerConfig.IsNull() && partnerChange {
		partnerConfig := partnerPlan
		if endCSP {
			// For CSP partners (AWS, Azure, Google, etc.), preserve the partner config from the plan
			partnerObj = planPartnerConfig
		} else {
			switch partnerPlan.Partner.ValueString() {
			case "transit":
				transitDiags, transitPartnerConfig, partnerConfigObj := createTransitPartnerConfig(ctx)
				diags.Append(transitDiags...)
				partnerObj = partnerConfigObj
				megaportPartnerConfig = transitPartnerConfig
			case "a-end":
				if partnerConfig.PartnerAEndConfig.IsNull() {
					diags.AddError(
						"Error updating VXC",
						"Could not update VXC with name "+name+": A-End Partner configuration is required",
					)
				}
				var partnerConfigAEnd vxcPartnerConfigAEndModel
				endDiags := partnerConfig.PartnerAEndConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
				diags.Append(endDiags...)
				prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, endPlanModel.RequestedProductUID.ValueString())
				if err != nil {
					diags.AddError(
						"Error updating VXC",
						"Could not update VXC with name "+name+": "+err.Error(),
					)
				}

				aEndDiags, aEndMegaportConfig, partnerConfigObj := createAEndPartnerConfig(ctx, partnerConfigAEnd, prefixFilterListRes)
				diags.Append(aEndDiags...)
				partnerObj = partnerConfigObj
				megaportPartnerConfig = aEndMegaportConfig
			case "vrouter":
				if partnerPlan.VrouterPartnerConfig.IsNull() {
					diags.AddError(
						"Error updating VXC",
						"Could not update VXC with name "+name+": Virtual router configuration is required",
					)
				}
				var partnerConfigAEnd vxcPartnerConfigVrouterModel
				endDiags := partnerPlan.VrouterPartnerConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
				diags.Append(endDiags...)
				prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, endStateModel.RequestedProductUID.ValueString())
				if err != nil {
					diags.AddError(
						"Error updating VXC",
						"Could not update VXC with name "+name+": "+err.Error(),
					)
				}
				vrouterDiags, vrouterPartnerConfig, partnerConfigObj := createVrouterPartnerConfig(ctx, partnerConfigAEnd, prefixFilterListRes)
				diags.Append(vrouterDiags...)
				partnerObj = partnerConfigObj
				megaportPartnerConfig = vrouterPartnerConfig
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

	return diags, stateObj, partnerObj, megaportPartnerConfig, requestedProductUID, endVLAN, endInnerVLAN, endCSP
}
