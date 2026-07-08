package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// v1PortState returns a complete V1 port state JSON with all fields that
// existed in V1 (including those removed in V2).
func v1PortState() []byte {
	state := map[string]interface{}{
		// Fields kept in V2
		"product_uid":            "uid-abc-123",
		"product_name":           "My Port",
		"port_speed":             10000,
		"location_id":            42,
		"marketplace_visibility": true,
		"company_uid":            "company-xyz",
		"cost_centre":            "CC-001",
		"contract_term_months":   12,
		"diversity_zone":         "red",
		"promo_code":             "PROMO1",
		"resources": map[string]interface{}{
			"interface": map[string]interface{}{
				"demarcation": "Test Demarc",
				"up":          1,
			},
		},
		"resource_tags": map[string]string{
			"env":  "staging",
			"team": "infra",
		},
		// Fields removed in V2
		"last_updated":        "2024-01-01T00:00:00Z",
		"product_id":          99,
		"provisioning_status": "LIVE",
		"create_date":         "2024-01-01T00:00:00Z",
		"created_by":          "user@example.com",
		"terminate_date":      "2025-01-01T00:00:00Z",
		"live_date":           "2024-01-02T00:00:00Z",
		"market":              "AU",
		"usage_algorithm":     "some-algo",
		"vxc_permitted":       true,
		"vxc_auto_approval":   false,
		"contract_start_date": "2024-01-01",
		"contract_end_date":   "2025-01-01",
		"virtual":             false,
		"locked":              false,
		"cancelable":          true,
	}
	b, _ := json.Marshal(state)
	return b
}

// v1LagPortState returns a complete V1 LAG port state JSON.
func v1LagPortState() []byte {
	state := map[string]interface{}{
		// Fields kept in V2
		"product_uid":            "lag-uid-456",
		"product_name":           "My LAG Port",
		"port_speed":             100000,
		"location_id":            7,
		"marketplace_visibility": false,
		"company_uid":            "company-lag",
		"cost_centre":            "CC-LAG",
		"contract_term_months":   24,
		"diversity_zone":         "blue",
		"promo_code":             "",
		"lag_count":              4,
		"lag_port_uids":          []string{"lag-1", "lag-2", "lag-3", "lag-4"},
		"resources": map[string]interface{}{
			"interface": map[string]interface{}{
				"demarcation": "LAG Demarc",
				"up":          1,
			},
		},
		"resource_tags": map[string]string{
			"env": "prod",
		},
		// Fields removed in V2
		"last_updated":        "2024-06-01T00:00:00Z",
		"product_id":          200,
		"provisioning_status": "LIVE",
		"create_date":         "2024-06-01T00:00:00Z",
		"created_by":          "admin@example.com",
		"terminate_date":      nil,
		"live_date":           "2024-06-02T00:00:00Z",
		"market":              "US",
		"usage_algorithm":     "lag-algo",
		"vxc_permitted":       true,
		"vxc_auto_approval":   true,
		"contract_start_date": "2024-06-01",
		"contract_end_date":   "2026-06-01",
		"virtual":             false,
		"locked":              false,
		"cancelable":          true,
	}
	b, _ := json.Marshal(state)
	return b
}

// invokePortStateMover decodes the raw V1 state against the mover's
// SourceSchema exactly as the framework server does, then runs the mover.
func invokePortStateMover(t *testing.T, r resource.ResourceWithMoveState, providerAddr, typeName string, rawJSON []byte) *resource.MoveStateResponse {
	t.Helper()
	ctx := context.Background()

	movers := r.MoveState(ctx)
	require.Len(t, movers, 1)
	require.NotNil(t, movers[0].SourceSchema)

	req := resource.MoveStateRequest{
		SourceProviderAddress: providerAddr,
		SourceTypeName:        typeName,
		SourceRawState:        &tfprotov6.RawState{JSON: rawJSON},
	}

	if rawJSON != nil {
		rawValue, err := req.SourceRawState.UnmarshalWithOpts(
			movers[0].SourceSchema.Type().TerraformType(ctx),
			tfprotov6.UnmarshalOpts{
				ValueFromJSONOpts: tftypes.ValueFromJSONOpts{
					IgnoreUndefinedAttributes: true,
				},
			},
		)
		require.NoError(t, err, "failed to decode V1 state against the V2 schema")
		req.SourceState = &tfsdk.State{
			Raw:    rawValue,
			Schema: *movers[0].SourceSchema,
		}
	}

	resp := &resource.MoveStateResponse{
		TargetState: tfsdk.State{
			Schema: *movers[0].SourceSchema,
			Raw:    tftypes.NewValue(movers[0].SourceSchema.Type().TerraformType(ctx), nil),
		},
	}

	movers[0].StateMover(ctx, req, resp)
	return resp
}

func TestMoveState_Port_V1ToV2(t *testing.T) {
	ctx := context.Background()

	resp := invokePortStateMover(t, &portResource{}, "registry.terraform.io/megaport/megaport", "megaport_port", v1PortState())
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)

	var model singlePortResourceModel
	diags := resp.TargetState.Get(ctx, &model)
	require.False(t, diags.HasError(), "failed to read target state: %v", diags)

	// Verify kept fields are correctly migrated.
	assert.Equal(t, "uid-abc-123", model.UID.ValueString())
	assert.Equal(t, "My Port", model.Name.ValueString())
	assert.Equal(t, int64(10000), model.PortSpeed.ValueInt64())
	assert.Equal(t, int64(42), model.LocationID.ValueInt64())
	assert.True(t, model.MarketplaceVisibility.ValueBool())
	assert.Equal(t, "company-xyz", model.CompanyUID.ValueString())
	assert.Equal(t, "CC-001", model.CostCentre.ValueString())
	assert.Equal(t, int64(12), model.ContractTermMonths.ValueInt64())
	assert.Equal(t, "red", model.DiversityZone.ValueString())
	assert.Equal(t, "PROMO1", model.PromoCode.ValueString())

	// Verify resources nested object.
	assert.False(t, model.Resources.IsNull())

	// Verify resource tags.
	assert.False(t, model.ResourceTags.IsNull())
	tagElements := model.ResourceTags.Elements()
	require.Len(t, tagElements, 2)
	envTag, ok := tagElements["env"].(types.String)
	require.True(t, ok)
	assert.Equal(t, "staging", envTag.ValueString())
	teamTag, ok := tagElements["team"].(types.String)
	require.True(t, ok)
	assert.Equal(t, "infra", teamTag.ValueString())
}

func TestMoveState_Port_V1ToV2_NilOptionals(t *testing.T) {
	ctx := context.Background()

	// V1 state with null optional fields.
	state := map[string]interface{}{
		"product_uid":            "uid-nil-test",
		"product_name":           "Nil Port",
		"port_speed":             1000,
		"location_id":            1,
		"marketplace_visibility": false,
		"company_uid":            "co",
		"cost_centre":            nil,
		"contract_term_months":   1,
		"diversity_zone":         nil,
		"promo_code":             nil,
		"resources":              nil,
		"resource_tags":          nil,
		// V1-only fields
		"last_updated":        nil,
		"product_id":          nil,
		"provisioning_status": nil,
	}
	rawJSON, _ := json.Marshal(state)

	resp := invokePortStateMover(t, &portResource{}, "registry.terraform.io/megaport/megaport", "megaport_port", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)

	var model singlePortResourceModel
	diags := resp.TargetState.Get(ctx, &model)
	require.False(t, diags.HasError(), "failed to read target state: %v", diags)

	assert.Equal(t, "uid-nil-test", model.UID.ValueString())
	assert.Equal(t, "Nil Port", model.Name.ValueString())
	assert.Equal(t, int64(1000), model.PortSpeed.ValueInt64())
	assert.True(t, model.CostCentre.IsNull())
	assert.True(t, model.DiversityZone.IsNull())
	assert.True(t, model.PromoCode.IsNull())
	assert.True(t, model.Resources.IsNull())
	assert.True(t, model.ResourceTags.IsNull())
}

func TestMoveState_LagPort_V1ToV2(t *testing.T) {
	ctx := context.Background()

	resp := invokePortStateMover(t, &lagPortResource{}, "registry.terraform.io/megaport/megaport", "megaport_lag_port", v1LagPortState())
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)

	var model lagPortResourceModel
	diags := resp.TargetState.Get(ctx, &model)
	require.False(t, diags.HasError(), "failed to read target state: %v", diags)

	// Verify kept fields.
	assert.Equal(t, "lag-uid-456", model.UID.ValueString())
	assert.Equal(t, "My LAG Port", model.Name.ValueString())
	assert.Equal(t, int64(100000), model.PortSpeed.ValueInt64())
	assert.Equal(t, int64(7), model.LocationID.ValueInt64())
	assert.False(t, model.MarketplaceVisibility.ValueBool())
	assert.Equal(t, "company-lag", model.CompanyUID.ValueString())
	assert.Equal(t, "CC-LAG", model.CostCentre.ValueString())
	assert.Equal(t, int64(24), model.ContractTermMonths.ValueInt64())
	assert.Equal(t, "blue", model.DiversityZone.ValueString())
	assert.Equal(t, "", model.PromoCode.ValueString())
	assert.Equal(t, int64(4), model.LagCount.ValueInt64())

	// Verify lag_port_uids.
	assert.False(t, model.LagPortUIDs.IsNull())
	uidElements := model.LagPortUIDs.Elements()
	require.Len(t, uidElements, 4)
	uid0, ok := uidElements[0].(types.String)
	require.True(t, ok)
	assert.Equal(t, "lag-1", uid0.ValueString())
	uid3, ok := uidElements[3].(types.String)
	require.True(t, ok)
	assert.Equal(t, "lag-4", uid3.ValueString())

	// Verify resources nested object.
	assert.False(t, model.Resources.IsNull())

	// Verify resource tags.
	assert.False(t, model.ResourceTags.IsNull())
	tagElements := model.ResourceTags.Elements()
	require.Len(t, tagElements, 1)
	envTag, ok := tagElements["env"].(types.String)
	require.True(t, ok)
	assert.Equal(t, "prod", envTag.ValueString())
}

func TestMoveState_Port_WrongProvider(t *testing.T) {
	resp := invokePortStateMover(t, &portResource{}, "registry.terraform.io/other/other", "megaport_port", v1PortState())

	// Should be a no-op (skipped): no diagnostics, no state set.
	assert.False(t, resp.Diagnostics.HasError())
	assert.True(t, resp.TargetState.Raw.IsNull(), "expected target state to remain null for wrong provider")
}

func TestMoveState_Port_WrongType(t *testing.T) {
	resp := invokePortStateMover(t, &portResource{}, "registry.terraform.io/megaport/megaport", "megaport_vxc", v1PortState())

	// Should be a no-op (skipped).
	assert.False(t, resp.Diagnostics.HasError())
	assert.True(t, resp.TargetState.Raw.IsNull(), "expected target state to remain null for wrong type")
}

func TestMoveState_LagPort_WrongType(t *testing.T) {
	resp := invokePortStateMover(t, &lagPortResource{}, "registry.terraform.io/megaport/megaport", "megaport_port", v1LagPortState())

	// Should be a no-op (skipped).
	assert.False(t, resp.Diagnostics.HasError())
	assert.True(t, resp.TargetState.Raw.IsNull(), "expected target state to remain null for wrong type")
}

func TestMoveState_Port_NilSourceState(t *testing.T) {
	// A nil SourceState means the framework could not decode the source raw
	// state against the source schema; the mover must error, not panic.
	resp := invokePortStateMover(t, &portResource{}, "registry.terraform.io/megaport/megaport", "megaport_port", nil)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Unable to migrate V1 state")
}

func TestMoveState_LagPort_NilSourceState(t *testing.T) {
	resp := invokePortStateMover(t, &lagPortResource{}, "registry.terraform.io/megaport/megaport", "megaport_lag_port", nil)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Unable to migrate V1 state")
}
