package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mcrV2Schema returns the V2 MCR schema for test use.
func mcrV2Schema() schema.Schema {
	r := &mcrResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	return resp.Schema
}

// newMCRTargetState creates an empty tfsdk.State initialized with the V2 MCR schema.
func newMCRTargetState(t *testing.T) tfsdk.State {
	t.Helper()
	s := mcrV2Schema()
	return tfsdk.State{
		Schema: s,
		Raw:    tftypes.NewValue(s.Type().TerraformType(context.Background()), nil),
	}
}

// invokeMCRStateMover calls the MCR StateMover function with the given parameters.
func invokeMCRStateMover(t *testing.T, providerAddr, typeName string, rawJSON []byte) (*resource.MoveStateResponse, *mcrResourceModel) {
	t.Helper()
	ctx := context.Background()
	r := &mcrResource{}
	movers := r.MoveState(ctx)
	require.Len(t, movers, 1, "expected exactly 1 state mover")

	req := resource.MoveStateRequest{
		SourceProviderAddress: providerAddr,
		SourceTypeName:        typeName,
		SourceRawState: &tfprotov6.RawState{
			JSON: rawJSON,
		},
	}

	resp := &resource.MoveStateResponse{
		TargetState: newMCRTargetState(t),
	}

	movers[0].StateMover(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		return resp, nil
	}

	// If the state was not set (skipped), return nil model
	if resp.TargetState.Raw.IsNull() {
		return resp, nil
	}

	var model mcrResourceModel
	diags := resp.TargetState.Get(ctx, &model)
	if diags.HasError() {
		t.Fatalf("failed to get target state: %v", diags)
	}
	return resp, &model
}

// fullV1State returns a complete V1 MCR state as JSON (with all V1 fields).
func fullV1State() map[string]interface{} {
	return map[string]interface{}{
		// Fields kept in V2
		"product_uid":            "mcr-uid-abc123",
		"product_name":           "My MCR",
		"cost_centre":            "CC-42",
		"port_speed":             float64(5000),
		"location_id":            float64(123),
		"marketplace_visibility": true,
		"company_uid":            "company-uid-xyz",
		"contract_term_months":   float64(12),
		"asn":                    float64(64512),
		"diversity_zone":         "blue",
		"promo_code":             "PROMO2024",
		"attribute_tags": map[string]interface{}{
			"account": "prod",
		},
		"resource_tags": map[string]interface{}{
			"env": "staging",
		},
		// Fields removed in V2
		"last_updated":           "2024-01-01T00:00:00Z",
		"id":                     "mcr-uid-abc123",
		"product_type":           "MCR2",
		"provisioning_status":    "LIVE",
		"create_date":            "2024-01-01T00:00:00Z",
		"created_by":             "user@example.com",
		"terminate_date":         nil,
		"live_date":              "2024-01-01T00:00:00Z",
		"market":                 "US",
		"usage_algorithm":        "POST_PAID_FIXED",
		"vxc_permitted":          true,
		"vxc_auto_approval":      false,
		"secondary_name":         "",
		"lag_primary":            false,
		"lag_id":                 float64(0),
		"aggregation_id":         float64(0),
		"company_name":           "Test Company",
		"contract_start_date":    "2024-01-01T00:00:00Z",
		"contract_end_date":      "2025-01-01T00:00:00Z",
		"virtual":                false,
		"buyout_port":            false,
		"locked":                 false,
		"admin_locked":           false,
		"cancelable":             true,
		"prefix_filter_lists":    nil,
	}
}

func TestMoveState_MCR_V1ToV2(t *testing.T) {
	v1 := fullV1State()
	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMCRStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mcr", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	// Verify all kept fields
	assert.Equal(t, "mcr-uid-abc123", model.UID.ValueString())
	assert.Equal(t, "My MCR", model.Name.ValueString())
	assert.Equal(t, "CC-42", model.CostCentre.ValueString())
	assert.Equal(t, int64(5000), model.PortSpeed.ValueInt64())
	assert.Equal(t, int64(123), model.LocationID.ValueInt64())
	assert.True(t, model.MarketplaceVisibility.ValueBool())
	assert.Equal(t, "company-uid-xyz", model.CompanyUID.ValueString())
	assert.Equal(t, int64(12), model.ContractTermMonths.ValueInt64())
	assert.Equal(t, int64(64512), model.ASN.ValueInt64())
	assert.Equal(t, "blue", model.DiversityZone.ValueString())
	assert.Equal(t, "PROMO2024", model.PromoCode.ValueString())

	// Verify tags
	assert.False(t, model.AttributeTags.IsNull())
	attrTags := model.AttributeTags.Elements()
	assert.Equal(t, "prod", attrTags["account"].(types.String).ValueString())

	assert.False(t, model.ResourceTags.IsNull())
	resTags := model.ResourceTags.Elements()
	assert.Equal(t, "staging", resTags["env"].(types.String).ValueString())
}

func TestMoveState_MCR_V1ToV2_NilOptionals(t *testing.T) {
	v1 := fullV1State()
	// Set optional fields to null/empty
	v1["asn"] = nil
	v1["attribute_tags"] = nil
	v1["resource_tags"] = nil
	v1["prefix_filter_lists"] = nil
	v1["promo_code"] = nil
	v1["diversity_zone"] = nil
	v1["cost_centre"] = nil

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMCRStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mcr", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	// Kept fields still present
	assert.Equal(t, "mcr-uid-abc123", model.UID.ValueString())
	assert.Equal(t, "My MCR", model.Name.ValueString())
	assert.Equal(t, int64(5000), model.PortSpeed.ValueInt64())

	// Null optional fields
	assert.True(t, model.ASN.IsNull(), "ASN should be null")
	assert.True(t, model.AttributeTags.IsNull(), "attribute_tags should be null")
	assert.True(t, model.ResourceTags.IsNull(), "resource_tags should be null")
	assert.True(t, model.PromoCode.IsNull(), "promo_code should be null")
	assert.True(t, model.DiversityZone.IsNull(), "diversity_zone should be null")
	assert.True(t, model.CostCentre.IsNull(), "cost_centre should be null")
}

func TestMoveState_MCR_V1ToV2_WithASN(t *testing.T) {
	v1 := fullV1State()
	v1["asn"] = float64(65000)

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMCRStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mcr", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	assert.False(t, model.ASN.IsNull(), "ASN should not be null")
	assert.Equal(t, int64(65000), model.ASN.ValueInt64())
}

func TestMoveState_MCR_V1ToV2_WithTags(t *testing.T) {
	v1 := fullV1State()
	v1["attribute_tags"] = map[string]interface{}{
		"account": "prod",
		"team":    "network",
		"region":  "us-west",
	}
	v1["resource_tags"] = map[string]interface{}{
		"env":   "production",
		"owner": "platform-team",
	}

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMCRStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mcr", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	// Verify attribute_tags
	assert.False(t, model.AttributeTags.IsNull())
	attrTags := model.AttributeTags.Elements()
	assert.Len(t, attrTags, 3)
	assert.Equal(t, "prod", attrTags["account"].(types.String).ValueString())
	assert.Equal(t, "network", attrTags["team"].(types.String).ValueString())
	assert.Equal(t, "us-west", attrTags["region"].(types.String).ValueString())

	// Verify resource_tags
	assert.False(t, model.ResourceTags.IsNull())
	resTags := model.ResourceTags.Elements()
	assert.Len(t, resTags, 2)
	assert.Equal(t, "production", resTags["env"].(types.String).ValueString())
	assert.Equal(t, "platform-team", resTags["owner"].(types.String).ValueString())
}

func TestMoveState_MCR_WrongProvider(t *testing.T) {
	v1 := fullV1State()
	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMCRStateMover(t, "registry.terraform.io/other/provider", "megaport_mcr", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	assert.Nil(t, model, "model should be nil when provider doesn't match (skipped)")
}

func TestMoveState_MCR_WrongType(t *testing.T) {
	v1 := fullV1State()
	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMCRStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_port", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	assert.Nil(t, model, "model should be nil when type doesn't match (skipped)")
}
