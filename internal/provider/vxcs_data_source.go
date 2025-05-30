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
var _ datasource.DataSource = &vxcsDataSource{}

// vxcsDataSource is the data source implementation.
type vxcsDataSource struct {
	client *megaport.Client
}

// vxcsModel maps the data source schema data.
type vxcsModel struct {
	UIDs   types.List    `tfsdk:"uids"`
	Filter []filterModel `tfsdk:"filter"`
	Tags   types.Map     `tfsdk:"tags"`
}

// NewVXCsDataSource creates a new VXCs data source.
func NewVXCsDataSource() datasource.DataSource {
	return &vxcsDataSource{}
}

// Metadata returns the data source type name.
func (d *vxcsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vxcs"
}

// Schema defines the schema for the data source.
func (d *vxcsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a list of VXC UIDs matching the specified filters.",
		Attributes: map[string]schema.Attribute{
			"uids": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of VXC UIDs that match the specified criteria.",
			},
			"tags": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Map of resource tags, each pair of which must exactly match a pair on the desired VXCs.",
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.ListNestedBlock{
				Description: "Custom filter block to select VXCs.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Name of the field to filter by. Available filters: name, rate-limit, provisioning-status, aend-uid, bend-uid, company-name.",
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
func (d *vxcsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *vxcsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data vxcsModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get all VXCs from the API
	vxcs, err := d.client.VXCService.ListVXCs(ctx, &megaport.ListVXCsRequest{
		IncludeInactive: false,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing VXCs",
			fmt.Sprintf("Unable to list VXCs: %v", err),
		)
		return
	}

	// Apply filtering
	filteredVXCs, filterDiags := d.filterVXCs(ctx, vxcs, data)
	resp.Diagnostics.Append(filterDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract VXC IDs
	vxcUIDs := make([]string, 0, len(filteredVXCs))
	for _, vxc := range filteredVXCs {
		vxcUIDs = append(vxcUIDs, vxc.UID)
	}

	// Set the VXC IDs in the model
	vxcUIDsList, diags := types.ListValueFrom(ctx, types.StringType, vxcUIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.UIDs = vxcUIDsList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// filterVXCs applies filters to the list of VXCs.
func (d *vxcsDataSource) filterVXCs(ctx context.Context, vxcs []*megaport.VXC, data vxcsModel) ([]*megaport.VXC, diag.Diagnostics) {
	var diags diag.Diagnostics

	// If no filters or tags are provided, return all VXCs
	if len(data.Filter) == 0 && data.Tags.IsNull() {
		return vxcs, diags
	}

	var filteredVXCs []*megaport.VXC

	// Handle tag filtering
	var tagFilters map[string]string
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		diags.Append(data.Tags.ElementsAs(ctx, &tagFilters, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}

	// Process each VXC
	for _, vxc := range vxcs {
		// Check tag filters if any
		if len(tagFilters) > 0 {
			// Get VXC tags
			vxcTags, err := d.client.VXCService.ListVXCResourceTags(ctx, vxc.UID)
			if err != nil {
				diags.AddWarning(
					"Error fetching VXC tags",
					fmt.Sprintf("Unable to fetch tags for VXC %s: %v", vxc.UID, err),
				)
				continue
			}

			// Check if VXC matches all tag filters
			if !matchesTags(vxcTags, tagFilters) {
				continue
			}
		}

		// Check custom filters
		match, filterDiags := matchesVXCFilters(ctx, vxc, data.Filter)
		diags.Append(filterDiags...)
		if !match {
			continue
		}

		filteredVXCs = append(filteredVXCs, vxc)
	}

	return filteredVXCs, diags
}

// matchesVXCFilters checks if a VXC matches the custom filters.
func matchesVXCFilters(ctx context.Context, vxc *megaport.VXC, filters []filterModel) (bool, diag.Diagnostics) {
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
			match = matchesNamePattern(filterValues, vxc.Name)
		case "rate-limit":
			match = containsInt(filterValues, vxc.RateLimit)
		case "provisioning-status":
			match = containsString(filterValues, vxc.ProvisioningStatus)
		case "aend-uid":
			match = containsString(filterValues, vxc.AEndConfiguration.UID)
		case "bend-uid":
			match = containsString(filterValues, vxc.BEndConfiguration.UID)
		case "company-name":
			match = containsString(filterValues, vxc.CompanyName)
		default:
			diags.AddWarning(
				"Unknown filter",
				fmt.Sprintf("Filter name '%s' is not supported", name),
			)
			// Don't reject the VXC based on unknown filter
			match = true
		}

		if !match {
			return false, diags
		}
	}

	return true, diags
}
