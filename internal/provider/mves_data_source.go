package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource              = &mvesDataSource{}
	_ datasource.DataSourceWithConfigure = &mvesDataSource{}

	mveDetailAttrs = map[string]attr.Type{
		"product_uid":            types.StringType,
		"product_name":           types.StringType,
		"provisioning_status":    types.StringType,
		"create_date":            types.StringType,
		"created_by":             types.StringType,
		"terminate_date":         types.StringType,
		"live_date":              types.StringType,
		"market":                 types.StringType,
		"location_id":            types.Int64Type,
		"marketplace_visibility": types.BoolType,
		"vxc_permitted":          types.BoolType,
		"vxc_auto_approval":      types.BoolType,
		"secondary_name":         types.StringType,
		"company_uid":            types.StringType,
		"company_name":           types.StringType,
		"cost_centre":            types.StringType,
		"contract_start_date":    types.StringType,
		"contract_end_date":      types.StringType,
		"contract_term_months":   types.Int64Type,
		"locked":                 types.BoolType,
		"admin_locked":           types.BoolType,
		"cancelable":             types.BoolType,
		"vendor":                 types.StringType,
		"size":                   types.StringType,
		"diversity_zone":         types.StringType,
		"attribute_tags":         types.MapType{ElemType: types.StringType},
		"resource_tags":          types.MapType{ElemType: types.StringType},
	}
)

// mvesDataSource is the data source implementation.
type mvesDataSource struct {
	client *megaport.Client
}

// mvesModel maps the data source schema data.
type mvesModel struct {
	ProductUID          types.String `tfsdk:"product_uid"`
	IncludeResourceTags types.Bool   `tfsdk:"include_resource_tags"`
	MVEs                types.List   `tfsdk:"mves"`
}

// mveDetailModel maps individual MVE detail attributes.
type mveDetailModel struct {
	UID                   types.String `tfsdk:"product_uid"`
	Name                  types.String `tfsdk:"product_name"`
	ProvisioningStatus    types.String `tfsdk:"provisioning_status"`
	CreateDate            types.String `tfsdk:"create_date"`
	CreatedBy             types.String `tfsdk:"created_by"`
	TerminateDate         types.String `tfsdk:"terminate_date"`
	LiveDate              types.String `tfsdk:"live_date"`
	Market                types.String `tfsdk:"market"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	MarketplaceVisibility types.Bool   `tfsdk:"marketplace_visibility"`
	VXCPermitted          types.Bool   `tfsdk:"vxc_permitted"`
	VXCAutoApproval       types.Bool   `tfsdk:"vxc_auto_approval"`
	SecondaryName         types.String `tfsdk:"secondary_name"`
	CompanyUID            types.String `tfsdk:"company_uid"`
	CompanyName           types.String `tfsdk:"company_name"`
	CostCentre            types.String `tfsdk:"cost_centre"`
	ContractStartDate     types.String `tfsdk:"contract_start_date"`
	ContractEndDate       types.String `tfsdk:"contract_end_date"`
	ContractTermMonths    types.Int64  `tfsdk:"contract_term_months"`
	Locked                types.Bool   `tfsdk:"locked"`
	AdminLocked           types.Bool   `tfsdk:"admin_locked"`
	Cancelable            types.Bool   `tfsdk:"cancelable"`
	Vendor                types.String `tfsdk:"vendor"`
	Size                  types.String `tfsdk:"size"`
	DiversityZone         types.String `tfsdk:"diversity_zone"`
	AttributeTags         types.Map    `tfsdk:"attribute_tags"`
	ResourceTags          types.Map    `tfsdk:"resource_tags"`
}

// NewMVEsDataSource creates a new MVEs data source.
func NewMVEsDataSource() datasource.DataSource {
	return &mvesDataSource{}
}

// Metadata returns the data source type name.
func (d *mvesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mves"
}

// Schema defines the schema for the data source.
func (d *mvesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up MVEs in the Megaport API. Optionally filter by product_uid to retrieve a specific MVE.",
		Attributes: map[string]schema.Attribute{
			"product_uid": schema.StringAttribute{
				Optional:    true,
				Description: "The unique identifier of a specific MVE to look up. If not provided, all active MVEs are returned.",
			},
			"include_resource_tags": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to fetch resource tags for each MVE. Enabling this causes an additional API call per MVE, which may be slow for accounts with many MVEs.",
			},
			"mves": schema.ListNestedAttribute{
				Description: "List of MVEs with detailed information.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"product_uid": schema.StringAttribute{
							Description: "The unique identifier of the MVE.",
							Computed:    true,
						},
						"product_name": schema.StringAttribute{
							Description: "The name of the MVE.",
							Computed:    true,
						},
						"provisioning_status": schema.StringAttribute{
							Description: "The provisioning status of the MVE.",
							Computed:    true,
						},
						"create_date": schema.StringAttribute{
							Description: "The date the MVE was created.",
							Computed:    true,
						},
						"created_by": schema.StringAttribute{
							Description: "The user who created the MVE.",
							Computed:    true,
						},
						"terminate_date": schema.StringAttribute{
							Description: "The date the MVE will be terminated.",
							Computed:    true,
						},
						"live_date": schema.StringAttribute{
							Description: "The date the MVE went live.",
							Computed:    true,
						},
						"market": schema.StringAttribute{
							Description: "The market the MVE is in.",
							Computed:    true,
						},
						"location_id": schema.Int64Attribute{
							Description: "The numeric location ID of the MVE.",
							Computed:    true,
						},
						"marketplace_visibility": schema.BoolAttribute{
							Description: "Whether the MVE is visible in the Marketplace.",
							Computed:    true,
						},
						"vxc_permitted": schema.BoolAttribute{
							Description: "Whether VXC connections are permitted on this MVE.",
							Computed:    true,
						},
						"vxc_auto_approval": schema.BoolAttribute{
							Description: "Whether VXC connections are auto-approved on this MVE.",
							Computed:    true,
						},
						"secondary_name": schema.StringAttribute{
							Description: "The secondary name of the MVE.",
							Computed:    true,
						},
						"company_uid": schema.StringAttribute{
							Description: "The Megaport Company UID of the MVE owner.",
							Computed:    true,
						},
						"company_name": schema.StringAttribute{
							Description: "The name of the company that owns the MVE.",
							Computed:    true,
						},
						"cost_centre": schema.StringAttribute{
							Description: "The cost centre of the MVE for billing purposes.",
							Computed:    true,
						},
						"contract_start_date": schema.StringAttribute{
							Description: "The contract start date of the MVE.",
							Computed:    true,
						},
						"contract_end_date": schema.StringAttribute{
							Description: "The contract end date of the MVE.",
							Computed:    true,
						},
						"contract_term_months": schema.Int64Attribute{
							Description: "The contract term of the MVE in months.",
							Computed:    true,
						},
						"locked": schema.BoolAttribute{
							Description: "Whether the MVE is locked.",
							Computed:    true,
						},
						"admin_locked": schema.BoolAttribute{
							Description: "Whether the MVE is admin locked.",
							Computed:    true,
						},
						"cancelable": schema.BoolAttribute{
							Description: "Whether the MVE can be cancelled.",
							Computed:    true,
						},
						"vendor": schema.StringAttribute{
							Description: "The vendor of the MVE.",
							Computed:    true,
						},
						"size": schema.StringAttribute{
							Description: "The size of the MVE.",
							Computed:    true,
						},
						"diversity_zone": schema.StringAttribute{
							Description: "The diversity zone of the MVE.",
							Computed:    true,
						},
						"attribute_tags": schema.MapAttribute{
							ElementType: types.StringType,
							Description: "The attribute tags of the MVE.",
							Computed:    true,
						},
						"resource_tags": schema.MapAttribute{
							ElementType: types.StringType,
							Description: "The resource tags associated with the MVE. Only populated when include_resource_tags is enabled.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *mvesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *mvesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data mvesModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var mves []*megaport.MVE

	if !data.ProductUID.IsNull() && !data.ProductUID.IsUnknown() {
		// Look up a specific MVE by UID
		mve, err := d.client.MVEService.GetMVE(ctx, data.ProductUID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading MVE",
				fmt.Sprintf("Unable to read MVE %s: %v", data.ProductUID.ValueString(), err),
			)
			return
		}
		if mve == nil {
			resp.Diagnostics.AddError(
				"Error reading MVE",
				"MVE not found: "+data.ProductUID.ValueString(),
			)
			return
		}
		mves = []*megaport.MVE{mve}
	} else {
		// List all MVEs
		var err error
		mves, err = d.client.MVEService.ListMVEs(ctx, &megaport.ListMVEsRequest{
			IncludeInactive: false,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error listing MVEs",
				fmt.Sprintf("Unable to list MVEs: %v", err),
			)
			return
		}
	}

	// Determine whether to fetch resource tags (opt-in to avoid N+1 API calls)
	fetchTags := !data.IncludeResourceTags.IsNull() && !data.IncludeResourceTags.IsUnknown() && data.IncludeResourceTags.ValueBool()

	// Build detail objects
	mveObjects := make([]types.Object, 0, len(mves))

	for _, mve := range mves {
		var tags map[string]string
		if fetchTags {
			var err error
			tags, err = d.client.MVEService.ListMVEResourceTags(ctx, mve.UID)
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Error fetching MVE tags",
					fmt.Sprintf("Unable to fetch resource tags for MVE %s: %v", mve.UID, err),
				)
				tags = map[string]string{}
			}
		}

		detail, detailDiags := fromAPIMVEDetail(mve, tags)
		resp.Diagnostics.Append(detailDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		obj, objDiags := types.ObjectValueFrom(ctx, mveDetailAttrs, &detail)
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		mveObjects = append(mveObjects, obj)
	}

	mvesList, mvesDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: mveDetailAttrs}, mveObjects)
	resp.Diagnostics.Append(mvesDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.MVEs = mvesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// fromAPIMVEDetail maps an API MVE and its resource tags to an mveDetailModel.
func fromAPIMVEDetail(m *megaport.MVE, tags map[string]string) (mveDetailModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	detail := mveDetailModel{
		UID:                   types.StringValue(m.UID),
		Name:                  types.StringValue(m.Name),
		ProvisioningStatus:    types.StringValue(m.ProvisioningStatus),
		CreatedBy:             types.StringValue(m.CreatedBy),
		Market:                types.StringValue(m.Market),
		LocationID:            types.Int64Value(int64(m.LocationID)),
		MarketplaceVisibility: types.BoolValue(m.MarketplaceVisibility),
		VXCPermitted:          types.BoolValue(m.VXCPermitted),
		VXCAutoApproval:       types.BoolValue(m.VXCAutoApproval),
		SecondaryName:         types.StringValue(m.SecondaryName),
		CompanyUID:            types.StringValue(m.CompanyUID),
		CompanyName:           types.StringValue(m.CompanyName),
		CostCentre:            types.StringValue(m.CostCentre),
		ContractTermMonths:    types.Int64Value(int64(m.ContractTermMonths)),
		Locked:                types.BoolValue(m.Locked),
		AdminLocked:           types.BoolValue(m.AdminLocked),
		Cancelable:            types.BoolValue(m.Cancelable),
		Vendor:                types.StringValue(m.Vendor),
		Size:                  types.StringValue(m.Size),
		DiversityZone:         types.StringValue(m.DiversityZone),
	}

	// Time fields — use RFC850 format for consistency with the MVE resource,
	// and null for nil dates rather than empty strings.
	if m.CreateDate != nil {
		detail.CreateDate = types.StringValue(m.CreateDate.Format(time.RFC850))
	} else {
		detail.CreateDate = types.StringNull()
	}
	if m.LiveDate != nil {
		detail.LiveDate = types.StringValue(m.LiveDate.Format(time.RFC850))
	} else {
		detail.LiveDate = types.StringNull()
	}
	if m.TerminateDate != nil {
		detail.TerminateDate = types.StringValue(m.TerminateDate.Format(time.RFC850))
	} else {
		detail.TerminateDate = types.StringNull()
	}
	if m.ContractStartDate != nil {
		detail.ContractStartDate = types.StringValue(m.ContractStartDate.Format(time.RFC850))
	} else {
		detail.ContractStartDate = types.StringNull()
	}
	if m.ContractEndDate != nil {
		detail.ContractEndDate = types.StringValue(m.ContractEndDate.Format(time.RFC850))
	} else {
		detail.ContractEndDate = types.StringNull()
	}

	// Attribute tags
	if m.AttributeTags != nil {
		attrTagValues := make(map[string]attr.Value, len(m.AttributeTags))
		for k, v := range m.AttributeTags {
			attrTagValues[k] = types.StringValue(v)
		}
		var attrTagDiags diag.Diagnostics
		detail.AttributeTags, attrTagDiags = types.MapValue(types.StringType, attrTagValues)
		diags.Append(attrTagDiags...)
	} else {
		detail.AttributeTags = types.MapNull(types.StringType)
	}

	// Resource tags — nil means tags were not fetched (include_resource_tags=false)
	// and maps to null; a non-nil (possibly empty) map means tags were fetched.
	if tags != nil {
		resourceTagValues := make(map[string]attr.Value, len(tags))
		for k, v := range tags {
			resourceTagValues[k] = types.StringValue(v)
		}
		var resourceTagDiags diag.Diagnostics
		detail.ResourceTags, resourceTagDiags = types.MapValue(types.StringType, resourceTagValues)
		diags.Append(resourceTagDiags...)
	} else {
		detail.ResourceTags = types.MapNull(types.StringType)
	}

	return detail, diags
}
