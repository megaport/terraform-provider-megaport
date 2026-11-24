package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The fake update returns 500, so only the request body matters.
func TestVXCUpdate_Shutdown(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)

	tests := []struct {
		name        string
		state, plan bool
	}{
		{name: "shutdown enabled", plan: true},
		{name: "shutdown disabled", state: true},
		{name: "shutdown unchanged"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ps := newPreflightServer(t, nil)
			// with mutates its input, so each value gets its own base.
			state := b.with(t, b.liveVXC(t, vxcEndSpec{}, bEndAt(200, nil), nil, ""), map[string]tftypes.Value{"shutdown": tftypes.NewValue(tftypes.Bool, tc.state)})
			plan := b.with(t, b.liveVXC(t, vxcEndSpec{}, bEndAt(200, nil), nil, ""), map[string]tftypes.Value{"shutdown": tftypes.NewValue(tftypes.Bool, tc.plan)})

			resp := fwresource.UpdateResponse{State: tfsdk.State{Schema: b.schema, Raw: state}}
			ps.resource(t).Update(ctx, fwresource.UpdateRequest{
				Plan:  tfsdk.Plan{Schema: b.schema, Raw: plan},
				State: tfsdk.State{Schema: b.schema, Raw: state},
			}, &resp)

			if len(ps.updateBodies) != 1 {
				t.Fatalf("expected one update request, got %d: %v", len(ps.updateBodies), resp.Diagnostics.Errors())
			}
			got, sent := ps.updateBodies[0]["shutdown"]
			if sent != (tc.state != tc.plan) || sent && got != tc.plan {
				t.Errorf("expected shutdown %v, got %v (sent %v)", tc.plan, got, sent)
			}
		})
	}
}
