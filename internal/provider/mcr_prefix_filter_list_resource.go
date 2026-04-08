package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	megaport "github.com/megaport/megaportgo"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &mcrPrefixFilterListResource{}
	_ resource.ResourceWithConfigure   = &mcrPrefixFilterListResource{}
	_ resource.ResourceWithImportState = &mcrPrefixFilterListResource{}
)

// NewMCRPrefixFilterListResource is a helper function to simplify the provider implementation.
func NewMCRPrefixFilterListResource() resource.Resource {
	return &mcrPrefixFilterListResource{}
}

// mcrPrefixFilterListResource defines the resource implementation.
type mcrPrefixFilterListResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *mcrPrefixFilterListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcr_prefix_filter_list"
}

// Schema defines the schema for the resource.
func (r *mcrPrefixFilterListResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = mcrPrefixFilterListResourceSchema()
}

// Configure adds the provider configured client to the resource.
func (r *mcrPrefixFilterListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := configureMegaportResource(req, resp)
	if !ok {
		return
	}
	r.client = providerData.client
}

// Create creates the resource and sets the initial Terraform state.
func (r *mcrPrefixFilterListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mcrPrefixFilterListResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert plan to API model
	apiPrefixFilterList, convertDiags := plan.planToAPI(ctx)
	resp.Diagnostics.Append(convertDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the prefix filter list via API
	createReq := &megaport.CreateMCRPrefixFilterListRequest{
		MCRID:            plan.MCRID.ValueString(),
		PrefixFilterList: *apiPrefixFilterList,
	}

	createResp, err := r.client.MCRService.CreatePrefixFilterList(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating MCR prefix filter list",
			fmt.Sprintf("Could not create prefix filter list for MCR %s: %s",
				plan.MCRID.ValueString(), err.Error()),
		)
		return
	}

	// Retrieve the created prefix filter list to get all attributes
	createdList, err := r.client.MCRService.GetMCRPrefixFilterList(ctx,
		plan.MCRID.ValueString(), createResp.PrefixFilterListID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading created MCR prefix filter list",
			fmt.Sprintf("Could not read created prefix filter list %d for MCR %s: %s",
				createResp.PrefixFilterListID, plan.MCRID.ValueString(), err.Error()),
		)
		return
	}

	// Extract planned entries for comparison during API response processing
	var plannedEntries []*mcrPrefixFilterListEntryResourceModel
	if !plan.Entries.IsNull() && !plan.Entries.IsUnknown() {
		entryDiags := plan.Entries.ElementsAs(ctx, &plannedEntries, false)
		resp.Diagnostics.Append(entryDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Update the model with API response, using plan for exact match comparison
	var state mcrPrefixFilterListResourceModel
	state.MCRID = plan.MCRID // Preserve the MCR ID from the plan
	fromAPIDiags := state.fromAPIWithPlan(ctx, createdList, plannedEntries)
	resp.Diagnostics.Append(fromAPIDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set last updated timestamp
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *mcrPrefixFilterListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mcrPrefixFilterListResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract current state entries to use for exact match comparison
	// This preserves exact match configurations during refresh
	var stateEntries []*mcrPrefixFilterListEntryResourceModel
	if !state.Entries.IsNull() && !state.Entries.IsUnknown() {
		entryDiags := state.Entries.ElementsAs(ctx, &stateEntries, false)
		resp.Diagnostics.Append(entryDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Get the prefix filter list from API
	prefixFilterList, err := r.client.MCRService.GetMCRPrefixFilterList(ctx,
		state.MCRID.ValueString(), int(state.ID.ValueInt64()))
	if err != nil {
		// Check if the resource was deleted
		if apiErr, ok := err.(*megaport.ErrorResponse); ok {
			if apiErr.Response.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error reading MCR prefix filter list",
			fmt.Sprintf("Could not read prefix filter list %d for MCR %s: %s",
				state.ID.ValueInt64(), state.MCRID.ValueString(), err.Error()),
		)
		return
	}

	// Update state from API response, using existing state for exact match comparison
	// Pass stateEntries for normal read operations to enable exact match normalization, or nil for import to return raw API values
	fromAPIDiags := state.fromAPIWithPlan(ctx, prefixFilterList, stateEntries)
	resp.Diagnostics.Append(fromAPIDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *mcrPrefixFilterListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state mcrPrefixFilterListResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert plan to API model
	apiPrefixFilterList, convertDiags := plan.planToAPI(ctx)
	resp.Diagnostics.Append(convertDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the prefix filter list via API
	_, err := r.client.MCRService.ModifyMCRPrefixFilterList(ctx,
		state.MCRID.ValueString(), int(state.ID.ValueInt64()), apiPrefixFilterList)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating MCR prefix filter list",
			fmt.Sprintf("Could not update prefix filter list %d for MCR %s: %s",
				state.ID.ValueInt64(), state.MCRID.ValueString(), err.Error()),
		)
		return
	}

	// Retrieve the updated prefix filter list
	updatedList, err := r.client.MCRService.GetMCRPrefixFilterList(ctx,
		state.MCRID.ValueString(), int(state.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading updated MCR prefix filter list",
			fmt.Sprintf("Could not read updated prefix filter list %d for MCR %s: %s",
				state.ID.ValueInt64(), state.MCRID.ValueString(), err.Error()),
		)
		return
	}

	// Extract planned entries for comparison during API response processing
	var plannedEntries []*mcrPrefixFilterListEntryResourceModel
	if !plan.Entries.IsNull() && !plan.Entries.IsUnknown() {
		entryDiags := plan.Entries.ElementsAs(ctx, &plannedEntries, false)
		resp.Diagnostics.Append(entryDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Update state from API response, using plan for exact match comparison
	fromAPIDiags := state.fromAPIWithPlan(ctx, updatedList, plannedEntries)
	resp.Diagnostics.Append(fromAPIDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set last updated timestamp
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mcrPrefixFilterListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mcrPrefixFilterListResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the prefix filter list via API, retrying on 409 Conflict (e.g. when a
	// VXC with a BGP connection referencing this list is still being deprovisioned).
	maxRetries := 12
	retryInterval := 10 * time.Second
	for attempt := range maxRetries {
		_, err := r.client.MCRService.DeleteMCRPrefixFilterList(ctx,
			state.MCRID.ValueString(), int(state.ID.ValueInt64()))
		if err == nil {
			return
		}
		if apiErr, ok := err.(*megaport.ErrorResponse); ok && apiErr.Response != nil {
			if apiErr.Response.StatusCode == http.StatusNotFound {
				// Resource was already deleted, which is fine
				return
			}
			if apiErr.Response.StatusCode == http.StatusConflict && attempt < maxRetries-1 {
				tflog.Debug(ctx, "Prefix filter list still associated with a BGP connection, retrying delete",
					map[string]interface{}{
						"prefix_list_id": state.ID.ValueInt64(),
						"mcr_id":         state.MCRID.ValueString(),
						"attempt":        attempt + 1,
					})
				time.Sleep(retryInterval)
				continue
			}
		}
		resp.Diagnostics.AddError(
			"Error deleting MCR prefix filter list",
			fmt.Sprintf("Could not delete prefix filter list %d for MCR %s: %s",
				state.ID.ValueInt64(), state.MCRID.ValueString(), err.Error()),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *mcrPrefixFilterListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID (format: mcr_uid:prefix_list_id)
	mcrUID, prefixListID, err := parseImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Error parsing import ID: %s\n\nExpected format: mcr_uid:prefix_list_id\nExample: 12345678-1234-1234-1234-123456789012:123", err.Error()),
		)
		return
	}

	// Set the parsed values in the state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mcr_id"), mcrUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), prefixListID)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Verify the resource exists by attempting to read it
	prefixFilterList, err := r.client.MCRService.GetMCRPrefixFilterList(ctx, mcrUID, int(prefixListID))
	if err != nil {
		if apiErr, ok := err.(*megaport.ErrorResponse); ok {
			if apiErr.Response.StatusCode == http.StatusNotFound {
				resp.Diagnostics.AddError(
					"Resource not found",
					fmt.Sprintf("Prefix filter list %d does not exist for MCR %s", prefixListID, mcrUID),
				)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error verifying resource during import",
			fmt.Sprintf("Could not verify prefix filter list %d for MCR %s: %s", prefixListID, mcrUID, err.Error()),
		)
		return
	}

	// Set the imported resource state
	var state mcrPrefixFilterListResourceModel
	state.MCRID = types.StringValue(mcrUID)
	fromAPIDiags := state.fromAPI(ctx, prefixFilterList)
	resp.Diagnostics.Append(fromAPIDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set last updated timestamp for imported resource
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Save the imported state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

