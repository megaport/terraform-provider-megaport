package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &vxcResource{}
	_ resource.ResourceWithConfigure   = &vxcResource{}
	_ resource.ResourceWithImportState = &vxcResource{}

	vxcEndConfigurationAttrs = map[string]attr.Type{
		"owner_uid":             types.StringType,
		"requested_product_uid": types.StringType,
		"current_product_uid":   types.StringType,
		"product_name":          types.StringType,
		"location_id":           types.Int64Type,
		"location":              types.StringType,
		"ordered_vlan":          types.Int64Type,
		"vlan":                  types.Int64Type,
		"inner_vlan":            types.Int64Type,
		"vnic_index":            types.Int64Type,
		"secondary_name":        types.StringType,
		"location_details":      types.ObjectType{}.WithAttributeTypes(productLocationDetailsAttrs),
	}

	cspConnectionFullAttrs = map[string]attr.Type{
		"connect_type":         types.StringType,
		"resource_name":        types.StringType,
		"resource_type":        types.StringType,
		"vlan":                 types.Int64Type,
		"account":              types.StringType,
		"amazon_address":       types.StringType,
		"asn":                  types.Int64Type,
		"auth_key":             types.StringType,
		"customer_address":     types.StringType,
		"customer_ip_address":  types.StringType,
		"customer_ip4_address": types.StringType,
		"id":                   types.Int64Type,
		"name":                 types.StringType,
		"owner_account":        types.StringType,
		"peer_asn":             types.Int64Type,
		"type":                 types.StringType,
		"vif_id":               types.StringType,
		"bandwidth":            types.Int64Type,
		"bandwidths":           types.ListType{}.WithElementType(types.Int64Type),
		"connection_id":        types.StringType,
		"managed":              types.BoolType,
		"service_key":          types.StringType,
		"csp_name":             types.StringType,
		"pairing_key":          types.StringType,
		"customer_ip6_network": types.StringType,
		"ipv4_gateway_address": types.StringType,
		"ipv6_gateway_address": types.StringType,
		"ip_addresses":         types.ListType{}.WithElementType(types.StringType),
		"virtual_router_name":  types.StringType,
	}

	virtualRouterAttrs = map[string]attr.Type{
		"mcr_asn":              types.Int64Type,
		"resource_name":        types.StringType,
		"resource_type":        types.StringType,
		"speed":                types.Int64Type,
		"bgp_shutdown_default": types.BoolType,
	}

	vllConfigAttrs = map[string]attr.Type{
		"a_vlan":          types.Int64Type,
		"b_vlan":          types.Int64Type,
		"description":     types.StringType,
		"id":              types.Int64Type,
		"name":            types.StringType,
		"rate_limit_mbps": types.Int64Type,
		"resource_name":   types.StringType,
		"resource_type":   types.StringType,
	}

	vxcApprovalAttrs = map[string]attr.Type{
		"status":    types.StringType,
		"message":   types.StringType,
		"uid":       types.StringType,
		"type":      types.StringType,
		"new_speed": types.Int64Type,
	}

	vxcPartnerConfigAttrs = map[string]attr.Type{
		"partner":              types.StringType,
		"aws_config":           types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigAWSAttrs),
		"azure_config":         types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigAzureAttrs),
		"google_config":        types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigGoogleAttrs),
		"oracle_config":        types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigOracleAttrs),
		"partner_a_end_config": types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigAEndAttrs),
	}

	vxcPartnerConfigAWSAttrs = map[string]attr.Type{
		"connect_type":        types.StringType,
		"type":                types.StringType,
		"owner_account":       types.StringType,
		"asn":                 types.Int64Type,
		"amazon_asn":          types.Int64Type,
		"auth_key":            types.StringType,
		"prefixes":            types.StringType,
		"customer_ip_address": types.StringType,
		"amazon_ip_address":   types.StringType,
		"name":                types.StringType,
	}

	vxcPartnerConfigAzureAttrs = map[string]attr.Type{
		"service_key": types.StringType,
	}

	vxcPartnerConfigGoogleAttrs = map[string]attr.Type{
		"pairing_key": types.StringType,
	}

	vxcPartnerConfigOracleAttrs = map[string]attr.Type{
		"virtual_circuit_id": types.StringType,
	}

	vxcPartnerConfigAEndAttrs = map[string]attr.Type{
		"interfaces": types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigInterfaceAttrs)),
	}

	vxcPartnerConfigInterfaceAttrs = map[string]attr.Type{
		"ip_addresses":     types.ListType{}.WithElementType(types.StringType),
		"ip_routes":        types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(ipRouteAttrs)),
		"nat_ip_addresses": types.ListType{}.WithElementType(types.StringType),
		"bfd":              types.ObjectType{}.WithAttributeTypes(bfdConfigAttrs),
		"bgp_connections":  types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(bgpConnectionConfig)),
	}

	ipRouteAttrs = map[string]attr.Type{
		"prefix":      types.StringType,
		"description": types.StringType,
		"next_hop":    types.StringType,
	}

	bfdConfigAttrs = map[string]attr.Type{
		"tx_interval": types.Int64Type,
		"rx_interval": types.Int64Type,
		"multiplier":  types.Int64Type,
	}

	bgpConnectionConfig = map[string]attr.Type{
		"peer_asn":         types.Int64Type,
		"local_ip_address": types.StringType,
		"peer_ip_address":  types.StringType,
		"password":         types.StringType,
		"shutdown":         types.BoolType,
		"description":      types.StringType,
		"med_in":           types.Int64Type,
		"med_out":          types.Int64Type,
		"bfd_enabled":      types.BoolType,
		"export_policy":    types.StringType,
		"permit_export_to": types.ListType{}.WithElementType(types.StringType),
		"deny_export_to":   types.ListType{}.WithElementType(types.StringType),
		"import_whitelist": types.StringType,
		"import_blacklist": types.StringType,
		"export_whitelist": types.StringType,
		"export_blacklist": types.StringType,
	}
)

// vxcResourceModel maps the resource schema data.
type vxcResourceModel struct {
	LastUpdated types.String `tfsdk:"last_updated"`

	ID                 types.Int64  `tfsdk:"product_id"`
	UID                types.String `tfsdk:"product_uid"`
	ServiceID          types.Int64  `tfsdk:"service_id"`
	Name               types.String `tfsdk:"product_name"`
	Type               types.String `tfsdk:"product_type"`
	RateLimit          types.Int64  `tfsdk:"rate_limit"`
	DistanceBand       types.String `tfsdk:"distance_band"`
	ProvisioningStatus types.String `tfsdk:"provisioning_status"`
	PromoCode          types.String `tfsdk:"promo_code"`

	SecondaryName  types.String `tfsdk:"secondary_name"`
	UsageAlgorithm types.String `tfsdk:"usage_algorithm"`
	CreatedBy      types.String `tfsdk:"created_by"`

	ContractTermMonths types.Int64  `tfsdk:"contract_term_months"`
	CompanyUID         types.String `tfsdk:"company_uid"`
	CompanyName        types.String `tfsdk:"company_name"`
	Locked             types.Bool   `tfsdk:"locked"`
	AdminLocked        types.Bool   `tfsdk:"admin_locked"`
	AttributeTags      types.Map    `tfsdk:"attribute_tags"`
	Cancelable         types.Bool   `tfsdk:"cancelable"`
	CostCentre         types.String `tfsdk:"cost_centre"`

	LiveDate          types.String `tfsdk:"live_date"`
	CreateDate        types.String `tfsdk:"create_date"`
	ContractStartDate types.String `tfsdk:"contract_start_date"`
	ContractEndDate   types.String `tfsdk:"contract_end_date"`
	Shutdown          types.Bool   `tfsdk:"shutdown"`

	AEndConfiguration types.Object `tfsdk:"a_end"`
	BEndConfiguration types.Object `tfsdk:"b_end"`

	AEndPartnerConfig types.Object `tfsdk:"a_end_partner_config"`
	BEndPartnerConfig types.Object `tfsdk:"b_end_partner_config"`

	VLL            types.Object `tfsdk:"vll"`
	VirtualRouter  types.Object `tfsdk:"virtual_router"`
	CSPConnections types.List   `tfsdk:"csp_connections"`
	PortInterfaces types.List   `tfsdk:"port_interfaces"`
	VXCApproval    types.Object `tfsdk:"vxc_approval"`
}

type cspConnectionModel struct {
	ConnectType        types.String `tfsdk:"connect_type"`
	ResourceName       types.String `tfsdk:"resource_name"`
	ResourceType       types.String `tfsdk:"resource_type"`
	VLAN               types.Int64  `tfsdk:"vlan"`
	Account            types.String `tfsdk:"account"`
	AmazonAddress      types.String `tfsdk:"amazon_address"`
	ASN                types.Int64  `tfsdk:"asn"`
	AuthKey            types.String `tfsdk:"auth_key"`
	CustomerAddress    types.String `tfsdk:"customer_address"`
	CustomerIPAddress  types.String `tfsdk:"customer_ip_address"`
	ID                 types.Int64  `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	OwnerAccount       types.String `tfsdk:"owner_account"`
	PeerASN            types.Int64  `tfsdk:"peer_asn"`
	Type               types.String `tfsdk:"type"`
	VIFID              types.String `tfsdk:"vif_id"`
	Bandwidth          types.Int64  `tfsdk:"bandwidth"`
	Bandwidths         types.List   `tfsdk:"bandwidths"`
	ConnectionID       types.String `tfsdk:"connection_id"`
	IPAddresses        types.List   `tfsdk:"ip_addresses"`
	VirtualRouterName  types.String `tfsdk:"virtual_router_name"`
	Managed            types.Bool   `tfsdk:"managed"`
	ServiceKey         types.String `tfsdk:"service_key"`
	CSPName            types.String `tfsdk:"csp_name"`
	PairingKey         types.String `tfsdk:"pairing_key"`
	CustomerIP4Address types.String `tfsdk:"customer_ip4_address"`
	CustomerIP6Network types.String `tfsdk:"customer_ip6_network"`
	IPv4GatewayAddress types.String `tfsdk:"ipv4_gateway_address"`
	IPv6GatewayAddress types.String `tfsdk:"ipv6_gateway_address"`
}

// virtualRouterModel maps the virtual router schema data.
type virtualRouterModel struct {
	MCRAsn             types.Int64  `tfsdk:"mcr_asn"`
	ResourceName       types.String `tfsdk:"resource_name"`
	ResourceType       types.String `tfsdk:"resource_type"`
	Speed              types.Int64  `tfsdk:"speed"`
	BGPShutdownDefault types.Bool   `tfsdk:"bgp_shutdown_default"`
}

// vllConfigModel maps the VLL configuration schema data.
type vllConfigModel struct {
	AEndVLAN      types.Int64  `tfsdk:"a_vlan"`
	BEndVLAN      types.Int64  `tfsdk:"b_vlan"`
	Description   types.String `tfsdk:"description"`
	ID            types.Int64  `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	RateLimitMBPS types.Int64  `tfsdk:"rate_limit_mbps"`
	ResourceName  types.String `tfsdk:"resource_name"`
	ResourceType  types.String `tfsdk:"resource_type"`
}

// vxcApprovalModel maps the approval schema data.
type vxcApprovalModel struct {
	Status   types.String `tfsdk:"status"`
	Message  types.String `tfsdk:"message"`
	UID      types.String `tfsdk:"uid"`
	Type     types.String `tfsdk:"type"`
	NewSpeed types.Int64  `tfsdk:"new_speed"`
}

// vxcEndConfigurationModel maps the end configuration schema data.
type vxcEndConfigurationModel struct {
	OwnerUID              types.String `tfsdk:"owner_uid"`
	RequestedProductUID   types.String `tfsdk:"requested_product_uid"`
	CurrentProductUID     types.String `tfsdk:"current_product_uid"`
	Name                  types.String `tfsdk:"product_name"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	Location              types.String `tfsdk:"location"`
	OrderedVLAN           types.Int64  `tfsdk:"ordered_vlan"`
	VLAN                  types.Int64  `tfsdk:"vlan"`
	InnerVLAN             types.Int64  `tfsdk:"inner_vlan"`
	NetworkInterfaceIndex types.Int64  `tfsdk:"vnic_index"`
	SecondaryName         types.String `tfsdk:"secondary_name"`
	LocationDetails       types.Object `tfsdk:"location_details"`
}

type vxcPartnerConfigurationModel struct {
	Partner             types.String `tfsdk:"partner"`
	AWSPartnerConfig    types.Object `tfsdk:"aws_config"`
	AzurePartnerConfig  types.Object `tfsdk:"azure_config"`
	GooglePartnerConfig types.Object `tfsdk:"google_config"`
	OraclePartnerConfig types.Object `tfsdk:"oracle_config"`
	PartnerAEndConfig   types.Object `tfsdk:"partner_a_end_config"`
}

type vxcPartnerConfig interface {
	isPartnerConfig()
}

// vxcPartnerConfigAWSModel maps the partner configuration schema data for AWS.
type vxcPartnerConfigAWSModel struct {
	vxcPartnerConfig
	ConnectType       types.String `tfsdk:"connect_type"`
	Type              types.String `tfsdk:"type"`
	OwnerAccount      types.String `tfsdk:"owner_account"`
	ASN               types.Int64  `tfsdk:"asn"`
	AmazonASN         types.Int64  `tfsdk:"amazon_asn"`
	AuthKey           types.String `tfsdk:"auth_key"`
	Prefixes          types.String `tfsdk:"prefixes"`
	CustomerIPAddress types.String `tfsdk:"customer_ip_address"`
	AmazonIPAddress   types.String `tfsdk:"amazon_ip_address"`
	ConnectionName    types.String `tfsdk:"name"`
}

// vxcPartnerConfigAzureModel maps the partner configuration schema data for Azure.
type vxcPartnerConfigAzureModel struct {
	vxcPartnerConfig
	ServiceKey types.String `tfsdk:"service_key"`
}

// vxcPartnerConfigGoogleModel maps the partner configuration schema data for Google.
type vxcPartnerConfigGoogleModel struct {
	vxcPartnerConfig
	PairingKey types.String `tfsdk:"pairing_key"`
}

// vxcPartnerConfigOracleModel maps the partner configuration schema data for Oracle.
type vxcPartnerConfigOracleModel struct {
	vxcPartnerConfig
	VirtualCircuitId types.String `tfsdk:"virtual_circuit_id"`
}

// vxcPartnerConfigAEndModel maps the partner configuration schema data for an A end.
type vxcPartnerConfigAEndModel struct {
	vxcPartnerConfig
	Interfaces types.List `tfsdk:"interfaces"`
}

// vxcPartnerConfigInterfaceModel maps the partner configuration schema data for an interface.
type vxcPartnerConfigInterfaceModel struct {
	IPAddresses    types.List   `tfsdk:"ip_addresses"`
	IPRoutes       types.List   `tfsdk:"ip_routes"`
	NatIPAddresses types.List   `tfsdk:"nat_ip_addresses"`
	Bfd            types.Object `tfsdk:"bfd"`
	BgpConnections types.List   `tfsdk:"bgp_connections"`
}

// ipRouteModel maps the IP route schema data.
type ipRouteModel struct {
	Prefix      types.String `tfsdk:"prefix"`
	Description types.String `tfsdk:"description"`
	NextHop     types.String `tfsdk:"next_hop"`
}

// BfdConfig represents the configuration of BFD.
type bfdConfigModel struct {
	TxInterval types.Int64 `tfsdk:"tx_interval"`
	RxInterval types.Int64 `tfsdk:"rx_interval"`
	Multiplier types.Int64 `tfsdk:"multiplier"`
}

// BgpConnectionConfig represents the configuration of a BGP connection.
type bgpConnectionConfigModel struct {
	PeerAsn         types.Int64  `tfsdk:"peer_asn"`
	LocalIPAddress  types.String `tfsdk:"local_ip_address"`
	PeerIPAddress   types.String `tfsdk:"peer_ip_address"`
	Password        types.String `tfsdk:"password"`
	Shutdown        types.Bool   `tfsdk:"shutdown"`
	Description     types.String `tfsdk:"description"`
	MedIn           types.Int64  `tfsdk:"med_in"`
	MedOut          types.Int64  `tfsdk:"med_out"`
	BfdEnabled      types.Bool   `tfsdk:"bfd_enabled"`
	ExportPolicy    types.String `tfsdk:"export_policy"`
	PermitExportTo  types.List   `tfsdk:"permit_export_to"`
	DenyExportTo    types.List   `tfsdk:"deny_export_to"`
	ImportWhitelist types.String `tfsdk:"import_whitelist"`
	ImportBlacklist types.String `tfsdk:"import_blacklist"`
	ExportWhitelist types.String `tfsdk:"export_whitelist"`
	ExportBlacklist types.String `tfsdk:"export_blacklist"`
}

func (orm *vxcResourceModel) fromAPIVXC(ctx context.Context, v *megaport.VXC) diag.Diagnostics {
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

	var aEndOrderedVLAN, bEndOrderedVLAN int64
	var aEndRequestedProductUID, bEndRequestedProductUID string
	if !orm.AEndConfiguration.IsNull() {
		existingAEnd := &vxcEndConfigurationModel{}
		aEndDiags := orm.AEndConfiguration.As(ctx, existingAEnd, basetypes.ObjectAsOptions{})
		apiDiags = append(apiDiags, aEndDiags...)
		aEndOrderedVLAN = existingAEnd.OrderedVLAN.ValueInt64()
		aEndRequestedProductUID = existingAEnd.RequestedProductUID.ValueString()
	}

	aEndModel := &vxcEndConfigurationModel{
		OwnerUID:              types.StringValue(v.AEndConfiguration.OwnerUID),
		RequestedProductUID:   types.StringValue(aEndRequestedProductUID),
		CurrentProductUID:     types.StringValue(v.AEndConfiguration.UID),
		Name:                  types.StringValue(v.AEndConfiguration.Name),
		LocationID:            types.Int64Value(int64(v.AEndConfiguration.LocationID)),
		Location:              types.StringValue(v.AEndConfiguration.Location),
		OrderedVLAN:           types.Int64Value(aEndOrderedVLAN),
		VLAN:                  types.Int64Value(int64(v.AEndConfiguration.VLAN)),
		InnerVLAN:             types.Int64Value(int64(v.AEndConfiguration.InnerVLAN)),
		NetworkInterfaceIndex: types.Int64Value(int64(v.AEndConfiguration.NetworkInterfaceIndex)),
		SecondaryName:         types.StringValue(v.AEndConfiguration.SecondaryName),
	}
	aEndLocationDetailsModel := &productLocationDetailsModel{
		Name:    types.StringValue(v.AEndConfiguration.LocationDetails.Name),
		City:    types.StringValue(v.AEndConfiguration.LocationDetails.City),
		Metro:   types.StringValue(v.AEndConfiguration.LocationDetails.Metro),
		Country: types.StringValue(v.AEndConfiguration.LocationDetails.Country),
	}
	aEndLocationDetails, locationDetailsDiags := types.ObjectValueFrom(ctx, productLocationDetailsAttrs, aEndLocationDetailsModel)
	apiDiags = append(apiDiags, locationDetailsDiags...)
	aEndModel.LocationDetails = aEndLocationDetails
	aEnd, aEndDiags := types.ObjectValueFrom(ctx, vxcEndConfigurationAttrs, aEndModel)
	apiDiags = append(apiDiags, aEndDiags...)
	orm.AEndConfiguration = aEnd

	if !orm.BEndConfiguration.IsNull() {
		existingBEnd := &vxcEndConfigurationModel{}
		bEndDiags := orm.BEndConfiguration.As(ctx, existingBEnd, basetypes.ObjectAsOptions{})
		apiDiags = append(apiDiags, bEndDiags...)
		bEndOrderedVLAN = existingBEnd.OrderedVLAN.ValueInt64()
		bEndRequestedProductUID = existingBEnd.RequestedProductUID.ValueString()
	}

	bEndModel := &vxcEndConfigurationModel{
		OwnerUID:              types.StringValue(v.BEndConfiguration.OwnerUID),
		RequestedProductUID:   types.StringValue(bEndRequestedProductUID),
		CurrentProductUID:     types.StringValue(v.BEndConfiguration.UID),
		Name:                  types.StringValue(v.BEndConfiguration.Name),
		LocationID:            types.Int64Value(int64(v.BEndConfiguration.LocationID)),
		Location:              types.StringValue(v.BEndConfiguration.Location),
		OrderedVLAN:           types.Int64Value(bEndOrderedVLAN),
		VLAN:                  types.Int64Value(int64(v.BEndConfiguration.VLAN)),
		InnerVLAN:             types.Int64Value(int64(v.BEndConfiguration.InnerVLAN)),
		NetworkInterfaceIndex: types.Int64Value(int64(v.BEndConfiguration.NetworkInterfaceIndex)),
		SecondaryName:         types.StringValue(v.BEndConfiguration.SecondaryName),
	}
	bEndLocationDetailsModel := &productLocationDetailsModel{
		Name:    types.StringValue(v.AEndConfiguration.LocationDetails.Name),
		City:    types.StringValue(v.AEndConfiguration.LocationDetails.City),
		Metro:   types.StringValue(v.AEndConfiguration.LocationDetails.Metro),
		Country: types.StringValue(v.AEndConfiguration.LocationDetails.Country),
	}
	bEndLocationDetails, locationDetailsDiags := types.ObjectValueFrom(ctx, productLocationDetailsAttrs, bEndLocationDetailsModel)
	apiDiags = append(apiDiags, locationDetailsDiags...)
	bEndModel.LocationDetails = bEndLocationDetails
	bEnd, bEndDiags := types.ObjectValueFrom(ctx, vxcEndConfigurationAttrs, bEndModel)
	apiDiags = append(apiDiags, bEndDiags...)
	orm.BEndConfiguration = bEnd

	vxcApprovalModel := &vxcApprovalModel{
		Status:   types.StringValue(v.VXCApproval.Status),
		Message:  types.StringValue(v.VXCApproval.Message),
		UID:      types.StringValue(v.VXCApproval.UID),
		Type:     types.StringValue(v.VXCApproval.Type),
		NewSpeed: types.Int64Value(int64(v.VXCApproval.NewSpeed)),
	}
	vxcApproval, vxcApprovalDiags := types.ObjectValueFrom(ctx, vxcApprovalAttrs, vxcApprovalModel)
	apiDiags = append(apiDiags, vxcApprovalDiags...)
	orm.VXCApproval = vxcApproval

	if v.Resources != nil {
		if v.Resources.Interface != nil {
			interfaceObjects := []types.Object{}
			for _, i := range v.Resources.Interface {
				interfaceObject, interfaceDiags := fromAPIPortInterface(ctx, i)
				apiDiags = append(apiDiags, interfaceDiags...)
				interfaceObjects = append(interfaceObjects, interfaceObject)
			}
			portInterfaceList, interfaceListDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(portInterfaceAttrs), interfaceObjects)
			apiDiags = append(apiDiags, interfaceListDiags...)
			orm.PortInterfaces = portInterfaceList
		} else {
			interfaceList := types.ListNull(types.ObjectType{}.WithAttributeTypes(portInterfaceAttrs))
			orm.PortInterfaces = interfaceList
		}
	}

	if v.Resources.VLL != nil {
		vllModel := &vllConfigModel{
			AEndVLAN:      types.Int64Value(int64(v.Resources.VLL.AEndVLAN)),
			BEndVLAN:      types.Int64Value(int64(v.Resources.VLL.BEndVLAN)),
			Description:   types.StringValue(v.Resources.VLL.Description),
			ID:            types.Int64Value(int64(v.Resources.VLL.ID)),
			Name:          types.StringValue(v.Resources.VLL.Name),
			RateLimitMBPS: types.Int64Value(int64(v.Resources.VLL.RateLimitMBPS)),
			ResourceName:  types.StringValue(v.Resources.VLL.ResourceName),
			ResourceType:  types.StringValue(v.Resources.VLL.ResourceType),
		}
		vll, vllDiags := types.ObjectValueFrom(ctx, vllConfigAttrs, vllModel)
		apiDiags = append(apiDiags, vllDiags...)
		orm.VLL = vll
	}

	if v.Resources.VirtualRouter != nil {
		virtualRouterModel := &virtualRouterModel{
			MCRAsn:             types.Int64Value(int64(v.Resources.VirtualRouter.MCRAsn)),
			ResourceName:       types.StringValue(v.Resources.VirtualRouter.ResourceName),
			ResourceType:       types.StringValue(v.Resources.VirtualRouter.ResourceType),
			Speed:              types.Int64Value(int64(v.Resources.VirtualRouter.Speed)),
			BGPShutdownDefault: types.BoolValue(v.Resources.VirtualRouter.BGPShutdownDefault),
		}
		virtualRouter, virtualRouterDiags := types.ObjectValueFrom(ctx, virtualRouterAttrs, virtualRouterModel)
		apiDiags = append(apiDiags, virtualRouterDiags...)
		orm.VirtualRouter = virtualRouter
	} else {
		orm.VirtualRouter = types.ObjectNull(virtualRouterAttrs)
	}
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
	return apiDiags
}

// NewPortResource is a helper function to simplify the provider implementation.
func NewVXCResource() resource.Resource {
	return &vxcResource{}
}

// vxcResource is the resource implementation.
type vxcResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *vxcResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vxc"
}

// Schema defines the schema for the resource.
func (r *vxcResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Virtual Cross Connect (VXC) Resource for the Megaport Terraform Provider. This resource allows you to create, modify, and update VXCs. VXCs are Layer 2 Ethernet circuits providing private, flexible, and on-demand connections between any of the locations on the Megaport network with 1 Mbps to 100 Gbps of capacity.",
		Attributes: map[string]schema.Attribute{
			"last_updated": schema.StringAttribute{
				Description: "The last time the resource was updated.",
				Computed:    true,
			},
			"product_uid": schema.StringAttribute{
				Description: "The unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_id": schema.Int64Attribute{
				Description: "The numeric ID of the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"product_name": schema.StringAttribute{
				Description: "The name of the product.",
				Required:    true,
			},
			"service_id": schema.Int64Attribute{
				Description: "The service ID of the VXC.",
				Computed:    true,
			},
			"rate_limit": schema.Int64Attribute{
				Description: "The rate limit of the product.",
				Required:    true,
			},
			"product_type": schema.StringAttribute{
				Description: "The type of the product.",
				Computed:    true,
			},
			"distance_band": schema.StringAttribute{
				Description: "The distance band of the product.",
				Computed:    true,
			},
			"provisioning_status": schema.StringAttribute{
				Description: "The provisioning status of the product.",
				Computed:    true,
			},
			"secondary_name": schema.StringAttribute{
				Description: "The secondary name of the product.",
				Computed:    true,
			},
			"usage_algorithm": schema.StringAttribute{
				Description: "The usage algorithm of the product.",
				Computed:    true,
			},
			"promo_code": schema.StringAttribute{
				Description: "Promo code is an optional string that can be used to enter a promotional code for the service order. The code is not validated, so if the code doesn't exist or doesn't work for the service, the request will still be successful.",
				Optional:    true,
			},
			"created_by": schema.StringAttribute{
				Description: "The user who created the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"live_date": schema.StringAttribute{
				Description: "The date the product went live.",
				Computed:    true,
			},
			"create_date": schema.StringAttribute{
				Description: "The date the product was created.",
				Computed:    true,
			},
			"contract_term_months": schema.Int64Attribute{
				Description: "The term of the contract in months: valid values are 1, 12, 24, and 36.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(1, 12, 24, 36),
				},
			},
			"shutdown": schema.BoolAttribute{
				Description: "Temporarily shut down and re-enable the VXC. Valid values are true (shut down) and false (enabled). If not provided, it defaults to false (enabled).",
				Computed:    true,
				Optional:    true,
			},
			"cost_centre": schema.StringAttribute{
				Description: "A customer reference number to be included in billing information and invoices. Also known as the service level reference (SLR) number. Specify a unique identifying number for the product to be used for billing purposes, such as a cost center number or a unique customer ID. The service level reference number appears for each service under the Product section of the invoice. You can also edit this field for an existing service.",
				Computed:    true,
				Optional:    true,
			},
			"vll": schema.SingleNestedAttribute{
				Description: "The VLL associated with the VXC.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"a_vlan": schema.Int64Attribute{
						Description: "The A-End VLAN of the VLL.",
						Computed:    true,
					},
					"b_vlan": schema.Int64Attribute{
						Description: "The B-End VLAN of the VLL.",
						Computed:    true,
					},
					"description": schema.StringAttribute{
						Description: "The description of the VLL.",
						Computed:    true,
					},
					"id": schema.Int64Attribute{
						Description: "The ID of the VLL.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "The name of the VLL.",
						Computed:    true,
					},
					"rate_limit_mbps": schema.Int64Attribute{
						Description: "The rate limit in Mbps of the VLL.",
						Computed:    true,
					},
					"resource_name": schema.StringAttribute{
						Description: "The resource name of the VLL.",
						Computed:    true,
					},
					"resource_type": schema.StringAttribute{
						Description: "The resource type of the VLL.",
						Computed:    true,
					},
				},
			},
			"csp_connections": schema.ListNestedAttribute{
				Description: "The Cloud Service Provider (CSP) connections associated with the VXC.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"connect_type": schema.StringAttribute{
							Description: "The connection type of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"resource_name": schema.StringAttribute{
							Description: "The resource name of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"resource_type": schema.StringAttribute{
							Description: "The resource type of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"vlan": schema.Int64Attribute{
							Description: "The VLAN of the CSP connection.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"owner_account": schema.StringAttribute{
							Description: "The owner's AWS account of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"bandwidth": schema.Int64Attribute{
							Description: "The bandwidth of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"bandwidths": schema.ListAttribute{
							Description: "The bandwidths of the CSP connection.",
							Optional:    true,
							Computed:    true,
							ElementType: types.Int64Type,
						},
						"customer_ip_address": schema.StringAttribute{
							Description: "The customer IP address of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"customer_ip4_address": schema.StringAttribute{
							Description: "The customer IPv4 address of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"account": schema.StringAttribute{
							Description: "The account of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"amazon_address": schema.StringAttribute{
							Description: "The Amazon address of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"asn": schema.Int64Attribute{
							Description: "The ASN of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"auth_key": schema.StringAttribute{
							Description: "The authentication key of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"customer_address": schema.StringAttribute{
							Description: "The customer address of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"id": schema.Int64Attribute{
							Description: "The ID of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"peer_asn": schema.Int64Attribute{
							Description: "The peer ASN of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The type of the AWS Virtual Interface.",
							Optional:    true,
							Computed:    true,
						},
						"vif_id": schema.StringAttribute{
							Description: "The ID of the AWS Virtual Interface.",
							Optional:    true,
							Computed:    true,
						},
						"connection_id": schema.StringAttribute{
							Description: "The hosted connection ID of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"managed": schema.BoolAttribute{
							Description: "Whether the CSP connection is managed.",
							Optional:    true,
							Computed:    true,
						},
						"service_key": schema.StringAttribute{
							Description: "The Azure service key of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"csp_name": schema.StringAttribute{
							Description: "The name of the CSP connection.",
							Optional:    true,
							Computed:    true,
						},
						"pairing_key": schema.StringAttribute{
							Description: "The pairing key of the Google Cloud connection.",
							Optional:    true,
							Computed:    true,
						},
						"ip_addresses": schema.ListAttribute{
							Description: "The IP addresses of the Virtual Router.",
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
						},
						"virtual_router_name": schema.StringAttribute{
							Description: "The name of the Virtual Router.",
							Optional:    true,
							Computed:    true,
						},
						"customer_ip6_network": schema.StringAttribute{
							Description: "The customer IPv6 network of the Transit VXC connection.",
							Optional:    true,
							Computed:    true,
						},
						"ipv4_gateway_address": schema.StringAttribute{
							Description: "The IPv4 gateway address of the Transit VXC connection.",
							Optional:    true,
							Computed:    true,
						},
						"ipv6_gateway_address": schema.StringAttribute{
							Description: "The IPv6 gateway address of the Transit VXC connection.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"port_interfaces": schema.ListNestedAttribute{
				Description: "The interfaces associated with the VXC.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"demarcation": schema.StringAttribute{
							Description: "The demarcation of the interface.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the interface.",
							Computed:    true,
						},
						"id": schema.Int64Attribute{
							Description: "The ID of the interface.",
							Computed:    true,
						},
						"loa_template": schema.StringAttribute{
							Description: "The LOA template of the interface.",
							Computed:    true,
						},
						"media": schema.StringAttribute{
							Description: "The media of the interface.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the interface.",
							Computed:    true,
						},
						"port_speed": schema.Int64Attribute{
							Description: "The port speed of the interface.",
							Computed:    true,
						},
						"resource_name": schema.StringAttribute{
							Description: "The resource name of the interface.",
							Computed:    true,
						},
						"resource_type": schema.StringAttribute{
							Description: "The resource type of the interface.",
							Computed:    true,
						},
						"up": schema.Int64Attribute{
							Description: "The up status of the interface.",
							Computed:    true,
						},
					},
				},
			},
			"virtual_router": schema.SingleNestedAttribute{
				Description: "The virtual router associated with the VXC.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"mcr_asn": schema.Int64Attribute{
						Description: "The MCR ASN of the virtual router.",
						Computed:    true,
					},
					"resource_name": schema.StringAttribute{
						Description: "The resource name of the virtual router.",
						Computed:    true,
					},
					"resource_type": schema.StringAttribute{
						Description: "The resource type of the virtual router.",
						Computed:    true,
					},
					"speed": schema.Int64Attribute{
						Description: "The speed of the virtual router.",
						Computed:    true,
					},
					"bgp_shutdown_default": schema.BoolAttribute{
						Description: "Whether BGP Shutdown is enabled by default on the virtual router.",
						Computed:    true,
					},
				},
			},
			"vxc_approval": schema.SingleNestedAttribute{
				Description: "The VXC approval details.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"status": schema.StringAttribute{
						Description: "The status of the VXC approval.",
						Computed:    true,
					},
					"message": schema.StringAttribute{
						Description: "The message of the VXC approval.",
						Computed:    true,
					},
					"uid": schema.StringAttribute{
						Description: "The UID of the VXC approval.",
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "The type of the VXC approval.",
						Computed:    true,
					},
					"new_speed": schema.Int64Attribute{
						Description: "The new speed of the VXC approval.",
						Computed:    true,
					},
				},
			},
			"contract_start_date": schema.StringAttribute{
				Description: "The date the contract starts.",
				Computed:    true,
			},
			"contract_end_date": schema.StringAttribute{
				Description: "The date the contract ends.",
				Computed:    true,
			},
			"company_uid": schema.StringAttribute{
				Description: "The UID of the company the product is associated with.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"company_name": schema.StringAttribute{
				Description: "The name of the company the product is associated with.",
				Computed:    true,
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the product is locked.",
				Computed:    true,
			},
			"admin_locked": schema.BoolAttribute{
				Description: "Whether the product is admin locked.",
				Computed:    true,
			},
			"attribute_tags": schema.MapAttribute{
				Description: "The attribute tags associated with the product.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"cancelable": schema.BoolAttribute{
				Description: "Whether the product is cancelable.",
				Computed:    true,
			},
			"a_end": schema.SingleNestedAttribute{
				Description: "The current A-End configuration of the VXC.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"owner_uid": schema.StringAttribute{
						Description: "The owner UID of the A-End configuration.",
						Computed:    true,
					},
					"requested_product_uid": schema.StringAttribute{
						Description: "The Product UID requested by the user for the A-End configuration.",
						Required:    true,
					},
					"current_product_uid": schema.StringAttribute{
						Description: "The current product UID of the A-End configuration. The Megaport API may change a Partner Port from the Requested Port to a different Port in the same location and diversity zone.",
						Optional:    true,
						Computed:    true,
					},
					"product_name": schema.StringAttribute{
						Description: "The product name of the A-End configuration.",
						Computed:    true,
					},
					"location_id": schema.Int64Attribute{
						Description: "The location ID of the A-End configuration.",
						Computed:    true,
					},
					"location": schema.StringAttribute{
						Description: "The location of the A-End configuration.",
						Computed:    true,
					},
					"location_details": schema.SingleNestedAttribute{
						Description: "The location details of the product.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"name": schema.StringAttribute{
								Description: "The name of the location.",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"city": schema.StringAttribute{
								Description: "The city of the location.",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"metro": schema.StringAttribute{
								Description: "The metro of the location.",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"country": schema.StringAttribute{
								Description: "The country of the location.",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"ordered_vlan": schema.Int64Attribute{
						Description: "The customer-ordered unique VLAN ID of the A-End configuration. Values can range from 2 to 4093. If this value is set to 0, or not included, the Megaport system allocates a valid VLAN ID.",
						Optional:    true,
						Computed:    true,
						Validators:  []validator.Int64{int64validator.Between(0, 4093), int64validator.NoneOf(1)},
					},
					"vlan": schema.Int64Attribute{
						Description: "The current VLAN of the A-End configuration. May be different from the ordered VLAN if the system allocated a different VLAN. Values can range from 2 to 4093. If the ordered_vlan was set to 0, the Megaport system allocated a valid VLAN.",
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"inner_vlan": schema.Int64Attribute{
						Description: "The inner VLAN of the A-End configuration.",
						Optional:    true,
						Computed:    true,
					},
					"vnic_index": schema.Int64Attribute{
						Description: "The network interface index of the A-End configuration.",
						Computed:    true,
						Optional:    true,
					},
					"secondary_name": schema.StringAttribute{
						Description: "The secondary name of the A-End configuration.",
						Computed:    true,
					},
				},
			},
			"b_end": schema.SingleNestedAttribute{
				Description: "The current B-End configuration of the VXC.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"owner_uid": schema.StringAttribute{
						Description: "The owner UID of the B-End configuration.",
						Computed:    true,
					},
					"requested_product_uid": schema.StringAttribute{
						Description: "The Product UID requested by the user for the B-End configuration.",
						Optional:    true,
						Computed:    true,
					},
					"current_product_uid": schema.StringAttribute{
						Description: "The current product UID of the B-End configuration. The Megaport API may change a Partner Port on the end configuration from the Requested Port UID to a different Port in the same location and diversity zone.",
						Optional:    true,
						Computed:    true,
					},
					"product_name": schema.StringAttribute{
						Description: "The product name of the B-End configuration.",
						Computed:    true,
					},
					"location_id": schema.Int64Attribute{
						Description: "The location ID of the B-End configuration.",
						Computed:    true,
					},
					"location": schema.StringAttribute{
						Description: "The location of the B-End configuration.",
						Computed:    true,
					},
					"location_details": schema.SingleNestedAttribute{
						Description: "The location details of the product.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"name": schema.StringAttribute{
								Description: "The name of the location.",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"city": schema.StringAttribute{
								Description: "The city of the location.",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"metro": schema.StringAttribute{
								Description: "The metro of the location.",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"country": schema.StringAttribute{
								Description: "The country of the location.",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"ordered_vlan": schema.Int64Attribute{
						Description: "The customer-ordered unique VLAN ID of the B-End configuration. Values can range from 2 to 4093. If this value is set to 0, or not included, the Megaport system allocates a valid VLAN ID.",
						Optional:    true,
						Computed:    true,
						Validators:  []validator.Int64{int64validator.Between(0, 4093), int64validator.NoneOf(1)},
					},
					"vlan": schema.Int64Attribute{
						Description: "The current VLAN of the B-End configuration. May be different from the ordered VLAN if the system allocated a different VLAN. Values can range from 2 to 4093. If the ordered_vlan was set to 0, the Megaport system allocated a valid VLAN.",
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"inner_vlan": schema.Int64Attribute{
						Description: "The inner VLAN of the B-End configuration.",
						Optional:    true,
						Computed:    true,
					},
					"vnic_index": schema.Int64Attribute{
						Description: "The network interface index of the B-End configuration.",
						Optional:    true,
						Computed:    true,
					},
					"secondary_name": schema.StringAttribute{
						Description: "The secondary name of the B-End configuration.",
						Computed:    true,
					},
				},
			},
			"a_end_partner_config": schema.SingleNestedAttribute{
				Description: "The partner configuration of the A-End order configuration.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"partner": schema.StringAttribute{
						Description: "The partner of the partner configuration.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("aws", "azure", "google", "oracle", "a-end", "transit"),
						},
					},
					"aws_config": schema.SingleNestedAttribute{
						Description: "The AWS partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"connect_type": schema.StringAttribute{
								Description: "The connection type of the partner configuration. Required for AWS partner configurations.",
								Required:    true,
							},
							"type": schema.StringAttribute{
								Description: "The type of the partner configuration. Required for AWS partner configurations.",
								Required:    true,
							},
							"owner_account": schema.StringAttribute{
								Description: "The owner AWS account of the partner configuration. Required for AWS partner configurations.",
								Required:    true,
							},
							"asn": schema.Int64Attribute{
								Description: "The ASN of the partner configuration.",
								Optional:    true,
							},
							"amazon_asn": schema.Int64Attribute{
								Description: "The Amazon ASN of the partner configuration.",
								Optional:    true,
							},
							"auth_key": schema.StringAttribute{
								Description: "The authentication key of the partner configuration.",
								Optional:    true,
							},
							"prefixes": schema.StringAttribute{
								Description: "The prefixes of the partner configuration.",
								Optional:    true,
							},
							"customer_ip_address": schema.StringAttribute{
								Description: "The customer IP address of the partner configuration.",
								Optional:    true,
							},
							"amazon_ip_address": schema.StringAttribute{
								Description: "The Amazon IP address of the partner configuration.",
								Optional:    true,
							},
							"name": schema.StringAttribute{
								Description: "The name of the partner configuration.",
								Required:    true,
							},
						},
					},
					"azure_config": schema.SingleNestedAttribute{
						Description: "The Azure partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"service_key": schema.StringAttribute{
								Description: "The service key of the partner configuration. Required for Azure partner configurations.",
								Required:    true,
							},
						},
					},
					"google_config": schema.SingleNestedAttribute{
						Description: "The Google partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"pairing_key": schema.StringAttribute{
								Description: "The pairing key of the partner configuration. Required for Google partner configurations.",
								Required:    true,
							},
						},
					},
					"oracle_config": schema.SingleNestedAttribute{
						Description: "The Oracle partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"virtual_circuit_id": schema.StringAttribute{
								Description: "The virtual circuit ID of the partner configuration. Required for Oracle partner configurations.",
								Required:    true,
							},
						},
					},
					"partner_a_end_config": schema.SingleNestedAttribute{
						Description: "The partner configuration of the A-End order configuration. Only exists for A-End Configurations.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"interfaces": schema.ListNestedAttribute{
								Description: "The interfaces of the partner configuration.",
								Required:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"ip_addresses": schema.ListAttribute{
											Description: "The IP addresses of the partner configuration.",
											Optional:    true,
											ElementType: types.StringType,
										},
										"ip_routes": schema.ListNestedAttribute{
											Description: "The IP routes of the partner configuration.",
											Optional:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"prefix": schema.StringAttribute{
														Description: "The prefix of the IP route.",
														Optional:    true,
													},
													"description": schema.StringAttribute{
														Description: "The description of the IP route.",
														Optional:    true,
													},
													"next_hop": schema.StringAttribute{
														Description: "The next hop of the IP route.",
														Optional:    true,
													},
												},
											},
										},
										"nat_ip_addresses": schema.ListAttribute{
											Description: "The NAT IP addresses of the partner configuration.",
											Optional:    true,
											ElementType: types.StringType,
										},
										"bfd": schema.SingleNestedAttribute{
											Description: "The BFD of the partner configuration interface.",
											Optional:    true,
											Attributes: map[string]schema.Attribute{
												"tx_interval": schema.Int64Attribute{
													Description: "The transmit interval of the BFD.",
													Optional:    true,
												},
												"rx_interval": schema.Int64Attribute{
													Description: "The receive interval of the BFD.",
													Optional:    true,
												},
												"multiplier": schema.Int64Attribute{
													Description: "The multiplier of the BFD.",
													Optional:    true,
												},
											},
										},
										"bgp_connections": schema.ListNestedAttribute{
											Description: "The BGP connections of the partner configuration interface.",
											Optional:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"peer_asn": schema.Int64Attribute{
														Description: "The peer ASN of the BGP connection.",
														Optional:    true,
													},
													"local_ip_address": schema.StringAttribute{
														Description: "The local IP address of the BGP connection.",
														Optional:    true,
													},
													"peer_ip_address": schema.StringAttribute{
														Description: "The peer IP address of the BGP connection.",
														Optional:    true,
													},
													"password": schema.StringAttribute{
														Description: "The password of the BGP connection.",
														Optional:    true,
													},
													"shutdown": schema.BoolAttribute{
														Description: "Whether the BGP connection is shut down.",
														Optional:    true,
													},
													"description": schema.StringAttribute{
														Description: "The description of the BGP connection.",
														Optional:    true,
													},
													"med_in": schema.Int64Attribute{
														Description: "The MED in of the BGP connection.",
														Optional:    true,
													},
													"med_out": schema.Int64Attribute{
														Description: "The MED out of the BGP connection.",
														Optional:    true,
													},
													"bfd_enabled": schema.BoolAttribute{
														Description: "Whether BFD is enabled for the BGP connection.",
														Optional:    true,
													},
													"export_policy": schema.StringAttribute{
														Description: "The export policy of the BGP connection.",
														Optional:    true,
													},
													"permit_export_to": schema.ListAttribute{
														Description: "The permitted export to of the BGP connection.",
														Optional:    true,
														ElementType: types.StringType,
													},
													"deny_export_to": schema.ListAttribute{
														Description: "The denied export to of the BGP connection.",
														Optional:    true,
														ElementType: types.StringType,
													},
													"import_whitelist": schema.StringAttribute{
														Description: "The import whitelist of the BGP connection.",
														Optional:    true,
													},
													"import_blacklist": schema.StringAttribute{
														Description: "The import blacklist of the BGP connection.",
														Optional:    true,
													},
													"export_whitelist": schema.StringAttribute{
														Description: "The export whitelist of the BGP connection.",
														Optional:    true,
													},
													"export_blacklist": schema.StringAttribute{
														Description: "The export blacklist of the BGP connection.",
														Optional:    true,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"b_end_partner_config": schema.SingleNestedAttribute{
				Description: "The partner configuration of the B-End order configuration.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"partner": schema.StringAttribute{
						Description: "The partner of the partner configuration.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("aws", "azure", "google", "oracle", "transit"),
						},
					},
					"aws_config": schema.SingleNestedAttribute{
						Description: "The AWS partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"connect_type": schema.StringAttribute{
								Description: "The connection type of the partner configuration. Required for AWS partner configurations.",
								Required:    true,
							},
							"type": schema.StringAttribute{
								Description: "The type of the partner configuration. Required for AWS partner configurations.",
								Required:    true,
							},
							"owner_account": schema.StringAttribute{
								Description: "The owner AWS account of the partner configuration. Required for AWS partner configurations.",
								Required:    true,
							},
							"asn": schema.Int64Attribute{
								Description: "The ASN of the partner configuration.",
								Optional:    true,
							},
							"amazon_asn": schema.Int64Attribute{
								Description: "The Amazon ASN of the partner configuration.",
								Optional:    true,
							},
							"auth_key": schema.StringAttribute{
								Description: "The authentication key of the partner configuration.",
								Optional:    true,
							},
							"prefixes": schema.StringAttribute{
								Description: "The prefixes of the partner configuration.",
								Optional:    true,
							},
							"customer_ip_address": schema.StringAttribute{
								Description: "The customer IP address of the partner configuration.",
								Optional:    true,
							},
							"amazon_ip_address": schema.StringAttribute{
								Description: "The Amazon IP address of the partner configuration.",
								Optional:    true,
							},
							"name": schema.StringAttribute{
								Description: "The name of the partner configuration.",
								Required:    true,
							},
						},
					},
					"azure_config": schema.SingleNestedAttribute{
						Description: "The Azure partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"service_key": schema.StringAttribute{
								Description: "The service key of the partner configuration. Required for Azure partner configurations.",
								Required:    true,
							},
						},
					},
					"google_config": schema.SingleNestedAttribute{
						Description: "The Google partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"pairing_key": schema.StringAttribute{
								Description: "The pairing key of the partner configuration. Required for Google partner configurations.",
								Required:    true,
							},
						},
					},
					"oracle_config": schema.SingleNestedAttribute{
						Description: "The Oracle partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"virtual_circuit_id": schema.StringAttribute{
								Description: "The virtual circuit ID of the partner configuration. Required for Oracle partner configurations.",
								Required:    true,
							},
						},
					},
					"partner_a_end_config": schema.SingleNestedAttribute{
						Description: "The partner configuration of the A-End order configuration. Only exists for A-End Configurations and does not apply to B-End Partner Configuration.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"interfaces": schema.ListNestedAttribute{
								Description: "The interfaces of the partner configuration.",
								Required:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"ip_addresses": schema.ListAttribute{
											Description: "The IP addresses of the partner configuration.",
											Optional:    true,
											ElementType: types.StringType,
										},
										"ip_routes": schema.ListNestedAttribute{
											Description: "The IP routes of the partner configuration.",
											Optional:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"prefix": schema.StringAttribute{
														Description: "The prefix of the IP route.",
														Optional:    true,
													},
													"description": schema.StringAttribute{
														Description: "The description of the IP route.",
														Optional:    true,
													},
													"next_hop": schema.StringAttribute{
														Description: "The next hop of the IP route.",
														Optional:    true,
													},
												},
											},
										},
										"nat_ip_addresses": schema.ListAttribute{
											Description: "The NAT IP addresses of the partner configuration.",
											Optional:    true,
											ElementType: types.StringType,
										},
										"bfd": schema.SingleNestedAttribute{
											Description: "The BFD of the partner configuration interface.",
											Optional:    true,
											Attributes: map[string]schema.Attribute{
												"tx_interval": schema.Int64Attribute{
													Description: "The transmit interval of the BFD.",
													Optional:    true,
												},
												"rx_interval": schema.Int64Attribute{
													Description: "The receive interval of the BFD.",
													Optional:    true,
												},
												"multiplier": schema.Int64Attribute{
													Description: "The multiplier of the BFD.",
													Optional:    true,
												},
											},
										},
										"bgp_connections": schema.ListNestedAttribute{
											Description: "The BGP connections of the partner configuration interface.",
											Optional:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"peer_asn": schema.Int64Attribute{
														Description: "The peer ASN of the BGP connection.",
														Optional:    true,
													},
													"local_ip_address": schema.StringAttribute{
														Description: "The local IP address of the BGP connection.",
														Optional:    true,
													},
													"peer_ip_address": schema.StringAttribute{
														Description: "The peer IP address of the BGP connection.",
														Optional:    true,
													},
													"password": schema.StringAttribute{
														Description: "The password of the BGP connection.",
														Optional:    true,
													},
													"shutdown": schema.BoolAttribute{
														Description: "Whether the BGP connection is shut down.",
														Optional:    true,
													},
													"description": schema.StringAttribute{
														Description: "The description of the BGP connection.",
														Optional:    true,
													},
													"med_in": schema.Int64Attribute{
														Description: "The MED in of the BGP connection.",
														Optional:    true,
													},
													"med_out": schema.Int64Attribute{
														Description: "The MED out of the BGP connection.",
														Optional:    true,
													},
													"bfd_enabled": schema.BoolAttribute{
														Description: "Whether BFD is enabled for the BGP connection.",
														Optional:    true,
													},
													"export_policy": schema.StringAttribute{
														Description: "The export policy of the BGP connection.",
														Optional:    true,
													},
													"permit_export_to": schema.ListAttribute{
														Description: "The permitted export to of the BGP connection.",
														Optional:    true,
														ElementType: types.StringType,
													},
													"deny_export_to": schema.ListAttribute{
														Description: "The denied export to of the BGP connection.",
														Optional:    true,
														ElementType: types.StringType,
													},
													"import_whitelist": schema.StringAttribute{
														Description: "The import whitelist of the BGP connection.",
														Optional:    true,
													},
													"import_blacklist": schema.StringAttribute{
														Description: "The import blacklist of the BGP connection.",
														Optional:    true,
													},
													"export_whitelist": schema.StringAttribute{
														Description: "The export whitelist of the BGP connection.",
														Optional:    true,
													},
													"export_blacklist": schema.StringAttribute{
														Description: "The export blacklist of the BGP connection.",
														Optional:    true,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Create a new resource.
func (r *vxcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan vxcResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	buyReq := &megaport.BuyVXCRequest{
		VXCName:    plan.Name.ValueString(),
		Term:       int(plan.ContractTermMonths.ValueInt64()),
		RateLimit:  int(plan.RateLimit.ValueInt64()),
		PromoCode:  plan.PromoCode.ValueString(),
		CostCentre: plan.CostCentre.ValueString(),

		WaitForProvision: true,
		WaitForTime:      waitForTime,
	}

	if !plan.Shutdown.IsNull() {
		buyReq.Shutdown = plan.Shutdown.ValueBool()
	}

	aEndObj := plan.AEndConfiguration
	bEndObj := plan.BEndConfiguration

	var a vxcEndConfigurationModel
	aEndDiags := aEndObj.As(ctx, &a, basetypes.ObjectAsOptions{})
	if aEndDiags.HasError() {
		resp.Diagnostics.Append(aEndDiags...)
		return
	}
	aEndConfig := &megaport.VXCOrderEndpointConfiguration{
		ProductUID: a.RequestedProductUID.ValueString(),
		VLAN:       int(a.VLAN.ValueInt64()),
	}
	buyReq.PortUID = a.RequestedProductUID.ValueString()

	if !a.OrderedVLAN.IsNull() {
		aEndConfig.VLAN = int(a.OrderedVLAN.ValueInt64())
	} else {
		aEndConfig.VLAN = 0
	}

	if !a.InnerVLAN.IsNull() && !a.NetworkInterfaceIndex.IsNull() {
		aEndConfig.VXCOrderMVEConfig = &megaport.VXCOrderMVEConfig{
			InnerVLAN:             int(a.InnerVLAN.ValueInt64()),
			NetworkInterfaceIndex: int(a.NetworkInterfaceIndex.ValueInt64()),
		}
	}

	if !plan.AEndPartnerConfig.IsNull() {
		var aPartnerConfig vxcPartnerConfigurationModel
		aPartnerDiags := plan.AEndPartnerConfig.As(ctx, &aPartnerConfig, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})
		resp.Diagnostics.Append(aPartnerDiags...)
		switch aPartnerConfig.Partner.ValueString() {
		case "aws":
			if aPartnerConfig.AWSPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": AWS Partner configuration is required",
				)
				return
			}
			var awsConfig vxcPartnerConfigAWSModel
			awsDiags := aPartnerConfig.AWSPartnerConfig.As(ctx, &awsConfig, basetypes.ObjectAsOptions{})
			if awsDiags.HasError() {
				resp.Diagnostics.Append(awsDiags...)
				return
			}
			aEndPartnerConfig := megaport.VXCPartnerConfigAWS{
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
			resp.Diagnostics.Append(awsDiags...)

			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:             aPartnerConfig.Partner,
				AWSPartnerConfig:    awsConfigObj,
				AzurePartnerConfig:  azure,
				GooglePartnerConfig: google,
				OraclePartnerConfig: oracle,
				PartnerAEndConfig:   aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = aEndPartnerConfig
		case "azure":
			if aPartnerConfig.AzurePartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Azure Partner configuration is required",
				)
				return
			}
			var azureConfig vxcPartnerConfigAzureModel
			azureDiags := aPartnerConfig.AzurePartnerConfig.As(ctx, &azureConfig, basetypes.ObjectAsOptions{})
			if azureDiags.HasError() {
				resp.Diagnostics.Append(azureDiags...)
				return
			}
			partnerPortReq := &megaport.LookupPartnerPortsRequest{
				Key:       azureConfig.ServiceKey.ValueString(),
				PortSpeed: int(plan.RateLimit.ValueInt64()),
				Partner:   "AZURE",
			}
			partnerPortReq.ProductID = a.RequestedProductUID.ValueString()
			partnerPortRes, err := r.client.VXCService.LookupPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", plan.Name.ValueString(), err.Error()),
				)
				return
			}
			aEndConfig.ProductUID = partnerPortRes.ProductUID
			aEndPartnerConfig := megaport.VXCPartnerConfigAzure{
				ConnectType: "AZURE",
				ServiceKey:  azureConfig.ServiceKey.ValueString(),
			}

			azureConfigObj, azureDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAzureAttrs, azureConfig)
			resp.Diagnostics.Append(azureDiags...)

			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:             aPartnerConfig.Partner,
				AWSPartnerConfig:    aws,
				AzurePartnerConfig:  azureConfigObj,
				GooglePartnerConfig: google,
				OraclePartnerConfig: oracle,
				PartnerAEndConfig:   aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = aEndPartnerConfig
		case "google":
			if aPartnerConfig.GooglePartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Google Partner configuration is required",
				)
				return
			}
			var googleConfig vxcPartnerConfigGoogleModel
			googleDiags := aPartnerConfig.GooglePartnerConfig.As(ctx, &googleConfig, basetypes.ObjectAsOptions{})
			if googleDiags.HasError() {
				resp.Diagnostics.Append(googleDiags...)
				return
			}
			aEndPartnerConfig := megaport.VXCPartnerConfigGoogle{
				ConnectType: "GOOGLE",
				PairingKey:  googleConfig.PairingKey.ValueString(),
			}
			googleConfigObj, googleDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigGoogleAttrs, googleConfig)
			resp.Diagnostics.Append(googleDiags...)

			partnerPortReq := &megaport.LookupPartnerPortsRequest{
				Key:       googleConfig.PairingKey.ValueString(),
				PortSpeed: int(plan.RateLimit.ValueInt64()),
				Partner:   "GOOGLE",
			}
			partnerPortReq.ProductID = a.RequestedProductUID.ValueString()
			partnerPortRes, err := r.client.VXCService.LookupPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", plan.Name.ValueString(), err.Error()),
				)
				return
			}
			aEndConfig.ProductUID = partnerPortRes.ProductUID

			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:             aPartnerConfig.Partner,
				AWSPartnerConfig:    aws,
				AzurePartnerConfig:  azure,
				GooglePartnerConfig: googleConfigObj,
				OraclePartnerConfig: oracle,
				PartnerAEndConfig:   aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = partnerConfigObj

			aEndConfig.PartnerConfig = aEndPartnerConfig
		case "oracle":
			if aPartnerConfig.OraclePartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Oracle Partner configuration is required",
				)
				return
			}
			var oracleConfig vxcPartnerConfigOracleModel
			oracleDiags := aPartnerConfig.OraclePartnerConfig.As(ctx, &oracleConfig, basetypes.ObjectAsOptions{})
			if oracleDiags.HasError() {
				resp.Diagnostics.Append(oracleDiags...)
				return
			}
			aEndPartnerConfig := &megaport.VXCPartnerConfigOracle{
				ConnectType:      "ORACLE",
				VirtualCircuitId: oracleConfig.VirtualCircuitId.ValueString(),
			}

			partnerPortReq := &megaport.LookupPartnerPortsRequest{
				Key:       oracleConfig.VirtualCircuitId.ValueString(),
				PortSpeed: int(plan.RateLimit.ValueInt64()),
				Partner:   "ORACLE",
			}
			partnerPortReq.ProductID = a.RequestedProductUID.ValueString()

			partnerPortRes, err := r.client.VXCService.LookupPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", plan.Name.ValueString(), err.Error()),
				)
				return
			}
			aEndConfig.ProductUID = partnerPortRes.ProductUID

			oracleConfigObj, oracleDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigOracleAttrs, oracleConfig)
			resp.Diagnostics.Append(oracleDiags...)

			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:             aPartnerConfig.Partner,
				AWSPartnerConfig:    aws,
				AzurePartnerConfig:  azure,
				GooglePartnerConfig: google,
				OraclePartnerConfig: oracleConfigObj,
				PartnerAEndConfig:   aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = aEndPartnerConfig
		case "a-end":
			if aPartnerConfig.PartnerAEndConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": A-End Partner configuration is required",
				)
				return
			}
			var partnerConfigAEnd vxcPartnerConfigAEndModel
			aEndDiags := aPartnerConfig.PartnerAEndConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
			if aEndDiags.HasError() {
				resp.Diagnostics.Append(aEndDiags...)
				return
			}
			prefixFilterListRes, err := r.client.MCRService.GetMCRPrefixFilterLists(ctx, a.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			aEndMegaportConfig := megaport.VXCOrderAEndPartnerConfig{}
			ifaceModels := []*vxcPartnerConfigInterfaceModel{}
			ifaceDiags := partnerConfigAEnd.Interfaces.ElementsAs(ctx, &ifaceModels, false)
			resp.Diagnostics = append(resp.Diagnostics, ifaceDiags...)
			for _, iface := range ifaceModels {
				toAppend := megaport.PartnerConfigInterface{}
				if !iface.IPAddresses.IsNull() {
					ipAddresses := []string{}
					ipDiags := iface.IPAddresses.ElementsAs(ctx, &ipAddresses, true)
					resp.Diagnostics = append(resp.Diagnostics, ipDiags...)
					toAppend.IpAddresses = ipAddresses
				}
				if !iface.IPRoutes.IsNull() {
					ipRoutes := []*ipRouteModel{}
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, ipRoutes, false)
					resp.Diagnostics = append(resp.Diagnostics, ipRouteDiags...)
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
					resp.Diagnostics = append(resp.Diagnostics, natDiags...)
					toAppend.NatIpAddresses = natIPAddresses
				}
				if !iface.Bfd.IsNull() {
					bfd := &bfdConfigModel{}
					bfdDiags := iface.Bfd.As(ctx, bfd, basetypes.ObjectAsOptions{})
					resp.Diagnostics = append(resp.Diagnostics, bfdDiags...)
					toAppend.Bfd = megaport.BfdConfig{
						TxInterval: int(bfd.TxInterval.ValueInt64()),
						RxInterval: int(bfd.RxInterval.ValueInt64()),
						Multiplier: int(bfd.Multiplier.ValueInt64()),
					}
				}
				if !iface.BgpConnections.IsNull() {
					bgpConnections := []*bgpConnectionConfigModel{}
					bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConnections, false)
					resp.Diagnostics = append(resp.Diagnostics, bgpDiags...)
					for _, bgpConnection := range bgpConnections {
						bgpToAppend := megaport.BgpConnectionConfig{
							PeerAsn:        int(bgpConnection.PeerAsn.ValueInt64()),
							LocalIpAddress: bgpConnection.LocalIPAddress.ValueString(),
							PeerIpAddress:  bgpConnection.PeerIPAddress.ValueString(),
							Password:       bgpConnection.Password.ValueString(),
							Shutdown:       bgpConnection.Shutdown.ValueBool(),
							Description:    bgpConnection.Description.ValueString(),
							MedIn:          int(bgpConnection.MedIn.ValueInt64()),
							MedOut:         int(bgpConnection.MedOut.ValueInt64()),
							BfdEnabled:     bgpConnection.BfdEnabled.ValueBool(),
							ExportPolicy:   bgpConnection.ExportPolicy.ValueString(),
						}
						if !bgpConnection.ImportWhitelist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ImportWhitelist.ValueString() {
									bgpToAppend.ImportWhitelist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ImportBlacklist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ImportBlacklist.ValueString() {
									bgpToAppend.ImportBlacklist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ExportWhitelist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ExportWhitelist.ValueString() {
									bgpToAppend.ExportWhitelist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ExportBlacklist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ExportBlacklist.ValueString() {
									bgpToAppend.ExportBlacklist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.PermitExportTo.IsNull() {
							permitExportTo := []string{}
							permitDiags := bgpConnection.PermitExportTo.ElementsAs(ctx, &permitExportTo, true)
							resp.Diagnostics = append(resp.Diagnostics, permitDiags...)
							bgpToAppend.PermitExportTo = permitExportTo
							bgpToAppend.PermitExportTo = permitExportTo
						}
						if !bgpConnection.DenyExportTo.IsNull() {
							denyExportTo := []string{}
							denyDiags := bgpConnection.DenyExportTo.ElementsAs(ctx, &denyExportTo, true)
							resp.Diagnostics = append(resp.Diagnostics, denyDiags...)
							bgpToAppend.DenyExportTo = denyExportTo
						}
						toAppend.BgpConnections = append(toAppend.BgpConnections, bgpToAppend)
					}
				}
				aEndMegaportConfig.Interfaces = append(aEndMegaportConfig.Interfaces, toAppend)
			}
			aEndConfigObj, aEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAEndAttrs, partnerConfigAEnd)
			resp.Diagnostics.Append(aEndDiags...)
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:             aPartnerConfig.Partner,
				AWSPartnerConfig:    aws,
				AzurePartnerConfig:  azure,
				GooglePartnerConfig: google,
				OraclePartnerConfig: oracle,
				PartnerAEndConfig:   aEndConfigObj,
			}
			aEndPartnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = aEndPartnerConfigObj
			aEndConfig.PartnerConfig = aEndMegaportConfig
		case "transit":
			aEndPartnerConfig := &megaport.VXCPartnerConfigTransit{
				ConnectType: "TRANSIT",
			}
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:             aPartnerConfig.Partner,
				AWSPartnerConfig:    aws,
				AzurePartnerConfig:  azure,
				GooglePartnerConfig: google,
				OraclePartnerConfig: oracle,
				PartnerAEndConfig:   aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = aEndPartnerConfig
		default:
			resp.Diagnostics.AddError(
				"Error creating VXC",
				"Could not create VXC with name "+plan.Name.ValueString()+": Partner configuration not supported",
			)
			return
		}
	}

	buyReq.AEndConfiguration = *aEndConfig

	var b vxcEndConfigurationModel
	bEndDiags := bEndObj.As(ctx, &b, basetypes.ObjectAsOptions{})
	if bEndDiags.HasError() {
		resp.Diagnostics.Append(bEndDiags...)
		return
	}
	bEndConfig := &megaport.VXCOrderEndpointConfiguration{
		ProductUID: b.RequestedProductUID.ValueString(),
		VLAN:       int(b.VLAN.ValueInt64()),
	}
	if !b.OrderedVLAN.IsNull() {
		bEndConfig.VLAN = int(b.OrderedVLAN.ValueInt64())
	} else {
		bEndConfig.VLAN = 0
	}
	if !b.InnerVLAN.IsNull() && !b.NetworkInterfaceIndex.IsNull() {
		bEndConfig.VXCOrderMVEConfig = &megaport.VXCOrderMVEConfig{
			InnerVLAN:             int(b.InnerVLAN.ValueInt64()),
			NetworkInterfaceIndex: int(b.NetworkInterfaceIndex.ValueInt64()),
		}
	}
	if !plan.BEndPartnerConfig.IsNull() {
		var bPartnerConfig vxcPartnerConfigurationModel
		bPartnerDiags := plan.BEndPartnerConfig.As(ctx, &bPartnerConfig, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(bPartnerDiags...)
		switch bPartnerConfig.Partner.ValueString() {
		case "aws":
			if bPartnerConfig.AWSPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": AWS Partner configuration is required",
				)
				return
			}
			var awsConfig vxcPartnerConfigAWSModel
			awsDiags := bPartnerConfig.AWSPartnerConfig.As(ctx, &awsConfig, basetypes.ObjectAsOptions{})
			if awsDiags.HasError() {
				resp.Diagnostics.Append(awsDiags...)
				return
			}
			bEndPartnerConfig := megaport.VXCPartnerConfigAWS{
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
			resp.Diagnostics.Append(awsDiags...)

			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:             bPartnerConfig.Partner,
				AWSPartnerConfig:    awsConfigObj,
				AzurePartnerConfig:  azure,
				GooglePartnerConfig: google,
				OraclePartnerConfig: oracle,
				PartnerAEndConfig:   aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = bEndPartnerConfig
		case "azure":
			if bPartnerConfig.AzurePartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Azure Partner configuration is required",
				)
				return
			}
			var azureConfig vxcPartnerConfigAzureModel
			azureDiags := bPartnerConfig.AzurePartnerConfig.As(ctx, &azureConfig, basetypes.ObjectAsOptions{})
			if azureDiags.HasError() {
				resp.Diagnostics.Append(azureDiags...)
				return
			}
			bEndPartnerConfig := megaport.VXCPartnerConfigAzure{
				ConnectType: "AZURE",
				ServiceKey:  azureConfig.ServiceKey.ValueString(),
			}

			azureConfigObj, azureDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAzureAttrs, azureConfig)
			resp.Diagnostics.Append(azureDiags...)

			partnerPortReq := &megaport.LookupPartnerPortsRequest{
				Key:       azureConfig.ServiceKey.ValueString(),
				PortSpeed: int(plan.RateLimit.ValueInt64()),
				Partner:   "AZURE",
			}
			if !b.RequestedProductUID.IsNull() {
				partnerPortReq.ProductID = b.RequestedProductUID.ValueString()
			}
			partnerPortRes, err := r.client.VXCService.LookupPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", plan.Name.ValueString(), err.Error()),
				)
				return
			}
			bEndConfig.ProductUID = partnerPortRes.ProductUID

			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:             bPartnerConfig.Partner,
				AWSPartnerConfig:    aws,
				AzurePartnerConfig:  azureConfigObj,
				GooglePartnerConfig: google,
				OraclePartnerConfig: oracle,
				PartnerAEndConfig:   aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = bEndPartnerConfig
		case "google":
			if bPartnerConfig.GooglePartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Google Partner configuration is required",
				)
				return
			}
			var googleConfig vxcPartnerConfigGoogleModel
			googleDiags := bPartnerConfig.GooglePartnerConfig.As(ctx, &googleConfig, basetypes.ObjectAsOptions{})
			if googleDiags.HasError() {
				resp.Diagnostics.Append(googleDiags...)
				return
			}
			bEndPartnerConfig := megaport.VXCPartnerConfigGoogle{
				ConnectType: "GOOGLE",
				PairingKey:  googleConfig.PairingKey.ValueString(),
			}
			googleConfigObj, googleDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigGoogleAttrs, googleConfig)
			resp.Diagnostics.Append(googleDiags...)

			partnerPortReq := &megaport.LookupPartnerPortsRequest{
				Key:       googleConfig.PairingKey.ValueString(),
				PortSpeed: int(plan.RateLimit.ValueInt64()),
				Partner:   "GOOGLE",
			}
			if !b.RequestedProductUID.IsNull() {
				partnerPortReq.ProductID = b.RequestedProductUID.ValueString()
			}
			partnerPortRes, err := r.client.VXCService.LookupPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", plan.Name.ValueString(), err.Error()),
				)
				return
			}
			bEndConfig.ProductUID = partnerPortRes.ProductUID

			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:             bPartnerConfig.Partner,
				AWSPartnerConfig:    aws,
				AzurePartnerConfig:  azure,
				GooglePartnerConfig: googleConfigObj,
				OraclePartnerConfig: oracle,
				PartnerAEndConfig:   aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = bEndPartnerConfig
		case "oracle":
			if bPartnerConfig.OraclePartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Oracle Partner configuration is required",
				)
				return
			}
			var oracleConfig vxcPartnerConfigOracleModel
			oracleDiags := bPartnerConfig.OraclePartnerConfig.As(ctx, &oracleConfig, basetypes.ObjectAsOptions{})
			if oracleDiags.HasError() {
				resp.Diagnostics.Append(oracleDiags...)
				return
			}
			bEndPartnerConfig := megaport.VXCPartnerConfigOracle{
				ConnectType:      "ORACLE",
				VirtualCircuitId: oracleConfig.VirtualCircuitId.ValueString(),
			}

			partnerPortReq := &megaport.LookupPartnerPortsRequest{
				Key:       oracleConfig.VirtualCircuitId.ValueString(),
				PortSpeed: int(plan.RateLimit.ValueInt64()),
				Partner:   "ORACLE",
			}
			if !b.RequestedProductUID.IsNull() {
				partnerPortReq.ProductID = b.RequestedProductUID.ValueString()
			}
			partnerPortRes, err := r.client.VXCService.LookupPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", plan.Name.ValueString(), err.Error()),
				)
				return
			}
			bEndConfig.ProductUID = partnerPortRes.ProductUID

			oracleConfigObj, oracleDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigOracleAttrs, oracleConfig)
			resp.Diagnostics.Append(oracleDiags...)

			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:             bPartnerConfig.Partner,
				AWSPartnerConfig:    aws,
				AzurePartnerConfig:  azure,
				GooglePartnerConfig: google,
				OraclePartnerConfig: oracleConfigObj,
				PartnerAEndConfig:   aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = bEndPartnerConfig
		case "transit":
			bEndPartnerConfig := &megaport.VXCPartnerConfigTransit{
				ConnectType: "TRANSIT",
			}
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:             bPartnerConfig.Partner,
				AWSPartnerConfig:    aws,
				AzurePartnerConfig:  azure,
				GooglePartnerConfig: google,
				OraclePartnerConfig: oracle,
				PartnerAEndConfig:   aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = bEndPartnerConfig
		default:
			resp.Diagnostics.AddError(
				"Error creating VXC",
				"Could not create VXC with name "+plan.Name.ValueString()+": Partner configuration not supported",
			)
			return
		}
	}

	buyReq.BEndConfiguration = *bEndConfig

	buyReq.BEndConfiguration = *bEndConfig

	createdVXC, err := r.client.VXCService.BuyVXC(ctx, buyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating VXC",
			"Could not order VXC with name "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	createdID := createdVXC.TechnicalServiceUID

	// get the created VXC
	vxc, err := r.client.VXCService.GetVXC(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading newly created VXC",
			"Could not read newly created VXC with ID "+createdID+": "+err.Error(),
		)
		return
	}

	// update the plan with the VXC info
	apiDiags := plan.fromAPIVXC(ctx, vxc)
	resp.Diagnostics.Append(apiDiags...)

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *vxcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state vxcResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed vxc value from API
	vxc, err := r.client.VXCService.GetVXC(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VXC",
			"Could not read VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	apiDiags := state.fromAPIVXC(ctx, vxc)
	resp.Diagnostics.Append(apiDiags...)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *vxcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vxcResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var name, costCentre, aEndProductUID, bEndProductUID string
	var aEndVlan, bEndVlan, rateLimit, term int
	var shutdown bool
	if !plan.Name.Equal(state.Name) {
		name = plan.Name.ValueString()
	}

	var aEndPlan, bEndPlan, aEndState, bEndState vxcEndConfigurationModel

	aEndPlanDiags := plan.AEndConfiguration.As(ctx, &aEndPlan, basetypes.ObjectAsOptions{})
	if aEndPlanDiags.HasError() {
		resp.Diagnostics.Append(aEndPlanDiags...)
		return
	}
	bEndPlanDiags := plan.BEndConfiguration.As(ctx, &bEndPlan, basetypes.ObjectAsOptions{})
	if bEndPlanDiags.HasError() {
		resp.Diagnostics.Append(bEndPlanDiags...)
		return
	}

	aEndStateDiags := state.AEndConfiguration.As(ctx, &aEndState, basetypes.ObjectAsOptions{})
	if aEndStateDiags.HasError() {
		resp.Diagnostics.Append(aEndStateDiags...)
		return
	}
	bEndStateDiags := state.BEndConfiguration.As(ctx, &bEndState, basetypes.ObjectAsOptions{})
	if bEndStateDiags.HasError() {
		resp.Diagnostics.Append(bEndStateDiags...)
		return
	}

	if !aEndPlan.VLAN.IsNull() && !aEndPlan.VLAN.Equal(aEndState.VLAN) {
		aEndVlan = int(aEndPlan.VLAN.ValueInt64())
		aEndState.VLAN = aEndPlan.VLAN
	} else {
		aEndVlan = int(aEndState.VLAN.ValueInt64())
	}
	if !bEndPlan.VLAN.IsNull() && !bEndPlan.VLAN.Equal(bEndState.VLAN) {
		bEndVlan = int(bEndPlan.VLAN.ValueInt64())
		bEndState.VLAN = bEndPlan.VLAN
	} else {
		bEndVlan = int(bEndState.VLAN.ValueInt64())
	}

	if !plan.RateLimit.IsNull() && !plan.RateLimit.Equal(state.RateLimit) {
		rateLimit = int(plan.RateLimit.ValueInt64())
	} else {
		rateLimit = int(state.RateLimit.ValueInt64())
	}
	if !plan.CostCentre.IsNull() && !plan.CostCentre.Equal(state.CostCentre) {
		costCentre = plan.CostCentre.ValueString()
	} else {
		costCentre = state.CostCentre.ValueString()
	}
	if !plan.Shutdown.IsNull() && !plan.Shutdown.Equal(state.Shutdown) {
		shutdown = plan.Shutdown.ValueBool()
	} else {
		shutdown = state.Shutdown.ValueBool()
	}
	if !plan.ContractTermMonths.IsNull() && !plan.ContractTermMonths.Equal(state.ContractTermMonths) {
		term = int(plan.ContractTermMonths.ValueInt64())
	} else {
		term = int(state.ContractTermMonths.ValueInt64())
	}

	updateReq := &megaport.UpdateVXCRequest{
		Name:          &name,
		AEndVLAN:      &aEndVlan,
		BEndVLAN:      &bEndVlan,
		CostCentre:    &costCentre,
		Shutdown:      &shutdown,
		RateLimit:     &rateLimit,
		Term:          &term,
		WaitForUpdate: true,
		WaitForTime:   waitForTime,
	}

	if !aEndPlan.RequestedProductUID.IsNull() && !aEndPlan.RequestedProductUID.Equal(aEndState.RequestedProductUID) {
		aEndProductUID = aEndPlan.RequestedProductUID.ValueString()
		updateReq.AEndProductUID = &aEndProductUID
		aEndState.RequestedProductUID = aEndPlan.RequestedProductUID
	}
	if !bEndPlan.RequestedProductUID.IsNull() && !bEndPlan.RequestedProductUID.Equal(bEndState.RequestedProductUID) {
		bEndProductUID = bEndPlan.RequestedProductUID.ValueString()
		updateReq.BEndProductUID = &bEndProductUID
		bEndState.RequestedProductUID = bEndPlan.RequestedProductUID
	}

	_, err := r.client.VXCService.UpdateVXC(ctx, plan.UID.ValueString(), updateReq)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating VXC",
			"Could not update VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state with any changes from plan configuration following successful update
	aEndStateObj, aEndStateDiags := types.ObjectValueFrom(ctx, vxcEndConfigurationAttrs, aEndState)
	resp.Diagnostics.Append(aEndStateDiags...)
	state.AEndConfiguration = aEndStateObj
	bEndStateObj, bEndStateDiags := types.ObjectValueFrom(ctx, vxcEndConfigurationAttrs, bEndState)
	resp.Diagnostics.Append(bEndStateDiags...)
	state.BEndConfiguration = bEndStateObj

	// Get refreshed vxc value from API
	vxc, err := r.client.VXCService.GetVXC(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VXC",
			"Could not read VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	apiDiags := state.fromAPIVXC(ctx, vxc)
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(apiDiags...)

	// Set refreshed state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *vxcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state vxcResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	err := r.client.VXCService.DeleteVXC(ctx, state.UID.ValueString(), &megaport.DeleteVXCRequest{
		DeleteNow: true,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting VXC",
			"Could not delete VXC, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *vxcResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*megaport.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *megaport.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *vxcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}

func fromAPICSPConnection(ctx context.Context, c megaport.CSPConnectionConfig) (types.Object, diag.Diagnostics) {
	apiDiags := diag.Diagnostics{}
	switch provider := c.(type) {
	case megaport.CSPConnectionAWS:
		awsModel := &cspConnectionModel{
			ConnectType:       types.StringValue(provider.ConnectType),
			ResourceName:      types.StringValue(provider.ResourceName),
			ResourceType:      types.StringValue(provider.ResourceType),
			VLAN:              types.Int64Value(int64(provider.VLAN)),
			Account:           types.StringValue(provider.Account),
			AmazonAddress:     types.StringValue(provider.AmazonAddress),
			ASN:               types.Int64Value(int64(provider.ASN)),
			AuthKey:           types.StringValue(provider.AuthKey),
			CustomerAddress:   types.StringValue(provider.CustomerAddress),
			CustomerIPAddress: types.StringValue(provider.CustomerIPAddress),
			ID:                types.Int64Value(int64(provider.ID)),
			Name:              types.StringValue(provider.Name),
			OwnerAccount:      types.StringValue(provider.OwnerAccount),
			PeerASN:           types.Int64Value(int64(provider.PeerASN)),
			Type:              types.StringValue(provider.Type),
			VIFID:             types.StringValue(provider.VIFID),
		}
		awsModel.Bandwidths = types.ListNull(types.Int64Type)
		awsModel.IPAddresses = types.ListNull(types.StringType)
		awsObject, awsDiags := types.ObjectValueFrom(ctx, cspConnectionFullAttrs, awsModel)
		apiDiags = append(apiDiags, awsDiags...)
		return awsObject, apiDiags
	case megaport.CSPConnectionAWSHC:
		awsHCModel := &cspConnectionModel{
			ConnectType:  types.StringValue(provider.ConnectType),
			ResourceName: types.StringValue(provider.ResourceName),
			ResourceType: types.StringValue(provider.ResourceType),
			Bandwidth:    types.Int64Value(int64(provider.Bandwidth)),
			Name:         types.StringValue(provider.Name),
			OwnerAccount: types.StringValue(provider.OwnerAccount),
			ConnectionID: types.StringValue(provider.ConnectionID),
		}
		bandwidths := []int64{}
		for _, b := range provider.Bandwidths {
			bandwidths = append(bandwidths, int64(b))
		}
		bandwidthList, bandwidthDiags := types.ListValueFrom(ctx, types.Int64Type, bandwidths)
		apiDiags = append(apiDiags, bandwidthDiags...)
		awsHCModel.Bandwidths = bandwidthList
		awsHCModel.IPAddresses = types.ListNull(types.StringType)
		awsHCObject, awsHCDiags := types.ObjectValueFrom(ctx, cspConnectionFullAttrs, awsHCModel)
		apiDiags = append(apiDiags, awsHCDiags...)
		return awsHCObject, apiDiags
	case megaport.CSPConnectionAzure:
		azureModel := &cspConnectionModel{
			ConnectType:  types.StringValue(provider.ConnectType),
			ResourceName: types.StringValue(provider.ResourceName),
			ResourceType: types.StringValue(provider.ResourceType),
			Bandwidth:    types.Int64Value(int64(provider.Bandwidth)),
			Managed:      types.BoolValue(provider.Managed),
			ServiceKey:   types.StringValue(provider.ServiceKey),
			VLAN:         types.Int64Value(int64(provider.VLAN)),
		}
		azureModel.Bandwidths = types.ListNull(types.Int64Type)
		azureModel.IPAddresses = types.ListNull(types.StringType)
		azureObject, azureObjDiags := types.ObjectValueFrom(ctx, cspConnectionFullAttrs, azureModel)
		apiDiags = append(apiDiags, azureObjDiags...)
		return azureObject, apiDiags
	case megaport.CSPConnectionGoogle:
		googleModel := &cspConnectionModel{
			ConnectType:  types.StringValue(provider.ConnectType),
			ResourceName: types.StringValue(provider.ResourceName),
			ResourceType: types.StringValue(provider.ResourceType),
			Bandwidth:    types.Int64Value(int64(provider.Bandwidth)),
			CSPName:      types.StringValue(provider.CSPName),
			PairingKey:   types.StringValue(provider.PairingKey),
		}
		bandwidths := []int64{}

		for _, b := range provider.Bandwidths {
			bandwidths = append(bandwidths, int64(b))
		}
		googleModel.IPAddresses = types.ListNull(types.StringType)
		bandwidthList, bwListDiags := types.ListValueFrom(ctx, types.Int64Type, bandwidths)
		apiDiags = append(apiDiags, bwListDiags...)
		googleModel.Bandwidths = bandwidthList
		googleObject, googleObjDiags := types.ObjectValueFrom(ctx, cspConnectionFullAttrs, googleModel)
		apiDiags = append(apiDiags, googleObjDiags...)
		return googleObject, apiDiags
	case megaport.CSPConnectionVirtualRouter:
		virtualRouterModel := &cspConnectionModel{
			ConnectType:       types.StringValue(provider.ConnectType),
			ResourceName:      types.StringValue(provider.ResourceName),
			ResourceType:      types.StringValue(provider.ResourceType),
			VLAN:              types.Int64Value(int64(provider.VLAN)),
			VirtualRouterName: types.StringValue(provider.VirtualRouterName),
		}
		virtualRouterModel.Bandwidths = types.ListNull(types.Int64Type)
		ipAddresses := []string{}
		ipAddresses = append(ipAddresses, ipAddresses...)
		ipList, ipListDiags := types.ListValueFrom(ctx, types.StringType, ipAddresses)
		apiDiags = append(apiDiags, ipListDiags...)
		virtualRouterModel.IPAddresses = ipList
		virtualRouterObject, vrObjDiags := types.ObjectValueFrom(ctx, cspConnectionFullAttrs, virtualRouterModel)
		apiDiags = append(apiDiags, vrObjDiags...)
		return virtualRouterObject, apiDiags
	case megaport.CSPConnectionTransit:
		transitModel := &cspConnectionModel{
			ConnectType:        types.StringValue(provider.ConnectType),
			ResourceName:       types.StringValue(provider.ResourceName),
			ResourceType:       types.StringValue(provider.ResourceType),
			CustomerIP4Address: types.StringValue(provider.CustomerIP4Address),
			CustomerIP6Network: types.StringValue(provider.CustomerIP6Network),
			IPv4GatewayAddress: types.StringValue(provider.IPv4GatewayAddress),
			IPv6GatewayAddress: types.StringValue(provider.IPv6GatewayAddress),
		}
		transitModel.Bandwidths = types.ListNull(types.Int64Type)
		transitModel.IPAddresses = types.ListNull(types.StringType)
		transitObject, transitObjectDiags := types.ObjectValueFrom(ctx, cspConnectionFullAttrs, transitModel)
		apiDiags = append(apiDiags, transitObjectDiags...)
		return transitObject, apiDiags
	}
	apiDiags.AddError("Error creating CSP Connection", "Could not create CSP Connection, unknown type")
	return types.ObjectNull(cspConnectionFullAttrs), apiDiags
}
