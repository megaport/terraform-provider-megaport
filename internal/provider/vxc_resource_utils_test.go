package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSupportVLANUpdates(t *testing.T) {
	cases := map[string]bool{
		"":        true,
		"transit": false,
		"aws":     false,
		"azure":   true,
		"google":  true,
		"oracle":  true,
		"vrouter": true,
		"a-end":   true,
	}
	for partner, want := range cases {
		if got := supportVLANUpdates(partner); got != want {
			t.Errorf("supportVLANUpdates(%q) = %v, want %v", partner, got, want)
		}
	}
}

// nullCSPConnectionObjectValues builds a minimal cspConnectionModel object
// with all fields null except connect_type, suitable for testing the helper.
func nullCSPConnectionObjectValues(t *testing.T, connectType string) attr.Value {
	t.Helper()
	values := map[string]attr.Value{}
	for name, typ := range cspConnectionFullAttrs {
		switch tt := typ.(type) {
		case types.ListType:
			values[name] = types.ListNull(tt.ElemType)
		default:
			switch typ {
			case types.StringType:
				values[name] = types.StringNull()
			case types.Int64Type:
				values[name] = types.Int64Null()
			case types.BoolType:
				values[name] = types.BoolNull()
			default:
				t.Fatalf("unhandled attr type for %q: %T", name, typ)
			}
		}
	}
	values["connect_type"] = types.StringValue(connectType)
	obj, diags := types.ObjectValue(cspConnectionFullAttrs, values)
	if diags.HasError() {
		t.Fatalf("ObjectValue: %v", diags)
	}
	return obj
}

func TestStateHasUnsupportedVLANCSPConnection(t *testing.T) {
	ctx := context.Background()
	objType := types.ObjectType{AttrTypes: cspConnectionFullAttrs}

	tests := []struct {
		name         string
		connectTypes []string
		nullList     bool
		want         bool
	}{
		{name: "null list", nullList: true, want: false},
		{name: "empty list", connectTypes: []string{}, want: false},
		{name: "supported only", connectTypes: []string{"AZURE", "GOOGLE", "ORACLE"}, want: false},
		{name: "transit alone", connectTypes: []string{"TRANSIT"}, want: true},
		{name: "aws alone", connectTypes: []string{"AWS"}, want: true},
		{name: "awshc alone", connectTypes: []string{"AWSHC"}, want: true},
		{name: "transit mixed with supported", connectTypes: []string{"AZURE", "TRANSIT"}, want: true},
		{name: "lowercase transit not matched", connectTypes: []string{"transit"}, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var list types.List
			if tc.nullList {
				list = types.ListNull(objType)
			} else {
				elems := make([]attr.Value, 0, len(tc.connectTypes))
				for _, ct := range tc.connectTypes {
					elems = append(elems, nullCSPConnectionObjectValues(t, ct))
				}
				l, diags := types.ListValue(objType, elems)
				if diags.HasError() {
					t.Fatalf("ListValue: %v", diags)
				}
				list = l
			}

			if got := stateHasUnsupportedVLANCSPConnection(ctx, list); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
