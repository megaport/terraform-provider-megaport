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
var _ datasource.DataSource = &mcrsDataSource{}

// mcrsDataSource is the data source implementation.
type mcrsDataSource struct {
	client *megaport.Client
}

// mcrsModel maps the data source schema data.
type mcrsModel struct {
	UIDs   types.List    `tfsdk:"uids"`
	Filter []filterModel `tfsdk:"filter"`
	Tags   types.Map     `tfsdk:"tags"`
}

// NewMCRsDataSource creates a new MCRs data source.
func NewMCRsDataSource() datasource.DataSource {
	return &mcrsDataSource{}
}

// Metadata returns the data source type name.
func (d *mcrsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcrs"
}

// Schema defines the schema for the data source.
func (d *mcrsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a list of MCR UIDs matching the specified filters.",
		Attributes: map[string]schema.Attribute{
			"uids": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of MCR UIDs that match the specified criteria.",
			},
			"tags": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Map of resource tags, each pair of which must exactly match a pair on the desired MCRs.",
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.ListNestedBlock{
				Description: "Custom filter block to select MCRs.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Name of the field to filter by. Available filters: name, port-speed, location-id, cost-centre, provisioning-status, market, company-name, vxc-permitted, asn, diversity-zone.",
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
func (d *mcrsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *mcrsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data mcrsModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get all MCRs from the API
	mcrs, err := d.client.MCRService.ListMCRs(ctx, &megaport.ListMCRsRequest{
		IncludeInactive: false,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing MCRs",
			fmt.Sprintf("Unable to list MCRs: %v", err),
		)
		return
	}

	// Apply filtering
	filteredMCRs, filterDiags := d.filterMCRs(ctx, mcrs, data)
	resp.Diagnostics.Append(filterDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract MCR IDs
	mcrUIDs := make([]string, 0, len(filteredMCRs))
	for _, mcr := range filteredMCRs {
		mcrUIDs = append(mcrUIDs, mcr.UID)
	}

	// Set the MCR IDs in the model
	mcrUIDsList, diags := types.ListValueFrom(ctx, types.StringType, mcrUIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.UIDs = mcrUIDsList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// filterMCRs applies filters to the list of MCRs.
func (d *mcrsDataSource) filterMCRs(ctx context.Context, mcrs []*megaport.MCR, data mcrsModel) ([]*megaport.MCR, diag.Diagnostics) {
	var diags diag.Diagnostics

	// If no filters or tags are provided, return all MCRs
	if len(data.Filter) == 0 && data.Tags.IsNull() {
		return mcrs, diags
	}

	var filteredMCRs []*megaport.MCR

	// Handle tag filtering
	var tagFilters map[string]string
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		diags.Append(data.Tags.ElementsAs(ctx, &tagFilters, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}

	// Process each MCR
	for _, mcr := range mcrs {
		// Check tag filters if any
		if len(tagFilters) > 0 {
			// Get MCR tags
			mcrTags, err := d.client.MCRService.ListMCRResourceTags(ctx, mcr.UID)
			if err != nil {
				diags.AddWarning(
					"Error fetching MCR tags",
					fmt.Sprintf("Unable to fetch tags for MCR %s: %v", mcr.UID, err),
				)
				continue
			}

			// Check if MCR matches all tag filters
			if !matchesTags(mcrTags, tagFilters) {
				continue
			}
		}

		// Check custom filters
		match, filterDiags := matchesMCRFilters(ctx, mcr, data.Filter)
		diags.Append(filterDiags...)
		if !match {
			continue
		}

		filteredMCRs = append(filteredMCRs, mcr)
	}

	return filteredMCRs, diags
}

// matchesMCRFilters checks if an MCR matches the custom filters.
func matchesMCRFilters(ctx context.Context, mcr *megaport.MCR, filters []filterModel) (bool, diag.Diagnostics) {
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
			match = matchesNamePattern(filterValues, mcr.Name)
		case "port-speed":
			match = containsInt(filterValues, mcr.PortSpeed)
		case "location-id":
			match = containsInt(filterValues, mcr.LocationID)
		case "provisioning-status":
			match = containsString(filterValues, mcr.ProvisioningStatus)
		case "market":
			match = containsString(filterValues, mcr.Market)
		case "company-name":
			match = containsString(filterValues, mcr.CompanyName)
		case "vxc-permitted":
			match = containsBool(filterValues, mcr.VXCPermitted)
		case "diversity-zone":
			match = containsString(filterValues, mcr.DiversityZone)
		case "cost-centre":
			match = containsString(filterValues, mcr.CostCentre)
		case "asn":
			match = containsInt(filterValues, mcr.Resources.VirtualRouter.ASN)
		default:
			diags.AddWarning(
				"Unknown filter",
				fmt.Sprintf("Filter name '%s' is not supported", name),
			)
			// Don't reject the MCR based on unknown filter
			match = true
		}

		if !match {
			return false, diags
		}
	}

	return true, diags
}
