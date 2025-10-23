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

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &mcrPrefixFilterListDataSource{}
	_ datasource.DataSourceWithConfigure = &mcrPrefixFilterListDataSource{}
)

// NewMCRPrefixFilterListDataSource is a helper function to simplify the provider implementation.
func NewMCRPrefixFilterListDataSource() datasource.DataSource {
	return &mcrPrefixFilterListDataSource{}
}

// mcrPrefixFilterListDataSource defines the data source implementation.
type mcrPrefixFilterListDataSource struct {
	client *megaport.Client
}

// mcrPrefixFilterListDataSourceModel describes the data source data model.
type mcrPrefixFilterListDataSourceModel struct {
	MCRID             types.String `tfsdk:"mcr_id"`
	PrefixFilterLists types.List   `tfsdk:"prefix_filter_lists"`
}

// mcrPrefixFilterListDataModel represents a single prefix filter list in the data source.
type mcrPrefixFilterListDataModel struct {
	ID            types.Int64  `tfsdk:"id"`
	Description   types.String `tfsdk:"description"`
	AddressFamily types.String `tfsdk:"address_family"`
	Entries       types.List   `tfsdk:"entries"`
}

var (
	// prefixFilterListDataAttributes defines the attribute types for the data source
	prefixFilterListDataAttributes = map[string]attr.Type{
		"id":             types.Int64Type,
		"description":    types.StringType,
		"address_family": types.StringType,
		"entries":        types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListEntryAttributes)),
	}
)

// Metadata returns the data source type name.
func (d *mcrPrefixFilterListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcr_prefix_filter_lists"
}

// Schema defines the schema for the data source.
func (d *mcrPrefixFilterListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for listing all prefix filter lists associated with an MCR. " +
			"This is particularly useful when migrating from inline prefix_filter_lists to " +
			"standalone megaport_mcr_prefix_filter_list resources.",
		Attributes: map[string]schema.Attribute{
			"mcr_id": schema.StringAttribute{
				Description: "The UID of the MCR instance to list prefix filter lists for.",
				Required:    true,
			},
			"prefix_filter_lists": schema.ListNestedAttribute{
				Description: "List of all prefix filter lists for this MCR.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "Numeric ID of the prefix filter list.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the prefix filter list.",
							Computed:    true,
						},
						"address_family": schema.StringAttribute{
							Description: "The IP address standard of the IP network addresses in the prefix filter list.",
							Computed:    true,
						},
						"entries": schema.ListNestedAttribute{
							Description: "Entries in the prefix filter list.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"action": schema.StringAttribute{
										Description: "The action to take for the network address in the filter list.",
										Computed:    true,
									},
									"prefix": schema.StringAttribute{
										Description: "The network address of the prefix filter list entry.",
										Computed:    true,
									},
									"ge": schema.Int64Attribute{
										Description: "The minimum starting prefix length to be matched.",
										Computed:    true,
									},
									"le": schema.Int64Attribute{
										Description: "The maximum ending prefix length to be matched.",
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

// Configure adds the provider configured client to the data source.
func (d *mcrPrefixFilterListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*megaportProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *megaportProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client.client
}

// Read refreshes the Terraform state with the latest data.
func (d *mcrPrefixFilterListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config mcrPrefixFilterListDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get prefix filter lists from API
	prefixFilterLists, err := d.client.MCRService.ListMCRPrefixFilterLists(ctx, config.MCRID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading MCR prefix filter lists",
			fmt.Sprintf("Could not read prefix filter lists for MCR %s: %s",
				config.MCRID.ValueString(), err.Error()),
		)
		return
	}

	// Convert API response to Terraform types
	prefixFilterListObjects := []types.Object{}
	for _, apiList := range prefixFilterLists {
		// Get detailed information including entries for each prefix filter list
		detailedList, err := d.client.MCRService.GetMCRPrefixFilterList(ctx, config.MCRID.ValueString(), apiList.Id)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Error reading prefix filter list details",
				fmt.Sprintf("Could not read details for prefix filter list %d: %s", apiList.Id, err.Error()),
			)
			continue
		}

		// Convert entries
		entryObjects := []types.Object{}
		for _, entry := range detailedList.Entries {
			entryModel := &mcrPrefixFilterListEntryResourceModel{
				Action: types.StringValue(entry.Action),
				Prefix: types.StringValue(entry.Prefix),
				Ge:     types.Int64Value(int64(entry.Ge)),
				Le:     types.Int64Value(int64(entry.Le)),
			}

			entryObj, entryDiags := types.ObjectValueFrom(ctx, mcrPrefixFilterListEntryAttributes, entryModel)
			resp.Diagnostics.Append(entryDiags...)
			if !resp.Diagnostics.HasError() {
				entryObjects = append(entryObjects, entryObj)
			}
		}

		entriesList, entriesDiags := types.ListValueFrom(ctx,
			types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListEntryAttributes), entryObjects)
		resp.Diagnostics.Append(entriesDiags...)
		if resp.Diagnostics.HasError() {
			continue
		}

		// Create the prefix filter list data model
		listModel := &mcrPrefixFilterListDataModel{
			ID:            types.Int64Value(int64(detailedList.ID)),
			Description:   types.StringValue(detailedList.Description),
			AddressFamily: types.StringValue(detailedList.AddressFamily),
			Entries:       entriesList,
		}

		listObj, listDiags := types.ObjectValueFrom(ctx, prefixFilterListDataAttributes, listModel)
		resp.Diagnostics.Append(listDiags...)
		if !resp.Diagnostics.HasError() {
			prefixFilterListObjects = append(prefixFilterListObjects, listObj)
		}
	}

	// Create the final list
	prefixFilterListsList, listDiags := types.ListValueFrom(ctx,
		types.ObjectType{}.WithAttributeTypes(prefixFilterListDataAttributes), prefixFilterListObjects)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.PrefixFilterLists = prefixFilterListsList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
