package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &mcrResource{}
	_ resource.ResourceWithConfigure   = &mcrResource{}
	_ resource.ResourceWithImportState = &mcrResource{}
)

// mcrResourceModel maps the resource schema data.
type mcrResourceModel struct {
	LastUpdated types.String `tfsdk:"last_updated"`

	ID                    types.Int64             		`tfsdk:"product_id"`
	UID                   types.String            		`tfsdk:"product_uid"`
	Name                  types.String            		`tfsdk:"product_name"`
	Type                  types.String            		`tfsdk:"product_type"`
	ProvisioningStatus    types.String            		`tfsdk:"provisioning_status"`
	CreateDate            types.String            		`tfsdk:"create_date"`
	CreatedBy             types.String            		`tfsdk:"created_by"`
	CostCentre            types.String            		`tfsdk:"cost_centre"`
	PortSpeed             types.Int64             		`tfsdk:"port_speed"`
	TerminateDate         types.String            		`tfsdk:"terminate_date"`
	LiveDate              types.String            		`tfsdk:"live_date"`
	Market                types.String            		`tfsdk:"market"`
	LocationID            types.Int64             		`tfsdk:"location_id"`
	UsageAlgorithm        types.String            		`tfsdk:"usage_algorithm"`
	MarketplaceVisibility types.Bool              		`tfsdk:"marketplace_visibility"`
	VXCPermitted          types.Bool              		`tfsdk:"vxcpermitted"`
	VXCAutoApproval       types.Bool              		`tfsdk:"vxc_auto_approval"`
	SecondaryName         types.String            		`tfsdk:"secondary_name"`
	LAGPrimary            types.Bool              		`tfsdk:"lag_primary"`
	LAGID                 types.Int64             		`tfsdk:"lag_id"`
	AggregationID         types.Int64             		`tfsdk:"aggregation_id"`
	CompanyUID            types.String            		`tfsdk:"company_uid"`
	CompanyName           types.String            		`tfsdk:"company_name"`
	ContractStartDate     types.String            		`tfsdk:"contract_start_date"`
	ContractEndDate       types.String            		`tfsdk:"contract_end_date"`
	ContractTermMonths    types.Int64             		`tfsdk:"contract_term_months"`
	ASN				   	  types.Int64             		`tfsdk:"asn"`
	
	Virtual               types.Bool              		`tfsdk:"virtual"`
	BuyoutPort            types.Bool              		`tfsdk:"buyout_port"`
	Locked                types.Bool              		`tfsdk:"locked"`
	AdminLocked           types.Bool              		`tfsdk:"admin_locked"`
	Cancelable            types.Bool              	    `tfsdk:"cancelable"`
	AttributeTags 		  map[types.String]types.String `tfsdk:"attribute_tags"`
	Resources          	  *mcrResourcesModel      		`tfsdk:"resources"`
}

// mcrResourcesModel represents the associated interface and virtual router
type mcrResourcesModel struct {
	Interface 	  *portInterfaceModel 		`tfsdk:"interface"`
	VirtualRouter *mcrVirtualRouterModel 	`tfsdk:"virtual_router"`
}

// mcrVirtualRouterModel represents the virtual router associated with the MCR
type mcrVirtualRouterModel struct {
	ID types.Int64			  `tfsdk:"id"`
	ASN types.Int64 		  `tfsdk:"mcr_asn"`
	Name types.String 		  `tfsdk:"name"`
	ResourceName types.String `tfsdk:"resource_name"`
	ResourceType types.String `tfsdk:"resource_type"`
	Speed types.Int64 	 	  `tfsdk:"speed"`
}

// fromAPIMCR maps the API MCR response to the resource schema.
func (orm *mcrResourceModel) fromAPIMCR(m *megaport.MCR) {
	orm.ID = types.Int64Value(int64(m.ID))
	orm.UID = types.StringValue(m.UID)
	orm.Name = types.StringValue(m.Name)
	orm.Type = types.StringValue(m.Type)
	orm.ProvisioningStatus = types.StringValue(m.ProvisioningStatus)
	orm.CreateDate = types.StringValue(m.CreateDate.String())
	orm.CreatedBy = types.StringValue(m.CreatedBy)
	orm.CostCentre = types.StringValue(m.CostCentre)
	orm.PortSpeed = types.Int64Value(int64(m.PortSpeed))
	orm.TerminateDate = types.StringValue(m.TerminateDate.String())
	orm.LiveDate = types.StringValue(m.LiveDate.String())
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
	orm.ContractStartDate = types.StringValue(m.ContractStartDate.String())
	orm.ContractEndDate = types.StringValue(m.ContractEndDate.String())
	orm.ContractTermMonths = types.Int64Value(int64(m.ContractTermMonths))
	orm.Virtual = types.BoolValue(m.Virtual)
	orm.BuyoutPort = types.BoolValue(m.BuyoutPort)
	orm.Locked = types.BoolValue(m.Locked)
	orm.AdminLocked = types.BoolValue(m.AdminLocked)
	orm.Cancelable = types.BoolValue(m.Cancelable)

	if m.AttributeTags != nil {
		orm.AttributeTags = make(map[types.String]types.String)
		for k, v := range m.AttributeTags {
			orm.AttributeTags[types.StringValue(k)] = types.StringValue(v)
		}
	}

	orm.Resources = &mcrResourcesModel{
		Interface: &portInterfaceModel{
			Demarcation: types.StringValue(m.Resources.Interface.Demarcation),
			Description: types.StringValue(m.Resources.Interface.Description),
			ID: types.Int64Value(int64(m.Resources.Interface.ID)),
			LOATemplate: types.StringValue(m.Resources.Interface.LOATemplate),
			Media: types.StringValue(m.Resources.Interface.Media),
			Name: types.StringValue(m.Resources.Interface.Name),
			PortSpeed: types.Int64Value(int64(m.Resources.Interface.PortSpeed)),
			ResourceName: types.StringValue(m.Resources.Interface.ResourceName),
			ResourceType: types.StringValue(m.Resources.Interface.ResourceType),
		},
		VirtualRouter: &mcrVirtualRouterModel{
			ID: types.Int64Value(int64(m.Resources.VirtualRouter.ID)),
			ASN: types.Int64Value(int64(m.Resources.VirtualRouter.ASN)),
			Name: types.StringValue(m.Resources.VirtualRouter.Name),
			ResourceName: types.StringValue(m.Resources.VirtualRouter.ResourceName),
			ResourceType: types.StringValue(m.Resources.VirtualRouter.ResourceType),
			Speed: types.Int64Value(int64(m.Resources.VirtualRouter.Speed)),
		},
	}
}

// NewPortResource is a helper function to simplify the provider implementation.
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
		Attributes: map[string]schema.Attribute{
			"last_updated": schema.StringAttribute{
				Description: "Last updated by the Terraform provider.",
				Computed: true,
			},
			"uid": schema.StringAttribute{
				Description: "UID identifier of the product.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_id": schema.Int64Attribute{
				Description: "Numeric ID of the product.",
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"product_name": schema.StringAttribute{
				Description: "Name of the product.",
				Required: true,
			},
			"provisioning_status": schema.StringAttribute{
				Description: "Provisioning status of the product.",
				Computed: true,
			},
			"create_date": schema.StringAttribute{
				Description: "Date the product was created.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				Description: "User who created the product.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"port_speed": schema.Int64Attribute{
				Description: "Bandwidth speed of the product.",
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"terminate_date": schema.StringAttribute{
				Description: "Date the product will be terminated.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"live_date": schema.StringAttribute{
				Description: "Date the product went live.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"market": schema.StringAttribute{
				Description: "Market the product is in.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"location_id": schema.Int64Attribute{
				Description: "Location ID of the product.",
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"contract_term_months": schema.Int64Attribute{
				Description: "Contract term in months.",
				Required: true,
				Validators: []validator.Int64{
					int64validator.OneOf(1, 12, 24, 36),
				},
			},
			"usage_algorithm": schema.StringAttribute{
				Description: "Usage algorithm of the product.",
				Computed: true,
			},
			"company_uid": schema.StringAttribute{
				Description: "Megaport Company UID of the product.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cost_centre": schema.StringAttribute{
				Description: "Cost centre of the product.",
				Required: true,
			},
			"contract_start_date": schema.BoolAttribute{
				Description: "Contract start date of the product.",
				Computed: true,
			},
			"contract_end_date": schema.BoolAttribute{
				Description: "Contract end date of the product.",
				Computed: true,
			},
			"marketplace_visibility": schema.BoolAttribute{
				Description: "Whether the product is visible in the Marketplace.",
				Required: true,
			},
			"asn": schema.Int64Attribute{
				Description: "ASN in the MCR order configuration.",
				Computed: true,
			},
			"vxc_permitted": schema.BoolAttribute{
				Description: "Whether VXC is permitted.",
				Computed: true,
			},
			"vxc_auto_approval": schema.BoolAttribute{
				Description: "Whether VXC is auto approved.",
				Computed: true,
			},
			"virtual": schema.BoolAttribute{
				Description: "Whether the product is virtual.",
				Computed: true,
			},
			"buyout_port": schema.BoolAttribute{
				Description: "Whether the product is bought out.",
				Optional: true,
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the product is locked.",
				Optional: true,
			},
			"admin_locked": schema.BoolAttribute{
				Description: "Whether the product is admin locked.",
				Optional: true,
			},
			"cancelable": schema.BoolAttribute{
				Description: "Whether the product is cancelable.",
				Computed: true,
			},
			"attribute_tags": schema.MapAttribute{
				ElementType: types.StringType,
				Description: "Attribute tags of the product.",
				Computed: true,
			},
			"resources": schema.SingleNestedAttribute{
				Computed: true,
				Description: "Resources associated with the product.",
				Attributes: map[string]schema.Attribute{
					"interface": schema.SingleNestedAttribute{
						Computed: true,
						Description: "Port interface associated with the product.",
						Attributes: map[string]schema.Attribute{
							"demarcation": schema.StringAttribute{
								Description: "Demarcation of the interface.",
								Computed: true,
							},
							"description": schema.StringAttribute{
								Description: "Description of the interface.",
								Computed: true,
							},
							"id": schema.Int64Attribute{
								Description: "Numeric ID of the interface.",
								Computed: true,
							},
							"loa_template": schema.StringAttribute{
								Description: "LOA template of the interface.",
								Computed: true,
							},
							"media": schema.StringAttribute{
								Description: "Media of the interface.",
								Computed: true,
							},
							"name": schema.StringAttribute{
								Description: "Name of the interface.",
								Computed: true,
							},
							"port_speed": schema.Int64Attribute{
								Description: "Bandwidth speed of the interface.",
								Computed: true,
							},
							"resource_name": schema.StringAttribute{
								Description: "Resource name of the interface.",
								Computed: true,
							},
							"resource_type": schema.StringAttribute{
								Description: "Resource type of the interface.",
								Computed: true,
							},
							"up": schema.BoolAttribute{
								Description: "Whether the interface is up.",
								Computed: true,
							},
						},
					},
					"virtual_router": schema.SingleNestedAttribute{
						Computed: true,
						Description: "Virtual router associated with the product.",
						Attributes: map[string]schema.Attribute{
							"id": schema.Int64Attribute{
								Description: "Numeric ID of the virtual router.",
								Computed: true,
							},
							"asn": schema.Int64Attribute{
								Description: "ASN of the virtual router.",
								Computed: true,
							},
							"name": schema.StringAttribute{
								Description: "Name of the virtual router.",
								Computed: true,
							},
							"resource_name": schema.StringAttribute{
								Description: "Resource name of the virtual router.",
								Computed: true,
							},
							"resource_type": schema.StringAttribute{
								Description: "Resource type of the virtual router.",
								Computed: true,
							},
							"speed": schema.Int64Attribute{
								Description: "Speed of the virtual router.",
								Computed: true,
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
		Name:                  plan.Name.ValueString(),
		Term:                  int(plan.ContractTermMonths.ValueInt64()),
		PortSpeed:             int(plan.PortSpeed.ValueInt64()),
		LocationID:            int(plan.LocationID.ValueInt64()),
		WaitForProvision:      true,
		WaitForTime:           5 * time.Minute,
	}

	if !plan.ASN.IsNull() {
		buyReq.MCRAsn = int(plan.ASN.ValueInt64())
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
	plan.fromAPIMCR(mcr)
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
		resp.Diagnostics.AddError(
			"Error Reading MCR",
			"Could not read MCR with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.fromAPIMCR(mcr)

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
	if !plan.Name.Equal(state.Name) {
		name = plan.Name.ValueString()
	}
	if !plan.CostCentre.Equal(state.CostCentre) {
		costCentre = plan.Name.ValueString()
	}

	_, err := r.client.MCRService.ModifyMCR(ctx, &megaport.ModifyMCRRequest{
		MCRID:                plan.UID.ValueString(),
		Name:                  name,
		MarketplaceVisibility: plan.MarketplaceVisibility.ValueBool(),
		CostCentre:            costCentre,
		WaitForUpdate:         true,
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

	state.fromAPIMCR(mcr)

	// Update the state with the new values
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags := resp.State.Set(ctx, plan)
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
		MCRID:    state.UID.String(),
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
	resource.ImportStatePassthroughID(ctx, path.Root("uid"), req, resp)
}