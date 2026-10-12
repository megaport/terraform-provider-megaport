package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &mcrIPsecDataSource{}
	_ datasource.DataSourceWithConfigure = &mcrIPsecDataSource{}
)

// NewMCRIPsecDataSource is a helper function to simplify the provider implementation.
func NewMCRIPsecDataSource() datasource.DataSource {
	return &mcrIPsecDataSource{}
}

// mcrIPsecDataSource defines the data source implementation.
type mcrIPsecDataSource struct {
	client *megaport.Client
}

// mcrIPsecDataSourceModel describes the data source data model.
type mcrIPsecDataSourceModel struct {
	MCRID               types.String `tfsdk:"mcr_id"`
	TotalTunnelCount    types.Int64  `tfsdk:"total_tunnel_count"`
	MaxTunnelCountLimit types.Int64  `tfsdk:"max_tunnel_count_limit"`
	VXCs                types.List   `tfsdk:"vxcs"`
}

// mcrIPsecVXCModel represents a VXC with IPsec tunnels in the data source.
type mcrIPsecVXCModel struct {
	Name       types.String `tfsdk:"name"`
	ProductUID types.String `tfsdk:"product_uid"`
	Tunnels    types.List   `tfsdk:"tunnels"`
}

// mcrIPsecTunnelModel represents a single IPsec tunnel in the data source.
type mcrIPsecTunnelModel struct {
	Description          types.String `tfsdk:"description"`
	SourceIPAddress      types.String `tfsdk:"source_ip_address"`
	DestinationIPAddress types.String `tfsdk:"destination_ip_address"`
	LocalID              types.String `tfsdk:"local_id"`
	RemoteID             types.String `tfsdk:"remote_id"`
	VLAN                 types.Int64  `tfsdk:"vlan"`
}

var (
	mcrIPsecTunnelAttributes = map[string]attr.Type{
		"description":            types.StringType,
		"source_ip_address":      types.StringType,
		"destination_ip_address": types.StringType,
		"local_id":               types.StringType,
		"remote_id":              types.StringType,
		"vlan":                   types.Int64Type,
	}

	mcrIPsecVXCAttributes = map[string]attr.Type{
		"name":        types.StringType,
		"product_uid": types.StringType,
		"tunnels":     types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(mcrIPsecTunnelAttributes)),
	}
)

// Metadata returns the data source type name.
func (d *mcrIPsecDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcr_ipsec"
}

// Schema defines the schema for the data source.
func (d *mcrIPsecDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading the IPsec tunnels on an MCR, grouped by VXC, and the number of tunnels configured. " +
			"The API does not return pre-shared keys or phase lifetimes.",
		Attributes: map[string]schema.Attribute{
			"mcr_id": schema.StringAttribute{
				Description: "The UID of the MCR to read IPsec tunnels for.",
				Required:    true,
			},
			"total_tunnel_count": schema.Int64Attribute{
				Description: "The number of IPsec tunnels configured on the MCR, across all its VXCs.",
				Computed:    true,
			},
			"max_tunnel_count_limit": schema.Int64Attribute{
				Description: "The highest tunnel count Megaport accepts for the MCR's IPsec add-on. " +
					"Megaport sets this limit for each MCR, and it is 30 unless Megaport raises it.",
				Computed: true,
			},
			"vxcs": schema.ListNestedAttribute{
				Description: "The VXCs on the MCR that have at least one IPsec tunnel.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the VXC.",
							Computed:    true,
						},
						"product_uid": schema.StringAttribute{
							Description: "The UID of the VXC.",
							Computed:    true,
						},
						"tunnels": schema.ListNestedAttribute{
							Description: "The IPsec tunnels configured on the VXC.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"description": schema.StringAttribute{
										Description: "The description of the tunnel interface.",
										Computed:    true,
									},
									"source_ip_address": schema.StringAttribute{
										Description: "Local (Megaport-side) IP address used as the tunnel source.",
										Computed:    true,
									},
									"destination_ip_address": schema.StringAttribute{
										Description: "Remote peer IP address the tunnel connects to.",
										Computed:    true,
									},
									"local_id": schema.StringAttribute{
										Description: "IKE local identifier override.",
										Computed:    true,
									},
									"remote_id": schema.StringAttribute{
										Description: "IKE remote identifier override.",
										Computed:    true,
									},
									"vlan": schema.Int64Attribute{
										Description: "The VLAN of the interface that holds `source_ip_address`. Null when no interface with a VLAN holds that address.",
										Computed:    true,
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

// Configure adds the provider configured client to the data source.
func (d *mcrIPsecDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*megaportProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *megaportProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client.client
}

// Read refreshes the Terraform state with the latest data.
func (d *mcrIPsecDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config mcrIPsecDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ipsec, err := d.client.MCRService.GetMCRIPsec(ctx, config.MCRID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading MCR IPsec tunnels",
			fmt.Sprintf("Could not read IPsec tunnels for MCR %s: %s", config.MCRID.ValueString(), err.Error()),
		)
		return
	}

	vxcObjects := []types.Object{}
	for _, vxc := range ipsec.IPsecConfiguredVXCs {
		tunnelObjects := []types.Object{}
		for _, tunnel := range vxc.Tunnels {
			tunnelObj, diags := types.ObjectValueFrom(ctx, mcrIPsecTunnelAttributes, &mcrIPsecTunnelModel{
				Description:          stringOrNull(tunnel.Description),
				SourceIPAddress:      types.StringValue(tunnel.SourceIpAddress),
				DestinationIPAddress: types.StringValue(tunnel.DestinationIpAddress),
				LocalID:              stringOrNull(tunnel.LocalId),
				RemoteID:             stringOrNull(tunnel.RemoteId),
				VLAN:                 int64PtrOrNull(tunnel.VLAN),
			})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			tunnelObjects = append(tunnelObjects, tunnelObj)
		}

		tunnelsList, diags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(mcrIPsecTunnelAttributes), tunnelObjects)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		vxcObj, diags := types.ObjectValueFrom(ctx, mcrIPsecVXCAttributes, &mcrIPsecVXCModel{
			Name:       types.StringValue(vxc.Name),
			ProductUID: types.StringValue(vxc.ProductUID),
			Tunnels:    tunnelsList,
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		vxcObjects = append(vxcObjects, vxcObj)
	}

	vxcsList, diags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(mcrIPsecVXCAttributes), vxcObjects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.TotalTunnelCount = types.Int64Value(int64(ipsec.TotalTunnelCount))
	config.MaxTunnelCountLimit = types.Int64Value(int64(ipsec.MaxTunnelCountLimit))
	config.VXCs = vxcsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
