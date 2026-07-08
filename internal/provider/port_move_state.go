package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// MoveState implements resource.ResourceWithMoveState to support automatic
// V1-to-V2 state migration when users move from megaport/megaport (v1) to v2.
func (r *portResource) MoveState(ctx context.Context) []resource.StateMover {
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	// The V2 schema is the V1 schema minus the removed legacy attributes, so
	// the V2 schema doubles as the source schema: the framework decodes the V1
	// raw state against it, silently dropping attributes that no longer exist.
	return []resource.StateMover{
		{
			SourceSchema: &schemaResp.Schema,
			StateMover:   moveStatePort,
		},
	}
}

// moveStatePort migrates a V1 megaport_port state to V2. The removed legacy
// fields are already dropped by the SourceSchema decode; all remaining
// attributes carry over unchanged.
func moveStatePort(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	if req.SourceProviderAddress != "registry.terraform.io/megaport/megaport" || req.SourceTypeName != "megaport_port" {
		return
	}

	if req.SourceState == nil {
		resp.Diagnostics.AddError(
			"Unable to migrate V1 state",
			"The source megaport_port state could not be decoded against the V2 schema.",
		)
		return
	}

	var model singlePortResourceModel
	resp.Diagnostics.Append(req.SourceState.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &model)...)
}

// MoveState implements resource.ResourceWithMoveState to support automatic
// V1-to-V2 state migration when users move from megaport/megaport (v1) to v2.
func (r *lagPortResource) MoveState(ctx context.Context) []resource.StateMover {
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	return []resource.StateMover{
		{
			SourceSchema: &schemaResp.Schema,
			StateMover:   moveStateLagPort,
		},
	}
}

// moveStateLagPort migrates a V1 megaport_lag_port state to V2. The removed
// legacy fields are already dropped by the SourceSchema decode; all remaining
// attributes carry over unchanged.
func moveStateLagPort(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	if req.SourceProviderAddress != "registry.terraform.io/megaport/megaport" || req.SourceTypeName != "megaport_lag_port" {
		return
	}

	if req.SourceState == nil {
		resp.Diagnostics.AddError(
			"Unable to migrate V1 state",
			"The source megaport_lag_port state could not be decoded against the V2 schema.",
		)
		return
	}

	var model lagPortResourceModel
	resp.Diagnostics.Append(req.SourceState.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &model)...)
}
