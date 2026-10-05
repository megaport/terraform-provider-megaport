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

func TestVXCModifyPlan_CSPConnectionsUnknownOnPartnerChange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)
	live := bEndAt(200, nil)
	conns := [][2]string{{"a_csp_connection", "VROUTER"}}

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
		planBVrouterConfig          bool
		unknownBEnd                 bool
		wantUnknown                 bool
	}{
		{name: "a-end partner config change", stateAPartner: "vrouter", planAPartner: "transit", wantUnknown: true},
		{name: "a-end partner config added", planAPartner: "vrouter", wantUnknown: true},
		{name: "b-end partner config change", stateBPartner: "vrouter", planBPartner: "transit", wantUnknown: true},
		{name: "b-end vrouter config change", stateBPartner: "vrouter", planBPartner: "vrouter", planBVrouterConfig: true, wantUnknown: true},
		{name: "partner config change with unknown b-end", stateAPartner: "vrouter", planAPartner: "transit", unknownBEnd: true, wantUnknown: true},
		{name: "partner configs unchanged", stateAPartner: "vrouter", planAPartner: "vrouter", stateBPartner: "transit", planBPartner: "transit"},
		{name: "no partner configs"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			state := aEndPartner(t, b.liveVXC(t, vxcEndSpec{}, live, conns, tc.stateBPartner), tc.stateAPartner)
			plan := aEndPartner(t, b.liveVXC(t, vxcEndSpec{}, live, conns, tc.planBPartner), tc.planAPartner)
			if tc.planBVrouterConfig {
				attrs := map[string]tftypes.Value{}
				if err := b.partner(tc.planBPartner).As(&attrs); err != nil {
					t.Fatalf("unpacking partner config: %v", err)
				}
				vrouterType, ok := b.partnerTyp.AttributeTypes["vrouter_config"].(tftypes.Object)
				if !ok {
					t.Fatal("vrouter_config type is not tftypes.Object")
				}
				attrs["vrouter_config"] = tftypes.NewValue(vrouterType, nullValueMap(vrouterType))
				plan = b.with(t, plan, map[string]tftypes.Value{"b_end_partner_config": tftypes.NewValue(b.partnerTyp, attrs)})
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

			var got types.List
			if diags := resp.Plan.GetAttribute(ctx, path.Root("csp_connections"), &got); diags.HasError() {
				t.Fatalf("reading csp_connections: %v", diags)
			}
			if got.IsUnknown() != tc.wantUnknown {
				t.Errorf("csp_connections unknown = %v, want %v", got.IsUnknown(), tc.wantUnknown)
			}
			if !tc.wantUnknown && len(got.Elements()) != len(conns) {
				t.Errorf("csp_connections = %v, want the state value", got)
			}
		})
	}
}
