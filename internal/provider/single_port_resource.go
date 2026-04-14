package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &portResource{}
	_ resource.ResourceWithConfigure   = &portResource{}
	_ resource.ResourceWithImportState = &portResource{}
	_ resource.ResourceWithMoveState   = &portResource{}
)

// singlePortResourceModel maps the resource schema data.
type singlePortResourceModel struct {
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

	Resources    types.Object `tfsdk:"resources"`
	ResourceTags types.Map    `tfsdk:"resource_tags"`
}

func (orm *singlePortResourceModel) fromAPIPort(ctx context.Context, p *megaport.Port, tags map[string]string) diag.Diagnostics {
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
func NewPortResource() resource.Resource {
	return &portResource{}
}

// portResource is the resource implementation.
type portResource struct {
	client            *megaport.Client
	cancelAtEndOfTerm bool
}

// Metadata returns the resource type name.
func (r *portResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port"
}

// Schema defines the schema for the resource.
func (r *portResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := commonPortSchemaAttrs()
	attrs["product_name"] = schema.StringAttribute{
		Description: "The name of the port. Specify a name that is easily identifiable, particularly if you plan on having more than one port.",
		Required:    true,
	}
	attrs["port_speed"] = schema.Int64Attribute{
		Description: "The speed of the port in Mbps. Can be 1000 (1G), 10000 (10G), 100000 (100G), or 400000 (400G) where available.",
		Required:    true,
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.RequiresReplace(),
		},
		Validators: []validator.Int64{
			int64validator.OneOf(1000, 10000, 100000, 400000),
		},
	}
	attrs["marketplace_visibility"] = schema.BoolAttribute{
		Description: "Whether the port is visible in the Megaport Marketplace. When false, the port is private to your organisation. When true, it is publicly searchable and available for inbound connection requests.",
		Required:    true,
	}
	resp.Schema = schema.Schema{
		Description: "Single Port Resource for the Megaport Terraform Provider. This can be used to create, modify, and delete Megaport Ports. Your organization's Port is the physical point of connection between your organization's network and the Megaport network. You will need to deploy a Port wherever you want to direct traffic.",
		Attributes:  attrs,
	}
}

// Create a new resource.
func (r *portResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan singlePortResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	if len(createdPort.TechnicalServiceUIDs) != 1 {
		resp.Diagnostics.AddError(
			"Unexpected number of ports created",
			fmt.Sprintf("Expected 1 port, got %d. Please report this issue to Megaport.", len(createdPort.TechnicalServiceUIDs)),
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
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read resource information.
func (r *portResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state singlePortResourceModel
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
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *portResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state singlePortResourceModel
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
		ContractTermMonths:    params.contractTermMonths,
		CostCentre:            params.costCentre,
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
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *portResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state singlePortResourceModel
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
func (r *portResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data, ok := configureMegaportResource(req, resp)
	if !ok {
		return
	}
	r.client = data.client
	r.cancelAtEndOfTerm = data.cancelAtEndOfTerm
}

func (r *portResource) fetchResourceTags(ctx context.Context, uid string) (map[string]string, error) {
	return r.client.PortService.ListPortResourceTags(ctx, uid)
}

func (r *portResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}

// MoveState implements resource.ResourceWithMoveState to support automatic
// V1-to-V2 state migration when users move from megaport/megaport (v1) to v2.
func (r *portResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		{
			StateMover: moveStatePort,
		},
	}
}

// moveStatePort migrates a V1 megaport_port state to V2 by extracting only the
// fields that exist in the V2 schema and dropping all removed fields.
func moveStatePort(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	if req.SourceProviderAddress != "registry.terraform.io/megaport/megaport" || req.SourceTypeName != "megaport_port" {
		return
	}

	if req.SourceRawState == nil {
		resp.Diagnostics.AddError("Unable to migrate V1 state", "Source raw state is nil")
		return
	}
	rawJSON := req.SourceRawState.JSON
	if len(rawJSON) == 0 {
		resp.Diagnostics.AddError("Unable to migrate V1 state", "Source raw state JSON is empty")
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		resp.Diagnostics.AddError("Unable to unmarshal V1 state", err.Error())
		return
	}

	model, diags := portModelFromV1RawState(ctx, raw)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.TargetState.Set(ctx, model)...)
}
