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
		"account_id":           types.StringType,
		"amazon_address":       types.StringType,
		"asn":                  types.Int64Type,
		"customer_asn":         types.Int64Type,
		"auth_key":             types.StringType,
		"customer_address":     types.StringType,
		"customer_ip_address":  types.StringType,
		"provider_ip_address":  types.StringType,
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
		"ibm_config":           types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigIbmAttrs),
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

	vxcPartnerConfigIbmAttrs = map[string]attr.Type{
		"account_id":          types.StringType,
		"customer_asn":        types.Int64Type,
		"name":                types.StringType,
		"customer_ip_address": types.StringType,
		"provider_ip_address": types.StringType,
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
		"local_asn":             types.Int64Type,
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
		"ip_mtu":           types.Int64Type,
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
		"local_asn":             types.Int64Type,
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

	ResourceTags types.Map `tfsdk:"resource_tags"`
}

type cspConnectionModel struct {
	ConnectType        types.String `tfsdk:"connect_type"`
	ResourceName       types.String `tfsdk:"resource_name"`
	ResourceType       types.String `tfsdk:"resource_type"`
	VLAN               types.Int64  `tfsdk:"vlan"`
	Account            types.String `tfsdk:"account"`
	AmazonAddress      types.String `tfsdk:"amazon_address"`
	AccountID          types.String `tfsdk:"account_id"`
	CustomerASN        types.Int64  `tfsdk:"customer_asn"`
	ASN                types.Int64  `tfsdk:"asn"`
	AuthKey            types.String `tfsdk:"auth_key"`
	CustomerAddress    types.String `tfsdk:"customer_address"`
	CustomerIPAddress  types.String `tfsdk:"customer_ip_address"`
	ProviderIPAddress  types.String `tfsdk:"provider_ip_address"`
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
	IBMPartnerConfig     types.Object `tfsdk:"ibm_config"`
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

type vxcPartnerConfigIbmModel struct {
	vxcPartnerConfig
	AccountID         types.String `tfsdk:"account_id"`          // Customer's IBM Acount ID.  32 Hexadecimal Characters. REQUIRED
	CustomerASN       types.Int64  `tfsdk:"customer_asn"`        // Customer's ASN. Valid ranges: 1-64495, 64999, 131072-4199999999, 4201000000-4201064511. Required unless the connection at the other end of the VXC is an MCR.
	Name              types.String `tfsdk:"name"`                // Description of this connection for identification purposes. Max 100 characters from 0-9 a-z A-Z / - _ , Defaults to "MEGAPORT".
	CustomerIPAddress types.String `tfsdk:"customer_ip_address"` // IPv4 network address including subnet mask. Default is /30 assigned from 169.254.0.0/16.
	ProviderIPAddress types.String `tfsdk:"provider_ip_address"` // IPv4 network address including subnet mask.
}

// vxcPartnerConfigInterfaceModel maps the partner configuration schema data for an interface.
type vxcPartnerConfigInterfaceModel struct {
	IpMtu          types.Int64  `tfsdk:"ip_mtu"`
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
	LocalAsn           types.Int64  `tfsdk:"local_asn"`
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
				Description: "The term of the contract in months: valid values are 1, 12, 24, and 36. To set the product to a month-to-month contract with no minimum term, set the value to 1.",
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
						"customer_asn": schema.Int64Attribute{
							Description: "The customer ASN of the CSP connection.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
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
						"account_id": schema.StringAttribute{
							Description: "The account ID of the CSP connection.",
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
						"provider_ip_address": schema.StringAttribute{
							Description: "The provider IP address of the CSP connection.",
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
			"resource_tags": schema.MapAttribute{
				Description: "The resource tags associated with the product.",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
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
						Description: "The Product UID requested by the user for the A-End configuration. Note: For cloud provider connections, the actual Product UID may differ from the requested UID due to Megaport's automatic port assignment for partner ports. This is expected behavior and ensures proper connectivity.",
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
						Computed:    true,
						Validators:  []validator.Int64{int64validator.Between(-1, 4093), int64validator.NoneOf(1)},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"vlan": schema.Int64Attribute{
						Description: "The current VLAN of the A-End configuration. May be different from the A-End ordered VLAN if the system allocated a different VLAN. Values can range from 2 to 4093. If the A-End ordered_vlan was set to 0, the Megaport system allocated a valid VLAN. If the A-End ordered_vlan was set to -1, the Megaport system will automatically set this value to null.",
						Computed:    true,
					},
					"inner_vlan": schema.Int64Attribute{
						Description: "The inner VLAN of the A-End configuration. If the A-End ordered_vlan is untagged and set as -1, this field cannot be set by the API, as the VLAN of the A-End is designated as untagged. Note: Setting inner_vlan to 0 for auto-assignment is not currently supported by the provider. This is a known limitation that will be resolved in a future release.",
						Optional:    true,
						Computed:    true,
						Validators:  []validator.Int64{int64validator.Between(-1, 4093), int64validator.NoneOf(1), int64validator.NoneOf(0)},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"vnic_index": schema.Int64Attribute{
						Description: "The network interface index of the A-End configuration. Required for MVE connections.",
						Optional:    true,
						Computed:    true,
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
						Description: "The Product UID requested by the user for the B-End configuration. Note: For cloud provider connections, the actual Product UID may differ from the requested UID due to Megaport's automatic port assignment for partner ports. This is expected behavior and ensures proper connectivity.",
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
						Computed:    true,
						Validators:  []validator.Int64{int64validator.Between(-1, 4093), int64validator.NoneOf(1)},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"vlan": schema.Int64Attribute{
						Description: "The current VLAN of the B-End configuration. May be different from the B-End ordered VLAN if the system allocated a different VLAN. Values can range from 2 to 4093. If the B-End ordered_vlan was set to 0, the Megaport system allocated a valid VLAN. If the B-End ordered_vlan was set to -1, the Megaport system will automatically set this value to null.",
						Computed:    true,
					},
					"inner_vlan": schema.Int64Attribute{
						Description: "The inner VLAN of the B-End configuration. If the B-End ordered_vlan is untagged and set as -1, this field cannot be set by the API, as the VLAN of the B-End is designated as untagged. Note: Setting inner_vlan to 0 for auto-assignment is not currently supported by the provider. This is a known limitation that will be resolved in a future release.",
						Optional:    true,
						Computed:    true,
						Validators:  []validator.Int64{int64validator.Between(-1, 4093), int64validator.NoneOf(1), int64validator.NoneOf(0)},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"vnic_index": schema.Int64Attribute{
						Description: "The network interface index of the B-End configuration. Required for MVE connections.",
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
							stringvalidator.OneOf("aws", "azure", "google", "oracle", "ibm", "vrouter", "transit", "a-end"),
						},
					},
					"aws_config":           awsPartnerConfigSchema,
					"azure_config":         azurePartnerConfigSchema,
					"google_config":        googlePartnerConfigSchema,
					"ibm_config":           ibmPartnerConfigSchema,
					"oracle_config":        oraclePartnerConfigSchema,
					"vrouter_config":       vrouterPartnerConfigSchema,
					"partner_a_end_config": aEndPartnerConfigSchema,
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
							stringvalidator.OneOf("aws", "azure", "google", "oracle", "ibm", "transit", "vrouter"),
						},
					},
					"aws_config":           awsPartnerConfigSchema,
					"azure_config":         azurePartnerConfigSchema,
					"google_config":        googlePartnerConfigSchema,
					"ibm_config":           ibmPartnerConfigSchema,
					"oracle_config":        oraclePartnerConfigSchema,
					"vrouter_config":       vrouterPartnerConfigSchema,
					"partner_a_end_config": aEndPartnerConfigSchema,
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

	var serviceKeyBEndUID string
	if !plan.ServiceKey.IsNull() && !plan.ServiceKey.IsUnknown() {
		// If a service key is provided, we should look up the product UID pertaining to that service key and use that B-End Product UID
		serviceKeyRes, err := r.client.ServiceKeyService.GetServiceKey(ctx, plan.ServiceKey.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating VXC",
				"Could not create VXC with name "+plan.Name.ValueString()+": looking up Service Key failed: "+err.Error(),
			)
			return
		}
		if serviceKeyRes.ProductUID == "" {
			resp.Diagnostics.AddError(
				"Error creating VXC",
				"Could not create VXC with name "+plan.Name.ValueString()+": the provided Service Key is not associated with a Product",
			)
			return
		}
		serviceKeyBEndUID = serviceKeyRes.ProductUID
	}

	if !plan.Shutdown.IsNull() {
		buyReq.Shutdown = plan.Shutdown.ValueBool()
	}

	if !plan.ResourceTags.IsNull() {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		buyReq.ResourceTags = tagMap
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

	// Check product type - if MVE, require VNIC Index
	productType, _ := r.client.ProductService.GetProductType(ctx, a.RequestedProductUID.ValueString())
	if productType == megaport.PRODUCT_MVE {
		if a.NetworkInterfaceIndex.IsNull() && a.NetworkInterfaceIndex.IsUnknown() {
			resp.Diagnostics.AddError(
				"Error creating VXC",
				"Could not create VXC with name "+plan.Name.ValueString()+": Network Interface Index is required for MVE products",
			)
			return
		}
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
			if awsConfig.ConnectType.ValueString() == "AWS" {
				// Only allow type of "public", "private", or "transit" for AWS VIFs
				if awsConfig.Type.ValueString() != "public" && awsConfig.Type.ValueString() != "private" && awsConfig.Type.ValueString() != "transit" {
					resp.Diagnostics.AddError(
						"Error creating VXC",
						"Could not create VXC with name "+plan.Name.ValueString()+": AWS Connect Type must be public, private, or transit",
					)
					return
				}
			}
			awsDiags, partnerConfig, partnerConfigObj := createAWSPartnerConfig(ctx, awsConfig)
			if awsDiags.HasError() {
				resp.Diagnostics.Append(awsDiags...)
				return
			}
			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = partnerConfig
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
			azureDiags, azurePartnerConfig, partnerConfigObj := createAzurePartnerConfig(ctx, azureConfig)
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

			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = azurePartnerConfig
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
			googleDiags, googlePartnerConfig, partnerConfigObj := createGooglePartnerConfig(ctx, googleConfig)
			if googleDiags.HasError() {
				resp.Diagnostics.Append(googleDiags...)
				return
			}
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

			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = googlePartnerConfig
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
			oracleDiags, oraclePartnerConfig, partnerConfigObj := createOraclePartnerConfig(ctx, oracleConfig)
			if oracleDiags.HasError() {
				resp.Diagnostics.Append(oracleDiags...)
				return
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
			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = oraclePartnerConfig
		case "ibm":
			if aPartnerConfig.IBMPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": IBM Partner configuration is required",
				)
				return
			}
			var ibmConfig vxcPartnerConfigIbmModel
			ibmDiags := aPartnerConfig.IBMPartnerConfig.As(ctx, &ibmConfig, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(ibmDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			ibmDiags, ibmPartnerConfig, partnerConfigObj := createIBMPartnerConfig(ctx, ibmConfig)
			if ibmDiags.HasError() {
				resp.Diagnostics.Append(ibmDiags...)
				return
			}
			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = ibmPartnerConfig
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
			prefixFilterList, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, a.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			vrouterDiags, vrouterMegaportConfig, partnerConfigObj := createVrouterPartnerConfig(ctx, partnerConfigAEnd, prefixFilterList)
			if vrouterDiags.HasError() {
				resp.Diagnostics.Append(vrouterDiags...)
				return
			}
			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = vrouterMegaportConfig
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
			prefixFilterList, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, a.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}
			aEndDiags, aEndMegaportConfig, partnerConfigObj := createAEndPartnerConfig(ctx, partnerConfigAEnd, prefixFilterList)
			if aEndDiags.HasError() {
				resp.Diagnostics.Append(aEndDiags...)
				return
			}

			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = aEndMegaportConfig
		case "transit":
			transitDiags, transitPartnerConfig, partnerConfigObj := createTransitPartnerConfig(ctx)
			if transitDiags.HasError() {
				resp.Diagnostics.Append(transitDiags...)
				return
			}
			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = transitPartnerConfig
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
	if serviceKeyBEndUID != "" {
		// If B End Requested Product UID was provided and it differs from the Service Key Product UID, warn that it is being overridden
		if b.RequestedProductUID.ValueString() != "" && b.RequestedProductUID.ValueString() != serviceKeyBEndUID {
			resp.Diagnostics.AddWarning(
				"Overriding B-End Product UID",
				"Overriding the requested B-End Product UID of "+b.RequestedProductUID.ValueString()+" with "+serviceKeyBEndUID+" based on the provided Service Key.",
			)
		}
		bEndConfig.ProductUID = serviceKeyBEndUID
	}
	if !b.OrderedVLAN.IsNull() {
		bEndConfig.VLAN = int(b.OrderedVLAN.ValueInt64())
	} else {
		bEndConfig.VLAN = 0
	}

	// Check product type - if MVE, require VNIC Index
	productType, _ = r.client.ProductService.GetProductType(ctx, b.RequestedProductUID.ValueString())
	if productType == megaport.PRODUCT_MVE {
		if b.NetworkInterfaceIndex.IsNull() && b.NetworkInterfaceIndex.IsUnknown() {
			resp.Diagnostics.AddError(
				"Error creating VXC",
				"Could not create VXC with name "+plan.Name.ValueString()+": Network Interface Index is required for MVE products",
			)
			return
		}
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
			if awsConfig.ConnectType.ValueString() == "AWS" {
				// Only allow type of "public", "private", or "transit" for AWS VIFs
				if awsConfig.Type.ValueString() != "public" && awsConfig.Type.ValueString() != "private" && awsConfig.Type.ValueString() != "transit" {
					resp.Diagnostics.AddError(
						"Error creating VXC",
						"Could not create VXC with name "+plan.Name.ValueString()+": AWS Connect Type must be public, private, or transit",
					)
					return
				}
			}
			awsDiags, partnerConfig, partnerConfigObj := createAWSPartnerConfig(ctx, awsConfig)
			if awsDiags.HasError() {
				resp.Diagnostics.Append(awsDiags...)
				return
			}
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = partnerConfig
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

			azureDiags, azurePartnerConfig, partnerConfigObj := createAzurePartnerConfig(ctx, azureConfig)
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
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = azurePartnerConfig
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

			googleDiags, googlePartnerConfig, partnerConfigObj := createGooglePartnerConfig(ctx, googleConfig)
			if googleDiags.HasError() {
				resp.Diagnostics.Append(googleDiags...)
				return
			}

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

			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = googlePartnerConfig
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
			oracleDiags, oraclePartnerConfig, partnerConfigObj := createOraclePartnerConfig(ctx, oracleConfig)
			if oracleDiags.HasError() {
				resp.Diagnostics.Append(oracleDiags...)
				return
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

			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = oraclePartnerConfig
		case "ibm":
			if bPartnerConfig.IBMPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": IBM Partner configuration is required",
				)
				return
			}
			var ibmConfig vxcPartnerConfigIbmModel
			ibmDiags := bPartnerConfig.IBMPartnerConfig.As(ctx, &ibmConfig, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(ibmDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			ibmDiags, ibmPartnerConfig, partnerConfigObj := createIBMPartnerConfig(ctx, ibmConfig)
			if ibmDiags.HasError() {
				resp.Diagnostics.Append(ibmDiags...)
				return
			}
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = ibmPartnerConfig
		case "transit":
			transitDiags, transitPartnerConfig, partnerConfigObj := createTransitPartnerConfig(ctx)
			if transitDiags.HasError() {
				resp.Diagnostics.Append(transitDiags...)
				return
			}
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = transitPartnerConfig
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
			prefixFilterList, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, b.RequestedProductUID.ValueString())
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
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouterConfigObj,
				IBMPartnerConfig:     ibmPartner,
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

	tags, err := r.client.VXCService.ListVXCResourceTags(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading tags for newly created VXC",
			"Could not read tags for newly created VXC with ID "+createdID+": "+err.Error(),
		)
		return
	}

	// update the plan with the VXC info
	apiDiags := plan.fromAPIVXC(ctx, vxc, tags)
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

	// Get tags
	tags, err := r.client.VXCService.ListVXCResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading tags for VXC",
			"Could not read tags for VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	apiDiags := state.fromAPIVXC(ctx, vxc, tags)
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

	var aEndPartnerChange, bEndPartnerChange bool

	// If Imported, AEndPartnerConfig will be null. Set the partner config to the existing one in the plan.
	if !plan.AEndPartnerConfig.Equal(state.AEndPartnerConfig) {
		aEndPartnerChange = true
	}
	if state.AEndPartnerConfig.IsNull() {
		state.AEndPartnerConfig = plan.AEndPartnerConfig
	}
	if !plan.BEndPartnerConfig.Equal(state.BEndPartnerConfig) {
		bEndPartnerChange = true
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

	aEndProductType, _ := r.client.ProductService.GetProductType(ctx, aEndPlan.RequestedProductUID.ValueString())
	bEndProductType, _ := r.client.ProductService.GetProductType(ctx, bEndPlan.RequestedProductUID.ValueString())

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

	// Check VNIC index for A End
	if !aEndPlan.NetworkInterfaceIndex.IsUnknown() && !aEndPlan.NetworkInterfaceIndex.IsNull() && !aEndPlan.NetworkInterfaceIndex.Equal(aEndState.NetworkInterfaceIndex) {
		updateReq.AVnicIndex = megaport.PtrTo(int(aEndPlan.NetworkInterfaceIndex.ValueInt64()))
	} else if aEndProductType == megaport.PRODUCT_MVE && aEndPlan.NetworkInterfaceIndex.IsNull() {
		resp.Diagnostics.AddError(
			"Error updating VXC",
			"Could not update VXC with name "+plan.Name.ValueString()+": Network Interface Index is required for MVE products - A End is MVE. Please specify which network interface on the MVE device this VXC should connect to.",
		)
		return
	} else {
		updateReq.AVnicIndex = megaport.PtrTo(int(aEndState.NetworkInterfaceIndex.ValueInt64()))
	}

	// If Ordered VLAN is different from actual VLAN, attempt to change it to the ordered VLAN value.
	if !bEndPlan.OrderedVLAN.IsUnknown() && !bEndPlan.OrderedVLAN.IsNull() && !bEndPlan.OrderedVLAN.Equal(bEndState.VLAN) {
		updateReq.BEndVLAN = megaport.PtrTo(int(bEndPlan.OrderedVLAN.ValueInt64()))
	}
	bEndState.OrderedVLAN = bEndPlan.OrderedVLAN

	// Prevent setting inner_vlan to 0 during updates (auto-assignment only works on creation)
	if !aEndPlan.InnerVLAN.IsUnknown() && !aEndPlan.InnerVLAN.IsNull() && !aEndPlan.InnerVLAN.Equal(aEndState.InnerVLAN) {
		updateReq.AEndInnerVLAN = megaport.PtrTo(int(aEndPlan.InnerVLAN.ValueInt64()))
	}
	aEndState.InnerVLAN = aEndPlan.InnerVLAN

	// Similarly add for B-End
	if !bEndPlan.InnerVLAN.IsUnknown() && !bEndPlan.InnerVLAN.IsNull() && !bEndPlan.InnerVLAN.Equal(bEndState.InnerVLAN) {
		updateReq.BEndInnerVLAN = megaport.PtrTo(int(bEndPlan.InnerVLAN.ValueInt64()))
	}
	bEndState.InnerVLAN = bEndPlan.InnerVLAN

	// Check VNIC index for B End
	if !bEndPlan.NetworkInterfaceIndex.IsUnknown() && !bEndPlan.NetworkInterfaceIndex.IsNull() && !bEndPlan.NetworkInterfaceIndex.Equal(bEndState.NetworkInterfaceIndex) {
		updateReq.BVnicIndex = megaport.PtrTo(int(bEndPlan.NetworkInterfaceIndex.ValueInt64()))
	} else if bEndProductType == megaport.PRODUCT_MVE && bEndPlan.NetworkInterfaceIndex.IsNull() {
		resp.Diagnostics.AddError(
			"Error updating VXC",
			"Could not update VXC with name "+plan.Name.ValueString()+": Network Interface Index is required for MVE products - B End is MVE. Please specify which network interface on the MVE device this VXC should connect to.",
		)
		return
	} else {
		updateReq.BVnicIndex = megaport.PtrTo(int(bEndState.NetworkInterfaceIndex.ValueInt64()))
	}

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

	if !plan.AEndPartnerConfig.IsNull() && aEndPartnerChange && !aEndCSP {
		aPartnerConfig := aEndPartnerPlan
		switch aEndPartnerPlan.Partner.ValueString() {
		case "transit":
			transitDiags, transitPartnerConfig, partnerConfigObj := createTransitPartnerConfig(ctx)
			if transitDiags.HasError() {
				resp.Diagnostics.Append(transitDiags...)
				return
			}
			state.AEndPartnerConfig = partnerConfigObj
			updateReq.AEndPartnerConfig = transitPartnerConfig
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
			prefixFilterList, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, aEndPlan.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}
			aEndDiags, aEndMegaportConfig, partnerConfigObj := createAEndPartnerConfig(ctx, partnerConfigAEnd, prefixFilterList)
			if aEndDiags.HasError() {
				resp.Diagnostics.Append(aEndDiags...)
				return
			}
			state.AEndPartnerConfig = partnerConfigObj
			updateReq.AEndPartnerConfig = aEndMegaportConfig
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
			prefixFilterList, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, aEndState.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}
			vrouterDiags, vrouterPartnerConfig, partnerConfigObj := createVrouterPartnerConfig(ctx, partnerConfigAEnd, prefixFilterList)
			if vrouterDiags.HasError() {
				resp.Diagnostics.Append(vrouterDiags...)
				return
			}
			state.AEndPartnerConfig = partnerConfigObj
			updateReq.AEndPartnerConfig = vrouterPartnerConfig
		default:
			resp.Diagnostics.AddError(
				"Error Updating VXC",
				"Could not update VXC with ID "+state.UID.ValueString()+": Partner configuration not supported",
			)
			return
		}
	}

	if !plan.BEndPartnerConfig.IsNull() && bEndPartnerChange && !bEndCSP {
		switch bEndPartnerPlan.Partner.ValueString() {
		case "transit":
			transitDiags, transitPartnerConfig, partnerConfigObj := createTransitPartnerConfig(ctx)
			if transitDiags.HasError() {
				resp.Diagnostics.Append(transitDiags...)
				return
			}
			state.BEndPartnerConfig = partnerConfigObj
			updateReq.AEndPartnerConfig = transitPartnerConfig
		case "vrouter":
			if bEndPartnerPlan.VrouterPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Virtual router configuration is required",
				)
				return
			}
			var vrouterModel vxcPartnerConfigVrouterModel
			bEndDiags := bEndPartnerPlan.VrouterPartnerConfig.As(ctx, &vrouterModel, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(bEndDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			prefixFilterList, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, bEndState.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			vrouterDiags, vrouterPartnerConfig, partnerConfigObj := createVrouterPartnerConfig(ctx, vrouterModel, prefixFilterList)
			if vrouterDiags.HasError() {
				resp.Diagnostics.Append(vrouterDiags...)
				return
			}

			state.BEndPartnerConfig = partnerConfigObj
			updateReq.BEndPartnerConfig = vrouterPartnerConfig
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

	if !plan.ResourceTags.Equal(state.ResourceTags) {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := r.client.VXCService.UpdateVXCResourceTags(ctx, state.UID.ValueString(), tagMap)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating tags for VXC",
				"Could not update tags for VXC with ID "+state.UID.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	// Get resource tags
	tags, err := r.client.VXCService.ListVXCResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VXC Tags",
			"Could not read VXC tags with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	apiDiags := state.fromAPIVXC(ctx, vxc, tags)
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
			CSPName:      types.StringValue(provider.CSPName),
			Bandwidth:    types.Int64Value(int64(provider.Bandwidth)),
		}

		// Set VirtualCircuitId if available
		if provider.VirtualCircuitId != "" {
			oracleModel.ConnectionID = types.StringValue(provider.VirtualCircuitId)
		} else {
			oracleModel.ConnectionID = types.StringNull()
		}

		// Set null values for fields that don't apply to Oracle connections
		oracleModel.Bandwidths = types.ListNull(types.Int64Type)
		oracleModel.IPAddresses = types.ListNull(types.StringType)

		// Convert the model to a Terraform Object
		oracleObj, oracleObjDiags := types.ObjectValueFrom(ctx, cspConnectionFullAttrs, oracleModel)
		apiDiags = append(apiDiags, oracleObjDiags...)
		return oracleObj, apiDiags
	case megaport.CSPConnectionIBM:
		ibmModel := &cspConnectionModel{
			ConnectType:       types.StringValue(provider.ConnectType),
			ResourceName:      types.StringValue(provider.ResourceName),
			ResourceType:      types.StringValue(provider.ResourceType),
			AccountID:         types.StringValue(provider.AccountID),
			CustomerASN:       types.Int64Value(int64(provider.CustomerASN)),
			CustomerIPAddress: types.StringValue(provider.CustomerIPAddress),
			ProviderIPAddress: types.StringValue(provider.ProviderIPAddress),
			Bandwidth:         types.Int64Value(int64(provider.Bandwidth)),
			CSPName:           types.StringValue(provider.CSPName),
		}
		bandwidths := []int64{}
		for _, bandwidth := range provider.Bandwidths {
			bandwidths = append(bandwidths, int64(bandwidth))
		}
		bandwidthList, bandwidthListDiags := types.ListValueFrom(ctx, types.Int64Type, bandwidths)
		apiDiags = append(apiDiags, bandwidthListDiags...)
		ibmModel.Bandwidths = bandwidthList
		ibmModel.IPAddresses = types.ListNull(types.StringType)
		ibmObject, ibmObjectDiags := types.ObjectValueFrom(ctx, cspConnectionFullAttrs, ibmModel)
		apiDiags = append(apiDiags, ibmObjectDiags...)
		return ibmObject, apiDiags
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
				if aEndPlanConfig.RequestedProductUID.IsNull() {
					aEndStateConfig.RequestedProductUID = aEndStateConfig.CurrentProductUID
					aEndPlanConfig.RequestedProductUID = aEndStateConfig.CurrentProductUID
				} else {
					aEndStateConfig.RequestedProductUID = aEndPlanConfig.RequestedProductUID
				}
			} else if aEndCSP {
				if !aEndPlanConfig.RequestedProductUID.IsNull() && !aEndPlanConfig.RequestedProductUID.Equal(aEndStateConfig.RequestedProductUID) {
					diags.AddWarning(
						"Cloud provider port mapping detected",
						fmt.Sprintf("Different A-End Product UIDs detected for cloud provider endpoint: requested=%s, actual=%s. This is normal - Megaport automatically manages cloud connection port assignments. Your configuration remains unchanged while the connection uses the provider-assigned Product UID. No action needed.",
							aEndPlanConfig.RequestedProductUID.ValueString(),
							aEndStateConfig.CurrentProductUID.ValueString()),
					)
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
				if bEndPlanConfig.RequestedProductUID.IsNull() {
					bEndStateConfig.RequestedProductUID = bEndStateConfig.CurrentProductUID
					bEndPlanConfig.RequestedProductUID = bEndStateConfig.CurrentProductUID
				} else {
					bEndStateConfig.RequestedProductUID = bEndPlanConfig.RequestedProductUID
				}
			} else if bEndCSP {
				if !bEndPlanConfig.RequestedProductUID.IsNull() && !bEndPlanConfig.RequestedProductUID.Equal(bEndStateConfig.CurrentProductUID) {
					diags.AddWarning(
						"Cloud provider port mapping detected",
						fmt.Sprintf("Different B-End Product UIDs detected for cloud provider endpoint: requested=%s, actual=%s. This is normal - Megaport automatically manages cloud connection port assignments. Your configuration remains unchanged while the connection uses the provider-assigned Product UID. No action needed.",
							bEndPlanConfig.RequestedProductUID.ValueString(),
							bEndStateConfig.CurrentProductUID.ValueString()),
					)
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
