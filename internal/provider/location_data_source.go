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
	_ datasource.DataSource              = &locationDataSource{}
	_ datasource.DataSourceWithConfigure = &locationDataSource{}

	locationProductsAttrs = map[string]attr.Type{
		"mcr":         types.BoolType,
		"mcr_version": types.Int64Type,
		"megaport":    types.ListType{}.WithElementType(types.Int64Type),
		"mve":         types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(locationMVEAttrs)),
		"mcr1":        types.ListType{}.WithElementType(types.Int64Type),
		"mcr2":        types.ListType{}.WithElementType(types.Int64Type),
	}

	locationMVEAttrs = map[string]attr.Type{
		"sizes":              types.ListType{}.WithElementType(types.StringType),
		"details":            types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(locationMVEDetailsAttrs)),
		"max_cpu_count":      types.Int64Type,
		"version":            types.StringType,
		"product":            types.StringType,
		"vendor":             types.StringType,
		"vendor_description": types.StringType,
		"id":                 types.Int64Type,
		"release_image":      types.BoolType,
	}

	locationMVEDetailsAttrs = map[string]attr.Type{
		"size":           types.StringType,
		"label":          types.StringType,
		"cpu_core_count": types.Int64Type,
		"ram_gb":         types.Int64Type,
		"bandwidth_mbps": types.Int64Type,
	}
)

// locationDataSource is the data source implementation.
type locationDataSource struct {
	client *megaport.Client
}

// locationModel maps the data source schema data.
type locationModel struct {
	Name             types.String  `tfsdk:"name"`
	Country          types.String  `tfsdk:"country"`
	LiveDate         types.String  `tfsdk:"live_date"`
	SiteCode         types.String  `tfsdk:"site_code"`
	NetworkRegion    types.String  `tfsdk:"network_region"`
	Address          types.Map     `tfsdk:"address"`
	Campus           types.String  `tfsdk:"campus"`
	Latitude         types.Float64 `tfsdk:"latitude"`
	Longitude        types.Float64 `tfsdk:"longitude"`
	Products         types.Object  `tfsdk:"products"`
	Market           types.String  `tfsdk:"market"`
	Metro            types.String  `tfsdk:"metro"`
	VRouterAvailable types.Bool    `tfsdk:"v_router_available"`
	ID               types.Int64   `tfsdk:"id"`
	Status           types.String  `tfsdk:"status"`
}

// locationProductsModel maps the data source schema data.
type locationProductsModel struct {
	MCR        types.Bool  `tfsdk:"mcr"`
	MCRVersion types.Int64 `tfsdk:"mcr_version"`
	Megaport   types.List  `tfsdk:"megaport"`
	MVE        types.List  `tfsdk:"mve"`
	MCR1       types.List  `tfsdk:"mcr1"`
	MCR2       types.List  `tfsdk:"mcr2"`
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
		Description: "Location data source for Megaport. Returns a list of data centers where you can order a Megaport, MCR, or MVE. While you can use 'id' or 'name' field to identify a specific data center, it is strongly recommended to use 'id' for consistent results. Location names can change over time, while IDs remain constant. Using the location ID ensures deterministic behavior in your Terraform configurations. The most up to date listing of locations can be retrieved from the Megaport API at GET /v3/locations",
		Attributes: map[string]schema.Attribute{
			"id": &schema.Int64Attribute{
				Description: "The ID of the location. Using ID is strongly recommended as the most reliable way to identify locations since IDs remain constant, unlike names and site codes which can change.",
				Optional:    true,
				Computed:    true,
			},
			"name": &schema.StringAttribute{
				Description: "The name of the location. Note that location names can change over time, which may lead to non-deterministic behavior. For consistent results, use the location ID instead.",
				Optional:    true,
				Computed:    true,
			},
			"site_code": &schema.StringAttribute{
				Description: "DEPRECATED: The site_code field is no longer available in the v3 locations API and will be removed in a future version. Use the location ID instead. Filtering by site_code is no longer supported.",
				Optional:    true,
				Computed:    true,
			},
			"country": &schema.StringAttribute{
				Description: "The country of the location.",
				Computed:    true,
			},
			"live_date": &schema.StringAttribute{
				Description: "DEPRECATED: The live_date field is no longer available in the v3 locations API and will be removed in a future version.",
				Computed:    true,
			},
			"network_region": &schema.StringAttribute{
				Description: "DEPRECATED: The network_region field is no longer available in the v3 locations API and will be removed in a future version.",
				Computed:    true,
			},
			"address": &schema.MapAttribute{
				Description: "The address of the location.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"campus": &schema.StringAttribute{
				Description: "DEPRECATED: The campus field is no longer available in the v3 locations API and will be removed in a future version.",
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
				Description: "DEPRECATED: The v_router_available field is no longer available in the v3 locations API and will be removed in a future version.",
				Computed:    true,
			},
			"status": &schema.StringAttribute{
				Description: "The status of the location.",
				Computed:    true,
			},
		},
	}
}

func (orm *locationModel) fromAPILocation(ctx context.Context, l *megaport.LocationV3) diag.Diagnostics {
	diags := diag.Diagnostics{}
	orm.Name = types.StringValue(l.Name)
	orm.Country = types.StringValue(l.Address.Country)

	// Deprecated fields - set to empty/zero values
	orm.LiveDate = types.StringNull()
	orm.SiteCode = types.StringNull()
	orm.NetworkRegion = types.StringNull()
	orm.Campus = types.StringNull()
	orm.VRouterAvailable = types.BoolNull()

	orm.Latitude = types.Float64Value(l.Latitude)
	orm.Longitude = types.Float64Value(l.Longitude)
	orm.Market = types.StringValue(l.Market)
	orm.Metro = types.StringValue(l.Metro)
	orm.ID = types.Int64Value(int64(l.ID))
	orm.Status = types.StringValue(l.Status)

	// Convert v3 address structure to map
	addressMap := map[string]string{
		"street":   l.Address.Street,
		"suburb":   l.Address.Suburb,
		"city":     l.Address.City,
		"state":    l.Address.State,
		"postcode": l.Address.Postcode,
		"country":  l.Address.Country,
	}
	address, addressDiags := types.MapValueFrom(ctx, types.StringType, addressMap)
	diags = append(diags, addressDiags...)
	orm.Address = address

	// Convert v3 diversity zones to legacy products structure
	products := &locationProductsModel{
		MCR:        types.BoolValue(l.HasMCRSupport()),
		MCRVersion: types.Int64Null(), // Not available in v3
	}

	megaportSpeeds := l.GetMegaportSpeeds()
	megaportsList, mpListDiags := types.ListValueFrom(ctx, types.Int64Type, megaportSpeeds)
	diags = append(diags, mpListDiags...)
	products.Megaport = megaportsList

	// MCR1 not available in v3 - set to empty list
	mcr1List, mcr1ListDiags := types.ListValueFrom(ctx, types.Int64Type, []int{})
	diags = append(diags, mcr1ListDiags...)
	products.MCR1 = mcr1List

	// MCR2 speeds from v3
	mcrSpeeds := l.GetMCRSpeeds()
	mcr2List, mcr2ListDiags := types.ListValueFrom(ctx, types.Int64Type, mcrSpeeds)
	diags = append(diags, mcr2ListDiags...)
	products.MCR2 = mcr2List

	// MVE not available in same format in v3 - set to empty list
	mveObjects := []types.Object{}
	mveList, mveListDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(locationMVEAttrs), mveObjects)
	diags = append(diags, mveListDiags...)
	products.MVE = mveList

	productsObj, productsDiags := types.ObjectValueFrom(ctx, locationProductsAttrs, products)
	diags = append(diags, productsDiags...)
	orm.Products = productsObj

	return diags
}

// Read refreshes the Terraform state with the latest data.
func (d *locationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state locationModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() && state.Name.IsNull() && state.SiteCode.IsNull() {
		resp.Diagnostics.AddError(
			"Either 'id' or 'name' must be set",
			"Either 'id' or 'name' must be set. The 'site_code' field is deprecated and no longer supported.",
		)
		return
	}

	// Return error if site_code is used for filtering
	if !state.SiteCode.IsNull() && state.ID.IsNull() && state.Name.IsNull() {
		resp.Diagnostics.AddError(
			"site_code filtering is no longer supported",
			"The site_code field is deprecated and no longer available in the v3 locations API. Please use 'id' or 'name' instead.",
		)
		return
	}

	// Prioritize 'id' over 'name'
	var location *megaport.LocationV3
	if !state.ID.IsNull() {
		l, err := d.client.LocationService.GetLocationByIDV3(ctx, int(state.ID.ValueInt64()))
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Get location by ID",
				err.Error(),
			)
			return
		}
		location = l
	} else if !state.Name.IsNull() {
		l, err := d.client.LocationService.GetLocationByNameV3(ctx, state.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Get location by name",
				err.Error(),
			)
			return
		}
		location = l
	}

	if location == nil {
		resp.Diagnostics.AddError(
			"Location not found",
			"Location not found",
		)
		return
	}

	apiDiags := state.fromAPILocation(ctx, location)
	resp.Diagnostics.Append(apiDiags...)

	// Set state
	diags = resp.State.Set(ctx, &state)
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

	data, ok := req.ProviderData.(*megaportProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *megaportProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	client := data.client
	d.client = client
}
