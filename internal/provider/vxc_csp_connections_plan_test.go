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

func TestVXCModifyPlan_VRouterInterfacesUnknownOnPartnerChange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)
	ty := newVXCPartnerTestTypes(ctx, t)
	live := bEndAt(200, nil)
	conns := [][2]string{{"a_csp_connection", "VROUTER"}, {"b_csp_connection", "AWS"}}

	aEndPartner := func(t *testing.T, v tftypes.Value, name string) tftypes.Value {
		t.Helper()
		if name == "" {
			return v
		}
		return b.with(t, v, map[string]tftypes.Value{"a_end_partner_config": b.partner(name)})
	}

	tests := []struct {
		name                        string
		stateAPartner, planAPartner string
		stateBPartner, planBPartner string
		vrouterConfigEnd            string
		unknownBEnd                 bool
		wantUnknown                 bool
	}{
		{name: "a-end partner config change", stateAPartner: "vrouter", planAPartner: "transit", wantUnknown: true},
		{name: "a-end partner config added", planAPartner: "vrouter", wantUnknown: true},
		{name: "b-end partner config change", stateBPartner: "vrouter", planBPartner: "transit", wantUnknown: true},
		{name: "a-end vrouter config change", stateAPartner: "vrouter", planAPartner: "vrouter", vrouterConfigEnd: "a_end_partner_config", wantUnknown: true},
		{name: "b-end vrouter config change", stateBPartner: "vrouter", planBPartner: "vrouter", vrouterConfigEnd: "b_end_partner_config", wantUnknown: true},
		{name: "partner config change with unknown b-end", stateAPartner: "vrouter", planAPartner: "transit", unknownBEnd: true, wantUnknown: true},
		{name: "partner configs unchanged", stateAPartner: "vrouter", planAPartner: "vrouter", stateBPartner: "transit", planBPartner: "transit"},
		{name: "no partner configs"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			state := aEndPartner(t, b.liveVXC(t, vxcEndSpec{}, live, conns, tc.stateBPartner), tc.stateAPartner)
			plan := aEndPartner(t, b.liveVXC(t, vxcEndSpec{}, live, conns, tc.planBPartner), tc.planAPartner)
			if tc.vrouterConfigEnd != "" {
				plan = b.with(t, plan, map[string]tftypes.Value{tc.vrouterConfigEnd: ty.vrouterVal(true)})
			}
			if tc.unknownBEnd {
				plan = b.with(t, plan, map[string]tftypes.Value{"b_end": tftypes.NewValue(b.endType, tftypes.UnknownValue)})
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
			vrouter, ok := got.Elements()[0].(types.Object)
			if !ok {
				t.Fatalf("VROUTER entry = %T, want types.Object", got.Elements()[0])
			}
			for _, name := range []string{"interfaces", "ip_addresses"} {
				if v := vrouter.Attributes()[name]; v.IsUnknown() != tc.wantUnknown {
					t.Errorf("VROUTER %s unknown = %v, want %v", name, v.IsUnknown(), tc.wantUnknown)
				}
			}
			if !got.Elements()[1].Equal(prior.Elements()[1]) {
				t.Errorf("AWS entry = %v, want the state value", got.Elements()[1])
			}
			if !tc.wantUnknown && !got.Equal(prior) {
				t.Errorf("csp_connections = %v, want the state value", got)
			}
		})
	}
}
