package provider

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces
var _ datasource.DataSource = &portsDataSource{}

// portsDataSource is the data source implementation.
type portsDataSource struct {
	client *megaport.Client
}

// portsModel maps the data source schema data.
type portsModel struct {
	UIDs   types.List    `tfsdk:"uids"`
	Filter []filterModel `tfsdk:"filter"`
	Tags   types.Map     `tfsdk:"tags"`
}

// filterModel maps filter block schema data.
type filterModel struct {
	Name   types.String `tfsdk:"name"`
	Values types.List   `tfsdk:"values"`
}

// NewPortsDataSource creates a new ports data source.
func NewPortsDataSource() datasource.DataSource {
	return &portsDataSource{}
}

// Metadata returns the data source type name.
func (d *portsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ports"
}

// Schema defines the schema for the data source.
func (d *portsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a list of port UIDs matching the specified filters.",
		Attributes: map[string]schema.Attribute{
			"uids": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of port UIDs that match the specified criteria.",
			},
			"tags": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Map of resource tags, each pair of which must exactly match a pair on the desired ports.",
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.ListNestedBlock{
				Description: "Custom filter block to select ports.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Name of the field to filter by. Available filters: name, port-speed, location-id, cost-centre, provisioning-status, market, company-name, vxc-permitted.",
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
func (d *portsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *portsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data portsModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get all ports from the API
	ports, err := d.client.PortService.ListPorts(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing ports",
			fmt.Sprintf("Unable to list ports: %v", err),
		)
		return
	}

	// Apply filtering
	filteredPorts, filterDiags := d.filterPorts(ctx, ports, data)
	resp.Diagnostics.Append(filterDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract port IDs
	portUIDs := make([]string, 0, len(filteredPorts))
	for _, port := range filteredPorts {
		portUIDs = append(portUIDs, port.UID)
	}

	// Set the port IDs in the model
	portUIDsList, diags := types.ListValueFrom(ctx, types.StringType, portUIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.UIDs = portUIDsList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// filterPorts applies filters to the list of ports.
func (d *portsDataSource) filterPorts(ctx context.Context, ports []*megaport.Port, data portsModel) ([]*megaport.Port, diag.Diagnostics) {
	var diags diag.Diagnostics

	// If no filters or tags are provided, return all ports
	if len(data.Filter) == 0 && data.Tags.IsNull() {
		return ports, diags
	}

	var filteredPorts []*megaport.Port

	// Handle tag filtering
	var tagFilters map[string]string
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		diags.Append(data.Tags.ElementsAs(ctx, &tagFilters, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}

	// Process each port
	for _, port := range ports {
		// Check tag filters if any
		if len(tagFilters) > 0 {
			// Get port tags
			portTags, err := d.client.PortService.ListPortResourceTags(ctx, port.UID)
			if err != nil {
				diags.AddWarning(
					"Error fetching port tags",
					fmt.Sprintf("Unable to fetch tags for port %s: %v", port.UID, err),
				)
				continue
			}

			// Check if port matches all tag filters
			if !matchesTags(portTags, tagFilters) {
				continue
			}
		}

		// Check custom filters
		match, filterDiags := matchesFilters(ctx, port, data.Filter)
		diags.Append(filterDiags...)
		if !match {
			continue
		}

		filteredPorts = append(filteredPorts, port)
	}

	return filteredPorts, diags
}

// matchesTags checks if port tags match the specified tag filters.
func matchesTags(portTags map[string]string, tagFilters map[string]string) bool {
	for key, value := range tagFilters {
		if portTags[key] != value {
			return false
		}
	}
	return true
}

// matchesFilters checks if a port matches the custom filters.
func matchesFilters(ctx context.Context, port *megaport.Port, filters []filterModel) (bool, diag.Diagnostics) {
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
			match = matchesNamePattern(filterValues, port.Name)
		case "port-speed":
			match = containsInt(filterValues, port.PortSpeed)
		case "location-id":
			match = containsInt(filterValues, port.LocationID)
		case "provisioning-status":
			match = containsString(filterValues, port.ProvisioningStatus)
		case "market":
			match = containsString(filterValues, port.Market)
		case "company-name":
			match = containsString(filterValues, port.CompanyName)
		case "cost-centre":
			match = containsString(filterValues, port.CostCentre)
		case "vxc-permitted":
			match = containsBool(filterValues, port.VXCPermitted)
		// Add more filters as needed
		default:
			diags.AddWarning(
				"Unknown filter",
				fmt.Sprintf("Filter name '%s' is not supported", name),
			)
			// Don't reject the port based on unknown filter
			match = true
		}

		if !match {
			return false, diags
		}
	}

	return true, diags
}

func matchesNamePattern(patterns []string, name string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// Helper functions for filter matching
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsInt(slice []string, item int) bool {
	strItem := strconv.Itoa(item)
	for _, s := range slice {
		if s == strItem {
			return true
		}
	}
	return false
}

func containsBool(slice []string, item bool) bool {
	strItem := strconv.FormatBool(item)
	for _, s := range slice {
		if s == strItem {
			return true
		}
	}
	return false
}
