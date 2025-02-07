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

// locationMVEMovel maps the data source schema data.
type locationMVEMovel struct {
	Sizes             types.List   `tfsdk:"sizes"`
	Details           types.List   `tfsdk:"details"`
	MaxCPUCount       types.Int64  `tfsdk:"max_cpu_count"`
	Version           types.String `tfsdk:"version"`
	Product           types.String `tfsdk:"product"`
	Vendor            types.String `tfsdk:"vendor"`
	VendorDescription types.String `tfsdk:"vendor_description"`
	ID                types.Int64  `tfsdk:"id"`
	ReleaseImage      types.Bool   `tfsdk:"release_image"`
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
		Description: "Location data source for Megaport. Returns a list of data centers where you can order a Megaport, MCR, or MVE. You use the 'id', 'name', or 'site_code' field to identify a specific data center. Please note that names and site_codes of data centers are subject to change (while IDs will remain constant), and the most up to date listing of locations can be retrieved from the Megaport API at GET /v2/locations",
		Attributes: map[string]schema.Attribute{
			"name": &schema.StringAttribute{
				Description: "The name of the location.",
				Optional:    true,
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
				Optional:    true,
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
				Optional:    true,
				Computed:    true,
			},
			"status": &schema.StringAttribute{
				Description: "The status of the location.",
				Computed:    true,
			},
		},
	}
}

func (orm *locationModel) fromAPILocation(ctx context.Context, l *megaport.Location) diag.Diagnostics {
	diags := diag.Diagnostics{}
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

	address, addressDiags := types.MapValueFrom(ctx, types.StringType, l.Address)
	diags = append(diags, addressDiags...)
	orm.Address = address

	products := &locationProductsModel{
		MCR:        types.BoolValue(l.Products.MCR),
		MCRVersion: types.Int64Value(int64(l.Products.MCRVersion)),
	}
	megaportsList, mpListDiags := types.ListValueFrom(ctx, types.Int64Type, l.Products.Megaport)
	diags = append(diags, mpListDiags...)
	products.Megaport = megaportsList

	mcr1List, mcr1ListDiags := types.ListValueFrom(ctx, types.Int64Type, l.Products.MCR1)
	diags = append(diags, mcr1ListDiags...)
	products.MCR1 = mcr1List

	mcr2List, mcr2ListDiags := types.ListValueFrom(ctx, types.Int64Type, l.Products.MCR2)
	diags = append(diags, mcr2ListDiags...)
	products.MCR2 = mcr2List

	mveObjects := []types.Object{}

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
		sizesList, sizesListDiags := types.ListValueFrom(ctx, types.StringType, mve.Sizes)
		diags = append(diags, sizesListDiags...)
		m.Sizes = sizesList
		detailsObjects := []types.Object{}
		for _, detail := range mve.Details {
			d := &locationMVEDetailsModel{
				Size:          types.StringValue(detail.Size),
				Label:         types.StringValue(detail.Label),
				CPUCoreCount:  types.Int64Value(int64(detail.CPUCoreCount)),
				RamGB:         types.Int64Value(int64(detail.RamGB)),
				BandwidthMbps: types.Int64Value(int64(detail.BandwidthMbps)),
			}
			detailObj, detailDiags := types.ObjectValueFrom(ctx, locationMVEDetailsAttrs, d)
			diags = append(diags, detailDiags...)
			detailsObjects = append(detailsObjects, detailObj)
		}
		detailsList, detailsListDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(locationMVEDetailsAttrs), detailsObjects)
		diags = append(diags, detailsListDiags...)
		m.Details = detailsList
		mveObj, mveDiags := types.ObjectValueFrom(ctx, locationMVEAttrs, m)
		diags = append(diags, mveDiags...)
		mveObjects = append(mveObjects, mveObj)
	}
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
			"Either 'id', 'name', or 'site_code' must be set",
			"Either 'id', 'name', or 'site_code' must be set",
		)
		return
	}

	// Prioritize 'name' over 'site_code'
	var location *megaport.Location
	if !state.ID.IsNull() {
		l, err := d.client.LocationService.GetLocationByID(ctx, int(state.ID.ValueInt64()))
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Get location by ID",
				err.Error(),
			)
			return
		}
		location = l
	} else if !state.Name.IsNull() {
		l, err := d.client.LocationService.GetLocationByName(ctx, state.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Get location by name",
				err.Error(),
			)
			return
		}
		location = l
	} else {
		locations, err := d.client.LocationService.ListLocations(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to List locations",
				err.Error(),
			)
			return
		}
		for _, l := range locations {
			if l.SiteCode == state.SiteCode.ValueString() {
				location = l
				break
			}
		}
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
