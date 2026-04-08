package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	megaport "github.com/megaport/megaportgo"
)

// Update Timeout for VXC Update Verification - will be configurable in future release.
const updateTimeout = 120 * time.Second

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &vxcResource{}
	_ resource.ResourceWithConfigure   = &vxcResource{}
	_ resource.ResourceWithImportState = &vxcResource{}

	// Leaf attr maps (no references to other maps) must be declared first.

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

	vxcVrouterInterfaceAttrs = map[string]attr.Type{
		"ip_mtu":           types.Int64Type,
		"ip_addresses":     types.ListType{}.WithElementType(types.StringType),
		"ip_routes":        types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(ipRouteAttrs)),
		"nat_ip_addresses": types.ListType{}.WithElementType(types.StringType),
		"bfd":              types.ObjectType{}.WithAttributeTypes(bfdConfigAttrs),
		"vlan":             types.Int64Type,
		"bgp_connections":  types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(bgpVrouterConnectionConfig)),
	}

	vxcPartnerConfigVrouterAttrs = map[string]attr.Type{
		"interfaces": types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(vxcVrouterInterfaceAttrs)),
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
		"peers":       types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(partnerOrderAzurePeeringConfigAttrs)),
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

	// Composite attr maps (reference leaf maps) declared after their dependencies.

	vxcAEndConfigAttrs = map[string]attr.Type{
		"product_uid":          types.StringType,
		"assigned_product_uid": types.StringType,
		"vlan":                 types.Int64Type,
		"inner_vlan":           types.Int64Type,
		"vnic_index":           types.Int64Type,
		"vrouter_config":       types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigVrouterAttrs),
	}

	vxcBEndConfigAttrs = map[string]attr.Type{
		"product_uid":          types.StringType,
		"assigned_product_uid": types.StringType,
		"vlan":                 types.Int64Type,
		"inner_vlan":           types.Int64Type,
		"vnic_index":           types.Int64Type,
		"aws_config":           types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigAWSAttrs),
		"azure_config":         types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigAzureAttrs),
		"google_config":        types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigGoogleAttrs),
		"oracle_config":        types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigOracleAttrs),
		"ibm_config":           types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigIbmAttrs),
		"vrouter_config":       types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigVrouterAttrs),
		"transit":              types.BoolType,
	}
)



// vxcResourceModel maps the resource schema data.
type vxcResourceModel struct {
	LastUpdated types.String `tfsdk:"last_updated"`

	UID                types.String `tfsdk:"product_uid"`
	ServiceID          types.Int64  `tfsdk:"service_id"`
	Name               types.String `tfsdk:"product_name"`
	RateLimit          types.Int64  `tfsdk:"rate_limit"`
	DistanceBand       types.String `tfsdk:"distance_band"`
	PromoCode          types.String `tfsdk:"promo_code"`
	ServiceKey         types.String `tfsdk:"service_key"`
	CreatedBy          types.String `tfsdk:"created_by"`
	ContractTermMonths types.Int64  `tfsdk:"contract_term_months"`
	CompanyUID         types.String `tfsdk:"company_uid"`
	AttributeTags      types.Map    `tfsdk:"attribute_tags"`
	CostCentre         types.String `tfsdk:"cost_centre"`
	Shutdown           types.Bool   `tfsdk:"shutdown"`

	AEndConfiguration types.Object `tfsdk:"a_end_config"`
	BEndConfiguration types.Object `tfsdk:"b_end_config"`

	ResourceTags types.Map `tfsdk:"resource_tags"`
}

// vxcAEndConfigModel maps the A-End configuration schema data.
type vxcAEndConfigModel struct {
	ProductUID            types.String `tfsdk:"product_uid"`
	AssignedProductUID    types.String `tfsdk:"assigned_product_uid"`
	VLAN                  types.Int64  `tfsdk:"vlan"`
	InnerVLAN             types.Int64  `tfsdk:"inner_vlan"`
	NetworkInterfaceIndex types.Int64  `tfsdk:"vnic_index"`
	VrouterPartnerConfig  types.Object `tfsdk:"vrouter_config"`
}

// vxcBEndConfigModel maps the B-End configuration schema data.
type vxcBEndConfigModel struct {
	ProductUID            types.String `tfsdk:"product_uid"`
	AssignedProductUID    types.String `tfsdk:"assigned_product_uid"`
	VLAN                  types.Int64  `tfsdk:"vlan"`
	InnerVLAN             types.Int64  `tfsdk:"inner_vlan"`
	NetworkInterfaceIndex types.Int64  `tfsdk:"vnic_index"`
	AWSPartnerConfig      types.Object `tfsdk:"aws_config"`
	AzurePartnerConfig    types.Object `tfsdk:"azure_config"`
	GooglePartnerConfig   types.Object `tfsdk:"google_config"`
	OraclePartnerConfig   types.Object `tfsdk:"oracle_config"`
	IBMPartnerConfig      types.Object `tfsdk:"ibm_config"`
	VrouterPartnerConfig  types.Object `tfsdk:"vrouter_config"`
	Transit               types.Bool   `tfsdk:"transit"`
}

type vxcPartnerConfig interface {
	isPartnerConfig()
}

// vxcPartnerConfigAWSModel maps the partner configuration schema data for AWS.
type vxcPartnerConfigAWSModel struct {
	vxcPartnerConfig  `tfsdk:"-"`
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
	vxcPartnerConfig `tfsdk:"-"`
	ServiceKey       types.String `tfsdk:"service_key"`
	Peers            types.List   `tfsdk:"peers"`
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
	vxcPartnerConfig `tfsdk:"-"`
	PairingKey       types.String `tfsdk:"pairing_key"`
}

// vxcPartnerConfigOracleModel maps the partner configuration schema data for Oracle.
type vxcPartnerConfigOracleModel struct {
	vxcPartnerConfig `tfsdk:"-"`
	VirtualCircuitId types.String `tfsdk:"virtual_circuit_id"`
}

// vxcPartnerConfigVrouterModel maps the partner configuration schema data for a vrouter configuration.
type vxcPartnerConfigVrouterModel struct {
	vxcPartnerConfig `tfsdk:"-"`
	Interfaces       types.List `tfsdk:"interfaces"`
}

type vxcPartnerConfigIbmModel struct {
	vxcPartnerConfig  `tfsdk:"-"`
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
			"distance_band": schema.StringAttribute{
				Description: "The distance band of the product.",
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
			"contract_term_months": schema.Int64Attribute{
				Description: "The term of the contract in months: valid values are 1, 12, 24, 36, 48, and 60. To set the product to a month-to-month contract with no minimum term, set the value to 1.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(1, 12, 24, 36, 48, 60),
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
			"resource_tags": schema.MapAttribute{
				Description: "The resource tags associated with the product.",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"company_uid": schema.StringAttribute{
				Description: "The UID of the company the product is associated with.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
			"a_end_config": schema.SingleNestedAttribute{
				Description: "The A-End configuration of the VXC.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"product_uid": schema.StringAttribute{
						Description: "The Product UID for the A-End configuration.",
						Required:    true,
					},
					"assigned_product_uid": schema.StringAttribute{
						Description: "The assigned product UID of the A-End configuration. The Megaport API may change a Partner Port from the requested UID to a different Port in the same location and diversity zone.",
						Optional:    true,
						Computed:    true,
					},
					"vlan": schema.Int64Attribute{
						Description: "The VLAN of the A-End configuration. Values can range from 2 to 4093. Set to 0 for auto-assignment. Set to -1 for untagged. If not set, the Megaport system allocates a valid VLAN.",
						Optional:    true,
						Computed:    true,
						Validators:  []validator.Int64{int64validator.Between(-1, 4093), int64validator.NoneOf(1)},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"inner_vlan": schema.Int64Attribute{
						Description: "The inner VLAN of the A-End configuration. This field is also used to specify the customer-side VLAN for Azure ExpressRoute single peering configurations. Note: Setting inner_vlan to 0 for auto-assignment is not currently supported by the provider.",
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
					"vrouter_config": vrouterPartnerConfigSchema,
				},
			},
			"b_end_config": schema.SingleNestedAttribute{
				Description: "The B-End configuration of the VXC.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"product_uid": schema.StringAttribute{
						Description: "The Product UID for the B-End configuration.",
						Optional:    true,
						Computed:    true,
					},
					"assigned_product_uid": schema.StringAttribute{
						Description: "The assigned product UID of the B-End configuration. The Megaport API may change a Partner Port from the requested UID to a different Port in the same location and diversity zone.",
						Optional:    true,
						Computed:    true,
					},
					"vlan": schema.Int64Attribute{
						Description: "The VLAN of the B-End configuration. Values can range from 2 to 4093. Set to 0 for auto-assignment. Set to -1 for untagged. If not set, the Megaport system allocates a valid VLAN.",
						Optional:    true,
						Computed:    true,
						Validators:  []validator.Int64{int64validator.Between(-1, 4093), int64validator.NoneOf(1)},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"inner_vlan": schema.Int64Attribute{
						Description: "The inner VLAN of the B-End configuration. Note: Setting inner_vlan to 0 for auto-assignment is not currently supported by the provider.",
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
					"aws_config":     awsPartnerConfigSchema,
					"azure_config":   azurePartnerConfigSchema,
					"google_config":  googlePartnerConfigSchema,
					"ibm_config":     ibmPartnerConfigSchema,
					"oracle_config":  oraclePartnerConfigSchema,
					"vrouter_config": vrouterPartnerConfigSchema,
					"transit": schema.BoolAttribute{
						Description: "Whether this is a transit VXC connection.",
						Optional:    true,
					},
				},
			},
		},
	}
}

// inferBEndPartnerType determines the partner type from the b-end config model.
func inferBEndPartnerType(endConfig *vxcBEndConfigModel) string {
	if isPresent(endConfig.AWSPartnerConfig) {
		return "aws"
	}
	if isPresent(endConfig.AzurePartnerConfig) {
		return "azure"
	}
	if isPresent(endConfig.GooglePartnerConfig) {
		return "google"
	}
	if isPresent(endConfig.OraclePartnerConfig) {
		return "oracle"
	}
	if isPresent(endConfig.IBMPartnerConfig) {
		return "ibm"
	}
	if isPresent(endConfig.VrouterPartnerConfig) {
		return "vrouter"
	}
	if isPresent(endConfig.Transit) && endConfig.Transit.ValueBool() {
		return "transit"
	}
	return ""
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
	if isPresent(plan.ServiceKey) {
		// If a service key is provided, look up the product UID pertaining to that service key
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

	// A-End config
	var a vxcAEndConfigModel
	aEndDiags := plan.AEndConfiguration.As(ctx, &a, basetypes.ObjectAsOptions{})
	if aEndDiags.HasError() {
		resp.Diagnostics.Append(aEndDiags...)
		return
	}
	aEndConfig := &megaport.VXCOrderEndpointConfiguration{
		ProductUID: a.ProductUID.ValueString(),
	}
	buyReq.PortUID = a.ProductUID.ValueString()

	if isPresent(a.VLAN) {
		aEndConfig.VLAN = int(a.VLAN.ValueInt64())
	} else {
		aEndConfig.VLAN = 0
	}

	// Check product type - if MVE, require VNIC Index
	productType, err := r.client.ProductService.GetProductType(ctx, a.ProductUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Could not determine product type",
			"Proceeding without product type validation for "+a.ProductUID.ValueString()+": "+err.Error(),
		)
	}
	if strings.EqualFold(productType, megaport.PRODUCT_MVE) {
		if !isPresent(a.NetworkInterfaceIndex) {
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

	// A-End vrouter partner config
	if isPresent(a.VrouterPartnerConfig) {
		var partnerConfigAEnd vxcPartnerConfigVrouterModel
		vrouterDiags := a.VrouterPartnerConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
		if vrouterDiags.HasError() {
			resp.Diagnostics.Append(vrouterDiags...)
			return
		}
		prefixFilterList, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, a.ProductUID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating VXC",
				"Could not create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
			)
			return
		}
		vrouterDiags2, vrouterMegaportConfig := createVrouterPartnerConfig(ctx, partnerConfigAEnd, prefixFilterList)
		if vrouterDiags2.HasError() {
			resp.Diagnostics.Append(vrouterDiags2...)
			return
		}
		aEndConfig.PartnerConfig = vrouterMegaportConfig
	}

	buyReq.AEndConfiguration = *aEndConfig

	// B-End config
	var b vxcBEndConfigModel
	bEndDiags := plan.BEndConfiguration.As(ctx, &b, basetypes.ObjectAsOptions{})
	if bEndDiags.HasError() {
		resp.Diagnostics.Append(bEndDiags...)
		return
	}
	bEndConfig := &megaport.VXCOrderEndpointConfiguration{
		ProductUID: b.ProductUID.ValueString(),
	}
	if serviceKeyBEndUID != "" {
		// If B End Product UID was provided and it differs from the Service Key Product UID, warn
		if b.ProductUID.ValueString() != "" && b.ProductUID.ValueString() != serviceKeyBEndUID {
			resp.Diagnostics.AddWarning(
				"Overriding B-End Product UID",
				"Overriding the requested B-End Product UID of "+b.ProductUID.ValueString()+" with "+serviceKeyBEndUID+" based on the provided Service Key.",
			)
		}
		bEndConfig.ProductUID = serviceKeyBEndUID
	}
	if isPresent(b.VLAN) {
		bEndConfig.VLAN = int(b.VLAN.ValueInt64())
	} else {
		bEndConfig.VLAN = 0
	}

	// Check product type - if MVE, require VNIC Index
	productType, err = r.client.ProductService.GetProductType(ctx, b.ProductUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Could not determine product type",
			"Proceeding without product type validation for "+b.ProductUID.ValueString()+": "+err.Error(),
		)
	}
	if strings.EqualFold(productType, megaport.PRODUCT_MVE) {
		if !isPresent(b.NetworkInterfaceIndex) {
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

	// Read B-end config from req.Config to access WriteOnly fields (auth_key, service_key)
	// that are stripped from the plan by the framework.
	var bFromConfig vxcBEndConfigModel
	{
		var cfgModel vxcResourceModel
		if cfgDiags := req.Config.Get(ctx, &cfgModel); !cfgDiags.HasError() {
			cfgModel.BEndConfiguration.As(ctx, &bFromConfig, basetypes.ObjectAsOptions{}) // diagnostics ignored: bFromConfig stays zero on error, WriteOnly injection is skipped
		}
	}

	// B-End partner configs
	bEndPartnerType := inferBEndPartnerType(&b)
	switch bEndPartnerType {
	case "aws":
		var awsConfig vxcPartnerConfigAWSModel
		awsDiags := b.AWSPartnerConfig.As(ctx, &awsConfig, basetypes.ObjectAsOptions{})
		if awsDiags.HasError() {
			resp.Diagnostics.Append(awsDiags...)
			return
		}
		// auth_key is WriteOnly — stripped from plan, inject from config
		if isPresent(bFromConfig.AWSPartnerConfig) {
			var awsCfg vxcPartnerConfigAWSModel
			if d := bFromConfig.AWSPartnerConfig.As(ctx, &awsCfg, basetypes.ObjectAsOptions{}); !d.HasError() {
				awsConfig.AuthKey = awsCfg.AuthKey
			}
		}
		if awsConfig.ConnectType.ValueString() == "AWS" {
			if awsConfig.Type.ValueString() != "public" && awsConfig.Type.ValueString() != "private" && awsConfig.Type.ValueString() != "transit" {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": AWS Connect Type must be public, private, or transit",
				)
				return
			}
		}
		awsDiags2, partnerConfig, _ := createAWSPartnerConfig(ctx, awsConfig)
		if awsDiags2.HasError() {
			resp.Diagnostics.Append(awsDiags2...)
			return
		}
		bEndConfig.PartnerConfig = partnerConfig
	case "azure":
		var azureConfig vxcPartnerConfigAzureModel
		azureDiags := b.AzurePartnerConfig.As(ctx, &azureConfig, basetypes.ObjectAsOptions{})
		if azureDiags.HasError() {
			resp.Diagnostics.Append(azureDiags...)
			return
		}
		// service_key is WriteOnly — stripped from plan, inject from config
		if isPresent(bFromConfig.AzurePartnerConfig) {
			var azureCfg vxcPartnerConfigAzureModel
			if d := bFromConfig.AzurePartnerConfig.As(ctx, &azureCfg, basetypes.ObjectAsOptions{}); !d.HasError() {
				azureConfig.ServiceKey = azureCfg.ServiceKey
			}
		}
		azureDiags2, azurePartnerConfig, _ := createAzurePartnerConfig(ctx, azureConfig)
		if azureDiags2.HasError() {
			resp.Diagnostics.Append(azureDiags2...)
			return
		}
		bEndConfig.PartnerConfig = azurePartnerConfig
	case "google":
		var googleConfig vxcPartnerConfigGoogleModel
		googleDiags := b.GooglePartnerConfig.As(ctx, &googleConfig, basetypes.ObjectAsOptions{})
		if googleDiags.HasError() {
			resp.Diagnostics.Append(googleDiags...)
			return
		}
		googleDiags2, googlePartnerConfig, _ := createGooglePartnerConfig(ctx, googleConfig)
		if googleDiags2.HasError() {
			resp.Diagnostics.Append(googleDiags2...)
			return
		}
		if bEndConfig.ProductUID == "" {
			partnerPortReq := &megaport.LookupPartnerPortsRequest{
				Key:       googleConfig.PairingKey.ValueString(),
				PortSpeed: int(plan.RateLimit.ValueInt64()),
				Partner:   "GOOGLE",
			}
			if !b.ProductUID.IsNull() {
				partnerPortReq.ProductID = b.ProductUID.ValueString()
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
		}
		bEndConfig.PartnerConfig = googlePartnerConfig
	case "oracle":
		var oracleConfig vxcPartnerConfigOracleModel
		oracleDiags := b.OraclePartnerConfig.As(ctx, &oracleConfig, basetypes.ObjectAsOptions{})
		if oracleDiags.HasError() {
			resp.Diagnostics.Append(oracleDiags...)
			return
		}
		oracleDiags2, oraclePartnerConfig, _ := createOraclePartnerConfig(ctx, oracleConfig)
		if oracleDiags2.HasError() {
			resp.Diagnostics.Append(oracleDiags2...)
			return
		}
		if bEndConfig.ProductUID == "" {
			partnerPortReq := &megaport.LookupPartnerPortsRequest{
				Key:       oracleConfig.VirtualCircuitId.ValueString(),
				PortSpeed: int(plan.RateLimit.ValueInt64()),
				Partner:   "ORACLE",
			}
			if !b.ProductUID.IsNull() {
				partnerPortReq.ProductID = b.ProductUID.ValueString()
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
		}
		bEndConfig.PartnerConfig = oraclePartnerConfig
	case "ibm":
		var ibmConfig vxcPartnerConfigIbmModel
		ibmDiags := b.IBMPartnerConfig.As(ctx, &ibmConfig, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(ibmDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		ibmDiags2, ibmPartnerConfig, _ := createIBMPartnerConfig(ctx, ibmConfig)
		if ibmDiags2.HasError() {
			resp.Diagnostics.Append(ibmDiags2...)
			return
		}
		bEndConfig.PartnerConfig = ibmPartnerConfig
	case "transit":
		bEndConfig.PartnerConfig = megaport.VXCPartnerConfigTransit{ConnectType: "TRANSIT"}
	case "vrouter":
		var partnerConfigBEnd vxcPartnerConfigVrouterModel
		vrouterDiags := b.VrouterPartnerConfig.As(ctx, &partnerConfigBEnd, basetypes.ObjectAsOptions{})
		if vrouterDiags.HasError() {
			resp.Diagnostics.Append(vrouterDiags...)
			return
		}
		prefixFilterList, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, b.ProductUID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating VXC",
				"Could not create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
			)
			return
		}
		vrouterDiags2, vrouterMegaportConfig := createVrouterPartnerConfig(ctx, partnerConfigBEnd, prefixFilterList)
		if vrouterDiags2.HasError() {
			resp.Diagnostics.Append(vrouterDiags2...)
			return
		}
		bEndConfig.PartnerConfig = vrouterMegaportConfig
	}

	buyReq.BEndConfiguration = *bEndConfig

	err = r.client.VXCService.ValidateVXCOrder(ctx, buyReq)
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
	apiDiags := plan.fromAPIVXC(ctx, vxc, tags, &plan)
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

	// In Read, state should preserve its own values, so pass nil
	apiDiags := state.fromAPIVXC(ctx, vxc, tags, nil)
	resp.Diagnostics.Append(apiDiags...)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *vxcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vxcResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var aEndPlan, aEndState vxcAEndConfigModel
	var bEndPlan, bEndPlanConfig, bEndStateConfig vxcBEndConfigModel
	var bEndState vxcBEndConfigModel

	aEndPlanDiags := plan.AEndConfiguration.As(ctx, &aEndPlan, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(aEndPlanDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	aEndStateDiags := state.AEndConfiguration.As(ctx, &aEndState, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(aEndStateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For B-end, we need full config model (with partner configs)
	bEndPlanFullDiags := plan.BEndConfiguration.As(ctx, &bEndPlanConfig, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(bEndPlanFullDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	bEndStateFullDiags := state.BEndConfiguration.As(ctx, &bEndStateConfig, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(bEndStateFullDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Also extract simple b-end for VLAN/VNIC updates
	bEndPlanDiags := plan.BEndConfiguration.As(ctx, &bEndPlan, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(bEndPlanDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	bEndStateDiags := state.BEndConfiguration.As(ctx, &bEndState, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(bEndStateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Safe fallback: productType will be empty string, EqualFold checks will return false,
	// and MVE-specific validation (vnic_index requirement) will be skipped. VLAN updates
	// proceed normally for non-MVE endpoints.
	aEndProductType, err := r.client.ProductService.GetProductType(ctx, aEndPlan.ProductUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Could not determine product type",
			"Proceeding without product type validation for "+aEndPlan.ProductUID.ValueString()+": "+err.Error(),
		)
	}
	bEndProductType, err2 := r.client.ProductService.GetProductType(ctx, bEndPlan.ProductUID.ValueString())
	if err2 != nil {
		resp.Diagnostics.AddWarning(
			"Could not determine product type",
			"Proceeding without product type validation for "+bEndPlan.ProductUID.ValueString()+": "+err2.Error(),
		)
	}

	updateReq := &megaport.UpdateVXCRequest{
		WaitForUpdate: true,
		WaitForTime:   waitForTime,
	}

	if !plan.Name.Equal(state.Name) {
		updateReq.Name = megaport.PtrTo(plan.Name.ValueString())
	}

	aEndPartnerType := inferBEndPartnerType(&vxcBEndConfigModel{
		VrouterPartnerConfig: aEndPlan.VrouterPartnerConfig,
	})
	bEndPartnerType := inferBEndPartnerType(&bEndPlanConfig)

	// Populate VLAN/VNIC fields for A-End and B-End.
	aEndDiags2 := r.buildEndVLANVnicUpdates(ctx, &aEndPlan, &aEndState, aEndProductType, aEndPartnerType, true, updateReq, plan.Name.ValueString())
	resp.Diagnostics.Append(aEndDiags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	// buildEndVLANVnicUpdates requires *vxcAEndConfigModel; adapt B-end for the call.
	bEndPlanBase := vxcAEndConfigModel{
		ProductUID: bEndPlan.ProductUID, AssignedProductUID: bEndPlan.AssignedProductUID,
		VLAN: bEndPlan.VLAN, InnerVLAN: bEndPlan.InnerVLAN, NetworkInterfaceIndex: bEndPlan.NetworkInterfaceIndex,
	}
	bEndStateBase := vxcAEndConfigModel{
		ProductUID: bEndState.ProductUID, AssignedProductUID: bEndState.AssignedProductUID,
		VLAN: bEndState.VLAN, InnerVLAN: bEndState.InnerVLAN, NetworkInterfaceIndex: bEndState.NetworkInterfaceIndex,
	}
	bEndDiags2 := r.buildEndVLANVnicUpdates(ctx, &bEndPlanBase, &bEndStateBase, bEndProductType, bEndPartnerType, false, updateReq, plan.Name.ValueString())
	resp.Diagnostics.Append(bEndDiags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Sync mutations from the adapter back to bEndState.
	bEndState.NetworkInterfaceIndex = bEndStateBase.NetworkInterfaceIndex
	bEndState.InnerVLAN = bEndStateBase.InnerVLAN

	if !plan.RateLimit.IsNull() && !plan.RateLimit.Equal(state.RateLimit) {
		updateReq.RateLimit = megaport.PtrTo(int(plan.RateLimit.ValueInt64()))
	}

	if !plan.Shutdown.IsNull() && !plan.Shutdown.Equal(state.Shutdown) {
		updateReq.Shutdown = megaport.PtrTo(plan.Shutdown.ValueBool())
	}

	// Always use the planned cost centre value, even if it's empty/null
	updateReq.CostCentre = megaport.PtrTo(plan.CostCentre.ValueString())

	if !plan.ContractTermMonths.IsNull() && !plan.ContractTermMonths.Equal(state.ContractTermMonths) {
		updateReq.Term = megaport.PtrTo(int(plan.ContractTermMonths.ValueInt64()))
	}

	// Determine if B-End is a CSP partner (not updatable)
	bEndCSP := false
	switch bEndPartnerType {
	case "aws", "azure", "google", "oracle", "ibm":
		bEndCSP = true
	}

	aEndCSP := false
	// A-End only uses vrouter, which is not a CSP

	if !aEndPlan.ProductUID.IsNull() && !aEndPlan.ProductUID.Equal(aEndState.ProductUID) {
		if !aEndCSP && !aEndPlan.ProductUID.Equal(aEndState.AssignedProductUID) {
			updateReq.AEndProductUID = megaport.PtrTo(aEndPlan.ProductUID.ValueString())
			aEndState.ProductUID = aEndPlan.ProductUID
		} else {
			aEndState.ProductUID = aEndState.AssignedProductUID
		}
	}
	if !bEndPlan.ProductUID.IsNull() && !bEndPlan.ProductUID.Equal(bEndState.ProductUID) {
		if !bEndCSP && !bEndPlan.ProductUID.Equal(bEndState.AssignedProductUID) {
			updateReq.BEndProductUID = megaport.PtrTo(bEndPlan.ProductUID.ValueString())
			bEndState.ProductUID = bEndPlan.ProductUID
		} else {
			bEndState.ProductUID = bEndState.AssignedProductUID
		}
	}

	// Detect A-End vrouter partner change
	r.applyAEndPartnerUpdate(ctx, plan, state, aEndPlan, aEndState, updateReq, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Detect B-End vrouter/transit partner change
	r.applyBEndPartnerUpdate(ctx, plan, state, bEndPlanConfig, bEndState, bEndPartnerType, bEndCSP, updateReq, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only send API call if there are changes to the VXC in the Update Request
	var isChanged bool
	var waitErr error
	if updateReq.Name != nil || updateReq.AEndInnerVLAN != nil || updateReq.BEndInnerVLAN != nil ||
		updateReq.AEndVLAN != nil || updateReq.BEndVLAN != nil ||
		updateReq.AVnicIndex != nil || updateReq.BVnicIndex != nil ||
		updateReq.RateLimit != nil || updateReq.CostCentre != nil ||
		updateReq.Shutdown != nil || updateReq.Term != nil ||
		updateReq.AEndProductUID != nil || updateReq.BEndProductUID != nil ||
		updateReq.AEndPartnerConfig != nil || updateReq.BEndPartnerConfig != nil {
		isChanged = true
	}
	if isChanged {
		_, err := r.client.VXCService.UpdateVXC(ctx, plan.UID.ValueString(), updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating VXC",
				"Could not update VXC with ID "+state.UID.ValueString()+": "+err.Error(),
			)
			return
		}

		// Add retry logic to wait for API propagation
		waitErr = r.waitForVXCUpdate(ctx, plan.UID.ValueString(), updateReq, updateTimeout)
		if waitErr != nil {
			resp.Diagnostics.AddWarning(
				"VXC Update Propagation Delay",
				fmt.Sprintf("VXC update completed but verification timed out: %s. The update may still be propagating.", waitErr.Error()),
			)
		}
	}

	// Update state with any changes from plan configuration following successful update
	aEndStateObj, aEndStateDiags := types.ObjectValueFrom(ctx, vxcAEndConfigAttrs, aEndState)
	resp.Diagnostics.Append(aEndStateDiags...)
	state.AEndConfiguration = aEndStateObj
	bEndStateObj, bEndStateDiags := types.ObjectValueFrom(ctx, vxcBEndConfigAttrs, bEndState)
	resp.Diagnostics.Append(bEndStateDiags...)
	state.BEndConfiguration = bEndStateObj

	// Build expected vnic_index values from the plan for the post-update read.
	var expectedAEndVnic, expectedBEndVnic *int
	if strings.EqualFold(aEndProductType, megaport.PRODUCT_MVE) && isPresent(aEndPlan.NetworkInterfaceIndex) {
		v := int(aEndPlan.NetworkInterfaceIndex.ValueInt64())
		expectedAEndVnic = &v
	}
	if strings.EqualFold(bEndProductType, megaport.PRODUCT_MVE) && isPresent(bEndPlan.NetworkInterfaceIndex) {
		v := int(bEndPlan.NetworkInterfaceIndex.ValueInt64())
		expectedBEndVnic = &v
	}

	// Get refreshed vxc value from API, waiting for vnic_index to propagate
	vxc, err := r.waitForVnicIndex(ctx, state.UID.ValueString(), expectedAEndVnic, expectedBEndVnic, updateTimeout)
	if err != nil {
		if vxc == nil {
			resp.Diagnostics.AddError(
				"Error Reading VXC",
				"Could not read VXC with ID "+state.UID.ValueString()+": "+err.Error(),
			)
			return
		}
		resp.Diagnostics.AddWarning(
			"VXC vnic_index Propagation Delay",
			fmt.Sprintf("VXC updated but vnic_index verification timed out: %s. The update may still be propagating.", err.Error()),
		)
	}

	// Only show VLAN mismatch warnings if waitForVXCUpdate failed (timed out or errored)
	if isChanged && waitErr != nil {
		if updateReq.AEndInnerVLAN != nil && vxc.AEndConfiguration.InnerVLAN != *updateReq.AEndInnerVLAN {
			resp.Diagnostics.AddWarning(
				"A-End Inner VLAN Mismatch",
				fmt.Sprintf("Expected A-End inner_vlan=%d but API returned %d. This may indicate API propagation delay.",
					*updateReq.AEndInnerVLAN, vxc.AEndConfiguration.InnerVLAN),
			)
		}

		if updateReq.BEndInnerVLAN != nil && vxc.BEndConfiguration.InnerVLAN != *updateReq.BEndInnerVLAN {
			resp.Diagnostics.AddWarning(
				"B-End Inner VLAN Mismatch",
				fmt.Sprintf("Expected B-End inner_vlan=%d but API returned %d. This may indicate API propagation delay.",
					*updateReq.BEndInnerVLAN, vxc.BEndConfiguration.InnerVLAN),
			)
		}
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

	// In Update, pass plan to preserve user-only configuration values
	apiDiags := state.fromAPIVXC(ctx, vxc, tags, &plan)
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

	data, ok := req.ProviderData.(*megaportProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *megaportProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	client := data.client

	r.client = client
}

func (r *vxcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}

// buildEndVLANVnicUpdates populates VLAN and VNIC-related fields in the update request
// for a single endpoint (A or B). The endPlan and endState are always typed vxcAEndConfigModel
// because B-end uses the same simplified model for VLAN/VNIC purposes.
// When isAEnd is true the A-end fields of updateReq are populated; otherwise B-end fields.
// It also mutates endState.InnerVLAN and endState.NetworkInterfaceIndex as needed.
// Returns a diag.Diagnostics with any errors encountered.
func (r *vxcResource) buildEndVLANVnicUpdates(
	ctx context.Context,
	endPlan, endState *vxcAEndConfigModel,
	productType string,
	partnerType string,
	isAEnd bool,
	updateReq *megaport.UpdateVXCRequest,
	planName string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	// Helper closures to set the right endpoint fields.
	setVLAN := func(v int) {
		if isAEnd {
			updateReq.AEndVLAN = megaport.PtrTo(v)
		} else {
			updateReq.BEndVLAN = megaport.PtrTo(v)
		}
	}
	getVLANPtr := func() *int {
		if isAEnd {
			return updateReq.AEndVLAN
		}
		return updateReq.BEndVLAN
	}
	setInnerVLAN := func(v int) {
		if isAEnd {
			updateReq.AEndInnerVLAN = megaport.PtrTo(v)
		} else {
			updateReq.BEndInnerVLAN = megaport.PtrTo(v)
		}
	}
	setVnicIndex := func(v int) {
		if isAEnd {
			updateReq.AVnicIndex = megaport.PtrTo(v)
		} else {
			updateReq.BVnicIndex = megaport.PtrTo(v)
		}
	}

	// VLAN: if changed, attempt to update.
	if isPresent(endPlan.VLAN) && !endPlan.VLAN.Equal(endState.VLAN) && supportVLANUpdates(partnerType) {
		setVLAN(int(endPlan.VLAN.ValueInt64()))
	}

	// VNIC index: required when endpoint is MVE.
	if strings.EqualFold(productType, megaport.PRODUCT_MVE) {
		setVnicIndex(int(endPlan.NetworkInterfaceIndex.ValueInt64()))

		if supportVLANUpdates(partnerType) && !endPlan.NetworkInterfaceIndex.Equal(endState.NetworkInterfaceIndex) {
			if !endPlan.VLAN.IsNull() && !endPlan.VLAN.Equal(endState.VLAN) {
				setVLAN(int(endPlan.VLAN.ValueInt64()))
			} else if !endState.VLAN.IsNull() && getVLANPtr() == nil {
				setVLAN(int(endState.VLAN.ValueInt64()))
			}
		}
	} else {
		endState.NetworkInterfaceIndex = types.Int64Null()
	}

	// Inner VLAN: if changed, attempt to update.
	if isPresent(endPlan.InnerVLAN) && !endPlan.InnerVLAN.Equal(endState.InnerVLAN) && supportVLANUpdates(partnerType) {
		setInnerVLAN(int(endPlan.InnerVLAN.ValueInt64()))
	}
	// Preserve -1 (untagged) in state.
	if isPresent(endPlan.InnerVLAN) && endPlan.InnerVLAN.ValueInt64() == -1 {
		endState.InnerVLAN = types.Int64Value(-1)
	}

	_ = planName // available for future error messages
	return diags
}

func (r *vxcResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
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
			aEndStateConfig := &vxcAEndConfigModel{}
			bEndStateConfig := &vxcBEndConfigModel{}
			aEndDiags := state.AEndConfiguration.As(ctx, aEndStateConfig, basetypes.ObjectAsOptions{})
			bEndDiags := state.BEndConfiguration.As(ctx, bEndStateConfig, basetypes.ObjectAsOptions{})
			diags = append(diags, aEndDiags...)
			diags = append(diags, bEndDiags...)

			aEndPlanConfig := &vxcAEndConfigModel{}
			bEndPlanConfig := &vxcBEndConfigModel{}
			aEndPlanDiags := plan.AEndConfiguration.As(ctx, aEndPlanConfig, basetypes.ObjectAsOptions{})
			bEndPlanDiags := plan.BEndConfiguration.As(ctx, bEndPlanConfig, basetypes.ObjectAsOptions{})
			diags = append(diags, aEndPlanDiags...)
			diags = append(diags, bEndPlanDiags...)

			// Infer partner types
			bEndCSP := false
			bEndPartnerType := inferBEndPartnerType(bEndPlanConfig)
			switch bEndPartnerType {
			case "aws", "azure", "google", "oracle", "ibm":
				bEndCSP = true
			}

			// Handle CSP partner config replace detection for B-End
			if bEndCSP {
				// Check if the b_end_config partner config changed
				planBEndCSPObj := extractBEndCSPObjForCompare(ctx, bEndPlanConfig)
				stateBEndCSPObj := extractBEndCSPObjForCompare(ctx, bEndStateConfig)
				if !planBEndCSPObj.Equal(stateBEndCSPObj) {
					resp.RequiresReplace = append(resp.RequiresReplace, path.Root("b_end_config"))
				}
			}

			// Transit is creation-time-only — any explicit change requires replace.
			// Only trigger replace when state had transit=true and plan no longer does (or vice versa).
			// Null-in-state after import with true-in-plan is NOT treated as a change here;
			// that case is handled by simply not sending transit partner config on Update.
			planTransit := isPresent(bEndPlanConfig.Transit) && bEndPlanConfig.Transit.ValueBool()
			stateTransit := isPresent(bEndStateConfig.Transit) && bEndStateConfig.Transit.ValueBool()
			if planTransit != stateTransit && isPresent(bEndStateConfig.Transit) {
				resp.RequiresReplace = append(resp.RequiresReplace, path.Root("b_end_config"))
			}

			// Handle product UID reconciliation for CSP connections
			if aEndStateConfig.ProductUID.IsNull() {
				if aEndPlanConfig.ProductUID.IsNull() {
					aEndStateConfig.ProductUID = aEndStateConfig.AssignedProductUID
					aEndPlanConfig.ProductUID = aEndStateConfig.AssignedProductUID
				} else {
					aEndStateConfig.ProductUID = aEndPlanConfig.ProductUID
				}
			}

			if bEndStateConfig.ProductUID.IsNull() {
				if bEndPlanConfig.ProductUID.IsNull() {
					bEndStateConfig.ProductUID = bEndStateConfig.AssignedProductUID
					bEndPlanConfig.ProductUID = bEndStateConfig.AssignedProductUID
				} else {
					bEndStateConfig.ProductUID = bEndPlanConfig.ProductUID
				}
			} else if bEndCSP {
				if !bEndPlanConfig.ProductUID.IsNull() && bEndPlanConfig.ProductUID.ValueString() != "" && !bEndPlanConfig.ProductUID.Equal(bEndStateConfig.ProductUID) {
					tflog.Info(ctx, "Cloud provider port mapping detected for B-End",
						map[string]any{
							"product_uid":          bEndPlanConfig.ProductUID.ValueString(),
							"assigned_product_uid": bEndStateConfig.AssignedProductUID.ValueString(),
						},
					)
				}
				bEndPlanConfig.ProductUID = bEndStateConfig.ProductUID
			}

			newPlanAEndObj, aEndDiags2 := types.ObjectValueFrom(ctx, vxcAEndConfigAttrs, aEndPlanConfig)
			newPlanBEndObj, bEndDiags2 := types.ObjectValueFrom(ctx, vxcBEndConfigAttrs, bEndPlanConfig)
			diags = append(diags, aEndDiags2...)
			diags = append(diags, bEndDiags2...)
			plan.AEndConfiguration = newPlanAEndObj
			plan.BEndConfiguration = newPlanBEndObj

			newStateAEndObj, aEndDiags3 := types.ObjectValueFrom(ctx, vxcAEndConfigAttrs, aEndStateConfig)
			newStateBEndObj, bEndDiags3 := types.ObjectValueFrom(ctx, vxcBEndConfigAttrs, bEndStateConfig)
			diags = append(diags, aEndDiags3...)
			diags = append(diags, bEndDiags3...)
			state.AEndConfiguration = newStateAEndObj
			state.BEndConfiguration = newStateBEndObj

			planDiags := req.Plan.Set(ctx, &plan)
			diags = append(diags, planDiags...)
			resp.Plan.Set(ctx, &plan)
			stateDiags2 := req.State.Set(ctx, &state)
			diags = append(diags, stateDiags2...)
		}
	}

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// extractBEndCSPObjForCompare extracts the active CSP partner config from a b-end config model,
// with WriteOnly fields (auth_key, service_key) nullified so plan/state comparisons are stable
// across applies. Without this, the framework's nullification of WriteOnly fields in state would
// cause the objects to never compare equal, triggering spurious RequiresReplace on every plan.
//
// WriteOnly fields nullified here (must be updated if new WriteOnly fields are added to partner configs):
//   - vxcPartnerConfigAWSModel.AuthKey    (vxc_schemas.go: aws_config.auth_key)
//   - vxcPartnerConfigAzureModel.ServiceKey (vxc_schemas.go: azure_config.service_key)
func extractBEndCSPObjForCompare(ctx context.Context, cfg *vxcBEndConfigModel) types.Object {
	if isPresent(cfg.AWSPartnerConfig) {
		var awsCfg vxcPartnerConfigAWSModel
		if diags := cfg.AWSPartnerConfig.As(ctx, &awsCfg, basetypes.ObjectAsOptions{}); !diags.HasError() {
			awsCfg.AuthKey = types.StringNull() // WriteOnly — always null in state post-apply
			if obj, d := types.ObjectValueFrom(ctx, vxcPartnerConfigAWSAttrs, &awsCfg); !d.HasError() {
				return obj
			}
		}
		return cfg.AWSPartnerConfig
	}
	if isPresent(cfg.AzurePartnerConfig) {
		var azureCfg vxcPartnerConfigAzureModel
		if diags := cfg.AzurePartnerConfig.As(ctx, &azureCfg, basetypes.ObjectAsOptions{}); !diags.HasError() {
			azureCfg.ServiceKey = types.StringNull() // WriteOnly — always null in state post-apply
			if obj, d := types.ObjectValueFrom(ctx, vxcPartnerConfigAzureAttrs, &azureCfg); !d.HasError() {
				return obj
			}
		}
		return cfg.AzurePartnerConfig
	}
	if isPresent(cfg.GooglePartnerConfig) {
		return cfg.GooglePartnerConfig
	}
	if isPresent(cfg.OraclePartnerConfig) {
		return cfg.OraclePartnerConfig
	}
	if isPresent(cfg.IBMPartnerConfig) {
		return cfg.IBMPartnerConfig
	}
	return types.ObjectNull(nil)
}

// nullifyVrouterPasswords returns a copy of the vrouter config object with all bgp_connection
// passwords set to null. Used before plan/state comparison so that the WriteOnly password field
// (always null in state after apply) does not cause spurious update API calls on every apply.
//
// WriteOnly fields nullified here (must be updated if new WriteOnly fields are added to vrouter_config):
//   - bgpConnectionConfigModel.Password  (vxc_schemas.go: bgp_connections[].password)
func nullifyVrouterPasswords(ctx context.Context, vrouterObj types.Object) types.Object {
	if !isPresent(vrouterObj) {
		return vrouterObj
	}
	var m vxcPartnerConfigVrouterModel
	if diags := vrouterObj.As(ctx, &m, basetypes.ObjectAsOptions{}); diags.HasError() {
		return vrouterObj
	}
	if !isPresent(m.Interfaces) {
		return vrouterObj
	}
	var ifaces []*vxcPartnerConfigInterfaceModel
	if diags := m.Interfaces.ElementsAs(ctx, &ifaces, false); diags.HasError() {
		return vrouterObj
	}
	for _, iface := range ifaces {
		if !isPresent(iface.BgpConnections) {
			continue
		}
		var bgpConns []*bgpConnectionConfigModel
		if diags := iface.BgpConnections.ElementsAs(ctx, &bgpConns, false); diags.HasError() {
			continue
		}
		for _, conn := range bgpConns {
			conn.Password = types.StringNull()
		}
		newBgpList, diags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(bgpVrouterConnectionConfig), bgpConns)
		if !diags.HasError() {
			iface.BgpConnections = newBgpList
		}
	}
	newIfaceList, diags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(vxcVrouterInterfaceAttrs), ifaces)
	if diags.HasError() {
		return vrouterObj
	}
	m.Interfaces = newIfaceList
	normalized, diags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, &m)
	if diags.HasError() {
		return vrouterObj
	}
	return normalized
}

// applyAEndPartnerUpdate detects A-end partner config changes and populates updateReq.AEndPartnerConfig.
func (r *vxcResource) applyAEndPartnerUpdate(
	ctx context.Context,
	plan, state vxcResourceModel,
	aEndPlan, aEndState vxcAEndConfigModel,
	updateReq *megaport.UpdateVXCRequest,
	diags *diag.Diagnostics,
) {
	if !plan.AEndConfiguration.Equal(state.AEndConfiguration) {
		aEndPlanVrouter := aEndPlan.VrouterPartnerConfig
		aEndStateVrouter := aEndState.VrouterPartnerConfig
		// Normalize passwords (WriteOnly, nullified in state post-apply) before comparing.
		aEndPlanVrouterNorm := nullifyVrouterPasswords(ctx, aEndPlanVrouter)
		if !aEndPlanVrouterNorm.Equal(aEndStateVrouter) && !aEndPlanVrouter.IsNull() {
			var partnerConfigAEnd vxcPartnerConfigVrouterModel
			vrouterDiags := aEndPlanVrouter.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
			diags.Append(vrouterDiags...)
			if diags.HasError() {
				return
			}
			prefixFilterList, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, aEndState.ProductUID.ValueString())
			if err != nil {
				diags.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}
			vrouterDiags2, vrouterPartnerConfig := createVrouterPartnerConfig(ctx, partnerConfigAEnd, prefixFilterList)
			if vrouterDiags2.HasError() {
				diags.Append(vrouterDiags2...)
				return
			}
			updateReq.AEndPartnerConfig = vrouterPartnerConfig
		}
	}
}

// applyBEndPartnerUpdate detects B-end partner config changes and populates updateReq.BEndPartnerConfig.
func (r *vxcResource) applyBEndPartnerUpdate(
	ctx context.Context,
	plan, state vxcResourceModel,
	bEndPlanConfig vxcBEndConfigModel,
	bEndState vxcBEndConfigModel,
	bEndPartnerType string,
	bEndCSP bool,
	updateReq *megaport.UpdateVXCRequest,
	diags *diag.Diagnostics,
) {
	// Detect B-End vrouter partner change (only vrouter is updatable)
	if !plan.BEndConfiguration.Equal(state.BEndConfiguration) && bEndPartnerType == "vrouter" && !bEndCSP {
		if !bEndPlanConfig.VrouterPartnerConfig.IsNull() {
			var vrouterModel vxcPartnerConfigVrouterModel
			vrouterDiags := bEndPlanConfig.VrouterPartnerConfig.As(ctx, &vrouterModel, basetypes.ObjectAsOptions{})
			diags.Append(vrouterDiags...)
			if diags.HasError() {
				return
			}
			prefixFilterList, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, bEndState.ProductUID.ValueString())
			if err != nil {
				diags.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}
			vrouterDiags2, vrouterPartnerConfig := createVrouterPartnerConfig(ctx, vrouterModel, prefixFilterList)
			if vrouterDiags2.HasError() {
				diags.Append(vrouterDiags2...)
				return
			}
			updateReq.BEndPartnerConfig = vrouterPartnerConfig
		}
	}
	// Transit is a creation-time-only property — the API rejects transit partner config on Update.
	// Changes to transit (e.g. true → false) are handled by RequiresReplace in ModifyPlan.
}
