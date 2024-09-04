package provider

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

// partnerPortModel maps the data source schema data.
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
		Description: "Partner Port Data Source. Returns the interfaces Megaport has with cloud service providers.",
		Attributes: map[string]schema.Attribute{
			"connect_type": &schema.StringAttribute{
				Description: "The type of connection for the partner port. Filters the locations based on the cloud providers, such as AWS (for Hosted VIF), AWSHC (for Hosted Connection), AZURE, GOOGLE, ORACLE, OUTSCALE, and IBM. Use TRANSIT fto display Ports that support a Megaport Internet connection. Use FRANCEIX to display France-IX Ports that you can connect to.",
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
				Validators: []validator.String{
					stringvalidator.OneOf("red", "blue"),
				},
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
				Description: "Whether VXCs are permitted on the partner port. If false, you can not create a VXC on this port. If true, you can create a VXC on this port.",
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

	partnerPorts, listErr := d.client.PartnerService.ListPartnerMegaports(ctx)
	if listErr != nil {
		resp.Diagnostics.AddError(
			"Error listing partner ports",
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

	// create some filters for partner ports, by default we remove ports where VXCs are not permitted
	filters := [](func(*megaport.PartnerMegaport) bool){filterByVXCPermitted(true)}

	// add filters for each requested attribute
	if !config.ProductName.IsNull() {
		filters = append(filters, filterByProductName(config.ProductName.ValueString()))
	}
	if !config.ConnectType.IsNull() {
		filters = append(filters, filterByConnectType(config.ConnectType.ValueString()))
	}
	if !config.LocationID.IsNull() {
		filters = append(filters, filterByLocationID(int(config.LocationID.ValueInt64())))
	}
	if !config.CompanyName.IsNull() {
		filters = append(filters, filterByCompanyName(config.CompanyName.ValueString()))
	}
	if !config.DiversityZone.IsNull() {
		filters = append(filters, filterByDiversityZone(config.DiversityZone.ValueString()))
	}

	// run the collected filters
	partnerPorts = runFiltersAndSort(partnerPorts, filters)

	if len(partnerPorts) == 0 {
		resp.Diagnostics.AddError(
			"No Matching Partner Ports Found",
			"No matching partner ports were found.",
		)
		return
	}

	if len(partnerPorts) > 1 {
		resp.Diagnostics.AddWarning(
			"More Than 1 Matching Partner Port Was Found",
			"There was more than 1 matching partner port for the search criteria, chose highest ranked port. Try narrowing your search criteria.",
		)
	}

	// pick the first matching port
	state.fromAPIPartnerPort(partnerPorts[0])

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

func runFiltersAndSort(ports []*megaport.PartnerMegaport, filters [](func(*megaport.PartnerMegaport) bool)) []*megaport.PartnerMegaport {
	toReturn := slices.Clone(ports)
	// delete all elements not matching filters, this won't have the closure issues https://go.dev/blog/loopvar-preview because we use 1.22
	for _, filter := range filters {
		toReturn = slices.DeleteFunc(toReturn, filter)
	}

	// sort remaining ports by rank
	slices.SortFunc(ports, func(a *megaport.PartnerMegaport, b *megaport.PartnerMegaport) int {
		if n := cmp.Compare(a.Rank, b.Rank); n != 0 {
			return n
		}

		// If ranks are equal, order by name
		return cmp.Compare(a.ProductName, b.ProductName)
	})

	return toReturn
}

func filterByVXCPermitted(permitted bool) func(*megaport.PartnerMegaport) bool {
	return func(pm *megaport.PartnerMegaport) bool {
		return pm.VXCPermitted != permitted
	}
}

func filterByProductName(name string) func(*megaport.PartnerMegaport) bool {
	return func(pm *megaport.PartnerMegaport) bool {
		return !strings.EqualFold(pm.ProductName, name)
	}
}

func filterByConnectType(connectType string) func(*megaport.PartnerMegaport) bool {
	return func(pm *megaport.PartnerMegaport) bool {
		return !strings.EqualFold(pm.ConnectType, connectType)
	}
}

func filterByLocationID(locationID int) func(*megaport.PartnerMegaport) bool {
	return func(pm *megaport.PartnerMegaport) bool {
		return pm.LocationId != locationID
	}
}

func filterByCompanyName(companyName string) func(*megaport.PartnerMegaport) bool {
	return func(pm *megaport.PartnerMegaport) bool {
		return !strings.EqualFold(pm.CompanyName, companyName)
	}
}

func filterByDiversityZone(diversityZone string) func(*megaport.PartnerMegaport) bool {
	return func(pm *megaport.PartnerMegaport) bool {
		return !strings.EqualFold(pm.DiversityZone, diversityZone)
	}
}
