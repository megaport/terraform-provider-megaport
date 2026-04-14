package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &vxcCSPConnectionDataSource{}
	_ datasource.DataSourceWithConfigure = &vxcCSPConnectionDataSource{}

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
)

// convertBandwidths converts a slice of int bandwidths from the API into a Terraform list value.
func convertBandwidths(ctx context.Context, src []int) (types.List, diag.Diagnostics) {
	vals := make([]int64, len(src))
	for i, v := range src {
		vals[i] = int64(v)
	}
	return types.ListValueFrom(ctx, types.Int64Type, vals)
}

// convertStringList converts a slice of strings into a Terraform list value.
func convertStringList(ctx context.Context, src []string) (types.List, diag.Diagnostics) {
	return types.ListValueFrom(ctx, types.StringType, src)
}

// newCSPConnectionModel returns a cspConnectionModel with list fields pre-initialised to null,
// so individual case handlers only need to set the fields relevant to their CSP type.
func newCSPConnectionModel() cspConnectionModel {
	return cspConnectionModel{
		Bandwidths:  types.ListNull(types.Int64Type),
		IPAddresses: types.ListNull(types.StringType),
	}
}

// NewVXCCSPConnectionDataSource is a helper function to simplify the provider implementation.
func NewVXCCSPConnectionDataSource() datasource.DataSource {
	return &vxcCSPConnectionDataSource{}
}

// vxcCSPConnectionDataSource is the data source implementation.
type vxcCSPConnectionDataSource struct {
	client *megaport.Client
}

// vxcCSPConnectionDataSourceModel maps the data source schema data.
type vxcCSPConnectionDataSourceModel struct {
	VXCUID         types.String `tfsdk:"vxc_uid"`
	CSPConnections types.List   `tfsdk:"csp_connections"`
}

// cspConnectionModel maps the CSP connection schema data.
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

// Metadata returns the data source type name.
func (d *vxcCSPConnectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vxc_csp_connection"
}

// Schema defines the schema for the data source.
func (d *vxcCSPConnectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "VXC CSP Connection Data Source for the Megaport Terraform Provider. Use this data source to retrieve the Cloud Service Provider (CSP) connections associated with a VXC.",
		Attributes: map[string]schema.Attribute{
			"vxc_uid": schema.StringAttribute{
				Description: "The UID of the VXC to retrieve CSP connections for.",
				Required:    true,
			},
			"csp_connections": schema.ListNestedAttribute{
				Description: "The Cloud Service Provider (CSP) connections associated with the VXC.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
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
						"customer_asn": schema.Int64Attribute{
							Description: "The customer ASN of the CSP connection.",
							Computed:    true,
						},
						"vlan": schema.Int64Attribute{
							Description: "The VLAN of the CSP connection.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the CSP connection.",
							Computed:    true,
						},
						"owner_account": schema.StringAttribute{
							Description: "The owner's AWS account of the CSP connection.",
							Computed:    true,
						},
						"account_id": schema.StringAttribute{
							Description: "The account ID of the CSP connection.",
							Computed:    true,
						},
						"bandwidth": schema.Int64Attribute{
							Description: "The bandwidth of the CSP connection.",
							Computed:    true,
						},
						"bandwidths": schema.ListAttribute{
							Description: "The bandwidths of the CSP connection.",
							Computed:    true,
							ElementType: types.Int64Type,
						},
						"customer_ip_address": schema.StringAttribute{
							Description: "The customer IP address of the CSP connection.",
							Computed:    true,
						},
						"provider_ip_address": schema.StringAttribute{
							Description: "The provider IP address of the CSP connection.",
							Computed:    true,
						},
						"customer_ip4_address": schema.StringAttribute{
							Description: "The customer IPv4 address of the CSP connection.",
							Computed:    true,
						},
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
							Sensitive:   true,
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
						"connection_id": schema.StringAttribute{
							Description: "The hosted connection ID of the CSP connection.",
							Computed:    true,
						},
						"managed": schema.BoolAttribute{
							Description: "Whether the CSP connection is managed.",
							Computed:    true,
						},
						"service_key": schema.StringAttribute{
							Description: "The Azure service key of the CSP connection.",
							Computed:    true,
							Sensitive:   true,
						},
						"csp_name": schema.StringAttribute{
							Description: "The name of the CSP connection.",
							Computed:    true,
						},
						"pairing_key": schema.StringAttribute{
							Description: "The pairing key of the Google Cloud connection.",
							Computed:    true,
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
	}
}

// Configure adds the provider configured client to the data source.
func (d *vxcCSPConnectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*megaportProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *megaportProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = data.client
}

// Read refreshes the Terraform state with the latest data.
func (d *vxcCSPConnectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state vxcCSPConnectionDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	vxc, err := d.client.VXCService.GetVXC(ctx, state.VXCUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VXC CSP Connections",
			"Could not read VXC with UID "+state.VXCUID.ValueString()+": "+err.Error(),
		)
		return
	}

	if vxc.Resources != nil && vxc.Resources.CSPConnection != nil {
		cspConnections := []types.Object{}
		for _, c := range vxc.Resources.CSPConnection.CSPConnection {
			cspConnection, cspDiags := fromAPICSPConnection(ctx, c)
			resp.Diagnostics.Append(cspDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			cspConnections = append(cspConnections, cspConnection)
		}
		cspConnectionsList, cspConnectionDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(cspConnectionFullAttrs), cspConnections)
		resp.Diagnostics.Append(cspConnectionDiags...)
		state.CSPConnections = cspConnectionsList
	} else {
		state.CSPConnections = types.ListNull(types.ObjectType{}.WithAttributeTypes(cspConnectionFullAttrs))
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func fromAPICSPConnection(ctx context.Context, c megaport.CSPConnectionConfig) (types.Object, diag.Diagnostics) {
	apiDiags := diag.Diagnostics{}
	m := newCSPConnectionModel()

	switch provider := c.(type) {
	case megaport.CSPConnectionAWS:
		m.ConnectType = types.StringValue(provider.ConnectType)
		m.ResourceName = types.StringValue(provider.ResourceName)
		m.ResourceType = types.StringValue(provider.ResourceType)
		m.VLAN = types.Int64Value(int64(provider.VLAN))
		m.Account = types.StringValue(provider.Account)
		m.AmazonAddress = types.StringValue(provider.AmazonAddress)
		m.ASN = types.Int64Value(int64(provider.ASN))
		m.AuthKey = types.StringValue(provider.AuthKey)
		m.CustomerAddress = types.StringValue(provider.CustomerAddress)
		m.CustomerIPAddress = types.StringValue(provider.CustomerIPAddress)
		m.ID = types.Int64Value(int64(provider.ID))
		m.Name = types.StringValue(provider.Name)
		m.OwnerAccount = types.StringValue(provider.OwnerAccount)
		m.PeerASN = types.Int64Value(int64(provider.PeerASN))
		m.Type = types.StringValue(provider.Type)
		m.VIFID = types.StringValue(provider.VIFID)
	case megaport.CSPConnectionAWSHC:
		m.ConnectType = types.StringValue(provider.ConnectType)
		m.ResourceName = types.StringValue(provider.ResourceName)
		m.ResourceType = types.StringValue(provider.ResourceType)
		m.Bandwidth = types.Int64Value(int64(provider.Bandwidth))
		m.Name = types.StringValue(provider.Name)
		m.OwnerAccount = types.StringValue(provider.OwnerAccount)
		m.ConnectionID = types.StringValue(provider.ConnectionID)
		bandwidthList, bandwidthDiags := convertBandwidths(ctx, provider.Bandwidths)
		apiDiags = append(apiDiags, bandwidthDiags...)
		m.Bandwidths = bandwidthList
	case megaport.CSPConnectionAzure:
		m.ConnectType = types.StringValue(provider.ConnectType)
		m.ResourceName = types.StringValue(provider.ResourceName)
		m.ResourceType = types.StringValue(provider.ResourceType)
		m.Bandwidth = types.Int64Value(int64(provider.Bandwidth))
		m.Managed = types.BoolValue(provider.Managed)
		m.ServiceKey = types.StringValue(provider.ServiceKey)
		m.VLAN = types.Int64Value(int64(provider.VLAN))
	case megaport.CSPConnectionGoogle:
		m.ConnectType = types.StringValue(provider.ConnectType)
		m.ResourceName = types.StringValue(provider.ResourceName)
		m.ResourceType = types.StringValue(provider.ResourceType)
		m.Bandwidth = types.Int64Value(int64(provider.Bandwidth))
		m.CSPName = types.StringValue(provider.CSPName)
		m.PairingKey = types.StringValue(provider.PairingKey)
		bandwidthList, bwListDiags := convertBandwidths(ctx, provider.Bandwidths)
		apiDiags = append(apiDiags, bwListDiags...)
		m.Bandwidths = bandwidthList
	case megaport.CSPConnectionVirtualRouter:
		m.ConnectType = types.StringValue(provider.ConnectType)
		m.ResourceName = types.StringValue(provider.ResourceName)
		m.ResourceType = types.StringValue(provider.ResourceType)
		m.VLAN = types.Int64Value(int64(provider.VLAN))
		m.VirtualRouterName = types.StringValue(provider.VirtualRouterName)
		ipList, ipListDiags := convertStringList(ctx, provider.IPAddresses)
		apiDiags = append(apiDiags, ipListDiags...)
		m.IPAddresses = ipList
	case megaport.CSPConnectionTransit:
		m.ConnectType = types.StringValue(provider.ConnectType)
		m.ResourceName = types.StringValue(provider.ResourceName)
		m.ResourceType = types.StringValue(provider.ResourceType)
		m.CustomerIP4Address = types.StringValue(provider.CustomerIP4Address)
		m.CustomerIP6Network = types.StringValue(provider.CustomerIP6Network)
		m.IPv4GatewayAddress = types.StringValue(provider.IPv4GatewayAddress)
		m.IPv6GatewayAddress = types.StringValue(provider.IPv6GatewayAddress)
	case megaport.CSPConnectionOracle:
		m.ConnectType = types.StringValue(provider.ConnectType)
		m.ResourceName = types.StringValue(provider.ResourceName)
		m.ResourceType = types.StringValue(provider.ResourceType)
		m.CSPName = types.StringValue(provider.CSPName)
		m.Bandwidth = types.Int64Value(int64(provider.Bandwidth))
		if provider.VirtualCircuitId != "" {
			m.ConnectionID = types.StringValue(provider.VirtualCircuitId)
		} else {
			m.ConnectionID = types.StringNull()
		}
	case megaport.CSPConnectionIBM:
		m.ConnectType = types.StringValue(provider.ConnectType)
		m.ResourceName = types.StringValue(provider.ResourceName)
		m.ResourceType = types.StringValue(provider.ResourceType)
		m.AccountID = types.StringValue(provider.AccountID)
		m.CustomerASN = types.Int64Value(int64(provider.CustomerASN))
		m.CustomerIPAddress = types.StringValue(provider.CustomerIPAddress)
		m.ProviderIPAddress = types.StringValue(provider.ProviderIPAddress)
		m.Bandwidth = types.Int64Value(int64(provider.Bandwidth))
		m.CSPName = types.StringValue(provider.CSPName)
		bandwidthList, bandwidthListDiags := convertBandwidths(ctx, provider.Bandwidths)
		apiDiags = append(apiDiags, bandwidthListDiags...)
		m.Bandwidths = bandwidthList
	default:
		apiDiags.AddError("Error creating CSP Connection", "Could not create CSP Connection, unknown type")
		return types.ObjectNull(cspConnectionFullAttrs), apiDiags
	}

	obj, objDiags := types.ObjectValueFrom(ctx, cspConnectionFullAttrs, &m)
	apiDiags = append(apiDiags, objDiags...)
	return obj, apiDiags
}
