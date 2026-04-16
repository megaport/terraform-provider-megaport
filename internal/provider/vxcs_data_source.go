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
	_ datasource.DataSource              = &vxcsDataSource{}
	_ datasource.DataSourceWithConfigure = &vxcsDataSource{}

	vxcDetailAttrs = map[string]attr.Type{
		"product_uid":          types.StringType,
		"product_name":         types.StringType,
		"rate_limit":           types.Int64Type,
		"provisioning_status":  types.StringType,
		"create_date":          types.StringType,
		"created_by":           types.StringType,
		"live_date":            types.StringType,
		"contract_term_months": types.Int64Type,
		"contract_start_date":  types.StringType,
		"contract_end_date":    types.StringType,
		"company_uid":          types.StringType,
		"company_name":         types.StringType,
		"cost_centre":          types.StringType,
		"distance_band":        types.StringType,
		"secondary_name":       types.StringType,
		"shutdown":             types.BoolType,
		"locked":               types.BoolType,
		"admin_locked":         types.BoolType,
		"cancelable":           types.BoolType,
		"a_end_uid":            types.StringType,
		"a_end_name":           types.StringType,
		"a_end_location_id":    types.Int64Type,
		"a_end_vlan":           types.Int64Type,
		"b_end_uid":            types.StringType,
		"b_end_name":           types.StringType,
		"b_end_location_id":    types.Int64Type,
		"b_end_vlan":           types.Int64Type,
		"attribute_tags":       types.MapType{ElemType: types.StringType},
		"resource_tags":        types.MapType{ElemType: types.StringType},
	}
)

// vxcsDataSource is the data source implementation.
type vxcsDataSource struct {
	client *megaport.Client
}

// vxcsModel maps the data source schema data.
type vxcsModel struct {
	ProductUID          types.String `tfsdk:"product_uid"`
	IncludeResourceTags types.Bool   `tfsdk:"include_resource_tags"`
	VXCs                types.List   `tfsdk:"vxcs"`
}

// vxcDetailModel maps individual VXC detail attributes.
type vxcDetailModel struct {
	UID                types.String `tfsdk:"product_uid"`
	Name               types.String `tfsdk:"product_name"`
	RateLimit          types.Int64  `tfsdk:"rate_limit"`
	ProvisioningStatus types.String `tfsdk:"provisioning_status"`
	CreateDate         types.String `tfsdk:"create_date"`
	CreatedBy          types.String `tfsdk:"created_by"`
	LiveDate           types.String `tfsdk:"live_date"`
	ContractTermMonths types.Int64  `tfsdk:"contract_term_months"`
	ContractStartDate  types.String `tfsdk:"contract_start_date"`
	ContractEndDate    types.String `tfsdk:"contract_end_date"`
	CompanyUID         types.String `tfsdk:"company_uid"`
	CompanyName        types.String `tfsdk:"company_name"`
	CostCentre         types.String `tfsdk:"cost_centre"`
	DistanceBand       types.String `tfsdk:"distance_band"`
	SecondaryName      types.String `tfsdk:"secondary_name"`
	Shutdown           types.Bool   `tfsdk:"shutdown"`
	Locked             types.Bool   `tfsdk:"locked"`
	AdminLocked        types.Bool   `tfsdk:"admin_locked"`
	Cancelable         types.Bool   `tfsdk:"cancelable"`
	AEndUID            types.String `tfsdk:"a_end_uid"`
	AEndName           types.String `tfsdk:"a_end_name"`
	AEndLocationID     types.Int64  `tfsdk:"a_end_location_id"`
	AEndVLAN           types.Int64  `tfsdk:"a_end_vlan"`
	BEndUID            types.String `tfsdk:"b_end_uid"`
	BEndName           types.String `tfsdk:"b_end_name"`
	BEndLocationID     types.Int64  `tfsdk:"b_end_location_id"`
	BEndVLAN           types.Int64  `tfsdk:"b_end_vlan"`
	AttributeTags      types.Map    `tfsdk:"attribute_tags"`
	ResourceTags       types.Map    `tfsdk:"resource_tags"`
}

// NewVXCsDataSource creates a new VXCs data source.
func NewVXCsDataSource() datasource.DataSource {
	return &vxcsDataSource{}
}

// Metadata returns the data source type name.
func (d *vxcsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vxcs"
}

// Schema defines the schema for the data source.
func (d *vxcsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up VXCs in the Megaport API. Optionally filter by product_uid to retrieve a specific VXC.",
		Attributes: map[string]schema.Attribute{
			"product_uid": schema.StringAttribute{
				Optional:    true,
				Description: "The unique identifier of a specific VXC to look up. If not provided, all active VXCs are returned.",
			},
			"include_resource_tags": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to fetch resource tags for each VXC. Defaults to false. Enabling this causes an additional API call per VXC, which may be slow for accounts with many VXCs.",
			},
			"vxcs": schema.ListNestedAttribute{
				Description: "List of VXCs with detailed information.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"product_uid": schema.StringAttribute{
							Description: "The unique identifier of the VXC.",
							Computed:    true,
						},
						"product_name": schema.StringAttribute{
							Description: "The name of the VXC.",
							Computed:    true,
						},
						"rate_limit": schema.Int64Attribute{
							Description: "The rate limit of the VXC in Mbps.",
							Computed:    true,
						},
						"provisioning_status": schema.StringAttribute{
							Description: "The provisioning status of the VXC.",
							Computed:    true,
						},
						"create_date": schema.StringAttribute{
							Description: "The date the VXC was created.",
							Computed:    true,
						},
						"created_by": schema.StringAttribute{
							Description: "The user who created the VXC.",
							Computed:    true,
						},
						"live_date": schema.StringAttribute{
							Description: "The date the VXC went live.",
							Computed:    true,
						},
						"contract_term_months": schema.Int64Attribute{
							Description: "The contract term of the VXC in months.",
							Computed:    true,
						},
						"contract_start_date": schema.StringAttribute{
							Description: "The contract start date of the VXC.",
							Computed:    true,
						},
						"contract_end_date": schema.StringAttribute{
							Description: "The contract end date of the VXC.",
							Computed:    true,
						},
						"company_uid": schema.StringAttribute{
							Description: "The Megaport Company UID of the VXC owner.",
							Computed:    true,
						},
						"company_name": schema.StringAttribute{
							Description: "The name of the company that owns the VXC.",
							Computed:    true,
						},
						"cost_centre": schema.StringAttribute{
							Description: "The cost centre of the VXC for billing purposes.",
							Computed:    true,
						},
						"distance_band": schema.StringAttribute{
							Description: "The distance band of the VXC.",
							Computed:    true,
						},
						"secondary_name": schema.StringAttribute{
							Description: "The secondary name of the VXC.",
							Computed:    true,
						},
						"shutdown": schema.BoolAttribute{
							Description: "Whether the VXC is shut down.",
							Computed:    true,
						},
						"locked": schema.BoolAttribute{
							Description: "Whether the VXC is locked.",
							Computed:    true,
						},
						"admin_locked": schema.BoolAttribute{
							Description: "Whether the VXC is admin locked.",
							Computed:    true,
						},
						"cancelable": schema.BoolAttribute{
							Description: "Whether the VXC can be cancelled.",
							Computed:    true,
						},
						"a_end_uid": schema.StringAttribute{
							Description: "The product UID of the A-End of the VXC.",
							Computed:    true,
						},
						"a_end_name": schema.StringAttribute{
							Description: "The product name of the A-End of the VXC.",
							Computed:    true,
						},
						"a_end_location_id": schema.Int64Attribute{
							Description: "The location ID of the A-End of the VXC.",
							Computed:    true,
						},
						"a_end_vlan": schema.Int64Attribute{
							Description: "The VLAN of the A-End of the VXC.",
							Computed:    true,
						},
						"b_end_uid": schema.StringAttribute{
							Description: "The product UID of the B-End of the VXC.",
							Computed:    true,
						},
						"b_end_name": schema.StringAttribute{
							Description: "The product name of the B-End of the VXC.",
							Computed:    true,
						},
						"b_end_location_id": schema.Int64Attribute{
							Description: "The location ID of the B-End of the VXC.",
							Computed:    true,
						},
						"b_end_vlan": schema.Int64Attribute{
							Description: "The VLAN of the B-End of the VXC.",
							Computed:    true,
						},
						"attribute_tags": schema.MapAttribute{
							ElementType: types.StringType,
							Description: "The attribute tags of the VXC.",
							Computed:    true,
						},
						"resource_tags": schema.MapAttribute{
							ElementType: types.StringType,
							Description: "The resource tags associated with the VXC.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *vxcsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *vxcsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data vxcsModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var vxcs []*megaport.VXC

	if !data.ProductUID.IsNull() && !data.ProductUID.IsUnknown() {
		// Look up a specific VXC by UID
		vxc, err := d.client.VXCService.GetVXC(ctx, data.ProductUID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading VXC",
				fmt.Sprintf("Unable to read VXC %s: %v", data.ProductUID.ValueString(), err),
			)
			return
		}
		vxcs = []*megaport.VXC{vxc}
	} else {
		// List all VXCs
		var err error
		vxcs, err = d.client.VXCService.ListVXCs(ctx, &megaport.ListVXCsRequest{
			IncludeInactive: false,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error listing VXCs",
				fmt.Sprintf("Unable to list VXCs: %v", err),
			)
			return
		}
	}

	// Determine whether to fetch resource tags (opt-in to avoid N+1 API calls)
	fetchTags := !data.IncludeResourceTags.IsNull() && data.IncludeResourceTags.ValueBool()

	// Build detail objects
	vxcObjects := make([]types.Object, 0, len(vxcs))

	for _, vxc := range vxcs {
		var tags map[string]string
		if fetchTags {
			var err error
			tags, err = d.client.VXCService.ListVXCResourceTags(ctx, vxc.UID)
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Error fetching VXC tags",
					fmt.Sprintf("Unable to fetch resource tags for VXC %s: %v", vxc.UID, err),
				)
				tags = map[string]string{}
			}
		}

		detail := fromAPIVXCDetail(vxc, tags)
		obj, objDiags := types.ObjectValueFrom(ctx, vxcDetailAttrs, &detail)
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		vxcObjects = append(vxcObjects, obj)
	}

	vxcsList, vxcsDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: vxcDetailAttrs}, vxcObjects)
	resp.Diagnostics.Append(vxcsDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.VXCs = vxcsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// fromAPIVXCDetail maps an API VXC and its resource tags to a vxcDetailModel.
func fromAPIVXCDetail(v *megaport.VXC, tags map[string]string) vxcDetailModel {
	detail := vxcDetailModel{
		UID:                types.StringValue(v.UID),
		Name:               types.StringValue(v.Name),
		RateLimit:          types.Int64Value(int64(v.RateLimit)),
		ProvisioningStatus: types.StringValue(v.ProvisioningStatus),
		CreatedBy:          types.StringValue(v.CreatedBy),
		ContractTermMonths: types.Int64Value(int64(v.ContractTermMonths)),
		CompanyUID:         types.StringValue(v.CompanyUID),
		CompanyName:        types.StringValue(v.CompanyName),
		CostCentre:         types.StringValue(v.CostCentre),
		DistanceBand:       types.StringValue(v.DistanceBand),
		SecondaryName:      types.StringValue(v.SecondaryName),
		Shutdown:           types.BoolValue(v.Shutdown),
		Locked:             types.BoolValue(v.Locked),
		AdminLocked:        types.BoolValue(v.AdminLocked),
		Cancelable:         types.BoolValue(v.Cancelable),
		AEndUID:            types.StringValue(v.AEndConfiguration.UID),
		AEndName:           types.StringValue(v.AEndConfiguration.Name),
		AEndLocationID:     types.Int64Value(int64(v.AEndConfiguration.LocationID)),
		AEndVLAN:           types.Int64Value(int64(v.AEndConfiguration.VLAN)),
		BEndUID:            types.StringValue(v.BEndConfiguration.UID),
		BEndName:           types.StringValue(v.BEndConfiguration.Name),
		BEndLocationID:     types.Int64Value(int64(v.BEndConfiguration.LocationID)),
		BEndVLAN:           types.Int64Value(int64(v.BEndConfiguration.VLAN)),
	}

	// Time fields
	if v.CreateDate != nil {
		detail.CreateDate = types.StringValue(v.CreateDate.String())
	} else {
		detail.CreateDate = types.StringValue("")
	}
	if v.LiveDate != nil {
		detail.LiveDate = types.StringValue(v.LiveDate.String())
	} else {
		detail.LiveDate = types.StringValue("")
	}
	if v.ContractStartDate != nil {
		detail.ContractStartDate = types.StringValue(v.ContractStartDate.String())
	} else {
		detail.ContractStartDate = types.StringValue("")
	}
	if v.ContractEndDate != nil {
		detail.ContractEndDate = types.StringValue(v.ContractEndDate.String())
	} else {
		detail.ContractEndDate = types.StringValue("")
	}

	// Attribute tags
	if v.AttributeTags != nil {
		attrTagValues := make(map[string]attr.Value, len(v.AttributeTags))
		for k, val := range v.AttributeTags {
			attrTagValues[k] = types.StringValue(val)
		}
		detail.AttributeTags, _ = types.MapValue(types.StringType, attrTagValues)
	} else {
		detail.AttributeTags = types.MapNull(types.StringType)
	}

	// Resource tags
	if len(tags) > 0 {
		resourceTagValues := make(map[string]attr.Value, len(tags))
		for k, val := range tags {
			resourceTagValues[k] = types.StringValue(val)
		}
		detail.ResourceTags, _ = types.MapValue(types.StringType, resourceTagValues)
	} else {
		detail.ResourceTags = types.MapNull(types.StringType)
	}

	return detail
}
