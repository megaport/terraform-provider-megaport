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

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource              = &ixsDataSource{}
	_ datasource.DataSourceWithConfigure = &ixsDataSource{}

	ixDetailAttrs = map[string]attr.Type{
		"product_uid":          types.StringType,
		"product_name":         types.StringType,
		"provisioning_status":  types.StringType,
		"create_date":          types.StringType,
		"deploy_date":          types.StringType,
		"location_id":          types.Int64Type,
		"rate_limit":           types.Int64Type,
		"term":                 types.Int64Type,
		"secondary_name":       types.StringType,
		"vlan":                 types.Int64Type,
		"mac_address":          types.StringType,
		"asn":                  types.Int64Type,
		"network_service_type": types.StringType,
		"attribute_tags":       types.MapType{ElemType: types.StringType},
	}
)

// ixsDataSource is the data source implementation.
type ixsDataSource struct {
	client *megaport.Client
}

// ixsModel maps the data source schema data.
type ixsModel struct {
	ProductUID types.String `tfsdk:"product_uid"`
	IXs        types.List   `tfsdk:"ixs"`
}

// ixDetailModel maps individual IX detail attributes.
type ixDetailModel struct {
	UID                types.String `tfsdk:"product_uid"`
	Name               types.String `tfsdk:"product_name"`
	ProvisioningStatus types.String `tfsdk:"provisioning_status"`
	CreateDate         types.String `tfsdk:"create_date"`
	DeployDate         types.String `tfsdk:"deploy_date"`
	LocationID         types.Int64  `tfsdk:"location_id"`
	RateLimit          types.Int64  `tfsdk:"rate_limit"`
	Term               types.Int64  `tfsdk:"term"`
	SecondaryName      types.String `tfsdk:"secondary_name"`
	VLAN               types.Int64  `tfsdk:"vlan"`
	MACAddress         types.String `tfsdk:"mac_address"`
	ASN                types.Int64  `tfsdk:"asn"`
	NetworkServiceType types.String `tfsdk:"network_service_type"`
	AttributeTags      types.Map    `tfsdk:"attribute_tags"`
}

// NewIXsDataSource creates a new IXs data source.
func NewIXsDataSource() datasource.DataSource {
	return &ixsDataSource{}
}

// Metadata returns the data source type name.
func (d *ixsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ixs"
}

// Schema defines the schema for the data source.
func (d *ixsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up IXs in the Megaport API. Optionally filter by product_uid to retrieve a specific IX.",
		Attributes: map[string]schema.Attribute{
			"product_uid": schema.StringAttribute{
				Optional:    true,
				Description: "The unique identifier of a specific IX to look up. If not provided, all active IXs are returned.",
			},
			"ixs": schema.ListNestedAttribute{
				Description: "List of IXs with detailed information.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"product_uid": schema.StringAttribute{
							Description: "The unique identifier of the IX.",
							Computed:    true,
						},
						"product_name": schema.StringAttribute{
							Description: "The name of the IX.",
							Computed:    true,
						},
						"provisioning_status": schema.StringAttribute{
							Description: "The provisioning status of the IX.",
							Computed:    true,
						},
						"create_date": schema.StringAttribute{
							Description: "The date the IX was created.",
							Computed:    true,
						},
						"deploy_date": schema.StringAttribute{
							Description: "The date the IX was deployed.",
							Computed:    true,
						},
						"location_id": schema.Int64Attribute{
							Description: "The numeric location ID of the IX.",
							Computed:    true,
						},
						"rate_limit": schema.Int64Attribute{
							Description: "The rate limit of the IX in Mbps.",
							Computed:    true,
						},
						"term": schema.Int64Attribute{
							Description: "The contract term of the IX in months.",
							Computed:    true,
						},
						"secondary_name": schema.StringAttribute{
							Description: "The secondary name of the IX.",
							Computed:    true,
						},
						"vlan": schema.Int64Attribute{
							Description: "The VLAN of the IX.",
							Computed:    true,
						},
						"mac_address": schema.StringAttribute{
							Description: "The MAC address of the IX.",
							Computed:    true,
						},
						"asn": schema.Int64Attribute{
							Description: "The Autonomous System Number (ASN) of the IX.",
							Computed:    true,
						},
						"network_service_type": schema.StringAttribute{
							Description: "The network service type of the IX.",
							Computed:    true,
						},
						"attribute_tags": schema.MapAttribute{
							ElementType: types.StringType,
							Description: "The attribute tags of the IX.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *ixsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *ixsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ixsModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ixs []*megaport.IX

	if !data.ProductUID.IsNull() && !data.ProductUID.IsUnknown() {
		// Look up a specific IX by UID
		ix, err := d.client.IXService.GetIX(ctx, data.ProductUID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading IX",
				fmt.Sprintf("Unable to read IX %s: %v", data.ProductUID.ValueString(), err),
			)
			return
		}
		ixs = []*megaport.IX{ix}
	} else {
		// List all IXs
		var err error
		ixs, err = d.client.IXService.ListIXs(ctx, &megaport.ListIXsRequest{})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error listing IXs",
				fmt.Sprintf("Unable to list IXs: %v", err),
			)
			return
		}
	}

	// Build detail objects
	ixObjects := make([]types.Object, 0, len(ixs))

	for _, ix := range ixs {
		detail := fromAPIIXDetail(ix)
		obj, objDiags := types.ObjectValueFrom(ctx, ixDetailAttrs, &detail)
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		ixObjects = append(ixObjects, obj)
	}

	ixsList, ixsDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: ixDetailAttrs}, ixObjects)
	resp.Diagnostics.Append(ixsDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.IXs = ixsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// fromAPIIXDetail maps an API IX to an ixDetailModel.
func fromAPIIXDetail(ix *megaport.IX) ixDetailModel {
	detail := ixDetailModel{
		UID:                types.StringValue(ix.ProductUID),
		Name:               types.StringValue(ix.ProductName),
		ProvisioningStatus: types.StringValue(ix.ProvisioningStatus),
		LocationID:         types.Int64Value(int64(ix.LocationID)),
		RateLimit:          types.Int64Value(int64(ix.RateLimit)),
		Term:               types.Int64Value(int64(ix.Term)),
		SecondaryName:      types.StringValue(ix.SecondaryName),
		VLAN:               types.Int64Value(int64(ix.VLAN)),
		MACAddress:         types.StringValue(ix.MACAddress),
		ASN:                types.Int64Value(int64(ix.ASN)),
		NetworkServiceType: types.StringValue(ix.NetworkServiceType),
	}

	// Time fields
	if ix.CreateDate != nil {
		detail.CreateDate = types.StringValue(ix.CreateDate.String())
	} else {
		detail.CreateDate = types.StringValue("")
	}
	if ix.DeployDate != nil {
		detail.DeployDate = types.StringValue(ix.DeployDate.String())
	} else {
		detail.DeployDate = types.StringValue("")
	}

	// Attribute tags
	if ix.AttributeTags != nil {
		attrTagValues := make(map[string]attr.Value, len(ix.AttributeTags))
		for k, v := range ix.AttributeTags {
			attrTagValues[k] = types.StringValue(v)
		}
		detail.AttributeTags, _ = types.MapValue(types.StringType, attrTagValues)
	} else {
		detail.AttributeTags = types.MapNull(types.StringType)
	}

	return detail
}
