package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// diversity_zone is fixed at order time. An empty value from Read is a backend
// data gap, not a real change; with RequiresReplace on the attribute, writing
// it back would plan a destroy-and-recreate of a live resource. This pins the
// preserve-on-empty behavior of the shared helper; TestDiversityZoneRequiresReplace
// below confirms a genuine config change still replaces, for every resource that
// carries a diversity_zone.
func TestDiversityZoneFromAPI(t *testing.T) {
	tests := []struct {
		name    string
		current types.String
		apiVal  string
		want    types.String
	}{
		{
			name:    "known zone, empty API echo preserves state",
			current: types.StringValue("red"),
			apiVal:  "",
			want:    types.StringValue("red"),
		},
		{
			name:    "empty state, empty API stays empty",
			current: types.StringValue(""),
			apiVal:  "",
			want:    types.StringValue(""),
		},
		{
			name:    "null state, empty API becomes concrete empty",
			current: types.StringNull(),
			apiVal:  "",
			want:    types.StringValue(""),
		},
		{
			name:    "unknown state (create), empty API becomes concrete empty",
			current: types.StringUnknown(),
			apiVal:  "",
			want:    types.StringValue(""),
		},
		{
			name:    "non-empty API is always reflected",
			current: types.StringValue("red"),
			apiVal:  "blue",
			want:    types.StringValue("blue"),
		},
		{
			name:    "non-empty API overwrites null state",
			current: types.StringNull(),
			apiVal:  "green",
			want:    types.StringValue("green"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diversityZoneFromAPI(tt.current, tt.apiVal)
			if !got.Equal(tt.want) {
				t.Errorf("diversityZoneFromAPI(%v, %q) = %v, want %v", tt.current, tt.apiVal, got, tt.want)
			}
		})
	}
}

// requiresReplaceOnChange runs every plan modifier declared on the resource's
// diversity_zone attribute for a state->plan transition and reports whether the
// resource is planned for replacement.
func requiresReplaceOnChange(t *testing.T, r resource.Resource, stateVal, planVal types.String) bool {
	t.Helper()
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	attr, ok := schemaResp.Schema.Attributes["diversity_zone"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("diversity_zone attribute missing or not a StringAttribute")
	}

	// Non-null Raw values mark this as an update (not create/destroy), which is
	// the only case where RequiresReplace fires.
	stateRaw := tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "state")}
	planRaw := tfsdk.Plan{Raw: tftypes.NewValue(tftypes.String, "plan")}

	requiresReplace := false
	// Modifiers run in schema order, and the framework threads each one's
	// PlanValue output into the next one's PlanValue input; replicate that here
	// rather than re-feeding every modifier the original, unmodified plan value.
	currentPlanVal := planVal
	for _, m := range attr.PlanModifiers {
		req := planmodifier.StringRequest{
			State:       stateRaw,
			Plan:        planRaw,
			StateValue:  stateVal,
			PlanValue:   currentPlanVal,
			ConfigValue: planVal,
		}
		resp := &planmodifier.StringResponse{PlanValue: currentPlanVal}
		m.PlanModifyString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("plan modifier returned diagnostics: %v", resp.Diagnostics)
		}
		currentPlanVal = resp.PlanValue
		if resp.RequiresReplace {
			requiresReplace = true
		}
	}
	return requiresReplace
}

func TestDiversityZoneRequiresReplace(t *testing.T) {
	resources := map[string]resource.Resource{
		"megaport_port":     NewPortResource(),
		"megaport_lag_port": NewLagPortResource(),
		"megaport_mcr":      NewMCRResource(),
		"megaport_mve":      NewMVEResource(),
	}

	for name, r := range resources {
		t.Run(name+" real change replaces", func(t *testing.T) {
			if !requiresReplaceOnChange(t, r, types.StringValue("red"), types.StringValue("blue")) {
				t.Errorf("%s: config change red->blue should require replacement", name)
			}
		})
		t.Run(name+" no change does not replace", func(t *testing.T) {
			if requiresReplaceOnChange(t, r, types.StringValue("red"), types.StringValue("red")) {
				t.Errorf("%s: unchanged diversity_zone should not require replacement", name)
			}
		})
	}
}
