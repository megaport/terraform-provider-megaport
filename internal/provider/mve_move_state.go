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

	// The V2 schema is the V1 schema minus the removed legacy fields and the
	// removed vnics.vlan attribute, so the V2 schema doubles as the source
	// schema: the framework decodes the V1 raw state against it, silently
	// dropping attributes that no longer exist. vendor_config is unchanged, so
	// it carries over unmodified.
	return []resource.StateMover{
		{
			SourceSchema: &schemaResp.Schema,
			StateMover:   moveStateMVE,
		},
	}
}

// moveStateMVE migrates a V1 megaport_mve state to V2. The removed legacy
// fields and vnics.vlan are already dropped by the SourceSchema decode; all
// remaining attributes carry over unchanged.
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
