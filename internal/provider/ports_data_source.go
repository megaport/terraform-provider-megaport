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

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource              = &portsDataSource{}
	_ datasource.DataSourceWithConfigure = &portsDataSource{}

	portDetailAttrs = map[string]attr.Type{
		"product_uid":            types.StringType,
		"product_name":           types.StringType,
		"provisioning_status":    types.StringType,
		"create_date":            types.StringType,
		"created_by":             types.StringType,
		"port_speed":             types.Int64Type,
		"terminate_date":         types.StringType,
		"live_date":              types.StringType,
		"market":                 types.StringType,
		"location_id":            types.Int64Type,
		"marketplace_visibility": types.BoolType,
		"vxc_permitted":          types.BoolType,
		"vxc_auto_approval":      types.BoolType,
		"secondary_name":         types.StringType,
		"lag_primary":            types.BoolType,
		"company_uid":            types.StringType,
		"company_name":           types.StringType,
		"cost_centre":            types.StringType,
		"contract_start_date":    types.StringType,
		"contract_end_date":      types.StringType,
		"contract_term_months":   types.Int64Type,
		"locked":                 types.BoolType,
		"admin_locked":           types.BoolType,
		"cancelable":             types.BoolType,
		"diversity_zone":         types.StringType,
		"resource_tags":          types.MapType{ElemType: types.StringType},
	}
)

// portsDataSource is the data source implementation.
type portsDataSource struct {
	client *megaport.Client
}

// portsModel maps the data source schema data.
type portsModel struct {
	ProductUID          types.String `tfsdk:"product_uid"`
	IncludeResourceTags types.Bool   `tfsdk:"include_resource_tags"`
	Ports               types.List   `tfsdk:"ports"`
}

// portDetailModel maps individual port detail attributes.
type portDetailModel struct {
	UID                   types.String `tfsdk:"product_uid"`
	Name                  types.String `tfsdk:"product_name"`
	ProvisioningStatus    types.String `tfsdk:"provisioning_status"`
	CreateDate            types.String `tfsdk:"create_date"`
	CreatedBy             types.String `tfsdk:"created_by"`
	PortSpeed             types.Int64  `tfsdk:"port_speed"`
	TerminateDate         types.String `tfsdk:"terminate_date"`
	LiveDate              types.String `tfsdk:"live_date"`
	Market                types.String `tfsdk:"market"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	MarketplaceVisibility types.Bool   `tfsdk:"marketplace_visibility"`
	VXCPermitted          types.Bool   `tfsdk:"vxc_permitted"`
	VXCAutoApproval       types.Bool   `tfsdk:"vxc_auto_approval"`
	SecondaryName         types.String `tfsdk:"secondary_name"`
	LAGPrimary            types.Bool   `tfsdk:"lag_primary"`
	CompanyUID            types.String `tfsdk:"company_uid"`
	CompanyName           types.String `tfsdk:"company_name"`
	CostCentre            types.String `tfsdk:"cost_centre"`
	ContractStartDate     types.String `tfsdk:"contract_start_date"`
	ContractEndDate       types.String `tfsdk:"contract_end_date"`
	ContractTermMonths    types.Int64  `tfsdk:"contract_term_months"`
	Locked                types.Bool   `tfsdk:"locked"`
	AdminLocked           types.Bool   `tfsdk:"admin_locked"`
	Cancelable            types.Bool   `tfsdk:"cancelable"`
	DiversityZone         types.String `tfsdk:"diversity_zone"`
	ResourceTags          types.Map    `tfsdk:"resource_tags"`
}

// NewPortsDataSource creates a new Ports data source.
func NewPortsDataSource() datasource.DataSource {
	return &portsDataSource{}
}

// Metadata returns the data source type name.
func (d *portsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ports"
}

// Schema defines the schema for the data source.
func (d *portsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up ports in the Megaport API. Optionally filter by product_uid to retrieve a specific port.",
		Attributes: map[string]schema.Attribute{
			"product_uid": schema.StringAttribute{
				Optional:    true,
				Description: "The unique identifier of a specific port to look up. If not provided, all ports are returned.",
			},
			"include_resource_tags": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to fetch resource tags for each port. Defaults to false. Enabling this causes an additional API call per port, which may be slow for accounts with many ports.",
			},
			"ports": schema.ListNestedAttribute{
				Description: "List of ports with detailed information.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"product_uid": schema.StringAttribute{
							Description: "The unique identifier of the port.",
							Computed:    true,
						},
						"product_name": schema.StringAttribute{
							Description: "The name of the port.",
							Computed:    true,
						},
						"provisioning_status": schema.StringAttribute{
							Description: "The provisioning status of the port.",
							Computed:    true,
						},
						"create_date": schema.StringAttribute{
							Description: "The date the port was created.",
							Computed:    true,
						},
						"created_by": schema.StringAttribute{
							Description: "The user who created the port.",
							Computed:    true,
						},
						"port_speed": schema.Int64Attribute{
							Description: "The bandwidth speed of the port in Mbps.",
							Computed:    true,
						},
						"terminate_date": schema.StringAttribute{
							Description: "The date the port will be terminated.",
							Computed:    true,
						},
						"live_date": schema.StringAttribute{
							Description: "The date the port went live.",
							Computed:    true,
						},
						"market": schema.StringAttribute{
							Description: "The market the port is in.",
							Computed:    true,
						},
						"location_id": schema.Int64Attribute{
							Description: "The numeric location ID of the port.",
							Computed:    true,
						},
						"marketplace_visibility": schema.BoolAttribute{
							Description: "Whether the port is visible in the Marketplace.",
							Computed:    true,
						},
						"vxc_permitted": schema.BoolAttribute{
							Description: "Whether VXC connections are permitted on this port.",
							Computed:    true,
						},
						"vxc_auto_approval": schema.BoolAttribute{
							Description: "Whether VXC connections are auto-approved on this port.",
							Computed:    true,
						},
						"secondary_name": schema.StringAttribute{
							Description: "The secondary name of the port.",
							Computed:    true,
						},
						"lag_primary": schema.BoolAttribute{
							Description: "Whether the port is a LAG primary.",
							Computed:    true,
						},
						"company_uid": schema.StringAttribute{
							Description: "The Megaport Company UID of the port owner.",
							Computed:    true,
						},
						"company_name": schema.StringAttribute{
							Description: "The name of the company that owns the port.",
							Computed:    true,
						},
						"cost_centre": schema.StringAttribute{
							Description: "The cost centre of the port for billing purposes.",
							Computed:    true,
						},
						"contract_start_date": schema.StringAttribute{
							Description: "The contract start date of the port.",
							Computed:    true,
						},
						"contract_end_date": schema.StringAttribute{
							Description: "The contract end date of the port.",
							Computed:    true,
						},
						"contract_term_months": schema.Int64Attribute{
							Description: "The contract term of the port in months.",
							Computed:    true,
						},
						"locked": schema.BoolAttribute{
							Description: "Whether the port is locked.",
							Computed:    true,
						},
						"admin_locked": schema.BoolAttribute{
							Description: "Whether the port is admin locked.",
							Computed:    true,
						},
						"cancelable": schema.BoolAttribute{
							Description: "Whether the port can be cancelled.",
							Computed:    true,
						},
						"diversity_zone": schema.StringAttribute{
							Description: "The diversity zone of the port.",
							Computed:    true,
						},
						"resource_tags": schema.MapAttribute{
							ElementType: types.StringType,
							Description: "The resource tags associated with the port.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *portsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *portsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data portsModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ports []*megaport.Port

	if !data.ProductUID.IsNull() && !data.ProductUID.IsUnknown() {
		// Look up a specific port by UID
		port, err := d.client.PortService.GetPort(ctx, data.ProductUID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading port",
				fmt.Sprintf("Unable to read port %s: %v", data.ProductUID.ValueString(), err),
			)
			return
		}
		if port == nil {
			resp.Diagnostics.AddError(
				"Error reading Port",
				"Port not found: "+data.ProductUID.ValueString(),
			)
			return
		}
		ports = []*megaport.Port{port}
	} else {
		// List all ports
		var err error
		ports, err = d.client.PortService.ListPorts(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error listing ports",
				fmt.Sprintf("Unable to list ports: %v", err),
			)
			return
		}
	}

	// Determine whether to fetch resource tags (opt-in to avoid N+1 API calls)
	fetchTags := !data.IncludeResourceTags.IsNull() && data.IncludeResourceTags.ValueBool()

	// Build detail objects
	portObjects := make([]types.Object, 0, len(ports))

	for _, port := range ports {
		var tags map[string]string
		if fetchTags {
			var err error
			tags, err = d.client.PortService.ListPortResourceTags(ctx, port.UID)
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Error fetching port tags",
					fmt.Sprintf("Unable to fetch resource tags for port %s: %v", port.UID, err),
				)
				tags = map[string]string{}
			}
		}

		detail, detailDiags := fromAPIPortDetail(port, tags)
		resp.Diagnostics.Append(detailDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		obj, objDiags := types.ObjectValueFrom(ctx, portDetailAttrs, &detail)
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		portObjects = append(portObjects, obj)
	}

	portsList, portsDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: portDetailAttrs}, portObjects)
	resp.Diagnostics.Append(portsDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Ports = portsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// fromAPIPortDetail maps an API Port and its resource tags to a portDetailModel.
func fromAPIPortDetail(p *megaport.Port, tags map[string]string) (portDetailModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	detail := portDetailModel{
		UID:                   types.StringValue(p.UID),
		Name:                  types.StringValue(p.Name),
		ProvisioningStatus:    types.StringValue(p.ProvisioningStatus),
		CreatedBy:             types.StringValue(p.CreatedBy),
		PortSpeed:             types.Int64Value(int64(p.PortSpeed)),
		Market:                types.StringValue(p.Market),
		LocationID:            types.Int64Value(int64(p.LocationID)),
		MarketplaceVisibility: types.BoolValue(p.MarketplaceVisibility),
		VXCPermitted:          types.BoolValue(p.VXCPermitted),
		VXCAutoApproval:       types.BoolValue(p.VXCAutoApproval),
		SecondaryName:         types.StringValue(p.SecondaryName),
		LAGPrimary:            types.BoolValue(p.LAGPrimary),
		CompanyUID:            types.StringValue(p.CompanyUID),
		CompanyName:           types.StringValue(p.CompanyName),
		CostCentre:            types.StringValue(p.CostCentre),
		ContractTermMonths:    types.Int64Value(int64(p.ContractTermMonths)),
		Locked:                types.BoolValue(p.Locked),
		AdminLocked:           types.BoolValue(p.AdminLocked),
		Cancelable:            types.BoolValue(p.Cancelable),
		DiversityZone:         types.StringValue(p.DiversityZone),
	}

	// Time fields
	if p.CreateDate != nil {
		detail.CreateDate = types.StringValue(p.CreateDate.String())
	} else {
		detail.CreateDate = types.StringValue("")
	}
	if p.LiveDate != nil {
		detail.LiveDate = types.StringValue(p.LiveDate.String())
	} else {
		detail.LiveDate = types.StringValue("")
	}
	if p.TerminateDate != nil {
		detail.TerminateDate = types.StringValue(p.TerminateDate.String())
	} else {
		detail.TerminateDate = types.StringValue("")
	}
	if p.ContractStartDate != nil {
		detail.ContractStartDate = types.StringValue(p.ContractStartDate.String())
	} else {
		detail.ContractStartDate = types.StringValue("")
	}
	if p.ContractEndDate != nil {
		detail.ContractEndDate = types.StringValue(p.ContractEndDate.String())
	} else {
		detail.ContractEndDate = types.StringValue("")
	}

	// Resource tags
	if len(tags) > 0 {
		resourceTagValues := make(map[string]attr.Value, len(tags))
		for k, v := range tags {
			resourceTagValues[k] = types.StringValue(v)
		}
		var mapDiags diag.Diagnostics
		detail.ResourceTags, mapDiags = types.MapValue(types.StringType, resourceTagValues)
		diags.Append(mapDiags...)
	} else {
		detail.ResourceTags = types.MapNull(types.StringType)
	}

	return detail, diags
}
