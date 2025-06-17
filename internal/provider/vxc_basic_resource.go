package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &vxcBasicResource{}
	_ resource.ResourceWithConfigure   = &vxcBasicResource{}
	_ resource.ResourceWithImportState = &vxcBasicResource{}
)

// NewVXCBasicResource is a helper function to simplify the provider implementation.
func NewVXCBasicResource() resource.Resource {
	return &vxcBasicResource{}
}

// vxcBasicResource is the resource implementation.
type vxcBasicResource struct {
	client *megaport.Client
}

// Configure adds the provider configured client to the resource.
func (r *vxcBasicResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create a new resource.
func (r *vxcBasicResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan vxcBasicResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	buyReq := &megaport.BuyVXCRequest{
		VXCName:    plan.Name.ValueString(),
		Term:       int(plan.ContractTermMonths.ValueInt64()),
		RateLimit:  int(plan.RateLimit.ValueInt64()),
		PromoCode:  plan.PromoCode.ValueString(),
		CostCentre: plan.CostCentre.ValueString(),
		ServiceKey: plan.ServiceKey.ValueString(),

		WaitForProvision: true,
		WaitForTime:      waitForTime,
	}

	if !plan.Shutdown.IsNull() {
		buyReq.Shutdown = plan.Shutdown.ValueBool()
	}

	if !plan.ResourceTags.IsNull() {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		buyReq.ResourceTags = tagMap
	}

	aEndObj := plan.AEndConfiguration
	bEndObj := plan.BEndConfiguration

	var a vxcBasicEndConfigurationModel
	aEndDiags := aEndObj.As(ctx, &a, basetypes.ObjectAsOptions{})
	if aEndDiags.HasError() {
		resp.Diagnostics.Append(aEndDiags...)
		return
	}

	buyReq.PortUID = a.RequestedProductUID.ValueString()

	endDiags, aEndMegaportConfig, aEndPartnerConfig := r.createVXCBasicEndConfiguration(ctx, plan.Name.ValueString(), int(plan.RateLimit.ValueInt64()), a, plan.AEndPartnerConfig)
	resp.Diagnostics.Append(endDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.AEndPartnerConfig = aEndPartnerConfig
	buyReq.AEndConfiguration = aEndMegaportConfig

	var b vxcBasicEndConfigurationModel
	bEndDiags := bEndObj.As(ctx, &b, basetypes.ObjectAsOptions{})
	if bEndDiags.HasError() {
		resp.Diagnostics.Append(bEndDiags...)
		return
	}

	endDiags, bEndMegaportConfig, bEndPartnerConfig := r.createVXCBasicEndConfiguration(ctx, plan.Name.ValueString(), int(plan.RateLimit.ValueInt64()), b, plan.BEndPartnerConfig)
	resp.Diagnostics.Append(endDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.BEndPartnerConfig = bEndPartnerConfig
	buyReq.BEndConfiguration = bEndMegaportConfig

	err := r.client.VXCService.ValidateVXCOrder(ctx, buyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Validation error while attempting to create VXC",
			"Validation error while attempting to create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	createdVXC, err := r.client.VXCService.BuyVXC(ctx, buyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating VXC",
			"Could not order VXC with name "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	createdID := createdVXC.TechnicalServiceUID

	// get the created VXC
	vxc, err := r.client.VXCService.GetVXC(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading newly created VXC",
			"Could not read newly created VXC with ID "+createdID+": "+err.Error(),
		)
		return
	}

	tags, err := r.client.VXCService.ListVXCResourceTags(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading tags for newly created VXC",
			"Could not read tags for newly created VXC with ID "+createdID+": "+err.Error(),
		)
		return
	}

	aEndProductType, _ := r.client.ProductService.GetProductType(ctx, vxc.AEndConfiguration.UID)
	bEndProductType, _ := r.client.ProductService.GetProductType(ctx, vxc.BEndConfiguration.UID)

	// update the plan with the VXC info
	apiDiags := plan.fromAPIVXC(ctx, vxc, tags, aEndProductType, bEndProductType)
	resp.Diagnostics.Append(apiDiags...)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *vxcBasicResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state vxcBasicResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed vxc value from API
	vxc, err := r.client.VXCService.GetVXC(ctx, state.UID.ValueString())
	if err != nil {
		// VXC has been deleted or is not found
		if mpErr, ok := err.(*megaport.ErrorResponse); ok {
			if mpErr.Response.StatusCode == http.StatusNotFound ||
				(mpErr.Response.StatusCode == http.StatusBadRequest && strings.Contains(mpErr.Message, "Could not find a service with UID")) {
				resp.State.RemoveResource(ctx)
				return
			}
		}

		resp.Diagnostics.AddError(
			"Error Reading VXC",
			"Could not read VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	// If the vxc has been deleted
	if vxc.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED {
		resp.State.RemoveResource(ctx)
		return
	}

	// Get tags
	tags, err := r.client.VXCService.ListVXCResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading tags for VXC",
			"Could not read tags for VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	aEndProductType, _ := r.client.ProductService.GetProductType(ctx, vxc.AEndConfiguration.UID)
	bEndProductType, _ := r.client.ProductService.GetProductType(ctx, vxc.BEndConfiguration.UID)

	apiDiags := state.fromAPIVXC(ctx, vxc, tags, aEndProductType, bEndProductType)
	resp.Diagnostics.Append(apiDiags...)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	aEndConfig := &vxcBasicEndConfigurationModel{}
	bEndConfig := &vxcBasicEndConfigurationModel{}
	aEndConfigDiags := state.AEndConfiguration.As(ctx, aEndConfig, basetypes.ObjectAsOptions{})
	bEndConfigDiags := state.BEndConfiguration.As(ctx, bEndConfig, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(aEndConfigDiags...)
	resp.Diagnostics.Append(bEndConfigDiags...)
}

func (r *vxcBasicResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vxcBasicResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &megaport.UpdateVXCRequest{
		WaitForUpdate: true,
		WaitForTime:   waitForTime,
	}

	if !plan.Name.Equal(state.Name) {
		updateReq.Name = megaport.PtrTo(plan.Name.ValueString())
	}

	aEndDiags, aEndState, aEndPartnerObj, aEndMegaportPartnerConfig, aEndRequestedProductUID, aEndVLAN, aEndInnerVLAN, aEndVnicIndex, aEndCSP := r.makeUpdateEndConfig(ctx, plan.Name.ValueString(), plan.AEndConfiguration, state.AEndConfiguration, plan.AEndPartnerConfig, state.AEndPartnerConfig)
	resp.Diagnostics.Append(aEndDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !aEndCSP && !aEndPartnerObj.IsNull() {
		// Only update the partner config if it is not a CSP partner config.
		state.AEndPartnerConfig = aEndPartnerObj
	}

	updateReq.AEndPartnerConfig = aEndMegaportPartnerConfig
	updateReq.AEndVLAN = aEndVLAN
	updateReq.AEndInnerVLAN = aEndInnerVLAN
	updateReq.AEndProductUID = aEndRequestedProductUID
	updateReq.AVnicIndex = aEndVnicIndex

	bEndDiags, bEndState, bEndPartnerObj, bEndMegaportPartnerConfig, bEndRequestedProductUID, bEndVLAN, bEndInnerVLAN, bEndVnicIndex, bEndCSP := r.makeUpdateEndConfig(ctx, plan.Name.ValueString(), plan.BEndConfiguration, state.BEndConfiguration, plan.BEndPartnerConfig, state.BEndPartnerConfig)
	resp.Diagnostics.Append(bEndDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Only update the partner config if it is not a CSP partner config.
	if !bEndCSP && !bEndPartnerObj.IsNull() {
		state.BEndPartnerConfig = bEndPartnerObj
	}
	updateReq.BEndPartnerConfig = bEndMegaportPartnerConfig
	updateReq.BEndVLAN = bEndVLAN
	updateReq.BEndInnerVLAN = bEndInnerVLAN
	updateReq.BEndProductUID = bEndRequestedProductUID
	updateReq.BVnicIndex = bEndVnicIndex

	if !plan.RateLimit.IsNull() && !plan.RateLimit.Equal(state.RateLimit) {
		updateReq.RateLimit = megaport.PtrTo(int(plan.RateLimit.ValueInt64()))
	}

	if !plan.CostCentre.IsNull() && !plan.CostCentre.Equal(state.CostCentre) {
		updateReq.CostCentre = megaport.PtrTo(plan.CostCentre.ValueString())
	}

	if !plan.Shutdown.IsNull() && !plan.Shutdown.Equal(state.Shutdown) {
		updateReq.Shutdown = megaport.PtrTo(plan.Shutdown.ValueBool())
	}

	if !plan.ContractTermMonths.IsNull() && !plan.ContractTermMonths.Equal(state.ContractTermMonths) {
		updateReq.Term = megaport.PtrTo(int(plan.ContractTermMonths.ValueInt64()))
	}

	_, err := r.client.VXCService.UpdateVXC(ctx, plan.UID.ValueString(), updateReq)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating VXC",
			"Could not update VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Set updated state
	state.AEndConfiguration = aEndState
	state.BEndConfiguration = bEndState

	// Get refreshed vxc value from API
	vxc, err := r.client.VXCService.GetVXC(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VXC",
			"Could not read VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Save partner configurations before calling fromAPIVXC
	savedAEndPartnerConfig := state.AEndPartnerConfig
	savedBEndPartnerConfig := state.BEndPartnerConfig

	if !plan.ResourceTags.Equal(state.ResourceTags) {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := r.client.VXCService.UpdateVXCResourceTags(ctx, state.UID.ValueString(), tagMap)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating tags for VXC",
				"Could not update tags for VXC with ID "+state.UID.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	// Get resource tags
	tags, err := r.client.VXCService.ListVXCResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VXC Tags",
			"Could not read VXC tags with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	aEndProductType, _ := r.client.ProductService.GetProductType(ctx, vxc.AEndConfiguration.UID)
	bEndProductType, _ := r.client.ProductService.GetProductType(ctx, vxc.BEndConfiguration.UID)

	apiDiags := state.fromAPIVXC(ctx, vxc, tags, aEndProductType, bEndProductType)
	resp.Diagnostics.Append(apiDiags...)

	state.AEndPartnerConfig = savedAEndPartnerConfig
	state.BEndPartnerConfig = savedBEndPartnerConfig

	// Set refreshed state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *vxcBasicResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state vxcBasicResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	err := r.client.VXCService.DeleteVXC(ctx, state.UID.ValueString(), &megaport.DeleteVXCRequest{
		DeleteNow: true,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting VXC",
			"Could not delete VXC, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *vxcBasicResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Get current state
	var plan, state vxcBasicResourceModel
	diags := diag.Diagnostics{}

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

	// If VXC is not yet created, return
	if !state.UID.IsNull() {
		if !req.Plan.Raw.IsNull() {
			aEndStateObj := state.AEndConfiguration
			bEndStateObj := state.BEndConfiguration

			diags, endPlanObj, statePlanObj, statePartner, requiresReplace := modifyPlanBasicEndConfig(ctx, plan.AEndConfiguration, aEndStateObj, plan.AEndPartnerConfig, state.AEndPartnerConfig)
			resp.Diagnostics.Append(diags...)
			if diags.HasError() {
				return
			}
			plan.AEndConfiguration = endPlanObj
			state.AEndConfiguration = statePlanObj
			state.AEndPartnerConfig = statePartner
			if requiresReplace != nil {
				resp.RequiresReplace = append(resp.RequiresReplace, requiresReplace...)
			}
			diags, endPlanObj, statePlanObj, statePartner, requiresReplace = modifyPlanBasicEndConfig(ctx, plan.BEndConfiguration, bEndStateObj, plan.BEndPartnerConfig, state.BEndPartnerConfig)
			resp.Diagnostics.Append(diags...)
			if diags.HasError() {
				return
			}
			if requiresReplace != nil {
				resp.RequiresReplace = append(resp.RequiresReplace, requiresReplace...)
			}
			plan.BEndConfiguration = endPlanObj
			state.BEndConfiguration = statePlanObj
			state.BEndPartnerConfig = statePartner

			req.Plan.Set(ctx, &plan)
			resp.Plan.Set(ctx, &plan)
			stateDiags := req.State.Set(ctx, &state)
			diags = append(diags, stateDiags...)
			resp.Diagnostics.Append(diags...)
		}
	}

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *vxcBasicResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}

// Metadata returns the resource type name.
func (r *vxcBasicResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vxc_basic"
}
