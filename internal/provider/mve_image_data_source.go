package provider

import (
	"context"
	"fmt"

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
		"id":                 types.StringType,
		"version":            types.StringType,
		"product":            types.StringType,
		"vendor":             types.StringType,
		"vendor_description": types.StringType,
		"release_image":      types.BoolType,
		"product_code":       types.StringType,
	}
)

// mveImageDataSource is the data source implementation.
type mveImageDataSource struct {
	client *megaport.Client
}

// mveImageModel is the model for the data source.
type mveImageModel struct {
	MVEImages types.List `tfsdk:"mve_images"`
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
						"id": schema.StringAttribute{
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
					},
				},
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

func (orm *mveImageDetailsModel) fromAPIMVEImage(image *megaport.MVEImage) {
	orm.ID = types.Int64Value(int64(image.ID))
	orm.Version = types.StringValue(image.Version)
	orm.Product = types.StringValue(image.Product)
	orm.Vendor = types.StringValue(image.Vendor)
	orm.VendorDescription = types.StringValue(image.VendorDescription)
	orm.ReleaseImage = types.BoolValue(image.ReleaseImage)
	orm.ProductCode = types.StringValue(image.ProductCode)
}
