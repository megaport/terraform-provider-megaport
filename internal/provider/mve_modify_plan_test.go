package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestMVEModifyPlan_ImportVendorMapping covers the import scenario where state
// has no vendor config block (blocks are null) and the API-reported vendor must
// be matched against the configured block. The API reports a coarse vendor value
// that is not always the uppercased block name (6wind -> SIX_WIND, prisma ->
// PALO_ALTO, vmware -> ARISTA), so a naive case-fold compare forced a spurious
// destroy-and-recreate on import for those vendors.
func TestMVEModifyPlan_ImportVendorMapping(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &mveResource{}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

	schemaObjType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}

	// blockValue builds a minimal valid vendor config block (the two required
	// attributes) for the named config attribute.
	blockValue := func(attrName, size string) tftypes.Value {
		blockType, ok := schemaObjType.AttributeTypes[attrName].(tftypes.Object)
		if !ok {
			t.Fatalf("%s type is not tftypes.Object", attrName)
		}
		attrs := nullValueMap(blockType)
		attrs["image_id"] = tftypes.NewValue(tftypes.Number, 42)
		attrs["product_size"] = tftypes.NewValue(tftypes.String, size)
		return tftypes.NewValue(blockType, attrs)
	}

	tests := []struct {
		name        string
		configAttr  string // vendor config block set in the plan
		apiVendor   string // vendor the API reports in state
		wantReplace bool
	}{
		{"6wind matches SIX_WIND", "sixwind_config", "SIX_WIND", false},
		{"prisma matches PALO_ALTO", "prisma_config", "PALO_ALTO", false},
		{"vmware matches ARISTA", "vmware_config", "ARISTA", false},
		{"cisco matches CISCO", "cisco_config", "CISCO", false},
		{"aruba matches ARUBA", "aruba_config", "ARUBA", false},
		{"6wind against CISCO replaces", "sixwind_config", "CISCO", true},
		{"vmware against PALO_ALTO replaces", "vmware_config", "PALO_ALTO", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// State: imported MVE — product_uid + API vendor/size set, all vendor
			// config blocks null.
			stateAttrs := nullValueMap(schemaObjType)
			stateAttrs["product_uid"] = tftypes.NewValue(tftypes.String, "mve-uid-123")
			stateAttrs["vendor"] = tftypes.NewValue(tftypes.String, tc.apiVendor)
			stateAttrs["mve_size"] = tftypes.NewValue(tftypes.String, "SMALL")
			stateVal := tftypes.NewValue(schemaObjType, stateAttrs)

			// Plan: the configured vendor block, same size.
			planAttrs := nullValueMap(schemaObjType)
			planAttrs["product_uid"] = tftypes.NewValue(tftypes.String, "mve-uid-123")
			planAttrs[tc.configAttr] = blockValue(tc.configAttr, "small")
			planVal := tftypes.NewValue(schemaObjType, planAttrs)

			req := fwresource.ModifyPlanRequest{
				State: tfsdk.State{Schema: s, Raw: stateVal},
				Plan:  tfsdk.Plan{Schema: s, Raw: planVal},
			}
			resp := fwresource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: s, Raw: planVal}}

			r.ModifyPlan(ctx, req, &resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics.Errors())
			}
			gotReplace := len(resp.RequiresReplace) > 0
			if gotReplace != tc.wantReplace {
				t.Errorf("RequiresReplace = %v, want %v (paths: %v)", gotReplace, tc.wantReplace, resp.RequiresReplace)
			}
		})
	}
}
