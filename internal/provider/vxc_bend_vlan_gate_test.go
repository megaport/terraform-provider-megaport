package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// gateEnd is the subset of a_end / b_end the B-End VLAN gate reads.
type gateEnd struct {
	orderedVLAN tftypes.Value
	vlan        tftypes.Value
	innerVLAN   tftypes.Value
}

func num(v int64) tftypes.Value { return tftypes.NewValue(tftypes.Number, v) }

var unknownNum = tftypes.NewValue(tftypes.Number, tftypes.UnknownValue)

func (b *vxcValueBuilder) gateEnd(uid string, e gateEnd) tftypes.Value {
	attrs := nullValueMap(b.endType)
	attrs["requested_product_uid"] = tftypes.NewValue(tftypes.String, uid)
	for name, v := range map[string]tftypes.Value{"ordered_vlan": e.orderedVLAN, "vlan": e.vlan, "inner_vlan": e.innerVLAN} {
		if v.Type() != nil {
			attrs[name] = v
		}
	}
	return tftypes.NewValue(b.endType, attrs)
}

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

func (b *vxcValueBuilder) partner(name string) tftypes.Value {
	attrs := nullValueMap(b.partnerTyp)
	attrs["partner"] = tftypes.NewValue(tftypes.String, name)
	return tftypes.NewValue(b.partnerTyp, attrs)
}

// gateVXC builds a VXC value with the given ends, csp_connections, and B-End
// partner config. A zero-value conns or partner stays null.
func (b *vxcValueBuilder) gateVXC(aEnd, bEnd, conns, partner tftypes.Value) tftypes.Value {
	attrs := nullValueMap(b.objType)
	attrs["product_uid"] = tftypes.NewValue(tftypes.String, "vxc-uid")
	attrs["product_name"] = tftypes.NewValue(tftypes.String, "test-vxc")
	attrs["rate_limit"] = tftypes.NewValue(tftypes.Number, 1000)
	attrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, 1)
	attrs["a_end"] = aEnd
	attrs["b_end"] = bEnd
	if conns.Type() != nil {
		attrs["csp_connections"] = conns
	}
	if partner.Type() != nil {
		attrs["b_end_partner_config"] = partner
	}
	return tftypes.NewValue(b.objType, attrs)
}

func TestVXCModifyPlan_BEndVLANGate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)

	liveBEnd := gateEnd{orderedVLAN: num(200), vlan: num(200)}
	bCSP := func(connectType string) [2]string { return [2]string{"b_csp_connection", connectType} }

	tests := []struct {
		name      string
		conns     [][2]string
		partner   string
		stateA    gateEnd
		planA     gateEnd
		stateB    gateEnd
		planB     gateEnd
		wantAttrs []string
		wantType  string
	}{
		{name: "aws ordered_vlan change rejected", conns: [][2]string{bCSP("AWS")}, stateB: liveBEnd, planB: gateEnd{orderedVLAN: num(300), vlan: num(200)}, wantAttrs: []string{"ordered_vlan"}, wantType: "AWS"},
		{name: "transit ordered_vlan change rejected", conns: [][2]string{bCSP("TRANSIT")}, stateB: liveBEnd, planB: gateEnd{orderedVLAN: num(300), vlan: num(200)}, wantAttrs: []string{"ordered_vlan"}, wantType: "TRANSIT"},
		{name: "google ordered_vlan change rejected", conns: [][2]string{bCSP("GOOGLE")}, stateB: liveBEnd, planB: gateEnd{orderedVLAN: num(300), vlan: num(200)}, wantAttrs: []string{"ordered_vlan"}, wantType: "GOOGLE"},
		{name: "ibm ordered_vlan change rejected", conns: [][2]string{bCSP("IBM")}, stateB: liveBEnd, planB: gateEnd{orderedVLAN: num(300), vlan: num(200)}, wantAttrs: []string{"ordered_vlan"}, wantType: "IBM"},
		{name: "azure ordered_vlan change rejected", conns: [][2]string{bCSP("AZURE")}, stateB: liveBEnd, planB: gateEnd{orderedVLAN: num(300), vlan: num(200)}, wantAttrs: []string{"ordered_vlan"}, wantType: "AZURE"},
		{name: "partner config stands in for empty csp_connections", partner: "transit", stateB: liveBEnd, planB: gateEnd{orderedVLAN: num(300), vlan: num(200)}, wantAttrs: []string{"ordered_vlan"}, wantType: "TRANSIT"},
		{name: "aws inner_vlan rejected", conns: [][2]string{bCSP("AWS")}, stateB: liveBEnd, planB: gateEnd{orderedVLAN: num(200), vlan: num(200), innerVLAN: num(10)}, wantAttrs: []string{"inner_vlan"}, wantType: "AWS"},
		{name: "outer and inner both reported", conns: [][2]string{bCSP("ORACLE")}, stateB: liveBEnd, planB: gateEnd{orderedVLAN: num(300), vlan: num(200), innerVLAN: num(10)}, wantAttrs: []string{"ordered_vlan", "inner_vlan"}, wantType: "ORACLE"},

		{name: "azure inner_vlan permitted", conns: [][2]string{bCSP("AZURE")}, stateB: liveBEnd, planB: gateEnd{orderedVLAN: num(200), vlan: num(200), innerVLAN: num(10)}},
		{name: "vrouter b-end permitted", conns: [][2]string{bCSP("VROUTER")}, stateB: liveBEnd, planB: gateEnd{orderedVLAN: num(300), vlan: num(200)}},
		{name: "a-end only csp connection permitted", conns: [][2]string{{"a_csp_connection", "AWS"}}, stateB: liveBEnd, planB: gateEnd{orderedVLAN: num(300), vlan: num(200)}},
		{name: "megaport b-end permitted", stateB: liveBEnd, planB: gateEnd{orderedVLAN: num(300), vlan: num(200)}},
		{name: "vrouter partner permitted", partner: "vrouter", stateB: liveBEnd, planB: gateEnd{orderedVLAN: num(300), vlan: num(200)}},
		{name: "auto-assign permitted", conns: [][2]string{bCSP("AWS")}, stateB: liveBEnd, planB: gateEnd{orderedVLAN: num(0), vlan: num(200)}},
		{name: "live vlan after import permitted", conns: [][2]string{bCSP("AWS")}, stateB: gateEnd{vlan: num(200)}, planB: gateEnd{orderedVLAN: num(200), vlan: num(200)}},
		{name: "unknown ordered_vlan permitted", conns: [][2]string{bCSP("AWS")}, stateB: liveBEnd, planB: gateEnd{orderedVLAN: unknownNum, vlan: num(200)}},
		{name: "untagged inner_vlan permitted", conns: [][2]string{bCSP("AWS")}, stateB: liveBEnd, planB: gateEnd{orderedVLAN: num(200), vlan: num(200), innerVLAN: num(-1)}},
		{name: "a-end change permitted on aws b-end", conns: [][2]string{bCSP("AWS")}, stateA: gateEnd{orderedVLAN: num(100), vlan: num(100)}, planA: gateEnd{orderedVLAN: num(150), vlan: num(100), innerVLAN: num(10)}, stateB: liveBEnd, planB: liveBEnd},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var conns, partner tftypes.Value
			if tc.conns != nil {
				conns = b.cspConnections(t, tc.conns...)
			}
			if tc.partner != "" {
				partner = b.partner(tc.partner)
			}
			state := b.gateVXC(b.gateEnd("port-a", tc.stateA), b.gateEnd("port-b", tc.stateB), conns, partner)
			plan := b.gateVXC(b.gateEnd("port-a", tc.planA), b.gateEnd("port-b", tc.planB), conns, partner)

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
				if !strings.Contains(errs[i].Detail(), "is a "+tc.wantType+" connection") {
					t.Errorf("error %d detail does not name %s: %q", i, tc.wantType, errs[i].Detail())
				}
			}
		})
	}
}

// The A-End VLAN reaches the update request. The B-End VLAN on a cloud end is
// the live VLAN here, and the request leaves it out.
func TestVXCUpdate_VLANSendsOnCloudBEnd(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)
	ps := newPreflightServer(t, nil)

	conns := b.cspConnections(t, [2]string{"b_csp_connection", "AWS"})
	partner := b.partner("aws")
	state := b.gateVXC(
		b.gateEnd("port-a", gateEnd{orderedVLAN: num(100), vlan: num(100)}),
		b.gateEnd("port-b", gateEnd{vlan: num(200)}),
		conns, partner,
	)
	plan := b.gateVXC(
		b.gateEnd("port-a", gateEnd{orderedVLAN: num(150), vlan: num(100), innerVLAN: num(10)}),
		b.gateEnd("port-b", gateEnd{orderedVLAN: num(200), vlan: num(200), innerVLAN: num(20)}),
		conns, partner,
	)

	resp := fwresource.UpdateResponse{State: tfsdk.State{Schema: b.schema, Raw: state}}
	ps.resource(t).Update(ctx, fwresource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: b.schema, Raw: plan},
		State: tfsdk.State{Schema: b.schema, Raw: state},
	}, &resp)

	if len(ps.updateBodies) != 1 {
		t.Fatalf("expected one update request, got %d: %v", len(ps.updateBodies), resp.Diagnostics.Errors())
	}
	body := ps.updateBodies[0]
	if body["aEndVlan"] != float64(150) || body["aEndInnerVlan"] != float64(10) {
		t.Errorf("expected aEndVlan 150 and aEndInnerVlan 10, got %v", body)
	}
	for _, field := range []string{"bEndVlan", "bEndInnerVlan"} {
		if v, ok := body[field]; ok {
			t.Errorf("expected no %s on a cloud B-End, got %v", field, v)
		}
	}
	for _, q := range ps.vlanQueries {
		if q.portUID == "port-b" {
			t.Errorf("expected no VLAN check on the cloud B-End, got %v", q)
		}
	}
}
