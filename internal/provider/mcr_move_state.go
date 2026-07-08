package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func (r *mcrResource) MoveState(ctx context.Context) []resource.StateMover {
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	return []resource.StateMover{
		{
			SourceSchema: &schemaResp.Schema,
			StateMover:   moveStateMCR,
		},
	}
}

func moveStateMCR(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	if req.SourceProviderAddress != "registry.terraform.io/megaport/megaport" || req.SourceTypeName != "megaport_mcr" {
		return
	}

	if req.SourceState == nil {
		resp.Diagnostics.AddError(
			"Unable to migrate V1 state",
			"The source megaport_mcr state could not be decoded against the V2 schema.",
		)
		return
	}

	var model mcrResourceModel
	resp.Diagnostics.Append(req.SourceState.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &model)...)
}
