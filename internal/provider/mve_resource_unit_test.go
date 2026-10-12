package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func vnicTestList(t *testing.T, descriptions ...string) types.List {
	t.Helper()
	objType := types.ObjectType{}.WithAttributeTypes(vnicAttrs)
	elems := make([]attr.Value, 0, len(descriptions))
	for i, d := range descriptions {
		obj, diags := types.ObjectValue(vnicAttrs, map[string]attr.Value{
			"description": types.StringValue(d),
			"vlan":        types.Int64Value(int64(100 + i)),
		})
		if diags.HasError() {
			t.Fatalf("building vnic object: %v", diags)
		}
		elems = append(elems, obj)
	}
	list, diags := types.ListValue(objType, elems)
	if diags.HasError() {
		t.Fatalf("building vnic list: %v", diags)
	}
	return list
}

func TestVnicCountChanged(t *testing.T) {
	objType := types.ObjectType{}.WithAttributeTypes(vnicAttrs)

	tests := []struct {
		name  string
		state types.List
		plan  types.List
		want  bool
	}{
		{"same count, descriptions edited", vnicTestList(t, "a", "b"), vnicTestList(t, "a2", "b2"), false},
		{"same count, identical", vnicTestList(t, "a", "b"), vnicTestList(t, "a", "b"), false},
		{"count increased", vnicTestList(t, "a", "b"), vnicTestList(t, "a", "b", "c"), true},
		{"count decreased", vnicTestList(t, "a", "b", "c"), vnicTestList(t, "a", "b"), true},
		// Create: no prior state, so nothing to replace.
		{"null state", types.ListNull(objType), vnicTestList(t, "a"), false},
		{"unknown state", types.ListUnknown(objType), vnicTestList(t, "a"), false},
		// Plan null/unknown reaches the modifier only when UseStateForUnknown
		// did not run; the safe default is "no replace".
		{"null plan", vnicTestList(t, "a", "b"), types.ListNull(objType), false},
		{"unknown plan", vnicTestList(t, "a", "b"), types.ListUnknown(objType), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &listplanmodifier.RequiresReplaceIfFuncResponse{}
			vnicCountChanged(context.Background(), planmodifier.ListRequest{
				StateValue: tt.state,
				PlanValue:  tt.plan,
			}, resp)
			if resp.RequiresReplace != tt.want {
				t.Fatalf("RequiresReplace = %v, want %v", resp.RequiresReplace, tt.want)
			}
		})
	}
}
