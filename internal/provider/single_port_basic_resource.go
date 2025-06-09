package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &portBasicResource{}
	_ resource.ResourceWithConfigure   = &portBasicResource{}
	_ resource.ResourceWithImportState = &portBasicResource{}
)

// singlePortBasicResourceModel maps the resource schema data.
type singlePortBasicResourceModel struct {
	UID  types.String `tfsdk:"product_uid"`
	ID   types.Int64  `tfsdk:"product_id"`
	Name types.String `tfsdk:"product_name"`

	PortSpeed             types.Int64  `tfsdk:"port_speed"`
	Market                types.String `tfsdk:"market"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	UsageAlgorithm        types.String `tfsdk:"usage_algorithm"`
	MarketplaceVisibility types.Bool   `tfsdk:"marketplace_visibility"`
	VXCPermitted          types.Bool   `tfsdk:"vxc_permitted"`
	VXCAutoApproval       types.Bool   `tfsdk:"vxc_auto_approval"`
	CompanyUID            types.String `tfsdk:"company_uid"`
	CostCentre            types.String `tfsdk:"cost_centre"`
	ContractTermMonths    types.Int64  `tfsdk:"contract_term_months"`
	Locked                types.Bool   `tfsdk:"locked"`
	Cancelable            types.Bool   `tfsdk:"cancelable"`
	DiversityZone         types.String `tfsdk:"diversity_zone"`
	PromoCode             types.String `tfsdk:"promo_code"`

	ResourceTags types.Map `tfsdk:"resource_tags"`
}

func (orm *singlePortBasicResourceModel) fromAPIPort(ctx context.Context, p *megaport.Port, tags map[string]string) diag.Diagnostics {
	diags := diag.Diagnostics{}
	orm.UID = types.StringValue(p.UID)
	orm.ID = types.Int64Value(int64(p.ID))
	orm.Cancelable = types.BoolValue(p.Cancelable)
	orm.CompanyUID = types.StringValue(p.CompanyUID)
	orm.ContractTermMonths = types.Int64Value(int64(p.ContractTermMonths))
	orm.CostCentre = types.StringValue(p.CostCentre)

	orm.DiversityZone = types.StringValue(p.DiversityZone)
	orm.LocationID = types.Int64Value(int64(p.LocationID))
	orm.Locked = types.BoolValue(p.Locked)
	orm.Market = types.StringValue(p.Market)
	orm.MarketplaceVisibility = types.BoolValue(p.MarketplaceVisibility)
	orm.Name = types.StringValue(p.Name)
	orm.PortSpeed = types.Int64Value(int64(p.PortSpeed))

	orm.UsageAlgorithm = types.StringValue(p.UsageAlgorithm)
	orm.VXCAutoApproval = types.BoolValue(p.VXCAutoApproval)
	orm.VXCPermitted = types.BoolValue(p.VXCPermitted)

	if len(tags) > 0 {
		resourceTags, tagDiags := types.MapValueFrom(ctx, types.StringType, tags)
		diags = append(diags, tagDiags...)
		orm.ResourceTags = resourceTags
	} else {
		orm.ResourceTags = types.MapNull(types.StringType)
	}

	return diags
}

// NewPortBasicResource is a helper function to simplify the provider implementation.
func NewPortBasicResource() resource.Resource {
	return &portBasicResource{}
}

// portResource is the resource implementation.
type portBasicResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *portBasicResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_basic"
}

// Schema defines the schema for the resource.
func (r *portBasicResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Single Port Resource for the Megaport Terraform Provider. This can be used to create, modify, and delete Megaport Ports. Your organization’s Port is the physical point of connection between your organization’s network and the Megaport network. You will need to deploy a Port wherever you want to direct traffic.",
		Attributes: map[string]schema.Attribute{
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
				Description: "The name of the product. Specify a name for the Port that is easily identifiable, particularly if you plan on having more than one Port.",
				Required:    true,
			},
			"port_speed": schema.Int64Attribute{
				Description: "The speed of the port in Mbps.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.OneOf(1000, 10000, 100000),
				},
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
				Description: "The term of the contract in months: valid values are 1, 12, 24, and 36. To set the product to a month-to-month contract with no minimum term, set the value to 1.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(1, 12, 24, 36),
				},
			},
			"usage_algorithm": schema.StringAttribute{
				Description: "The usage algorithm for the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"company_uid": schema.StringAttribute{
				Description: "The unique identifier of the company.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cost_centre": schema.StringAttribute{
				Description: "A customer reference number to be included in billing information and invoices. Also known as the service level reference (SLR) number. Specify a unique identifying number for the product to be used for billing purposes, such as a cost center number or a unique customer ID. The service level reference number appears for each service under the Product section of the invoice. You can also edit this field for an existing service. Please note that a VXC associated with the Port is not automatically updated with the Port service level reference number.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"promo_code": schema.StringAttribute{
				Description: "Promo code is an optional string that can be used to enter a promotional code for the service order. The code is not validated, so if the code doesn't exist or doesn't work for the service, the request will still be successful.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"marketplace_visibility": schema.BoolAttribute{
				Description: "Whether the product is visible in the marketplace. By default, the Port is private to your enterprise and consumes services from the Megaport network for your own internal company, team, and resources. When set to Private, the Port is not searchable in the Megaport Marketplace (however, others can still connect to you using a service key). Click Public to make the new Port and profile visible on the Megaport network for inbound connection requests. It is possible to change the Port from Private to Public after the initial setup.",
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
				Description: "The diversity zone of the product.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
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
func (r *portBasicResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan singlePortBasicResourceModel
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
	if err != nil {
		resp.Diagnostics.AddError(
			"Error buying port",
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

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *portBasicResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state singlePortBasicResourceModel
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

	apiDiags := state.fromAPIPort(ctx, port, tags)
	resp.Diagnostics.Append(apiDiags...)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *portBasicResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state singlePortBasicResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check on changes
	var name, costCentre string
	var marketplaceVisibility bool
	var contractTermMonths *int
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
	if !plan.ContractTermMonths.Equal(state.ContractTermMonths) {
		months := int(plan.ContractTermMonths.ValueInt64())
		contractTermMonths = &months
	}

	_, modifyErr := r.client.PortService.ModifyPort(ctx, &megaport.ModifyPortRequest{
		PortID:                plan.UID.ValueString(),
		Name:                  name,
		MarketplaceVisibility: &marketplaceVisibility,
		ContractTermMonths:    contractTermMonths,
		CostCentre:            costCentre,
		WaitForUpdate:         true,
		WaitForTime:           waitForTime,
	})
	if modifyErr != nil {
		resp.Diagnostics.AddError(
			"Error Updating port",
			"Could not update port with ID "+plan.UID.ValueString()+": "+modifyErr.Error(),
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

	// If change in resource tags from state
	if !plan.ResourceTags.Equal(state.ResourceTags) {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := r.client.PortService.UpdatePortResourceTags(ctx, plan.UID.ValueString(), tagMap)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating port resource tags",
				"Could not update port resource tags with ID "+plan.UID.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	tags, err := r.client.PortService.ListPortResourceTags(ctx, plan.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading port tags",
			"Could not read port tags with ID "+plan.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update the state
	state.fromAPIPort(ctx, port, tags)

	// Set state to fully populated data
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *portBasicResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Retrieve values from state
	var state singlePortBasicResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	_, err := r.client.PortService.DeletePort(ctx, &megaport.DeletePortRequest{
		PortID:     state.UID.ValueString(),
		DeleteNow:  true,
		SafeDelete: true,
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
func (r *portBasicResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *portBasicResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)

	if resp.Diagnostics.HasError() {
		return
	}
}
