package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &partnerPortDataSource{}
	_ datasource.DataSourceWithConfigure = &partnerPortDataSource{}

	// mveImageDetailsAttrs is a map of the attributes for the MVE Image data source.
	mveImageDetailsAttrs = map[string]attr.Type{
		"id":                 types.Int64Type,
		"version":            types.StringType,
		"product":            types.StringType,
		"vendor":             types.StringType,
		"vendor_description": types.StringType,
		"release_image":      types.BoolType,
		"product_code":       types.StringType,
		"available_sizes":    types.ListType{ElemType: types.StringType},
	}
)

// mveImageDataSource is the data source implementation.
type mveImageDataSource struct {
	client *megaport.Client
}

// mveImageModel is the model for the data source.
type mveImageModel struct {
	MVEImages          types.List   `tfsdk:"mve_images"`
	IDFilter           types.Int64  `tfsdk:"id_filter"`
	VersionFilter      types.String `tfsdk:"version_filter"`
	ProductFilter      types.String `tfsdk:"product_filter"`
	VendorFilter       types.String `tfsdk:"vendor_filter"`
	ReleaseImageFilter types.Bool   `tfsdk:"release_image_filter"`
	ProductCodeFilter  types.String `tfsdk:"product_code_filter"`
}

// mveImageDetailsModel is the model for the data source.
type mveImageDetailsModel struct {
	ID                types.Int64  `tfsdk:"id"`
	Version           types.String `tfsdk:"version"`
	Product           types.String `tfsdk:"product"`
	Vendor            types.String `tfsdk:"vendor"`
	VendorDescription types.String `tfsdk:"vendor_description"`
	ReleaseImage      types.Bool   `tfsdk:"release_image"`
	ProductCode       types.String `tfsdk:"product_code"`
	AvailableSizes    types.List   `tfsdk:"available_sizes"`
}

// NewMVEImageDataSource is a helper function to simplify the provider implementation.
func NewMVEImageDataSource() datasource.DataSource {
	return &mveImageDataSource{}
}

// Metadata returns the data source type name.
func (d *mveImageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mve_images"
}

// Schema defines the schema for the data source.
func (d *mveImageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "MVE Images",
		Attributes: map[string]schema.Attribute{
			"mve_images": &schema.ListNestedAttribute{
				Description: "List of MVE Images. Returns a list of currently supported MVE images and details for each image, including image ID, version, product, and vendor.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The ID of the MVE Image. The image id returned indicates the software version and key configuration parameters of the image.",
							Computed:    true,
						},
						"version": schema.StringAttribute{
							Description: "The version of the MVE Image",
							Computed:    true,
						},
						"product": schema.StringAttribute{
							Description: "The product of the MVE Image",
							Computed:    true,
						},
						"vendor": schema.StringAttribute{
							Description: "The vendor of the MVE Image",
							Computed:    true,
						},
						"vendor_description": schema.StringAttribute{
							Description: "The vendor description of the MVE Image",
							Computed:    true,
						},
						"release_image": schema.BoolAttribute{
							Description: "Indicates whether the MVE image is available for selection when ordering an MVE.",
							Computed:    true,
						},
						"product_code": schema.StringAttribute{
							Description: "The product code of the MVE Image",
							Computed:    true,
						},
						"available_sizes": schema.ListAttribute{
							Description: "List of available MVE sizes for this image (e.g., 'MVE 2/8', 'MVE 4/16')",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"id_filter": schema.Int64Attribute{
				Description: "Filter the MVE Images by ID",
				Optional:    true,
			},
			"version_filter": schema.StringAttribute{
				Description: "Filter the MVE Images by Version",
				Optional:    true,
			},
			"product_filter": schema.StringAttribute{
				Description: "Filter the MVE Images by Product",
				Optional:    true,
			},
			"vendor_filter": schema.StringAttribute{
				Description: "Filter the MVE Images by Vendor Name",
				Optional:    true,
			},
			"release_image_filter": schema.BoolAttribute{
				Description: "Filter the MVE Images by Release Image",
				Optional:    true,
			},
			"product_code_filter": schema.StringAttribute{
				Description: "Filter the MVE Images by Product Code",
				Optional:    true,
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
	if !config.IDFilter.IsNull() {
		filters = append(filters, filterMVEImageByID(int(config.IDFilter.ValueInt64())))
	}
	if !config.ProductFilter.IsNull() {
		filters = append(filters, filterMVEImageByProduct(config.ProductFilter.ValueString()))
	}
	if !config.VersionFilter.IsNull() {
		filters = append(filters, filterMVEImageByVersion(config.VersionFilter.ValueString()))
	}
	if !config.VendorFilter.IsNull() {
		filters = append(filters, filterMVEImageByVendor(config.VendorFilter.ValueString()))
	}
	if !config.ProductCodeFilter.IsNull() {
		filters = append(filters, filterMVEImageByProductCode(config.ProductCodeFilter.ValueString()))
	}
	if !config.ReleaseImageFilter.IsNull() {
		filters = append(filters, filterMVEImageByIsReleaseImage(config.ReleaseImageFilter.ValueBool()))
	}

	mveImages = runImageFiltersAndSort(mveImages, filters)

	if len(mveImages) == 0 {
		resp.Diagnostics.AddError(
			"No Matching MVE Images Found",
			"No matching MVE images were found.",
		)
		return
	}

	imageObjects := []types.Object{}

	for _, image := range mveImages {
		imageDetailsModel := &mveImageDetailsModel{}
		imageDetailsModel.fromAPIMVEImage(image)
		imageDetailsObject, diags := types.ObjectValueFrom(ctx, mveImageDetailsAttrs, &imageDetailsModel)
		resp.Diagnostics.Append(diags...)
		imageObjects = append(imageObjects, imageDetailsObject)
	}
	imageList, imageListDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(mveImageDetailsAttrs), imageObjects)
	resp.Diagnostics.Append(imageListDiags...)
	state.MVEImages = imageList

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

func (orm *mveImageDetailsModel) fromAPIMVEImage(image *megaport.MVEImage) {
	orm.ID = types.Int64Value(int64(image.ID))
	orm.Version = types.StringValue(image.Version)
	orm.Product = types.StringValue(image.Product)
	vendorStr := image.Vendor
	if strings.EqualFold(vendorStr, "PALO ALTO") {
		vendorStr = "PALO_ALTO"
	}
	orm.Vendor = types.StringValue(vendorStr)
	orm.VendorDescription = types.StringValue(image.VendorDescription)
	orm.ReleaseImage = types.BoolValue(image.ReleaseImage)
	orm.ProductCode = types.StringValue(image.ProductCode)

	// Convert AvailableSizes from []string to types.List
	if len(image.AvailableSizes) > 0 {
		sizeValues := make([]attr.Value, len(image.AvailableSizes))
		for i, size := range image.AvailableSizes {
			sizeValues[i] = types.StringValue(size)
		}
		orm.AvailableSizes, _ = types.ListValue(types.StringType, sizeValues)
	} else {
		orm.AvailableSizes = types.ListNull(types.StringType)
	}
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
		if strings.EqualFold(vendor, "PALO_ALTO") {
			vendor = "PALO ALTO"
		}
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
