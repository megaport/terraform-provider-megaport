package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// mcrComputedWithoutModifier lists the Computed attributes on megaport_mcr that
// carry no plan modifier, so the framework marks each one unknown whenever the
// plan differs from prior state. Removing the deprecated prefix_filter_lists
// attribute from a configuration is enough to trigger that.
var mcrComputedWithoutModifier = []string{
	"aggregation_id",
	"attribute_tags",
	"cancelable",
	"company_name",
	"contract_end_date",
	"contract_start_date",
	"lag_id",
	"lag_primary",
	"last_updated",
	"live_date",
	"provisioning_status",
	"terminate_date",
}

// mcrSchemaObjectType returns the megaport_mcr schema as a tftypes object.
func mcrSchemaObjectType(t *testing.T, ctx context.Context, r *mcrResource) (tfsdk.State, tftypes.Object) {
	t.Helper()

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}
	return tfsdk.State{Schema: schemaResp.Schema}, objType
}

// mcrPriorStateAttrs builds a prior state holding a value for every attribute
// the fix has to restore, plus the identifying attributes around them.
func mcrPriorStateAttrs(objType tftypes.Object) map[string]tftypes.Value {
	attrs := nullValueMap(objType)
	attrs["product_uid"] = tftypes.NewValue(tftypes.String, "mcr-uid-1")
	attrs["product_name"] = tftypes.NewValue(tftypes.String, "mcr-one")
	attrs["cost_centre"] = tftypes.NewValue(tftypes.String, "cc-1")
	attrs["aggregation_id"] = tftypes.NewValue(tftypes.Number, 7)
	attrs["attribute_tags"] = tftypes.NewValue(objType.AttributeTypes["attribute_tags"], map[string]tftypes.Value{
		"env": tftypes.NewValue(tftypes.String, "prod"),
	})
	attrs["cancelable"] = tftypes.NewValue(tftypes.Bool, true)
	attrs["company_name"] = tftypes.NewValue(tftypes.String, "Test Company")
	attrs["contract_end_date"] = tftypes.NewValue(tftypes.String, "2027-09-01")
	attrs["contract_start_date"] = tftypes.NewValue(tftypes.String, "2026-09-01")
	attrs["lag_id"] = tftypes.NewValue(tftypes.Number, 0)
	attrs["lag_primary"] = tftypes.NewValue(tftypes.Bool, false)
	attrs["last_updated"] = tftypes.NewValue(tftypes.String, "Monday, 08-Sep-26 09:00:00 UTC")
	attrs["live_date"] = tftypes.NewValue(tftypes.String, "2026-09-01")
	attrs["provisioning_status"] = tftypes.NewValue(tftypes.String, "LIVE")
	attrs["terminate_date"] = tftypes.NewValue(tftypes.String, "")
	return attrs
}

// copyAttrs returns a shallow copy so a case can diverge from prior state
// without disturbing the others.
func copyAttrs(src map[string]tftypes.Value) map[string]tftypes.Value {
	dst := make(map[string]tftypes.Value, len(src))
	for name, value := range src {
		dst[name] = value
	}
	return dst
}

// markUnknown sets each named attribute to unknown, which is what the framework
// does to a Computed attribute the configuration leaves null.
func markUnknown(attrs map[string]tftypes.Value, objType tftypes.Object, names ...string) {
	for _, name := range names {
		attrs[name] = tftypes.NewValue(objType.AttributeTypes[name], tftypes.UnknownValue)
	}
}

// TestMCRResourceModifyPlan_ConvergesMigrationDrift covers the reported bug.
// A plan whose only content is the unknowns left behind by removing
// prefix_filter_lists has to come back equal to prior state, so Terraform
// reports no changes.
func TestMCRResourceModifyPlan_ConvergesMigrationDrift(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &mcrResource{}
	base, objType := mcrSchemaObjectType(t, ctx, r)

	stateAttrs := mcrPriorStateAttrs(objType)
	stateVal := tftypes.NewValue(objType, stateAttrs)

	planAttrs := copyAttrs(stateAttrs)
	markUnknown(planAttrs, objType, mcrComputedWithoutModifier...)
	planVal := tftypes.NewValue(objType, planAttrs)

	// The configuration no longer sets prefix_filter_lists, and never set any
	// of the twelve, so all of them are null here.
	configAttrs := nullValueMap(objType)
	configAttrs["product_name"] = tftypes.NewValue(tftypes.String, "mcr-one")
	configAttrs["cost_centre"] = tftypes.NewValue(tftypes.String, "cc-1")
	configVal := tftypes.NewValue(objType, configAttrs)

	plan := tfsdk.Plan{Schema: base.Schema, Raw: planVal}
	req := fwresource.ModifyPlanRequest{
		Config: tfsdk.Config{Schema: base.Schema, Raw: configVal},
		State:  tfsdk.State{Schema: base.Schema, Raw: stateVal},
		Plan:   plan,
	}
	resp := fwresource.ModifyPlanResponse{Plan: plan}

	r.ModifyPlan(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors, got: %v", resp.Diagnostics.Errors())
	}
	if !resp.Plan.Raw.Equal(stateVal) {
		t.Errorf("expected the plan to converge to prior state, got: %v", resp.Plan.Raw)
	}
}

// TestMCRResourceModifyPlan_KeepsUnknownsOnRealChange guards the other half.
// Update writes a fresh last_updated and reads the rest back from the API, so a
// plan that carries a real change has to leave every unknown alone.
func TestMCRResourceModifyPlan_KeepsUnknownsOnRealChange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &mcrResource{}
	base, objType := mcrSchemaObjectType(t, ctx, r)

	stateAttrs := mcrPriorStateAttrs(objType)
	stateVal := tftypes.NewValue(objType, stateAttrs)

	tests := []struct {
		name   string
		mutate func(planAttrs, configAttrs map[string]tftypes.Value)
	}{
		{
			// The AC case: renaming an MCR must still show last_updated and
			// provisioning_status as changing.
			name: "product_name changed",
			mutate: func(planAttrs, configAttrs map[string]tftypes.Value) {
				renamed := tftypes.NewValue(tftypes.String, "mcr-renamed")
				planAttrs["product_name"] = renamed
				configAttrs["product_name"] = renamed
			},
		},
		{
			// cost_centre is Optional and Computed. Sourcing it from a resource
			// that has not applied yet leaves it unknown in the configuration,
			// which is not the framework's null-config marking.
			name: "cost_centre unknown in config",
			mutate: func(planAttrs, configAttrs map[string]tftypes.Value) {
				planAttrs["cost_centre"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
				configAttrs["cost_centre"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			planAttrs := copyAttrs(stateAttrs)
			markUnknown(planAttrs, objType, mcrComputedWithoutModifier...)

			configAttrs := nullValueMap(objType)
			configAttrs["product_name"] = tftypes.NewValue(tftypes.String, "mcr-one")
			configAttrs["cost_centre"] = tftypes.NewValue(tftypes.String, "cc-1")

			tc.mutate(planAttrs, configAttrs)
			planVal := tftypes.NewValue(objType, planAttrs)

			plan := tfsdk.Plan{Schema: base.Schema, Raw: planVal}
			req := fwresource.ModifyPlanRequest{
				Config: tfsdk.Config{Schema: base.Schema, Raw: tftypes.NewValue(objType, configAttrs)},
				State:  tfsdk.State{Schema: base.Schema, Raw: stateVal},
				Plan:   plan,
			}
			resp := fwresource.ModifyPlanResponse{Plan: plan}

			r.ModifyPlan(ctx, req, &resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("expected no errors, got: %v", resp.Diagnostics.Errors())
			}
			if !resp.Plan.Raw.Equal(planVal) {
				t.Errorf("expected the plan to be left alone, got: %v", resp.Plan.Raw)
			}
		})
	}
}

// TestMCRResourceModifyPlan_SkipsCreateAndDestroy checks the two walks that
// have nothing to restore.
func TestMCRResourceModifyPlan_SkipsCreateAndDestroy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &mcrResource{}
	base, objType := mcrSchemaObjectType(t, ctx, r)

	populated := tftypes.NewValue(objType, mcrPriorStateAttrs(objType))
	null := tftypes.NewValue(objType, nil)

	tests := []struct {
		name  string
		state tftypes.Value
		plan  tftypes.Value
	}{
		{"create has no prior state", null, populated},
		{"destroy has no plan", populated, null},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plan := tfsdk.Plan{Schema: base.Schema, Raw: tc.plan}
			req := fwresource.ModifyPlanRequest{
				Config: tfsdk.Config{Schema: base.Schema, Raw: tftypes.NewValue(objType, nullValueMap(objType))},
				State:  tfsdk.State{Schema: base.Schema, Raw: tc.state},
				Plan:   plan,
			}
			resp := fwresource.ModifyPlanResponse{Plan: plan}

			r.ModifyPlan(ctx, req, &resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("expected no errors, got: %v", resp.Diagnostics.Errors())
			}
			if !resp.Plan.Raw.Equal(tc.plan) {
				t.Errorf("expected the plan to be left alone, got: %v", resp.Plan.Raw)
			}
		})
	}
}
