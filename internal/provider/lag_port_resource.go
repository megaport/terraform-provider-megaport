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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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
	Name                  types.String `tfsdk:"product_name"`
	PortSpeed             types.Int64  `tfsdk:"port_speed"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	MarketplaceVisibility types.Bool   `tfsdk:"marketplace_visibility"`
	CompanyUID            types.String `tfsdk:"company_uid"`
	CostCentre            types.String `tfsdk:"cost_centre"`
	ContractTermMonths    types.Int64  `tfsdk:"contract_term_months"`
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
	orm.CompanyUID = types.StringValue(p.CompanyUID)
	orm.ContractTermMonths = types.Int64Value(int64(p.ContractTermMonths))
	orm.CostCentre = types.StringValue(p.CostCentre)
	orm.DiversityZone = types.StringValue(p.DiversityZone)
	orm.LocationID = types.Int64Value(int64(p.LocationID))
	orm.MarketplaceVisibility = types.BoolValue(p.MarketplaceVisibility)
	orm.Name = types.StringValue(p.Name)
	orm.PortSpeed = types.Int64Value(int64(p.PortSpeed))
	orm.LagCount = types.Int64Value(int64(p.LagCount))

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

// NewLagPortResource is a helper function to simplify the provider implementation.
func NewLagPortResource() resource.Resource {
	return &lagPortResource{}
}

// lagPortResource is the resource implementation.
type lagPortResource struct {
	client            *megaport.Client
	cancelAtEndOfTerm bool
}

// Metadata returns the resource type name.
func (r *lagPortResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lag_port"
}

// Schema defines the schema for the resource.
func (r *lagPortResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := commonPortSchemaAttrs()
	attrs["product_name"] = schema.StringAttribute{
		Description: "The name of the LAG port.",
		Required:    true,
	}
	attrs["port_speed"] = schema.Int64Attribute{
		Description: "The speed of the port in Mbps. Can be 10000 (10G), 100000 (100G), or 400000 (400G) where available.",
		Required:    true,
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.RequiresReplace(),
		},
		Validators: []validator.Int64{
			int64validator.OneOf(10000, 100000, 400000),
		},
	}
	attrs["marketplace_visibility"] = schema.BoolAttribute{
		Description: "Whether the port is visible in the Megaport Marketplace. When false, the port is private to your organisation. When true, it is publicly searchable and available for inbound connection requests.",
		Required:    true,
	}
	attrs["lag_count"] = schema.Int64Attribute{
		Description: "The number of LAG ports. Valid values are between 1 and 8.",
		Required:    true,
		Validators: []validator.Int64{
			int64validator.Between(1, 8),
		},
	}
	attrs["lag_port_uids"] = schema.ListAttribute{
		ElementType: types.StringType,
		Description: "The unique identifiers of the individual LAG member ports.",
		Computed:    true,
		PlanModifiers: []planmodifier.List{
			listplanmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema = schema.Schema{
		Description: "Link Aggregation Group (LAG) Port Resource for the Megaport Terraform Provider. This can be used to create, modify, and delete Megaport LAG Ports. A LAG bundles physical ports to create a single data path, where the traffic load is distributed among the ports to increase overall connection reliability.",
		Attributes:  attrs,
	}
}

// Create a new resource.
func (r *lagPortResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan lagPortResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	if err := r.client.PortService.ValidatePortOrder(ctx, buyPortReq); err != nil {
		resp.Diagnostics.AddError(
			"Validation error while attempting to create port",
			fmt.Sprintf("Could not validate port %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	createdPort, err := r.client.PortService.BuyPort(ctx, buyPortReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating port",
			fmt.Sprintf("Could not create port %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	if len(createdPort.TechnicalServiceUIDs) < 1 {
		resp.Diagnostics.AddError(
			"Unexpected number of ports created",
			fmt.Sprintf("Expected at least one port, got %d. Please report this issue to Megaport.", len(createdPort.TechnicalServiceUIDs)),
		)
		return
	}

	createdID := createdPort.TechnicalServiceUIDs[0]

	port, err := r.client.PortService.GetPort(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading created port",
			fmt.Sprintf("Could not read created port %s: %s", createdID, err),
		)
		return
	}

	tags, err := r.fetchResourceTags(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading created port tags",
			fmt.Sprintf("Could not read tags for created port %s: %s", createdID, err),
		)
		return
	}

	resp.Diagnostics.Append(plan.fromAPIPort(ctx, port, tags)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lagPortUIDs, lagDiags := lagPortUIDsList(createdPort.TechnicalServiceUIDs)
	resp.Diagnostics.Append(lagDiags...)
	plan.LagPortUIDs = lagPortUIDs
	plan.UID = types.StringValue(createdID)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read resource information.
func (r *lagPortResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state lagPortResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	port, err := r.client.PortService.GetPort(ctx, state.UID.ValueString())
	if err != nil {
		if isPortNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading port",
			fmt.Sprintf("Could not read port %s: %s", state.UID.ValueString(), err),
		)
		return
	}

	if port.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED {
		resp.State.RemoveResource(ctx)
		return
	}

	tags, err := r.fetchResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading port tags",
			fmt.Sprintf("Could not read tags for port %s: %s", state.UID.ValueString(), err),
		)
		return
	}

	resp.Diagnostics.Append(state.fromAPIPort(ctx, port, tags)...)

	lagPortUIDs, lagDiags := lagPortUIDsList(port.LagPortUIDs)
	resp.Diagnostics.Append(lagDiags...)
	state.LagPortUIDs = lagPortUIDs

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *lagPortResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state lagPortResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := resolvePortModifyParams(
		plan.Name, state.Name,
		plan.MarketplaceVisibility, state.MarketplaceVisibility,
		plan.CostCentre,
		plan.ContractTermMonths, state.ContractTermMonths,
	)

	_, err := r.client.PortService.ModifyPort(ctx, &megaport.ModifyPortRequest{
		PortID:                plan.UID.ValueString(),
		Name:                  params.name,
		MarketplaceVisibility: &params.marketplaceVisibility,
		CostCentre:            params.costCentre,
		ContractTermMonths:    params.contractTermMonths,
		WaitForUpdate:         true,
		WaitForTime:           waitForTime,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating port",
			fmt.Sprintf("Could not update port %s: %s", plan.UID.ValueString(), err),
		)
		return
	}

	port, err := r.client.PortService.GetPort(ctx, plan.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading port",
			fmt.Sprintf("Could not read port %s: %s", plan.UID.ValueString(), err),
		)
		return
	}

	resp.Diagnostics.Append(syncPortResourceTags(ctx, plan.ResourceTags, state.ResourceTags, plan.UID.ValueString(), r.client)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, err := r.fetchResourceTags(ctx, plan.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading port tags",
			fmt.Sprintf("Could not read tags for port %s: %s", plan.UID.ValueString(), err),
		)
		return
	}

	resp.Diagnostics.Append(state.fromAPIPort(ctx, port, tags)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lagPortUIDs, lagDiags := lagPortUIDsList(port.LagPortUIDs)
	resp.Diagnostics.Append(lagDiags...)
	state.LagPortUIDs = lagPortUIDs
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *lagPortResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state lagPortResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.PortService.DeletePort(ctx, &megaport.DeletePortRequest{
		PortID:     state.UID.ValueString(),
		DeleteNow:  !r.cancelAtEndOfTerm,
		SafeDelete: true,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting port",
			fmt.Sprintf("Could not delete port %s: %s", state.UID.ValueString(), err),
		)
	}
}

// Configure adds the provider configured client to the resource.
func (r *lagPortResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data, ok := configureMegaportResource(req, resp)
	if !ok {
		return
	}
	r.client = data.client
	r.cancelAtEndOfTerm = data.cancelAtEndOfTerm
}

func (r *lagPortResource) fetchResourceTags(ctx context.Context, uid string) (map[string]string, error) {
	return r.client.PortService.ListPortResourceTags(ctx, uid)
}

func (r *lagPortResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}

func (r *lagPortResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var plan, state lagPortResourceModel

	if !req.Plan.Raw.IsNull() {
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// If lag_count changes and we have confirmed state, require replacement.
	if !state.UID.IsNull() && !plan.LagCount.IsNull() && !state.LagPortUIDs.IsNull() {
		if len(state.LagPortUIDs.Elements()) != int(plan.LagCount.ValueInt64()) {
			resp.RequiresReplace = append(resp.RequiresReplace, path.Root("lag_count"))
		}
	}
}
