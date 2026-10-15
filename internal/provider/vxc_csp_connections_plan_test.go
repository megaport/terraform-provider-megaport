package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestVXCModifyPlan_VRouterInterfacesUnknownOnPartnerChange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)
	ty := newVXCPartnerTestTypes(ctx, t)
	live := bEndAt(200, nil)
	vrouterAndAWS := [][2]string{{"a_csp_connection", "VROUTER"}, {"b_csp_connection", "AWS"}}

	aEndPartner := func(t *testing.T, v tftypes.Value, name string) tftypes.Value {
		t.Helper()
		if name == "" {
			return v
		}
		return b.with(t, v, map[string]tftypes.Value{"a_end_partner_config": b.partner(name)})
	}

	tests := []struct {
		name                        string
		conns                       [][2]string
		stateAPartner, planAPartner string
		stateBPartner, planBPartner string
		vrouterConfigEnd            string
		unknownBEnd                 bool
		rateLimitChange             bool
		wantUnknown                 bool
	}{
		{name: "a-end partner config change", stateAPartner: "vrouter", planAPartner: "transit", wantUnknown: true},
		{name: "a-end partner config added", planAPartner: "vrouter", wantUnknown: true},
		{name: "b-end partner config change", stateBPartner: "vrouter", planBPartner: "transit", wantUnknown: true},
		{name: "a-end vrouter config change", stateAPartner: "vrouter", planAPartner: "vrouter", vrouterConfigEnd: "a_end_partner_config", wantUnknown: true},
		{name: "b-end vrouter config change", stateBPartner: "vrouter", planBPartner: "vrouter", vrouterConfigEnd: "b_end_partner_config", wantUnknown: true},
		{name: "vrouter config change with two VROUTER entries", conns: [][2]string{{"a_csp_connection", "VROUTER"}, {"b_csp_connection", "VROUTER"}}, stateBPartner: "vrouter", planBPartner: "vrouter", vrouterConfigEnd: "b_end_partner_config", wantUnknown: true},
		{name: "partner config change with unknown b-end", stateAPartner: "vrouter", planAPartner: "transit", unknownBEnd: true, wantUnknown: true},
		{name: "partner configs unchanged", stateAPartner: "vrouter", planAPartner: "vrouter", stateBPartner: "transit", planBPartner: "transit"},
		{name: "rate limit change with partner configs unchanged", stateAPartner: "vrouter", planAPartner: "vrouter", stateBPartner: "transit", planBPartner: "transit", rateLimitChange: true},
		{name: "no partner configs"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			conns := tc.conns
			if conns == nil {
				conns = vrouterAndAWS
			}
			state := aEndPartner(t, b.liveVXC(t, vxcEndSpec{}, live, conns, tc.stateBPartner), tc.stateAPartner)
			plan := aEndPartner(t, b.liveVXC(t, vxcEndSpec{}, live, conns, tc.planBPartner), tc.planAPartner)
			if tc.vrouterConfigEnd != "" {
				plan = b.with(t, plan, map[string]tftypes.Value{tc.vrouterConfigEnd: ty.vrouterVal(true)})
			}
			if tc.unknownBEnd {
				plan = b.with(t, plan, map[string]tftypes.Value{"b_end": tftypes.NewValue(b.endType, tftypes.UnknownValue)})
			}
			if tc.rateLimitChange {
				plan = b.with(t, plan, map[string]tftypes.Value{"rate_limit": tftypes.NewValue(tftypes.Number, 500)})
			}

			resp := fwresource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: b.schema, Raw: plan}}
			(&vxcResource{}).ModifyPlan(ctx, fwresource.ModifyPlanRequest{
				State: tfsdk.State{Schema: b.schema, Raw: state},
				Plan:  tfsdk.Plan{Schema: b.schema, Raw: plan},
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("ModifyPlan: %v", resp.Diagnostics)
			}

			var got, prior types.List
			diags := resp.Plan.GetAttribute(ctx, path.Root("csp_connections"), &got)
			diags.Append(tfsdk.State{Schema: b.schema, Raw: state}.GetAttribute(ctx, path.Root("csp_connections"), &prior)...)
			if diags.HasError() {
				t.Fatalf("reading csp_connections: %v", diags)
			}
			if len(got.Elements()) != len(conns) {
				t.Fatalf("csp_connections = %v, want %d known entries", got, len(conns))
			}
			for i, elem := range got.Elements() {
				gotEntry, ok := elem.(types.Object)
				priorEntry, priorOK := prior.Elements()[i].(types.Object)
				if !ok || !priorOK || gotEntry.IsNull() || gotEntry.IsUnknown() {
					t.Fatalf("csp_connections[%d] = %v, want a known object", i, elem)
				}
				for name, v := range gotEntry.Attributes() {
					if tc.wantUnknown && conns[i][1] == "VROUTER" && (name == "interfaces" || name == "ip_addresses") {
						if !v.IsUnknown() {
							t.Errorf("%s %s = %v, want unknown", conns[i][0], name, v)
						}
					} else if !v.Equal(priorEntry.Attributes()[name]) {
						t.Errorf("%s %s = %v, want the state value", conns[i][0], name, v)
					}
				}
			}
		})
	}
}

func TestVRouterInterfacesUnknown_NullOrUnknownList(t *testing.T) {
	t.Parallel()
	elemType := types.ObjectType{}.WithAttributeTypes(cspConnectionFullAttrs)
	for _, conns := range []types.List{types.ListNull(elemType), types.ListUnknown(elemType)} {
		var diags diag.Diagnostics
		if got := vrouterInterfacesUnknown(context.Background(), conns, &diags); !got.Equal(conns) || diags.HasError() {
			t.Errorf("vrouterInterfacesUnknown(%v) = %v, %v; want it unchanged", conns, got, diags)
		}
	}
}
