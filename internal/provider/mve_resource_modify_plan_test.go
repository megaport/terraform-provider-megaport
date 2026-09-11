package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// mveComputedWithoutModifier lists the Computed attributes on megaport_mve that
// carry no plan modifier, so the framework marks each one unknown whenever the
// plan differs from prior state. Removing the vnics attribute from a
// configuration is enough to trigger that.
var mveComputedWithoutModifier = []string{
	"contract_end_date",
	"contract_start_date",
	"cost_centre",
	"last_updated",
	"live_date",
	"provisioning_status",
	"terminate_date",
}

// mveSchemaObjectType returns the megaport_mve schema as a tftypes object.
func mveSchemaObjectType(t *testing.T, ctx context.Context, r *mveResource) (tfsdk.State, tftypes.Object) {
	t.Helper()

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}
	return tfsdk.State{Schema: schemaResp.Schema}, objType
}

// mveVendorConfig builds the Required vendor_config object. Every plan, state
// and configuration in these tests carries one.
func mveVendorConfig(t *testing.T, objType tftypes.Object, productSize string) tftypes.Value {
	t.Helper()

	vcType, ok := objType.AttributeTypes["vendor_config"].(tftypes.Object)
	if !ok {
		t.Fatal("vendor_config is not tftypes.Object")
	}
	attrs := nullValueMap(vcType)
	attrs["vendor"] = tftypes.NewValue(tftypes.String, "cisco")
	attrs["image_id"] = tftypes.NewValue(tftypes.Number, 42)
	attrs["product_size"] = tftypes.NewValue(tftypes.String, productSize)
	return tftypes.NewValue(vcType, attrs)
}

// mveVnics builds the two-element vnics list the removal case starts from.
func mveVnics(t *testing.T, objType tftypes.Object) tftypes.Value {
	t.Helper()

	listType, ok := objType.AttributeTypes["vnics"].(tftypes.List)
	if !ok {
		t.Fatal("vnics is not tftypes.List")
	}
	elemType := listType.ElementType
	return tftypes.NewValue(listType, []tftypes.Value{
		tftypes.NewValue(elemType, map[string]tftypes.Value{
			"description": tftypes.NewValue(tftypes.String, "Data Plane"),
			"vlan":        tftypes.NewValue(tftypes.Number, 0),
		}),
		tftypes.NewValue(elemType, map[string]tftypes.Value{
			"description": tftypes.NewValue(tftypes.String, "Control Plane"),
			"vlan":        tftypes.NewValue(tftypes.Number, 0),
		}),
	})
}

// mvePriorStateAttrs builds a prior state holding a value for every attribute
// the fix has to restore, plus the identifying attributes around them.
func mvePriorStateAttrs(t *testing.T, objType tftypes.Object) map[string]tftypes.Value {
	t.Helper()

	attrs := nullValueMap(objType)
	attrs["product_uid"] = tftypes.NewValue(tftypes.String, "mve-uid-1")
	attrs["product_name"] = tftypes.NewValue(tftypes.String, "mve-one")
	attrs["location_id"] = tftypes.NewValue(tftypes.Number, 5)
	attrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, 12)
	// The API uppercases both, the configuration below sends them lowercase.
	attrs["vendor"] = tftypes.NewValue(tftypes.String, "CISCO")
	attrs["mve_size"] = tftypes.NewValue(tftypes.String, "SMALL")
	attrs["vendor_config"] = mveVendorConfig(t, objType, "SMALL")
	attrs["vnics"] = mveVnics(t, objType)
	attrs["contract_end_date"] = tftypes.NewValue(tftypes.String, "2027-09-01")
	attrs["contract_start_date"] = tftypes.NewValue(tftypes.String, "2026-09-01")
	attrs["cost_centre"] = tftypes.NewValue(tftypes.String, "cc-1")
	attrs["last_updated"] = tftypes.NewValue(tftypes.String, "Monday, 08-Sep-26 09:00:00 UTC")
	attrs["live_date"] = tftypes.NewValue(tftypes.String, "2026-09-01")
	attrs["provisioning_status"] = tftypes.NewValue(tftypes.String, "LIVE")
	attrs["terminate_date"] = tftypes.NewValue(tftypes.String, "")
	return attrs
}

// mveConfigAttrs builds the configuration left after vnics is deleted. The
// seven restorable attributes are null here, which is what makes the framework
// mark them unknown.
func mveConfigAttrs(t *testing.T, objType tftypes.Object) map[string]tftypes.Value {
	t.Helper()

	attrs := nullValueMap(objType)
	attrs["product_name"] = tftypes.NewValue(tftypes.String, "mve-one")
	attrs["location_id"] = tftypes.NewValue(tftypes.Number, 5)
	attrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, 12)
	attrs["vendor_config"] = mveVendorConfig(t, objType, "SMALL")
	return attrs
}

// TestMVEResourceModifyPlan_ConvergesVnicsRemoval covers the reported bug. A
// plan whose only content is the unknowns left by removing vnics has to come
// back equal to prior state, so Terraform reports no changes.
func TestMVEResourceModifyPlan_ConvergesVnicsRemoval(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &mveResource{}
	base, objType := mveSchemaObjectType(t, ctx, r)

	stateAttrs := mvePriorStateAttrs(t, objType)
	stateVal := tftypes.NewValue(objType, stateAttrs)

	// vnics carries UseStateForUnknown, so by the time the resource-level
	// method runs it already holds prior state again. The seven below do not.
	planAttrs := copyAttrs(stateAttrs)
	markUnknown(planAttrs, objType, mveComputedWithoutModifier...)
	planVal := tftypes.NewValue(objType, planAttrs)

	configVal := tftypes.NewValue(objType, mveConfigAttrs(t, objType))

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
	if len(resp.RequiresReplace) != 0 {
		t.Errorf("expected no replacement, got: %v", resp.RequiresReplace)
	}
	if !resp.Plan.Raw.Equal(stateVal) {
		t.Errorf("expected the plan to converge to prior state, got: %v", resp.Plan.Raw)
	}
}

// TestMVEResourceModifyPlan_KeepsUnknownsOnRealChange guards the other half.
// Update writes a fresh last_updated and reads the rest back from the API, so a
// plan that carries a real change has to leave every unknown alone.
func TestMVEResourceModifyPlan_KeepsUnknownsOnRealChange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &mveResource{}
	base, objType := mveSchemaObjectType(t, ctx, r)

	stateVal := tftypes.NewValue(objType, mvePriorStateAttrs(t, objType))
	resizedVendorConfig := mveVendorConfig(t, objType, "MEDIUM")

	tests := []struct {
		name            string
		mutate          func(planAttrs, configAttrs map[string]tftypes.Value)
		requiresReplace bool
	}{
		{
			// The AC case: renaming an MVE must still show last_updated and
			// provisioning_status as changing.
			name: "product_name changed",
			mutate: func(planAttrs, configAttrs map[string]tftypes.Value) {
				renamed := tftypes.NewValue(tftypes.String, "mve-renamed")
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
		{
			// The size change still replaces the MVE, and the plan it replaces
			// from keeps its unknowns.
			name: "vendor_config size changed",
			mutate: func(planAttrs, configAttrs map[string]tftypes.Value) {
				planAttrs["vendor_config"] = resizedVendorConfig
				configAttrs["vendor_config"] = resizedVendorConfig
			},
			requiresReplace: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			planAttrs := copyAttrs(mvePriorStateAttrs(t, objType))
			markUnknown(planAttrs, objType, mveComputedWithoutModifier...)
			configAttrs := mveConfigAttrs(t, objType)

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

			replaced := len(resp.RequiresReplace) != 0
			if replaced != tc.requiresReplace {
				t.Errorf("expected RequiresReplace %v, got: %v", tc.requiresReplace, resp.RequiresReplace)
			}
			if tc.requiresReplace && !resp.RequiresReplace.Contains(path.Root("vendor_config")) {
				t.Errorf("expected vendor_config to require replacement, got: %v", resp.RequiresReplace)
			}
		})
	}
}

// TestMVEResourceModifyPlan_SkipsCreateAndDestroy checks the two walks that
// have nothing to restore.
func TestMVEResourceModifyPlan_SkipsCreateAndDestroy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &mveResource{}
	base, objType := mveSchemaObjectType(t, ctx, r)

	populated := tftypes.NewValue(objType, mvePriorStateAttrs(t, objType))
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
