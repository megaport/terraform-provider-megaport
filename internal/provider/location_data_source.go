package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &locationDataSource{}
	_ datasource.DataSourceWithConfigure = &locationDataSource{}
)

// locationDataSource is the data source implementation.
type locationDataSource struct {
	client *megaport.Client
}

type locationDataSourceModel struct {
	Locations []*locationModel `tfsdk:"locations"`
}

// locationDataSourceModel maps the data source schema data.
type locationModel struct {
	Name             types.String            `tfsdk:"name"`
	Country          types.String            `tfsdk:"country"`
	LiveDate         types.String            `tfsdk:"live_date"`
	SiteCode         types.String            `tfsdk:"site_code"`
	NetworkRegion    types.String            `tfsdk:"network_region"`
	Address          map[string]types.String `tfsdk:"address"`
	Campus           types.String            `tfsdk:"campus"`
	Latitude         types.Float64           `tfsdk:"latitude"`
	Longitude        types.Float64           `tfsdk:"longitude"`
	Products         *locationProductsModel  `tfsdk:"products"`
	Market           types.String            `tfsdk:"market"`
	Metro            types.String            `tfsdk:"metro"`
	VRouterAvailable types.Bool              `tfsdk:"v_router_available"`
	ID               types.Int64             `tfsdk:"id"`
	Status           types.String            `tfsdk:"status"`
}

// locationProductsModel maps the data source schema data.
type locationProductsModel struct {
	MCR        bool                `tfsdk:"mcr"`
	MCRVersion int                 `tfsdk:"mcr_version"`
	Megaport   []int               `tfsdk:"megaport"`
	MVE        []*locationMVEMovel `tfsdk:"mve"`
	MCR1       []int               `tfsdk:"mcr1"`
	MCR2       []int               `tfsdk:"mcr2"`
}

// locationMVEMovel maps the data source schema data.
type locationMVEMovel struct {
	Sizes             []types.String             `tfsdk:"sizes"`
	Details           []*locationMVEDetailsModel `tfsdk:"details"`
	MaxCPUCount       types.Int64                `tfsdk:"max_cpu_count"`
	Version           types.String               `tfsdk:"version"`
	Product           types.String               `tfsdk:"product"`
	Vendor            types.String               `tfsdk:"vendor"`
	VendorDescription types.String               `tfsdk:"vendor_description"`
	ID                types.Int64                `tfsdk:"id"`
	ReleaseImage      types.Bool                 `tfsdk:"release_image"`
}

// locationMVEDetailsModel maps the data source schema data.
type locationMVEDetailsModel struct {
	Size          types.String `tfsdk:"size"`
	Label         types.String `tfsdk:"label"`
	CPUCoreCount  types.Int64  `tfsdk:"cpu_core_count"`
	RamGB         types.Int64  `tfsdk:"ram_gb"`
	BandwidthMbps types.Int64  `tfsdk:"bandwidth_mbps"`
}

// NewlocationDataSource is a helper function to simplify the provider implementation.
func NewlocationDataSource() datasource.DataSource {
	return &locationDataSource{}
}

// Metadata returns the data source type name.
func (d *locationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_location"
}

// Schema defines the schema for the data source.
func (d *locationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Locations Data Source",
		Attributes: map[string]schema.Attribute{
			"locations": &schema.ListNestedAttribute{
				Description: "List of locations.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": &schema.StringAttribute{
							Description: "The name of the location.",
							Computed:    true,
						},
						"country": &schema.StringAttribute{
							Description: "The country of the location.",
							Computed:    true,
						},
						"live_date": &schema.StringAttribute{
							Description: "The live date of the location.",
							Computed:    true,
						},
						"site_code": &schema.StringAttribute{
							Description: "The site code of the location.",
							Computed:    true,
						},
						"network_region": &schema.StringAttribute{
							Description: "The network region of the location.",
							Computed:    true,
						},
						"address": &schema.MapAttribute{
							Description: "The address of the location.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"campus": &schema.StringAttribute{
							Description: "The campus of the location.",
							Computed:    true,
						},
						"latitude": &schema.Float64Attribute{
							Description: "The latitude of the location.",
							Computed:    true,
						},
						"longitude": &schema.Float64Attribute{
							Description: "The longitude of the location.",
							Computed:    true,
						},
						"products": &schema.SingleNestedAttribute{
							Description: "The products available in the location.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"mcr": &schema.BoolAttribute{
									Description: "The MCR availability of the location.",
									Computed:    true,
								},
								"mcr_version": &schema.Int64Attribute{
									Description: "The MCR version available at the location.",
									Computed:    true,
								},
								"megaport": &schema.ListAttribute{
									Description: "The Megaport availability of the location.",
									Computed:    true,
									ElementType: types.Int64Type,
								},
								"mve": &schema.ListNestedAttribute{
									Description: "The MVE availability of the location.",
									Computed:    true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"sizes": &schema.ListAttribute{
												Description: "The sizes available in the location.",
												Computed:    true,
												ElementType: types.StringType,
											},
											"details": &schema.ListNestedAttribute{
												Description: "The details of the MVE available in the location.",
												Computed:    true,
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"size": &schema.StringAttribute{
															Description: "The size of the MVE available in the location.",
															Computed:    true,
														},
														"label": &schema.StringAttribute{
															Description: "The label of the MVE available in the location.",
															Computed:    true,
														},
														"cpu_core_count": &schema.Int64Attribute{
															Description: "The CPU core count of the MVE available in the location.",
															Computed:    true,
														},
														"ram_gb": &schema.Int64Attribute{
															Description: "The RAM GB of the MVE available in the location.",
															Computed:    true,
														},
														"bandwidth_mbps": &schema.Int64Attribute{
															Description: "The bandwidth Mbps of the MVE available in the location.",
															Computed:    true,
														},
													},
												},
											},
											"max_cpu_count": &schema.Int64Attribute{
												Description: "The maximum CPU count of the MVE available in the location.",
												Computed:    true,
											},
											"version": &schema.StringAttribute{
												Description: "The version of the MVE available in the location.",
												Computed:    true,
											},
											"product": &schema.StringAttribute{
												Description: "The product of the MVE available in the location.",
												Computed:    true,
											},
											"vendor": &schema.StringAttribute{
												Description: "The vendor of the MVE available in the location.",
												Computed:    true,
											},
											"vendor_description": &schema.StringAttribute{
												Description: "The vendor description of the MVE available in the location.",
												Computed:    true,
											},
											"id": &schema.Int64Attribute{
												Description: "The ID of the MVE available in the location.",
												Computed:    true,
											},
											"release_image": &schema.BoolAttribute{
												Description: "Whether there is a release image or not.",
												Computed:    true,
											},
										},
									},
								},
								"mcr1": &schema.ListAttribute{
									Description: "The MCR1 bandwidth availability of the location.",
									Computed:    true,
									ElementType: types.Int64Type,
								},
								"mcr2": &schema.ListAttribute{
									Description: "The MCR2 bandwidth availability of the location.",
									Computed:    true,
									ElementType: types.Int64Type,
								},
							},
						},
						"market": &schema.StringAttribute{
							Description: "The market of the location.",
							Computed:    true,
						},
						"metro": &schema.StringAttribute{
							Description: "The metro of the location.",
							Computed:    true,
						},
						"v_router_available": &schema.BoolAttribute{
							Description: "The vRouter availability of the location.",
							Computed:    true,
						},
						"id": &schema.Int64Attribute{
							Description: "The ID of the location.",
							Computed:    true,
						},
						"status": &schema.StringAttribute{
							Description: "The status of the location.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (orm *locationModel) fromAPILocation(l *megaport.Location) {
	orm.Name = types.StringValue(l.Name)
	orm.Country = types.StringValue(l.Country)
	if l.LiveDate != nil {
		orm.LiveDate = types.StringValue(l.LiveDate.Format(time.RFC850))
	}
	orm.SiteCode = types.StringValue(l.SiteCode)
	orm.NetworkRegion = types.StringValue(l.NetworkRegion)
	orm.Campus = types.StringValue(l.Campus)
	orm.Latitude = types.Float64Value(l.Latitude)
	orm.Longitude = types.Float64Value(l.Longitude)
	orm.Market = types.StringValue(l.Market)
	orm.Metro = types.StringValue(l.Metro)
	orm.VRouterAvailable = types.BoolValue(l.VRouterAvailable)
	orm.ID = types.Int64Value(int64(l.ID))
	orm.Status = types.StringValue(l.Status)
	orm.Address = make(map[string]types.String)

	for k, v := range l.Address {
		orm.Address[k] = types.StringValue(v)
	}

	products := &locationProductsModel{
		MCR:        l.Products.MCR,
		MCRVersion: l.Products.MCRVersion,
		Megaport:   l.Products.Megaport,
		MCR1:       l.Products.MCR1,
		MCR2:       l.Products.MCR2,
	}
	for _, mve := range l.Products.MVE {
		m := &locationMVEMovel{
			MaxCPUCount:       types.Int64Value(int64(mve.MaxCPUCount)),
			Version:           types.StringValue(mve.Version),
			Product:           types.StringValue(mve.Product),
			Vendor:            types.StringValue(mve.Vendor),
			VendorDescription: types.StringValue(mve.VendorDescription),
			ID:                types.Int64Value(int64(mve.ID)),
			ReleaseImage:      types.BoolValue(mve.ReleaseImage),
		}
		for _, detail := range mve.Details {
			d := &locationMVEDetailsModel{
				Size:          types.StringValue(detail.Size),
				Label:         types.StringValue(detail.Label),
				CPUCoreCount:  types.Int64Value(int64(detail.CPUCoreCount)),
				RamGB:         types.Int64Value(int64(detail.RamGB)),
				BandwidthMbps: types.Int64Value(int64(detail.BandwidthMbps)),
			}
			m.Details = append(m.Details, d)
		}
		products.MVE = append(products.MVE, m)
	}
	orm.Products = products
}

// Read refreshes the Terraform state with the latest data.
func (d *locationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state locationDataSourceModel

	locations, err := d.client.LocationService.ListLocations(ctx)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read location",
			err.Error(),
		)
		return
	}

	for _, location := range locations {
		locationState := &locationModel{}
		locationState.fromAPILocation(location)
		state.Locations = append(state.Locations, locationState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *locationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
