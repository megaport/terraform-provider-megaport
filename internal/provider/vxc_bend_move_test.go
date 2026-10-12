package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// bEndMoveCases change the B-End from port-b to port-c. Update never moves a
// cloud B-End, and state's connect type marks one that has no partner config.
var bEndMoveCases = []struct {
	name     string
	conns    [][2]string
	partner  string
	wantMove bool
}{
	{name: "alibaba b-end", conns: bCSP("ALIBABA")},
	{name: "aws b-end without partner config", conns: bCSP("AWS")},
	{name: "aws partner config without csp_connections", partner: "aws"},
	{name: "transit b-end", conns: bCSP("TRANSIT"), partner: "transit", wantMove: true},
	{name: "transit b-end without partner config", conns: bCSP("TRANSIT"), wantMove: true},
	{name: "mcr b-end", conns: bCSP("VROUTER"), wantMove: true},
	{name: "megaport b-end", wantMove: true},
}

// The fake update returns 500, so only the request body matters.
func TestVXCUpdate_BEndMove(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)

	for _, tc := range bEndMoveCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ps := newPreflightServer(t, nil)
			state := b.liveVXC(t, vxcEndSpec{}, bEndAt(200, nil), tc.conns, tc.partner)
			moved := bEndAt(200, nil)
			moved.productUID = "port-c"
			plan := b.with(t, b.liveVXC(t, vxcEndSpec{}, moved, tc.conns, tc.partner),
				map[string]tftypes.Value{"rate_limit": tftypes.NewValue(tftypes.Number, 2000)})

			resp := fwresource.UpdateResponse{State: tfsdk.State{Schema: b.schema, Raw: state}}
			ps.resource(t).Update(ctx, fwresource.UpdateRequest{
				Plan:  tfsdk.Plan{Schema: b.schema, Raw: plan},
				State: tfsdk.State{Schema: b.schema, Raw: state},
			}, &resp)

			if len(ps.updateBodies) != 1 {
				t.Fatalf("expected one update request, got %d: %v", len(ps.updateBodies), resp.Diagnostics.Errors())
			}
			body := ps.updateBodies[0]
			if body["rateLimit"] != float64(2000) {
				t.Errorf("expected rateLimit 2000, got %v", body["rateLimit"])
			}
			uid, sent := body["bEndProductUid"]
			if sent != tc.wantMove || sent && uid != "port-c" {
				t.Errorf("expected move %v, got bEndProductUid %v", tc.wantMove, uid)
			}
		})
	}
}

func TestVXCModifyPlan_BEndProductUIDPin(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)

	plannedUID := func(t *testing.T, state, plan tftypes.Value) string {
		t.Helper()
		resp := fwresource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: b.schema, Raw: plan}}
		(&vxcResource{}).ModifyPlan(ctx, fwresource.ModifyPlanRequest{
			State: tfsdk.State{Schema: b.schema, Raw: state},
			Plan:  tfsdk.Plan{Schema: b.schema, Raw: plan},
		}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ModifyPlan failed: %v", resp.Diagnostics.Errors())
		}
		var got types.String
		resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("b_end").AtName("requested_product_uid"), &got)...)
		return got.ValueString()
	}

	for _, tc := range bEndMoveCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			state := b.liveVXC(t, vxcEndSpec{}, bEndAt(200, nil), tc.conns, tc.partner)
			moved := bEndAt(200, nil)
			moved.productUID = "port-c"
			plan := b.liveVXC(t, vxcEndSpec{}, moved, tc.conns, tc.partner)

			want := "port-b"
			if tc.wantMove {
				want = "port-c"
			}
			if got := plannedUID(t, state, plan); got != want {
				t.Errorf("expected b_end.requested_product_uid %q, got %q", want, got)
			}
		})
	}

	// A partner unknown at plan time may still resolve to a cloud partner.
	t.Run("unknown partner", func(t *testing.T) {
		t.Parallel()
		state := b.liveVXC(t, vxcEndSpec{}, bEndAt(200, nil), nil, "aws")
		moved := bEndAt(200, nil)
		moved.productUID = "port-c"
		partner := nullValueMap(b.partnerTyp)
		partner["partner"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
		plan := b.with(t, b.liveVXC(t, vxcEndSpec{}, moved, nil, ""),
			map[string]tftypes.Value{"b_end_partner_config": tftypes.NewValue(b.partnerTyp, partner)})

		if got := plannedUID(t, state, plan); got != "port-b" {
			t.Errorf("expected b_end.requested_product_uid %q, got %q", "port-b", got)
		}
	})
}
