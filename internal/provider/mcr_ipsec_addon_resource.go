package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &mcrIpsecAddonResource{}
	_ resource.ResourceWithConfigure   = &mcrIpsecAddonResource{}
	_ resource.ResourceWithImportState = &mcrIpsecAddonResource{}
)

// NewMCRIpsecAddonResource is a helper function to simplify the provider implementation.
func NewMCRIpsecAddonResource() resource.Resource {
	return &mcrIpsecAddonResource{}
}

// mcrIpsecAddonResource defines the resource implementation.
type mcrIpsecAddonResource struct {
	client *megaport.Client
}

// mcrIpsecAddonResourceModel maps the resource schema data.
type mcrIpsecAddonResourceModel struct {
	MCRID       types.String `tfsdk:"mcr_id"`
	TunnelCount types.Int64  `tfsdk:"tunnel_count"`
	AddOnUID    types.String `tfsdk:"add_on_uid"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

// Metadata returns the resource type name.
func (r *mcrIpsecAddonResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcr_ipsec_addon"
}

// Schema defines the schema for the resource.
func (r *mcrIpsecAddonResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Megaport MCR IPSec Add-On Resource. Attaches an IPSec tunnel add-on to an existing MCR. Valid tunnel counts are 10, 20, and 30. Only one IPSec add-on may be attached to an MCR at a time.",
		Attributes: map[string]schema.Attribute{
			"mcr_id": schema.StringAttribute{
				Description: "The UID of the MCR to attach the IPSec add-on to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tunnel_count": schema.Int64Attribute{
				Description: "Number of IPSec tunnels. Valid values are 10, 20, and 30.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(10, 20, 30),
				},
			},
			"add_on_uid": schema.StringAttribute{
				Description: "UID of the IPSec add-on, assigned by the API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Last updated by the Terraform provider.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *mcrIpsecAddonResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := configureMegaportResource(req, resp)
	if !ok {
		return
	}
	r.client = providerData.client
}

// Create creates the resource and sets the initial Terraform state.
func (r *mcrIpsecAddonResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mcrIpsecAddonResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.MCRService.UpdateMCRWithAddOn(ctx, plan.MCRID.ValueString(), megaport.MCRAddOnRequest{
		AddOn: &megaport.MCRAddOnIPsecConfig{
			TunnelCount: int(plan.TunnelCount.ValueInt64()),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating MCR IPSec add-on",
			fmt.Sprintf("Could not create IPSec add-on for MCR %s: %s", plan.MCRID.ValueString(), err.Error()),
		)
		return
	}

	if waitErr := r.waitForMCRReady(ctx, plan.MCRID.ValueString()); waitErr != nil {
		resp.Diagnostics.AddError(
			"Error waiting for MCR to become ready",
			fmt.Sprintf("MCR %s did not reach ready state after IPSec add-on creation: %s", plan.MCRID.ValueString(), waitErr.Error()),
		)
		return
	}

	mcr, err := r.client.MCRService.GetMCR(ctx, plan.MCRID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading MCR after IPSec add-on creation",
			fmt.Sprintf("Could not read MCR %s: %s", plan.MCRID.ValueString(), err.Error()),
		)
		return
	}

	state := r.fromAPI(plan.MCRID.ValueString(), mcr)

	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *mcrIpsecAddonResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mcrIpsecAddonResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mcr, err := r.client.MCRService.GetMCR(ctx, state.MCRID.ValueString())
	if err != nil {
		if apiErr, ok := err.(*megaport.ErrorResponse); ok && apiErr.Response != nil {
			if apiErr.Response.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error reading MCR for IPSec add-on",
			fmt.Sprintf("Could not read MCR %s: %s", state.MCRID.ValueString(), err.Error()),
		)
		return
	}

	newState := r.fromAPI(state.MCRID.ValueString(), mcr)

	// If add-on is no longer present on the MCR, remove from state
	if newState.AddOnUID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	newState.LastUpdated = state.LastUpdated
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *mcrIpsecAddonResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state mcrIpsecAddonResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.MCRService.UpdateMCRIPsecAddOn(ctx,
		state.MCRID.ValueString(),
		state.AddOnUID.ValueString(),
		int(plan.TunnelCount.ValueInt64()),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating MCR IPSec add-on",
			fmt.Sprintf("Could not update IPSec add-on %s for MCR %s: %s",
				state.AddOnUID.ValueString(), state.MCRID.ValueString(), err.Error()),
		)
		return
	}

	if waitErr := r.waitForMCRReady(ctx, state.MCRID.ValueString()); waitErr != nil {
		resp.Diagnostics.AddError(
			"Error waiting for MCR to become ready",
			fmt.Sprintf("MCR %s did not reach ready state after IPSec add-on update: %s", state.MCRID.ValueString(), waitErr.Error()),
		)
		return
	}

	mcr, err := r.client.MCRService.GetMCR(ctx, state.MCRID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading MCR after IPSec add-on update",
			fmt.Sprintf("Could not read MCR %s: %s", state.MCRID.ValueString(), err.Error()),
		)
		return
	}

	newState := r.fromAPI(state.MCRID.ValueString(), mcr)

	newState.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mcrIpsecAddonResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mcrIpsecAddonResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Disable IPSec by setting tunnel count to 0
	err := r.client.MCRService.UpdateMCRIPsecAddOn(ctx,
		state.MCRID.ValueString(),
		state.AddOnUID.ValueString(),
		0,
	)
	if err != nil {
		if apiErr, ok := err.(*megaport.ErrorResponse); ok && apiErr.Response != nil {
			if apiErr.Response.StatusCode == http.StatusNotFound {
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error deleting MCR IPSec add-on",
			fmt.Sprintf("Could not disable IPSec add-on %s for MCR %s: %s",
				state.AddOnUID.ValueString(), state.MCRID.ValueString(), err.Error()),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *mcrIpsecAddonResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: mcr_uid:add_on_uid
	mcrUID, addOnUID, err := parseImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Error parsing import ID: %s\n\nExpected format: mcr_uid:add_on_uid\nExample: 12345678-1234-1234-1234-123456789012:add-on-uid", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mcr_id"), mcrUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("add_on_uid"), addOnUID)...)

	if resp.Diagnostics.HasError() {
		return
	}

	mcr, err := r.client.MCRService.GetMCR(ctx, mcrUID)
	if err != nil {
		if apiErr, ok := err.(*megaport.ErrorResponse); ok && apiErr.Response != nil {
			if apiErr.Response.StatusCode == http.StatusNotFound {
				resp.Diagnostics.AddError(
					"Resource not found",
					fmt.Sprintf("MCR %s does not exist", mcrUID),
				)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error verifying resource during import",
			fmt.Sprintf("Could not read MCR %s: %s", mcrUID, err.Error()),
		)
		return
	}

	state := r.fromAPI(mcrUID, mcr)

	if state.AddOnUID.IsNull() {
		resp.Diagnostics.AddError(
			"Resource not found",
			fmt.Sprintf("No IPSec add-on found on MCR %s", mcrUID),
		)
		return
	}

	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// fromAPI maps an MCR API response to the resource model.
// Returns a model with AddOnUID null if no IPSec add-on is present.
func (r *mcrIpsecAddonResource) fromAPI(mcrID string, mcr *megaport.MCR) mcrIpsecAddonResourceModel {
	state := mcrIpsecAddonResourceModel{
		MCRID: types.StringValue(mcrID),
	}
	for _, addOn := range mcr.AddOns {
		if addOn == nil {
			continue
		}
		state.TunnelCount = types.Int64Value(int64(addOn.TunnelCount))
		state.AddOnUID = types.StringValue(addOn.AddOnUID)
		return state
	}
	state.TunnelCount = types.Int64Null()
	state.AddOnUID = types.StringNull()
	return state
}

// waitForMCRReady polls GetMCR until the MCR reaches a ready state or the
// configured waitForTime elapses.
func (r *mcrIpsecAddonResource) waitForMCRReady(ctx context.Context, mcrID string) error {
	toWait := waitForTime
	if toWait == 0 {
		toWait = 5 * time.Minute
	}

	ticker := time.NewTicker(30 * time.Second)
	timer := time.NewTimer(toWait)
	defer ticker.Stop()
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return fmt.Errorf("time expired waiting for MCR %s to become ready", mcrID)
		case <-ctx.Done():
			return fmt.Errorf("context expired waiting for MCR %s to become ready", mcrID)
		case <-ticker.C:
			mcr, err := r.client.MCRService.GetMCR(ctx, mcrID)
			if err != nil {
				return err
			}
			for _, s := range megaport.SERVICE_STATE_READY {
				if mcr.ProvisioningStatus == s {
					return nil
				}
			}
		}
	}
}
