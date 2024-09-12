package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"

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

// mveImageDataSource is the data source implementation.
type mveImageDataSource struct {
	client *megaport.Client
}

// mveImageDetailsModel is the model for the data source.
type mveImageModel struct {
	ID                types.Int64  `tfsdk:"id"`
	Version           types.String `tfsdk:"version"`
	Product           types.String `tfsdk:"product"`
	Vendor            types.String `tfsdk:"vendor"`
	VendorDescription types.String `tfsdk:"vendor_description"`
	ReleaseImage      types.Bool   `tfsdk:"release_image"`
	ProductCode       types.String `tfsdk:"product_code"`
}

// NewMVEImageDataSource is a helper function to simplify the provider implementation.
func NewMVEImageDataSource() datasource.DataSource {
	return &mveImageDataSource{}
}

// Metadata returns the data source type name.
func (d *mveImageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mve_image"
}

// Schema defines the schema for the data source.
func (d *mveImageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "MVE Images",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the MVE Image. The image id returned indicates the software version and key configuration parameters of the image.",
				Optional:    true,
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: "The version of the MVE Image",
				Optional:    true,
				Computed:    true,
			},
			"product": schema.StringAttribute{
				Description: "The product of the MVE Image",
				Optional:    true,
				Computed:    true,
			},
			"vendor": schema.StringAttribute{
				Description: "The vendor of the MVE Image",
				Optional:    true,
				Computed:    true,
			},
			"vendor_description": schema.StringAttribute{
				Description: "The vendor description of the MVE Image",
				Optional:    true,
				Computed:    true,
			},
			"release_image": schema.BoolAttribute{
				Description: "Indicates whether the MVE image is available for selection when ordering an MVE.",
				Optional:    true,
				Computed:    true,
			},
			"product_code": schema.StringAttribute{
				Description: "The product code of the MVE Image",
				Optional:    true,
				Computed:    true,
			},
		},
	}

}

// Read refreshes the Terraform state with the latest data.
func (d *mveImageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config, state mveImageModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	mveImages, listErr := d.client.MVEService.ListMVEImages(ctx)
	if listErr != nil {
		resp.Diagnostics.AddError(
			"Error Reading MVE Images",
			"Could not list MVE Images: "+listErr.Error(),
		)
		return
	}

	if len(mveImages) == 0 {
		resp.Diagnostics.AddError(
			"No MVE Images Found",
			"No MVE Images were found.",
		)
		return
	}

	// create some filters for MVE images
	filters := [](func(*megaport.MVEImage) bool){}
	// add filters for each requested attribute
	if !config.ID.IsNull() {
		filters = append(filters, filterMVEImageByID(int(config.ID.ValueInt64())))
	}
	if !config.Product.IsNull() {
		filters = append(filters, filterMVEImageByProduct(config.Product.ValueString()))
	}
	if !config.Version.IsNull() {
		filters = append(filters, filterMVEImageByVersion(config.Version.ValueString()))
	}
	if !config.Vendor.IsNull() {
		filters = append(filters, filterMVEImageByVendor(config.Vendor.ValueString()))
	}
	if !config.ProductCode.IsNull() {
		filters = append(filters, filterMVEImageByProductCode(config.ProductCode.ValueString()))
	}
	if !config.ReleaseImage.IsNull() {
		filters = append(filters, filterMVEImageByIsReleaseImage(config.ReleaseImage.ValueBool()))
	}

	mveImages = runImageFiltersAndSort(mveImages, filters)

	if len(mveImages) == 0 {
		resp.Diagnostics.AddError(
			"No Matching MVE Images Found",
			"No matching MVE images were found.",
		)
		return
	}

	if len(mveImages) > 1 {
		resp.Diagnostics.AddError(
			"Multiple Matching MVE Images Found",
			"Multiple matching MVE images were found.",
		)
		return
	}

	state.fromAPIMVEImage(mveImages[0])

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *mveImageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (orm *mveImageModel) fromAPIMVEImage(image *megaport.MVEImage) {
	orm.ID = types.Int64Value(int64(image.ID))
	orm.Version = types.StringValue(image.Version)
	orm.Product = types.StringValue(image.Product)
	orm.Vendor = types.StringValue(image.Vendor)
	orm.VendorDescription = types.StringValue(image.VendorDescription)
	orm.ReleaseImage = types.BoolValue(image.ReleaseImage)
	orm.ProductCode = types.StringValue(image.ProductCode)
}

func runImageFiltersAndSort(images []*megaport.MVEImage, filters [](func(*megaport.MVEImage) bool)) []*megaport.MVEImage {
	toReturn := slices.Clone(images)
	// delete all elements not matching filters, this won't have the closure issues https://go.dev/blog/loopvar-preview because we use 1.22
	for _, filter := range filters {
		toReturn = slices.DeleteFunc(toReturn, filter)
	}

	return toReturn
}

func filterMVEImageByID(id int) func(*megaport.MVEImage) bool {
	return func(i *megaport.MVEImage) bool {
		return i.ID != id
	}
}

func filterMVEImageByProduct(product string) func(*megaport.MVEImage) bool {
	return func(i *megaport.MVEImage) bool {
		return !strings.EqualFold(i.Product, product)
	}
}

func filterMVEImageByVersion(version string) func(*megaport.MVEImage) bool {
	return func(i *megaport.MVEImage) bool {
		return !strings.EqualFold(i.Version, version)
	}
}

func filterMVEImageByVendor(vendor string) func(*megaport.MVEImage) bool {
	return func(i *megaport.MVEImage) bool {
		return !strings.EqualFold(i.Vendor, vendor)
	}
}

func filterMVEImageByProductCode(productCode string) func(*megaport.MVEImage) bool {
	return func(i *megaport.MVEImage) bool {
		return !strings.EqualFold(i.ProductCode, productCode)
	}
}

func filterMVEImageByIsReleaseImage(isReleaseImage bool) func(*megaport.MVEImage) bool {
	return func(i *megaport.MVEImage) bool {
		return i.ReleaseImage != isReleaseImage
	}
}
