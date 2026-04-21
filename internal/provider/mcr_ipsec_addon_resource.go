package provider

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
}

// Metadata returns the resource type name.
func (r *mcrIpsecAddonResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcr_ipsec_addon"
}

// Schema defines the schema for the resource.
func (r *mcrIpsecAddonResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an IPSec tunnel add-on for a Megaport Cloud Router (MCR). This resource allows you to attach, update, and remove IPSec tunnel packs on an existing MCR.",
		Attributes: map[string]schema.Attribute{
			"mcr_id": schema.StringAttribute{
				Description: "The UID of the MCR to attach the IPSec add-on to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tunnel_count": schema.Int64Attribute{
				Description: "The number of IPSec tunnels. Valid values are 10, 20, or 30.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(10, 20, 30),
				},
			},
			"add_on_uid": schema.StringAttribute{
				Description: "The UID of the IPSec add-on, assigned by the API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *mcrIpsecAddonResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the resource and sets the initial Terraform state.
func (r *mcrIpsecAddonResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mcrIpsecAddonResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mcrID := plan.MCRID.ValueString()
	tunnelCount := int(plan.TunnelCount.ValueInt64())

	tflog.Info(ctx, "Creating MCR IPSec add-on", map[string]interface{}{
		"mcr_id":       mcrID,
		"tunnel_count": tunnelCount,
	})

	// Create the IPSec add-on
	addOnReq := megaport.MCRAddOnRequest{
		AddOn: &megaport.MCRAddOnIPsecConfig{
			AddOnType:   megaport.AddOnTypeIPsec,
			TunnelCount: tunnelCount,
		},
	}

	err := r.client.MCRService.UpdateMCRWithAddOn(ctx, mcrID, addOnReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating MCR IPSec add-on",
			fmt.Sprintf("Could not create IPSec add-on for MCR %s: %s", mcrID, err.Error()),
		)
		return
	}

	// Wait for the MCR to reach a ready state
	if err := r.waitForMCRReady(ctx, mcrID); err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for MCR to be ready",
			fmt.Sprintf("MCR %s did not reach a ready state after adding IPSec add-on: %s", mcrID, err.Error()),
		)
		return
	}

	// Read the MCR to get the add-on UID
	mcr, err := r.client.MCRService.GetMCR(ctx, mcrID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading MCR after creating IPSec add-on",
			fmt.Sprintf("Could not read MCR %s: %s", mcrID, err.Error()),
		)
		return
	}

	// Find the IPSec add-on
	addOn := r.findIPsecAddOn(mcr, "")
	if addOn == nil {
		resp.Diagnostics.AddError(
			"Error reading MCR IPSec add-on",
			fmt.Sprintf("IPSec add-on was created but could not be found on MCR %s", mcrID),
		)
		return
	}

	plan.AddOnUID = types.StringValue(addOn.AddOnUID)
	plan.TunnelCount = types.Int64Value(int64(addOn.TunnelCount))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *mcrIpsecAddonResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mcrIpsecAddonResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mcrID := state.MCRID.ValueString()

	mcr, err := r.client.MCRService.GetMCR(ctx, mcrID)
	if err != nil {
		if isMCRNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading MCR",
			fmt.Sprintf("Could not read MCR %s: %s", mcrID, err.Error()),
		)
		return
	}

	// If the MCR has been decommissioned, treat the add-on as deleted.
	if mcr.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED {
		resp.State.RemoveResource(ctx)
		return
	}

	// Find the add-on by UID
	addOn := r.findIPsecAddOn(mcr, state.AddOnUID.ValueString())
	if addOn == nil {
		// Add-on was deleted outside Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	state.TunnelCount = types.Int64Value(int64(addOn.TunnelCount))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *mcrIpsecAddonResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state mcrIpsecAddonResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mcrID := state.MCRID.ValueString()
	addOnUID := state.AddOnUID.ValueString()
	tunnelCount := int(plan.TunnelCount.ValueInt64())

	tflog.Info(ctx, "Updating MCR IPSec add-on", map[string]interface{}{
		"mcr_id":       mcrID,
		"add_on_uid":   addOnUID,
		"tunnel_count": tunnelCount,
	})

	err := r.client.MCRService.UpdateMCRIPsecAddOn(ctx, mcrID, addOnUID, tunnelCount)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating MCR IPSec add-on",
			fmt.Sprintf("Could not update IPSec add-on %s for MCR %s: %s", addOnUID, mcrID, err.Error()),
		)
		return
	}

	// Wait for the MCR to reach a ready state
	if err := r.waitForMCRReady(ctx, mcrID); err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for MCR to be ready",
			fmt.Sprintf("MCR %s did not reach a ready state after updating IPSec add-on: %s", mcrID, err.Error()),
		)
		return
	}

	// Read the MCR to confirm the update
	mcr, err := r.client.MCRService.GetMCR(ctx, mcrID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading MCR after updating IPSec add-on",
			fmt.Sprintf("Could not read MCR %s: %s", mcrID, err.Error()),
		)
		return
	}

	addOn := r.findIPsecAddOn(mcr, addOnUID)
	if addOn == nil {
		resp.Diagnostics.AddError(
			"Error reading MCR IPSec add-on",
			fmt.Sprintf("IPSec add-on %s could not be found on MCR %s after update", addOnUID, mcrID),
		)
		return
	}

	state.TunnelCount = types.Int64Value(int64(addOn.TunnelCount))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mcrIpsecAddonResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mcrIpsecAddonResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mcrID := state.MCRID.ValueString()
	addOnUID := state.AddOnUID.ValueString()

	tflog.Info(ctx, "Deleting MCR IPSec add-on", map[string]interface{}{
		"mcr_id":     mcrID,
		"add_on_uid": addOnUID,
	})

	// Setting tunnel count to 0 disables/removes the add-on
	err := r.client.MCRService.UpdateMCRIPsecAddOn(ctx, mcrID, addOnUID, 0)
	if err != nil {
		if isMCRNotFoundError(err) {
			// MCR or add-on already deleted
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting MCR IPSec add-on",
			fmt.Sprintf("Could not delete IPSec add-on %s for MCR %s: %s", addOnUID, mcrID, err.Error()),
		)
		return
	}

	// Wait for the MCR to return to a ready state so follow-on operations
	// (e.g., deleting the MCR itself in the same destroy) don't race.
	if err := r.waitForMCRReady(ctx, mcrID); err != nil {
		if isMCRNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError(
			"Error waiting for MCR after deleting IPSec add-on",
			fmt.Sprintf("IPSec add-on %s was disabled for MCR %s, but the MCR did not return to a ready state: %s", addOnUID, mcrID, err.Error()),
		)
	}
}

// ImportState imports the resource state.
func (r *mcrIpsecAddonResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID (format: mcr_uid:add_on_uid)
	mcrUID, addOnUID, err := parseImportIDStrings(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Error parsing import ID: %s\n\nExpected format: mcr_uid:add_on_uid\nExample: 12345678-1234-1234-1234-123456789012:abcdef12-3456-7890-abcd-ef1234567890", err.Error()),
		)
		return
	}

	// Set the parsed values in the state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mcr_id"), mcrUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("add_on_uid"), addOnUID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Verify the resource exists
	mcr, err := r.client.MCRService.GetMCR(ctx, mcrUID)
	if err != nil {
		if isMCRNotFoundError(err) {
			resp.Diagnostics.AddError(
				"Resource not found",
				fmt.Sprintf("MCR %s does not exist", mcrUID),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error verifying resource during import",
			fmt.Sprintf("Could not read MCR %s: %s", mcrUID, err.Error()),
		)
		return
	}

	addOn := r.findIPsecAddOn(mcr, addOnUID)
	if addOn == nil {
		resp.Diagnostics.AddError(
			"Resource not found",
			fmt.Sprintf("IPSec add-on %s does not exist on MCR %s", addOnUID, mcrUID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tunnel_count"), int64(addOn.TunnelCount))...)
}

// waitForMCRReady polls the MCR until it reaches a ready provisioning state.
func (r *mcrIpsecAddonResource) waitForMCRReady(ctx context.Context, mcrID string) error {
	toWait := waitForTime
	if toWait == 0 {
		toWait = 10 * time.Minute
	}

	// Check immediately before entering the poll loop.
	mcr, err := r.client.MCRService.GetMCR(ctx, mcrID)
	if err != nil {
		if isMCRNotFoundError(err) {
			return fmt.Errorf("MCR %s has been deleted", mcrID)
		}
		return fmt.Errorf("error polling MCR %s: %w", mcrID, err)
	}
	if mcr.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED {
		return fmt.Errorf("MCR %s has been decommissioned", mcrID)
	}
	if slices.Contains(megaport.SERVICE_STATE_READY, mcr.ProvisioningStatus) {
		return nil
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	timeout := time.After(toWait)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timed out waiting for MCR %s to reach a ready state", mcrID)
		case <-ticker.C:
			mcr, err := r.client.MCRService.GetMCR(ctx, mcrID)
			if err != nil {
				if isMCRNotFoundError(err) {
					return fmt.Errorf("MCR %s has been deleted", mcrID)
				}
				return fmt.Errorf("error polling MCR %s: %w", mcrID, err)
			}
			if mcr.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED {
				return fmt.Errorf("MCR %s has been decommissioned", mcrID)
			}
			if slices.Contains(megaport.SERVICE_STATE_READY, mcr.ProvisioningStatus) {
				return nil
			}
			tflog.Debug(ctx, "MCR not yet ready, continuing to poll", map[string]interface{}{
				"mcr_id": mcrID,
				"status": mcr.ProvisioningStatus,
			})
		}
	}
}

// findIPsecAddOn finds an IPSec add-on in the MCR's add-ons list.
// If addOnUID is empty, it returns the first IPSec add-on found.
// If addOnUID is provided, it returns the add-on with that specific UID.
func (r *mcrIpsecAddonResource) findIPsecAddOn(mcr *megaport.MCR, addOnUID string) *megaport.MCRAddOnIPsecConfig {
	for _, addOn := range mcr.AddOns {
		if addOn.AddOnType != megaport.AddOnTypeIPsec {
			continue
		}
		if addOnUID == "" || addOn.AddOnUID == addOnUID {
			return addOn
		}
	}
	return nil
}

// isMCRNotFoundError checks if an error indicates the MCR was not found.
// The API can return either HTTP 404 or HTTP 400 with "Could not find a service with UID".
func isMCRNotFoundError(err error) bool {
	apiErr, ok := err.(*megaport.ErrorResponse)
	if !ok || apiErr.Response == nil {
		return false
	}
	if apiErr.Response.StatusCode == http.StatusNotFound {
		return true
	}
	if apiErr.Response.StatusCode == http.StatusBadRequest &&
		strings.Contains(apiErr.Message, "Could not find a service with UID") {
		return true
	}
	return false
}

// parseImportIDStrings parses an import ID in the format "part1:part2" and returns both parts as strings.
func parseImportIDStrings(importID string) (string, string, error) {
	parts := strings.Split(importID, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid import ID format, expected 'mcr_uid:add_on_uid' (exactly one colon), got '%s'", importID)
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid import ID format, both mcr_uid and add_on_uid must be non-empty, got '%s'", importID)
	}
	return parts[0], parts[1], nil
}
