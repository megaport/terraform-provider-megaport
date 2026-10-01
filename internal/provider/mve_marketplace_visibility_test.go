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
		{name: "configured true", planned: true, api: true, wantSent: megaport.PtrTo(true)},
		{name: "configured false", planned: false, api: false, wantSent: megaport.PtrTo(false)},
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
		{name: "changed to true", state: false, planned: true, api: true, wantSent: megaport.PtrTo(true)},
		{name: "changed to false", state: true, planned: false, api: false, wantSent: megaport.PtrTo(false)},
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

// An MVE in state keeps its prior value when the attribute is left out of the
// configuration, and a changed value never forces a replace.
func TestMVEResourceSchema_MarketplaceVisibilityPlanModifiers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &mveResource{}
	base, objType := mveSchemaObjectType(t, ctx, r)

	attr, ok := base.Schema.(schema.Schema).Attributes["marketplace_visibility"].(schema.BoolAttribute)
	require.True(t, ok)
	assert.True(t, attr.Optional)
	assert.True(t, attr.Computed)

	tests := []struct {
		name   string
		state  bool
		config types.Bool
		plan   types.Bool
	}{
		{name: "left out of the configuration", state: true, config: types.BoolNull(), plan: types.BoolUnknown()},
		{name: "changed in the configuration", state: false, config: types.BoolValue(true), plan: types.BoolValue(true)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stateAttrs := mvePriorStateAttrs(t, objType)
			stateAttrs["marketplace_visibility"] = tftypes.NewValue(tftypes.Bool, tc.state)
			state := tfsdk.State{Schema: base.Schema, Raw: tftypes.NewValue(objType, stateAttrs)}

			planRaw, err := tc.plan.ToTerraformValue(ctx)
			require.NoError(t, err)
			planAttrs := copyAttrs(stateAttrs)
			planAttrs["marketplace_visibility"] = planRaw

			req := planmodifier.BoolRequest{
				Path:        path.Root("marketplace_visibility"),
				Plan:        tfsdk.Plan{Schema: base.Schema, Raw: tftypes.NewValue(objType, planAttrs)},
				State:       state,
				ConfigValue: tc.config,
				PlanValue:   tc.plan,
				StateValue:  types.BoolValue(tc.state),
			}
			resp := &planmodifier.BoolResponse{PlanValue: req.PlanValue}
			for _, m := range attr.PlanModifiers {
				m.PlanModifyBool(ctx, req, resp)
				req.PlanValue = resp.PlanValue
			}

			assert.Equal(t, types.BoolValue(true), resp.PlanValue)
			assert.False(t, resp.RequiresReplace)
		})
	}
}
