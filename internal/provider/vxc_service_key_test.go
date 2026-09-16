package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	megaport "github.com/megaport/megaportgo"
)

// service_key is an order-time input that the API never returns, so an
// imported VXC holds it null the same as an ordinary VXC that was simply
// never given a key. Only a private-state flag that Read sets on the import
// read tells the two apart, and this harness cannot fabricate that flag
// (the framework's private-state type is internal to the dependency, so no
// package outside it can construct one). The genuine post-import case, where
// the flag is set and the plan updates in place, is covered end to end by
// TestAccMegaportVXC_ImportDrift_ServiceKey instead.
func TestVXCServiceKeyRequiresReplace(t *testing.T) {
	tests := []struct {
		name  string
		state types.String
		plan  types.String
		want  bool
	}{
		{
			name:  "added to an ordinary VXC that was never given a key",
			state: types.StringNull(),
			plan:  types.StringValue("key-1"),
			want:  true,
		},
		{
			name:  "changed to another key",
			state: types.StringValue("key-1"),
			plan:  types.StringValue("key-2"),
			want:  true,
		},
		{
			name:  "removed from configuration",
			state: types.StringValue("key-1"),
			plan:  types.StringNull(),
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requiresReplaceOnAttrChange(t, NewVXCResource(), "service_key", tt.state, tt.plan)
			if got != tt.want {
				t.Errorf("requires replace = %v, want %v", got, tt.want)
			}
		})
	}
}

// The API never returns the service key, so Update has to carry it from the
// plan. Without that, the apply after an import writes null back over the
// planned key and the framework fails with an inconsistent result.
func TestVXCUpdate_KeepsServiceKeyFromPlan(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	r := &vxcResource{client: &megaport.Client{
		VXCService: &MockVXCService{
			GetVXCResult: &megaport.VXC{
				UID:               "vxc-uid-123",
				Name:              "test-vxc",
				AEndConfiguration: megaport.VXCEndConfiguration{UID: "a-end-uid"},
				BEndConfiguration: megaport.VXCEndConfiguration{UID: "b-end-uid"},
			},
		},
		ProductService: &MockProductService{
			GetProductTypeFunc: func(_ context.Context, _ string) (string, error) {
				return megaport.PRODUCT_MEGAPORT, nil
			},
		},
	}}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

	schemaObjType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}
	endType, ok := schemaObjType.AttributeTypes["a_end"].(tftypes.Object)
	if !ok {
		t.Fatal("a_end type is not tftypes.Object")
	}
	endValue := func(uid string) tftypes.Value {
		attrs := nullValueMap(endType)
		attrs["requested_product_uid"] = tftypes.NewValue(tftypes.String, uid)
		return tftypes.NewValue(endType, attrs)
	}

	// State is what an imported VXC holds: the key is null because the API
	// does not return it.
	stateAttrs := nullValueMap(schemaObjType)
	stateAttrs["product_uid"] = tftypes.NewValue(tftypes.String, "vxc-uid-123")
	stateAttrs["product_name"] = tftypes.NewValue(tftypes.String, "test-vxc")
	stateAttrs["a_end"] = endValue("a-end-uid")
	stateAttrs["b_end"] = endValue("b-end-uid")

	planAttrs := nullValueMap(schemaObjType)
	for k, v := range stateAttrs {
		planAttrs[k] = v
	}
	planAttrs["service_key"] = tftypes.NewValue(tftypes.String, "service-key-1")

	state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(schemaObjType, stateAttrs)}
	plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(schemaObjType, planAttrs)}

	resp := fwresource.UpdateResponse{State: state}
	r.Update(ctx, fwresource.UpdateRequest{State: state, Plan: plan}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics.Errors())
	}

	var got vxcResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading state back: %v", diags.Errors())
	}
	if got.ServiceKey.ValueString() != "service-key-1" {
		t.Errorf("service_key in state = %q, want %q", got.ServiceKey.ValueString(), "service-key-1")
	}
}
