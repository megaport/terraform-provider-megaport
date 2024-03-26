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
	_ resource.Resource                = &portResource{}
	_ resource.ResourceWithConfigure   = &portResource{}
	_ resource.ResourceWithImportState = &portResource{}
)

// singlePortResourceModel maps the resource schema data.
type singlePortResourceModel struct {
	LastUpdated types.String `tfsdk:"last_updated"`

	UID                   types.String `tfsdk:"uid"`
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

	// AttributeTags         PortAttributeTags `tfsdk:"attribute_tags"`
	// VXCResources          PortResources     `tfsdk:"resources"`
}

func (orm *singlePortResourceModel) fromAPIPort(p *megaport.Port) {
	orm.UID = types.StringValue(p.UID)
	orm.ID = types.Int64Value(int64(p.ID))
	orm.Cancelable = types.BoolValue(p.Cancelable)
	orm.CompanyUID = types.StringValue(p.CompanyUID)
	orm.ContractEndDate = types.StringValue(p.ContractEndDate.Format(time.RFC850))
	orm.ContractStartDate = types.StringValue(p.ContractStartDate.Format(time.RFC850))
	orm.ContractTermMonths = types.Int64Value(int64(p.ContractTermMonths))
	orm.CostCentre = types.StringValue(p.CostCentre)
	orm.CreateDate = types.StringValue(p.CreateDate.Format(time.RFC850))
	orm.CreatedBy = types.StringValue(p.CreatedBy)
	orm.DiversityZone = types.StringValue(p.DiversityZone)
	orm.LiveDate = types.StringValue(p.LiveDate.Format(time.RFC850))
	orm.LocationID = types.Int64Value(int64(p.LocationID))
	orm.Locked = types.BoolValue(p.Locked)
	orm.Market = types.StringValue(p.Market)
	orm.MarketplaceVisibility = types.BoolValue(p.MarketplaceVisibility)
	orm.Name = types.StringValue(p.Name)
	orm.PortSpeed = types.Int64Value(int64(p.PortSpeed))
	orm.ProvisioningStatus = types.StringValue(p.ProvisioningStatus)
	orm.TerminateDate = types.StringValue(p.TerminateDate.Format(time.RFC850))
	orm.UsageAlgorithm = types.StringValue(p.UsageAlgorithm)
	orm.VXCAutoApproval = types.BoolValue(p.VXCAutoApproval)
	orm.VXCPermitted = types.BoolValue(p.VXCPermitted)
	orm.Virtual = types.BoolValue(p.Virtual)
}

// NewPortResource is a helper function to simplify the provider implementation.
func NewPortResource() resource.Resource {
	return &portResource{}
}

// portResource is the resource implementation.
type portResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *portResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port"
}

// Schema defines the schema for the resource.
func (r *portResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"uid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_id": schema.Int64Attribute{
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"product_name": schema.StringAttribute{
				Required: true,
			},
			"provisioning_status": schema.StringAttribute{
				Computed: true,
			},
			"create_date": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"port_speed": schema.Int64Attribute{
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"terminate_date": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"live_date": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"market": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"location_id": schema.Int64Attribute{
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"contract_term_months": schema.Int64Attribute{
				Required: true,
				Validators: []validator.Int64{
					int64validator.OneOf(1, 12, 24, 36),
				},
			},
			"usage_algorithm": schema.StringAttribute{
				Computed: true,
			},
			"company_uid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cost_centre": schema.StringAttribute{
				Required: true,
			},
			"contract_start_date": schema.BoolAttribute{
				Computed: true,
			},
			"contract_end_date": schema.BoolAttribute{
				Computed: true,
			},
			"marketplace_visibility": schema.BoolAttribute{
				Required: true,
			},
			"vxc_permitted": schema.BoolAttribute{
				Computed: true,
			},
			"vxc_auto_approval": schema.BoolAttribute{
				Computed: true,
			},
			"virtual": schema.BoolAttribute{
				Computed: true,
			},
			"locked": schema.BoolAttribute{
				Optional: true,
			},
			"cancelable": schema.BoolAttribute{
				Computed: true,
			},
			"diversity_zone": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create a new resource.
func (r *portResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan singlePortResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createdPort, err := r.client.PortService.BuyPort(ctx, &megaport.BuyPortRequest{
		Name:                  plan.Name.ValueString(),
		Term:                  int(plan.ContractTermMonths.ValueInt64()),
		PortSpeed:             int(plan.PortSpeed.ValueInt64()),
		LocationId:            int(plan.LocationID.ValueInt64()),
		Market:                plan.Market.ValueString(),
		MarketPlaceVisibility: plan.MarketplaceVisibility.ValueBool(),
		DiversityZone:         plan.DiversityZone.ValueString(),
		WaitForProvision:      true,
		WaitForTime:           5 * time.Minute,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading port",
			"Could not create port with name "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	if len(createdPort.TechnicalServiceUIDs) != 1 {
		resp.Diagnostics.AddError(
			"Unexpected number of ports created",
			fmt.Sprintf("Expected 1 port, got: %d. The IDs were: %v Please report this issue to Megaport.", len(createdPort.TechnicalServiceUIDs), createdPort.TechnicalServiceUIDs),
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
	plan.fromAPIPort(port)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *portResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state singlePortResourceModel
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

	state.fromAPIPort(port)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *portResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state singlePortResourceModel

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

	r.client.PortService.ModifyPort(ctx, &megaport.ModifyPortRequest{
		PortID:                plan.UID.ValueString(),
		Name:                  name,
		MarketplaceVisibility: plan.MarketplaceVisibility.ValueBool(),
		CostCentre:            costCentre,
		WaitForUpdate:         true,
	})

	// // Generate API request body from plan
	// var hashicupsItems []hashicups.OrderItem
	// for _, item := range plan.Items {
	// 	hashicupsItems = append(hashicupsItems, hashicups.OrderItem{
	// 		Coffee: hashicups.Coffee{
	// 			ID: int(item.Coffee.ID.ValueInt64()),
	// 		},
	// 		Quantity: int(item.Quantity.ValueInt64()),
	// 	})
	// }

	// // Update existing order
	// _, err := r.client.UpdateOrder(plan.ID.ValueString(), hashicupsItems)
	// if err != nil {
	// 	resp.Diagnostics.AddError(
	// 		"Error Updating HashiCups Order",
	// 		"Could not update order, unexpected error: "+err.Error(),
	// 	)
	// 	return
	// }

	// // Fetch updated items from GetOrder as UpdateOrder items are not
	// // populated.
	// order, err := r.client.GetOrder(plan.ID.ValueString())
	// if err != nil {
	// 	resp.Diagnostics.AddError(
	// 		"Error Reading HashiCups Order",
	// 		"Could not read HashiCups order ID "+plan.ID.ValueString()+": "+err.Error(),
	// 	)
	// 	return
	// }

	// // Update resource state with updated items and timestamp
	// plan.Items = []orderItemModel{}
	// for _, item := range order.Items {
	// 	plan.Items = append(plan.Items, orderItemModel{
	// 		Coffee: orderItemCoffeeModel{
	// 			ID:          types.Int64Value(int64(item.Coffee.ID)),
	// 			Name:        types.StringValue(item.Coffee.Name),
	// 			Teaser:      types.StringValue(item.Coffee.Teaser),
	// 			Description: types.StringValue(item.Coffee.Description),
	// 			Price:       types.Float64Value(item.Coffee.Price),
	// 			Image:       types.StringValue(item.Coffee.Image),
	// 		},
	// 		Quantity: types.Int64Value(int64(item.Quantity)),
	// 	})
	// }
	// plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// diags = resp.State.Set(ctx, plan)
	// resp.Diagnostics.Append(diags...)
	// if resp.Diagnostics.HasError() {
	// 	return
	// }
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *portResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state singlePortResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	_, err := r.client.PortService.DeletePort(ctx, &megaport.DeletePortRequest{
		PortID:    state.UID.String(),
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
func (r *portResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *portResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("uid"), req, resp)
}
