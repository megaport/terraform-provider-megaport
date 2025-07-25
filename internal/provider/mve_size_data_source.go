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

	// mveSizeDetailsAttrs is a map of the attributes for the MVE Size data source.
	mveSizeDetailsAttrs = map[string]attr.Type{
		"size":           types.StringType,
		"label":          types.StringType,
		"cpu_core_count": types.Int64Type,
		"ram_gb":         types.Int64Type,
	}
)

// mveSizeDataSource is the data source implementation.
type mveSizeDataSource struct {
	client *megaport.Client
}

// mveSizeModel is the model for the data source.
type mveSizeModel struct {
	MVESizes types.List `tfsdk:"mve_sizes"`
}

// mveSizeDetailsModel is the model for the data source.
type mveSizeDetailsModel struct {
	Size         types.String `json:"size"`
	Label        types.String `json:"label"`
	CPUCoreCount types.Int64  `json:"cpu_core_count"`
	RamGB        types.Int64  `json:"ram_gb"`
}

// NewMVESizeDataSource is a helper function to simplify the provider implementation.
func NewMVESizeDataSource() datasource.DataSource {
	return &mveSizeDataSource{}
}

// Metadata returns the data source type name.
func (d *mveSizeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mve_sizes"
}

// Schema defines the schema for the data source.
func (d *mveSizeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "MVE Size Data Source. Returns a list of currently available MVE sizes and details for each size.",
		Attributes: map[string]schema.Attribute{
			"mve_sizes": &schema.ListNestedAttribute{
				Description: "List of MVE Sizes",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.StringAttribute{
							Description: "Size of the MVE",
							Computed:    true,
						},
						"label": schema.StringAttribute{
							Description: "Label of the MVE Size. The compute sizes are 2/8, 4/16, 8/32, and 12/48, where the first number is the CPU and the second number is the GB of available RAM. Each size has 4 GB of RAM for every vCPU allocated.",
							Computed:    true,
						},
						"cpu_core_count": schema.Int64Attribute{
							Description: "Number of CPU Cores.",
							Computed:    true,
						},
						"ram_gb": schema.StringAttribute{
							Description: "Amount of RAM in GB.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *mveSizeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config, state mveSizeModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	mveSizes, listErr := d.client.MVEService.ListAvailableMVESizes(ctx)
	if listErr != nil {
		resp.Diagnostics.AddError(
			"Error Reading MVE Images",
			"Could not list MVE Images: "+listErr.Error(),
		)
		return
	}

	if len(mveSizes) == 0 {
		resp.Diagnostics.AddError(
			"No Available MVE Sizes Found",
			"No Available MVE Sizes were found.",
		)
		return
	}

	sizeObjects := []types.Object{}

	for _, size := range mveSizes {
		sizeDetailsModel := &mveSizeDetailsModel{}
		sizeDetailsModel.fromAPIMVESize(size)
		sizeDetailsObject, objectDiags := types.ObjectValueFrom(ctx, mveSizeDetailsAttrs, &sizeDetailsModel)
		resp.Diagnostics.Append(objectDiags...)
		sizeObjects = append(sizeObjects, sizeDetailsObject)
	}
	sizeList, sizeListDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(mveSizeDetailsAttrs), sizeObjects)
	resp.Diagnostics.Append(sizeListDiags...)
	state.MVESizes = sizeList

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *mveSizeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (orm *mveSizeDetailsModel) fromAPIMVESize(size *megaport.MVESize) {
	orm.Size = types.StringValue(size.Size)
	orm.Label = types.StringValue(size.Label)
	orm.CPUCoreCount = types.Int64Value(int64(size.CPUCoreCount))
	orm.RamGB = types.Int64Value(int64(size.RamGB))
}
