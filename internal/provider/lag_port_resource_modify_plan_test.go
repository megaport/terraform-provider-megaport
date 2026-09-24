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

// lagPortSchemaObjectType returns the megaport_lag_port schema as a tftypes object.
func lagPortSchemaObjectType(t *testing.T, ctx context.Context, r *lagPortResource) (fwresource.SchemaResponse, tftypes.Object) {
	t.Helper()

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}
	return schemaResp, objType
}

// lagPortStateAttrs builds prior state for a live two-port LAG.
func lagPortStateAttrs(objType tftypes.Object) map[string]tftypes.Value {
	attrs := nullValueMap(objType)
	attrs["product_uid"] = tftypes.NewValue(tftypes.String, "lag-uid-1")
	attrs["product_name"] = tftypes.NewValue(tftypes.String, "lag-one")
	attrs["port_speed"] = tftypes.NewValue(tftypes.Number, 10000)
	attrs["location_id"] = tftypes.NewValue(tftypes.Number, 5)
	attrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, 12)
	attrs["lag_count"] = tftypes.NewValue(tftypes.Number, 2)
	attrs["lag_port_uids"] = tftypes.NewValue(objType.AttributeTypes["lag_port_uids"], []tftypes.Value{
		tftypes.NewValue(tftypes.String, "lag-uid-1"),
		tftypes.NewValue(tftypes.String, "lag-uid-2"),
	})
	return attrs
}

// TestLagPortModifyPlan_LagCount covers all three directions the count can move.
// A decrease has to keep forcing a replacement, because the API has no call to
// remove a LAG member. An increase has to stay an update, so the existing ports
// survive it.
func TestLagPortModifyPlan_LagCount(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &lagPortResource{}

	schemaResp, objType := lagPortSchemaObjectType(t, ctx, r)

	stateAttrs := lagPortStateAttrs(objType)
	stateVal := tftypes.NewValue(objType, stateAttrs)

	tests := []struct {
		name            string
		plannedLagCount tftypes.Value
		wantReplace     bool
		wantUIDsUnknown bool
	}{
		{
			name:            "increase adds ports in place",
			plannedLagCount: tftypes.NewValue(tftypes.Number, 3),
			wantReplace:     false,
			wantUIDsUnknown: true,
		},
		{
			name:            "decrease still replaces",
			plannedLagCount: tftypes.NewValue(tftypes.Number, 1),
			wantReplace:     true,
			wantUIDsUnknown: false,
		},
		{
			name:            "same count changes nothing",
			plannedLagCount: tftypes.NewValue(tftypes.Number, 2),
			wantReplace:     false,
			wantUIDsUnknown: false,
		},
		{
			// A count sourced from a resource that has not applied yet cannot decide
			// a replacement. Replacing on it would cancel a live LAG over a value
			// that may resolve to the count state already holds. It can still land
			// above that count, so the UID list has to be unknown.
			name:            "unknown count holds the replacement and the UID list",
			plannedLagCount: tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
			wantReplace:     false,
			wantUIDsUnknown: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			planAttrs := copyAttrs(stateAttrs)
			planAttrs["lag_count"] = tc.plannedLagCount
			planVal := tftypes.NewValue(objType, planAttrs)

			plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal}
			req := fwresource.ModifyPlanRequest{
				Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: planVal},
				State:  tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
				Plan:   plan,
			}
			resp := fwresource.ModifyPlanResponse{Plan: plan}

			r.ModifyPlan(ctx, req, &resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("expected no errors, got: %v", resp.Diagnostics.Errors())
			}

			gotReplace := len(resp.RequiresReplace) > 0
			if gotReplace != tc.wantReplace {
				t.Errorf("RequiresReplace = %v, want %v (paths: %v)", gotReplace, tc.wantReplace, resp.RequiresReplace)
			}
			if gotReplace && !resp.RequiresReplace[0].Equal(path.Root("lag_count")) {
				t.Errorf("RequiresReplace path = %v, want lag_count", resp.RequiresReplace[0])
			}

			var uids types.List
			resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("lag_port_uids"), &uids)...)
			if resp.Diagnostics.HasError() {
				t.Fatalf("could not read lag_port_uids from the plan: %v", resp.Diagnostics.Errors())
			}
			if uids.IsUnknown() != tc.wantUIDsUnknown {
				t.Errorf("lag_port_uids unknown = %v, want %v", uids.IsUnknown(), tc.wantUIDsUnknown)
			}
			if !tc.wantUIDsUnknown && !uids.Equal(mustLagPortUIDList(t, ctx, "lag-uid-1", "lag-uid-2")) {
				t.Errorf("lag_port_uids = %v, want the two ports prior state holds", uids)
			}
		})
	}
}

func mustLagPortUIDList(t *testing.T, ctx context.Context, uids ...string) types.List {
	t.Helper()

	list, diags := types.ListValueFrom(ctx, types.StringType, uids)
	if diags.HasError() {
		t.Fatalf("could not build the expected UID list: %v", diags.Errors())
	}
	return list
}

// TestLagPortModifyPlan_NullUIDList covers a LAG whose members the API never
// reported. lag_count is the only count state holds there, so both decisions have
// to come from it. Reading the member list alone plans an update that then orders
// nothing, and the count never converges.
func TestLagPortModifyPlan_NullUIDList(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &lagPortResource{}
	schemaResp, objType := lagPortSchemaObjectType(t, ctx, r)

	stateAttrs := lagPortStateAttrs(objType)
	stateAttrs["lag_port_uids"] = tftypes.NewValue(objType.AttributeTypes["lag_port_uids"], nil)
	stateVal := tftypes.NewValue(objType, stateAttrs)

	tests := []struct {
		name            string
		plannedLagCount int64
		wantReplace     bool
		wantUIDsUnknown bool
	}{
		{name: "increase stays an update", plannedLagCount: 3, wantUIDsUnknown: true},
		{name: "decrease replaces", plannedLagCount: 1, wantReplace: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			planAttrs := copyAttrs(stateAttrs)
			planAttrs["lag_count"] = tftypes.NewValue(tftypes.Number, tc.plannedLagCount)
			planVal := tftypes.NewValue(objType, planAttrs)

			plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal}
			resp := fwresource.ModifyPlanResponse{Plan: plan}
			r.ModifyPlan(ctx, fwresource.ModifyPlanRequest{
				Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: planVal},
				State:  tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
				Plan:   plan,
			}, &resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("expected no errors, got: %v", resp.Diagnostics.Errors())
			}
			if gotReplace := len(resp.RequiresReplace) > 0; gotReplace != tc.wantReplace {
				t.Errorf("RequiresReplace = %v, want %v (paths: %v)", gotReplace, tc.wantReplace, resp.RequiresReplace)
			}

			var uids types.List
			resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("lag_port_uids"), &uids)...)
			if resp.Diagnostics.HasError() {
				t.Fatalf("could not read lag_port_uids from the plan: %v", resp.Diagnostics.Errors())
			}
			if uids.IsUnknown() != tc.wantUIDsUnknown {
				t.Errorf("lag_port_uids unknown = %v, want %v", uids.IsUnknown(), tc.wantUIDsUnknown)
			}
		})
	}
}
