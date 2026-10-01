package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	megaport "github.com/megaport/megaportgo"
)

func boolRef(b bool) *bool { return &b }

// mveVisibilityAPIMVE is the MVE the mock API returns after an order or update.
func mveVisibilityAPIMVE(visible bool) *megaport.MVE {
	return &megaport.MVE{
		UID:                   "mve-uid-1",
		Name:                  "mve-one",
		LocationID:            5,
		ContractTermMonths:    12,
		Vendor:                "CISCO",
		Size:                  "SMALL",
		MarketplaceVisibility: visible,
		NetworkInterfaces:     []*megaport.MVENetworkInterface{{Description: "Data Plane"}},
	}
}

func TestMVEResourceCreate_MarketplaceVisibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		planned  any // true, false, or tftypes.UnknownValue when unset
		api      bool
		wantSent *bool
	}{
		{name: "configured true", planned: true, api: true, wantSent: boolRef(true)},
		{name: "configured false", planned: false, api: false, wantSent: boolRef(false)},
		{name: "unset", planned: tftypes.UnknownValue, api: true, wantSent: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			var sent *megaport.BuyMVERequest
			svc := &MockMVEService{
				BuyMVEFunc: func(_ context.Context, req *megaport.BuyMVERequest) (*megaport.BuyMVEResponse, error) {
					sent = req
					return &megaport.BuyMVEResponse{TechnicalServiceUID: "mve-uid-1"}, nil
				},
				GetMVEResult: mveVisibilityAPIMVE(tc.api),
			}
			r := &mveResource{client: &megaport.Client{MVEService: svc}}
			base, objType := mveSchemaObjectType(t, ctx, r)

			planAttrs := mveConfigAttrs(t, objType)
			configAttrs := mveConfigAttrs(t, objType)
			planAttrs["marketplace_visibility"] = tftypes.NewValue(tftypes.Bool, tc.planned)
			if tc.planned != tftypes.UnknownValue {
				configAttrs["marketplace_visibility"] = tftypes.NewValue(tftypes.Bool, tc.planned)
			}

			req := fwresource.CreateRequest{
				Config: tfsdk.Config{Schema: base.Schema, Raw: tftypes.NewValue(objType, configAttrs)},
				Plan:   tfsdk.Plan{Schema: base.Schema, Raw: tftypes.NewValue(objType, planAttrs)},
			}
			resp := &fwresource.CreateResponse{
				State: tfsdk.State{Schema: base.Schema, Raw: tftypes.NewValue(objType, nil)},
			}

			r.Create(ctx, req, resp)

			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
			require.NotNil(t, sent)
			assert.Equal(t, tc.wantSent, sent.MarketplaceVisibility)

			var got types.Bool
			require.False(t, resp.State.GetAttribute(ctx, path.Root("marketplace_visibility"), &got).HasError())
			assert.Equal(t, types.BoolValue(tc.api), got)
		})
	}
}

func TestMVEResourceUpdate_MarketplaceVisibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		state    bool
		planned  any // true, false, or tftypes.UnknownValue
		api      bool
		wantSent *bool
	}{
		{name: "changed to true", state: false, planned: true, api: true, wantSent: boolRef(true)},
		{name: "changed to false", state: true, planned: false, api: false, wantSent: boolRef(false)},
		// The rename is the only change, so the update must leave visibility out.
		{name: "unchanged", state: true, planned: true, api: true, wantSent: nil},
		{name: "unknown", state: true, planned: tftypes.UnknownValue, api: true, wantSent: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			var sent *megaport.ModifyMVERequest
			svc := &MockMVEService{
				ModifyMVEFunc: func(_ context.Context, req *megaport.ModifyMVERequest) (*megaport.ModifyMVEResponse, error) {
					sent = req
					return &megaport.ModifyMVEResponse{MVEUpdated: true}, nil
				},
				GetMVEResult: mveVisibilityAPIMVE(tc.api),
			}
			r := &mveResource{client: &megaport.Client{MVEService: svc}}
			base, objType := mveSchemaObjectType(t, ctx, r)

			stateAttrs := mvePriorStateAttrs(t, objType)
			stateAttrs["marketplace_visibility"] = tftypes.NewValue(tftypes.Bool, tc.state)
			stateVal := tftypes.NewValue(objType, stateAttrs)

			planAttrs := copyAttrs(stateAttrs)
			planAttrs["product_name"] = tftypes.NewValue(tftypes.String, "mve-renamed")
			planAttrs["marketplace_visibility"] = tftypes.NewValue(tftypes.Bool, tc.planned)

			req := fwresource.UpdateRequest{
				Plan:  tfsdk.Plan{Schema: base.Schema, Raw: tftypes.NewValue(objType, planAttrs)},
				State: tfsdk.State{Schema: base.Schema, Raw: stateVal},
			}
			resp := &fwresource.UpdateResponse{
				State: tfsdk.State{Schema: base.Schema, Raw: stateVal},
			}

			r.Update(ctx, req, resp)

			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
			require.NotNil(t, sent)
			assert.Equal(t, "mve-renamed", sent.Name)
			assert.Equal(t, tc.wantSent, sent.MarketplaceVisibility)

			var got types.Bool
			require.False(t, resp.State.GetAttribute(ctx, path.Root("marketplace_visibility"), &got).HasError())
			assert.Equal(t, types.BoolValue(tc.api), got)
		})
	}
}

// An MVE already in state, with the attribute left out of the configuration,
// plans no change: the attribute is Optional and the planned value falls back to
// prior state.
func TestMVEResourceSchema_MarketplaceVisibilityKeepsStateWhenUnset(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &mveResource{}
	base, objType := mveSchemaObjectType(t, ctx, r)

	attr, ok := base.Schema.(schema.Schema).Attributes["marketplace_visibility"].(schema.BoolAttribute)
	require.True(t, ok)
	assert.True(t, attr.Optional)
	assert.True(t, attr.Computed)

	stateAttrs := mvePriorStateAttrs(t, objType)
	stateAttrs["marketplace_visibility"] = tftypes.NewValue(tftypes.Bool, true)
	state := tfsdk.State{Schema: base.Schema, Raw: tftypes.NewValue(objType, stateAttrs)}

	req := planmodifier.BoolRequest{
		Path:        path.Root("marketplace_visibility"),
		State:       state,
		ConfigValue: types.BoolNull(),
		PlanValue:   types.BoolUnknown(),
		StateValue:  types.BoolValue(true),
	}
	resp := &planmodifier.BoolResponse{PlanValue: req.PlanValue}
	for _, m := range attr.PlanModifiers {
		m.PlanModifyBool(ctx, req, resp)
	}

	assert.Equal(t, types.BoolValue(true), resp.PlanValue)
}

func TestMVEResourceModifyPlan_MarketplaceVisibility(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &mveResource{}
	base, objType := mveSchemaObjectType(t, ctx, r)

	tests := []struct {
		name string
		// state, config and plan carry marketplace_visibility. A nil config is
		// an attribute left out of the configuration.
		state, config, plan any
		wantConverged       bool
	}{
		{
			// The upgrade case: vnics removal leaves unknowns, and the plan has
			// to come back equal to prior state.
			name:          "left out of the configuration",
			state:         true,
			config:        nil,
			plan:          true,
			wantConverged: true,
		},
		{
			name:   "changed in place",
			state:  false,
			config: true,
			plan:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stateAttrs := mvePriorStateAttrs(t, objType)
			stateAttrs["marketplace_visibility"] = tftypes.NewValue(tftypes.Bool, tc.state)
			stateVal := tftypes.NewValue(objType, stateAttrs)

			planAttrs := copyAttrs(stateAttrs)
			markUnknown(planAttrs, objType, mveComputedWithoutModifier...)
			planAttrs["marketplace_visibility"] = tftypes.NewValue(tftypes.Bool, tc.plan)
			planVal := tftypes.NewValue(objType, planAttrs)

			configAttrs := mveConfigAttrs(t, objType)
			configAttrs["marketplace_visibility"] = tftypes.NewValue(tftypes.Bool, tc.config)

			plan := tfsdk.Plan{Schema: base.Schema, Raw: planVal}
			req := fwresource.ModifyPlanRequest{
				Config: tfsdk.Config{Schema: base.Schema, Raw: tftypes.NewValue(objType, configAttrs)},
				State:  tfsdk.State{Schema: base.Schema, Raw: stateVal},
				Plan:   plan,
			}
			resp := fwresource.ModifyPlanResponse{Plan: plan}

			r.ModifyPlan(ctx, req, &resp)

			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
			assert.Empty(t, resp.RequiresReplace, "marketplace_visibility must not force a replace")
			if tc.wantConverged {
				assert.True(t, resp.Plan.Raw.Equal(stateVal), "expected the plan to converge to prior state")
			} else {
				assert.True(t, resp.Plan.Raw.Equal(planVal), "expected the plan to be left alone")
			}
		})
	}
}
