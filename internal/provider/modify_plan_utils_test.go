package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestRestoreComputedOnNoOpPlan_NestedObjects checks that the walk into a
// nested object restores the same unknowns the top level does, and still
// leaves the plan alone on a real change.
func TestRestoreComputedOnNoOpPlan_NestedObjects(t *testing.T) {
	t.Parallel()

	endType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"port":   tftypes.String,
		"status": tftypes.String,
	}}
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":   tftypes.String,
		"status": tftypes.String,
		"end":    endType,
	}}

	str := func(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }
	unknown := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	null := tftypes.NewValue(tftypes.String, nil)
	nullEnd := tftypes.NewValue(endType, nil)
	end := func(port, status tftypes.Value) tftypes.Value {
		return tftypes.NewValue(endType, map[string]tftypes.Value{"port": port, "status": status})
	}
	obj := func(name, status, endValue tftypes.Value) tftypes.Value {
		return tftypes.NewValue(objType, map[string]tftypes.Value{"name": name, "status": status, "end": endValue})
	}

	state := obj(str("vxc"), str("LIVE"), end(str("port-1"), str("CONFIGURED")))

	// plan is a func so the expected value does not share a map with the input.
	tests := []struct {
		name      string
		plan      func() tftypes.Value
		config    tftypes.Value
		converges bool
	}{
		{
			name:      "nested unknowns converge",
			plan:      func() tftypes.Value { return obj(str("vxc"), unknown, end(str("port-1"), unknown)) },
			config:    obj(str("vxc"), null, end(str("port-1"), null)),
			converges: true,
		},
		{
			name:      "null config object counts as null for each child",
			plan:      func() tftypes.Value { return obj(str("vxc"), unknown, end(str("port-1"), unknown)) },
			config:    obj(str("vxc"), null, nullEnd),
			converges: true,
		},
		{
			name:   "nested real change",
			plan:   func() tftypes.Value { return obj(str("vxc"), unknown, end(str("port-2"), unknown)) },
			config: obj(str("vxc"), null, end(str("port-2"), null)),
		},
		{
			name:   "nested unknown set in config",
			plan:   func() tftypes.Value { return obj(str("vxc"), unknown, end(str("port-1"), unknown)) },
			config: obj(str("vxc"), null, end(str("port-1"), unknown)),
		},
		{
			name:   "nested object removed",
			plan:   func() tftypes.Value { return obj(str("vxc"), unknown, nullEnd) },
			config: obj(str("vxc"), null, nullEnd),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			want := tc.plan()
			if tc.converges {
				want = state
			}

			got, err := restoreComputedOnNoOpPlan(tc.plan(), state, tc.config)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if !got.Equal(want) {
				t.Errorf("expected %v, got: %v", want, got)
			}
		})
	}
}
