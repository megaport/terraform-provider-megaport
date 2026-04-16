package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &vxcCSPConnectionDataSource{}
	_ datasource.DataSourceWithConfigure = &vxcCSPConnectionDataSource{}
)

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
