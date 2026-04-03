package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource = &mcrsDataSource{}

	mcrDetailAttrs = map[string]attr.Type{
		"product_uid":            types.StringType,
		"product_name":           types.StringType,
		"provisioning_status":    types.StringType,
		"create_date":            types.StringType,
		"created_by":             types.StringType,
		"port_speed":             types.Int64Type,
		"location_id":            types.Int64Type,
		"market":                 types.StringType,
		"company_uid":            types.StringType,
		"company_name":           types.StringType,
		"cost_centre":            types.StringType,
		"contract_term_months":   types.Int64Type,
		"contract_start_date":    types.StringType,
		"contract_end_date":      types.StringType,
		"live_date":              types.StringType,
		"terminate_date":         types.StringType,
		"diversity_zone":         types.StringType,
		"secondary_name":         types.StringType,
		"vxc_permitted":          types.BoolType,
		"vxc_auto_approval":      types.BoolType,
		"marketplace_visibility": types.BoolType,
		"asn":                    types.Int64Type,
		"locked":                 types.BoolType,
		"admin_locked":           types.BoolType,
		"cancelable":             types.BoolType,
		"attribute_tags":         types.MapType{ElemType: types.StringType},
		"resource_tags":          types.MapType{ElemType: types.StringType},
	}
)

// mcrsDataSource is the data source implementation.
type mcrsDataSource struct {
	client *megaport.Client
}

// mcrsModel maps the data source schema data.
type mcrsModel struct {
	UIDs   types.List    `tfsdk:"uids"`
	MCRs   types.List    `tfsdk:"mcrs"`
	Filter []filterModel `tfsdk:"filter"`
	Tags   types.Map     `tfsdk:"tags"`
}

// mcrDetailModel maps individual MCR detail attributes.
type mcrDetailModel struct {
	UID                   types.String `tfsdk:"product_uid"`
	Name                  types.String `tfsdk:"product_name"`
	ProvisioningStatus    types.String `tfsdk:"provisioning_status"`
	CreateDate            types.String `tfsdk:"create_date"`
	CreatedBy             types.String `tfsdk:"created_by"`
	PortSpeed             types.Int64  `tfsdk:"port_speed"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	Market                types.String `tfsdk:"market"`
	CompanyUID            types.String `tfsdk:"company_uid"`
	CompanyName           types.String `tfsdk:"company_name"`
	CostCentre            types.String `tfsdk:"cost_centre"`
	ContractTermMonths    types.Int64  `tfsdk:"contract_term_months"`
	ContractStartDate     types.String `tfsdk:"contract_start_date"`
	ContractEndDate       types.String `tfsdk:"contract_end_date"`
	LiveDate              types.String `tfsdk:"live_date"`
	TerminateDate         types.String `tfsdk:"terminate_date"`
	DiversityZone         types.String `tfsdk:"diversity_zone"`
	SecondaryName         types.String `tfsdk:"secondary_name"`
	VXCPermitted          types.Bool   `tfsdk:"vxc_permitted"`
	VXCAutoApproval       types.Bool   `tfsdk:"vxc_auto_approval"`
	MarketplaceVisibility types.Bool   `tfsdk:"marketplace_visibility"`
	ASN                   types.Int64  `tfsdk:"asn"`
	Locked                types.Bool   `tfsdk:"locked"`
	AdminLocked           types.Bool   `tfsdk:"admin_locked"`
	Cancelable            types.Bool   `tfsdk:"cancelable"`
	AttributeTags         types.Map    `tfsdk:"attribute_tags"`
	ResourceTags          types.Map    `tfsdk:"resource_tags"`
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
		Description: "Provides a list of MCRs matching the specified filters, with detailed information about each MCR.",
		Attributes: map[string]schema.Attribute{
			"uids": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of MCR UIDs that match the specified criteria.",
			},
			"mcrs": schema.ListNestedAttribute{
				Description: "List of MCRs matching the specified criteria with detailed information.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"product_uid": schema.StringAttribute{
							Description: "The unique identifier of the MCR.",
							Computed:    true,
						},
						"product_name": schema.StringAttribute{
							Description: "The name of the MCR.",
							Computed:    true,
						},
						"provisioning_status": schema.StringAttribute{
							Description: "The provisioning status of the MCR.",
							Computed:    true,
						},
						"create_date": schema.StringAttribute{
							Description: "The date the MCR was created.",
							Computed:    true,
						},
						"created_by": schema.StringAttribute{
							Description: "The user who created the MCR.",
							Computed:    true,
						},
						"port_speed": schema.Int64Attribute{
							Description: "The bandwidth speed of the MCR in Mbps.",
							Computed:    true,
						},
						"location_id": schema.Int64Attribute{
							Description: "The numeric location ID of the MCR.",
							Computed:    true,
						},
						"market": schema.StringAttribute{
							Description: "The market the MCR is in.",
							Computed:    true,
						},
						"company_uid": schema.StringAttribute{
							Description: "The Megaport Company UID of the MCR owner.",
							Computed:    true,
						},
						"company_name": schema.StringAttribute{
							Description: "The name of the company that owns the MCR.",
							Computed:    true,
						},
						"cost_centre": schema.StringAttribute{
							Description: "The cost centre of the MCR for billing purposes.",
							Computed:    true,
						},
						"contract_term_months": schema.Int64Attribute{
							Description: "The contract term of the MCR in months.",
							Computed:    true,
						},
						"contract_start_date": schema.StringAttribute{
							Description: "The contract start date of the MCR.",
							Computed:    true,
						},
						"contract_end_date": schema.StringAttribute{
							Description: "The contract end date of the MCR.",
							Computed:    true,
						},
						"live_date": schema.StringAttribute{
							Description: "The date the MCR went live.",
							Computed:    true,
						},
						"terminate_date": schema.StringAttribute{
							Description: "The date the MCR will be terminated.",
							Computed:    true,
						},
						"diversity_zone": schema.StringAttribute{
							Description: "The diversity zone of the MCR.",
							Computed:    true,
						},
						"secondary_name": schema.StringAttribute{
							Description: "The secondary name of the MCR.",
							Computed:    true,
						},
						"vxc_permitted": schema.BoolAttribute{
							Description: "Whether VXC connections are permitted on this MCR.",
							Computed:    true,
						},
						"vxc_auto_approval": schema.BoolAttribute{
							Description: "Whether VXC connections are auto-approved on this MCR.",
							Computed:    true,
						},
						"marketplace_visibility": schema.BoolAttribute{
							Description: "Whether the MCR is visible in the Marketplace.",
							Computed:    true,
						},
						"asn": schema.Int64Attribute{
							Description: "The Autonomous System Number (ASN) of the MCR.",
							Computed:    true,
						},
						"locked": schema.BoolAttribute{
							Description: "Whether the MCR is locked.",
							Computed:    true,
						},
						"admin_locked": schema.BoolAttribute{
							Description: "Whether the MCR is admin locked.",
							Computed:    true,
						},
						"cancelable": schema.BoolAttribute{
							Description: "Whether the MCR can be cancelled.",
							Computed:    true,
						},
						"attribute_tags": schema.MapAttribute{
							ElementType: types.StringType,
							Description: "The attribute tags of the MCR.",
							Computed:    true,
						},
						"resource_tags": schema.MapAttribute{
							ElementType: types.StringType,
							Description: "The resource tags associated with the MCR.",
							Computed:    true,
						},
					},
				},
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
							Description: "Name of the field to filter by. Available filters: name, port-speed, location-id, cost-centre, provisioning-status, market, company-name, company-uid, contract-term-months, vxc-permitted, vxc-auto-approval, marketplace-visibility, asn, diversity-zone, secondary-name, locked, admin-locked, cancelable.",
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

	data, ok := req.ProviderData.(*megaportProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *megaportProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	client := data.client

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

	// Extract MCR UIDs and build detail objects
	mcrUIDs := make([]string, 0, len(filteredMCRs))
	mcrObjects := make([]types.Object, 0, len(filteredMCRs))

	for _, mcr := range filteredMCRs {
		mcrUIDs = append(mcrUIDs, mcr.UID)

		// Fetch resource tags for each MCR
		tags, err := d.client.MCRService.ListMCRResourceTags(ctx, mcr.UID)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Error fetching MCR tags",
				fmt.Sprintf("Unable to fetch resource tags for MCR %s: %v", mcr.UID, err),
			)
			tags = map[string]string{}
		}

		detail := fromAPIMCRDetail(mcr, tags)
		obj, objDiags := types.ObjectValueFrom(ctx, mcrDetailAttrs, &detail)
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		mcrObjects = append(mcrObjects, obj)
	}

	// Set the MCR UIDs in the model
	mcrUIDsList, diags := types.ListValueFrom(ctx, types.StringType, mcrUIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.UIDs = mcrUIDsList

	// Set the MCR details in the model
	mcrsList, mcrsDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: mcrDetailAttrs}, mcrObjects)
	resp.Diagnostics.Append(mcrsDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.MCRs = mcrsList

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
		case "company-uid":
			match = containsString(filterValues, mcr.CompanyUID)
		case "contract-term-months":
			match = containsInt(filterValues, mcr.ContractTermMonths)
		case "vxc-permitted":
			match = containsBool(filterValues, mcr.VXCPermitted)
		case "vxc-auto-approval":
			match = containsBool(filterValues, mcr.VXCAutoApproval)
		case "marketplace-visibility":
			match = containsBool(filterValues, mcr.MarketplaceVisibility)
		case "diversity-zone":
			match = containsString(filterValues, mcr.DiversityZone)
		case "secondary-name":
			match = matchesNamePattern(filterValues, mcr.SecondaryName)
		case "cost-centre":
			match = containsString(filterValues, mcr.CostCentre)
		case "asn":
			match = containsInt(filterValues, mcr.Resources.VirtualRouter.ASN)
		case "locked":
			match = containsBool(filterValues, mcr.Locked)
		case "admin-locked":
			match = containsBool(filterValues, mcr.AdminLocked)
		case "cancelable":
			match = containsBool(filterValues, mcr.Cancelable)
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

// fromAPIMCRDetail maps an API MCR and its resource tags to an mcrDetailModel.
func fromAPIMCRDetail(m *megaport.MCR, tags map[string]string) mcrDetailModel {
	detail := mcrDetailModel{
		UID:                   types.StringValue(m.UID),
		Name:                  types.StringValue(m.Name),
		ProvisioningStatus:    types.StringValue(m.ProvisioningStatus),
		CreatedBy:             types.StringValue(m.CreatedBy),
		PortSpeed:             types.Int64Value(int64(m.PortSpeed)),
		LocationID:            types.Int64Value(int64(m.LocationID)),
		Market:                types.StringValue(m.Market),
		CompanyUID:            types.StringValue(m.CompanyUID),
		CompanyName:           types.StringValue(m.CompanyName),
		CostCentre:            types.StringValue(m.CostCentre),
		ContractTermMonths:    types.Int64Value(int64(m.ContractTermMonths)),
		DiversityZone:         types.StringValue(m.DiversityZone),
		SecondaryName:         types.StringValue(m.SecondaryName),
		VXCPermitted:          types.BoolValue(m.VXCPermitted),
		VXCAutoApproval:       types.BoolValue(m.VXCAutoApproval),
		MarketplaceVisibility: types.BoolValue(m.MarketplaceVisibility),
		ASN:                   types.Int64Value(int64(m.Resources.VirtualRouter.ASN)),
		Locked:                types.BoolValue(m.Locked),
		AdminLocked:           types.BoolValue(m.AdminLocked),
		Cancelable:            types.BoolValue(m.Cancelable),
	}

	// Time fields
	if m.CreateDate != nil {
		detail.CreateDate = types.StringValue(m.CreateDate.String())
	} else {
		detail.CreateDate = types.StringValue("")
	}
	if m.LiveDate != nil {
		detail.LiveDate = types.StringValue(m.LiveDate.String())
	} else {
		detail.LiveDate = types.StringValue("")
	}
	if m.TerminateDate != nil {
		detail.TerminateDate = types.StringValue(m.TerminateDate.String())
	} else {
		detail.TerminateDate = types.StringValue("")
	}
	if m.ContractStartDate != nil {
		detail.ContractStartDate = types.StringValue(m.ContractStartDate.String())
	} else {
		detail.ContractStartDate = types.StringValue("")
	}
	if m.ContractEndDate != nil {
		detail.ContractEndDate = types.StringValue(m.ContractEndDate.String())
	} else {
		detail.ContractEndDate = types.StringValue("")
	}

	// Attribute tags
	if m.AttributeTags != nil {
		attrTagValues := make(map[string]attr.Value, len(m.AttributeTags))
		for k, v := range m.AttributeTags {
			attrTagValues[k] = types.StringValue(v)
		}
		detail.AttributeTags, _ = types.MapValue(types.StringType, attrTagValues)
	} else {
		detail.AttributeTags = types.MapNull(types.StringType)
	}

	// Resource tags
	if len(tags) > 0 {
		resourceTagValues := make(map[string]attr.Value, len(tags))
		for k, v := range tags {
			resourceTagValues[k] = types.StringValue(v)
		}
		detail.ResourceTags, _ = types.MapValue(types.StringType, resourceTagValues)
	} else {
		detail.ResourceTags = types.MapNull(types.StringType)
	}

	return detail
}
