package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const v1ProviderAddress = "registry.terraform.io/megaport/megaport"

// buildV1VXCState builds a V1 VXC state JSON payload from the given fields.
func buildV1VXCState(t *testing.T, fields map[string]interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(fields)
	require.NoError(t, err)
	return b
}

// runMoveState runs the VXC MoveState logic and returns the response.
func runMoveState(t *testing.T, providerAddr, typeName string, rawJSON []byte) resource.MoveStateResponse {
	t.Helper()

	ctx := context.Background()
	r := &vxcResource{}

	movers := r.MoveState(ctx)
	require.Len(t, movers, 1)

	// Build a target schema from the resource.
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, schemaReq, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError(), "schema diagnostics: %s", schemaResp.Diagnostics)

	req := resource.MoveStateRequest{
		SourceProviderAddress: providerAddr,
		SourceTypeName:        typeName,
		SourceSchemaVersion:   0,
		SourceRawState: &tfprotov6.RawState{
			JSON: rawJSON,
		},
	}

	resp := resource.MoveStateResponse{
		TargetState: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
		},
	}

	movers[0].StateMover(ctx, req, &resp)
	return resp
}

func TestMoveState_VXC_V1ToV2_Basic(t *testing.T) {
	rawJSON := buildV1VXCState(t, map[string]interface{}{
		"product_uid":          "abc-123",
		"service_id":          42,
		"product_name":        "My VXC",
		"rate_limit":          1000,
		"distance_band":       "ZONE",
		"promo_code":          "PROMO",
		"service_key":         "skey",
		"created_by":          "user@example.com",
		"contract_term_months": 12,
		"company_uid":         "comp-uid",
		"cost_centre":         "CC001",
		"shutdown":            false,
		"attribute_tags":      map[string]string{"env": "test"},
		"resource_tags":       map[string]string{"team": "eng"},
		"a_end": map[string]interface{}{
			"requested_product_uid": "port-a-uid",
			"current_product_uid":   "port-a-assigned",
			"vlan":                  100,
			"inner_vlan":            200,
			"vnic_index":            0,
			"owner_uid":             "owner-dropped",
			"product_name":          "dropped-name",
			"location_id":           999,
			"location":              "dropped-loc",
			"ordered_vlan":          50,
			"secondary_name":        "dropped-sec",
		},
		"b_end": map[string]interface{}{
			"requested_product_uid": "port-b-uid",
			"current_product_uid":   "port-b-assigned",
			"vlan":                  300,
			"inner_vlan":            nil,
			"vnic_index":            1,
		},
		// V1-only fields that should be dropped.
		"last_updated":         "2024-01-01",
		"product_id":           999,
		"product_type":         "VXC",
		"provisioning_status":  "LIVE",
		"secondary_name":       "sec",
		"usage_algorithm":      "alg",
		"company_name":         "MyCo",
		"locked":               false,
		"admin_locked":         false,
		"cancelable":           true,
		"live_date":            "2024-01-01",
		"create_date":          "2024-01-01",
		"contract_start_date":  "2024-01-01",
		"contract_end_date":    "2025-01-01",
	})

	resp := runMoveState(t, v1ProviderAddress, "megaport_vxc", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "diagnostics: %s", resp.Diagnostics)

	// Extract the state into our model.
	var state vxcResourceModel
	diags := resp.TargetState.Get(context.Background(), &state)
	require.False(t, diags.HasError(), "get state: %s", diags)

	// Verify scalar fields.
	assert.Equal(t, "abc-123", state.UID.ValueString())
	assert.Equal(t, int64(42), state.ServiceID.ValueInt64())
	assert.Equal(t, "My VXC", state.Name.ValueString())
	assert.Equal(t, int64(1000), state.RateLimit.ValueInt64())
	assert.Equal(t, "ZONE", state.DistanceBand.ValueString())
	assert.Equal(t, "PROMO", state.PromoCode.ValueString())
	assert.Equal(t, "skey", state.ServiceKey.ValueString())
	assert.Equal(t, "user@example.com", state.CreatedBy.ValueString())
	assert.Equal(t, int64(12), state.ContractTermMonths.ValueInt64())
	assert.Equal(t, "comp-uid", state.CompanyUID.ValueString())
	assert.Equal(t, "CC001", state.CostCentre.ValueString())
	assert.Equal(t, false, state.Shutdown.ValueBool())

	// Verify A-end config.
	var aEnd vxcAEndConfigModel
	diags = state.AEndConfiguration.As(context.Background(), &aEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError(), "a_end: %s", diags)
	assert.Equal(t, "port-a-uid", aEnd.ProductUID.ValueString())
	assert.Equal(t, "port-a-assigned", aEnd.AssignedProductUID.ValueString())
	assert.Equal(t, int64(100), aEnd.VLAN.ValueInt64())
	assert.Equal(t, int64(200), aEnd.InnerVLAN.ValueInt64())
	assert.Equal(t, int64(0), aEnd.NetworkInterfaceIndex.ValueInt64())
	assert.True(t, aEnd.VrouterPartnerConfig.IsNull())

	// Verify B-end config.
	var bEnd vxcBEndConfigModel
	diags = state.BEndConfiguration.As(context.Background(), &bEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError(), "b_end: %s", diags)
	assert.Equal(t, "port-b-uid", bEnd.ProductUID.ValueString())
	assert.Equal(t, "port-b-assigned", bEnd.AssignedProductUID.ValueString())
	assert.Equal(t, int64(300), bEnd.VLAN.ValueInt64())
	assert.True(t, bEnd.InnerVLAN.IsNull())
	assert.Equal(t, int64(1), bEnd.NetworkInterfaceIndex.ValueInt64())
	assert.True(t, bEnd.AWSPartnerConfig.IsNull())
	assert.True(t, bEnd.AzurePartnerConfig.IsNull())
	assert.True(t, bEnd.GooglePartnerConfig.IsNull())
	assert.True(t, bEnd.OraclePartnerConfig.IsNull())
	assert.True(t, bEnd.IBMPartnerConfig.IsNull())
	assert.True(t, bEnd.VrouterPartnerConfig.IsNull())
	assert.True(t, bEnd.Transit.IsNull())
}

func TestMoveState_VXC_V1ToV2_EndConfigRename(t *testing.T) {
	rawJSON := buildV1VXCState(t, map[string]interface{}{
		"product_uid":  "uid-1",
		"product_name": "test",
		"a_end": map[string]interface{}{
			"requested_product_uid": "req-uid-a",
			"current_product_uid":   "cur-uid-a",
			"vlan":                  10,
		},
		"b_end": map[string]interface{}{
			"requested_product_uid": "req-uid-b",
			"current_product_uid":   "cur-uid-b",
			"vlan":                  20,
		},
	})

	resp := runMoveState(t, v1ProviderAddress, "megaport_vxc", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "diagnostics: %s", resp.Diagnostics)

	var state vxcResourceModel
	diags := resp.TargetState.Get(context.Background(), &state)
	require.False(t, diags.HasError())

	// Verify requested_product_uid -> product_uid rename.
	var aEnd vxcAEndConfigModel
	diags = state.AEndConfiguration.As(context.Background(), &aEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, "req-uid-a", aEnd.ProductUID.ValueString(), "V1 requested_product_uid should map to V2 product_uid")
	assert.Equal(t, "cur-uid-a", aEnd.AssignedProductUID.ValueString(), "V1 current_product_uid should map to V2 assigned_product_uid")

	var bEnd vxcBEndConfigModel
	diags = state.BEndConfiguration.As(context.Background(), &bEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, "req-uid-b", bEnd.ProductUID.ValueString(), "V1 requested_product_uid should map to V2 product_uid")
	assert.Equal(t, "cur-uid-b", bEnd.AssignedProductUID.ValueString(), "V1 current_product_uid should map to V2 assigned_product_uid")
}

func TestMoveState_VXC_V1ToV2_DroppedFields(t *testing.T) {
	// Build state with V1-only fields — they must not appear in V2 state.
	rawJSON := buildV1VXCState(t, map[string]interface{}{
		"product_uid":          "uid-drop",
		"product_name":         "drop-test",
		"last_updated":         "2024-01-01T00:00:00Z",
		"product_id":           123,
		"product_type":         "VXC",
		"provisioning_status":  "LIVE",
		"secondary_name":       "sec-name",
		"usage_algorithm":      "POST_PAID",
		"company_name":         "AcmeCorp",
		"locked":               true,
		"admin_locked":         false,
		"cancelable":           true,
		"live_date":            "2024-01-01",
		"create_date":          "2024-01-01",
		"contract_start_date":  "2024-01-01",
		"contract_end_date":    "2025-01-01",
		"a_end": map[string]interface{}{
			"requested_product_uid": "port-a",
			"current_product_uid":   "port-a-cur",
			"vlan":                  100,
			"owner_uid":             "should-be-dropped",
			"product_name":          "should-be-dropped",
			"location_id":           42,
			"location":              "should-be-dropped",
			"ordered_vlan":          50,
			"secondary_name":        "should-be-dropped",
		},
		"b_end": map[string]interface{}{
			"requested_product_uid": "port-b",
			"current_product_uid":   "port-b-cur",
			"vlan":                  200,
		},
	})

	resp := runMoveState(t, v1ProviderAddress, "megaport_vxc", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "diagnostics: %s", resp.Diagnostics)

	var state vxcResourceModel
	diags := resp.TargetState.Get(context.Background(), &state)
	require.False(t, diags.HasError())

	// The V2 model struct does not have the dropped fields at all,
	// so successfully unmarshalling proves they are not present.
	// Verify the fields that DO exist are correct.
	assert.Equal(t, "uid-drop", state.UID.ValueString())
	assert.Equal(t, "drop-test", state.Name.ValueString())

	// Verify end configs don't have V1-only fields (they can't — the V2 type doesn't include them).
	var aEnd vxcAEndConfigModel
	diags = state.AEndConfiguration.As(context.Background(), &aEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, "port-a", aEnd.ProductUID.ValueString())
	assert.Equal(t, "port-a-cur", aEnd.AssignedProductUID.ValueString())
	assert.Equal(t, int64(100), aEnd.VLAN.ValueInt64())
}

func TestMoveState_VXC_WrongProvider(t *testing.T) {
	rawJSON := buildV1VXCState(t, map[string]interface{}{
		"product_uid":  "uid-1",
		"product_name": "test",
	})

	resp := runMoveState(t, "registry.terraform.io/other/other", "megaport_vxc", rawJSON)
	// Should not error — just skip (no state set).
	require.False(t, resp.Diagnostics.HasError())

	// State should remain unset (raw is still null).
	assert.True(t, resp.TargetState.Raw.IsNull(), "state should remain null for wrong provider")
}

func TestMoveState_VXC_WrongType(t *testing.T) {
	rawJSON := buildV1VXCState(t, map[string]interface{}{
		"product_uid":  "uid-1",
		"product_name": "test",
	})

	resp := runMoveState(t, v1ProviderAddress, "megaport_port", rawJSON)
	// Should not error — just skip.
	require.False(t, resp.Diagnostics.HasError())

	// State should remain unset.
	assert.True(t, resp.TargetState.Raw.IsNull(), "state should remain null for wrong resource type")
}
