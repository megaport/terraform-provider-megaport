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
	_ datasource.DataSource              = &productAvailabilityDataSource{}
	_ datasource.DataSourceWithConfigure = &productAvailabilityDataSource{}
)

// productAvailabilityDataSource is the data source implementation.
type productAvailabilityDataSource struct {
	client *megaport.Client
}

// productAvailabilityDataSourceModel maps the data source schema data.
type productAvailabilityDataSourceModel struct {
	CompanyUID types.String               `tfsdk:"company_uid"`
	Products   []productAvailabilityModel `tfsdk:"products"`
}

// productAvailabilityModel maps one available product.
type productAvailabilityModel struct {
	Category    types.String                     `tfsdk:"category"`
	DisplayName types.String                     `tfsdk:"display_name"`
	Variant     types.String                     `tfsdk:"variant"`
	Markets     []productAvailabilityMarketModel `tfsdk:"markets"`
}

// productAvailabilityMarketModel maps one market a product is available in.
type productAvailabilityMarketModel struct {
	Market      types.String `tfsdk:"market"`
	MarketName  types.String `tfsdk:"market_name"`
	MarketOwner types.String `tfsdk:"market_owner"`
	CountryCode types.String `tfsdk:"country_code"`
	Source      types.String `tfsdk:"source"`
}

// NewProductAvailabilityDataSource is a helper function to simplify the provider implementation.
func NewProductAvailabilityDataSource() datasource.DataSource {
	return &productAvailabilityDataSource{}
}

// Metadata returns the data source type name.
func (d *productAvailabilityDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product_availability"
}

// Schema defines the schema for the data source.
func (d *productAvailabilityDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the products your company can order and the markets each one is available in. The company is the one the provider credentials act in, which is the managed account when `managed_account_uid` is set. Filter the list with a `for` expression.",
		Attributes: map[string]schema.Attribute{
			"company_uid": schema.StringAttribute{
				Description: "The UID of the company the availability applies to.",
				Computed:    true,
			},
			"products": schema.ListNestedAttribute{
				Description: "The products the company can order. The API sorts them by category, then variant.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"category": schema.StringAttribute{
							Description: "The product category: `PRODUCT`, `ADDON`, or `BUNDLE`.",
							Computed:    true,
						},
						"display_name": schema.StringAttribute{
							Description: "The human-readable name of the product.",
							Computed:    true,
						},
						"variant": schema.StringAttribute{
							Description: "The specific variant of the product, such as `VXC_TRANSIT` or `PORT`.",
							Computed:    true,
						},
						"markets": schema.ListNestedAttribute{
							Description: "The markets the company can order the product in.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"market": schema.StringAttribute{
										Description: "The market code, such as `MEGAPORT_AUSTRALIA`.",
										Computed:    true,
									},
									"market_name": schema.StringAttribute{
										Description: "The human-readable name of the market.",
										Computed:    true,
									},
									"market_owner": schema.StringAttribute{
										Description: "The name of the Megaport entity that owns the market.",
										Computed:    true,
									},
									"country_code": schema.StringAttribute{
										Description: "The short code of the market's country.",
										Computed:    true,
									},
									// The API lists only available markets, so source is never UNCONFIGURED_MARKET.
									"source": schema.StringAttribute{
										Description: "Where the availability comes from: `MARKET` (the market default) or `OVERRIDE` (a rule for your company).",
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

// Read refreshes the Terraform state with the latest data.
func (d *productAvailabilityDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	company, err := d.client.CompanyService.GetCurrentCompany(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get the current company",
			err.Error(),
		)
		return
	}

	available, err := d.client.ProductAvailabilityService.ListCompanyProductAvailability(ctx, company.CompanyUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to list product availability",
			err.Error(),
		)
		return
	}

	// Start both slices non-nil so an empty response sets an empty list, not null.
	state := productAvailabilityDataSourceModel{
		CompanyUID: types.StringValue(company.CompanyUID),
		Products:   make([]productAvailabilityModel, 0, len(available)),
	}
	for _, p := range available {
		if p == nil {
			continue
		}
		product := productAvailabilityModel{
			Category:    types.StringValue(string(p.Category)),
			DisplayName: types.StringValue(p.DisplayName),
			Variant:     types.StringValue(p.Variant),
			Markets:     make([]productAvailabilityMarketModel, 0, len(p.Markets)),
		}
		for _, m := range p.Markets {
			product.Markets = append(product.Markets, productAvailabilityMarketModel{
				Market:      types.StringValue(m.Market),
				MarketName:  types.StringValue(m.MarketName),
				MarketOwner: types.StringValue(m.MarketOwner),
				CountryCode: types.StringValue(m.CountryCode),
				Source:      types.StringValue(string(m.Source)),
			})
		}
		state.Products = append(state.Products, product)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Configure adds the provider configured client to the data source.
func (d *productAvailabilityDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = data.client
}
