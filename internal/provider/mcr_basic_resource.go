package provider

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

var (
	// Ensure the implementation satisfies the expected interfaces.
	_ resource.Resource                = &mcrBasicResource{}
	_ resource.ResourceWithConfigure   = &mcrBasicResource{}
	_ resource.ResourceWithImportState = &mcrBasicResource{}
)

// mcrBasicResourceModel maps the resource schema data.
type mcrBasicResourceModel struct {
	ID                    types.Int64  `tfsdk:"product_id"`
	UID                   types.String `tfsdk:"product_uid"`
	Name                  types.String `tfsdk:"product_name"`
	Type                  types.String `tfsdk:"product_type"`
	CostCentre            types.String `tfsdk:"cost_centre"`
	PortSpeed             types.Int64  `tfsdk:"port_speed"`
	Market                types.String `tfsdk:"market"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	UsageAlgorithm        types.String `tfsdk:"usage_algorithm"`
	MarketplaceVisibility types.Bool   `tfsdk:"marketplace_visibility"`
	VXCPermitted          types.Bool   `tfsdk:"vxc_permitted"`
	VXCAutoApproval       types.Bool   `tfsdk:"vxc_auto_approval"`
	CompanyUID            types.String `tfsdk:"company_uid"`
	ContractTermMonths    types.Int64  `tfsdk:"contract_term_months"`
	ASN                   types.Int64  `tfsdk:"asn"`
	DiversityZone         types.String `tfsdk:"diversity_zone"`
	PromoCode             types.String `tfsdk:"promo_code"`

	Locked      types.Bool `tfsdk:"locked"`
	AdminLocked types.Bool `tfsdk:"admin_locked"`
	Cancelable  types.Bool `tfsdk:"cancelable"`

	PrefixFilterLists types.List `tfsdk:"prefix_filter_lists"`

	ResourceTags types.Map `tfsdk:"resource_tags"`
}

// fromAPIMCR maps the API MCR response to the resource schema.
func (orm *mcrBasicResourceModel) fromAPIMCR(ctx context.Context, m *megaport.MCR, tags map[string]string) diag.Diagnostics {
	apiDiags := diag.Diagnostics{}

	asn := m.Resources.VirtualRouter.ASN
	if asn != 0 {
		orm.ASN = types.Int64Value(int64(asn))
	}

	orm.ID = types.Int64Value(int64(m.ID))
	orm.UID = types.StringValue(m.UID)
	orm.Name = types.StringValue(m.Name)
	orm.Type = types.StringValue(m.Type)
	orm.CostCentre = types.StringValue(m.CostCentre)
	orm.PortSpeed = types.Int64Value(int64(m.PortSpeed))
	orm.Market = types.StringValue(m.Market)
	orm.LocationID = types.Int64Value(int64(m.LocationID))
	orm.UsageAlgorithm = types.StringValue(m.UsageAlgorithm)
	orm.MarketplaceVisibility = types.BoolValue(m.MarketplaceVisibility)
	orm.VXCPermitted = types.BoolValue(m.VXCPermitted)
	orm.VXCAutoApproval = types.BoolValue(m.VXCAutoApproval)

	orm.CompanyUID = types.StringValue(m.CompanyUID)

	orm.ContractTermMonths = types.Int64Value(int64(m.ContractTermMonths))

	orm.Locked = types.BoolValue(m.Locked)
	orm.AdminLocked = types.BoolValue(m.AdminLocked)
	orm.Cancelable = types.BoolValue(m.Cancelable)
	orm.DiversityZone = types.StringValue(m.DiversityZone)

	if len(tags) > 0 {
		resourceTags, tagDiags := types.MapValueFrom(ctx, types.StringType, tags)
		apiDiags = append(apiDiags, tagDiags...)
		orm.ResourceTags = resourceTags
	} else {
		orm.ResourceTags = types.MapNull(types.StringType)
	}

	return apiDiags
}

// NewPortResource is a helper function to simplify the provider implemeantation.
func NewMCRBasicResource() resource.Resource {
	return &mcrBasicResource{}
}

// mcrBasicResource is the resource implementation.
type mcrBasicResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *mcrBasicResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcr_basic"
}

// Schema defines the schema for the resource.
func (r *mcrBasicResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Megaport Cloud Router (MCR) Resource for the Megaport Terraform Provider. This can be used to create, modify, and delete Megaport MCRs. The MCR is a managed virtual router service that establishes Layer 3 connectivity on the worldwide Megaport software-defined network (SDN). MCR instances are preconfigured in data centers in key global routing zones. An MCR enables data transfer between multi-cloud or hybrid cloud networks, network service providers, and cloud service providers.",
		Attributes: map[string]schema.Attribute{
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"diversity_zone": schema.StringAttribute{
				Description: "Diversity zone of the product. If the parameter is not provided, a diversity zone will be automatically allocated.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"promo_code": schema.StringAttribute{
				Description: "Promo code is an optional string that can be used to enter a promotional code for the service order. The code is not validated, so if the code doesn't exist or doesn't work for the service, the request will still be successful.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"port_speed": schema.Int64Attribute{
				Description: "Bandwidth speed of the product. The MCR can scale from 1 Gbps to 100 Gbps. The rate limit is an aggregate capacity that determines the speed for all connections through the MCR. MCR bandwidth is shared between all the Cloud Service Provider (CSP) connections added to it. The rate limit is fixed for the life of the service. MCR2 supports seven speeds: 1000, 2500, 5000, 10000, 25000, 50000, and 100000 MBPS",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.OneOf(1000, 2500, 5000, 10000, 25000, 50000, 100000),
				},
			},
			"market": schema.StringAttribute{
				Description: "Market the product is in.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"location_id": schema.Int64Attribute{
				Description: "The numeric location ID of the product. This value can be retrieved from the data source megaport_location.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"contract_term_months": schema.Int64Attribute{
				Description: "The term of the contract in months: valid values are 1, 12, 24, and 36. To set the product to a month-to-month contract with no minimum term, set the value to 1.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(1, 12, 24, 36),
				},
			},
			"usage_algorithm": schema.StringAttribute{
				Description: "Usage algorithm of the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"aggregation_id": schema.Int64Attribute{
				Description: "Numeric ID of the aggregation.",
				Computed:    true,
			},
			"marketplace_visibility": schema.BoolAttribute{
				Description: "Whether the product is visible in the Marketplace.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"asn": schema.Int64Attribute{
				Description: "Autonomous System Number (ASN) of the MCR in the MCR order configuration. Defaults to 133937 if not specified. For most configurations, the default ASN is appropriate. The ASN is used for BGP peering sessions on any VXCs connected to this MCR. See the documentation for your cloud providers before overriding the default value. For example, some public cloud services require the use of a public ASN and Microsoft blocks an ASN value of 65515 for Azure connections.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"vxc_permitted": schema.BoolAttribute{
				Description: "Whether VXC is permitted.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"vxc_auto_approval": schema.BoolAttribute{
				Description: "Whether VXC is auto approved.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the product is locked.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"admin_locked": schema.BoolAttribute{
				Description: "Whether the product is admin locked.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"cancelable": schema.BoolAttribute{
				Description: "Whether the product is cancelable.",
				Computed:    true,
			},
			"resource_tags": schema.MapAttribute{
				Description: "The resource tags associated with the product.",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"prefix_filter_lists": schema.ListNestedAttribute{
				Description: "Prefix filter list associated with the product.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
					EmptyPrefixFilterListIfNull(),
				},
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
func (r *mcrBasicResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan mcrBasicResourceModel
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

	if !plan.ResourceTags.IsNull() {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		buyReq.ResourceTags = tagMap
	}

	err := r.client.MCRService.ValidateMCROrder(ctx, buyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Validation error while attempting to create MCR",
			"Validation error while attempting to create MCR with name "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
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

	tags, err := r.client.MCRService.ListMCRResourceTags(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading resource tags",
			"Could not read resource tags for MCR with ID "+createdID+": "+err.Error(),
		)
		return
	}

	// update the plan with the MCR info
	apiDiags := plan.fromAPIMCR(ctx, mcr, tags)
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
		emptyList := []types.Object{}
		pfFilterLists, pfFilterListDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListModelAttributes), emptyList)
		resp.Diagnostics.Append(pfFilterListDiags...)
		plan.PrefixFilterLists = pfFilterLists
	}
	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *mcrBasicResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state mcrBasicResourceModel
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

	tags, err := r.client.MCRService.ListMCRResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading resource tags",
			"Could not read resource tags for MCR with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	apiDiags := state.fromAPIMCR(ctx, mcr, tags)
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

	// Create a RateLimiter with a burst size of 10 and a refill speed of 100 milliseconds
	rateLimiter := NewRateLimiter(10, 1000*time.Millisecond)

	for _, l := range prefixFilterLists {
		wg.Add(1)
		go func(list *megaport.PrefixFilterList) {
			defer wg.Done()
			// Get a token from the rate limiter to apply rate limiting

			<-rateLimiter.rateLimitCh

			detailedList, err := r.client.MCRService.GetMCRPrefixFilterList(ctx, state.UID.ValueString(), list.Id)
			if err != nil {
				mux.Lock()
				errs = append(errs, err)
				mux.Unlock()
				return
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
	} else {
		emptyList := []types.Object{}
		pfFilterLists, pfFilterListDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListModelAttributes), emptyList)
		resp.Diagnostics.Append(pfFilterListDiags...)
		state.PrefixFilterLists = pfFilterLists
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *mcrBasicResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state mcrBasicResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check on changes
	var name, costCentre string
	var marketplaceVisibility bool
	var contractTermMonths *int
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
	if !plan.ContractTermMonths.IsNull() && !plan.ContractTermMonths.Equal(state.ContractTermMonths) {
		months := int(plan.ContractTermMonths.ValueInt64())
		contractTermMonths = &months
	}

	_, err := r.client.MCRService.ModifyMCR(ctx, &megaport.ModifyMCRRequest{
		MCRID:                 plan.UID.ValueString(),
		Name:                  name,
		MarketplaceVisibility: &marketplaceVisibility,
		ContractTermMonths:    contractTermMonths,
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

	// If change in resource tags from state
	if !plan.ResourceTags.Equal(state.ResourceTags) {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := r.client.MCRService.UpdateMCRResourceTags(ctx, plan.UID.ValueString(), tagMap)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating mcr resource tags",
				"Could not update mcr resource tags with ID "+plan.UID.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	tags, err := r.client.MCRService.ListMCRResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading resource tags",
			"Could not read resource tags for MCR with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	apiDiags := state.fromAPIMCR(ctx, mcr, tags)
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

	rateLimiter := NewRateLimiter(10, 1000*time.Millisecond)

	for _, planModel := range planPrefixFilterLists {
		wg.Add(1)
		go func(planModel *mcrPrefixFilterListModel) {
			defer wg.Done()
			// Get a token from the rate limiter to apply rate limiting
			<-rateLimiter.rateLimitCh

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
			"Error Modifying Prefix Filter Lists",
			"Could not modify prefix filter lists for MCR with ID "+state.UID.ValueString()+": "+errStr,
		)
		return
	}

	// Create a RateLimiter with a burst size of 10 and a refill speed of 100 milliseconds for delete operations
	deleteRateLimiter := NewRateLimiter(10, 1000*time.Millisecond)

	for _, stateModel := range statePrefixFilterLists {
		wg.Add(1)
		go func(stateModel *mcrPrefixFilterListModel) {
			defer wg.Done()

			<-deleteRateLimiter.rateLimitCh

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
	wg.Wait()

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

	for _, l := range prefixFilterLists {
		wg.Add(1)
		go func(list *megaport.PrefixFilterList) {
			defer wg.Done()

			<-rateLimiter.rateLimitCh

			detailedList, err := r.client.MCRService.GetMCRPrefixFilterList(ctx, state.UID.ValueString(), list.Id)
			if err != nil {
				mux.Lock()
				errs = append(errs, err)
				mux.Unlock()
				return
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

	} else {
		emptyList := []types.Object{}
		pfFilterLists, pfFilterListDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListModelAttributes), emptyList)
		resp.Diagnostics.Append(pfFilterListDiags...)
		state.PrefixFilterLists = pfFilterLists
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mcrBasicResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state mcrBasicResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	_, err := r.client.MCRService.DeleteMCR(ctx, &megaport.DeleteMCRRequest{
		MCRID:      state.UID.ValueString(),
		DeleteNow:  true,
		SafeDelete: true,
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
func (r *mcrBasicResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *mcrBasicResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}
