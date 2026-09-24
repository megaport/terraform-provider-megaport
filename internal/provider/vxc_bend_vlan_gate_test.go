package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	megaport "github.com/megaport/megaportgo"
)

// cspConnections builds csp_connections with one entry per resource_name to
// connect_type pair.
func (b *vxcValueBuilder) cspConnections(t *testing.T, conns ...[2]string) tftypes.Value {
	t.Helper()
	listType, ok := b.objType.AttributeTypes["csp_connections"].(tftypes.List)
	if !ok {
		t.Fatal("csp_connections type is not tftypes.List")
	}
	elemType, ok := listType.ElementType.(tftypes.Object)
	if !ok {
		t.Fatal("csp_connections element type is not tftypes.Object")
	}
	elems := []tftypes.Value{}
	for _, c := range conns {
		attrs := nullValueMap(elemType)
		attrs["resource_name"] = tftypes.NewValue(tftypes.String, c[0])
		attrs["connect_type"] = tftypes.NewValue(tftypes.String, c[1])
		elems = append(elems, tftypes.NewValue(elemType, attrs))
	}
	return tftypes.NewValue(listType, elems)
}

// liveVXC builds a created VXC with the given csp_connections and B-End
// partner, either of which may be empty.
func (b *vxcValueBuilder) liveVXC(t *testing.T, aEnd, bEnd vxcEndSpec, conns [][2]string, partner string) tftypes.Value {
	t.Helper()
	aEnd.productUID, bEnd.productUID = "port-a", "port-b"
	var bEndPartner *tftypes.Value
	if partner != "" {
		p := b.partner(partner)
		bEndPartner = &p
	}
	set := map[string]tftypes.Value{"product_uid": tftypes.NewValue(tftypes.String, "vxc-uid")}
	if conns != nil {
		set["csp_connections"] = b.cspConnections(t, conns...)
	}
	return b.with(t, b.vxc(b.end(aEnd), b.end(bEnd), bEndPartner), set)
}

func bCSP(connectType string) [][2]string {
	return [][2]string{{"b_csp_connection", connectType}}
}

// bEndAt is a B-End on live VLAN 200.
func bEndAt(orderedVLAN int64, innerVLAN *int64) vxcEndSpec {
	return vxcEndSpec{orderedVLAN: int64p(orderedVLAN), vlan: int64p(200), innerVLAN: innerVLAN}
}

func TestVXCModifyPlan_BEndVLANGate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)
	live := bEndAt(200, nil)

	tests := []struct {
		name          string
		conns         [][2]string
		partner       string
		stateA, planA vxcEndSpec
		stateB, planB vxcEndSpec
		wantAttrs     []string
		wantType      string
	}{
		{name: "aws ordered_vlan change rejected", conns: bCSP("AWS"),
			stateB: live, planB: bEndAt(300, nil), wantAttrs: []string{"ordered_vlan"}, wantType: "AWS"},
		{name: "azure ordered_vlan change rejected", conns: bCSP("AZURE"),
			stateB: live, planB: bEndAt(300, nil), wantAttrs: []string{"ordered_vlan"}, wantType: "AZURE"},
		{name: "partner config stands in for empty csp_connections", partner: "transit",
			stateB: live, planB: bEndAt(300, nil), wantAttrs: []string{"ordered_vlan"}, wantType: "TRANSIT"},
		{name: "aws inner_vlan rejected", conns: bCSP("AWS"),
			stateB: live, planB: bEndAt(200, int64p(10)), wantAttrs: []string{"inner_vlan"}, wantType: "AWS"},
		{name: "aws untagging a tagged inner_vlan rejected", conns: bCSP("AWS"),
			stateB: bEndAt(200, int64p(10)), planB: bEndAt(200, int64p(-1)), wantAttrs: []string{"inner_vlan"}, wantType: "AWS"},
		{name: "outer and inner both reported", conns: bCSP("ORACLE"),
			stateB: live, planB: bEndAt(300, int64p(10)), wantAttrs: []string{"ordered_vlan", "inner_vlan"}, wantType: "ORACLE"},

		{name: "azure inner_vlan permitted", conns: bCSP("AZURE"), stateB: live, planB: bEndAt(200, int64p(10))},
		{name: "vrouter b-end permitted", conns: bCSP("VROUTER"), stateB: live, planB: bEndAt(300, nil)},
		{name: "a-end only csp connection permitted", conns: [][2]string{{"a_csp_connection", "AWS"}},
			stateB: live, planB: bEndAt(300, nil)},
		{name: "megaport b-end permitted", stateB: live, planB: bEndAt(300, nil)},
		{name: "vrouter partner permitted", partner: "vrouter", stateB: live, planB: bEndAt(300, nil)},
		{name: "auto-assign permitted", conns: bCSP("AWS"), stateB: live, planB: bEndAt(0, nil)},
		{name: "live vlan after import permitted", conns: bCSP("AWS"),
			stateB: vxcEndSpec{vlan: int64p(200)}, planB: live},
		{name: "unsent ordered_vlan already in state permitted", conns: bCSP("AWS"),
			stateB: bEndAt(300, nil), planB: bEndAt(300, nil)},
		{name: "unknown ordered_vlan permitted", conns: bCSP("AWS"),
			stateB: live, planB: vxcEndSpec{orderedVLANUnknown: true, vlan: int64p(200)}},
		{name: "unchanged inner_vlan permitted", conns: bCSP("AWS"),
			stateB: bEndAt(200, int64p(10)), planB: bEndAt(200, int64p(10))},
		{name: "untagged inner_vlan permitted", conns: bCSP("AWS"), stateB: live, planB: bEndAt(200, int64p(-1))},
		{name: "a-end change permitted on aws b-end", conns: bCSP("AWS"),
			stateA: vxcEndSpec{orderedVLAN: int64p(100), vlan: int64p(100)},
			planA:  vxcEndSpec{orderedVLAN: int64p(150), vlan: int64p(100), innerVLAN: int64p(10)},
			stateB: live, planB: live},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			state := b.liveVXC(t, tc.stateA, tc.stateB, tc.conns, tc.partner)
			plan := b.liveVXC(t, tc.planA, tc.planB, tc.conns, tc.partner)

			resp := fwresource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: b.schema, Raw: plan}}
			(&vxcResource{}).ModifyPlan(ctx, fwresource.ModifyPlanRequest{
				State: tfsdk.State{Schema: b.schema, Raw: state},
				Plan:  tfsdk.Plan{Schema: b.schema, Raw: plan},
			}, &resp)

			errs := resp.Diagnostics.Errors()
			if len(errs) != len(tc.wantAttrs) {
				t.Fatalf("expected %d errors, got %v", len(tc.wantAttrs), errs)
			}
			for i, attr := range tc.wantAttrs {
				withPath, ok := errs[i].(interface{ Path() path.Path })
				if !ok {
					t.Fatalf("error %d has no attribute path: %v", i, errs[i])
				}
				if want := path.Root("b_end").AtName(attr); !withPath.Path().Equal(want) {
					t.Errorf("error %d path = %s, want %s", i, withPath.Path(), want)
				}
				if !strings.Contains(errs[i].Detail(), "has connect type "+tc.wantType+",") {
					t.Errorf("error %d detail does not name %s: %q", i, tc.wantType, errs[i].Detail())
				}
			}
		})
	}
}

// Each case checks which VLAN fields reach the update request. The fake
// update returns 500, so only the request body matters.
func TestVXCUpdate_VLANSends(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)

	tests := []struct {
		name          string
		conns         [][2]string
		partner       string
		mve           bool
		stateA, planA vxcEndSpec
		stateB, planB vxcEndSpec
		want          map[string]float64
		wantAbsent    []string
		wantRejected  bool
	}{
		{
			name: "aws b-end change rejected before the request", conns: bCSP("AWS"),
			stateB: bEndAt(200, nil), planB: bEndAt(300, nil), wantRejected: true,
		},
		{
			name: "a-end sends on an aws b-end", conns: bCSP("AWS"), partner: "aws",
			stateA: vxcEndSpec{orderedVLAN: int64p(100), vlan: int64p(100)},
			planA:  vxcEndSpec{orderedVLAN: int64p(150), vlan: int64p(100), innerVLAN: int64p(10)},
			stateB: vxcEndSpec{vlan: int64p(200)}, planB: bEndAt(200, int64p(-1)),
			want:       map[string]float64{"aEndVlan": 150, "aEndInnerVlan": 10},
			wantAbsent: []string{"bEndVlan", "bEndInnerVlan"},
		},
		{
			name: "azure b-end sends inner vlan", conns: bCSP("AZURE"),
			stateB: bEndAt(200, int64p(10)), planB: bEndAt(200, int64p(20)),
			want:       map[string]float64{"bEndInnerVlan": 20},
			wantAbsent: []string{"bEndVlan"},
		},
		{
			name:   "megaport b-end sends vlan",
			stateB: bEndAt(200, nil), planB: bEndAt(300, nil),
			want: map[string]float64{"bEndVlan": 300},
		},
		{
			name: "vnic change resends the live vlan on both ends", mve: true,
			stateA: vxcEndSpec{orderedVLAN: int64p(100), vlan: int64p(100), vnicIndex: int64p(0)},
			planA:  vxcEndSpec{orderedVLAN: int64p(100), vlan: int64p(100), vnicIndex: int64p(1)},
			stateB: vxcEndSpec{orderedVLAN: int64p(200), vlan: int64p(200), vnicIndex: int64p(0)},
			planB:  vxcEndSpec{orderedVLAN: int64p(200), vlan: int64p(200), vnicIndex: int64p(1)},
			want:   map[string]float64{"aEndVlan": 100, "bEndVlan": 200, "aVnicIndex": 1, "bVnicIndex": 1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ps := newPreflightServer(t, nil)
			if tc.mve {
				ps.productTypeOf = map[string]string{"port-a": megaport.PRODUCT_MVE, "port-b": megaport.PRODUCT_MVE}
			}
			state := b.liveVXC(t, tc.stateA, tc.stateB, tc.conns, tc.partner)
			plan := b.liveVXC(t, tc.planA, tc.planB, tc.conns, tc.partner)

			resp := fwresource.UpdateResponse{State: tfsdk.State{Schema: b.schema, Raw: state}}
			ps.resource(t).Update(ctx, fwresource.UpdateRequest{
				Plan:  tfsdk.Plan{Schema: b.schema, Raw: plan},
				State: tfsdk.State{Schema: b.schema, Raw: state},
			}, &resp)

			if tc.wantRejected {
				if len(ps.updateBodies) != 0 || !resp.Diagnostics.HasError() {
					t.Fatalf("expected a rejection and no request, got %d requests: %v", len(ps.updateBodies), resp.Diagnostics.Errors())
				}
				return
			}
			if len(ps.updateBodies) != 1 {
				t.Fatalf("expected one update request, got %d: %v", len(ps.updateBodies), resp.Diagnostics.Errors())
			}
			body := ps.updateBodies[0]
			for field, want := range tc.want {
				if body[field] != want {
					t.Errorf("expected %s %v, got %v", field, want, body[field])
				}
			}
			for _, field := range tc.wantAbsent {
				if v, ok := body[field]; ok {
					t.Errorf("expected no %s, got %v", field, v)
				}
			}
		})
	}
}
