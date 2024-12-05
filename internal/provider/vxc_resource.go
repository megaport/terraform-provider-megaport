package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
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

	vxcPartnerConfigAttrs = map[string]attr.Type{
		"partner":              types.StringType,
		"aws_config":           types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigAWSAttrs),
		"azure_config":         types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigAzureAttrs),
		"google_config":        types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigGoogleAttrs),
		"oracle_config":        types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigOracleAttrs),
		"vrouter_config":       types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigVrouterAttrs),
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
		"port_choice": types.StringType,
		"peers":       types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(partnerOrderAzurePeeringConfigAttrs)),
	}

	partnerOrderAzurePeeringConfigAttrs = map[string]attr.Type{
		"type":             types.StringType,
		"peer_asn":         types.StringType,
		"primary_subnet":   types.StringType,
		"secondary_subnet": types.StringType,
		"prefixes":         types.StringType,
		"shared_key":       types.StringType,
		"vlan":             types.Int64Type,
	}

	vxcPartnerConfigGoogleAttrs = map[string]attr.Type{
		"pairing_key": types.StringType,
	}

	vxcPartnerConfigOracleAttrs = map[string]attr.Type{
		"virtual_circuit_id": types.StringType,
	}

	// the below structs are deprecated, but need to be here and different than the vrouter_partner_config, because we would need
	// to keep the schema updated for both if they used the same structs.

	// deprecated
	vxcPartnerConfigAEndAttrs = map[string]attr.Type{
		"interfaces": types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(vxcInterfaceAttrs)),
	}

	// deprecated
	vxcInterfaceAttrs = map[string]attr.Type{
		"ip_addresses":     types.ListType{}.WithElementType(types.StringType),
		"ip_routes":        types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(ipRouteAttrs)),
		"nat_ip_addresses": types.ListType{}.WithElementType(types.StringType),
		"bfd":              types.ObjectType{}.WithAttributeTypes(bfdConfigAttrs),
		"bgp_connections":  types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(bgpConnectionConfig)),
	}

	// deprecated
	bgpConnectionConfig = map[string]attr.Type{
		"peer_asn":              types.Int64Type,
		"local_ip_address":      types.StringType,
		"peer_ip_address":       types.StringType,
		"password":              types.StringType,
		"shutdown":              types.BoolType,
		"description":           types.StringType,
		"med_in":                types.Int64Type,
		"med_out":               types.Int64Type,
		"bfd_enabled":           types.BoolType,
		"export_policy":         types.StringType,
		"permit_export_to":      types.ListType{}.WithElementType(types.StringType),
		"deny_export_to":        types.ListType{}.WithElementType(types.StringType),
		"import_whitelist":      types.StringType,
		"import_blacklist":      types.StringType,
		"export_whitelist":      types.StringType,
		"export_blacklist":      types.StringType,
		"as_path_prepend_count": types.Int64Type,
	}

	vxcPartnerConfigVrouterAttrs = map[string]attr.Type{
		"interfaces": types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(vxcVrouterInterfaceAttrs)),
	}

	vxcVrouterInterfaceAttrs = map[string]attr.Type{
		"ip_addresses":     types.ListType{}.WithElementType(types.StringType),
		"ip_routes":        types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(ipRouteAttrs)),
		"nat_ip_addresses": types.ListType{}.WithElementType(types.StringType),
		"bfd":              types.ObjectType{}.WithAttributeTypes(bfdConfigAttrs),
		"vlan":             types.Int64Type,
		"bgp_connections":  types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(bgpVrouterConnectionConfig)),
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

	bgpVrouterConnectionConfig = map[string]attr.Type{
		"peer_type":             types.StringType,
		"peer_asn":              types.Int64Type,
		"local_ip_address":      types.StringType,
		"peer_ip_address":       types.StringType,
		"password":              types.StringType,
		"shutdown":              types.BoolType,
		"description":           types.StringType,
		"med_in":                types.Int64Type,
		"med_out":               types.Int64Type,
		"bfd_enabled":           types.BoolType,
		"export_policy":         types.StringType,
		"permit_export_to":      types.ListType{}.WithElementType(types.StringType),
		"deny_export_to":        types.ListType{}.WithElementType(types.StringType),
		"import_whitelist":      types.StringType,
		"import_blacklist":      types.StringType,
		"export_whitelist":      types.StringType,
		"export_blacklist":      types.StringType,
		"as_path_prepend_count": types.Int64Type,
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
	ServiceKey         types.String `tfsdk:"service_key"`

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

	CSPConnections types.List `tfsdk:"csp_connections"`
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
}

type vxcPartnerConfigurationModel struct {
	Partner              types.String `tfsdk:"partner"`
	AWSPartnerConfig     types.Object `tfsdk:"aws_config"`
	AzurePartnerConfig   types.Object `tfsdk:"azure_config"`
	GooglePartnerConfig  types.Object `tfsdk:"google_config"`
	OraclePartnerConfig  types.Object `tfsdk:"oracle_config"`
	VrouterPartnerConfig types.Object `tfsdk:"vrouter_config"`
	PartnerAEndConfig    types.Object `tfsdk:"partner_a_end_config"` // DEPRECATED: Use vrouter_config instead.
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
	PortChoice types.String `tfsdk:"port_choice"`
	Peers      types.List   `tfsdk:"peers"`
}

type partnerOrderAzurePeeringConfigModel struct {
	Type            types.String `tfsdk:"type"`
	PeerASN         types.String `tfsdk:"peer_asn"`
	PrimarySubnet   types.String `tfsdk:"primary_subnet"`
	SecondarySubnet types.String `tfsdk:"secondary_subnet"`
	Prefixes        types.String `tfsdk:"prefixes"`
	SharedKey       types.String `tfsdk:"shared_key"`
	VLAN            types.Int64  `tfsdk:"vlan"`
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

// vxcPartnerConfigVrouterModel maps the partner configuration schema data for a vrouter configuration.
type vxcPartnerConfigVrouterModel struct {
	vxcPartnerConfig
	Interfaces types.List `tfsdk:"interfaces"`
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
	VLAN           types.Int64  `tfsdk:"vlan"`
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
	PeerAsn            types.Int64  `tfsdk:"peer_asn"`
	PeerType           types.String `tfsdk:"peer_type"`
	LocalIPAddress     types.String `tfsdk:"local_ip_address"`
	PeerIPAddress      types.String `tfsdk:"peer_ip_address"`
	Password           types.String `tfsdk:"password"`
	Shutdown           types.Bool   `tfsdk:"shutdown"`
	Description        types.String `tfsdk:"description"`
	MedIn              types.Int64  `tfsdk:"med_in"`
	MedOut             types.Int64  `tfsdk:"med_out"`
	BfdEnabled         types.Bool   `tfsdk:"bfd_enabled"`
	ExportPolicy       types.String `tfsdk:"export_policy"`
	PermitExportTo     types.List   `tfsdk:"permit_export_to"`
	DenyExportTo       types.List   `tfsdk:"deny_export_to"`
	ImportWhitelist    types.String `tfsdk:"import_whitelist"`
	ImportBlacklist    types.String `tfsdk:"import_blacklist"`
	ExportWhitelist    types.String `tfsdk:"export_whitelist"`
	ExportBlacklist    types.String `tfsdk:"export_blacklist"`
	AsPathPrependCount types.Int64  `tfsdk:"as_path_prepend_count"`
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
	var aEndOrderedVLAN, bEndOrderedVLAN, aEndInnerVLAN, bEndInnerVLAN *int64
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
	if aEndInnerVLAN != nil {
		aEndModel.InnerVLAN = types.Int64Value(*aEndInnerVLAN)
	} else {
		aEndModel.InnerVLAN = types.Int64PointerValue(nil)
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
	if bEndInnerVLAN != nil {
		bEndModel.InnerVLAN = types.Int64Value(*bEndInnerVLAN)
	} else {
		bEndModel.InnerVLAN = types.Int64PointerValue(nil)
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
			"service_key": schema.StringAttribute{
				Description: "The service key of the VXC.",
				Optional:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"product_name": schema.StringAttribute{
				Description: "The name of the product.",
				Required:    true,
			},
			"service_id": schema.Int64Attribute{
				Description: "The service ID of the VXC.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"rate_limit": schema.Int64Attribute{
				Description: "The rate limit of the product.",
				Required:    true,
			},
			"product_type": schema.StringAttribute{
				Description: "The type of the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"distance_band": schema.StringAttribute{
				Description: "The distance band of the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"provisioning_status": schema.StringAttribute{
				Description: "The provisioning status of the product.",
				Computed:    true,
			},
			"secondary_name": schema.StringAttribute{
				Description: "The secondary name of the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"usage_algorithm": schema.StringAttribute{
				Description: "The usage algorithm of the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"promo_code": schema.StringAttribute{
				Description: "Promo code is an optional string that can be used to enter a promotional code for the service order. The code is not validated, so if the code doesn't exist or doesn't work for the service, the request will still be successful.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"cost_centre": schema.StringAttribute{
				Description: "A customer reference number to be included in billing information and invoices. Also known as the service level reference (SLR) number. Specify a unique identifying number for the product to be used for billing purposes, such as a cost center number or a unique customer ID. The service level reference number appears for each service under the Product section of the invoice. You can also edit this field for an existing service.",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"csp_connections": schema.ListNestedAttribute{
				Description: "The Cloud Service Provider (CSP) connections associated with the VXC.",
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"connect_type": schema.StringAttribute{
							Description: "The connection type of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"resource_name": schema.StringAttribute{
							Description: "The resource name of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"resource_type": schema.StringAttribute{
							Description: "The resource type of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"vlan": schema.Int64Attribute{
							Description: "The VLAN of the CSP connection.",
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"name": schema.StringAttribute{
							Description: "The name of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"owner_account": schema.StringAttribute{
							Description: "The owner's AWS account of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"bandwidth": schema.Int64Attribute{
							Description: "The bandwidth of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"bandwidths": schema.ListAttribute{
							Description: "The bandwidths of the CSP connection.",
							Optional:    true,
							Computed:    true,
							ElementType: types.Int64Type,
							PlanModifiers: []planmodifier.List{
								listplanmodifier.UseStateForUnknown(),
							},
						},
						"customer_ip_address": schema.StringAttribute{
							Description: "The customer IP address of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"customer_ip4_address": schema.StringAttribute{
							Description: "The customer IPv4 address of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"account": schema.StringAttribute{
							Description: "The account of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"amazon_address": schema.StringAttribute{
							Description: "The Amazon address of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"asn": schema.Int64Attribute{
							Description: "The ASN of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"auth_key": schema.StringAttribute{
							Description: "The authentication key of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"customer_address": schema.StringAttribute{
							Description: "The customer address of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"id": schema.Int64Attribute{
							Description: "The ID of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"peer_asn": schema.Int64Attribute{
							Description: "The peer ASN of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"type": schema.StringAttribute{
							Description: "The type of the AWS Virtual Interface.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"vif_id": schema.StringAttribute{
							Description: "The ID of the AWS Virtual Interface.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"connection_id": schema.StringAttribute{
							Description: "The hosted connection ID of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"managed": schema.BoolAttribute{
							Description: "Whether the CSP connection is managed.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"service_key": schema.StringAttribute{
							Description: "The Azure service key of the CSP connection.",
							Optional:    true,
							Computed:    true,
							Sensitive:   true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"csp_name": schema.StringAttribute{
							Description: "The name of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"pairing_key": schema.StringAttribute{
							Description: "The pairing key of the Google Cloud connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"ip_addresses": schema.ListAttribute{
							Description: "The IP addresses of the Virtual Router.",
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.List{
								listplanmodifier.UseStateForUnknown(),
							},
						},
						"virtual_router_name": schema.StringAttribute{
							Description: "The name of the Virtual Router.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"customer_ip6_network": schema.StringAttribute{
							Description: "The customer IPv6 network of the Transit VXC connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"ipv4_gateway_address": schema.StringAttribute{
							Description: "The IPv4 gateway address of the Transit VXC connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"ipv6_gateway_address": schema.StringAttribute{
							Description: "The IPv6 gateway address of the Transit VXC connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the product is locked.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"admin_locked": schema.BoolAttribute{
				Description: "Whether the product is admin locked.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"attribute_tags": schema.MapAttribute{
				Description: "The attribute tags associated with the product.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"cancelable": schema.BoolAttribute{
				Description: "Whether the product is cancelable.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"a_end": schema.SingleNestedAttribute{
				Description: "The current A-End configuration of the VXC.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"owner_uid": schema.StringAttribute{
						Description: "The owner UID of the A-End configuration.",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"requested_product_uid": schema.StringAttribute{
						Description: "The Product UID requested by the user for the A-End configuration.",
						Required:    true,
						// PlanModifiers: []planmodifier.String{
						// 	stringplanmodifier.RequiresReplaceIf(
						// 		func(ctx context.Context, sr planmodifier.StringRequest, rrifr *stringplanmodifier.RequiresReplaceIfFuncResponse) {
						// 			if sr.PlanValue.IsUnknown() {
						// 				rrifr.RequiresReplace = true
						// 			}
						// 		},
						// 		"This modifier will replace the VXC if the new `requested_product_uid` is unknown. This allows the provider to better handle situations when the connected product (Port, MVE, MCR) is being replaced. To avoid replacement, make sure the new `requested_product_uid` is a known value (i.e. an existing product in the state).",
						// 		"This modifier will replace the VXC if the new `requested_product_uid` is unknown. This allows the provider to better handle situations when the connected product (Port, MVE, MCR) is being replaced. To avoid replacement, make sure the new `requested_product_uid` is a known value (i.e. an existing product in the state).",
						// 	),
						// 	stringplanmodifier.UseStateForUnknown(),
						// },
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
					"ordered_vlan": schema.Int64Attribute{
						Description: "The customer-ordered unique VLAN ID of the A-End configuration. Values can range from 2 to 4093. If this value is set to 0, or not included, the Megaport system allocates a valid VLAN ID to the A-End configuration.  To set this VLAN to untagged, set the VLAN value to -1. Please note that if the A-End ordered_vlan is set to -1, the Megaport API will not allow for the A-End inner_vlan field to be set as the VLAN for this end configuration will be untagged.",
						Optional:    true,
						Validators:  []validator.Int64{int64validator.Between(-1, 4093), int64validator.NoneOf(1)},
					},
					"vlan": schema.Int64Attribute{
						Description: "The current VLAN of the A-End configuration. May be different from the A-End ordered VLAN if the system allocated a different VLAN. Values can range from 2 to 4093. If the A-End ordered_vlan was set to 0, the Megaport system allocated a valid VLAN. If the A-End ordered_vlan was set to -1, the Megaport system will automatically set this value to null.",
						Computed:    true,
					},
					"inner_vlan": schema.Int64Attribute{
						Description: "The inner VLAN of the A-End configuration. If the A-End ordered_vlan is untagged and set as -1, this field cannot be set by the API, as the VLAN of the A-End is designated as untagged.",
						Optional:    true,
					},
					"vnic_index": schema.Int64Attribute{
						Description: "The network interface index of the A-End configuration.",
						Computed:    true,
						Optional:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"secondary_name": schema.StringAttribute{
						Description: "The secondary name of the A-End configuration.",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
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
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"requested_product_uid": schema.StringAttribute{
						Description: "The Product UID requested by the user for the B-End configuration.",
						Optional:    true,
						Computed:    true,
						// PlanModifiers: []planmodifier.String{
						// 	stringplanmodifier.RequiresReplaceIf(
						// 		func(ctx context.Context, sr planmodifier.StringRequest, rrifr *stringplanmodifier.RequiresReplaceIfFuncResponse) {
						// 			if sr.PlanValue.IsUnknown() {
						// 				rrifr.RequiresReplace = true
						// 			}
						// 		},
						// 		"This modifier will replace the VXC if the new `requested_product_uid` is unknown. This allows the provider to better handle situations when the connected product (Port, MVE, MCR) is being replaced. To avoid replacement, make sure the new `requested_product_uid` is a known value (i.e. an existing product in the state).",
						// 		"This modifier will replace the VXC if the new `requested_product_uid` is unknown. This allows the provider to better handle situations when the connected product (Port, MVE, MCR) is being replaced. To avoid replacement, make sure the new `requested_product_uid` is a known value (i.e. an existing product in the state).",
						// 	),
						// 	stringplanmodifier.UseStateForUnknown(),
						// },
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
					"ordered_vlan": schema.Int64Attribute{
						Description: "The customer-ordered unique VLAN ID of the B-End configuration. Values can range from 2 to 4093. If this value is set to 0, or not included, the Megaport system allocates a valid VLAN ID to the B-End configuration.  To set this VLAN to untagged, set the VLAN value to -1. Please note that if the B-End ordered_vlan is set to -1, the Megaport API will not allow for the B-End inner_vlan field to be set as the VLAN for this end configuration will be untagged.",
						Optional:    true,
						Validators:  []validator.Int64{int64validator.Between(-1, 4093), int64validator.NoneOf(1)},
					},
					"vlan": schema.Int64Attribute{
						Description: "The current VLAN of the B-End configuration. May be different from the B-End ordered VLAN if the system allocated a different VLAN. Values can range from 2 to 4093. If the B-End ordered_vlan was set to 0, the Megaport system allocated a valid VLAN. If the B-End ordered_vlan was set to -1, the Megaport system will automatically set this value to null.",
						Computed:    true,
					},
					"inner_vlan": schema.Int64Attribute{
						Description: "The inner VLAN of the B-End configuration. If the B-End ordered_vlan is untagged and set as -1, this field cannot be set by the API, as the VLAN of the B-End is designated as untagged.",
						Optional:    true,
					},
					"vnic_index": schema.Int64Attribute{
						Description: "The network interface index of the B-End configuration.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"secondary_name": schema.StringAttribute{
						Description: "The secondary name of the B-End configuration.",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"a_end_partner_config": schema.SingleNestedAttribute{
				Description: `The partner configuration of the A-End order configuration. Contains CSP and/or BGP Configuration settings. For any partner configuration besides "vrouter", this configuration cannot be changed after the VXC is created and if it is modified, the VXC will be deleted and re-created. Imported VXCs do not have this field populated by the API, so the initially provided configuration will be ignored as it can't be verified to be correct. If the user wants to change the configuration after importing the resource, they can then do so by changing the field after importing the resource and running terraform apply.`,
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"partner": schema.StringAttribute{
						Description: "The partner of the partner configuration.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("aws", "azure", "google", "oracle", "vrouter", "transit", "a-end"),
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
								Sensitive:   true,
							},
							"port_choice": schema.StringAttribute{
								Description: "Which port to choose when building the VXC. Can either be 'primary' or 'secondary'.",
								Required:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("primary", "secondary"),
								},
							},
							"peers": schema.ListNestedAttribute{
								Description: "The peers of the partner configuration. If this is set, the user must delete any Azure resources associated with the VXC on Azure before deleting the VXC.",
								Optional:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											Description: "The type of the peer.",
											Required:    true,
										},
										"peer_asn": schema.StringAttribute{
											Description: "The peer ASN of the peer.",
											Optional:    true,
										},
										"primary_subnet": schema.StringAttribute{
											Description: "The primary subnet of the peer.",
											Optional:    true,
										},
										"secondary_subnet": schema.StringAttribute{
											Description: "The secondary subnet of the peer.",
											Optional:    true,
										},
										"prefixes": schema.StringAttribute{
											Description: "The prefixes of the peer.",
											Optional:    true,
										},
										"shared_key": schema.StringAttribute{
											Description: "The shared key of the peer.",
											Optional:    true,
										},
										"vlan": schema.Int64Attribute{
											Description: "The VLAN of the peer.",
											Optional:    true,
										},
									},
								},
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
					"vrouter_config": schema.SingleNestedAttribute{
						Description: "The partner configuration of the virtual router configuration.",
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
										"vlan": schema.Int64Attribute{
											Description: "Inner-VLAN for implicit Q-inQ VXCs. Typically used only for Azure VXCs. The default is no inner-vlan.",
											Optional:    true,
										},
										"bgp_connections": schema.ListNestedAttribute{
											Description: "The BGP connections of the partner configuration interface.",
											Optional:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"peer_type": schema.StringAttribute{
														Description: "Defines the default BGP routing policy for this BGP connection. The default depends on the CSP type of the far end of this VXC.",
														Optional:    true,
														Validators: []validator.String{
															stringvalidator.OneOf("NON_CLOUD", "PRIV_CLOUD", "PUB_CLOUD"),
														},
													},
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
													"as_path_prepend_count": schema.Int64Attribute{
														Description: "The AS path prepend count of the BGP connection. Minimum value of 0 and maximum value of 10.",
														Optional:    true,
														Validators:  []validator.Int64{int64validator.Between(0, 10)},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					"partner_a_end_config": schema.SingleNestedAttribute{
						Description:        "The partner configuration of the A-End order configuration. Only exists for A-End Configurations. DEPRECATED: Use vrouter_config instead.",
						Optional:           true,
						DeprecationMessage: "Deprecated: Use `vrouter_config` instead.",
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
													"as_path_prepend_count": schema.Int64Attribute{
														Description: "The AS path prepend count of the BGP connection. Minimum value of 0 and maximum value of 10.",
														Optional:    true,
														Validators:  []validator.Int64{int64validator.Between(0, 10)},
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
				Description: `The partner configuration of the B-End order configuration. Contains CSP and/or BGP Configuration settings. For any partner configuration besides "vrouter", this configuration cannot be changed after the VXC is created and if it is modified, the VXC will be deleted and re-created. Imported VXCs do not have this field populated by the API, so the initially provided configuration will be ignored as it can't be verified to be correct. If the user wants to change the configuration after importing the resource, they can then do so by changing the field after importing the resource and running terraform apply.`,
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"partner": schema.StringAttribute{
						Description: "The partner of the partner configuration.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("aws", "azure", "google", "oracle", "transit", "vrouter"),
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
								Sensitive:   true,
							},
							"port_choice": schema.StringAttribute{
								Description: "Which port to choose when building the VXC. Can either be 'primary' or 'secondary'.",
								Required:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("primary", "secondary"),
								},
							},
							"peers": schema.ListNestedAttribute{
								Description: "The peers of the partner configuration. If this is set, the user must delete any Azure resources associated with the VXC on Azure before deleting the VXC.",
								Optional:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											Description: "The type of the peer.",
											Required:    true,
										},
										"peer_asn": schema.StringAttribute{
											Description: "The peer ASN of the peer.",
											Optional:    true,
										},
										"primary_subnet": schema.StringAttribute{
											Description: "The primary subnet of the peer.",
											Optional:    true,
										},
										"secondary_subnet": schema.StringAttribute{
											Description: "The secondary subnet of the peer.",
											Optional:    true,
										},
										"prefixes": schema.StringAttribute{
											Description: "The prefixes of the peer.",
											Optional:    true,
										},
										"shared_key": schema.StringAttribute{
											Description: "The shared key of the peer.",
											Optional:    true,
										},
										"vlan": schema.Int64Attribute{
											Description: "The VLAN of the peer.",
											Optional:    true,
										},
									},
								},
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
					"vrouter_config": schema.SingleNestedAttribute{
						Description: "The partner configuration of the virtual router configuration.",
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
										"vlan": schema.Int64Attribute{
											Description: "Inner-VLAN for implicit Q-inQ VXCs. Typically used only for Azure VXCs. The default is no inner-vlan.",
											Optional:    true,
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
													"peer_type": schema.StringAttribute{
														Description: "Defines the default BGP routing policy for this BGP connection. The default depends on the CSP type of the far end of this VXC.",
														Optional:    true,
														Validators: []validator.String{
															stringvalidator.OneOf("NON_CLOUD", "PRIV_CLOUD", "PUB_CLOUD"),
														},
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
													"as_path_prepend_count": schema.Int64Attribute{
														Description: "The AS path prepend count of the BGP connection. Minimum value of 0 and maximum value of 10.",
														Optional:    true,
														Validators:  []validator.Int64{int64validator.Between(0, 10)},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					"partner_a_end_config": schema.SingleNestedAttribute{
						Description:        "The partner configuration of the A-End order configuration. Only exists for A-End Configurations, invalid on B-End Partner Config. DEPRECATED: Use vrouter_config instead.",
						Optional:           true,
						DeprecationMessage: "Deprecated: Use `vrouter_config` instead.",
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
													"as_path_prepend_count": schema.Int64Attribute{
														Description: "The AS path prepend count of the BGP connection. Minimum value of 0 and maximum value of 10.",
														Optional:    true,
														Validators:  []validator.Int64{int64validator.Between(0, 10)},
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
		ServiceKey: plan.ServiceKey.ValueString(),

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

	if !a.InnerVLAN.IsNull() || !a.NetworkInterfaceIndex.IsNull() {
		vxcOrderMVEConfig := &megaport.VXCOrderMVEConfig{}
		if !a.InnerVLAN.IsNull() {
			vxcOrderMVEConfig.InnerVLAN = int(a.InnerVLAN.ValueInt64())
		}
		if !a.NetworkInterfaceIndex.IsNull() {
			vxcOrderMVEConfig.NetworkInterfaceIndex = int(a.NetworkInterfaceIndex.ValueInt64())
		}
		aEndConfig.VXCOrderMVEConfig = vxcOrderMVEConfig
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
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     awsConfigObj,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
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
			partnerPortReq := &megaport.ListPartnerPortsRequest{
				Key:     azureConfig.ServiceKey.ValueString(),
				Partner: "AZURE",
			}
			partnerPortRes, err := r.client.VXCService.ListPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", plan.Name.ValueString(), err.Error()),
				)
				return
			}
			// find primary or secondary port
			for _, port := range partnerPortRes.Data.Megaports {
				p := &port
				if p.Type == azureConfig.PortChoice.ValueString() {
					aEndConfig.ProductUID = p.ProductUID
				}
			}
			if aEndConfig.ProductUID == "" {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not find azure port with type: %s", azureConfig.PortChoice.ValueString()),
				)
				return
			}

			aEndPartnerConfig := megaport.VXCPartnerConfigAzure{
				ConnectType: "AZURE",
				ServiceKey:  azureConfig.ServiceKey.ValueString(),
			}

			azurePeerModels := []partnerOrderAzurePeeringConfigModel{}
			azurePeerDiags := azureConfig.Peers.ElementsAs(ctx, &azurePeerModels, false)
			resp.Diagnostics.Append(azurePeerDiags...)
			if len(azurePeerModels) > 0 {
				aEndPartnerConfig.Peers = []megaport.PartnerOrderAzurePeeringConfig{}
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
					aEndPartnerConfig.Peers = append(aEndPartnerConfig.Peers, peeringConfig)
				}
			}

			azureConfigObj, azureDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAzureAttrs, azureConfig)
			resp.Diagnostics.Append(azureDiags...)

			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azureConfigObj,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
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
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  googleConfigObj,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
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
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracleConfigObj,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = aEndPartnerConfig
		case "vrouter":
			if aPartnerConfig.VrouterPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Virtual router configuration is required",
				)
				return
			}
			var partnerConfigAEnd vxcPartnerConfigVrouterModel
			aEndDiags := aPartnerConfig.VrouterPartnerConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
			if aEndDiags.HasError() {
				resp.Diagnostics.Append(aEndDiags...)
				return
			}
			prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, a.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			aEndMegaportConfig := megaport.VXCOrderVrouterPartnerConfig{}
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
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
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
				if !iface.VLAN.IsNull() {
					toAppend.VLAN = int(iface.VLAN.ValueInt64())
				}
				if !iface.BgpConnections.IsNull() {
					bgpConnections := []*bgpConnectionConfigModel{}
					bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConnections, false)
					resp.Diagnostics = append(resp.Diagnostics, bgpDiags...)
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
			vRouterConfigObj, aEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, partnerConfigAEnd)
			resp.Diagnostics.Append(aEndDiags...)
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vRouterConfigObj,
				PartnerAEndConfig:    aEndPartner,
			}
			aEndPartnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = aEndPartnerConfigObj
			aEndConfig.PartnerConfig = aEndMegaportConfig
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
			prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, a.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			aEndMegaportConfig := megaport.VXCOrderVrouterPartnerConfig{}
			ifaceModels := []*vxcPartnerConfigInterfaceModel{}
			ifaceDiags := partnerConfigAEnd.Interfaces.ElementsAs(ctx, &ifaceModels, true)
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
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
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
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				PartnerAEndConfig:    aEndConfigObj,
				VrouterPartnerConfig: vrouter,
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
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
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
	if !b.InnerVLAN.IsNull() || !b.NetworkInterfaceIndex.IsNull() {
		vxcOrderMVEConfig := &megaport.VXCOrderMVEConfig{}
		if !b.InnerVLAN.IsNull() {
			vxcOrderMVEConfig.InnerVLAN = int(b.InnerVLAN.ValueInt64())
		}
		if !b.NetworkInterfaceIndex.IsNull() {
			vxcOrderMVEConfig.NetworkInterfaceIndex = int(b.NetworkInterfaceIndex.ValueInt64())
		}
		bEndConfig.VXCOrderMVEConfig = vxcOrderMVEConfig
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
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     awsConfigObj,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
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

			azurePeerModels := []partnerOrderAzurePeeringConfigModel{}
			azurePeerDiags := azureConfig.Peers.ElementsAs(ctx, &azurePeerModels, false)
			resp.Diagnostics.Append(azurePeerDiags...)
			if len(azurePeerModels) > 0 {
				bEndPartnerConfig.Peers = []megaport.PartnerOrderAzurePeeringConfig{}
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
					bEndPartnerConfig.Peers = append(bEndPartnerConfig.Peers, peeringConfig)
				}
			}

			azureConfigObj, azureDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAzureAttrs, azureConfig)
			resp.Diagnostics.Append(azureDiags...)

			partnerPortReq := &megaport.ListPartnerPortsRequest{
				Key:     azureConfig.ServiceKey.ValueString(),
				Partner: "AZURE",
			}
			partnerPortRes, err := r.client.VXCService.ListPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", plan.Name.ValueString(), err.Error()),
				)
				return
			}
			// find primary or secondary port
			for _, port := range partnerPortRes.Data.Megaports {
				p := &port
				if p.Type == azureConfig.PortChoice.ValueString() {
					bEndConfig.ProductUID = p.ProductUID
				}
			}
			if bEndConfig.ProductUID == "" {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not find azure port with type: %s", azureConfig.PortChoice.ValueString()),
				)
				return
			}

			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azureConfigObj,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
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
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  googleConfigObj,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
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
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracleConfigObj,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
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
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = bEndPartnerConfig
		case "vrouter":
			if bPartnerConfig.VrouterPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Virtual router configuration is required",
				)
				return
			}
			var partnerConfigBEnd vxcPartnerConfigVrouterModel
			bEndDiags := bPartnerConfig.VrouterPartnerConfig.As(ctx, &partnerConfigBEnd, basetypes.ObjectAsOptions{})
			if aEndDiags.HasError() {
				resp.Diagnostics.Append(bEndDiags...)
				return
			}
			prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, b.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			bEndMegaportConfig := megaport.VXCOrderVrouterPartnerConfig{}
			ifaceModels := []*vxcPartnerConfigInterfaceModel{}
			ifaceDiags := partnerConfigBEnd.Interfaces.ElementsAs(ctx, &ifaceModels, false)
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
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
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
				if !iface.VLAN.IsNull() {
					toAppend.VLAN = int(iface.VLAN.ValueInt64())
				}
				if !iface.BgpConnections.IsNull() {
					bgpConnections := []*bgpConnectionConfigModel{}
					bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConnections, false)
					resp.Diagnostics = append(resp.Diagnostics, bgpDiags...)
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
				bEndMegaportConfig.Interfaces = append(bEndMegaportConfig.Interfaces, toAppend)
			}
			vrouterConfigObj, bEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, partnerConfigBEnd)
			resp.Diagnostics.Append(bEndDiags...)
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouterConfigObj,
				PartnerAEndConfig:    aEndPartner,
			}
			bEndPartnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.BEndPartnerConfig = bEndPartnerConfigObj
			bEndConfig.PartnerConfig = bEndMegaportConfig
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

	err := r.client.VXCService.ValidateVXCOrder(ctx, buyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Validation error while attempting to create VXC",
			"Validation error while attempting to create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

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
		// VXC has been deleted or is not found
		if mpErr, ok := err.(*megaport.ErrorResponse); ok {
			if mpErr.Response.StatusCode == http.StatusNotFound ||
				(mpErr.Response.StatusCode == http.StatusBadRequest && strings.Contains(mpErr.Message, "Could not find a service with UID")) {
				resp.State.RemoveResource(ctx)
				return
			}
		}

		resp.Diagnostics.AddError(
			"Error Reading VXC",
			"Could not read VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	// If the vxc has been deleted
	if vxc.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED {
		resp.State.RemoveResource(ctx)
		return
	}

	apiDiags := state.fromAPIVXC(ctx, vxc)
	resp.Diagnostics.Append(apiDiags...)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	aEndConfig := &vxcEndConfigurationModel{}
	bEndConfig := &vxcEndConfigurationModel{}
	aEndConfigDiags := state.AEndConfiguration.As(ctx, aEndConfig, basetypes.ObjectAsOptions{})
	bEndConfigDiags := state.BEndConfiguration.As(ctx, bEndConfig, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(aEndConfigDiags...)
	resp.Diagnostics.Append(bEndConfigDiags...)
}

func (r *vxcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vxcResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If Imported, AEndPartnerConfig will be null. Set the partner config to the existing one in the plan.
	if state.AEndPartnerConfig.IsNull() {
		state.AEndPartnerConfig = plan.AEndPartnerConfig
	}
	if state.BEndPartnerConfig.IsNull() {
		state.BEndPartnerConfig = plan.BEndPartnerConfig
	}

	var aEndPlan, bEndPlan, aEndState, bEndState *vxcEndConfigurationModel
	var aEndPartnerPlan, bEndPartnerPlan, aEndPartnerState, bEndPartnerState *vxcPartnerConfigurationModel

	// Check if AEnd or BEnd is a CSP Partner Configuration
	var aEndCSP, bEndCSP bool

	aEndPlanDiags := plan.AEndConfiguration.As(ctx, &aEndPlan, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(aEndPlanDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	bEndPlanDiags := plan.BEndConfiguration.As(ctx, &bEndPlan, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(bEndPlanDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	aEndStateDiags := state.AEndConfiguration.As(ctx, &aEndState, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(aEndStateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	bEndStateDiags := state.BEndConfiguration.As(ctx, &bEndState, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(bEndStateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	aEndPartnerPlanDiags := plan.AEndPartnerConfig.As(ctx, &aEndPartnerPlan, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(aEndPartnerPlanDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.AEndPartnerConfig.IsNull() {
		if !aEndPartnerPlan.Partner.IsNull() {
			// Check if the partner is a CSP Partner
			if aEndPartnerPlan.Partner.ValueString() != "a-end" && aEndPartnerPlan.Partner.ValueString() != "vrouter" && aEndPartnerPlan.Partner.ValueString() != "transit" {
				aEndCSP = true
			}
		}
	}
	bEndPartnerPlanDiags := plan.BEndPartnerConfig.As(ctx, &bEndPartnerPlan, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(bEndPartnerPlanDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.BEndPartnerConfig.IsNull() {
		if !bEndPartnerPlan.Partner.IsNull() {
			// Check if the partner is a CSP Partner
			if bEndPartnerPlan.Partner.ValueString() != "a-end" && bEndPartnerPlan.Partner.ValueString() != "vrouter" && bEndPartnerPlan.Partner.ValueString() != "transit" {
				bEndCSP = true
			}
		}
	}

	aEndPartnerStateDiags := state.AEndPartnerConfig.As(ctx, &aEndPartnerState, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(aEndPartnerStateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	bEndPartnerStateDiags := state.BEndPartnerConfig.As(ctx, &bEndPartnerState, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(bEndPartnerStateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &megaport.UpdateVXCRequest{
		WaitForUpdate: true,
		WaitForTime:   waitForTime,
	}

	if !plan.Name.Equal(state.Name) {
		updateReq.Name = megaport.PtrTo(plan.Name.ValueString())
	}

	// If Ordered VLAN is different from actual VLAN, attempt to change it to the ordered VLAN value.
	if !aEndPlan.OrderedVLAN.IsUnknown() && !aEndPlan.OrderedVLAN.IsNull() && !aEndPlan.OrderedVLAN.Equal(aEndState.VLAN) {
		updateReq.AEndVLAN = megaport.PtrTo(int(aEndPlan.OrderedVLAN.ValueInt64()))
	}
	aEndState.OrderedVLAN = aEndPlan.OrderedVLAN

	if !aEndPlan.InnerVLAN.IsUnknown() && !aEndPlan.InnerVLAN.IsNull() && !aEndPlan.InnerVLAN.Equal(aEndState.InnerVLAN) {
		updateReq.AEndInnerVLAN = megaport.PtrTo(int(aEndPlan.InnerVLAN.ValueInt64()))
	}
	aEndState.InnerVLAN = aEndPlan.InnerVLAN

	// If Ordered VLAN is different from actual VLAN, attempt to change it to the ordered VLAN value.
	if !bEndPlan.OrderedVLAN.IsUnknown() && !bEndPlan.OrderedVLAN.IsNull() && !bEndPlan.OrderedVLAN.Equal(bEndState.VLAN) {
		updateReq.BEndVLAN = megaport.PtrTo(int(bEndPlan.OrderedVLAN.ValueInt64()))
	}
	bEndState.OrderedVLAN = bEndPlan.OrderedVLAN

	if !bEndPlan.InnerVLAN.IsUnknown() && !bEndPlan.InnerVLAN.IsNull() && !bEndPlan.InnerVLAN.Equal(bEndState.InnerVLAN) {
		updateReq.BEndInnerVLAN = megaport.PtrTo(int(bEndPlan.InnerVLAN.ValueInt64()))
	}
	bEndState.InnerVLAN = bEndPlan.InnerVLAN

	if !plan.RateLimit.IsNull() && !plan.RateLimit.Equal(state.RateLimit) {
		updateReq.RateLimit = megaport.PtrTo(int(plan.RateLimit.ValueInt64()))
	}

	if !plan.CostCentre.IsNull() && !plan.CostCentre.Equal(state.CostCentre) {
		updateReq.CostCentre = megaport.PtrTo(plan.CostCentre.ValueString())
	}

	if !plan.Shutdown.IsNull() && !plan.Shutdown.Equal(state.Shutdown) {
		updateReq.Shutdown = megaport.PtrTo(plan.Shutdown.ValueBool())
	}

	if !plan.ContractTermMonths.IsNull() && !plan.ContractTermMonths.Equal(state.ContractTermMonths) {
		updateReq.Term = megaport.PtrTo(int(plan.ContractTermMonths.ValueInt64()))
	}

	if !aEndPlan.RequestedProductUID.IsNull() && !aEndPlan.RequestedProductUID.Equal(aEndState.RequestedProductUID) {
		// Do not update the product UID if the partner is a CSP
		if !aEndCSP {
			updateReq.AEndProductUID = megaport.PtrTo(aEndPlan.RequestedProductUID.ValueString())
			aEndState.RequestedProductUID = aEndPlan.RequestedProductUID
		}
	}
	if !bEndPlan.RequestedProductUID.IsNull() && !bEndPlan.RequestedProductUID.Equal(bEndState.RequestedProductUID) {
		// Do not update the product UID if the partner is a CSP
		if !bEndCSP {
			updateReq.BEndProductUID = megaport.PtrTo(bEndPlan.RequestedProductUID.ValueString())
			bEndState.RequestedProductUID = bEndPlan.RequestedProductUID
		}
	}

	if !plan.AEndPartnerConfig.IsNull() && !plan.AEndPartnerConfig.Equal(state.AEndPartnerConfig) {
		aPartnerConfig := aEndPartnerPlan
		switch aEndPartnerPlan.Partner.ValueString() {
		case "transit":
			aEndPartnerConfig := &megaport.VXCPartnerConfigTransit{
				ConnectType: "TRANSIT",
			}
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			state.AEndPartnerConfig = partnerConfigObj
			updateReq.AEndPartnerConfig = aEndPartnerConfig
		case "a-end":
			if aPartnerConfig.PartnerAEndConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": A-End Partner configuration is required",
				)
				return
			}
			var partnerConfigAEnd vxcPartnerConfigAEndModel
			aEndDiags := aPartnerConfig.PartnerAEndConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(aEndDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, aEndPlan.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			aEndPartnerConfig := megaport.VXCOrderVrouterPartnerConfig{}
			ifaceModels := []*vxcPartnerConfigInterfaceModel{}
			ifaceDiags := partnerConfigAEnd.Interfaces.ElementsAs(ctx, &ifaceModels, true)
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
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
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
				aEndPartnerConfig.Interfaces = append(aEndPartnerConfig.Interfaces, toAppend)
			}
			aEndConfigObj, aEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAEndAttrs, partnerConfigAEnd)
			resp.Diagnostics.Append(aEndDiags...)
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				PartnerAEndConfig:    aEndConfigObj,
				VrouterPartnerConfig: vrouter,
			}
			aEndPartnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			state.AEndPartnerConfig = aEndPartnerConfigObj
			updateReq.AEndPartnerConfig = aEndPartnerConfig
		case "vrouter":
			if aEndPartnerPlan.VrouterPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": Virtual router configuration is required",
				)
				return
			}
			var partnerConfigAEnd vxcPartnerConfigVrouterModel
			aEndDiags := aEndPartnerPlan.VrouterPartnerConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(aEndDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, aEndState.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			aEndMegaportConfig := &megaport.VXCOrderVrouterPartnerConfig{}
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
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
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
				if !iface.VLAN.IsNull() {
					toAppend.VLAN = int(iface.VLAN.ValueInt64())
				}
				if !iface.BgpConnections.IsNull() {
					bgpConnections := []*bgpConnectionConfigModel{}
					bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConnections, false)
					resp.Diagnostics = append(resp.Diagnostics, bgpDiags...)
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
			vRouterConfigObj, aEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, partnerConfigAEnd)
			resp.Diagnostics.Append(aEndDiags...)
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aEndPartnerPlan.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vRouterConfigObj,
				PartnerAEndConfig:    aEndPartner,
			}
			aEndPartnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			state.AEndPartnerConfig = aEndPartnerConfigObj
			updateReq.AEndPartnerConfig = aEndMegaportConfig
		default:
			resp.Diagnostics.AddError(
				"Error Updating VXC",
				"Could not update VXC with ID "+state.UID.ValueString()+": Partner configuration not supported",
			)
			return
		}
	}

	if !plan.BEndPartnerConfig.IsNull() && !plan.BEndPartnerConfig.Equal(state.BEndPartnerConfig) {
		bPartnerConfig := bEndPartnerPlan
		switch bEndPartnerPlan.Partner.ValueString() {
		case "transit":
			bEndPartnerConfig := &megaport.VXCPartnerConfigTransit{
				ConnectType: "TRANSIT",
			}
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			state.AEndPartnerConfig = partnerConfigObj
			updateReq.AEndPartnerConfig = bEndPartnerConfig
		case "vrouter":
			if bEndPartnerPlan.VrouterPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Virtual router configuration is required",
				)
				return
			}
			var partnerConfigBEnd vxcPartnerConfigVrouterModel
			bEndDiags := bEndPartnerPlan.VrouterPartnerConfig.As(ctx, &partnerConfigBEnd, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(bEndDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, bEndState.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			bEndMegaportConfig := &megaport.VXCOrderVrouterPartnerConfig{}
			ifaceModels := []*vxcPartnerConfigInterfaceModel{}
			ifaceDiags := partnerConfigBEnd.Interfaces.ElementsAs(ctx, &ifaceModels, false)
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
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
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
				if !iface.VLAN.IsNull() {
					toAppend.VLAN = int(iface.VLAN.ValueInt64())
				}
				if !iface.BgpConnections.IsNull() {
					bgpConnections := []*bgpConnectionConfigModel{}
					bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConnections, false)
					resp.Diagnostics = append(resp.Diagnostics, bgpDiags...)
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
				bEndMegaportConfig.Interfaces = append(bEndMegaportConfig.Interfaces, toAppend)
			}
			vrouterConfigObj, bEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, partnerConfigBEnd)
			resp.Diagnostics.Append(bEndDiags...)
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bEndPartnerPlan.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouterConfigObj,
				PartnerAEndConfig:    aEndPartner,
			}
			bEndPartnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			state.BEndPartnerConfig = bEndPartnerConfigObj
			updateReq.BEndPartnerConfig = bEndMegaportConfig
		default:
			resp.Diagnostics.AddError(
				"Error Updating VXC",
				"Could not update VXC with ID "+state.UID.ValueString()+": Partner configuration not supported",
			)
			return
		}
	}

	if !plan.AEndPartnerConfig.IsNull() && !plan.AEndPartnerConfig.Equal(state.AEndPartnerConfig) {
		aPartnerConfig := aEndPartnerPlan
		switch aEndPartnerPlan.Partner.ValueString() {
		case "transit":
			aEndPartnerConfig := &megaport.VXCPartnerConfigTransit{
				ConnectType: "TRANSIT",
			}
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			state.AEndPartnerConfig = partnerConfigObj
			updateReq.AEndPartnerConfig = aEndPartnerConfig
		case "a-end":
			if aPartnerConfig.PartnerAEndConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": A-End Partner configuration is required",
				)
				return
			}
			var partnerConfigAEnd vxcPartnerConfigAEndModel
			aEndDiags := aPartnerConfig.PartnerAEndConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(aEndDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, aEndPlan.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			aEndPartnerConfig := megaport.VXCOrderVrouterPartnerConfig{}
			ifaceModels := []*vxcPartnerConfigInterfaceModel{}
			ifaceDiags := partnerConfigAEnd.Interfaces.ElementsAs(ctx, &ifaceModels, true)
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
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
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
				aEndPartnerConfig.Interfaces = append(aEndPartnerConfig.Interfaces, toAppend)
			}
			aEndConfigObj, aEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAEndAttrs, partnerConfigAEnd)
			resp.Diagnostics.Append(aEndDiags...)
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				PartnerAEndConfig:    aEndConfigObj,
				VrouterPartnerConfig: vrouter,
			}
			aEndPartnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			state.AEndPartnerConfig = aEndPartnerConfigObj
			updateReq.AEndPartnerConfig = aEndPartnerConfig
		case "vrouter":
			if aEndPartnerPlan.VrouterPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": Virtual router configuration is required",
				)
				return
			}
			var partnerConfigAEnd vxcPartnerConfigVrouterModel
			aEndDiags := aEndPartnerPlan.VrouterPartnerConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(aEndDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, aEndState.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			aEndMegaportConfig := &megaport.VXCOrderVrouterPartnerConfig{}
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
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
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
				if !iface.VLAN.IsNull() {
					toAppend.VLAN = int(iface.VLAN.ValueInt64())
				}
				if !iface.BgpConnections.IsNull() {
					bgpConnections := []*bgpConnectionConfigModel{}
					bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConnections, false)
					resp.Diagnostics = append(resp.Diagnostics, bgpDiags...)
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
			vRouterConfigObj, aEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, partnerConfigAEnd)
			resp.Diagnostics.Append(aEndDiags...)
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aEndPartnerPlan.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vRouterConfigObj,
				PartnerAEndConfig:    aEndPartner,
			}
			aEndPartnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			state.AEndPartnerConfig = aEndPartnerConfigObj
			updateReq.AEndPartnerConfig = aEndMegaportConfig
		default:
			resp.Diagnostics.AddError(
				"Error Updating VXC",
				"Could not update VXC with ID "+state.UID.ValueString()+": Partner configuration not supported",
			)
			return
		}
	}

	if !plan.BEndPartnerConfig.IsNull() && !plan.BEndPartnerConfig.Equal(state.BEndPartnerConfig) {
		bPartnerConfig := bEndPartnerPlan
		switch bEndPartnerPlan.Partner.ValueString() {
		case "transit":
			bEndPartnerConfig := &megaport.VXCPartnerConfigTransit{
				ConnectType: "TRANSIT",
			}
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			state.AEndPartnerConfig = partnerConfigObj
			updateReq.AEndPartnerConfig = bEndPartnerConfig
		case "vrouter":
			if bEndPartnerPlan.VrouterPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Virtual router configuration is required",
				)
				return
			}
			var partnerConfigBEnd vxcPartnerConfigVrouterModel
			bEndDiags := bEndPartnerPlan.VrouterPartnerConfig.As(ctx, &partnerConfigBEnd, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(bEndDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, bEndState.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			bEndMegaportConfig := &megaport.VXCOrderVrouterPartnerConfig{}
			ifaceModels := []*vxcPartnerConfigInterfaceModel{}
			ifaceDiags := partnerConfigBEnd.Interfaces.ElementsAs(ctx, &ifaceModels, false)
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
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
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
				if !iface.VLAN.IsNull() {
					toAppend.VLAN = int(iface.VLAN.ValueInt64())
				}
				if !iface.BgpConnections.IsNull() {
					bgpConnections := []*bgpConnectionConfigModel{}
					bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConnections, false)
					resp.Diagnostics = append(resp.Diagnostics, bgpDiags...)
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
				bEndMegaportConfig.Interfaces = append(bEndMegaportConfig.Interfaces, toAppend)
			}
			vrouterConfigObj, bEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, partnerConfigBEnd)
			resp.Diagnostics.Append(bEndDiags...)
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bEndPartnerPlan.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouterConfigObj,
				PartnerAEndConfig:    aEndPartner,
			}
			bEndPartnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			state.BEndPartnerConfig = bEndPartnerConfigObj
			updateReq.BEndPartnerConfig = bEndMegaportConfig
		default:
			resp.Diagnostics.AddError(
				"Error Updating VXC",
				"Could not update VXC with ID "+state.UID.ValueString()+": Partner configuration not supported",
			)
			return
		}
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
	case megaport.CSPConnectionOracle:
		oracleModel := &cspConnectionModel{
			ConnectType:  types.StringValue(provider.ConnectType),
			ResourceName: types.StringValue(provider.ResourceName),
			ResourceType: types.StringValue(provider.ResourceType),
			Bandwidth:    types.Int64Value(int64(provider.Bandwidth)),
			CSPName:      types.StringValue(provider.CSPName),
		}
		oracleModel.Bandwidths = types.ListNull(types.Int64Type)
		oracleModel.IPAddresses = types.ListNull(types.StringType)
		oracleObject, oracleObjDiags := types.ObjectValueFrom(ctx, cspConnectionFullAttrs, oracleModel)
		apiDiags = append(apiDiags, oracleObjDiags...)
		return oracleObject, apiDiags
	}
	apiDiags.AddError("Error creating CSP Connection", "Could not create CSP Connection, unknown type")
	return types.ObjectNull(cspConnectionFullAttrs), apiDiags
}

func (r *vxcResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Get current state
	var plan, state vxcResourceModel
	diags := diag.Diagnostics{}

	if !req.Plan.Raw.IsNull() {
		planDiags := req.Plan.Get(ctx, &plan)
		resp.Diagnostics.Append(planDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if !req.State.Raw.IsNull() {
		stateDiags := req.State.Get(ctx, &state)
		resp.Diagnostics.Append(stateDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// If VXC is not yet created, return
	if !state.UID.IsNull() {
		if !req.Plan.Raw.IsNull() {
			var aEndCSP, bEndCSP bool
			aEndStateObj := state.AEndConfiguration
			bEndStateObj := state.BEndConfiguration
			aEndStateConfig := &vxcEndConfigurationModel{}
			bEndStateConfig := &vxcEndConfigurationModel{}
			aEndDiags := aEndStateObj.As(ctx, aEndStateConfig, basetypes.ObjectAsOptions{})
			bEndDiags := bEndStateObj.As(ctx, bEndStateConfig, basetypes.ObjectAsOptions{})
			diags = append(diags, aEndDiags...)
			diags = append(diags, bEndDiags...)
			aEndPlanObj := plan.AEndConfiguration
			bEndPlanObj := plan.BEndConfiguration
			aEndPlanConfig := &vxcEndConfigurationModel{}
			bEndPlanConfig := &vxcEndConfigurationModel{}
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{}
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{}
			aEndDiags = aEndPlanObj.As(ctx, aEndPlanConfig, basetypes.ObjectAsOptions{})
			bEndDiags = bEndPlanObj.As(ctx, bEndPlanConfig, basetypes.ObjectAsOptions{})
			diags = append(diags, aEndDiags...)
			diags = append(diags, bEndDiags...)
			if aEndStateConfig.OrderedVLAN.IsUnknown() {
				aEndPlanConfig.OrderedVLAN = aEndStateConfig.VLAN
			}
			if bEndStateConfig.OrderedVLAN.IsUnknown() {
				bEndPlanConfig.OrderedVLAN = bEndStateConfig.VLAN
			}
			partnerConfigDiags := plan.AEndPartnerConfig.As(ctx, &aEndPartnerConfigModel, basetypes.ObjectAsOptions{})
			diags = append(diags, partnerConfigDiags...)
			if !plan.AEndPartnerConfig.IsNull() {
				if !aEndPartnerConfigModel.Partner.IsNull() {
					if aEndPartnerConfigModel.Partner.ValueString() != "transit" && aEndPartnerConfigModel.Partner.ValueString() != "vrouter" && aEndPartnerConfigModel.Partner.ValueString() != "a-end" {
						aEndCSP = true
					}
				}
			}
			if state.AEndPartnerConfig.IsNull() {
				if !plan.AEndPartnerConfig.IsNull() {
					state.AEndPartnerConfig = plan.AEndPartnerConfig
				} else {
					state.AEndPartnerConfig = types.ObjectNull(vxcPartnerConfigAttrs)
				}
			} else {
				if !plan.AEndPartnerConfig.Equal(state.AEndPartnerConfig) && aEndCSP {
					resp.RequiresReplace = append(resp.RequiresReplace, path.Root("a_end_partner_config"))
				}
			}

			if aEndStateConfig.RequestedProductUID.IsNull() {
				aEndStateConfig.RequestedProductUID = aEndStateConfig.CurrentProductUID
				aEndPlanConfig.RequestedProductUID = aEndStateConfig.CurrentProductUID
			} else if aEndCSP {
				if !aEndPlanConfig.RequestedProductUID.IsNull() && !aEndPlanConfig.RequestedProductUID.Equal(aEndStateConfig.RequestedProductUID) {
					diags.AddWarning("VXC A-End product UID is from a partner port, therefore it will not be changed.", "VXC A-End product UID is from a CSP partner port, therefore it will not be changed.")
				}
				aEndPlanConfig.RequestedProductUID = aEndStateConfig.RequestedProductUID
			}

			partnerConfigDiags = plan.BEndPartnerConfig.As(ctx, &bEndPartnerConfigModel, basetypes.ObjectAsOptions{})
			diags = append(diags, partnerConfigDiags...)
			if !plan.BEndPartnerConfig.IsNull() {
				if !bEndPartnerConfigModel.Partner.IsNull() {
					if !bEndPartnerConfigModel.Partner.IsNull() {
						if bEndPartnerConfigModel.Partner.ValueString() != "transit" && bEndPartnerConfigModel.Partner.ValueString() != "vrouter" && bEndPartnerConfigModel.Partner.ValueString() != "a-end" {
							bEndCSP = true
						}
					}
				}
			}

			if state.BEndPartnerConfig.IsNull() {
				if !plan.BEndPartnerConfig.IsNull() {
					state.BEndPartnerConfig = plan.BEndPartnerConfig
				} else {
					state.BEndPartnerConfig = types.ObjectNull(vxcPartnerConfigAttrs)
				}
			} else {
				if !plan.BEndPartnerConfig.Equal(state.BEndPartnerConfig) && bEndCSP {
					resp.RequiresReplace = append(resp.RequiresReplace, path.Root("b_end_partner_config"))
				}
			}

			if bEndStateConfig.RequestedProductUID.IsNull() {
				bEndStateConfig.RequestedProductUID = bEndStateConfig.CurrentProductUID
				bEndPlanConfig.RequestedProductUID = bEndStateConfig.CurrentProductUID
			} else if bEndCSP {
				if !bEndPlanConfig.RequestedProductUID.IsNull() && !bEndPlanConfig.RequestedProductUID.Equal(bEndStateConfig.CurrentProductUID) {
					diags.AddWarning("VXC B-End product UID is from a partner port, therefore it will not be changed.", "VXC B-End product UID is from a CSP partner port, therefore it will not be changed.")
				}
				bEndPlanConfig.RequestedProductUID = bEndStateConfig.RequestedProductUID
			}

			newPlanAEndObj, aEndDiags := types.ObjectValueFrom(ctx, vxcEndConfigurationAttrs, aEndPlanConfig)
			newPlanBEndObj, bEndDiags := types.ObjectValueFrom(ctx, vxcEndConfigurationAttrs, bEndPlanConfig)
			diags = append(diags, aEndDiags...)
			diags = append(diags, bEndDiags...)
			plan.AEndConfiguration = newPlanAEndObj
			plan.BEndConfiguration = newPlanBEndObj
			newStateAEndObj, aEndDiags := types.ObjectValueFrom(ctx, vxcEndConfigurationAttrs, aEndStateConfig)
			newStateBEndObj, bEndDiags := types.ObjectValueFrom(ctx, vxcEndConfigurationAttrs, bEndStateConfig)
			diags = append(diags, aEndDiags...)
			diags = append(diags, bEndDiags...)
			state.AEndConfiguration = newStateAEndObj
			state.BEndConfiguration = newStateBEndObj
			req.Plan.Set(ctx, &plan)
			resp.Plan.Set(ctx, &plan)
			stateDiags := req.State.Set(ctx, &state)
			diags = append(diags, stateDiags...)
		}
	}

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
