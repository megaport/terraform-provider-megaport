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
var _ datasource.DataSource = &ixsDataSource{}

// ixsDataSource is the data source implementation.
type ixsDataSource struct {
	client *megaport.Client
}

// ixsModel maps the data source schema data.
type ixsModel struct {
	UIDs   types.List    `tfsdk:"uids"`
	Filter []filterModel `tfsdk:"filter"`
	Tags   types.Map     `tfsdk:"tags"`
}

// NewIXsDataSource creates a new IXs data source.
func NewIXsDataSource() datasource.DataSource {
	return &ixsDataSource{}
}

// Metadata returns the data source type name.
func (d *ixsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ixs"
}

// Schema defines the schema for the data source.
func (d *ixsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a list of IX UIDs matching the specified filters.",
		Attributes: map[string]schema.Attribute{
			"uids": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of IX UIDs that match the specified criteria.",
			},
			"tags": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Map of resource tags, each pair of which must exactly match a pair on the desired IXs.",
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.ListNestedBlock{
				Description: "Custom filter block to select IXs.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Name of the field to filter by. Available filters: name, vlan, asn, network-service-type, location-id, rate-limit, provisioning-status, company-name.",
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
func (d *ixsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *ixsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ixsModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get all IXs from the API
	ixs, err := d.client.IXService.ListIXs(ctx, &megaport.ListIXsRequest{
		IncludeInactive: false,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing IXs",
			fmt.Sprintf("Unable to list IXs: %v", err),
		)
		return
	}

	// Apply filtering
	filteredIXs, filterDiags := d.filterIXs(ctx, ixs, data)
	resp.Diagnostics.Append(filterDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract IX IDs
	ixUIDs := make([]string, 0, len(filteredIXs))
	for _, ix := range filteredIXs {
		ixUIDs = append(ixUIDs, ix.ProductUID)
	}

	// Set the IX IDs in the model
	ixUIDsList, diags := types.ListValueFrom(ctx, types.StringType, ixUIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.UIDs = ixUIDsList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// filterIXs applies filters to the list of IXs.
func (d *ixsDataSource) filterIXs(ctx context.Context, ixs []*megaport.IX, data ixsModel) ([]*megaport.IX, diag.Diagnostics) {
	var diags diag.Diagnostics

	// If no filters or tags are provided, return all IXs
	if len(data.Filter) == 0 && data.Tags.IsNull() {
		return ixs, diags
	}

	var filteredIXs []*megaport.IX

	// Handle tag filtering
	var tagFilters map[string]string
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		diags.Append(data.Tags.ElementsAs(ctx, &tagFilters, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}

	// Process each IX
	for _, ix := range ixs {
		// Check tag filters if any
		if len(tagFilters) > 0 {
			// Use attribute tags if available, otherwise try to fetch from API
			ixTags := ix.AttributeTags
			if len(ixTags) == 0 {
				var err error
				ixTags, err = d.client.IXService.ListIXResourceTags(ctx, ix.ProductUID)
				if err != nil {
					diags.AddWarning(
						"Error fetching IX tags",
						fmt.Sprintf("Unable to fetch tags for IX %s: %v", ix.ProductUID, err),
					)
					continue
				}
			}

			// Check if IX matches all tag filters
			if !matchesTags(ixTags, tagFilters) {
				continue
			}
		}

		// Check custom filters
		match, filterDiags := matchesIXFilters(ctx, ix, data.Filter)
		diags.Append(filterDiags...)
		if !match {
			continue
		}

		filteredIXs = append(filteredIXs, ix)
	}

	return filteredIXs, diags
}

// matchesIXFilters checks if an IX matches the custom filters.
func matchesIXFilters(ctx context.Context, ix *megaport.IX, filters []filterModel) (bool, diag.Diagnostics) {
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
			match = matchesNamePattern(filterValues, ix.ProductName)
		case "vlan":
			match = containsInt(filterValues, ix.VLAN)
		case "asn":
			match = containsInt(filterValues, ix.ASN)
		case "network-service-type":
			match = containsString(filterValues, ix.NetworkServiceType)
		case "location-id":
			match = containsInt(filterValues, ix.LocationID)
		case "rate-limit":
			match = containsInt(filterValues, ix.RateLimit)
		case "provisioning-status":
			match = containsString(filterValues, ix.ProvisioningStatus)
		case "company-name":
			// IX may not have a company name directly, so we'll just check if the filter includes empty string
			if ix.LocationDetail.Name != "" {
				match = containsString(filterValues, ix.LocationDetail.Name)
			} else {
				match = containsString(filterValues, "")
			}
		default:
			diags.AddWarning(
				"Unknown filter",
				fmt.Sprintf("Filter name '%s' is not supported", name),
			)
			// Don't reject the IX based on unknown filter
			match = true
		}

		if !match {
			return false, diags
		}
	}

	return true, diags
}
