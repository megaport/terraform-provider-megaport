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
	_ datasource.DataSource              = &partnerPortDataSource{}
	_ datasource.DataSourceWithConfigure = &partnerPortDataSource{}
)

// partnerPortDataSource is the data source implementation.
type partnerPortDataSource struct {
	client *megaport.Client
}

// locationModel maps the data source schema data.
type partnerPortModel struct {
	ConnectType   types.String `tfsdk:"connect_type"`
	ProductUID    types.String `tfsdk:"product_uid"`
	ProductName   types.String `tfsdk:"product_name"`
	CompanyUID    types.String `tfsdk:"company_uid"`
	CompanyName   types.String `tfsdk:"company_name"`
	DiversityZone types.String `tfsdk:"diversity_zone"`
	LocationID    types.Int64  `tfsdk:"location_id"`
	Speed         types.Int64  `tfsdk:"speed"`
	Rank          types.Int64  `tfsdk:"rank"`
	VXCPermitted  types.Bool   `tfsdk:"vxc_permitted"`
}

// NewpartnerPortDataSource is a helper function to simplify the provider implementation.
func NewPartnerPortDataSource() datasource.DataSource {
	return &partnerPortDataSource{}
}

// Metadata returns the data source type name.
func (d *partnerPortDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_partner"
}

// Schema defines the schema for the data source.
func (d *partnerPortDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Partner Port",
		Attributes: map[string]schema.Attribute{
			"connect_type": &schema.StringAttribute{
				Description: "The type of connection for the partner port.",
				Optional:    true,
				Computed:    true,
			},
			"product_uid": &schema.StringAttribute{
				Description: "The unique identifier of the partner port.",
				Computed:    true,
			},
			"product_name": &schema.StringAttribute{
				Description: "The name of the partner port.",
				Optional:    true,
				Computed:    true,
			},
			"company_uid": &schema.StringAttribute{
				Description: "The unique identifier of the company that owns the partner port.",
				Optional:    true,
				Computed:    true,
			},
			"company_name": &schema.StringAttribute{
				Description: "The name of the company that owns the partner port.",
				Optional:    true,
				Computed:    true,
			},
			"diversity_zone": &schema.StringAttribute{
				Description: "The diversity zone of the partner port.",
				Optional:    true,
				Computed:    true,
			},
			"location_id": &schema.Int64Attribute{
				Description: "The unique identifier of the location of the partner port.",
				Optional:    true,
				Computed:    true,
			},
			"speed": &schema.Int64Attribute{
				Description: "The speed of the partner port.",
				Computed:    true,
			},
			"rank": &schema.Int64Attribute{
				Description: "The rank of the partner port.",
				Computed:    true,
			},
			"vxc_permitted": &schema.BoolAttribute{
				Description: "Whether VXCs are permitted on the partner port.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *partnerPortDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config, state partnerPortModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// READ LOGIC GOES HERE
	partnerPorts, listErr := d.client.PartnerService.ListPartnerMegaports(ctx)
	if listErr != nil {
		resp.Diagnostics.AddError(
			"Error Reading VXC",
			"Could not list partner ports: "+listErr.Error(),
		)
		return
	}

	if len(partnerPorts) == 0 {
		resp.Diagnostics.AddError(
			"No Partner Ports Found",
			"No partner ports were found.",
		)
		return
	}

	var isFound bool
	var filters = []bool{}
	filterByProductName := !config.ProductName.IsNull()
	filterByConnectType := !config.ConnectType.IsNull()
	filterByLocationID := !config.LocationID.IsNull()
	filterByCompanyName := !config.CompanyName.IsNull()
	filterByDiversityZone := !config.DiversityZone.IsNull()

	filteredProductNamePartners := []*megaport.PartnerMegaport{}
	filterdConnectTypePartners := []*megaport.PartnerMegaport{}
	filteredLocationIDPartners := []*megaport.PartnerMegaport{}
	filteredCompanyNamePartners := []*megaport.PartnerMegaport{}
	filteredDiversityZonePartners := []*megaport.PartnerMegaport{}

	var filteredPartners []*megaport.PartnerMegaport

	if filterByProductName {
		filters = append(filters, filterByProductName)
	}
	if filterByConnectType {
		filters = append(filters, filterByConnectType)
	}
	if filterByLocationID {
		filters = append(filters, filterByLocationID)
	}
	if filterByCompanyName {
		filters = append(filters, filterByCompanyName)
	}
	if filterByDiversityZone {
		filters = append(filters, filterByDiversityZone)
	}

	if filterByProductName {
		filtered, err := d.client.PartnerService.FilterPartnerMegaportByProductName(ctx, partnerPorts, config.ProductName.ValueString(), true)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Filtering Partner Port",
				"Could not filter partner ports by product name: "+err.Error(),
			)
		}
		if len(filtered) == 1 {
			isFound = true
			state.fromAPIPartnerPort(filtered[0])
		}
		filteredProductNamePartners = append(filteredProductNamePartners, filtered...)
		filters = filters[1:]
	}

	filteredPartners = FilterPartnerMegaports(filteredProductNamePartners, filterByProductName)

	if !isFound && filterByProductName && len(filters) < 1 {
		apiPartner := filteredPartners[0]
		state.fromAPIPartnerPort(apiPartner)
		isFound = true
	}

	filteredPartners = FilterPartnerMegaports(filteredProductNamePartners, filterByProductName)

	if !isFound && len(filteredPartners) > 1 && filterByConnectType {
		filtered, err := d.client.PartnerService.FilterPartnerMegaportByConnectType(ctx, filteredPartners, config.ConnectType.ValueString(), true)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Filtering Partner Port",
				"Could not filter partner ports by connect type: "+err.Error(),
			)
		}
		if len(filtered) == 1 {
			state.fromAPIPartnerPort(filtered[0])
			isFound = true
		}
		filterdConnectTypePartners = append(filterdConnectTypePartners, filtered...)
		filters = filters[1:]
	}

	filteredPartners = FilterPartnerMegaports(filterdConnectTypePartners, filterByConnectType)

	if !isFound && filterByConnectType && len(filters) < 1 {
		apiPartner := filteredPartners[0]
		state.fromAPIPartnerPort(apiPartner)
		isFound = true
	}

	if !isFound && len(filteredPartners) > 1 && filterByLocationID {
		filtered, err := d.client.PartnerService.FilterPartnerMegaportByLocationId(ctx, filteredPartners, int(config.LocationID.ValueInt64()))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Filtering Partner Port",
				"Could not filter partner ports by location ID: "+err.Error(),
			)
			return
		}
		if len(filtered) == 1 {
			state.fromAPIPartnerPort(filtered[0])
			isFound = true
		}
		filteredLocationIDPartners = append(filteredLocationIDPartners, filtered...)
		filters = filters[1:]
	}

	filteredPartners = FilterPartnerMegaports(filteredLocationIDPartners, filterByLocationID)

	if !isFound && filterByLocationID && len(filters) < 1 {
		apiPartner := filteredPartners[0]
		state.fromAPIPartnerPort(apiPartner)
		isFound = true
	}

	if !isFound && len(filteredPartners) > 1 && filterByCompanyName {
		filtered, err := d.client.PartnerService.FilterPartnerMegaportByCompanyName(ctx, filteredPartners, config.CompanyName.ValueString(), true)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Filtering Partner Port",
				"Could not filter partner ports by company name: "+err.Error(),
			)
			return
		}
		if len(filtered) == 1 {
			state.fromAPIPartnerPort(partnerPorts[0])
			isFound = true
		}
		filteredCompanyNamePartners = append(filteredCompanyNamePartners, filtered...)
		filters = filters[1:]
	}

	filteredPartners = FilterPartnerMegaports(filteredCompanyNamePartners, filterByCompanyName)

	if !isFound && filterByCompanyName && len(filters) < 1 {
		apiPartner := filteredPartners[0]
		state.fromAPIPartnerPort(apiPartner)
		isFound = true
	}

	if !isFound && len(filteredPartners) > 1 && filterByDiversityZone {
		filtered, err := d.client.PartnerService.FilterPartnerMegaportByDiversityZone(ctx, partnerPorts, config.DiversityZone.ValueString(), true)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Filtering Partner Port",
				"Could not filter partner ports by diversity zone: "+err.Error(),
			)
			return
		}
		if len(filtered) == 1 {
			state.fromAPIPartnerPort(filtered[0])
			isFound = true
		}
		filteredDiversityZonePartners = append(filteredDiversityZonePartners, filtered...)
	}

	filteredPartners = FilterPartnerMegaports(filteredDiversityZonePartners, filterByDiversityZone)

	if !isFound {
		apiPartner := filteredPartners[0]
		state.fromAPIPartnerPort(apiPartner)
		isFound = true
	}

	if !isFound {
		resp.Diagnostics.AddError(
			"No Partner Port Found",
			"No partner port was found.",
		)
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *partnerPortDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}

func (orm *partnerPortModel) fromAPIPartnerPort(port *megaport.PartnerMegaport) {
	orm.ConnectType = types.StringValue(port.ConnectType)
	orm.ProductUID = types.StringValue(port.ProductUID)
	orm.ProductName = types.StringValue(port.ProductName)
	orm.CompanyUID = types.StringValue(port.CompanyUID)
	orm.CompanyName = types.StringValue(port.CompanyName)
	orm.DiversityZone = types.StringValue(port.DiversityZone)
	orm.LocationID = types.Int64Value(int64(port.LocationId))
	orm.Speed = types.Int64Value(int64(port.Speed))
	orm.Rank = types.Int64Value(int64(port.Rank))
	orm.VXCPermitted = types.BoolValue(port.VXCPermitted)
}

func FilterPartnerMegaports(new []*megaport.PartnerMegaport, toFilter bool) []*megaport.PartnerMegaport {
	var filtered []*megaport.PartnerMegaport
	if toFilter {
		filtered = append(filtered, new...)
	}
	return filtered
}
