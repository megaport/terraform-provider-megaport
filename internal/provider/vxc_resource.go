package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
		"owner_uid":      types.StringType,
		"product_uid":    types.StringType,
		"product_name":   types.StringType,
		"location_id":    types.Int64Type,
		"location":       types.StringType,
		"vlan":           types.Int64Type,
		"inner_vlan":     types.Int64Type,
		"vnic_index":     types.Int64Type,
		"secondary_name": types.StringType,
		"partner_config": types.ObjectType{},
	}

	vxcResourcesAttrs = map[string]attr.Type{
		"interface":      types.ListType{},
		"virtual_router": types.ObjectType{},
		"vll":            types.ObjectType{},
		"csp_connection": types.ObjectType{},
	}

	vxcCSPConnectionAttrs = map[string]attr.Type{
		"csp_connections": types.ListType{},
	}

	cspConnectionAWSAttrs = map[string]attr.Type{
		"connect_type":        types.StringType,
		"resource_name":       types.StringType,
		"resource_type":       types.StringType,
		"vlan":                types.Int64Type,
		"account":             types.StringType,
		"amazon_address":      types.StringType,
		"asn":                 types.Int64Type,
		"auth_key":            types.StringType,
		"customer_address":    types.StringType,
		"customer_ip_address": types.StringType,
		"id":                  types.Int64Type,
		"name":                types.StringType,
		"owner_account":       types.StringType,
		"peer_asn":            types.Int64Type,
		"type":                types.StringType,
		"vif_id":              types.StringType,
	}

	cspConnectionAWSHCAttrs = map[string]attr.Type{
		"connect_type":  types.StringType,
		"resource_name": types.StringType,
		"resource_type": types.StringType,
		"bandwidth":     types.Int64Type,
		"name":          types.StringType,
		"owner_account": types.StringType,
		"bandwidths":    types.ListType{},
		"connection_id": types.StringType,
	}

	cspConnectionAzureAttrs = map[string]attr.Type{
		"connect_type":  types.StringType,
		"resource_name": types.StringType,
		"resource_type": types.StringType,
		"bandwidth":     types.Int64Type,
		"managed":       types.BoolType,
		"megaports":     types.ListType{},
		"ports":         types.ListType{},
		"service_key":   types.StringType,
		"vlan":          types.Int64Type,
	}

	cspConnectionAzureMegaportAttrs = map[string]attr.Type{
		"port": types.Int64Type,
		"type": types.StringType,
		"vxc":  types.Int64Type,
	}

	cspConnectionAzurePortAttrs = map[string]attr.Type{
		"service_id":      types.Int64Type,
		"type":            types.StringType,
		"vxc_service_ids": types.ListType{},
	}

	cspConnectionGoogleAttrs = map[string]attr.Type{
		"bandwidth":     types.Int64Type,
		"connect_type":  types.StringType,
		"resource_name": types.StringType,
		"resource_type": types.StringType,
		"bandwidths":    types.ListType{},
		"megaports":     types.ListType{},
		"ports":         types.ListType{},
		"csp_name":      types.StringType,
		"pairing_key":   types.StringType,
	}

	cspConnectionGoogleMegaportAttrs = map[string]attr.Type{
		"port": types.Int64Type,
		"vxc":  types.Int64Type,
	}

	cspConnectionGooglePortAttrs = map[string]attr.Type{
		"service_id":      types.Int64Type,
		"vxc_service_ids": types.ListType{},
	}

	cspConnectionVirtualRouterAttrs = map[string]attr.Type{
		"connect_type":        types.StringType,
		"resource_name":       types.StringType,
		"resource_type":       types.StringType,
		"vlan":                types.Int64Type,
		"interfaces":          types.ListType{},
		"ip_addresses":        types.ListType{},
		"virtual_router_name": types.StringType,
	}

	cspConnectionVirtualRouterInterfaceAttrs = map[string]attr.Type{
		"ip_addresses": types.ListType{},
	}

	cspConnectionTransitAttrs = map[string]attr.Type{
		"connect_type":         types.StringType,
		"resource_name":        types.StringType,
		"resource_type":        types.StringType,
		"customer_ip4_address": types.StringType,
		"customer_ip6_network": types.StringType,
		"ipv4_gateway_address": types.StringType,
		"ipv6_gateway_address": types.StringType,
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

	PortUID           types.String `tfsdk:"port_uid"`
	AEndConfiguration types.Object `tfsdk:"a_end"`
	BEndConfiguration types.Object `tfsdk:"b_end"`

	Resources   types.Object `tfsdk:"resources"`
	VXCApproval types.Object `tfsdk:"vxc_approval"`
}

// vxcResourcesModel represents the resources associated with a VXC.
type vxcResourcesModel struct {
	Interface     types.List   `tfsdk:"interface"`
	VirtualRouter types.Object `tfsdk:"virtual_router"`
	VLL           types.Object `tfsdk:"vll"`
	CSPConnection types.Object `tfsdk:"csp_connection"`
}

// vxcCSPConnectionModel represents the CSP connection schema data.
type vxcCSPConnectionModel struct {
	CSPConnections types.List `tfsdk:"csp_connections"`
}

// cspConnection is an interface to ensure the CSP connection is implemented.
type cspConnection interface {
	isCSPConnection()
}

// cspConnectionAWSModel represents the configuration of a CSP connection for AWS Virtual Interface.
type cspConnectionAWSModel struct {
	cspConnection
	ConnectType       types.String `tfsdk:"connect_type"`
	ResourceName      types.String `tfsdk:"resource_name"`
	ResourceType      types.String `tfsdk:"resource_type"`
	VLAN              types.Int64  `tfsdk:"vlan"`
	Account           types.String `tfsdk:"account"`
	AmazonAddress     types.String `tfsdk:"amazon_address"`
	ASN               types.Int64  `tfsdk:"asn"`
	AuthKey           types.String `tfsdk:"auth_key"`
	CustomerAddress   types.String `tfsdk:"customer_address"`
	CustomerIPAddress types.String `tfsdk:"customer_ip_address"`
	ID                types.Int64  `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	OwnerAccount      types.String `tfsdk:"owner_account"`
	PeerASN           types.Int64  `tfsdk:"peer_asn"`
	Type              types.String `tfsdk:"type"`
	VIFID             types.String `tfsdk:"vif_id"`
}

// cspConnectionAWSHCModel represents the configuration of a CSP connection for AWS Hosted Connection.
type cspConnectionAWSHCModel struct {
	cspConnection
	ConnectType  types.String `tfsdk:"connect_type"`
	ResourceName types.String `tfsdk:"resource_name"`
	ResourceType types.String `tfsdk:"resource_type"`
	Bandwidth    types.Int64  `tfsdk:"bandwidth"`
	Name         types.String `tfsdk:"name"`
	OwnerAccount types.String `tfsdk:"owner_account"`
	Bandwidths   types.List   `tfsdk:"bandwidths"`
	ConnectionID types.String `tfsdk:"connection_id"`
}

// cspConnectionAzureModel represents the configuration of a CSP connection for Azure ExpressRoute.
type cspConnectionAzureModel struct {
	cspConnection
	ConnectType  types.String `tfsdk:"connect_type"`
	ResourceName types.String `tfsdk:"resource_name"`
	ResourceType types.String `tfsdk:"resource_type"`
	Bandwidth    types.Int64  `tfsdk:"bandwidth"`
	Managed      types.Bool   `tfsdk:"managed"`
	Megaports    types.List   `tfsdk:"megaports"`
	Ports        types.List   `tfsdk:"ports"`
	ServiceKey   types.String `tfsdk:"service_key"`
	VLAN         types.Int64  `tfsdk:"vlan"`
}

// CSPConnectionAzureMegaport represents the configuration of a CSP connection for Azure ExpressRoute megaport.
type cspConnectionAzureMegaportModel struct {
	Port types.Int64  `tfsdk:"port"`
	Type types.String `tfsdk:"type"`
	VXC  types.Int64  `tfsdk:"vxc,omitempty"`
}

// cspConnectionAzurePortModel represents the configuration of a CSP connection for Azure ExpressRoute port.
type cspConnectionAzurePortModel struct {
	ServiceID     types.Int64  `tfsdk:"service_id"`
	Type          types.String `tfsdk:"type"`
	VXCServiceIDs types.List   `tfsdk:"vxc_service_ids"`
}

// cspConnectionGoogleModel represents the configuration of a CSP connection for Google Cloud Interconnect.
type cspConnectionGoogleModel struct {
	cspConnection
	Bandwidth    types.Int64  `tfsdk:"bandwidth"`
	ConnectType  types.String `tfsdk:"connect_type"`
	ResourceName types.String `tfsdk:"resource_name"`
	ResourceType types.String `tfsdk:"resource_type"`
	Bandwidths   types.List   `tfsdk:"bandwidths"`
	Megaports    types.List   `tfsdk:"megaports"`
	Ports        types.List   `tfsdk:"ports"`
	CSPName      types.String `tfsdk:"csp_name"`
	PairingKey   types.String `tfsdk:"pairing_key"`
}

// cspConnectionGoogleMegaportModel represents the configuration of a CSP connection for Google Cloud Interconnect megaport.
type cspConnectionGoogleMegaportModel struct {
	Port types.Int64 `tfsdk:"port"`
	VXC  types.Int64 `tfsdk:"vxc"`
}

// cspConnectionGooglePortModel represents the configuration of a CSP connection for Google Cloud Interconnect port.
type cspConnectionGooglePortModel struct {
	ServiceID     types.Int64 `tfsdk:"service_id"`
	VXCServiceIDs types.List  `tfsdk:"vxc_service_ids"`
}

// cspConnectionVirtualRouterModel represents the configuration of a CSP connection for Virtual Router.
type cspConnectionVirtualRouterModel struct {
	cspConnection
	ConnectType       types.String `tfsdk:"connect_type"`
	ResourceName      types.String `tfsdk:"resource_name"`
	ResourceType      types.String `tfsdk:"resource_type"`
	VLAN              types.Int64  `tfsdk:"vlan"`
	Interfaces        types.List   `tfsdk:"interfaces"`
	IPAddresses       types.List   `tfsdk:"ip_addresses"`
	VirtualRouterName types.String `tfsdk:"virtual_router_name"`
}

// cspConnectionVirtualRouterInterfaceModel represents the configuration of a CSP connection for Virtual Router interface.
type cspConnectionVirtualRouterInterfaceModel struct {
	IPAddresses types.List `tfsdk:"ip_addresses"`
}

// cspConnectionTransit represents the configuration of a CSP connection for a Transit VXC.
type cspConnectionTransit struct {
	cspConnection
	ConnectType        types.String `tfsdk:"connect_type"`
	ResourceName       types.String `tfsdk:"resource_name"`
	ResourceType       types.String `tfsdk:"resource_type"`
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
	UID                   types.String `tfsdk:"product_uid,omitempty"`
	Name                  types.String `tfsdk:"product_name"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	Location              types.String `tfsdk:"location"`
	VLAN                  types.Int64  `tfsdk:"vlan,omitempty"`
	InnerVLAN             types.Int64  `tfsdk:"inner_vlan,omitempty"`
	ProductUID            types.String `tfsdk:"product_uid,omitempty"`
	NetworkInterfaceIndex types.Int64  `tfsdk:"vnic_index,omitempty"`
	SecondaryName         types.String `tfsdk:"secondary_name"`
	PartnerConfig         types.Object `tfsdk:"partner_config,omitempty"`
}

type vxcPartnerConfigurationModel struct {
	Partner             types.String `tfsdk:"partner"`
	AWSPartnerConfig    types.Object `tfsdk:"aws_config,omitempty"`
	AzurePartnerConfig  types.Object `tfsdk:"azure_config,omitempty"`
	GooglePartnerConfig types.Object `tfsdk:"google_config,omitempty"`
	OraclePartnerConfig types.Object `tfsdk:"oracle_config,omitempty"`
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
	ASN               types.Int64  `tfsdk:"asn,omitempty"`
	AmazonASN         types.Int64  `tfsdk:"amazon_asn,omitempty"`
	AuthKey           types.String `tfsdk:"auth_key,omitempty"`
	Prefixes          types.String `tfsdk:"prefixes,omitempty"`
	CustomerIPAddress types.String `tfsdk:"customer_ip_address,omitempty"`
	AmazonIPAddress   types.String `tfsdk:"amazon_ip_address,omitempty"`
	ConnectionName    types.String `tfsdk:"name,omitempty"`
}

// vxcPartnerConfigAzureModel maps the partner configuration schema data for Azure.
type vxcPartnerConfigAzureModel struct {
	vxcPartnerConfig
	ConnectType types.String `tfsdk:"connect_type"`
	ServiceKey  types.String `tfsdk:"service_key"`
}

// vxcPartnerConfigGoogleModel maps the partner configuration schema data for Google.
type vxcPartnerConfigGoogleModel struct {
	vxcPartnerConfig
	ConnectType types.String `tfsdk:"connect_type"`
	PairingKey  types.String `tfsdk:"pairing_key"`
}

// vxcPartnerConfigOracleModel maps the partner configuration schema data for Oracle.
type vxcPartnerConfigOracleModel struct {
	vxcPartnerConfig
	ConnectType      types.String `tfsdk:"connect_type"`
	VirtualCircuitId types.String `tfsdk:"virtual_circuit_id"`
}

func (orm *vxcResourceModel) fromAPIVXC(ctx context.Context, v *megaport.VXC) error {
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
	orm.LiveDate = types.StringValue(v.LiveDate.Format(time.RFC850))
	orm.CreateDate = types.StringValue(v.CreateDate.Format(time.RFC850))
	orm.ContractStartDate = types.StringValue(v.ContractStartDate.Format(time.RFC850))
	orm.ContractEndDate = types.StringValue(v.ContractEndDate.Format(time.RFC850))

	aEndModel := &vxcEndConfigurationModel{
		OwnerUID:              types.StringValue(v.AEndConfiguration.OwnerUID),
		UID:                   types.StringValue(v.AEndConfiguration.UID),
		Name:                  types.StringValue(v.AEndConfiguration.Name),
		LocationID:            types.Int64Value(int64(v.AEndConfiguration.LocationID)),
		Location:              types.StringValue(v.AEndConfiguration.Location),
		VLAN:                  types.Int64Value(int64(v.AEndConfiguration.VLAN)),
		InnerVLAN:             types.Int64Value(int64(v.AEndConfiguration.InnerVLAN)),
		NetworkInterfaceIndex: types.Int64Value(int64(v.AEndConfiguration.NetworkInterfaceIndex)),
		SecondaryName:         types.StringValue(v.AEndConfiguration.SecondaryName),
	}
	aEnd, diag := types.ObjectValueFrom(ctx, vxcEndConfigurationAttrs, aEndModel)
	if diag.HasError() {
		return errors.New("failed to convert a-end configuration")
	}
	orm.AEndConfiguration = aEnd

	bEndModel := &vxcEndConfigurationModel{
		OwnerUID:              types.StringValue(v.BEndConfiguration.OwnerUID),
		UID:                   types.StringValue(v.BEndConfiguration.UID),
		Name:                  types.StringValue(v.BEndConfiguration.Name),
		LocationID:            types.Int64Value(int64(v.BEndConfiguration.LocationID)),
		Location:              types.StringValue(v.BEndConfiguration.Location),
		VLAN:                  types.Int64Value(int64(v.BEndConfiguration.VLAN)),
		InnerVLAN:             types.Int64Value(int64(v.BEndConfiguration.InnerVLAN)),
		NetworkInterfaceIndex: types.Int64Value(int64(v.BEndConfiguration.NetworkInterfaceIndex)),
		SecondaryName:         types.StringValue(v.BEndConfiguration.SecondaryName),
	}
	bEnd, diag := types.ObjectValueFrom(ctx, vxcEndConfigurationAttrs, bEndModel)
	if diag.HasError() {
		return errors.New("failed to convert b-end configuration")
	}
	orm.BEndConfiguration = bEnd

	vxcApprovalModel := &vxcApprovalModel{
		Status:   types.StringValue(v.VXCApproval.Status),
		Message:  types.StringValue(v.VXCApproval.Message),
		UID:      types.StringValue(v.VXCApproval.UID),
		Type:     types.StringValue(v.VXCApproval.Type),
		NewSpeed: types.Int64Value(int64(v.VXCApproval.NewSpeed)),
	}
	vxcApproval, diag := types.ObjectValueFrom(ctx, vxcApprovalAttrs, vxcApprovalModel)
	if diag.HasError() {
		return errors.New("failed to convert VXC approval")
	}
	orm.VXCApproval = vxcApproval

	var resourcesModel vxcResourcesModel
	if v.Resources.Interface != nil {
		interfaces := []types.Object{}
		for _, i := range v.Resources.Interface {
			interfaceModel := portInterfaceModel{
				Demarcation:  types.StringValue(i.Demarcation),
				Description:  types.StringValue(i.Description),
				ID:           types.Int64Value(int64(i.ID)),
				LOATemplate:  types.StringValue(i.LOATemplate),
				Media:        types.StringValue(i.Media),
				Name:         types.StringValue(i.Name),
				PortSpeed:    types.Int64Value(int64(i.PortSpeed)),
				ResourceName: types.StringValue(i.ResourceName),
				ResourceType: types.StringValue(i.ResourceType),
				Up:           types.Int64Value(int64(i.Up)),
			}
			interfaceObj, diag := types.ObjectValueFrom(ctx, vxcResourcesAttrs, interfaceModel)
			if diag.HasError() {
				return errors.New("failed to convert port interface")
			}
			interfaces = append(interfaces, interfaceObj)
		}
		portInterface, diag := types.ListValueFrom(ctx, types.ObjectType{}, interfaces)
		if diag.HasError() {
			return errors.New("failed to convert port interfaces list")
		}
		resourcesModel.Interface = portInterface
	}
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
	vll, diag := types.ObjectValueFrom(ctx, vllConfigAttrs, vllModel)
	if diag.HasError() {
		return errors.New("failed to convert VLL configuration")
	}
	resourcesModel.VLL = vll

	virtualRouterModel := &virtualRouterModel{
		MCRAsn:             types.Int64Value(int64(v.Resources.VirtualRouter.MCRAsn)),
		ResourceName:       types.StringValue(v.Resources.VirtualRouter.ResourceName),
		ResourceType:       types.StringValue(v.Resources.VirtualRouter.ResourceType),
		Speed:              types.Int64Value(int64(v.Resources.VirtualRouter.Speed)),
		BGPShutdownDefault: types.BoolValue(v.Resources.VirtualRouter.BGPShutdownDefault),
	}
	virtualRouter, diag := types.ObjectValueFrom(ctx, virtualRouterAttrs, virtualRouterModel)
	if diag.HasError() {
		return errors.New("failed to convert virtual router configuration")
	}
	resourcesModel.VirtualRouter = virtualRouter

	cspConnectionModel := &vxcCSPConnectionModel{}
	cspConnections := []types.Object{}
	for _, c := range v.Resources.CSPConnection.CSPConnection {
		cspConnection, err := fromAPICSPConnection(ctx, c)
		if err != nil {
			return err
		}
		cspConnections = append(cspConnections, *cspConnection)
	}
	cspConnection, diag := types.ListValueFrom(ctx, types.ObjectType{}, cspConnections)
	if diag.HasError() {
		return errors.New("failed to convert CSP connections")
	}
	cspConnectionModel.CSPConnections = cspConnection
	cspConnectionObj, diag := types.ObjectValueFrom(ctx, vxcCSPConnectionAttrs, cspConnectionModel)
	if diag.HasError() {
		return errors.New("failed to convert CSP connection object")
	}
	resourcesModel.CSPConnection = cspConnectionObj
	resourcesObj, diag := types.ObjectValueFrom(ctx, vxcResourcesAttrs, resourcesModel)
	if diag.HasError() {
		return errors.New("failed to convert resources object")
	}
	orm.Resources = resourcesObj
	return nil
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
		Attributes: map[string]schema.Attribute{
			"last_updated": schema.StringAttribute{
				Description: "The last time the resource was updated.",
				Computed:    true,
			},
			"uid": schema.StringAttribute{
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
			"port_uid": schema.StringAttribute{
				Description: "The UID of the port the VXC is connected to.",
				Required:    true,
			},
			"type": schema.StringAttribute{
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"create_date": schema.StringAttribute{
				Description: "The date the product was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				Optional:    true,
			},
			"cost_centre": schema.StringAttribute{
				Description: "A customer reference number to be included in billing information and invoices.",
				Optional:    true,
			},
			"resources": schema.SingleNestedAttribute{
				Description: "The resources associated with the VXC.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"interface": schema.ListNestedAttribute{
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
					"csp_connection": schema.SingleNestedAttribute{
						Description: "The CSP connection associated with the VXC.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"csp_connections": schema.ListNestedAttribute{
								Description: "The CSP connections associated with the VXC.",
								Computed:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										// Fields for all CSP Connections
										"connect_type": schema.StringAttribute{
											Description: "The connection type of the CSP connection.",
											Computed:    true,
										},
										"resource_name": schema.StringAttribute{
											Description: "The resource name of the CSP connection.",
											Computed:    true,
										},
										"resource_type": schema.StringAttribute{
											Description: "The resource type of the CSP connection.",
											Computed:    true,
										},
										// Fields for AWS, Azure, and Virtual Router
										"vlan": schema.Int64Attribute{
											Description: "The VLAN of the CSP connection.",
											Computed:    true,
										},
										// AWS CSP Connection Fields
										"name": schema.StringAttribute{
											Description: "The name of the CSP connection.",
											Computed:    true,
										},
										"owner_account": schema.StringAttribute{
											Description: "The owner's AWS account of the CSP connection.",
											Computed:    true,
										},
										// Fields for AWS, Azure, and Google Cloud
										"bandwidth": schema.Int64Attribute{
											Description: "The bandwidth of the CSP connection.",
											Computed:    true,
										},
										// Field for AWS and Google Cloud
										"bandwidths": schema.ListAttribute{
											Description: "The bandwidths of the CSP connection.",
											Computed:    true,
											ElementType: types.Int64Type,
										},
										// Fields for AWS VIF and Transit VXC
										"customer_ip_address": schema.StringAttribute{
											Description: "The customer IP address of the CSP connection.",
											Computed:    true,
										},
										// AWS VIF CSP Connection Fields
										"account": schema.StringAttribute{
											Description: "The account of the CSP connection.",
											Computed:    true,
										},
										"amazon_address": schema.StringAttribute{
											Description: "The Amazon address of the CSP connection.",
											Computed:    true,
										},
										"asn": schema.Int64Attribute{
											Description: "The ASN of the CSP connection.",
											Computed:    true,
										},
										"auth_key": schema.StringAttribute{
											Description: "The authentication key of the CSP connection.",
											Computed:    true,
										},
										"customer_address": schema.StringAttribute{
											Description: "The customer address of the CSP connection.",
											Computed:    true,
										},

										"id": schema.Int64Attribute{
											Description: "The ID of the CSP connection.",
											Computed:    true,
										},
										"peer_asn": schema.Int64Attribute{
											Description: "The peer ASN of the CSP connection.",
											Computed:    true,
										},
										"type": schema.StringAttribute{
											Description: "The type of the AWS Virtual Interface.",
											Computed:    true,
										},
										"vif_id": schema.StringAttribute{
											Description: "The ID of the AWS Virtual Interface.",
											Computed:    true,
										},
										// AWS Hosted Connection Fields
										"connection_id": schema.StringAttribute{
											Description: "The hosted connection ID of the CSP connection.",
											Computed:    true,
										},
										// Azure and Google Cloud CSP Connection Fields
										"ports": schema.ListNestedAttribute{
											Description: "The ports of the CSP connection.",
											Computed:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"service_id": schema.Int64Attribute{
														Description: "The service ID of the port.",
														Computed:    true,
													},
													"vxc_service_ids": schema.ListAttribute{
														Description: "The VXC service IDs of the port.",
														Computed:    true,
														ElementType: types.Int64Type,
													},
													"type": schema.StringAttribute{
														Description: "The type of the port.",
														Computed:    true,
													},
												},
											},
										},
										"megaports": schema.ListNestedAttribute{
											Description: "The Megaports of the CSP connection.",
											Computed:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"port": schema.Int64Attribute{
														Description: "The numeric identifier for the port.",
														Computed:    true,
													},
													"type": schema.StringAttribute{
														Description: "The type of the Megaport.",
														Computed:    true,
													},
													"vxc": schema.Int64Attribute{
														Description: "The numeric identifier for the VXC (Google).",
														Computed:    true,
													},
												},
											},
										},
										// Azure Fields
										"managed": schema.BoolAttribute{
											Description: "Whether the CSP connection is managed.",
											Computed:    true,
										},
										"service_key": schema.StringAttribute{
											Description: "The service key of the CSP connection.",
											Computed:    true,
										},
										// Google Cloud Fields
										"csp_name": schema.StringAttribute{
											Description: "The name of the CSP connection.",
											Computed:    true,
										},
										"pairing_key": schema.StringAttribute{
											Description: "The pairing key of the Google Cloud connection.",
											Computed:    true,
										},
										// Virtual Router Fields
										"interfaces": schema.ListNestedAttribute{
											Description: "The interfaces of the Virtual Router connection.",
											Computed:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"ip_addresses": schema.ListAttribute{
														Description: "The IP addresses of the interface.",
														Computed:    true,
														ElementType: types.StringType,
													},
												},
											},
										},
										"ip_addresses": schema.ListAttribute{
											Description: "The IP addresses of the Virtual Router.",
											Computed:    true,
											ElementType: types.StringType,
										},
										"virtual_router_name": schema.StringAttribute{
											Description: "The name of the Virtual Router.",
											Computed:    true,
										},
										// Transit VXC Fields
										"customer_ip6_network": schema.StringAttribute{
											Description: "The customer IPv6 network of the Transit VXC connection.",
											Computed:    true,
										},
										"ipv4_gateway_address": schema.StringAttribute{
											Description: "The IPv4 gateway address of the Transit VXC connection.",
											Computed:    true,
										},
										"ipv6_gateway_address": schema.StringAttribute{
											Description: "The IPv6 gateway address of the Transit VXC connection.",
											Computed:    true,
										},
									},
								},
							},
						},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"contract_end_date": schema.StringAttribute{
				Description: "The date the contract ends.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"owner_uid": schema.StringAttribute{
						Description: "The owner UID of the A-End configuration.",
						Computed:    true,
					},
					"product_uid": schema.StringAttribute{
						Description: "The product UID of the A-End configuration.",
						Optional:    true,
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
					"vlan": schema.Int64Attribute{
						Description: "The VLAN of the A-End configuration.",
						Optional:    true,
					},
					"inner_vlan": schema.Int64Attribute{
						Description: "The inner VLAN of the A-End configuration.",
						Optional:    true,
					},
					"vnic_index": schema.Int64Attribute{
						Description: "The network interface index of the A-End configuration.",
						Optional:    true,
					},
					"secondary_name": schema.StringAttribute{
						Description: "The secondary name of the A-End configuration.",
						Computed:    true,
					},
					"partner_config": schema.SingleNestedAttribute{
						Description: "The partner configuration of the A-End order configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"partner": schema.StringAttribute{
								Description: "The partner of the partner configuration.",
								Required:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("aws", "azure", "google", "oracle"),
								},
							},
							"aws_config": schema.SingleNestedAttribute{
								Description: "The AWS partner configuration.",
								Optional:    true,
								Validators: []validator.Object{
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("azure_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("google_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("oracle_config")),
								},
								Attributes: map[string]schema.Attribute{
									"connect_type": schema.StringAttribute{
										Description: "The connection type of the partner configuration. Required for all partner configurations.",
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
										Required:    true,
									},
									"amazon_asn": schema.Int64Attribute{
										Description: "The Amazon ASN of the partner configuration.",
										Required:    true,
									},
									"auth_key": schema.StringAttribute{
										Description: "The authentication key of the partner configuration.",
										Required:    true,
									},
									"prefixes": schema.StringAttribute{
										Description: "The prefixes of the partner configuration.",
										Required:    true,
									},
									"customer_ip_address": schema.StringAttribute{
										Description: "The customer IP address of the partner configuration.",
										Required:    true,
									},
									"amazon_ip_address": schema.StringAttribute{
										Description: "The Amazon IP address of the partner configuration.",
										Required:    true,
									},
									"name": schema.StringAttribute{
										Description: "The name of the partner configuration.",
										Optional:    true,
									},
								},
							},
							"azure_config": schema.SingleNestedAttribute{
								Description: "The Azure partner configuration.",
								Optional:    true,
								Validators: []validator.Object{
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("aws_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("google_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("oracle_config")),
								},
								Attributes: map[string]schema.Attribute{
									"connect_type": schema.StringAttribute{
										Description: "The connection type of the partner configuration. Required for all partner configurations.",
										Required:    true,
									},
									"service_key": schema.StringAttribute{
										Description: "The service key of the partner configuration. Required for Azure partner configurations.",
										Required:    true,
									},
								},
							},
							"google_config": schema.SingleNestedAttribute{
								Description: "The Google partner configuration.",
								Optional:    true,
								Validators: []validator.Object{
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("aws_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("azure_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("oracle_config")),
								},
								Attributes: map[string]schema.Attribute{
									"connect_type": schema.StringAttribute{
										Description: "The connection type of the partner configuration. Required for all partner configurations.",
										Required:    true,
									},
									"pairing_key": schema.StringAttribute{
										Description: "The pairing key of the partner configuration. Required for Google partner configurations.",
										Required:    true,
									},
								},
							},
							"oracle_config": schema.SingleNestedAttribute{
								Description: "The Oracle partner configuration.",
								Optional:    true,
								Validators: []validator.Object{
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("aws_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("azure_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("google_config")),
								},
								Attributes: map[string]schema.Attribute{
									"connect_type": schema.StringAttribute{
										Description: "The connection type of the partner configuration. Required for all partner configurations.",
										Required:    true,
									},
									"virtual_circuit_id": schema.StringAttribute{
										Description: "The virtual circuit ID of the partner configuration. Required for Oracle partner configurations.",
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
			"b_end": schema.SingleNestedAttribute{
				Description: "The current B-End configuration of the VXC.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"owner_uid": schema.StringAttribute{
						Description: "The owner UID of the B-End configuration.",
						Computed:    true,
					},
					"product_uid": schema.StringAttribute{
						Description: "The product UID of the B-End configuration.",
						Optional:    true,
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
					"vlan": schema.Int64Attribute{
						Description: "The VLAN of the B-End configuration.",
						Optional:    true,
					},
					"inner_vlan": schema.Int64Attribute{
						Description: "The inner VLAN of the B-End configuration.",
						Optional:    true,
					},
					"vnic_index": schema.Int64Attribute{
						Description: "The network interface index of the B-End configuration.",
						Optional:    true,
					},
					"secondary_name": schema.StringAttribute{
						Description: "The secondary name of the B-End configuration.",
						Computed:    true,
					},
					"partner_config": schema.SingleNestedAttribute{
						Description: "The partner configuration of the B-End order configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"partner": schema.StringAttribute{
								Description: "The partner of the partner configuration.",
								Required:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("aws", "azure", "google", "oracle"),
								},
							},
							"aws_config": schema.SingleNestedAttribute{
								Description: "The AWS partner configuration.",
								Optional:    true,
								Validators: []validator.Object{
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("azure_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("google_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("oracle_config")),
								},
								Attributes: map[string]schema.Attribute{
									"connect_type": schema.StringAttribute{
										Description: "The connection type of the partner configuration. Required for all partner configurations.",
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
										Required:    true,
									},
									"amazon_asn": schema.Int64Attribute{
										Description: "The Amazon ASN of the partner configuration.",
										Required:    true,
									},
									"auth_key": schema.StringAttribute{
										Description: "The authentication key of the partner configuration.",
										Required:    true,
									},
									"prefixes": schema.StringAttribute{
										Description: "The prefixes of the partner configuration.",
										Required:    true,
									},
									"customer_ip_address": schema.StringAttribute{
										Description: "The customer IP address of the partner configuration.",
										Required:    true,
									},
									"amazon_ip_address": schema.StringAttribute{
										Description: "The Amazon IP address of the partner configuration.",
										Required:    true,
									},
									"name": schema.StringAttribute{
										Description: "The name of the partner configuration.",
										Optional:    true,
									},
								},
							},
							"azure_config": schema.SingleNestedAttribute{
								Description: "The Azure partner configuration.",
								Optional:    true,
								Validators: []validator.Object{
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("aws_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("google_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("oracle_config")),
								},
								Attributes: map[string]schema.Attribute{
									"connect_type": schema.StringAttribute{
										Description: "The connection type of the partner configuration. Required for all partner configurations.",
										Required:    true,
									},
									"service_key": schema.StringAttribute{
										Description: "The service key of the partner configuration. Required for Azure partner configurations.",
										Required:    true,
									},
								},
							},
							"google_config": schema.SingleNestedAttribute{
								Description: "The Google partner configuration.",
								Optional:    true,
								Validators: []validator.Object{
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("aws_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("azure_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("oracle_config")),
								},
								Attributes: map[string]schema.Attribute{
									"connect_type": schema.StringAttribute{
										Description: "The connection type of the partner configuration. Required for all partner configurations.",
										Required:    true,
									},
									"pairing_key": schema.StringAttribute{
										Description: "The pairing key of the partner configuration. Required for Google partner configurations.",
										Required:    true,
									},
								},
							},
							"oracle_config": schema.SingleNestedAttribute{
								Description: "The Oracle partner configuration.",
								Optional:    true,
								Validators: []validator.Object{
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("aws_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("azure_config")),
									objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("google_config")),
								},
								Attributes: map[string]schema.Attribute{
									"connect_type": schema.StringAttribute{
										Description: "The connection type of the partner configuration. Required for all partner configurations.",
										Required:    true,
									},
									"virtual_circuit_id": schema.StringAttribute{
										Description: "The virtual circuit ID of the partner configuration. Required for Oracle partner configurations.",
										Required:    true,
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
		PortUID:   plan.PortUID.ValueString(),
		VXCName:   plan.Name.ValueString(),
		Term:      int(plan.ContractTermMonths.ValueInt64()),
		RateLimit: int(plan.RateLimit.ValueInt64()),

		WaitForProvision: true,
		WaitForTime:      10 * time.Minute,
	}

	if !plan.Shutdown.IsNull() {
		buyReq.Shutdown = plan.Shutdown.ValueBool()
	}

	aEndObj := plan.AEndConfiguration
	bEndOBj := plan.BEndConfiguration

	var a vxcEndConfigurationModel
	aEndDiags := aEndObj.As(ctx, a, basetypes.ObjectAsOptions{})
	if aEndDiags.HasError() {
		resp.Diagnostics.Append(aEndDiags...)
		return
	}
	var aPartnerConfig vxcPartnerConfigurationModel
	aPartnerDiags := a.PartnerConfig.As(ctx, aPartnerConfig, basetypes.ObjectAsOptions{})
	if aPartnerDiags.HasError() {
		resp.Diagnostics.Append(aPartnerDiags...)
		return
	}
	aEndConfig := &megaport.VXCOrderEndpointConfiguration{
		ProductUID: a.ProductUID.ValueString(),
		VLAN:       int(a.VLAN.ValueInt64()),
	}
	if !a.InnerVLAN.IsNull() && !a.NetworkInterfaceIndex.IsNull() {
		aEndConfig.VXCOrderMVEConfig = &megaport.VXCOrderMVEConfig{
			InnerVLAN:             int(a.InnerVLAN.ValueInt64()),
			NetworkInterfaceIndex: int(a.NetworkInterfaceIndex.ValueInt64()),
		}
	}
	switch aPartnerConfig.Partner.ValueString() {
	case "aws":
		var awsConfig vxcPartnerConfigAWSModel
		awsDiags := aPartnerConfig.AWSPartnerConfig.As(ctx, awsConfig, basetypes.ObjectAsOptions{})
		if awsDiags.HasError() {
			resp.Diagnostics.Append(awsDiags...)
			return
		}
		aEndPartnerConfig := &megaport.VXCPartnerConfigAWS{
			ConnectType:  awsConfig.ConnectType.ValueString(),
			Type:         awsConfig.Type.ValueString(),
			OwnerAccount: awsConfig.OwnerAccount.ValueString(),
			ASN:          int(awsConfig.ASN.ValueInt64()),
			AmazonASN:    int(awsConfig.AmazonASN.ValueInt64()),
			AuthKey:      awsConfig.AuthKey.ValueString(),
			Prefixes:     awsConfig.Prefixes.ValueString(),
		}
		aEndConfig.PartnerConfig = aEndPartnerConfig
	case "azure":
		var azureConfig vxcPartnerConfigAzureModel
		azureDiags := aPartnerConfig.AzurePartnerConfig.As(ctx, azureConfig, basetypes.ObjectAsOptions{})
		if azureDiags.HasError() {
			resp.Diagnostics.Append(azureDiags...)
			return
		}
		aEndPartnerConfig := &megaport.VXCPartnerConfigAzure{
			ConnectType: azureConfig.ConnectType.ValueString(),
			ServiceKey:  azureConfig.ServiceKey.ValueString(),
		}
		aEndConfig.PartnerConfig = aEndPartnerConfig
	case "google":
		var googleConfig vxcPartnerConfigGoogleModel
		googleDiags := aPartnerConfig.GooglePartnerConfig.As(ctx, googleConfig, basetypes.ObjectAsOptions{})
		if googleDiags.HasError() {
			resp.Diagnostics.Append(googleDiags...)
			return
		}
		aEndPartnerConfig := &megaport.VXCPartnerConfigGoogle{
			ConnectType: googleConfig.ConnectType.ValueString(),
			PairingKey:  googleConfig.PairingKey.ValueString(),
		}
		aEndConfig.PartnerConfig = aEndPartnerConfig
	case "oracle":
		var oracleConfig vxcPartnerConfigOracleModel
		oracleDiags := aPartnerConfig.OraclePartnerConfig.As(ctx, oracleConfig, basetypes.ObjectAsOptions{})
		if oracleDiags.HasError() {
			resp.Diagnostics.Append(oracleDiags...)
			return
		}
		aEndPartnerConfig := &megaport.VXCPartnerConfigOracle{
			ConnectType:      oracleConfig.ConnectType.ValueString(),
			VirtualCircuitId: oracleConfig.VirtualCircuitId.ValueString(),
		}
		aEndConfig.PartnerConfig = aEndPartnerConfig
	default:
		resp.Diagnostics.AddError(
			"Error creating VXC",
			"Could not create VXC with name "+plan.Name.ValueString()+": Partner configuration not supported",
		)
		return
	}

	var b vxcEndConfigurationModel
	bEndDiags := bEndOBj.As(ctx, b, basetypes.ObjectAsOptions{})
	if bEndDiags.HasError() {
		resp.Diagnostics.Append(bEndDiags...)
		return
	}
	bEndConfig := &megaport.VXCOrderEndpointConfiguration{
		ProductUID: a.ProductUID.ValueString(),
		VLAN:       int(a.VLAN.ValueInt64()),
	}
	if !b.InnerVLAN.IsNull() && !b.NetworkInterfaceIndex.IsNull() {
		bEndConfig.VXCOrderMVEConfig = &megaport.VXCOrderMVEConfig{
			InnerVLAN:             int(b.InnerVLAN.ValueInt64()),
			NetworkInterfaceIndex: int(b.NetworkInterfaceIndex.ValueInt64()),
		}
	}
	var bPartnerConfig vxcPartnerConfigurationModel
	bPartnerDiags := b.PartnerConfig.As(ctx, bPartnerConfig, basetypes.ObjectAsOptions{})
	if bPartnerDiags.HasError() {
		resp.Diagnostics.Append(aPartnerDiags...)
		return
	}

	switch bPartnerConfig.Partner.ValueString() {
	case "aws":
		var awsConfig vxcPartnerConfigAWSModel
		awsDiags := aPartnerConfig.AWSPartnerConfig.As(ctx, awsConfig, basetypes.ObjectAsOptions{})
		if awsDiags.HasError() {
			resp.Diagnostics.Append(awsDiags...)
			return
		}
		bEndPartnerConfig := &megaport.VXCPartnerConfigAWS{
			ConnectType:  awsConfig.ConnectType.ValueString(),
			Type:         awsConfig.Type.ValueString(),
			OwnerAccount: awsConfig.OwnerAccount.ValueString(),
			ASN:          int(awsConfig.ASN.ValueInt64()),
			AmazonASN:    int(awsConfig.AmazonASN.ValueInt64()),
			AuthKey:      awsConfig.AuthKey.ValueString(),
			Prefixes:     awsConfig.Prefixes.ValueString(),
		}
		bEndConfig.PartnerConfig = bEndPartnerConfig
	case "azure":
		var azureConfig vxcPartnerConfigAzureModel
		azureDiags := aPartnerConfig.AzurePartnerConfig.As(ctx, azureConfig, basetypes.ObjectAsOptions{})
		if azureDiags.HasError() {
			resp.Diagnostics.Append(azureDiags...)
			return
		}
		bEndPartnerConfig := &megaport.VXCPartnerConfigAzure{
			ConnectType: azureConfig.ConnectType.ValueString(),
			ServiceKey:  azureConfig.ServiceKey.ValueString(),
		}
		bEndConfig.PartnerConfig = bEndPartnerConfig
	case "google":
		var googleConfig vxcPartnerConfigGoogleModel
		googleDiags := aPartnerConfig.GooglePartnerConfig.As(ctx, googleConfig, basetypes.ObjectAsOptions{})
		if googleDiags.HasError() {
			resp.Diagnostics.Append(googleDiags...)
			return
		}
		bEndPartnerConfig := &megaport.VXCPartnerConfigGoogle{
			ConnectType: googleConfig.ConnectType.ValueString(),
			PairingKey:  googleConfig.PairingKey.ValueString(),
		}
		bEndConfig.PartnerConfig = bEndPartnerConfig
	case "oracle":
		var oracleConfig vxcPartnerConfigOracleModel
		oracleDiags := aPartnerConfig.OraclePartnerConfig.As(ctx, oracleConfig, basetypes.ObjectAsOptions{})
		if oracleDiags.HasError() {
			resp.Diagnostics.Append(oracleDiags...)
			return
		}
		bEndPartnerConfig := &megaport.VXCPartnerConfigOracle{
			ConnectType:      oracleConfig.ConnectType.ValueString(),
			VirtualCircuitId: oracleConfig.VirtualCircuitId.ValueString(),
		}
		bEndConfig.PartnerConfig = bEndPartnerConfig
	default:
		resp.Diagnostics.AddError(
			"Error creating VXC",
			"Could not create VXC with name "+plan.Name.ValueString()+": Partner configuration not supported",
		)
		return
	}

	buyReq.AEndConfiguration = *aEndConfig
	buyReq.BEndConfiguration = *bEndConfig

	createdVXC, err := r.client.VXCService.BuyVXC(ctx, buyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating VXC",
			"Could not create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
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
	err = plan.fromAPIVXC(ctx, vxc)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VXC",
			"Could not read VXC with ID "+plan.UID.ValueString()+": "+err.Error(),
		)
		return
	}

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

	err = state.fromAPIVXC(ctx, vxc)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VXC",
			"Could not read VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

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

	aEndPlanDiags := plan.AEndConfiguration.As(ctx, aEndPlan, basetypes.ObjectAsOptions{})
	if aEndPlanDiags.HasError() {
		resp.Diagnostics.Append(aEndPlanDiags...)
		return
	}
	bEndPlanDiags := plan.BEndConfiguration.As(ctx, bEndPlan, basetypes.ObjectAsOptions{})
	if bEndPlanDiags.HasError() {
		resp.Diagnostics.Append(bEndPlanDiags...)
		return
	}

	aEndStateDiags := state.AEndConfiguration.As(ctx, aEndState, basetypes.ObjectAsOptions{})
	if aEndStateDiags.HasError() {
		resp.Diagnostics.Append(aEndStateDiags...)
		return
	}
	bEndStateDiags := state.BEndConfiguration.As(ctx, bEndState, basetypes.ObjectAsOptions{})
	if bEndStateDiags.HasError() {
		resp.Diagnostics.Append(bEndStateDiags...)
		return
	}

	if aEndPlan.VLAN.Equal(aEndState.VLAN) {
		aEndVlan = int(aEndPlan.VLAN.ValueInt64())
	}
	if bEndPlan.VLAN.Equal(bEndState.VLAN) {
		bEndVlan = int(bEndPlan.VLAN.ValueInt64())
	}
	if !aEndPlan.ProductUID.Equal(aEndState.ProductUID) {
		aEndProductUID = aEndPlan.ProductUID.ValueString()
	}
	if !bEndPlan.ProductUID.Equal(bEndState.ProductUID) {
		bEndProductUID = bEndPlan.ProductUID.ValueString()
	}
	if !plan.RateLimit.Equal(state.RateLimit) {
		rateLimit = int(plan.RateLimit.ValueInt64())
	}
	if !plan.CostCentre.Equal(state.CostCentre) {
		costCentre = plan.CostCentre.ValueString()
	}
	if !plan.Shutdown.Equal(state.Shutdown) {
		shutdown = plan.Shutdown.ValueBool()
	}
	if !plan.ContractTermMonths.Equal(state.ContractTermMonths) {
		term = int(plan.ContractTermMonths.ValueInt64())
	}

	updateReq := &megaport.UpdateVXCRequest{
		Name:           &name,
		AEndVLAN:       &aEndVlan,
		BEndVLAN:       &bEndVlan,
		AEndProductUID: &aEndProductUID,
		BEndProductUID: &bEndProductUID,
		CostCentre:     &costCentre,
		Shutdown:       &shutdown,
		RateLimit:      &rateLimit,
		Term:           &term,
	}

	_, err := r.client.VXCService.UpdateVXC(ctx, plan.ID.String(), updateReq)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating VXC",
			"Could not update VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
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

	err = state.fromAPIVXC(ctx, vxc)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VXC",
			"Could not read VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

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
	err := r.client.VXCService.DeleteVXC(ctx, state.UID.String(), &megaport.DeleteVXCRequest{
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
	resource.ImportStatePassthroughID(ctx, path.Root("uid"), req, resp)
}

func fromAPICSPConnection(ctx context.Context, c megaport.CSPConnectionConfig) (*types.Object, error) {
	switch provider := c.(type) {
	case *megaport.CSPConnectionAWS:
		awsModel := &cspConnectionAWSModel{
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
		awsObject, diags := types.ObjectValueFrom(ctx, cspConnectionAWSAttrs, awsModel)
		if diags.HasError() {
			return nil, errors.New("error creating object from CSPConnectionAWSModel")
		}
		return &awsObject, nil
	case *megaport.CSPConnectionAWSHC:
		awsHCModel := &cspConnectionAWSHCModel{
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
		bandwidthList, diags := types.ListValueFrom(ctx, types.Int64Type, bandwidths)
		if diags.HasError() {
			return nil, errors.New("error creating list from bandwidths")
		}
		awsHCModel.Bandwidths = bandwidthList
		awsHCObject, diags := types.ObjectValueFrom(ctx, cspConnectionAWSHCAttrs, awsHCModel)
		if diags.HasError() {
			return nil, errors.New("error creating object from CSPConnectionAWSHCModel")
		}
		return &awsHCObject, nil
	case *megaport.CSPConnectionAzure:
		azureModel := &cspConnectionAzureModel{
			ConnectType:  types.StringValue(provider.ConnectType),
			ResourceName: types.StringValue(provider.ResourceName),
			ResourceType: types.StringValue(provider.ResourceType),
			Bandwidth:    types.Int64Value(int64(provider.Bandwidth)),
			Managed:      types.BoolValue(provider.Managed),
			ServiceKey:   types.StringValue(provider.ServiceKey),
			VLAN:         types.Int64Value(int64(provider.VLAN)),
		}
		megaports := []types.Object{}
		for _, m := range provider.Megaports {
			megaportModel := &cspConnectionAzureMegaportModel{
				Port: types.Int64Value(int64(m.Port)),
				Type: types.StringValue(m.Type),
				VXC:  types.Int64Value(int64(m.VXC)),
			}
			megaportObject, diags := types.ObjectValueFrom(ctx, cspConnectionAzureMegaportAttrs, megaportModel)
			if diags.HasError() {
				return nil, errors.New("error creating object from CSPConnectionAzureMegaportModel")
			}
			megaports = append(megaports, megaportObject)
		}
		megaportsList, diags := types.ListValueFrom(ctx, types.ObjectType{}, megaports)
		if diags.HasError() {
			return nil, errors.New("error creating list from megaports")
		}
		azureModel.Megaports = megaportsList
		ports := []types.Object{}
		for _, p := range provider.Ports {
			portModel := &cspConnectionAzurePortModel{
				ServiceID: types.Int64Value(int64(p.ServiceID)),
				Type:      types.StringValue(p.Type),
			}
			vxcServiceIDs := []int64{}

			for _, v := range p.VXCServiceIDs {
				vxcServiceIDs = append(vxcServiceIDs, int64(v))
			}
			vxcServiceIDList, diags := types.ListValueFrom(ctx, types.Int64Type, vxcServiceIDs)
			if diags.HasError() {
				return nil, errors.New("error creating list from VXCServiceIDs")
			}
			portModel.VXCServiceIDs = vxcServiceIDList
			portObject, diags := types.ObjectValueFrom(ctx, cspConnectionAzurePortAttrs, portModel)
			if diags.HasError() {
				return nil, errors.New("error creating object from CSPConnectionAzurePortModel")
			}
			ports = append(ports, portObject)
		}
		portsList, diags := types.ListValueFrom(ctx, types.ObjectType{}, ports)
		if diags.HasError() {
			return nil, errors.New("error creating list from ports")
		}
		azureModel.Ports = portsList
		azureObject, diags := types.ObjectValueFrom(ctx, cspConnectionAzureAttrs, azureModel)
		if diags.HasError() {
			return nil, errors.New("error creating object from CSPConnectionAzureModel")
		}
		return &azureObject, nil
	case *megaport.CSPConnectionGoogle:
		googleModel := &cspConnectionGoogleModel{
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
		bandwidthList, diags := types.ListValueFrom(ctx, types.Int64Type, bandwidths)
		if diags.HasError() {
			return nil, errors.New("error creating list from bandwidths")
		}
		googleModel.Bandwidths = bandwidthList
		megaports := []types.Object{}
		for _, m := range provider.Megaports {
			megaportModel := &cspConnectionGoogleMegaportModel{
				Port: types.Int64Value(int64(m.Port)),
				VXC:  types.Int64Value(int64(m.VXC)),
			}
			megaportObject, diags := types.ObjectValueFrom(ctx, cspConnectionGoogleMegaportAttrs, megaportModel)
			if diags.HasError() {
				return nil, errors.New("error creating object from CSPConnectionGoogleMegaportModel")
			}
			megaports = append(megaports, megaportObject)
		}
		megaportsList, diags := types.ListValueFrom(ctx, types.ObjectType{}, megaports)
		if diags.HasError() {
			return nil, errors.New("error creating list from megaports")
		}
		googleModel.Megaports = megaportsList
		ports := []types.Object{}
		for _, p := range provider.Ports {
			portModel := &cspConnectionGooglePortModel{
				ServiceID: types.Int64Value(int64(p.ServiceID)),
			}
			vxcServiceIDs := []int64{}
			for _, v := range p.VXCServiceIDs {
				vxcServiceIDs = append(vxcServiceIDs, int64(v))
			}
			vxcServiceIDList, diags := types.ListValueFrom(ctx, types.Int64Type, vxcServiceIDs)
			if diags.HasError() {
				return nil, errors.New("error creating list from VXCServiceIDs")
			}
			portModel.VXCServiceIDs = vxcServiceIDList
			portObject, diags := types.ObjectValueFrom(ctx, cspConnectionGooglePortAttrs, portModel)
			if diags.HasError() {
				return nil, errors.New("error creating object from CSPConnectionGooglePortModel")
			}
			ports = append(ports, portObject)
		}
		portsList, diags := types.ListValueFrom(ctx, types.ObjectType{}, ports)
		if diags.HasError() {
			return nil, errors.New("error creating list from ports")
		}
		googleModel.Ports = portsList
		googleObject, diags := types.ObjectValueFrom(ctx, cspConnectionGoogleAttrs, googleModel)
		if diags.HasError() {
			return nil, errors.New("error creating object from CSPConnectionGoogleModel")
		}
		return &googleObject, nil
	case *megaport.CSPConnectionVirtualRouter:
		virtualRouterModel := &cspConnectionVirtualRouterModel{
			ConnectType:       types.StringValue(provider.ConnectType),
			ResourceName:      types.StringValue(provider.ResourceName),
			ResourceType:      types.StringValue(provider.ResourceType),
			VLAN:              types.Int64Value(int64(provider.VLAN)),
			VirtualRouterName: types.StringValue(provider.VirtualRouterName),
		}
		interfaces := []types.Object{}
		for _, i := range provider.Interfaces {
			interfaceModel := &cspConnectionVirtualRouterInterfaceModel{}
			ipAddresses := []string{}
			ipAddresses = append(ipAddresses, i.IPAddresses...)
			ipAddressesList, diags := types.ListValueFrom(ctx, types.StringType, ipAddresses)
			if diags.HasError() {
				return nil, errors.New("error creating list from ipAddresses")
			}
			interfaceModel.IPAddresses = ipAddressesList
			interfaceObject, diags := types.ObjectValueFrom(ctx, cspConnectionVirtualRouterInterfaceAttrs, interfaceModel)
			if diags.HasError() {
				return nil, errors.New("error creating object from CSPConnectionVirtualRouterInterfaceModel")
			}
			interfaces = append(interfaces, interfaceObject)
		}
		interfacesList, diags := types.ListValueFrom(ctx, types.ObjectType{}, interfaces)
		if diags.HasError() {
			return nil, errors.New("error creating list from interfaces")
		}
		virtualRouterModel.Interfaces = interfacesList
		virtualRouterObject, diags := types.ObjectValueFrom(ctx, cspConnectionVirtualRouterAttrs, virtualRouterModel)
		if diags.HasError() {
			return nil, errors.New("error creating object from CSPConnectionVirtualRouterModel")
		}
		return &virtualRouterObject, nil
	case *megaport.CSPConnectionTransit:
		transitModel := &cspConnectionTransit{
			ConnectType:        types.StringValue(provider.ConnectType),
			ResourceName:       types.StringValue(provider.ResourceName),
			ResourceType:       types.StringValue(provider.ResourceType),
			CustomerIP4Address: types.StringValue(provider.CustomerIP4Address),
			CustomerIP6Network: types.StringValue(provider.CustomerIP6Network),
			IPv4GatewayAddress: types.StringValue(provider.IPv4GatewayAddress),
			IPv6GatewayAddress: types.StringValue(provider.IPv6GatewayAddress),
		}
		transitObject, diags := types.ObjectValueFrom(ctx, cspConnectionTransitAttrs, transitModel)
		if diags.HasError() {
			return nil, errors.New("error creating object from CSPConnectionTransitModel")
		}
		return &transitObject, nil
	}
	return nil, errors.New("unknown CSPConnection type")
}
