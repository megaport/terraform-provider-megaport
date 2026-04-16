package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

var (
	_ resource.Resource                   = &serviceKeyResource{}
	_ resource.ResourceWithConfigure      = &serviceKeyResource{}
	_ resource.ResourceWithImportState    = &serviceKeyResource{}
	_ resource.ResourceWithValidateConfig = &serviceKeyResource{}
)

// NewServiceKeyResource is a helper function to simplify the provider implementation.
func NewServiceKeyResource() resource.Resource {
	return &serviceKeyResource{}
}

type serviceKeyResource struct {
	client *megaport.Client
}

func (r *serviceKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_key"
}

func (r *serviceKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = serviceKeyResourceSchema()
}

func (r *serviceKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*megaportProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *megaportProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client.client
}

func (r *serviceKeyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config serviceKeyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// When single_use is true, vlan must be set
	if !config.SingleUse.IsNull() && !config.SingleUse.IsUnknown() && config.SingleUse.ValueBool() {
		if config.VLAN.IsNull() || config.VLAN.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("vlan"),
				"Missing required attribute",
				"vlan is required when single_use is true.",
			)
		}
	}
}

func (r *serviceKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, convertDiags := plan.planToCreateRequest(ctx)
	resp.Diagnostics.Append(convertDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createResp, err := r.client.ServiceKeyService.CreateServiceKey(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating service key",
			fmt.Sprintf("Could not create service key for product %s: %s",
				plan.ProductUID.ValueString(), err.Error()),
		)
		return
	}

	// Fetch the full service key details
	apiKey, err := r.client.ServiceKeyService.GetServiceKey(ctx, createResp.ServiceKeyUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading created service key",
			fmt.Sprintf("Could not read created service key %s: %s",
				createResp.ServiceKeyUID, err.Error()),
		)
		return
	}

	var state serviceKeyResourceModel
	fromDiags := state.fromAPI(ctx, apiKey)
	resp.Diagnostics.Append(fromDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serviceKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save ForceNew fields from prior state — the API may not return them
	// correctly (e.g. description comes back empty, max_speed returns port
	// speed). These fields are immutable so prior state is authoritative.
	priorProductUID := state.ProductUID
	priorDescription := state.Description
	priorMaxSpeed := state.MaxSpeed
	priorSingleUse := state.SingleUse
	priorVLAN := state.VLAN
	priorPreApproved := state.PreApproved

	apiKey, err := r.client.ServiceKeyService.GetServiceKey(ctx, state.Key.ValueString())
	if err != nil {
		if apiErr, ok := err.(*megaport.ErrorResponse); ok {
			if apiErr.Response.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error reading service key",
			fmt.Sprintf("Could not read service key: %s", err.Error()),
		)
		return
	}

	fromDiags := state.fromAPI(ctx, apiKey)
	resp.Diagnostics.Append(fromDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Restore ForceNew fields from prior state
	state.ProductUID = priorProductUID
	state.Description = priorDescription
	state.MaxSpeed = priorMaxSpeed
	state.SingleUse = priorSingleUse
	state.VLAN = priorVLAN
	state.PreApproved = priorPreApproved

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serviceKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state serviceKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq, convertDiags := planToUpdateRequest(ctx, &plan, &state)
	resp.Diagnostics.Append(convertDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.ServiceKeyService.UpdateServiceKey(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating service key",
			fmt.Sprintf("Could not update service key %s: %s",
				state.Key.ValueString(), err.Error()),
		)
		return
	}

	// Re-fetch updated state
	apiKey, err := r.client.ServiceKeyService.GetServiceKey(ctx, state.Key.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading updated service key",
			fmt.Sprintf("Could not read updated service key %s: %s",
				state.Key.ValueString(), err.Error()),
		)
		return
	}

	fromDiags := state.fromAPI(ctx, apiKey)
	resp.Diagnostics.Append(fromDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve ForceNew fields from the plan — the API may not return them
	// correctly after an update since they are not part of the update request.
	state.ProductUID = plan.ProductUID
	state.Description = plan.Description
	state.MaxSpeed = plan.MaxSpeed
	state.SingleUse = plan.SingleUse
	state.VLAN = plan.VLAN
	state.PreApproved = plan.PreApproved

	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serviceKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No delete API — deactivate the key instead
	updateReq := &megaport.UpdateServiceKeyRequest{
		Key:        state.Key.ValueString(),
		ProductUID: state.ProductUID.ValueString(),
		SingleUse:  state.SingleUse.ValueBool(),
		Active:     false,
	}

	_, err := r.client.ServiceKeyService.UpdateServiceKey(ctx, updateReq)
	if err != nil {
		if apiErr, ok := err.(*megaport.ErrorResponse); ok {
			if apiErr.Response.StatusCode == http.StatusNotFound {
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error deactivating service key",
			fmt.Sprintf("Could not deactivate service key %s: %s",
				state.Key.ValueString(), err.Error()),
		)
		return
	}
}

func (r *serviceKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	apiKey, err := r.client.ServiceKeyService.GetServiceKey(ctx, req.ID)
	if err != nil {
		if apiErr, ok := err.(*megaport.ErrorResponse); ok {
			if apiErr.Response.StatusCode == http.StatusNotFound {
				resp.Diagnostics.AddError(
					"Resource not found",
					fmt.Sprintf("Service key %s does not exist", req.ID),
				)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error importing service key",
			fmt.Sprintf("Could not read service key %s: %s", req.ID, err.Error()),
		)
		return
	}

	var state serviceKeyResourceModel
	fromDiags := state.fromAPI(ctx, apiKey)
	resp.Diagnostics.Append(fromDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
