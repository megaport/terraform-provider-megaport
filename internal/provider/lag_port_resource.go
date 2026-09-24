package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &lagPortResource{}
	_ resource.ResourceWithConfigure   = &lagPortResource{}
	_ resource.ResourceWithImportState = &lagPortResource{}
)

// lagPortResourceModel maps the resource schema data.
type lagPortResourceModel struct {
	LastUpdated types.String `tfsdk:"last_updated"`

	UID                   types.String `tfsdk:"product_uid"`
	ID                    types.Int64  `tfsdk:"product_id"`
	Name                  types.String `tfsdk:"product_name"`
	ProvisioningStatus    types.String `tfsdk:"provisioning_status"`
	CreateDate            types.String `tfsdk:"create_date"`
	CreatedBy             types.String `tfsdk:"created_by"`
	PortSpeed             types.Int64  `tfsdk:"port_speed"`
	TerminateDate         types.String `tfsdk:"terminate_date"`
	LiveDate              types.String `tfsdk:"live_date"`
	Market                types.String `tfsdk:"market"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	UsageAlgorithm        types.String `tfsdk:"usage_algorithm"`
	MarketplaceVisibility types.Bool   `tfsdk:"marketplace_visibility"`
	VXCPermitted          types.Bool   `tfsdk:"vxc_permitted"`
	VXCAutoApproval       types.Bool   `tfsdk:"vxc_auto_approval"`
	CompanyUID            types.String `tfsdk:"company_uid"`
	CostCentre            types.String `tfsdk:"cost_centre"`
	ContractStartDate     types.String `tfsdk:"contract_start_date"`
	ContractEndDate       types.String `tfsdk:"contract_end_date"`
	ContractTermMonths    types.Int64  `tfsdk:"contract_term_months"`
	Virtual               types.Bool   `tfsdk:"virtual"`
	Locked                types.Bool   `tfsdk:"locked"`
	Cancelable            types.Bool   `tfsdk:"cancelable"`
	DiversityZone         types.String `tfsdk:"diversity_zone"`
	PromoCode             types.String `tfsdk:"promo_code"`

	LagCount    types.Int64 `tfsdk:"lag_count"`
	LagPortUIDs types.List  `tfsdk:"lag_port_uids"`

	Resources    types.Object `tfsdk:"resources"`
	ResourceTags types.Map    `tfsdk:"resource_tags"`
}

func (orm *lagPortResourceModel) fromAPIPort(ctx context.Context, p *megaport.Port, tags map[string]string) diag.Diagnostics {
	diags := diag.Diagnostics{}

	orm.UID = types.StringValue(p.UID)
	orm.ID = types.Int64Value(int64(p.ID))
	orm.Cancelable = types.BoolValue(p.Cancelable)
	orm.CompanyUID = types.StringValue(p.CompanyUID)
	orm.ContractTermMonths = types.Int64Value(int64(p.ContractTermMonths))
	orm.CostCentre = types.StringValue(p.CostCentre)
	orm.CreatedBy = types.StringValue(p.CreatedBy)
	orm.DiversityZone = diversityZoneFromAPI(orm.DiversityZone, p.DiversityZone, p.UID, &diags)
	orm.LocationID = types.Int64Value(int64(p.LocationID))
	orm.Locked = types.BoolValue(p.Locked)
	orm.Market = types.StringValue(p.Market)
	orm.MarketplaceVisibility = types.BoolValue(p.MarketplaceVisibility)
	orm.Name = types.StringValue(p.Name)
	orm.PortSpeed = types.Int64Value(int64(p.PortSpeed))
	orm.ProvisioningStatus = types.StringValue(p.ProvisioningStatus)
	orm.UsageAlgorithm = types.StringValue(p.UsageAlgorithm)
	orm.VXCAutoApproval = types.BoolValue(p.VXCAutoApproval)
	orm.VXCPermitted = types.BoolValue(p.VXCPermitted)
	orm.Virtual = types.BoolValue(p.Virtual)
	orm.LagCount = types.Int64Value(int64(p.LagCount))

	if p.CreateDate != nil {
		orm.CreateDate = types.StringValue(p.CreateDate.Format(time.RFC850))
	} else {
		orm.CreateDate = types.StringNull()
	}
	if p.LiveDate != nil {
		orm.LiveDate = types.StringValue(p.LiveDate.Format(time.RFC850))
	} else {
		orm.LiveDate = types.StringNull()
	}
	if p.ContractStartDate != nil {
		orm.ContractStartDate = types.StringValue(p.ContractStartDate.Format(time.RFC850))
	} else {
		orm.ContractStartDate = types.StringNull()
	}
	if p.ContractEndDate != nil {
		orm.ContractEndDate = types.StringValue(p.ContractEndDate.Format(time.RFC850))
	} else {
		orm.ContractEndDate = types.StringNull()
	}

	if p.TerminateDate != nil {
		orm.TerminateDate = types.StringValue(p.TerminateDate.Format(time.RFC850))
	} else {
		orm.TerminateDate = types.StringNull()
	}

	resourcesModel := &portResourcesModel{}
	interfaceObj, interfaceDiags := fromAPIPortInterface(ctx, &p.VXCResources.Interface)
	diags = append(diags, interfaceDiags...)
	resourcesModel.Interface = interfaceObj
	resourcesObject, resourcesDiags := types.ObjectValueFrom(ctx, portResourcesAttrs, resourcesModel)
	diags = append(diags, resourcesDiags...)
	orm.Resources = resourcesObject

	if len(tags) > 0 {
		resourceTags, tagDiags := types.MapValueFrom(ctx, types.StringType, tags)
		diags = append(diags, tagDiags...)
		orm.ResourceTags = resourceTags
	} else {
		orm.ResourceTags = types.MapNull(types.StringType)
	}

	return diags
}

// NewPortResource is a helper function to simplify the provider implementation.
func NewLagPortResource() resource.Resource {
	return &lagPortResource{}
}

// lagPortResource is the resource implementation.
type lagPortResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *lagPortResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lag_port"
}

// Schema defines the schema for the resource.
func (r *lagPortResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Link Aggregation Group (LAG) Port Resource for the Megaport Terraform Provider. This can be used to create, modify, and delete Megaport LAG Ports. A LAG bundles physical ports to create a single data path, where the traffic load is distributed among the ports to increase overall connection reliability.",
		Attributes: map[string]schema.Attribute{
			"last_updated": schema.StringAttribute{
				Description: "The last time the resource was updated.",
				Computed:    true,
			},
			"product_uid": schema.StringAttribute{
				Description: "The unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_id": schema.Int64Attribute{
				Description: "The numeric ID of the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"product_name": schema.StringAttribute{
				Description: "The name of the product.",
				Required:    true,
			},
			"provisioning_status": schema.StringAttribute{
				Description: "The provisioning status of the LAG port. This field represents the current state (e.g., CONFIGURED, LIVE, DECOMMISSIONED) and may transition through multiple states during the port lifecycle. During import, this field will populate from the API and may show as changing from unknown to its actual value on first apply - this is expected behavior.",
				Computed:    true,
			},
			"create_date": schema.StringAttribute{
				Description: "The date the product was created.",
				Computed:    true,
			},
			"created_by": schema.StringAttribute{
				Description: "The user who created the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"port_speed": schema.Int64Attribute{
				Description: "The speed of the port in Mbps. Can be 10000 (10 G), 10000 (10 G), 100000 (100 G), or 400000 (400G) where available..",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.OneOf(10000, 100000, 400000),
				},
			},
			"terminate_date": schema.StringAttribute{
				Description: "The date the product will be terminated.",
				Computed:    true,
			},
			"live_date": schema.StringAttribute{
				Description: "The date the product went live.",
				Computed:    true,
			},
			"market": schema.StringAttribute{
				Description: "The market the product is in.",
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
				Description: "The term of the contract in months: valid values are 1, 12, 24, 36, 48, and 60. To set the product to a month-to-month contract with no minimum term, set the value to 1. For a managed account whose partner requires order approval, a term increase on a live LAG creates an approval request for each port. Until approval, each port also keeps its old `name`, `cost_centre`, and `marketplace_visibility`, and the apply fails with an inconsistent result error.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(1, 12, 24, 36, 48, 60),
				},
			},
			"promo_code": schema.StringAttribute{
				Description: "Promo code is an optional string that can be used to enter a promotional code for the service order. The code is not validated, so if the code doesn't exist or doesn't work for the service, the request will still be successful.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"usage_algorithm": schema.StringAttribute{
				Description: "The usage algorithm for the product.",
				Computed:    true,
			},
			"company_uid": schema.StringAttribute{
				Description: "The unique identifier of the company.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cost_centre": schema.StringAttribute{
				Description: "A customer reference number to be included in billing information and invoices. Also known as the service level reference (SLR) number. Specify a unique identifying number for the product to be used for billing purposes, such as a cost center number or a unique customer ID. The service level reference number appears for each service under the Product section of the invoice. You can also edit this field for an existing service.",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"contract_start_date": schema.StringAttribute{
				Description: "The date the contract started.",
				Computed:    true,
			},
			"contract_end_date": schema.StringAttribute{
				Description: "The date the contract ends.",
				Computed:    true,
			},
			"marketplace_visibility": schema.BoolAttribute{
				Description: "Whether the product is visible in the marketplace.",
				Required:    true,
			},
			"vxc_permitted": schema.BoolAttribute{
				Description: "Whether VXC is permitted on this product.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"vxc_auto_approval": schema.BoolAttribute{
				Description: "Whether VXC is auto-approved on this product.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"virtual": schema.BoolAttribute{
				Description: "Whether the product is virtual. Always false for LAG orders.",
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
			"cancelable": schema.BoolAttribute{
				Description: "Whether the product is cancelable.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"diversity_zone": schema.StringAttribute{
				Description: "The diversity zone of the product. Once known, this value is preserved if a later read reports it empty, since that's typically a transient backend gap rather than a real change. If the empty value is a genuine correction rather than a gap, remove or update `diversity_zone` in your configuration first; optionally run `terraform state rm` followed by `terraform import` to reset the stored value.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"lag_count": schema.Int64Attribute{
				Description: "The number of LAG ports. Valid values are between 1 and 8. Raising it adds ports to the existing LAG and leaves the current ports in place. Lowering it replaces the LAG, because the API has no call to remove a member.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 8),
				},
			},
			"lag_port_uids": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "The unique identifiers of the LAG ports.",
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"resources": schema.SingleNestedAttribute{
				Description: "Resources attached to port.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"interface": schema.SingleNestedAttribute{
						Description: "Port interface details.",
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"demarcation": schema.StringAttribute{
								Description: "The demarcation of the interface.",
								Computed:    true,
							},
							"up": schema.Int64Attribute{
								Description: "The up status of the interface.",
								Computed:    true,
							},
						},
					},
				},
			},
			"resource_tags": schema.MapAttribute{
				Description: "The resource tags associated with the product.",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create a new resource.
func (r *lagPortResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan lagPortResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	buyPortReq := &megaport.BuyPortRequest{
		Name:                  plan.Name.ValueString(),
		Term:                  int(plan.ContractTermMonths.ValueInt64()),
		PortSpeed:             int(plan.PortSpeed.ValueInt64()),
		LocationId:            int(plan.LocationID.ValueInt64()),
		LagCount:              int(plan.LagCount.ValueInt64()),
		MarketPlaceVisibility: plan.MarketplaceVisibility.ValueBool(),
		DiversityZone:         plan.DiversityZone.ValueString(),
		CostCentre:            plan.CostCentre.ValueString(),
		PromoCode:             plan.PromoCode.ValueString(),
		WaitForProvision:      true,
		WaitForTime:           waitForTime,
	}

	if !plan.ResourceTags.IsNull() {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		buyPortReq.ResourceTags = tagMap
	}

	err := r.client.PortService.ValidatePortOrder(ctx, buyPortReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Validation error while attempting to create port",
			"Validation error while attempting to create port with name "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	createdPort, err := r.client.PortService.BuyPort(ctx, buyPortReq)
	if err != nil && createdPort == nil {
		resp.Diagnostics.AddError(
			"Error Creating Port",
			"Could not create port with name "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	if len(createdPort.TechnicalServiceUIDs) < 1 {
		resp.Diagnostics.AddError(
			"Unexpected number of ports created",
			fmt.Sprintf("Expected greater than one port, got: %d. The IDs were: %v Please report this issue to Megaport.", len(createdPort.TechnicalServiceUIDs), createdPort.TechnicalServiceUIDs),
		)
		return
	}

	createdID := createdPort.TechnicalServiceUIDs[0]
	if !saveCreatedUID(ctx, resp, "LAG port", plan.Name.ValueString(), createdID, err) {
		return
	}

	// get the created port
	port, err := r.client.PortService.GetPort(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading newly created port",
			"Could not read newly created port with ID "+createdID+": "+err.Error(),
		)
		return
	}

	tags, err := r.client.PortService.ListPortResourceTags(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading newly created port tags",
			"Could not read newly created port tags with ID "+createdID+": "+err.Error(),
		)
		return
	}

	// update the plan with the port info
	apiDiags := plan.fromAPIPort(ctx, port, tags)
	resp.Diagnostics.Append(apiDiags...)
	lagPortUids := []types.String{}
	for _, uid := range createdPort.TechnicalServiceUIDs {
		lagPortUids = append(lagPortUids, types.StringValue(uid))
	}
	lagPortUidList, listDiags := types.ListValueFrom(ctx, types.StringType, lagPortUids)
	resp.Diagnostics.Append(listDiags...)
	plan.LagPortUIDs = lagPortUidList
	plan.UID = types.StringValue(createdID)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *lagPortResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state lagPortResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed port value from API
	port, err := r.client.PortService.GetPort(ctx, state.UID.ValueString())
	if err != nil {
		// Port has been deleted or is not found
		if mpErr, ok := err.(*megaport.ErrorResponse); ok {
			if mpErr.Response.StatusCode == http.StatusNotFound ||
				(mpErr.Response.StatusCode == http.StatusBadRequest && strings.Contains(mpErr.Message, "Could not find a service with UID")) {
				resp.State.RemoveResource(ctx)
				return
			}
		}

		resp.Diagnostics.AddError(
			"Error Reading port",
			"Could not read port with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	// If the port has been deleted
	if port.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED {
		resp.State.RemoveResource(ctx)
		return
	}

	tags, err := r.client.PortService.ListPortResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading port tags",
			"Could not read port tags with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Populate the state with the port details
	apiDiags := state.fromAPIPort(ctx, port, tags)
	resp.Diagnostics.Append(apiDiags...)

	// Populate the LAG port UIDs
	if len(port.LagPortUIDs) > 0 {
		lagPortUIDsList := []attr.Value{}
		for _, uid := range port.LagPortUIDs {
			lagPortUIDsList = append(lagPortUIDsList, types.StringValue(uid))
		}

		lagPortUIDs, diags := types.ListValue(types.StringType, lagPortUIDsList)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		state.LagPortUIDs = lagPortUIDs
	} else {
		state.LagPortUIDs = types.ListNull(types.StringType)
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *lagPortResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state lagPortResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Grow the LAG before anything else, so a rejected order leaves it untouched. A
	// decrease never arrives here, because ModifyPlan turns it into a replacement.
	var lagPortUIDs []string
	if plannedCount, currentCount := int(plan.LagCount.ValueInt64()), lagMemberCount(&state); plannedCount > currentCount {
		uids, addDiags := r.addLagPorts(ctx, &plan, plannedCount, currentCount)
		resp.Diagnostics.Append(addDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		lagPortUIDs = uids
	}

	// Check on changes
	var name, costCentre string
	var marketplaceVisibility bool
	if !plan.Name.Equal(state.Name) {
		name = plan.Name.ValueString()
	} else {
		name = state.Name.ValueString()
	}
	// Always use the planned cost centre value, even if it's empty/null
	costCentre = plan.CostCentre.ValueString()
	if !plan.MarketplaceVisibility.Equal(state.MarketplaceVisibility) {
		marketplaceVisibility = plan.MarketplaceVisibility.ValueBool()
	} else {
		marketplaceVisibility = state.MarketplaceVisibility.ValueBool()
	}

	contractTermMonths := int(plan.ContractTermMonths.ValueInt64())

	// The API modifies only the port named in the call, so each member gets its own.
	if !plan.Name.Equal(state.Name) || !plan.CostCentre.Equal(state.CostCentre) ||
		!plan.MarketplaceVisibility.Equal(state.MarketplaceVisibility) || !plan.ContractTermMonths.Equal(state.ContractTermMonths) {
		members, err := r.lagMembers(ctx, plan.UID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading LAG ports",
				"Could not read the ports in LAG "+plan.UID.ValueString()+": "+err.Error()+lagGrowNote(lagPortUIDs),
			)
			return
		}
		if missing := missingLagMember(members, &state); missing != "" {
			resp.Diagnostics.AddError(
				"LAG port missing from the product list",
				"The product list read does not hold port "+missing+" in LAG "+plan.UID.ValueString()+
					". This points to an incomplete read, so this apply modified no port. Run it again."+lagGrowNote(lagPortUIDs),
			)
			return
		}

		planTermChanged := !plan.ContractTermMonths.Equal(state.ContractTermMonths)
		for _, member := range members {
			// A cancelled port never reads back ready, so the modify would wait out wait_time.
			if member.ProvisioningStatus == megaport.STATUS_CANCELLED || member.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED {
				continue
			}
			// Send the term only when the plan changes it. Sending a port the term it already has
			// extends its contract, or fails on month-to-month.
			termChanged := planTermChanged && member.ContractTermMonths != contractTermMonths
			// Skip ports that already match: ports the grow just ordered, or ports an earlier failed apply reached.
			if !termChanged && member.Name == name && member.CostCentre == costCentre &&
				member.MarketplaceVisibility == marketplaceVisibility {
				continue
			}

			// No wait: the API applies these fields before it responds, and the port's status never changes.
			modifyReq := &megaport.ModifyPortRequest{
				PortID:                member.UID,
				Name:                  name,
				MarketplaceVisibility: &marketplaceVisibility,
				CostCentre:            costCentre,
			}
			if termChanged {
				modifyReq.ContractTermMonths = &contractTermMonths
			}

			if _, err := r.client.PortService.ModifyPort(ctx, modifyReq); err != nil {
				resp.Diagnostics.AddError(
					"Error modifying port",
					"The modify of port "+member.UID+" in LAG "+plan.UID.ValueString()+" failed: "+err.Error()+
						". Run the apply again to modify the remaining ports."+lagGrowNote(lagPortUIDs),
				)
				return
			}
		}
	}

	port, portErr := r.client.PortService.GetPort(ctx, plan.UID.ValueString())
	if portErr != nil {
		resp.Diagnostics.AddError(
			"Error Reading port",
			"Could not read port with ID "+plan.UID.ValueString()+": "+portErr.Error()+lagGrowNote(lagPortUIDs),
		)
		return
	}

	if !plan.ResourceTags.Equal(state.ResourceTags) {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := r.client.PortService.UpdatePortResourceTags(ctx, plan.UID.ValueString(), tagMap)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating port tags",
				"Could not update port tags with ID "+plan.UID.ValueString()+": "+err.Error()+lagGrowNote(lagPortUIDs),
			)
			return
		}
	}

	tags, tagErr := r.client.PortService.ListPortResourceTags(ctx, plan.UID.ValueString())
	if tagErr != nil {
		resp.Diagnostics.AddError(
			"Error reading port tags",
			"Could not read port tags with ID "+plan.UID.ValueString()+": "+tagErr.Error()+lagGrowNote(lagPortUIDs),
		)
		return
	}

	// Update the state
	resp.Diagnostics.Append(state.fromAPIPort(ctx, port, tags)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	state.PromoCode = plan.PromoCode

	if len(lagPortUIDs) > 0 {
		uidList, listDiags := types.ListValueFrom(ctx, types.StringType, lagPortUIDs)
		resp.Diagnostics.Append(listDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.LagPortUIDs = uidList
	}

	// Set state to fully populated data
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *lagPortResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state lagPortResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order. LAG ports only support immediate cancellation
	// (CANCEL_NOW); delayed cancellation was removed in megaportgo and the
	// API now rejects DeleteNow=false for LAG ports.
	err := retryTransientDelete(ctx, 3, func() error {
		_, deleteErr := r.client.PortService.DeletePort(ctx, &megaport.DeletePortRequest{
			PortID:     state.UID.ValueString(),
			DeleteNow:  true,
			SafeDelete: true,
		})
		return deleteErr
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting port",
			"Could not delete port, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *lagPortResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = data.client
}

func (r *lagPortResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}

func (r *lagPortResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var plan, state lagPortResourceModel

	// Get plan and state
	if !req.Plan.Raw.IsNull() {
		planDiags := req.Plan.Get(ctx, &plan)
		resp.Diagnostics.Append(planDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !req.State.Raw.IsNull() {
		stateDiags := req.State.Get(ctx, &state)
		resp.Diagnostics.Append(stateDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Only check if we have both state and plan
	if !state.UID.IsNull() && !plan.LagCount.IsNull() {
		// An unknown count decides nothing about a replacement, and the framework calls
		// ModifyPlan again on the apply walk once the value resolves. Mark the UID list
		// unknown in case the count lands above the current one and Update grows the LAG.
		if plan.LagCount.IsUnknown() {
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("lag_port_uids"), types.ListUnknown(types.StringType))...)
			return
		}

		plannedLagCount := int(plan.LagCount.ValueInt64())
		currentLagCount := lagMemberCount(&state)

		switch {
		case plannedLagCount < currentLagCount:
			// The API has no call to remove a LAG member, so shrinking replaces the LAG.
			resp.RequiresReplace = append(resp.RequiresReplace, path.Root("lag_count"))
		case plannedLagCount > currentLagCount:
			// Update orders the extra ports, so the stored UID list is no longer the answer.
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("lag_port_uids"), types.ListUnknown(types.StringType))...)
		}
	}
}

// lagMemberCount reports the ports state holds for the LAG. The UID list stays null
// until the API reports an aggregation, so lag_count answers for it.
func lagMemberCount(state *lagPortResourceModel) int {
	if state.LagPortUIDs.IsNull() {
		return int(state.LagCount.ValueInt64())
	}
	return len(state.LagPortUIDs.Elements())
}

// lagMembers returns every port in the LAG. The product list read supplies the members,
// because it carries the values each one holds.
func (r *lagPortResource) lagMembers(ctx context.Context, uid string) ([]*megaport.Port, error) {
	ports, err := r.client.PortService.ListPorts(ctx)
	if err != nil {
		return nil, err
	}

	var primary *megaport.Port
	for _, p := range ports {
		if p.UID == uid {
			primary = p
		}
	}
	if primary == nil {
		return nil, fmt.Errorf("port %s is not in the product list", uid)
	}

	members := []*megaport.Port{}
	if primary.AggregationID != 0 {
		for _, p := range ports {
			if p.AggregationID == primary.AggregationID && p.UID != uid {
				members = append(members, p)
			}
		}
	}
	// The primary goes last. Read takes its values from the primary, so a failed apply still shows a diff.
	return append(members, primary), nil
}

// missingLagMember returns a port state holds that the member list lacks, or "" when none is missing.
func missingLagMember(members []*megaport.Port, state *lagPortResourceModel) string {
	listed := map[string]bool{}
	for _, m := range members {
		listed[m.UID] = true
	}
	for _, v := range state.LagPortUIDs.Elements() {
		if uid, ok := v.(types.String); ok && !listed[uid.ValueString()] {
			return uid.ValueString()
		}
	}
	return ""
}

// addLagPorts brings the planned LAG up to target ports and returns the members it ends with.
// The API builds each new port from this request rather than from the primary, so the LAG's own
// location and speed have to be sent.
func (r *lagPortResource) addLagPorts(ctx context.Context, plan *lagPortResourceModel, target, current int) ([]string, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	primary, err := r.client.PortService.GetPort(ctx, plan.UID.ValueString())
	if err != nil {
		diags.AddError(
			"Error reading LAG port",
			"Could not read LAG port with ID "+plan.UID.ValueString()+": "+err.Error(),
		)
		return nil, diags
	}

	if primary.AggregationID == 0 {
		diags.AddError(
			"Port is not part of a LAG",
			"Port "+plan.UID.ValueString()+" reports no aggregation ID, so ports cannot be added to it. Please report this issue to Megaport.",
		)
		return nil, diags
	}

	// A product list read that drops an entry it cannot parse looks like a shrunken LAG.
	// Ordering against that count overshoots, and the next plan then proposes a replace.
	if primary.LagCount < current {
		diags.AddError(
			"LAG member count went backwards",
			fmt.Sprintf("LAG %s reports %d member ports, and state holds %d. A count that drops points to an incomplete read, so this apply ordered nothing. Run it again.",
				plan.UID.ValueString(), primary.LagCount, current),
		)
		return nil, diags
	}

	// Count against the live read, not against state, to count ports an earlier apply ordered.
	// A new port gets its interface in the order call, so the read counts it at once.
	count := target - primary.LagCount
	if count < 1 {
		return primary.LagPortUIDs, diags
	}

	buyPortReq := &megaport.BuyPortRequest{
		Name:                  plan.Name.ValueString(),
		Term:                  int(plan.ContractTermMonths.ValueInt64()),
		PortSpeed:             int(plan.PortSpeed.ValueInt64()),
		LocationId:            int(plan.LocationID.ValueInt64()),
		LagCount:              count,
		AggregationID:         primary.AggregationID,
		MarketPlaceVisibility: plan.MarketplaceVisibility.ValueBool(),
		DiversityZone:         plan.DiversityZone.ValueString(),
		CostCentre:            plan.CostCentre.ValueString(),
		PromoCode:             plan.PromoCode.ValueString(),
		WaitForProvision:      true,
		WaitForTime:           waitForTime,
	}

	if !plan.ResourceTags.IsNull() {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		diags.Append(tagDiags...)
		if diags.HasError() {
			return nil, diags
		}
		buyPortReq.ResourceTags = tagMap
	}

	if err := r.client.PortService.ValidatePortOrder(ctx, buyPortReq); err != nil {
		diags.AddError(
			"Validation error while adding ports to the LAG",
			fmt.Sprintf("Validation error while adding %d ports to LAG %s: %s", count, plan.UID.ValueString(), err.Error()),
		)
		return nil, diags
	}

	order, err := r.client.PortService.BuyPort(ctx, buyPortReq)
	if err != nil {
		diags.AddError(
			"Error adding ports to the LAG",
			fmt.Sprintf("Could not add %d ports to LAG %s: %s. If the order went through, the next refresh counts the new ports, and applying again does not order more.",
				count, plan.UID.ValueString(), err.Error()),
		)
		return nil, diags
	}

	if len(order.TechnicalServiceUIDs) < 1 {
		diags.AddError(
			"Unexpected number of ports added",
			fmt.Sprintf("Expected %d new ports on LAG %s, got none. Please report this issue to Megaport.", count, plan.UID.ValueString()),
		)
		return nil, diags
	}

	return append(primary.LagPortUIDs, order.TechnicalServiceUIDs...), diags
}

// lagGrowNote names the ports a grow ordered. Update writes state after the calls that
// follow the grow, so a failure in one of them leaves the new ports live and unrecorded.
func lagGrowNote(uids []string) string {
	if len(uids) == 0 {
		return ""
	}
	return fmt.Sprintf(" The grow step already ordered ports. The LAG now holds %d: %s. State does not have them yet, so run the apply again.",
		len(uids), strings.Join(uids, ", "))
}
