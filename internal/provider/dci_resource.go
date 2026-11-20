package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &dciResource{}
	_ resource.ResourceWithConfigure   = &dciResource{}
	_ resource.ResourceWithImportState = &dciResource{}
	_ resource.ResourceWithModifyPlan  = &dciResource{}
)

// dciResourceModel maps the resource schema data.
type dciResourceModel struct {
	UID                types.String `tfsdk:"solution_uid"`
	Name               types.String `tfsdk:"name"`
	AEndLocationID     types.Int64  `tfsdk:"a_end_location_id"`
	BEndLocationID     types.Int64  `tfsdk:"b_end_location_id"`
	Speed              types.Int64  `tfsdk:"speed"`
	ContractTermMonths types.Int64  `tfsdk:"contract_term_months"`
	AutoRenew          types.Bool   `tfsdk:"auto_renew"`
	CostCentre         types.String `tfsdk:"cost_centre"`
	ProvisioningStatus types.String `tfsdk:"provisioning_status"`
	AEndPortUID        types.String `tfsdk:"a_end_port_uid"`
	BEndPortUID        types.String `tfsdk:"b_end_port_uid"`
	VXCUID             types.String `tfsdk:"vxc_uid"`
}

// fromAPI fills the model from a DCI and its billing.
func (orm *dciResourceModel) fromAPI(solution *megaport.Solution, billable *megaport.SolutionBillable) {
	orm.UID = types.StringValue(solution.UID)
	orm.Name = types.StringValue(solution.Name)
	orm.CostCentre = types.StringValue(solution.ServiceLevelReference)
	orm.ProvisioningStatus = types.StringValue(solution.ProvisioningStatus)
	orm.AEndPortUID = types.StringValue(solution.MemberUID(megaport.SolutionMemberRoleAEndPort))
	orm.BEndPortUID = types.StringValue(solution.MemberUID(megaport.SolutionMemberRoleBEndPort))
	orm.VXCUID = types.StringValue(solution.MemberUID(megaport.SolutionMemberRoleVXC))

	// The API reports the DCI's locations and speed from its VXC, so they are empty until the VXC is created.
	// Keep the ordered values until then.
	if id := dciLocationID(solution, solution.ALocationID, megaport.SolutionMemberRoleAEndPort); id != 0 {
		orm.AEndLocationID = types.Int64Value(int64(id))
	}
	if id := dciLocationID(solution, solution.BLocationID, megaport.SolutionMemberRoleBEndPort); id != 0 {
		orm.BEndLocationID = types.Int64Value(int64(id))
	}
	if solution.SpeedMbps != 0 {
		orm.Speed = types.Int64Value(int64(solution.SpeedMbps))
	}
	// A DCI rolls to month-to-month (1) when its term ends without auto-renew.
	// Keep the ordered term then, so the config does not drift from state.
	if billable.ContractTermMonths != 1 || orm.ContractTermMonths.IsNull() || orm.ContractTermMonths.IsUnknown() {
		orm.ContractTermMonths = types.Int64Value(int64(billable.ContractTermMonths))
	}
	orm.AutoRenew = types.BoolValue(billable.AutoRenewTerm)
}

// fillUnknown takes from another model each value this one does not know yet.
func (orm *dciResourceModel) fillUnknown(from dciResourceModel) {
	fillString := func(to *types.String, from types.String) {
		if to.IsUnknown() {
			*to = from
		}
	}
	fillInt64 := func(to *types.Int64, from types.Int64) {
		if to.IsUnknown() {
			*to = from
		}
	}
	fillString(&orm.UID, from.UID)
	fillString(&orm.Name, from.Name)
	fillInt64(&orm.AEndLocationID, from.AEndLocationID)
	fillInt64(&orm.BEndLocationID, from.BEndLocationID)
	fillInt64(&orm.Speed, from.Speed)
	fillInt64(&orm.ContractTermMonths, from.ContractTermMonths)
	if orm.AutoRenew.IsUnknown() {
		orm.AutoRenew = from.AutoRenew
	}
	fillString(&orm.CostCentre, from.CostCentre)
	fillString(&orm.ProvisioningStatus, from.ProvisioningStatus)
	fillString(&orm.AEndPortUID, from.AEndPortUID)
	fillString(&orm.BEndPortUID, from.BEndPortUID)
	fillString(&orm.VXCUID, from.VXCUID)
}

// dciLocationID returns the location of one end of a DCI: the DCI's own, or else that of the Port at that end.
func dciLocationID(s *megaport.Solution, locationID int, portRole string) int {
	if locationID != 0 {
		return locationID
	}
	for _, m := range s.Members {
		if m.Role == portRole {
			return m.LocationID
		}
	}
	return 0
}

// NewDCIResource is a helper function to simplify the provider implementation.
func NewDCIResource() resource.Resource {
	return &dciResource{}
}

// dciResource is the resource implementation.
type dciResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *dciResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dci"
}

// Schema defines the schema for the resource.
func (r *dciResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data Centre Interconnect (DCI) Resource for the Megaport Terraform Provider. A DCI is a Port at each of two locations and a VXC between them, ordered, priced and billed as one item. This resource orders a DCI, reads it, and changes its name and cost centre. Other changes and destroying a DCI with Terraform are not supported yet: cancel a DCI in the Megaport Portal.",
		Attributes: map[string]schema.Attribute{
			"solution_uid": schema.StringAttribute{
				Description: "The unique identifier of the DCI.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the DCI. Both Ports and the VXC are ordered with this name.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"a_end_location_id": schema.Int64Attribute{
				Description: "The numeric location ID of the A-End Port. This value can be retrieved from the data source megaport_location.",
				Required:    true,
			},
			"b_end_location_id": schema.Int64Attribute{
				Description: "The numeric location ID of the B-End Port. This value can be retrieved from the data source megaport_location.",
				Required:    true,
			},
			"speed": schema.Int64Attribute{
				Description: "The speed of the DCI in Mbps. Both Ports and the VXC are ordered at this speed, and both locations must offer it.",
				Required:    true,
			},
			"contract_term_months": schema.Int64Attribute{
				Description: "The term of the contract in months. When the term ends without auto-renew, the DCI moves to a month-to-month term, and this attribute keeps the term that was ordered. After an import, it reads 1 if the DCI already moved to month-to-month.",
				Required:    true,
			},
			"auto_renew": schema.BoolAttribute{
				Description: "Whether the term renews when it ends. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"cost_centre": schema.StringAttribute{
				Description: "A customer reference number to be included in billing information and invoices. Also known as the service level reference (SLR) number. To clear it, set it to an empty string: removing the attribute keeps the current value.",
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{stringvalidator.LengthAtMost(255)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"provisioning_status": schema.StringAttribute{
				Description: "The provisioning status of the DCI, rolled up from its Ports and VXC.",
				Computed:    true,
			},
			"a_end_port_uid": schema.StringAttribute{
				Description: "The unique identifier of the A-End Port.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"b_end_port_uid": schema.StringAttribute{
				Description: "The unique identifier of the B-End Port.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vxc_uid": schema.StringAttribute{
				Description: "The unique identifier of the VXC between the two Ports.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create a new resource.
func (r *dciResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan dciResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// One order holds the DCI, both Ports and the VXC, so the DCI is priced and billed as one item.
	createdDCI, err := r.client.SolutionService.BuyDCI(ctx, &megaport.BuyDCIRequest{
		Name:                  plan.Name.ValueString(),
		AEndLocationID:        int(plan.AEndLocationID.ValueInt64()),
		BEndLocationID:        int(plan.BEndLocationID.ValueInt64()),
		PortSpeed:             int(plan.Speed.ValueInt64()),
		Term:                  int(plan.ContractTermMonths.ValueInt64()),
		AutoRenewTerm:         plan.AutoRenew.ValueBool(),
		ServiceLevelReference: plan.CostCentre.ValueString(),
		WaitForProvision:      true,
		WaitForTime:           waitForTime,
	})
	if err != nil && createdDCI == nil {
		resp.Diagnostics.AddError(
			"Error buying DCI",
			"Could not create DCI with name "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	// Save the UID first, so a failed create leaves a tainted resource instead of an orphan.
	createdID := createdDCI.SolutionUID
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("solution_uid"), createdID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"DCI ordered but not ready",
			fmt.Sprintf("DCI %s (%s) was ordered but is not ready yet: %s. Do not apply again: Terraform would try to replace the DCI. Once it is ready, run `terraform untaint` on the resource, then `terraform apply -refresh-only`.", plan.Name.ValueString(), createdID, err),
		)
		return
	}

	// get the created DCI
	solution, billable, err := r.readDCI(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"DCI ordered but could not be read",
			fmt.Sprintf("DCI %s (%s) was ordered but could not be read: %s. Do not apply again: Terraform would try to replace the DCI. Run `terraform untaint` on the resource, then `terraform apply -refresh-only`.", plan.Name.ValueString(), createdID, err),
		)
		return
	}

	// update the plan with the DCI info
	plan.fromAPI(solution, billable)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *dciResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state dciResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed DCI value from API
	solution, billable, err := r.readDCI(ctx, state.UID.ValueString())
	if err != nil {
		detail := err.Error()
		// The API never deletes a DCI: a canceled one still reads back, with active set to false.
		// It answers 404 when DCI ordering is off for the account or the account does not own the DCI,
		// so a 404 keeps the DCI in state. Dropping it would leave a billed DCI that Terraform no longer tracks.
		var mpErr *megaport.ErrorResponse
		if errors.As(err, &mpErr) && mpErr.Response != nil && mpErr.Response.StatusCode == http.StatusNotFound {
			detail = "DCI is not available for this account, or the account does not own it: " + detail
		}
		resp.Diagnostics.AddError(
			"Error Reading DCI",
			"Could not read DCI with ID "+state.UID.ValueString()+": "+detail,
		)
		return
	}

	// A Solution of another type cannot be managed as a DCI.
	if solution.Type != megaport.SolutionTypeDCI {
		resp.Diagnostics.AddError(
			"Error Reading DCI",
			"Could not read DCI with ID "+state.UID.ValueString()+": it is a Solution of type "+solution.Type+", not a DCI.",
		)
		return
	}

	// If the DCI has been canceled
	if !solution.Active {
		resp.State.RemoveResource(ctx)
		return
	}

	state.fromAPI(solution, billable)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// readDCI reads a DCI and its billing, which holds the term and the auto-renew setting.
func (r *dciResource) readDCI(ctx context.Context, uid string) (*megaport.Solution, *megaport.SolutionBillable, error) {
	solution, err := r.client.SolutionService.GetSolution(ctx, uid)
	if err != nil {
		return nil, nil, err
	}
	billable, err := r.client.SolutionService.GetSolutionBillable(ctx, uid)
	if err != nil {
		return nil, nil, err
	}
	return solution, billable, nil
}

// ModifyPlan refuses, at plan time, the changes the API cannot make to a DCI.
// No attribute forces a replace instead: a replace would order a second DCI under create_before_destroy,
// and Delete could not cancel the first.
func (r *dciResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state dciResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A value the state does not hold yet, as after a create that timed out, is not a change.
	// A value not known until apply is checked then, when Terraform plans the DCI again.
	check := func(name string, planned, current attr.Value) {
		if current.IsNull() || planned.IsUnknown() || planned.Equal(current) {
			return
		}
		resp.Diagnostics.AddAttributeError(
			path.Root(name),
			"Changing a DCI is not supported",
			fmt.Sprintf("Only the name and the cost centre of a DCI can be changed, and the configuration changes %s from %s to %s. The locations and the speed of a DCI cannot be changed, and this version of the provider cannot change the term or the auto-renew setting. Set %s back to %s.", name, current, planned, name, current),
		)
	}
	check("a_end_location_id", plan.AEndLocationID, state.AEndLocationID)
	check("b_end_location_id", plan.BEndLocationID, state.BEndLocationID)
	check("speed", plan.Speed, state.Speed)
	check("contract_term_months", plan.ContractTermMonths, state.ContractTermMonths)
	check("auto_renew", plan.AutoRenew, state.AutoRenew)
}

// Update changes the name and the cost center of a DCI. ModifyPlan refuses every other change before it gets here.
func (r *dciResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state dciResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid := state.UID.ValueString()
	// As in ModifyPlan, a value the state does not hold yet, or one not known, is not a change:
	// sending it would overwrite the value the DCI has, and an unknown cost centre would clear it.
	changed := func(planned, current types.String) bool {
		return !current.IsNull() && !planned.IsUnknown() && !planned.Equal(current)
	}
	updated := changed(plan.Name, state.Name) || changed(plan.CostCentre, state.CostCentre)
	if updated {
		// The API sets both on every update, and an empty reference clears it,
		// so the cost centre is sent even when only the name changes.
		err := r.client.SolutionService.UpdateSolution(ctx, &megaport.UpdateSolutionRequest{
			SolutionUID:           uid,
			Name:                  plan.Name.ValueString(),
			ServiceLevelReference: plan.CostCentre.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating DCI",
				"Could not update DCI with ID "+uid+": "+err.Error(),
			)
			return
		}
	}

	// get the updated DCI
	solution, billable, err := r.readDCI(ctx, uid)
	if err != nil {
		if !updated {
			resp.Diagnostics.AddError(
				"Error Reading DCI",
				"Could not read DCI with ID "+uid+": "+err.Error(),
			)
			return
		}
		resp.Diagnostics.AddError(
			"DCI updated but could not be read",
			"DCI with ID "+uid+" was updated but could not be read: "+err.Error()+". Run `terraform apply -refresh-only` so the Terraform state matches.",
		)
		return
	}

	// Take from the read only what the plan left unknown: Terraform rejects a result that differs from what it planned,
	// and the read can differ, as when the VXC is created after the refresh. The next refresh picks up the rest.
	read := plan
	read.fromAPI(solution, billable)
	plan.fillUnknown(read)

	// Set state to fully populated data
	diags := resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete is not supported yet. It keeps the DCI in state, because the DCI is still live and billed.
func (r *dciResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError(
		"Canceling a DCI is not supported",
		"This version of the provider cannot cancel a DCI. Cancel it in the Megaport Portal, then remove it from the Terraform state with `terraform state rm`.",
	)
}

// Configure adds the provider configured client to the resource.
func (r *dciResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dciResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("solution_uid"), req, resp)
}
