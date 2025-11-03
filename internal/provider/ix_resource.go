package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &ixResource{}
	_ resource.ResourceWithConfigure   = &ixResource{}
	_ resource.ResourceWithImportState = &ixResource{}

	interfaceAttrTypes = map[string]attr.Type{
		"demarcation":   types.StringType,
		"loa_template":  types.StringType,
		"media":         types.StringType,
		"port_speed":    types.Int64Type,
		"resource_name": types.StringType,
		"resource_type": types.StringType,
		"up":            types.Int64Type,
		"shutdown":      types.BoolType,
	}

	// BGP Connection object attributes
	bgpConnectionAttrTypes = map[string]attr.Type{
		"asn":                 types.Int64Type,
		"customer_asn":        types.Int64Type,
		"customer_ip_address": types.StringType,
		"isp_asn":             types.Int64Type,
		"isp_ip_address":      types.StringType,
		"ix_peer_policy":      types.StringType,
		"max_prefixes":        types.Int64Type,
		"resource_name":       types.StringType,
		"resource_type":       types.StringType,
	}

	// IP Address object attributes
	ipAddressAttrTypes = map[string]attr.Type{
		"address":       types.StringType,
		"resource_name": types.StringType,
		"resource_type": types.StringType,
		"version":       types.Int64Type,
		"reverse_dns":   types.StringType,
	}

	// VPLS Interface object attributes
	vplsInterfaceAttrTypes = map[string]attr.Type{
		"mac_address":     types.StringType,
		"rate_limit_mbps": types.Int64Type,
		"resource_name":   types.StringType,
		"resource_type":   types.StringType,
		"vlan":            types.Int64Type,
		"shutdown":        types.BoolType,
	}

	// Resources object attributes
	resourcesAttrTypes = map[string]attr.Type{
		"interface":       types.ObjectType{AttrTypes: interfaceAttrTypes},
		"bgp_connections": types.ListType{ElemType: types.ObjectType{AttrTypes: bgpConnectionAttrTypes}},
		"ip_addresses":    types.ListType{ElemType: types.ObjectType{AttrTypes: ipAddressAttrTypes}},
		"vpls_interface":  types.ObjectType{AttrTypes: vplsInterfaceAttrTypes},
	}
)

// ixResourceModel maps the resource schema data.
type ixResourceModel struct {
	RequestedProductUID types.String `tfsdk:"requested_product_uid"`
	ProductUID          types.String `tfsdk:"product_uid"`
	ProductID           types.Int64  `tfsdk:"product_id"`
	ProductName         types.String `tfsdk:"product_name"`
	NetworkServiceType  types.String `tfsdk:"network_service_type"`
	ASN                 types.Int64  `tfsdk:"asn"`
	MACAddress          types.String `tfsdk:"mac_address"`
	RateLimit           types.Int64  `tfsdk:"rate_limit"`
	VLAN                types.Int64  `tfsdk:"vlan"`
	Shutdown            types.Bool   `tfsdk:"shutdown"`
	PromoCode           types.String `tfsdk:"promo_code"`
	CostCentre          types.String `tfsdk:"cost_centre"`
	PublicGraph         types.Bool   `tfsdk:"public_graph"`
	ReverseDNS          types.String `tfsdk:"reverse_dns"`
	ProvisioningStatus  types.String `tfsdk:"provisioning_status"`
	CreateDate          types.String `tfsdk:"create_date"`
	Term                types.Int64  `tfsdk:"term"`
	LocationID          types.Int64  `tfsdk:"location_id"`
	AttributeTags       types.Map    `tfsdk:"attribute_tags"`
	DeployDate          types.String `tfsdk:"deploy_date"`
	SecondaryName       types.String `tfsdk:"secondary_name"`
	IXPeerMacro         types.String `tfsdk:"ix_peer_macro"`
	UsageAlgorithm      types.String `tfsdk:"usage_algorithm"`

	Resources types.Object `tfsdk:"resources"`
}

type ixResourcesModel struct {
	Interface      types.Object `tfsdk:"interface"`
	BGPConnections types.List   `tfsdk:"bgp_connections"`
	IPAddresses    types.List   `tfsdk:"ip_addresses"`
	VPLSInterface  types.Object `tfsdk:"vpls_interface"`
}

type ixInterfaceModel struct {
	Demarcation  types.String `tfsdk:"demarcation"`
	LOATemplate  types.String `tfsdk:"loa_template"`
	Media        types.String `tfsdk:"media"`
	PortSpeed    types.Int64  `tfsdk:"port_speed"`
	ResourceName types.String `tfsdk:"resource_name"`
	ResourceType types.String `tfsdk:"resource_type"`
	Up           types.Int64  `tfsdk:"up"`
	Shutdown     types.Bool   `tfsdk:"shutdown"`
}

type ixBGPConnectionModel struct {
	ASN               types.Int64  `tfsdk:"asn"`
	CustomerASN       types.Int64  `tfsdk:"customer_asn"`
	CustomerIPAddress types.String `tfsdk:"customer_ip_address"`
	ISPASN            types.Int64  `tfsdk:"isp_asn"`
	ISPIPAddress      types.String `tfsdk:"isp_ip_address"`
	IXPeerPolicy      types.String `tfsdk:"ix_peer_policy"`
	MaxPrefixes       types.Int64  `tfsdk:"max_prefixes"`
	ResourceName      types.String `tfsdk:"resource_name"`
	ResourceType      types.String `tfsdk:"resource_type"`
}

type ixIPAddressModel struct {
	Address      types.String `tfsdk:"address"`
	ResourceName types.String `tfsdk:"resource_name"`
	ResourceType types.String `tfsdk:"resource_type"`
	Version      types.Int64  `tfsdk:"version"`
	ReverseDNS   types.String `tfsdk:"reverse_dns"`
}

type ixVPLSInterfaceModel struct {
	MACAddress    types.String `tfsdk:"mac_address"`
	RateLimitMbps types.Int64  `tfsdk:"rate_limit_mbps"`
	ResourceName  types.String `tfsdk:"resource_name"`
	ResourceType  types.String `tfsdk:"resource_type"`
	VLAN          types.Int64  `tfsdk:"vlan"`
	Shutdown      types.Bool   `tfsdk:"shutdown"`
}

// fromAPI maps the API IX response to the resource schema.
func (orm *ixResourceModel) fromAPI(ctx context.Context, ix *megaport.IX) {
	// Map basic fields
	orm.ProductUID = types.StringValue(ix.ProductUID)
	orm.ProductID = types.Int64Value(int64(ix.ProductID))
	orm.ProductName = types.StringValue(ix.ProductName)
	orm.NetworkServiceType = types.StringValue(ix.NetworkServiceType)
	orm.ASN = types.Int64Value(int64(ix.ASN))
	orm.RateLimit = types.Int64Value(int64(ix.RateLimit))
	orm.VLAN = types.Int64Value(int64(ix.VLAN))
	orm.PromoCode = types.StringValue(ix.PromoCode)
	orm.PublicGraph = types.BoolValue(ix.PublicGraph)
	orm.ProvisioningStatus = types.StringValue(ix.ProvisioningStatus)
	orm.Term = types.Int64Value(int64(ix.Term))
	orm.LocationID = types.Int64Value(int64(ix.LocationID))
	orm.SecondaryName = types.StringValue(ix.SecondaryName)
	orm.IXPeerMacro = types.StringValue(ix.IXPeerMacro)
	orm.UsageAlgorithm = types.StringValue(ix.UsageAlgorithm)

	if ix.PromoCode != "" {
		orm.PromoCode = types.StringValue(ix.PromoCode)
	} else {
		orm.PromoCode = types.StringNull()
	}

	// Handle dates
	if ix.CreateDate != nil {
		orm.CreateDate = types.StringValue(ix.CreateDate.Format(time.RFC3339))
	} else {
		orm.CreateDate = types.StringNull()
	}

	if ix.DeployDate != nil {
		orm.DeployDate = types.StringValue(ix.DeployDate.Format(time.RFC3339))
	} else {
		orm.DeployDate = types.StringNull()
	}

	// Build a resources model
	res := &ixResourcesModel{}

	// Interface
	if ix.Resources.Interface.ResourceType != "" {
		iface := &ixInterfaceModel{
			Demarcation:  types.StringValue(ix.Resources.Interface.Demarcation),
			LOATemplate:  types.StringValue(ix.Resources.Interface.LOATemplate),
			Media:        types.StringValue(ix.Resources.Interface.Media),
			PortSpeed:    types.Int64Value(int64(ix.Resources.Interface.PortSpeed)),
			ResourceName: types.StringValue(ix.Resources.Interface.ResourceName),
			ResourceType: types.StringValue(ix.Resources.Interface.ResourceType),
			Up:           types.Int64Value(int64(ix.Resources.Interface.Up)),
			Shutdown:     types.BoolValue(ix.Resources.Interface.Shutdown),
		}
		ifObj, diags := types.ObjectValueFrom(ctx, interfaceAttrTypes, iface)
		if diags.HasError() {
			res.Interface = types.ObjectNull(interfaceAttrTypes)
		} else {
			res.Interface = ifObj
		}
	} else {
		res.Interface = types.ObjectNull(interfaceAttrTypes)
	}

	// BGP Connections
	if len(ix.Resources.BGPConnections) > 0 {
		bgpModels := make([]ixBGPConnectionModel, 0, len(ix.Resources.BGPConnections))
		for _, conn := range ix.Resources.BGPConnections {
			bgpModels = append(bgpModels, ixBGPConnectionModel{
				ASN:               types.Int64Value(int64(conn.ASN)),
				CustomerASN:       types.Int64Value(int64(conn.CustomerASN)),
				CustomerIPAddress: types.StringValue(conn.CustomerIPAddress),
				ISPASN:            types.Int64Value(int64(conn.ISPASN)),
				ISPIPAddress:      types.StringValue(conn.ISPIPAddress),
				IXPeerPolicy:      types.StringValue(conn.IXPeerPolicy),
				MaxPrefixes:       types.Int64Value(int64(conn.MaxPrefixes)),
				ResourceName:      types.StringValue(conn.ResourceName),
				ResourceType:      types.StringValue(conn.ResourceType),
			})
		}
		bgpList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: bgpConnectionAttrTypes}, bgpModels)
		if diags.HasError() {
			res.BGPConnections = types.ListNull(types.ObjectType{AttrTypes: bgpConnectionAttrTypes})
		} else {
			res.BGPConnections = bgpList
		}
	} else {
		res.BGPConnections = types.ListNull(types.ObjectType{AttrTypes: bgpConnectionAttrTypes})
	}

	// IP Addresses
	if len(ix.Resources.IPAddresses) > 0 {
		ipModels := make([]ixIPAddressModel, 0, len(ix.Resources.IPAddresses))
		for _, addr := range ix.Resources.IPAddresses {
			ipModels = append(ipModels, ixIPAddressModel{
				Address:      types.StringValue(addr.Address),
				ResourceName: types.StringValue(addr.ResourceName),
				ResourceType: types.StringValue(addr.ResourceType),
				Version:      types.Int64Value(int64(addr.Version)),
				ReverseDNS:   types.StringValue(addr.ReverseDNS),
			})
		}
		ipList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: ipAddressAttrTypes}, ipModels)
		if diags.HasError() {
			res.IPAddresses = types.ListNull(types.ObjectType{AttrTypes: ipAddressAttrTypes})
		} else {
			res.IPAddresses = ipList
		}
	} else {
		res.IPAddresses = types.ListNull(types.ObjectType{AttrTypes: ipAddressAttrTypes})
	}

	// VPLS Interface
	if ix.Resources.VPLSInterface.ResourceType != "" {
		vpls := &ixVPLSInterfaceModel{
			MACAddress:    types.StringValue(ix.Resources.VPLSInterface.MACAddress),
			RateLimitMbps: types.Int64Value(int64(ix.Resources.VPLSInterface.RateLimitMbps)),
			ResourceName:  types.StringValue(ix.Resources.VPLSInterface.ResourceName),
			ResourceType:  types.StringValue(ix.Resources.VPLSInterface.ResourceType),
			VLAN:          types.Int64Value(int64(ix.Resources.VPLSInterface.VLAN)),
			Shutdown:      types.BoolValue(ix.Resources.VPLSInterface.Shutdown),
		}
		vplsObj, diags := types.ObjectValueFrom(ctx, vplsInterfaceAttrTypes, vpls)
		if diags.HasError() {
			res.VPLSInterface = types.ObjectNull(vplsInterfaceAttrTypes)
		} else {
			res.VPLSInterface = vplsObj
		}
	} else {
		res.VPLSInterface = types.ObjectNull(vplsInterfaceAttrTypes)
	}

	// Convert ixResourcesModel to a Terraform object
	resObj, diags := types.ObjectValueFrom(ctx, resourcesAttrTypes, res)
	if diags.HasError() {
		orm.Resources = types.ObjectNull(resourcesAttrTypes)
	} else {
		orm.Resources = resObj
	}
}

// NewIXResource is a helper function to simplify the provider implementation.
func NewIXResource() resource.Resource {
	return &ixResource{}
}

// ixResource is the resource implementation.
type ixResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *ixResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ix"
}

// Schema defines the schema for the resource.
func (r *ixResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Megaport Internet Exchange (IX).",
		Attributes: map[string]schema.Attribute{
			"requested_product_uid": schema.StringAttribute{
				Description: "UID identifier of the product to attach the IX to.",
				Required:    true,
			},
			"product_uid": schema.StringAttribute{
				Description: "UID identifier of the IX product.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_id": schema.Int64Attribute{
				Description: "Numeric ID of the IX product.",
				Computed:    true,
			},
			"product_name": schema.StringAttribute{
				Description: "Name of the IX.",
				Required:    true,
			},
			"network_service_type": schema.StringAttribute{
				Description: "The type of IX service, e.g., 'Los Angeles IX', 'Sydney IX'.",
				Required:    true,
			},
			"asn": schema.Int64Attribute{
				Description: "The ASN (Autonomous System Number) for the IX connection.",
				Optional:    true,
				Computed:    true,
			},
			"mac_address": schema.StringAttribute{
				Description: "The MAC address for the IX interface.",
				Required:    true,
			},
			"rate_limit": schema.Int64Attribute{
				Description: "The rate limit in Mbps for the IX connection.",
				Required:    true,
			},
			"vlan": schema.Int64Attribute{
				Description: "The VLAN ID for the IX connection.",
				Required:    true,
			},
			"shutdown": schema.BoolAttribute{
				Description: "Whether the IX connection is shut down. Default is false.",
				Optional:    true,
			},
			"promo_code": schema.StringAttribute{
				Description: "Promo code to apply to the IX.",
				Optional:    true,
			},
			"cost_centre": schema.StringAttribute{
				Description: "Cost centre for invoicing purposes.",
				Optional:    true,
			},
			"public_graph": schema.BoolAttribute{
				Description: "Whether the IX usage statistics are publicly viewable.",
				Optional:    true,
				Computed:    true,
			},
			"reverse_dns": schema.StringAttribute{
				Description: "Custom hostname for your IP address.",
				Optional:    true,
			},
			"provisioning_status": schema.StringAttribute{
				Description: "The provisioning status of the IX.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"create_date": schema.StringAttribute{
				Description: "The date the IX was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"term": schema.Int64Attribute{
				Description: "The term of the IX in months.",
				Computed:    true,
			},
			"location_id": schema.Int64Attribute{
				Description: "The ID of the location where the IX is provisioned.",
				Computed:    true,
			},
			"attribute_tags": schema.MapAttribute{
				Description: "Attribute tags associated with the IX.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"deploy_date": schema.StringAttribute{
				Description: "The date the IX was deployed.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secondary_name": schema.StringAttribute{
				Description: "Secondary name for the IX.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ix_peer_macro": schema.StringAttribute{
				Description: "IX peer macro configuration.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"usage_algorithm": schema.StringAttribute{
				Description: "Usage algorithm for the IX.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resources": schema.SingleNestedAttribute{
				Description: "Resources associated with the IX.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"interface": schema.SingleNestedAttribute{
						Description: "Interface details for the IX.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"demarcation": schema.StringAttribute{
								Description: "Demarcation point for the interface.",
								Computed:    true,
							},
							"loa_template": schema.StringAttribute{
								Description: "LOA template for the interface.",
								Computed:    true,
							},
							"media": schema.StringAttribute{
								Description: "Media type for the interface.",
								Computed:    true,
							},
							"port_speed": schema.Int64Attribute{
								Description: "Port speed in Mbps.",
								Computed:    true,
							},
							"resource_name": schema.StringAttribute{
								Description: "Resource name.",
								Computed:    true,
							},
							"resource_type": schema.StringAttribute{
								Description: "Resource type.",
								Computed:    true,
							},
							"up": schema.Int64Attribute{
								Description: "Interface up status.",
								Computed:    true,
							},
							"shutdown": schema.BoolAttribute{
								Description: "Whether the interface is shut down.",
								Computed:    true,
							},
						},
					},
					"bgp_connections": schema.ListNestedAttribute{
						Description: "BGP connections for the IX.",
						Computed:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"asn": schema.Int64Attribute{
									Description: "ASN for the BGP connection.",
									Computed:    true,
								},
								"customer_asn": schema.Int64Attribute{
									Description: "Customer ASN for the BGP connection.",
									Computed:    true,
								},
								"customer_ip_address": schema.StringAttribute{
									Description: "Customer IP address for the BGP connection.",
									Computed:    true,
								},
								"isp_asn": schema.Int64Attribute{
									Description: "ISP ASN for the BGP connection.",
									Computed:    true,
								},
								"isp_ip_address": schema.StringAttribute{
									Description: "ISP IP address for the BGP connection.",
									Computed:    true,
								},
								"ix_peer_policy": schema.StringAttribute{
									Description: "IX peer policy.",
									Computed:    true,
								},
								"max_prefixes": schema.Int64Attribute{
									Description: "Maximum prefixes.",
									Computed:    true,
								},
								"resource_name": schema.StringAttribute{
									Description: "Resource name.",
									Computed:    true,
								},
								"resource_type": schema.StringAttribute{
									Description: "Resource type.",
									Computed:    true,
								},
							},
						},
					},
					"ip_addresses": schema.ListNestedAttribute{
						Description: "IP addresses for the IX.",
						Computed:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"address": schema.StringAttribute{
									Description: "IP address.",
									Computed:    true,
								},
								"resource_name": schema.StringAttribute{
									Description: "Resource name.",
									Computed:    true,
								},
								"resource_type": schema.StringAttribute{
									Description: "Resource type.",
									Computed:    true,
								},
								"version": schema.Int64Attribute{
									Description: "IP version (4 or 6).",
									Computed:    true,
								},
								"reverse_dns": schema.StringAttribute{
									Description: "Reverse DNS for this IP address.",
									Computed:    true,
								},
							},
						},
					},
					"vpls_interface": schema.SingleNestedAttribute{
						Description: "VPLS interface details for the IX.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"mac_address": schema.StringAttribute{
								Description: "MAC address for the VPLS interface.",
								Computed:    true,
							},
							"rate_limit_mbps": schema.Int64Attribute{
								Description: "Rate limit in Mbps for the VPLS interface.",
								Computed:    true,
							},
							"resource_name": schema.StringAttribute{
								Description: "Resource name.",
								Computed:    true,
							},
							"resource_type": schema.StringAttribute{
								Description: "Resource type.",
								Computed:    true,
							},
							"vlan": schema.Int64Attribute{
								Description: "VLAN ID for the VPLS interface.",
								Computed:    true,
							},
							"shutdown": schema.BoolAttribute{
								Description: "Whether the VPLS interface is shut down.",
								Computed:    true,
							},
						},
					},
				},
			},
		},
	}
}

// Create a new resource.
func (r *ixResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan ixResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	buyReq := &megaport.BuyIXRequest{
		ProductUID:         plan.RequestedProductUID.ValueString(),
		Name:               plan.ProductName.ValueString(),
		NetworkServiceType: plan.NetworkServiceType.ValueString(),
		ASN:                int(plan.ASN.ValueInt64()),
		MACAddress:         plan.MACAddress.ValueString(),
		RateLimit:          int(plan.RateLimit.ValueInt64()),
		VLAN:               int(plan.VLAN.ValueInt64()),
		Shutdown:           plan.Shutdown.ValueBool(),
		PromoCode:          plan.PromoCode.ValueString(),
		WaitForProvision:   true,
		WaitForTime:        10 * time.Minute,
	}

	// Create the IX
	ixResp, err := r.client.IXService.BuyIX(ctx, buyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating IX",
			"Could not create IX, unexpected error: "+err.Error(),
		)
		return
	}

	// Get the created IX
	ix, err := r.client.IXService.GetIX(ctx, ixResp.TechnicalServiceUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading IX",
			"Could not read IX after creation, unexpected error: "+err.Error(),
		)
		return
	}

	// Update the plan with the IX info
	plan.fromAPI(ctx, ix)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *ixResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state ixResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed IX value from API
	ix, err := r.client.IXService.GetIX(ctx, state.ProductUID.ValueString())
	if err != nil {
		// IX has been deleted or is not found
		resp.State.RemoveResource(ctx)
		return
	}

	// Update the state with the IX info
	state.fromAPI(ctx, ix)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update the resource.
func (r *ixResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ixResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create update request with only fields that have changed
	updateReq := &megaport.UpdateIXRequest{
		WaitForUpdate: true,
		WaitForTime:   10 * time.Minute,
	}

	if !plan.ProductName.Equal(state.ProductName) {
		name := plan.ProductName.ValueString()
		updateReq.Name = &name
	}
	if !plan.RateLimit.Equal(state.RateLimit) {
		rateLimit := int(plan.RateLimit.ValueInt64())
		updateReq.RateLimit = &rateLimit
	}
	if !plan.VLAN.Equal(state.VLAN) {
		vlan := int(plan.VLAN.ValueInt64())
		updateReq.VLAN = &vlan
	}
	if !plan.MACAddress.Equal(state.MACAddress) {
		macAddress := plan.MACAddress.ValueString()
		updateReq.MACAddress = &macAddress
	}
	if !plan.ASN.Equal(state.ASN) {
		asn := int(plan.ASN.ValueInt64())
		updateReq.ASN = &asn
	}
	if !plan.Shutdown.Equal(state.Shutdown) {
		shutdown := plan.Shutdown.ValueBool()
		updateReq.Shutdown = &shutdown
	}

	// Apply the update if any fields changed
	if updateReq.Name != nil ||
		updateReq.RateLimit != nil ||
		updateReq.VLAN != nil ||
		updateReq.MACAddress != nil ||
		updateReq.ASN != nil ||
		updateReq.Shutdown != nil {
		_, err := r.client.IXService.UpdateIX(ctx, state.ProductUID.ValueString(), updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating IX",
				"Could not update IX, unexpected error: "+err.Error(),
			)
			return
		}
	}

	// Refetch the updated IX
	updatedIX, err := r.client.IXService.GetIX(ctx, state.ProductUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading IX",
			"Could not read IX after update, unexpected error: "+err.Error(),
		)
		return
	}

	// Update the state with the IX info
	state.fromAPI(ctx, updatedIX)

	// Persist the new state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ixResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state ixResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the IX
	err := r.client.IXService.DeleteIX(ctx, state.ProductUID.ValueString(), &megaport.DeleteIXRequest{
		DeleteNow: true,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting IX",
			"Could not delete IX, unexpected error: "+err.Error(),
		)
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r *ixResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}

// Configure adds the provider configured client to the resource.
func (r *ixResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
