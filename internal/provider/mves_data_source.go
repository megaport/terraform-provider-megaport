package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces
var _ datasource.DataSource = &mvesDataSource{}

// mvesDataSource is the data source implementation.
type mvesDataSource struct {
	client *megaport.Client
}

// mvesModel maps the data source schema data.
type mvesModel struct {
	UIDs   types.List    `tfsdk:"uids"`
	Filter []filterModel `tfsdk:"filter"`
	Tags   types.Map     `tfsdk:"tags"`
}

// NewMVEsDataSource creates a new MVEs data source.
func NewMVEsDataSource() datasource.DataSource {
	return &mvesDataSource{}
}

// Metadata returns the data source type name.
func (d *mvesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mves"
}

// Schema defines the schema for the data source.
func (d *mvesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a list of MVE UIDs matching the specified filters.",
		Attributes: map[string]schema.Attribute{
			"uids": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of MVE UIDs that match the specified criteria.",
			},
			"tags": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Map of resource tags, each pair of which must exactly match a pair on the desired MVEs.",
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.ListNestedBlock{
				Description: "Custom filter block to select MVEs.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Name of the field to filter by. Available filters: name, vendor, size, location-id, provisioning-status, market, company-name, cost-centre, vxc-permitted.",
						},
						"values": schema.ListAttribute{
							ElementType: types.StringType,
							Required:    true,
							Description: "Set of values that are accepted for the given field.",
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *mvesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*megaport.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *megaport.Client, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *mvesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data mvesModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get all MVEs from the API
	mves, err := d.client.MVEService.ListMVEs(ctx, &megaport.ListMVEsRequest{
		IncludeInactive: false,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing MVEs",
			fmt.Sprintf("Unable to list MVEs: %v", err),
		)
		return
	}

	// Apply filtering
	filteredMVEs, filterDiags := d.filterMVEs(ctx, mves, data)
	resp.Diagnostics.Append(filterDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract MVE IDs
	mveUIDs := make([]string, 0, len(filteredMVEs))
	for _, mve := range filteredMVEs {
		mveUIDs = append(mveUIDs, mve.UID)
	}

	// Set the MVE IDs in the model
	mveUIDsList, diags := types.ListValueFrom(ctx, types.StringType, mveUIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.UIDs = mveUIDsList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// filterMVEs applies filters to the list of MVEs.
func (d *mvesDataSource) filterMVEs(ctx context.Context, mves []*megaport.MVE, data mvesModel) ([]*megaport.MVE, diag.Diagnostics) {
	var diags diag.Diagnostics

	// If no filters or tags are provided, return all MVEs
	if len(data.Filter) == 0 && data.Tags.IsNull() {
		return mves, diags
	}

	var filteredMVEs []*megaport.MVE

	// Handle tag filtering
	var tagFilters map[string]string
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		diags.Append(data.Tags.ElementsAs(ctx, &tagFilters, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}

	// Process each MVE
	for _, mve := range mves {
		// Check tag filters if any
		if len(tagFilters) > 0 {
			// Get MVE tags
			mveTags, err := d.client.MVEService.ListMVEResourceTags(ctx, mve.UID)
			if err != nil {
				diags.AddWarning(
					"Error fetching MVE tags",
					fmt.Sprintf("Unable to fetch tags for MVE %s: %v", mve.UID, err),
				)
				continue
			}

			// Check if MVE matches all tag filters
			if !matchesTags(mveTags, tagFilters) {
				continue
			}
		}

		// Check custom filters
		match, filterDiags := matchesMVEFilters(ctx, mve, data.Filter)
		diags.Append(filterDiags...)
		if !match {
			continue
		}

		filteredMVEs = append(filteredMVEs, mve)
	}

	return filteredMVEs, diags
}

// matchesMVEFilters checks if an MVE matches the custom filters.
func matchesMVEFilters(ctx context.Context, mve *megaport.MVE, filters []filterModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	for _, filter := range filters {
		var filterValues []string
		valDiags := filter.Values.ElementsAs(ctx, &filterValues, false)
		diags.Append(valDiags...)
		if diags.HasError() {
			return false, diags
		}

		name := filter.Name.ValueString()
		match := false

		switch name {
		case "name":
			match = matchesNamePattern(filterValues, mve.Name)
		case "vendor":
			match = containsString(filterValues, mve.Vendor)
		case "size":
			match = containsString(filterValues, mve.Size)
		case "location-id":
			match = containsInt(filterValues, mve.LocationID)
		case "provisioning-status":
			match = containsString(filterValues, mve.ProvisioningStatus)
		case "market":
			match = containsString(filterValues, mve.Market)
		case "company-name":
			match = containsString(filterValues, mve.CompanyName)
		case "vxc-permitted":
			match = containsBool(filterValues, mve.VXCPermitted)
		case "diversity-zone":
			match = containsString(filterValues, mve.DiversityZone)
		case "cost-centre":
			match = containsString(filterValues, mve.CostCentre)
		default:
			diags.AddWarning(
				"Unknown filter",
				fmt.Sprintf("Filter name '%s' is not supported", name),
			)
			// Don't reject the MVE based on unknown filter
			match = true
		}

		if !match {
			return false, diags
		}
	}

	return true, diags
}
