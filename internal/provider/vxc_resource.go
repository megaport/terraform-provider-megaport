package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &vxcResource{}
	_ resource.ResourceWithConfigure   = &vxcResource{}
	_ resource.ResourceWithImportState = &vxcResource{}
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

	ContractTermMonths types.Int64                   `tfsdk:"contract_term_months"`
	CompanyUID         types.String                  `tfsdk:"company_uid"`
	CompanyName        types.String                  `tfsdk:"company_name"`
	Locked             types.Bool                    `tfsdk:"locked"`
	AdminLocked        types.Bool                    `tfsdk:"admin_locked"`
	AttributeTags      map[types.String]types.String `tfsdk:"attribute_tags"`
	Cancelable         types.Bool                    `tfsdk:"cancelable"`

	LiveDate          types.String `tfsdk:"live_date"`
	CreateDate        types.String `tfsdk:"create_date"`
	ContractStartDate types.String `tfsdk:"contract_start_date"`
	ContractEndDate   types.String `tfsdk:"contract_end_date"`

	PortUID           types.String              `tfsdk:"port_uid"`
	AEndConfiguration *vxcEndConfigurationModel `tfsdk:"a_end"`
	BEndConfiguration *vxcEndConfigurationModel `tfsdk:"b_end"`

	Resources   *vxcResourcesModel `tfsdk:"resources"`
	VXCApproval *vxcApprovalModel  `tfsdk:"vxc_approval"`
}

// vxcResourcesModel represents the resources associated with a VXC.
type vxcResourcesModel struct {
	Interface     []*portInterfaceModel  `tfsdk:"interface"`
	VirtualRouter *virtualRouterModel    `tfsdk:"virtual_router"`
	VLL           *vllConfigModel        `tfsdk:"vll"`
	CSPConnection *vxcCSPConnectionModel `tfsdk:"csp_connection"`
}

// vxcCSPConnectionModel represents the CSP connection schema data.
type vxcCSPConnectionModel struct {
	CSPConnections []cspConnection
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
	ConnectType  types.String  `tfsdk:"connect_type"`
	ResourceName types.String  `tfsdk:"resource_name"`
	ResourceType types.String  `tfsdk:"resource_type"`
	Bandwidth    types.Int64   `tfsdk:"bandwidth"`
	Name         types.String  `tfsdk:"name"`
	OwnerAccount types.String  `tfsdk:"owner_account"`
	Bandwidths   []types.Int64 `tfsdk:"bandwidths"`
	ConnectionID types.String  `tfsdk:"connection_id"`
}

// cspConnectionAzureModel represents the configuration of a CSP connection for Azure ExpressRoute.
type cspConnectionAzureModel struct {
	cspConnection
	ConnectType  types.String                       `tfsdk:"connect_type"`
	ResourceName types.String                       `tfsdk:"resource_name"`
	ResourceType types.String                       `tfsdk:"resource_type"`
	Bandwidth    types.Int64                        `tfsdk:"bandwidth"`
	Managed      types.Bool                         `tfsdk:"managed"`
	Megaports    []*cspConnectionAzureMegaportModel `tfsdk:"megaports"`
	Ports        []*cspConnectionAzurePortModel     `tfsdk:"ports"`
	ServiceKey   types.String                       `tfsdk:"service_key"`
	VLAN         types.Int64                        `tfsdk:"vlan"`
}

// CSPConnectionAzureMegaport represents the configuration of a CSP connection for Azure ExpressRoute megaport.
type cspConnectionAzureMegaportModel struct {
	Port types.Int64  `tfsdk:"port"`
	Type types.String `tfsdk:"type"`
	VXC  types.Int64  `tfsdk:"vxc,omitempty"`
}

// cspConnectionAzurePortModel represents the configuration of a CSP connection for Azure ExpressRoute port.
type cspConnectionAzurePortModel struct {
	ServiceID     types.Int64   `tfsdk:"service_id"`
	Type          types.String  `tfsdk:"type"`
	VXCServiceIDs []types.Int64 `tfsdk:"vxc_service_ids"`
}

// cspConnectionGoogleModel represents the configuration of a CSP connection for Google Cloud Interconnect.
type cspConnectionGoogleModel struct {
	cspConnection
	Bandwidth    types.Int64                         `tfsdk:"bandwidth"`
	ConnectType  types.String                        `tfsdk:"connect_type"`
	ResourceName types.String                        `tfsdk:"resource_name"`
	ResourceType types.String                        `tfsdk:"resource_type"`
	Bandwidths   []types.Int64                       `tfsdk:"bandwidths"`
	Megaports    []*cspConnectionGoogleMegaportModel `tfsdk:"megaports"`
	Ports        []*cspConnectionGooglePortModel     `tfsdk:"ports"`
	CSPName      types.String                        `tfsdk:"csp_name"`
	PairingKey   types.String                        `tfsdk:"pairing_key"`
}

// cspConnectionGoogleMegaportModel represents the configuration of a CSP connection for Google Cloud Interconnect megaport.
type cspConnectionGoogleMegaportModel struct {
	Port types.Int64 `tfsdk:"port"`
	VXC  types.Int64 `tfsdk:"vxc"`
}

// cspConnectionGooglePortModel represents the configuration of a CSP connection for Google Cloud Interconnect port.
type cspConnectionGooglePortModel struct {
	ServiceID     types.Int64   `tfsdk:"service_id"`
	VXCServiceIDs []types.Int64 `tfsdk:"vxc_service_ids"`
}

// cspConnectionVirtualRouterModel represents the configuration of a CSP connection for Virtual Router.
type cspConnectionVirtualRouterModel struct {
	cspConnection
	ConnectType       types.String                                `tfsdk:"connect_type"`
	ResourceName      types.String                                `tfsdk:"resource_name"`
	ResourceType      types.String                                `tfsdk:"resource_type"`
	VLAN              types.Int64                                 `tfsdk:"vlan"`
	Interfaces        []*cspConnectionVirtualRouterInterfaceModel `tfsdk:"interfaces"`
	IPAddresses       []types.String                              `tfsdk:"ip_addresses"`
	VirtualRouterName types.String                                `tfsdk:"virtual_router_name"`
}

// cspConnectionVirtualRouterInterfaceModel represents the configuration of a CSP connection for Virtual Router interface.
type cspConnectionVirtualRouterInterfaceModel struct {
	IPAddresses []types.String `tfsdk:"ip_addresses"`
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
	OwnerUID              types.String             `tfsdk:"owner_uid"`
	UID                   types.String             `tfsdk:"product_uid,omitempty"`
	Name                  types.String             `tfsdk:"product_name"`
	LocationID            types.Int64              `tfsdk:"location_id"`
	Location              types.String             `tfsdk:"location"`
	VLAN                  types.Int64              `tfsdk:"vlan,omitempty"`
	InnerVLAN             types.Int64              `tfsdk:"inner_vlan,omitempty"`
	ProductUID            types.String             `tfsdk:"product_uid,omitempty"`
	NetworkInterfaceIndex types.Int64              `tfsdk:"vnic_index,omitempty"`
	SecondaryName         types.String             `tfsdk:"secondary_name"`
	PartnerConfig         *vxcPartnerConfiguration `tfsdk:"partner_config,omitempty"`
}

type vxcPartnerConfiguration struct {
	Partner             types.String                 `tfsdk:"partner"`
	AWSPartnerConfig    *vxcPartnerConfigAWSModel    `tfsdk:"aws_config,omitempty"`
	AzurePartnerConfig  *vxcPartnerConfigAzureModel  `tfsdk:"azure_config,omitempty"`
	GooglePartnerConfig *vxcPartnerConfigGoogleModel `tfsdk:"google_config,omitempty"`
	OraclePartnerConfig *vxcPartnerConfigOracleModel `tfsdk:"oracle_config,omitempty"`
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

func (orm *vxcResourceModel) fromAPIVXC(v *megaport.VXC) {
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
	orm.Locked = types.BoolValue(v.Locked)
	orm.AdminLocked = types.BoolValue(v.AdminLocked)
	orm.Cancelable = types.BoolValue(v.Cancelable)
	orm.LiveDate = types.StringValue(v.LiveDate.Format(time.RFC850))
	orm.CreateDate = types.StringValue(v.CreateDate.Format(time.RFC850))
	orm.ContractStartDate = types.StringValue(v.ContractStartDate.Format(time.RFC850))
	orm.ContractEndDate = types.StringValue(v.ContractEndDate.Format(time.RFC850))

	orm.AEndConfiguration = &vxcEndConfigurationModel{
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

	orm.BEndConfiguration = &vxcEndConfigurationModel{
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

	orm.VXCApproval = &vxcApprovalModel{
		Status:   types.StringValue(v.VXCApproval.Status),
		Message:  types.StringValue(v.VXCApproval.Message),
		UID:      types.StringValue(v.VXCApproval.UID),
		Type:     types.StringValue(v.VXCApproval.Type),
		NewSpeed: types.Int64Value(int64(v.VXCApproval.NewSpeed)),
	}

	resources := &vxcResourcesModel{
		VLL: &vllConfigModel{
			AEndVLAN:      types.Int64Value(int64(v.Resources.VLL.AEndVLAN)),
			BEndVLAN:      types.Int64Value(int64(v.Resources.VLL.BEndVLAN)),
			Description:   types.StringValue(v.Resources.VLL.Description),
			ID:            types.Int64Value(int64(v.Resources.VLL.ID)),
			Name:          types.StringValue(v.Resources.VLL.Name),
			RateLimitMBPS: types.Int64Value(int64(v.Resources.VLL.RateLimitMBPS)),
			ResourceName:  types.StringValue(v.Resources.VLL.ResourceName),
			ResourceType:  types.StringValue(v.Resources.VLL.ResourceType),
		},
		VirtualRouter: &virtualRouterModel{
			MCRAsn:             types.Int64Value(int64(v.Resources.VirtualRouter.MCRAsn)),
			ResourceName:       types.StringValue(v.Resources.VirtualRouter.ResourceName),
			ResourceType:       types.StringValue(v.Resources.VirtualRouter.ResourceType),
			Speed:              types.Int64Value(int64(v.Resources.VirtualRouter.Speed)),
			BGPShutdownDefault: types.BoolValue(v.Resources.VirtualRouter.BGPShutdownDefault),
		},
	}
	for _, i := range v.Resources.Interface {
		resources.Interface = append(resources.Interface, &portInterfaceModel{
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
		})
	}
	cspConnection := &vxcCSPConnectionModel{}
	for _, c := range v.Resources.CSPConnection.CSPConnection {
		cspConnection.CSPConnections = append(cspConnection.CSPConnections, fromAPICSPConnection(c))
	}
	orm.Resources.CSPConnection = cspConnection
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

	a := plan.AEndConfiguration
	b := plan.BEndConfiguration

	var aEnd megaport.VXCOrderEndpointConfiguration
	if !a.InnerVLAN.IsNull() {
		aEnd.InnerVLAN = int(a.InnerVLAN.ValueInt64())
	}
	if !a.VLAN.IsNull() {
		aEnd.VLAN = int(a.VLAN.ValueInt64())
	}
	if !a.NetworkInterfaceIndex.IsNull() {
		aEnd.NetworkInterfaceIndex = int(a.NetworkInterfaceIndex.ValueInt64())
	}
	if a.PartnerConfig != nil {
		switch a.PartnerConfig.Partner {
		case types.StringValue("aws"):
			aEnd.PartnerConfig = toAPIPartnerConfig(*a.PartnerConfig.AWSPartnerConfig)
		case types.StringValue("azure"):
			aEnd.PartnerConfig = toAPIPartnerConfig(*a.PartnerConfig.AzurePartnerConfig)
		case types.StringValue("google"):
			aEnd.PartnerConfig = toAPIPartnerConfig(*a.PartnerConfig.GooglePartnerConfig)
		case types.StringValue("oracle"):
			aEnd.PartnerConfig = toAPIPartnerConfig(*a.PartnerConfig.OraclePartnerConfig)
		}
	}

	var bEnd megaport.VXCOrderEndpointConfiguration
	if !b.InnerVLAN.IsNull() {
		bEnd.InnerVLAN = int(b.InnerVLAN.ValueInt64())
	}
	if !b.VLAN.IsNull() {
		bEnd.VLAN = int(b.VLAN.ValueInt64())
	}
	if !b.NetworkInterfaceIndex.IsNull() {
		bEnd.NetworkInterfaceIndex = int(b.NetworkInterfaceIndex.ValueInt64())
	}
	if b.PartnerConfig != nil {
		switch b.PartnerConfig.Partner {
		case types.StringValue("aws"):
			aEnd.PartnerConfig = toAPIPartnerConfig(*b.PartnerConfig.AWSPartnerConfig)
		case types.StringValue("azure"):
			aEnd.PartnerConfig = toAPIPartnerConfig(*b.PartnerConfig.AzurePartnerConfig)
		case types.StringValue("google"):
			aEnd.PartnerConfig = toAPIPartnerConfig(*b.PartnerConfig.GooglePartnerConfig)
		case types.StringValue("oracle"):
			aEnd.PartnerConfig = toAPIPartnerConfig(*b.PartnerConfig.OraclePartnerConfig)
		}
	}

	buyReq.AEndConfiguration = aEnd
	buyReq.BEndConfiguration = bEnd

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
	plan.fromAPIVXC(vxc)
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

	state.fromAPIVXC(vxc)

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

	var name string
	var aEndVlan, bEndVlan, rateLimit int
	if !plan.Name.Equal(state.Name) {
		name = plan.Name.ValueString()
	}
	if !plan.AEndConfiguration.VLAN.Equal(state.AEndConfiguration.VLAN) {
		aEndVlan = int(plan.AEndConfiguration.VLAN.ValueInt64())
	}
	if !plan.BEndConfiguration.VLAN.Equal(state.BEndConfiguration.VLAN) {
		bEndVlan = int(plan.BEndConfiguration.VLAN.ValueInt64())
	}
	if !plan.RateLimit.Equal(state.RateLimit) {
		rateLimit = int(plan.RateLimit.ValueInt64())
	}

	updateReq := &megaport.UpdateVXCRequest{
		Name:      &name,
		AEndVLAN:  &aEndVlan,
		BEndVLAN:  &bEndVlan,
		RateLimit: &rateLimit,
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

	state.fromAPIVXC(vxc)

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

func toAPIPartnerConfig(c vxcPartnerConfig) megaport.VXCPartnerConfiguration {
	switch p := c.(type) {
	case *vxcPartnerConfigAWSModel:
		return &megaport.VXCPartnerConfigAWS{
			ConnectType:       p.ConnectType.ValueString(),
			Type:              p.Type.ValueString(),
			OwnerAccount:      p.OwnerAccount.ValueString(),
			ASN:               int(p.ASN.ValueInt64()),
			AmazonASN:         int(p.AmazonASN.ValueInt64()),
			AuthKey:           p.AuthKey.ValueString(),
			Prefixes:          p.Prefixes.ValueString(),
			CustomerIPAddress: p.CustomerIPAddress.ValueString(),
			AmazonIPAddress:   p.AmazonIPAddress.ValueString(),
			ConnectionName:    p.ConnectionName.ValueString(),
		}
	case *vxcPartnerConfigAzureModel:
		return &megaport.VXCPartnerConfigAzure{
			ConnectType: p.ConnectType.ValueString(),
			ServiceKey:  p.ServiceKey.ValueString(),
		}
	case *vxcPartnerConfigGoogleModel:
		return &megaport.VXCPartnerConfigGoogle{
			ConnectType: p.ConnectType.ValueString(),
			PairingKey:  p.PairingKey.ValueString(),
		}
	case *vxcPartnerConfigOracleModel:
		return &megaport.VXCPartnerConfigOracle{
			ConnectType:      p.ConnectType.ValueString(),
			VirtualCircuitId: p.VirtualCircuitId.ValueString(),
		}
	}
	return nil
}

func fromAPICSPConnection(c megaport.CSPConnectionConfig) cspConnection {
	switch provider := c.(type) {
	case *megaport.CSPConnectionAWS:
		return &cspConnectionAWSModel{
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
	case *megaport.CSPConnectionAWSHC:
		toReturn := &cspConnectionAWSHCModel{
			ConnectType:  types.StringValue(provider.ConnectType),
			ResourceName: types.StringValue(provider.ResourceName),
			ResourceType: types.StringValue(provider.ResourceType),
			Bandwidth:    types.Int64Value(int64(provider.Bandwidth)),
			Name:         types.StringValue(provider.Name),
			OwnerAccount: types.StringValue(provider.OwnerAccount),
			ConnectionID: types.StringValue(provider.ConnectionID),
		}
		for _, b := range provider.Bandwidths {
			toReturn.Bandwidths = append(toReturn.Bandwidths, types.Int64Value(int64(b)))
		}
		return toReturn
	case *megaport.CSPConnectionAzure:
		toReturn := &cspConnectionAzureModel{
			ConnectType:  types.StringValue(provider.ConnectType),
			ResourceName: types.StringValue(provider.ResourceName),
			ResourceType: types.StringValue(provider.ResourceType),
			Bandwidth:    types.Int64Value(int64(provider.Bandwidth)),
			Managed:      types.BoolValue(provider.Managed),
			ServiceKey:   types.StringValue(provider.ServiceKey),
			VLAN:         types.Int64Value(int64(provider.VLAN)),
		}
		for _, m := range provider.Megaports {
			toReturn.Megaports = append(toReturn.Megaports, &cspConnectionAzureMegaportModel{
				Port: types.Int64Value(int64(m.Port)),
				Type: types.StringValue(m.Type),
				VXC:  types.Int64Value(int64(m.VXC)),
			})
		}
		for _, p := range provider.Ports {
			port := &cspConnectionAzurePortModel{
				ServiceID: types.Int64Value(int64(p.ServiceID)),
				Type:      types.StringValue(p.Type),
			}
			for _, v := range p.VXCServiceIDs {
				port.VXCServiceIDs = append(port.VXCServiceIDs, types.Int64Value(int64(v)))
			}
			toReturn.Ports = append(toReturn.Ports, port)
		}
		return toReturn
	case *megaport.CSPConnectionGoogle:
		toReturn := &cspConnectionGoogleModel{
			ConnectType:  types.StringValue(provider.ConnectType),
			ResourceName: types.StringValue(provider.ResourceName),
			ResourceType: types.StringValue(provider.ResourceType),
			Bandwidth:    types.Int64Value(int64(provider.Bandwidth)),
			CSPName:      types.StringValue(provider.CSPName),
			PairingKey:   types.StringValue(provider.PairingKey),
		}
		for _, b := range provider.Bandwidths {
			toReturn.Bandwidths = append(toReturn.Bandwidths, types.Int64Value(int64(b)))
		}
		for _, m := range provider.Megaports {
			toReturn.Megaports = append(toReturn.Megaports, &cspConnectionGoogleMegaportModel{
				Port: types.Int64Value(int64(m.Port)),
				VXC:  types.Int64Value(int64(m.VXC)),
			})
		}
		for _, p := range provider.Ports {
			port := &cspConnectionGooglePortModel{
				ServiceID: types.Int64Value(int64(p.ServiceID)),
			}
			for _, v := range p.VXCServiceIDs {
				port.VXCServiceIDs = append(port.VXCServiceIDs, types.Int64Value(int64(v)))
			}
			toReturn.Ports = append(toReturn.Ports, port)
		}
		return toReturn
	case *megaport.CSPConnectionVirtualRouter:
		toReturn := &cspConnectionVirtualRouterModel{
			ConnectType:       types.StringValue(provider.ConnectType),
			ResourceName:      types.StringValue(provider.ResourceName),
			ResourceType:      types.StringValue(provider.ResourceType),
			VLAN:              types.Int64Value(int64(provider.VLAN)),
			VirtualRouterName: types.StringValue(provider.VirtualRouterName),
		}
		for _, i := range provider.Interfaces {
			interfaceModel := &cspConnectionVirtualRouterInterfaceModel{}
			for _, ip := range i.IPAddresses {
				interfaceModel.IPAddresses = append(interfaceModel.IPAddresses, types.StringValue(ip))
			}
			toReturn.Interfaces = append(toReturn.Interfaces, interfaceModel)
		}
		for _, ip := range provider.IPAddresses {
			toReturn.IPAddresses = append(toReturn.IPAddresses, types.StringValue(ip))
		}
		return toReturn
	case *megaport.CSPConnectionTransit:
		return &cspConnectionTransit{
			ConnectType:        types.StringValue(provider.ConnectType),
			ResourceName:       types.StringValue(provider.ResourceName),
			ResourceType:       types.StringValue(provider.ResourceType),
			CustomerIP4Address: types.StringValue(provider.CustomerIP4Address),
			CustomerIP6Network: types.StringValue(provider.CustomerIP6Network),
			IPv4GatewayAddress: types.StringValue(provider.IPv4GatewayAddress),
			IPv6GatewayAddress: types.StringValue(provider.IPv6GatewayAddress),
		}
	}
	return nil
}
