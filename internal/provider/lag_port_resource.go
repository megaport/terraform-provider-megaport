package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
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

	AttributeTags   types.Object `tfsdk:"attribute_tags"`
	Resources       types.Object `tfsdk:"resources"`
	LocationDetails types.Object `tfsdk:"location_details"`
}

func (orm *lagPortResourceModel) fromAPIPort(ctx context.Context, p *megaport.Port) diag.Diagnostics {
	diags := diag.Diagnostics{}

	orm.UID = types.StringValue(p.UID)
	orm.ID = types.Int64Value(int64(p.ID))
	orm.Cancelable = types.BoolValue(p.Cancelable)
	orm.CompanyUID = types.StringValue(p.CompanyUID)
	orm.ContractTermMonths = types.Int64Value(int64(p.ContractTermMonths))
	orm.CostCentre = types.StringValue(p.CostCentre)
	orm.CreateDate = types.StringValue(p.CreateDate.Format(time.RFC850))
	orm.CreatedBy = types.StringValue(p.CreatedBy)
	orm.DiversityZone = types.StringValue(p.DiversityZone)
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

	attributeTagsModel := &portAttributeTagsModel{}
	terminatedServiceDetailsModel := &portTerminatedServiceDetailsModel{
		Device: types.StringValue(p.AttributeTags.TerminatedServiceDetails.Device),
	}
	locationModel := &portTerminatedServiceDetailsLocationModel{
		ID:       types.Int64Value(int64(p.AttributeTags.TerminatedServiceDetails.Location.ID)),
		Name:     types.StringValue(p.AttributeTags.TerminatedServiceDetails.Location.Name),
		SiteCode: types.StringValue(p.AttributeTags.TerminatedServiceDetails.Location.SiteCode),
	}
	interfaceModel := &portTerminatedServiceDetailsInterfaceModel{
		ResourceType: types.StringValue(p.AttributeTags.TerminatedServiceDetails.Interface.ResourceType),
		Demarcation:  types.StringValue(p.AttributeTags.TerminatedServiceDetails.Interface.Demarcation),
		LOATemplate:  types.StringValue(p.AttributeTags.TerminatedServiceDetails.Interface.LOATemplate),
		Media:        types.StringValue(p.AttributeTags.TerminatedServiceDetails.Interface.Media),
		PortSpeed:    types.Int64Value(int64(p.AttributeTags.TerminatedServiceDetails.Interface.PortSpeed)),
		ResourceName: types.StringValue(p.AttributeTags.TerminatedServiceDetails.Interface.ResourceName),
		Up:           types.Int64Value(int64(p.AttributeTags.TerminatedServiceDetails.Interface.Up)),
		Shutdown:     types.BoolValue(p.AttributeTags.TerminatedServiceDetails.Interface.Shutdown),
	}
	locationObject, locationDiags := types.ObjectValueFrom(ctx, portTerminatedServiceDetailsLocationAttrs, locationModel)
	diags = append(diags, locationDiags...)
	interfaceObject, interfaceDiags := types.ObjectValueFrom(ctx, portTerminatedServiceDetailsInterfaceAttrs, interfaceModel)
	diags = append(diags, interfaceDiags...)
	terminatedServiceDetailsModel.Location = locationObject
	terminatedServiceDetailsModel.Interface = interfaceObject
	terminatedServiceDetailsObject, terminatedServiceDetailsDiags := types.ObjectValueFrom(ctx, portTerminatedServiceDetailsAttrs, terminatedServiceDetailsModel)
	diags = append(diags, terminatedServiceDetailsDiags...)
	attributeTagsModel.TerminatedServiceDetails = terminatedServiceDetailsObject
	attributeTagsObject, attributeTagsDiags := types.ObjectValueFrom(ctx, portAttributeTagsAttrs, attributeTagsModel)
	diags = append(diags, attributeTagsDiags...)
	orm.AttributeTags = attributeTagsObject

	resourcesModel := &portResourcesModel{}
	interfaceObj, interfaceDiags := fromAPIPortInterface(ctx, &p.VXCResources.Interface)
	diags = append(diags, interfaceDiags...)
	resourcesModel.Interface = interfaceObj
	resourcesObject, resourcesDiags := types.ObjectValueFrom(ctx, portResourcesAttrs, resourcesModel)
	diags = append(diags, resourcesDiags...)
	orm.Resources = resourcesObject

	if p.LocationDetails != nil {
		locationDetailsModel := &productLocationDetailsModel{
			Name:    types.StringValue(p.LocationDetails.Name),
			City:    types.StringValue(p.LocationDetails.City),
			Metro:   types.StringValue(p.LocationDetails.Metro),
			Country: types.StringValue(p.LocationDetails.Country),
		}
		locationDetailsObject, locationDetailsDiags := types.ObjectValueFrom(ctx, productLocationDetailsAttrs, locationDetailsModel)
		diags = append(diags, locationDetailsDiags...)
		orm.LocationDetails = locationDetailsObject
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
		Description: "LAG Port Resource for the Megaport Terraform Provider. This can be used to create, modify, and delete Megaport LAG Ports. A Port is a physical or virtual network connection that allows you to connect to the Megaport network. Ports are used to create Virtual Cross Connects (VXCs) to connect to other services on the Megaport network.",
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
				Description: "The provisioning status of the product.",
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
				Description: "The speed of the port in Mbps. Can be 10000 (10 G) or 100000 (100 G, where available).",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.OneOf(10000, 100000),
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
				Description: "The numeric location ID of the product.",
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
			"promo_code": schema.StringAttribute{
				Description: "Promo code is an optional string that can be used to enter a promotional code for the service order. The code is not validated, so if the code doesn't exist or doesn't work for the service, the request will still be successful.",
				Optional:    true,
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
			},
			"vxc_auto_approval": schema.BoolAttribute{
				Description: "Whether VXC is auto-approved on this product.",
				Computed:    true,
			},
			"virtual": schema.BoolAttribute{
				Description: "Whether the product is virtual. Always false for LAG orders.",
				Computed:    true,
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the product is locked.",
				Optional:    true,
				Computed:    true,
			},
			"cancelable": schema.BoolAttribute{
				Description: "Whether the product is cancelable.",
				Computed:    true,
			},
			"diversity_zone": schema.StringAttribute{
				Description: "The diversity zone of the product.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"lag_count": schema.Int64Attribute{
				Description: "The number of LAG ports. Valid values are between 1 and 8.",
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
			"location_details": schema.SingleNestedAttribute{
				Description: "The location details of the product.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: "The name of the location.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"city": schema.StringAttribute{
						Description: "The city of the location.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"metro": schema.StringAttribute{
						Description: "The metro of the location.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"country": schema.StringAttribute{
						Description: "The country of the location.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"attribute_tags": schema.SingleNestedAttribute{
				Description: "The attribute tags of the product.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"terminated_service_details": schema.SingleNestedAttribute{
						Description: "The terminated service details of the product.",
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"location": schema.SingleNestedAttribute{
								Description: "The location of the terminated service.",
								Optional:    true,
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"id": schema.Int64Attribute{
										Description: "The ID of the location.",
										Computed:    true,
									},
									"name": schema.StringAttribute{
										Description: "The name of the location.",
										Computed:    true,
									},
									"site_code": schema.StringAttribute{
										Description: "The site code of the location.",
										Computed:    true,
									},
								},
							},
							"interface": schema.SingleNestedAttribute{
								Description: "The interface of the terminated service.",
								Optional:    true,
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"resource_type": schema.StringAttribute{
										Description: "The resource type of the interface.",
										Optional:    true,
										Computed:    true,
									},
									"demarcation": schema.StringAttribute{
										Description: "The demarcation of the interface.",
										Optional:    true,
										Computed:    true,
									},
									"loa_template": schema.StringAttribute{
										Description: "The LOA template of the interface.",
										Optional:    true,
										Computed:    true,
									},
									"media": schema.StringAttribute{
										Description: "The media of the interface.",
										Optional:    true,
										Computed:    true,
									},
									"port_speed": schema.Int64Attribute{
										Description: "The port speed of the interface.",
										Optional:    true,
										Computed:    true,
									},
									"resource_name": schema.StringAttribute{
										Description: "The resource name of the interface.",
										Optional:    true,
										Computed:    true,
									},
									"up": schema.Int64Attribute{
										Description: "The up status of the interface.",
										Optional:    true,
										Computed:    true,
									},
									"shutdown": schema.BoolAttribute{
										Description: "The shutdown status of the interface.",
										Optional:    true,
										Computed:    true,
									},
								},
							},
							"device": schema.StringAttribute{
								Description: "The device of the terminated service.",
								Optional:    true,
								Computed:    true,
							},
						},
					},
				},
			},
			"resources": schema.SingleNestedAttribute{
				Description: "Resources attached to port.",
				Optional:    true,
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
							"description": schema.StringAttribute{
								Description: "The description of the interface.",
								Computed:    true,
							},
							"id": schema.Int64Attribute{
								Description: "The ID of the interface.",
								Computed:    true,
							},
							"loa_template": schema.StringAttribute{
								Description: "The LOA template of the interface.",
								Computed:    true,
							},
							"media": schema.StringAttribute{
								Description: "The media of the interface.",
								Computed:    true,
							},
							"name": schema.StringAttribute{
								Description: "The name of the interface.",
								Computed:    true,
							},
							"port_speed": schema.Int64Attribute{
								Description: "The port speed of the interface.",
								Computed:    true,
							},
							"resource_name": schema.StringAttribute{
								Description: "The resource name of the interface.",
								Computed:    true,
							},
							"resource_type": schema.StringAttribute{
								Description: "The resource type of the interface.",
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

	createdPort, err := r.client.PortService.BuyPort(ctx, buyPortReq)
	if err != nil {
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

	// get the created port
	port, err := r.client.PortService.GetPort(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading newly created port",
			"Could not read newly created port with ID "+createdID+": "+err.Error(),
		)
		return
	}

	// update the plan with the port info
	apiDiags := plan.fromAPIPort(ctx, port)
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
		resp.Diagnostics.AddError(
			"Error Reading port",
			"Could not read port with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	apiDiags := state.fromAPIPort(ctx, port)
	resp.Diagnostics.Append(apiDiags...)

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

	// Check on changes
	var name, costCentre string
	var marketplaceVisibility bool
	if !plan.Name.Equal(state.Name) {
		name = plan.Name.ValueString()
	} else {
		name = state.Name.ValueString()
	}
	if !plan.CostCentre.Equal(state.CostCentre) {
		costCentre = plan.CostCentre.ValueString()
	} else {
		costCentre = state.CostCentre.ValueString()
	}
	if !plan.MarketplaceVisibility.Equal(state.MarketplaceVisibility) {
		marketplaceVisibility = plan.MarketplaceVisibility.ValueBool()
	} else {
		marketplaceVisibility = state.MarketplaceVisibility.ValueBool()
	}

	_, err := r.client.PortService.ModifyPort(ctx, &megaport.ModifyPortRequest{
		PortID:                plan.UID.ValueString(),
		Name:                  name,
		MarketplaceVisibility: &marketplaceVisibility,
		CostCentre:            costCentre,
		WaitForUpdate:         true,
		WaitForTime:           waitForTime,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error modifying port",
			"Could not modify port with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	port, portErr := r.client.PortService.GetPort(ctx, plan.UID.ValueString())
	if portErr != nil {
		resp.Diagnostics.AddError(
			"Error Reading port",
			"Could not read port with ID "+plan.UID.ValueString()+": "+portErr.Error(),
		)
		return
	}

	// Update the state
	state.fromAPIPort(ctx, port)
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

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

	// Delete existing order
	_, err := r.client.PortService.DeletePort(ctx, &megaport.DeletePortRequest{
		PortID:    state.UID.ValueString(),
		DeleteNow: true,
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

func (r *lagPortResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}
