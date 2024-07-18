package provider

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

var (
	// Ensure the implementation satisfies the expected interfaces.
	_ resource.Resource                = &mcrResource{}
	_ resource.ResourceWithConfigure   = &mcrResource{}
	_ resource.ResourceWithImportState = &mcrResource{}

	mcrPrefixFilterListModelAttributes = map[string]attr.Type{
		"id":             types.Int64Type,
		"description":    types.StringType,
		"address_family": types.StringType,
		"entries":        types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(mcrPrefixListEntryAttributes)),
	}

	mcrPrefixListEntryAttributes = map[string]attr.Type{
		"action": types.StringType,
		"prefix": types.StringType,
		"ge":     types.Int64Type,
		"le":     types.Int64Type,
	}
)

// mcrResourceModel maps the resource schema data.
type mcrResourceModel struct {
	LastUpdated types.String `tfsdk:"last_updated"`

	ID                    types.Int64  `tfsdk:"product_id"`
	UID                   types.String `tfsdk:"product_uid"`
	Name                  types.String `tfsdk:"product_name"`
	Type                  types.String `tfsdk:"product_type"`
	ProvisioningStatus    types.String `tfsdk:"provisioning_status"`
	CreateDate            types.String `tfsdk:"create_date"`
	CreatedBy             types.String `tfsdk:"created_by"`
	CostCentre            types.String `tfsdk:"cost_centre"`
	PortSpeed             types.Int64  `tfsdk:"port_speed"`
	TerminateDate         types.String `tfsdk:"terminate_date"`
	LiveDate              types.String `tfsdk:"live_date"`
	Market                types.String `tfsdk:"market"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	UsageAlgorithm        types.String `tfsdk:"usage_algorithm"`
	MarketplaceVisibility types.Bool   `tfsdk:"marketplace_visibility"`
	VXCPermitted          types.Bool   `tfsdk:"vxc_permitted"`
	VXCAutoApproval       types.Bool   `tfsdk:"vxc_auto_approval"`
	SecondaryName         types.String `tfsdk:"secondary_name"`
	LAGPrimary            types.Bool   `tfsdk:"lag_primary"`
	LAGID                 types.Int64  `tfsdk:"lag_id"`
	AggregationID         types.Int64  `tfsdk:"aggregation_id"`
	CompanyUID            types.String `tfsdk:"company_uid"`
	CompanyName           types.String `tfsdk:"company_name"`
	ContractStartDate     types.String `tfsdk:"contract_start_date"`
	ContractEndDate       types.String `tfsdk:"contract_end_date"`
	ContractTermMonths    types.Int64  `tfsdk:"contract_term_months"`
	ASN                   types.Int64  `tfsdk:"asn"`
	DiversityZone         types.String `tfsdk:"diversity_zone"`
	PromoCode             types.String `tfsdk:"promo_code"`

	Virtual       types.Bool `tfsdk:"virtual"`
	BuyoutPort    types.Bool `tfsdk:"buyout_port"`
	Locked        types.Bool `tfsdk:"locked"`
	AdminLocked   types.Bool `tfsdk:"admin_locked"`
	Cancelable    types.Bool `tfsdk:"cancelable"`
	AttributeTags types.Map  `tfsdk:"attribute_tags"`

	PrefixFilterLists types.List `tfsdk:"prefix_filter_lists"`
}

// mcrPrefixFilterListModel represents the prefix filter list associated with the MCR
type mcrPrefixFilterListModel struct {
	ID            types.Int64  `tfsdk:"id"`
	Description   types.String `tfsdk:"description"`
	AddressFamily types.String `tfsdk:"address_family"`
	Entries       types.List   `tfsdk:"entries"`
}

// MCRPrefixListEntry represents an entry in a prefix filter list.
type mcrPrefixListEntryModel struct {
	Action types.String `tfsdk:"action"`
	Prefix types.String `tfsdk:"prefix"`
	Ge     types.Int64  `tfsdk:"ge"`
	Le     types.Int64  `tfsdk:"le"`
}

// fromAPIMCR maps the API MCR response to the resource schema.
func (orm *mcrResourceModel) fromAPIMCR(_ context.Context, m *megaport.MCR) diag.Diagnostics {
	apiDiags := diag.Diagnostics{}

	orm.ID = types.Int64Value(int64(m.ID))
	orm.UID = types.StringValue(m.UID)
	orm.Name = types.StringValue(m.Name)
	orm.Type = types.StringValue(m.Type)
	orm.ProvisioningStatus = types.StringValue(m.ProvisioningStatus)
	orm.CreatedBy = types.StringValue(m.CreatedBy)
	orm.CostCentre = types.StringValue(m.CostCentre)
	orm.PortSpeed = types.Int64Value(int64(m.PortSpeed))
	orm.Market = types.StringValue(m.Market)
	orm.LocationID = types.Int64Value(int64(m.LocationID))
	orm.UsageAlgorithm = types.StringValue(m.UsageAlgorithm)
	orm.MarketplaceVisibility = types.BoolValue(m.MarketplaceVisibility)
	orm.VXCPermitted = types.BoolValue(m.VXCPermitted)
	orm.VXCAutoApproval = types.BoolValue(m.VXCAutoApproval)
	orm.SecondaryName = types.StringValue(m.SecondaryName)
	orm.LAGPrimary = types.BoolValue(m.LAGPrimary)
	orm.LAGID = types.Int64Value(int64(m.LAGID))
	orm.AggregationID = types.Int64Value(int64(m.AggregationID))
	orm.CompanyUID = types.StringValue(m.CompanyUID)
	orm.CompanyName = types.StringValue(m.CompanyName)
	orm.ContractTermMonths = types.Int64Value(int64(m.ContractTermMonths))
	orm.Virtual = types.BoolValue(m.Virtual)
	orm.BuyoutPort = types.BoolValue(m.BuyoutPort)
	orm.Locked = types.BoolValue(m.Locked)
	orm.AdminLocked = types.BoolValue(m.AdminLocked)
	orm.Cancelable = types.BoolValue(m.Cancelable)

	if m.CreateDate != nil {
		orm.CreateDate = types.StringValue(m.CreateDate.String())
	} else {
		orm.CreateDate = types.StringValue("")
	}
	if m.TerminateDate != nil {
		orm.TerminateDate = types.StringValue(m.TerminateDate.String())
	} else {
		orm.TerminateDate = types.StringValue("")
	}
	if m.LiveDate != nil {
		orm.LiveDate = types.StringValue(m.LiveDate.String())
	} else {
		orm.LiveDate = types.StringValue("")
	}
	if m.ContractStartDate != nil {
		orm.ContractStartDate = types.StringValue(m.ContractStartDate.String())
	} else {
		orm.ContractStartDate = types.StringValue("")
	}
	if m.ContractEndDate != nil {
		orm.ContractEndDate = types.StringValue(m.ContractEndDate.String())
	} else {
		orm.ContractEndDate = types.StringValue("")
	}

	if m.AttributeTags != nil {
		attributeTags := make(map[string]attr.Value)
		for k, v := range m.AttributeTags {
			attributeTags[k] = types.StringValue(v)
		}
		tags, tagDiags := types.MapValue(types.StringType, attributeTags)
		apiDiags = append(apiDiags, tagDiags...)
		orm.AttributeTags = tags
	}

	return apiDiags
}

func (orm *mcrPrefixFilterListModel) fromAPIMCRPrefixFilterList(ctx context.Context, m *megaport.MCRPrefixFilterList) diag.Diagnostics {
	diags := diag.Diagnostics{}
	orm.ID = types.Int64Value(int64(m.ID))
	orm.Description = types.StringValue(m.Description)
	orm.AddressFamily = types.StringValue(m.AddressFamily)
	entriesList := []types.Object{}
	for _, entry := range m.Entries {
		var le, ge int
		// Get Mask Length if not provided by API
		if entry.Le == 0 && entry.Ge == 0 {
			_, net, err := net.ParseCIDR(entry.Prefix)
			if err != nil {
				diags.AddError("Error parsing prefix", fmt.Sprintf("Error parsing prefix %s: %s", entry.Prefix, err))
				return diags
			}
			length, _ := net.Mask.Size()
			le = length
			ge = length
		} else if entry.Le != 0 && entry.Ge == 0 {
			_, net, err := net.ParseCIDR(entry.Prefix)
			if err != nil {
				diags.AddError("Error parsing prefix", fmt.Sprintf("Error parsing prefix %s: %s", entry.Prefix, err))
				return diags
			}
			length, _ := net.Mask.Size()
			ge = length
			le = entry.Le
		} else {
			le = entry.Le
			ge = entry.Ge
		}
		entryModel := &mcrPrefixListEntryModel{
			Action: types.StringValue(entry.Action),
			Prefix: types.StringValue(entry.Prefix),
			Ge:     types.Int64Value(int64(ge)),
			Le:     types.Int64Value(int64(le)),
		}
		entryObj, entryDiags := types.ObjectValueFrom(ctx, mcrPrefixListEntryAttributes, entryModel)
		diags = append(diags, entryDiags...)
		entriesList = append(entriesList, entryObj)
	}
	entries, entriesDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(mcrPrefixListEntryAttributes), entriesList)
	diags = append(diags, entriesDiags...)
	orm.Entries = entries
	return diags
}

func (pfFilterListModel *mcrPrefixFilterListModel) toAPIMCRPrefixFilterList(ctx context.Context) (*megaport.MCRPrefixFilterList, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	megaportPrefixFilterList := &megaport.MCRPrefixFilterList{
		Description:   pfFilterListModel.Description.ValueString(),
		AddressFamily: pfFilterListModel.AddressFamily.ValueString(),
	}

	if !pfFilterListModel.Entries.IsNull() {
		listEntries := []*mcrPrefixListEntryModel{}
		prefixListEntriesDiags := pfFilterListModel.Entries.ElementsAs(ctx, &listEntries, false)
		diags = append(diags, prefixListEntriesDiags...)
		for _, entry := range listEntries {
			megaportPrefixFilterList.Entries = append(megaportPrefixFilterList.Entries, &megaport.MCRPrefixListEntry{
				Action: entry.Action.ValueString(),
				Prefix: entry.Prefix.ValueString(),
				Ge:     int(entry.Ge.ValueInt64()),
				Le:     int(entry.Le.ValueInt64()),
			})
		}
	}
	return megaportPrefixFilterList, diags
}

// NewPortResource is a helper function to simplify the provider implemeantation.
func NewMCRResource() resource.Resource {
	return &mcrResource{}
}

// mcrResource is the resource implementation.
type mcrResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *mcrResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcr"
}

// Schema defines the schema for the resource.
func (r *mcrResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Megaport Cloud Router (MCR) Resource for the Megaport Terraform Provider. This can be used to create, modify, and delete Megaport MCRs. The MCR is a managed virtual router service that establishes Layer 3 connectivity on the worldwide Megaport software-defined network (SDN). MCR instances are preconfigured in data centers in key global routing zones. An MCR enables data transfer between multi-cloud or hybrid cloud networks, network service providers, and cloud service providers.",
		Attributes: map[string]schema.Attribute{
			"last_updated": schema.StringAttribute{
				Description: "Last updated by the Terraform provider.",
				Computed:    true,
			},
			"product_uid": schema.StringAttribute{
				Description: "UID identifier of the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_id": schema.Int64Attribute{
				Description: "Numeric ID of the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"product_name": schema.StringAttribute{
				Description: "Name of the product. Specify a name for the MCR that is easily identifiable as yours, particularly if you plan on provisioning more than one MCR.",
				Required:    true,
			},
			"product_type": schema.StringAttribute{
				Description: "Type of the product.",
				Computed:    true,
			},
			"provisioning_status": schema.StringAttribute{
				Description: "Provisioning status of the product.",
				Computed:    true,
			},
			"diversity_zone": schema.StringAttribute{
				Description: "Diversity zone of the product. If the parameter is not provided, a diversity zone will be automatically allocated.",
				Optional:    true,
			},
			"promo_code": schema.StringAttribute{
				Description: "Promo code is an optional string that can be used to enter a promotional code for the service order. The code is not validated, so if the code doesn't exist or doesn't work for the service, the request will still be successful.",
				Optional:    true,
			},
			"create_date": schema.StringAttribute{
				Description: "Date the product was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				Description: "User who created the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"port_speed": schema.Int64Attribute{
				Description: "Bandwidth speed of the product. The MCR can scale from 1 Gbps to 10 Gbps. The rate limit is an aggregate capacity that determines the speed for all connections through the MCR. MCR bandwidth is shared between all the Cloud Service Provider (CSP) connections added to it. The rate limit is fixed for the life of the service. MCR2 supports four speeds: 1000, 2500, 5000, and 10000 MBPS",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.OneOf(1000, 2500, 5000, 10000),
				},
			},
			"terminate_date": schema.StringAttribute{
				Description: "Date the product will be terminated.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"live_date": schema.StringAttribute{
				Description: "Date the product went live.",
				Computed:    true,
			},
			"market": schema.StringAttribute{
				Description: "Market the product is in.",
				Computed:    true,
			},
			"location_id": schema.Int64Attribute{
				Description: "The numeric location ID of the product. This value can be retrieved from the data source megaport_location.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"contract_term_months": schema.Int64Attribute{
				Description: "The term of the contract in months: valid values are 1, 12, 24, and 36.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(1, 12, 24, 36),
				},
			},
			"usage_algorithm": schema.StringAttribute{
				Description: "Usage algorithm of the product.",
				Computed:    true,
			},
			"company_uid": schema.StringAttribute{
				Description: "Megaport Company UID of the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cost_centre": schema.StringAttribute{
				Description: "A customer reference number to be included in billing information and invoices. Also known as the service level reference (SLR) number. Specify a unique identifying number for the product to be used for billing purposes, such as a cost center number or a unique customer ID. The service level reference number appears for each service under the Product section of the invoice. You can also edit this field for an existing service. Please note that a VXC associated with the MCR is not automatically updated with the MCR service level reference number.",
				Computed:    true,
				Optional:    true,
			},
			"contract_start_date": schema.StringAttribute{
				Description: "Contract start date of the product.",
				Computed:    true,
			},
			"contract_end_date": schema.StringAttribute{
				Description: "Contract end date of the product.",
				Computed:    true,
			},
			"secondary_name": schema.StringAttribute{
				Description: "Secondary name of the product.",
				Computed:    true,
			},
			"lag_primary": schema.BoolAttribute{
				Description: "Whether the product is a LAG primary.",
				Computed:    true,
			},
			"lag_id": schema.Int64Attribute{
				Description: "Numeric ID of the LAG.",
				Computed:    true,
			},
			"aggregation_id": schema.Int64Attribute{
				Description: "Numeric ID of the aggregation.",
				Computed:    true,
			},
			"company_name": schema.StringAttribute{
				Description: "Name of the company.",
				Computed:    true,
			},
			"marketplace_visibility": schema.BoolAttribute{
				Description: "Whether the product is visible in the Marketplace.",
				Computed:    true,
			},
			"asn": schema.Int64Attribute{
				Description: "Autonomous System Number (ASN) of the MCR in the MCR order configuration. Defaults to 133937 if not specified. For most configurations, the default ASN is appropriate. The ASN is used for BGP peering sessions on any VXCs connected to this MCR. See the documentation for your cloud providers before overriding the default value. For example, some public cloud services require the use of a public ASN and Microsoft blocks an ASN value of 65515 for Azure connections.",
				Optional:    true,
			},
			"vxc_permitted": schema.BoolAttribute{
				Description: "Whether VXC is permitted.",
				Computed:    true,
			},
			"vxc_auto_approval": schema.BoolAttribute{
				Description: "Whether VXC is auto approved.",
				Computed:    true,
			},
			"virtual": schema.BoolAttribute{
				Description: "Whether the product is virtual.",
				Computed:    true,
			},
			"buyout_port": schema.BoolAttribute{
				Description: "Whether the product is bought out.",
				Computed:    true,
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the product is locked.",
				Computed:    true,
			},
			"admin_locked": schema.BoolAttribute{
				Description: "Whether the product is admin locked.",
				Computed:    true,
			},
			"cancelable": schema.BoolAttribute{
				Description: "Whether the product is cancelable.",
				Computed:    true,
			},
			"attribute_tags": schema.MapAttribute{
				ElementType: types.StringType,
				Description: "Attribute tags of the product.",
				Computed:    true,
			},
			"prefix_filter_lists": schema.ListNestedAttribute{
				Description: "Prefix filter list associated with the product.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "Numeric ID of the prefix filter list.",
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"description": schema.StringAttribute{
							Description: "Description of the prefix filter list.",
							Required:    true,
						},
						"address_family": schema.StringAttribute{
							Description: "The IP address standard of the IP network addresses in the prefix filter list.",
							Required:    true,
						},
						"entries": schema.ListNestedAttribute{
							Description: "Entries in the prefix filter list.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.List{
								listvalidator.SizeBetween(1, 200),
							},
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"action": schema.StringAttribute{
										Description: "The action to take for the network address in the filter list. Accepted values are permit and deny.",
										Required:    true,
										Validators: []validator.String{
											stringvalidator.OneOf("permit", "deny"),
										},
									},
									"prefix": schema.StringAttribute{
										Description: "The network address of the prefix filter list entry.",
										Required:    true,
									},
									"ge": schema.Int64Attribute{
										Description: "The minimum starting prefix length to be matched. Valid values are from 0 to 32 (IPv4), or 0 to 128 (IPv6). The minimum (ge) must be no greater than or equal to the maximum value (le).",
										Optional:    true,
									},
									"le": schema.Int64Attribute{
										Description: "The maximum ending prefix length to be matched. The prefix length is greater than or equal to the minimum value (ge). Valid values are from 0 to 32 (IPv4), or 0 to 128 (IPv6), but the maximum must be no less than the minimum value (ge).",
										Optional:    true,
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

// Create a new resource.
func (r *mcrResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan mcrResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	buyReq := &megaport.BuyMCRRequest{
		Name:             plan.Name.ValueString(),
		Term:             int(plan.ContractTermMonths.ValueInt64()),
		PortSpeed:        int(plan.PortSpeed.ValueInt64()),
		LocationID:       int(plan.LocationID.ValueInt64()),
		CostCentre:       plan.CostCentre.ValueString(),
		PromoCode:        plan.PromoCode.ValueString(),
		WaitForProvision: true,
		WaitForTime:      waitForTime,
	}

	if !plan.ASN.IsNull() {
		buyReq.MCRAsn = int(plan.ASN.ValueInt64())
	}

	if !plan.DiversityZone.IsNull() {
		buyReq.DiversityZone = plan.DiversityZone.ValueString()
	}

	createdMCR, err := r.client.MCRService.BuyMCR(ctx, buyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating mcr",
			"Could not mcr with name "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	createdID := createdMCR.TechnicalServiceUID

	// get the created MCR
	mcr, err := r.client.MCRService.GetMCR(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading newly created mcr",
			"Could not read newly created mcr with ID "+createdID+": "+err.Error(),
		)
		return
	}

	// update the plan with the MCR info
	apiDiags := plan.fromAPIMCR(ctx, mcr)
	resp.Diagnostics.Append(apiDiags...)

	pfFilterLists := []*mcrPrefixFilterListModel{}

	// Create Prefix Filter List for MCR Upon Creation
	if !plan.PrefixFilterLists.IsNull() && len(plan.PrefixFilterLists.Elements()) > 0 { // Check if Prefix Filter List is not null and has elements
		listDiags := plan.PrefixFilterLists.ElementsAs(ctx, &pfFilterLists, false)
		resp.Diagnostics.Append(listDiags...)

		prefixFilterListObjs := []types.Object{}
		for _, pfFilterListModel := range pfFilterLists {
			megaportPrefixFilterList, listDiags := pfFilterListModel.toAPIMCRPrefixFilterList(ctx)
			resp.Diagnostics.Append(listDiags...)
			prefixFilterListReq := &megaport.CreateMCRPrefixFilterListRequest{
				MCRID:            createdID,
				PrefixFilterList: *megaportPrefixFilterList,
			}
			if pfFilterListModel.Entries.IsNull() {
				pfFilterListModel.Entries = types.ListNull(types.ObjectType{}.WithAttributeTypes(mcrPrefixListEntryAttributes))
			}
			createRes, err := r.client.MCRService.CreatePrefixFilterList(ctx, prefixFilterListReq)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating prefix filter list",
					"Could not create prefix filter list for MCR with ID "+createdID+": "+err.Error(),
				)
				return
			}
			pfFilterListModel.ID = types.Int64Value(int64(createRes.PrefixFilterListID))
			prefixFilterListObj, prefixFilterDiags := types.ObjectValueFrom(ctx, mcrPrefixFilterListModelAttributes, pfFilterListModel)
			resp.Diagnostics.Append(prefixFilterDiags...)
			prefixFilterListObjs = append(prefixFilterListObjs, prefixFilterListObj)
		}
		prefixFilterList, prefixFilterListsDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListModelAttributes), prefixFilterListObjs)
		resp.Diagnostics.Append(prefixFilterListsDiags...)
		plan.PrefixFilterLists = prefixFilterList
	} else {
		plan.PrefixFilterLists = types.ListNull(types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListModelAttributes))
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *mcrResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state mcrResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed mcr value from API
	mcr, err := r.client.MCRService.GetMCR(ctx, state.UID.ValueString())
	if err != nil {
		// MCR has been deleted or is not found
		if mpErr, ok := err.(*megaport.ErrorResponse); ok {
			if mpErr.Response.StatusCode == http.StatusNotFound ||
				(mpErr.Response.StatusCode == http.StatusBadRequest && strings.Contains(mpErr.Message, "Could not find a service with UID")) {
				resp.State.RemoveResource(ctx)
				return
			}
		}

		resp.Diagnostics.AddError(
			"Error Reading MCR",
			"Could not read MCR with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	// If the MCR has been deleted
	if mcr.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED {
		resp.State.RemoveResource(ctx)
		return
	}

	apiDiags := state.fromAPIMCR(ctx, mcr)
	resp.Diagnostics.Append(apiDiags...)

	pfFilterListModels := []*mcrPrefixFilterListModel{}

	if !state.PrefixFilterLists.IsNull() {
		pfFilterListStateDiags := state.PrefixFilterLists.ElementsAs(ctx, &pfFilterListModels, false)
		resp.Diagnostics.Append(pfFilterListStateDiags...)
	} else {
		state.PrefixFilterLists = types.ListNull(types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListModelAttributes))
	}

	prefixFilterLists, prefixFilterListErr := r.client.MCRService.ListMCRPrefixFilterLists(ctx, state.UID.ValueString())
	if prefixFilterListErr != nil {
		resp.Diagnostics.AddError(
			"Error Reading Prefix Filter Lists",
			"Could not read prefix filter lists for MCR with ID "+state.UID.ValueString()+": "+prefixFilterListErr.Error(),
		)
		return
	}
	detailedPrefixFilterLists := []*megaport.MCRPrefixFilterList{}
	wg := sync.WaitGroup{}
	mux := sync.Mutex{}
	errs := []error{}
	for _, l := range prefixFilterLists {
		wg.Add(1)
		go func(list *megaport.PrefixFilterList) {
			defer wg.Done()
			detailedList, err := r.client.MCRService.GetMCRPrefixFilterList(ctx, state.UID.ValueString(), list.Id)
			if err != nil {
				mux.Lock()
				errs = append(errs, err)
				mux.Unlock()
			}
			mux.Lock()
			detailedPrefixFilterLists = append(detailedPrefixFilterLists, detailedList)
			mux.Unlock()
		}(l)
	}
	wg.Wait()
	if len(errs) > 0 {
		var errStr string
		for _, err := range errs {
			errStr += err.Error() + ", "
		}
		resp.Diagnostics.AddError(
			"Error Reading Prefix Filter Lists",
			"Could not read prefix filter lists for MCR with ID "+state.UID.ValueString()+": "+errStr,
		)
		return
	}

	sort.Slice(detailedPrefixFilterLists, func(i, j int) bool {
		return detailedPrefixFilterLists[i].ID < detailedPrefixFilterLists[j].ID
	})

	parsedListObjs := []types.Object{}

	if len(detailedPrefixFilterLists) > 0 {
		for _, detailedList := range detailedPrefixFilterLists {
			parsedModel := &mcrPrefixFilterListModel{}
			parsedDiags := parsedModel.fromAPIMCRPrefixFilterList(ctx, detailedList)
			if parsedDiags.HasError() {
				return
			}
			resp.Diagnostics.Append(parsedDiags...)
			parsedObj, parsedDiags := types.ObjectValueFrom(ctx, mcrPrefixFilterListModelAttributes, parsedModel)
			resp.Diagnostics.Append(parsedDiags...)
			parsedListObjs = append(parsedListObjs, parsedObj)
		}
		parsedLists, parsedDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListModelAttributes), parsedListObjs)
		resp.Diagnostics.Append(parsedDiags...)
		state.PrefixFilterLists = parsedLists
	} else if !state.PrefixFilterLists.IsNull() && len(state.PrefixFilterLists.Elements()) == 0 { // If list is empty but not null
		emptyList := []types.Object{}
		pfFilterLists, pfFilterListDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListModelAttributes), emptyList)
		resp.Diagnostics.Append(pfFilterListDiags...)
		state.PrefixFilterLists = pfFilterLists
	} else { // If list is empty and null
		state.PrefixFilterLists = types.ListNull(types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListModelAttributes))
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *mcrResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state mcrResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check on changes
	var name, costCentre string
	var marketplaceVisibility bool
	if !plan.Name.IsNull() && !plan.Name.Equal(state.Name) {
		name = plan.Name.ValueString()
	} else {
		name = state.Name.ValueString()
	}
	if !plan.CostCentre.IsNull() && !plan.CostCentre.Equal(state.CostCentre) {
		costCentre = plan.CostCentre.ValueString()
	} else {
		costCentre = state.CostCentre.ValueString()
	}
	if !plan.MarketplaceVisibility.IsNull() && !plan.MarketplaceVisibility.Equal(state.MarketplaceVisibility) {
		marketplaceVisibility = plan.MarketplaceVisibility.ValueBool()
	} else {
		marketplaceVisibility = state.MarketplaceVisibility.ValueBool()
	}

	_, err := r.client.MCRService.ModifyMCR(ctx, &megaport.ModifyMCRRequest{
		MCRID:                 plan.UID.ValueString(),
		Name:                  name,
		MarketplaceVisibility: &marketplaceVisibility,
		CostCentre:            costCentre,
		WaitForUpdate:         true,
		WaitForTime:           waitForTime,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating MCR",
			"Could not update MCR, unexpected error: "+err.Error(),
		)
		return
	}

	// Get refreshed mcr value from API
	mcr, err := r.client.MCRService.GetMCR(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading MCR",
			"Could not read MCR with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	apiDiags := state.fromAPIMCR(ctx, mcr)
	resp.Diagnostics.Append(apiDiags...)

	statePrefixFilterListMap := map[int64]*mcrPrefixFilterListModel{}
	statePrefixFilterLists := []*mcrPrefixFilterListModel{}
	planPrefixFilterLists := []*mcrPrefixFilterListModel{}
	planPrefixFilterListMap := map[int64]*mcrPrefixFilterListModel{}

	statePrefixFilterListDiags := state.PrefixFilterLists.ElementsAs(ctx, &statePrefixFilterLists, false)
	resp.Diagnostics.Append(statePrefixFilterListDiags...)

	for _, statePrefixFilterList := range statePrefixFilterLists {
		statePrefixFilterListMap[statePrefixFilterList.ID.ValueInt64()] = statePrefixFilterList
	}

	planPrefixFilterListDiags := plan.PrefixFilterLists.ElementsAs(ctx, &planPrefixFilterLists, false)
	resp.Diagnostics.Append(planPrefixFilterListDiags...)

	for _, planPrefixFilterList := range planPrefixFilterLists {
		planPrefixFilterListMap[planPrefixFilterList.ID.ValueInt64()] = planPrefixFilterList
	}

	wg := sync.WaitGroup{}
	mux := sync.Mutex{}
	errs := []error{}
	for _, planModel := range planPrefixFilterLists {
		wg.Add(1)
		go func(planModel *mcrPrefixFilterListModel) {
			defer wg.Done()
			// Check if the prefix filter list exists in the state
			if statePrefixFilterList, ok := statePrefixFilterListMap[planModel.ID.ValueInt64()]; ok {
				// Check if there are any changes to the prefix filter list, if so, update.
				if !planModel.Description.Equal(statePrefixFilterList.Description) || !planModel.AddressFamily.Equal(statePrefixFilterList.AddressFamily) || !planModel.Entries.Equal(statePrefixFilterList.Entries) {
					apiPrefixFilterList, apiPrefixFilterListDiags := planModel.toAPIMCRPrefixFilterList(ctx)
					resp.Diagnostics.Append(apiPrefixFilterListDiags...)
					_, modifyErr := r.client.MCRService.ModifyMCRPrefixFilterList(ctx, state.UID.ValueString(), int(planModel.ID.ValueInt64()), apiPrefixFilterList)
					if modifyErr != nil {
						mux.Lock()
						errs = append(errs, modifyErr)
						mux.Unlock()
					}
				}
				// If the prefix filter list does not exist in the state, create it.
			} else {
				apiPrefixFilterList, apiPrefixFilterListDiags := planModel.toAPIMCRPrefixFilterList(ctx)
				resp.Diagnostics.Append(apiPrefixFilterListDiags...)
				_, createErr := r.client.MCRService.CreatePrefixFilterList(ctx, &megaport.CreateMCRPrefixFilterListRequest{
					MCRID:            state.UID.ValueString(),
					PrefixFilterList: *apiPrefixFilterList,
				})
				if createErr != nil {
					mux.Lock()
					errs = append(errs, createErr)
					mux.Unlock()
				}
			}
		}(planModel)
	}
	wg.Wait()
	if len(errs) > 0 {
		var errStr string
		for _, err := range errs {
			errStr += err.Error() + ", "
		}
		resp.Diagnostics.AddError(
			"Error Updating Prefix Filter Lists",
			"Could not update prefix filter lists for MCR with ID "+state.UID.ValueString()+": "+errStr,
		)
		return
	}

	wg2 := sync.WaitGroup{}
	for _, stateModel := range statePrefixFilterLists {
		wg2.Add(1)
		go func(stateModel *mcrPrefixFilterListModel) {
			defer wg2.Done()
			// If the prefix filter list does not exist in the plan, delete it.
			if _, ok := planPrefixFilterListMap[stateModel.ID.ValueInt64()]; !ok {
				_, deleteErr := r.client.MCRService.DeleteMCRPrefixFilterList(ctx, state.UID.ValueString(), int(stateModel.ID.ValueInt64()))
				if deleteErr != nil {
					mux.Lock()
					errs = append(errs, deleteErr)
					mux.Unlock()
				}
			}
		}(stateModel)
	}
	wg2.Wait()
	if len(errs) > 0 {
		var errStr string
		for _, err := range errs {
			errStr += err.Error() + ", "
		}
		resp.Diagnostics.AddError(
			"Error Deleting Prefix Filter Lists",
			"Could not delete prefix filter lists for MCR with ID "+state.UID.ValueString()+": "+errStr,
		)
		return
	}

	pfFilterListModels := []*mcrPrefixFilterListModel{}

	if !state.PrefixFilterLists.IsNull() {
		pfFilterListStateDiags := state.PrefixFilterLists.ElementsAs(ctx, &pfFilterListModels, false)
		resp.Diagnostics.Append(pfFilterListStateDiags...)
	} else {
		state.PrefixFilterLists = types.ListNull(types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListModelAttributes))
	}

	prefixFilterLists, prefixFilterListErr := r.client.MCRService.ListMCRPrefixFilterLists(ctx, state.UID.ValueString())
	if prefixFilterListErr != nil {
		resp.Diagnostics.AddError(
			"Error Reading Prefix Filter Lists",
			"Could not read prefix filter lists for MCR with ID "+state.UID.ValueString()+": "+prefixFilterListErr.Error(),
		)
		return
	}
	detailedPrefixFilterLists := []*megaport.MCRPrefixFilterList{}
	wg3 := sync.WaitGroup{}
	for _, l := range prefixFilterLists {
		wg3.Add(1)
		go func(list *megaport.PrefixFilterList) {
			defer wg3.Done()
			detailedList, err := r.client.MCRService.GetMCRPrefixFilterList(ctx, state.UID.ValueString(), list.Id)
			if err != nil {
				mux.Lock()
				errs = append(errs, err)
				mux.Unlock()
			}
			mux.Lock()
			detailedPrefixFilterLists = append(detailedPrefixFilterLists, detailedList)
			mux.Unlock()
		}(l)
	}
	wg3.Wait()
	if len(errs) > 0 {
		var errStr string
		for _, err := range errs {
			errStr += err.Error() + ", "
		}
		resp.Diagnostics.AddError(
			"Error Reading Prefix Filter Lists",
			"Could not read prefix filter lists for MCR with ID "+state.UID.ValueString()+": "+errStr,
		)
		return
	}

	sort.Slice(detailedPrefixFilterLists, func(i, j int) bool {
		return detailedPrefixFilterLists[i].ID < detailedPrefixFilterLists[j].ID
	})

	parsedListObjs := []types.Object{}

	if len(detailedPrefixFilterLists) > 0 {
		for _, detailedList := range detailedPrefixFilterLists {
			parsedModel := &mcrPrefixFilterListModel{}
			parsedDiags := parsedModel.fromAPIMCRPrefixFilterList(ctx, detailedList)
			if parsedDiags.HasError() {
				return
			}
			resp.Diagnostics.Append(parsedDiags...)
			parsedObj, parsedDiags := types.ObjectValueFrom(ctx, mcrPrefixFilterListModelAttributes, parsedModel)
			resp.Diagnostics.Append(parsedDiags...)
			parsedListObjs = append(parsedListObjs, parsedObj)
		}
		parsedLists, parsedDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListModelAttributes), parsedListObjs)
		resp.Diagnostics.Append(parsedDiags...)
		state.PrefixFilterLists = parsedLists

	} else if !plan.PrefixFilterLists.IsNull() && len(plan.PrefixFilterLists.Elements()) == 0 { // If list is empty but not null
		state.PrefixFilterLists = plan.PrefixFilterLists
	} else { // If list is empty and null
		state.PrefixFilterLists = types.ListNull(types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListModelAttributes))
	}

	// Update the state with the new values
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mcrResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state mcrResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	_, err := r.client.MCRService.DeleteMCR(ctx, &megaport.DeleteMCRRequest{
		MCRID:     state.UID.ValueString(),
		DeleteNow: true,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting MCR",
			"Could not delete MCR, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *mcrResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client
}

func (r *mcrResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}
