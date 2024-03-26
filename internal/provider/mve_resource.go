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
	_ resource.Resource                = &mveResource{}
	_ resource.ResourceWithConfigure   = &mveResource{}
	_ resource.ResourceWithImportState = &mveResource{}
)

// mveResourceModel maps the resource schema data.
type mveResourceModel struct {
	LastUpdated types.String `tfsdk:"last_updated"`

	ID                    types.Int64                    `tfsdk:"product_id"`
	UID                   types.String                 `tfsdk:"product_uid"`
	Name                  types.String                 `tfsdk:"product_name"`
	Type                  types.String                 `tfsdk:"product_type"`
	ProvisioningStatus    types.String                 `tfsdk:"provisioning_status"`
	CreateDate            types.String                  `tfsdk:"create_date"`
	CreatedBy             types.String                 `tfsdk:"created_by"`
	TerminateDate         types.String                  `tfsdk:"terminate_date"`
	LiveDate              types.Int64                    `tfsdk:"live_date"`
	Market                types.String                 `tfsdk:"market"`
	LocationID            types.Int64                    `tfsdk:"location_id"`
	UsageAlgorithm        types.String                 `tfsdk:"usage_algorithm"`
	MarketplaceVisibility types.Bool                   `tfsdk:"marketplace_visibility"`
	VXCPermitted          types.Bool                   `tfsdk:"vxcpermitted"`
	VXCAutoApproval       types.Bool                   `tfsdk:"vxc_auto_approval"`
	SecondaryName         types.String                 `tfsdk:"secondary_name"`
	CompanyUID            types.String                 `tfsdk:"company_uid"`
	CompanyName           types.String                 `tfsdk:"company_name"`
	ContractStartDate     types.String                  `tfsdk:"contract_start_date"`
	ContractEndDate       types.String                  `tfsdk:"contract_end_date"`
	ContractTermMonths    types.Int64                    `tfsdk:"contract_term_months"`
	
	Virtual               types.Bool                   `tfsdk:"virtual"`
	BuyoutPort            types.Bool                   `tfsdk:"buyout_port"`
	Locked                types.Bool                   `tfsdk:"locked"`
	AdminLocked           types.Bool                   `tfsdk:"admin_locked"`
	Cancelable            types.Bool                   `tfsdk:"cancelable"`
	
	Vendor                types.String                 `tfsdk:"vendor"`
	Size                  types.String                 `tfsdk:"mve_size"`
	
	// TODO - MODELS FOR RESOURCES AND VENDOR CONFIGS
	// NetworkInterfaces     []*MVENetworkInterface `tfsdk:"vnics"`
	// AttributeTags         map[string]string      `tfsdk:"attribute_tags"`
	// Resources             *MVEResources          `tfsdk:"resources"`
}

func (orm *mveResourceModel) fromAPIMVE(p *megaport.MVE) {
	orm.ID = types.Int64Value(int64(p.ID))
	orm.UID = types.StringValue(p.UID)
	orm.Name = types.StringValue(p.Name)
	orm.Type = types.StringValue(p.Type)
	orm.ProvisioningStatus = types.StringValue(p.ProvisioningStatus)
	orm.CreateDate = types.StringValue(p.CreateDate.String())
	orm.CreatedBy = types.StringValue(p.CreatedBy)
	orm.TerminateDate = types.StringValue(p.TerminateDate.String())
	orm.LiveDate = types.Int64Value(int64(p.LiveDate))
	orm.Market = types.StringValue(p.Market)
	orm.LocationID = types.Int64Value(int64(p.LocationID))
	orm.UsageAlgorithm = types.StringValue(p.UsageAlgorithm)
	orm.MarketplaceVisibility = types.BoolValue(p.MarketplaceVisibility)
	orm.VXCPermitted = types.BoolValue(p.VXCPermitted)
	orm.VXCAutoApproval = types.BoolValue(p.VXCAutoApproval)
	orm.SecondaryName = types.StringValue(p.SecondaryName)
	orm.CompanyUID = types.StringValue(p.CompanyUID)
	orm.CompanyName = types.StringValue(p.CompanyName)
	orm.ContractStartDate = types.StringValue(p.ContractStartDate.String())
	orm.ContractEndDate = types.StringValue(p.ContractEndDate.String())
	orm.ContractTermMonths = types.Int64Value(int64(p.ContractTermMonths))
	orm.Virtual = types.BoolValue(p.Virtual)
	orm.BuyoutPort = types.BoolValue(p.BuyoutPort)
	orm.Locked = types.BoolValue(p.Locked)
	orm.AdminLocked = types.BoolValue(p.AdminLocked)
	orm.Cancelable = types.BoolValue(p.Cancelable)
	orm.Vendor = types.StringValue(p.Vendor)
	orm.Size = types.StringValue(p.Size)
}

// NewPortResource is a helper function to simplify the provider implementation.
func NewMVEResource() resource.Resource {
	return &mveResource{}
}

// mveResource is the resource implementation.
type mveResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *mveResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mve"
}

// Schema defines the schema for the resource.
func (r *mveResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
		"last_updated": schema.StringAttribute{
				Description: "The last time the MVE was updated by the Terraform Provider.",
				Computed: true,
		},
		"uid": schema.StringAttribute{
			Description: "The unique identifier of the MVE.",
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"product_id": schema.Int64Attribute{
			Description: "The Numeric ID of the MVE.",
			Computed: true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"product_name": schema.StringAttribute{
			Description: "The name of the MVE.",
			Required: true,
		},
		"provisioning_status": schema.StringAttribute{
			Description: "The provisioning status of the MVE.",
			Computed: true,
		},
		"create_date": schema.StringAttribute{
			Description: "The date the MVE was created.",
			Computed: true,
			PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
			},
		},
		"created_by": schema.StringAttribute{
			Description: "The user who created the MVE.",
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"terminate_date": schema.StringAttribute{
			Description: "The date the MVE will be terminated.",
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"live_date": schema.StringAttribute{
			Description: "The date the MVE went live.",
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"market": schema.StringAttribute{
			Description: "The market the MVE is in.",
			Required: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"location_id": schema.Int64Attribute{
			Description: "The location ID of the MVE.",
			Required: true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplace(),
			},
		},
		"contract_term_months": schema.Int64Attribute{
			Description: "The contract term in months.",
			Required: true,
			Validators: []validator.Int64{
				int64validator.OneOf(1, 12, 24, 36),
			},
		},
		"usage_algorithm": schema.StringAttribute{
			Description: "The usage algorithm of the MVE.",
			Computed: true,
		},
		"company_uid": schema.StringAttribute{
			Description: "The company UID of the MVE.",
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"contract_start_date": schema.BoolAttribute{
			Description: "The contract start date of the MVE.",
			Computed: true,
		},
		"contract_end_date": schema.BoolAttribute{
			Description: "The contract end date of the MVE.",
			Computed: true,
		},
		"marketplace_visibility": schema.BoolAttribute{
			Description: "Whether the MVE is visible in the marketplace.",
			Required: true,
		},
		"vxc_permitted": schema.BoolAttribute{
			Description: "Whether VXC is permitted.",
			Computed: true,
		},
		"vxc_auto_approval": schema.BoolAttribute{
			Description: "Whether VXC is auto approved.",
			Computed: true,
		},
		"secondary_name": schema.StringAttribute{
			Description: "The secondary name of the MVE.",
			Computed: true,
		},
		"company_name": schema.StringAttribute{
			Description: "The company name of the MVE.",
			Computed: true,
		},
		"virtual": schema.BoolAttribute{
			Description: "Whether the MVE is virtual.",
			Computed: true,
		},
		"buyout_port": schema.BoolAttribute{
			Description: "Whether the port is buyout.",
			Computed: true,
		},
		"locked": schema.BoolAttribute{
			Description: "Whether the MVE is locked.",
			Computed: true,
		},
		"admin_locked": schema.BoolAttribute{
			Description: "Whether the MVE is admin locked.",
			Computed: true,
		},
		"cancelable": schema.BoolAttribute{
			Description: "Whether the MVE is cancelable.",
			Computed: true,
		},
		"vendor": schema.StringAttribute{
			Description: "The vendor of the MVE.",
			Computed: true,
		},
		"mve_size": schema.StringAttribute{
			Description: "The size of the MVE.",
			Computed: true,
		},
	},
	}
}

// Create a new resource.
func (r *mveResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan mveResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createdMVE, err := r.client.MVEService.BuyMVE(ctx, &megaport.BuyMVERequest{
		LocationID: 		 int(plan.LocationID.ValueInt64()),
		Name:                 plan.Name.ValueString(),
		Term: 			   int(plan.ContractTermMonths.ValueInt64()),

		// TODO - VNICS
		// TODO - Vendor Config

		WaitForProvision:      true,
		WaitForTime:           5 * time.Minute,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading MVE",
			"Could not create MVE with name "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}


	createdID := createdMVE.TechnicalServiceUID

	// get the created MVE
	mve, err := r.client.MVEService.GetMVE(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading newly created MVE",
			"Could not read newly created MVE with ID "+createdID+": "+err.Error(),
		)
		return
	}

	// update the plan with the MVE info
	plan.fromAPIMVE(mve)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *mveResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state mveResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed MVE value from API
	mve, err := r.client.MVEService.GetMVE(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading MVE",
			"Could not read MVE with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.fromAPIMVE(mve)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *mveResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state mveResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check on changes
	var name types.String
	if !plan.Name.Equal(state.Name) {
		name = plan.Name
	}

	r.client.MVEService.ModifyMVE(ctx, &megaport.ModifyMVERequest{
		MVEID: state.UID.ValueString(),
		Name: name.String(),
		WaitForUpdate:         true,
	})

	// // Generate API request body from plan
	// var hashicupsItems []hashicups.OrderItem
	// for _, item := range plan.Items {
	// 	hashicupsItems = append(hashicupsItems, hashicups.OrderItem{
	// 		Coffee: hashicups.Coffee{
	// 			ID: types.Int64(item.Coffee.ID.ValueInt64()),
	// 		},
	// 		Quantity: types.Int64(item.Quantity.ValueInt64()),
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
func (r *mveResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state mveResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	_, err := r.client.MVEService.DeleteMVE(ctx, &megaport.DeleteMVERequest{
		MVEID: state.UID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting MVE",
			"Could not delete MVE, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *mveResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *mveResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("uid"), req, resp)
}
