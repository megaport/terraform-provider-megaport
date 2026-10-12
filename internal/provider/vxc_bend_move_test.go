package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// bEndMoveCases change the B-End from port-b to port-c. The API never moves a
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

	for _, tc := range bEndMoveCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			state := b.liveVXC(t, vxcEndSpec{}, bEndAt(200, nil), tc.conns, tc.partner)
			moved := bEndAt(200, nil)
			moved.productUID = "port-c"
			plan := b.liveVXC(t, vxcEndSpec{}, moved, tc.conns, tc.partner)

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
			want := "port-b"
			if tc.wantMove {
				want = "port-c"
			}
			if got.ValueString() != want || got.IsNull() {
				t.Errorf("expected b_end.requested_product_uid %q, got %s", want, got)
			}
		})
	}
}

// A taint orders the VXC again from config, which cannot rebuild a cloud
// B-End that has no partner config.
func TestVXCModifyPlan_BEndVLANGateTaintAdvice(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)

	tests := []struct {
		name          string
		conns         [][2]string
		partner       string
		stateB, planB vxcEndSpec
		wantTaint     bool
	}{
		{name: "error without partner config", conns: bCSP("ALIBABA"),
			stateB: bEndAt(200, nil), planB: bEndAt(300, nil)},
		{name: "error with partner config", conns: bCSP("AWS"), partner: "aws",
			stateB: bEndAt(200, nil), planB: bEndAt(300, nil), wantTaint: true},
		{name: "warning without partner config", conns: bCSP("AWS"),
			stateB: bEndAt(300, nil), planB: bEndAt(300, nil)},
		{name: "warning with partner config", conns: bCSP("AWS"), partner: "aws",
			stateB: bEndAt(300, nil), planB: bEndAt(300, nil), wantTaint: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			state := b.liveVXC(t, vxcEndSpec{}, tc.stateB, tc.conns, tc.partner)
			plan := b.liveVXC(t, vxcEndSpec{}, tc.planB, tc.conns, tc.partner)

			resp := fwresource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: b.schema, Raw: plan}}
			(&vxcResource{}).ModifyPlan(ctx, fwresource.ModifyPlanRequest{
				State: tfsdk.State{Schema: b.schema, Raw: state},
				Plan:  tfsdk.Plan{Schema: b.schema, Raw: plan},
			}, &resp)

			if len(resp.Diagnostics) != 1 {
				t.Fatalf("expected one diagnostic, got %v", resp.Diagnostics)
			}
			detail := resp.Diagnostics[0].Detail()
			if got := strings.Contains(detail, "terraform taint"); got != tc.wantTaint {
				t.Errorf("taint advice = %v, want %v: %q", got, tc.wantTaint, detail)
			}
			if !strings.HasSuffix(detail, ".") || strings.HasSuffix(detail, "..") {
				t.Errorf("detail does not end in one period: %q", detail)
			}
		})
	}
}
