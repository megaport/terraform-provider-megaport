package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// MoveState implements resource.ResourceWithMoveState to support automatic
// V1-to-V2 state migration when users move from megaport/megaport (v1) to v2.
func (r *mveResource) MoveState(ctx context.Context) []resource.StateMover {
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	// The V2 schema is the source schema. The framework decodes the V1 raw
	// state against it with IgnoreUndefinedAttributes, silently dropping the
	// removed legacy fields, the old union vendor_config block, and the removed
	// vnics.vlan attribute. The per-vendor config blocks are absent from V1
	// state, so they decode to null and are reconciled from config on the first
	// apply after the move.
	return []resource.StateMover{
		{
			SourceSchema: &schemaResp.Schema,
			StateMover:   moveStateMVE,
		},
	}
}

// moveStateMVE migrates a V1 megaport_mve state to V2.
func moveStateMVE(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	if req.SourceProviderAddress != "registry.terraform.io/megaport/megaport" || req.SourceTypeName != "megaport_mve" {
		return
	}

	if req.SourceState == nil {
		resp.Diagnostics.AddError(
			"Unable to migrate V1 state",
			"The source megaport_mve state could not be decoded against the V2 schema.",
		)
		return
	}

	var model mveResourceModel
	resp.Diagnostics.Append(req.SourceState.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &model)...)
}
