package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// v1MCRState returns a complete V1 MCR state JSON with all fields that
// existed in V1 (including those removed in V2).
func v1MCRState() []byte {
	state := map[string]interface{}{
		// Fields kept in V2
		"product_uid":            "mcr-uid-123",
		"product_name":           "My MCR",
		"cost_centre":            "CC-001",
		"port_speed":             10000,
		"location_id":            7,
		"marketplace_visibility": false,
		"company_uid":            "company-xyz",
		"contract_term_months":   12,
		"asn":                    64512,
		"diversity_zone":         "red",
		"promo_code":             "PROMO1",
		"attribute_tags":         map[string]string{"env": "staging"},
		"resource_tags":          map[string]string{"env": "staging", "team": "infra"},
		// Fields removed in V2
		"last_updated":        "2024-01-01T00:00:00Z",
		"product_id":          99,
		"product_type":        "MCR2",
		"provisioning_status": "LIVE",
		"create_date":         "2024-01-01T00:00:00Z",
		"created_by":          "user@example.com",
		"terminate_date":      "2025-01-01T00:00:00Z",
		"live_date":           "2024-01-02T00:00:00Z",
		"market":              "AU",
		"usage_algorithm":     "some-algo",
		"contract_start_date": "2024-01-01",
		"contract_end_date":   "2025-01-01",
		"secondary_name":      "mcr-secondary",
		"lag_primary":         false,
		"lag_id":              0,
		"aggregation_id":      0,
		"company_name":        "My Company",
		"vxc_permitted":       true,
		"vxc_auto_approval":   false,
		"virtual":             false,
		"buyout_port":         false,
		"locked":              false,
		"admin_locked":        false,
		"cancelable":          true,
		"prefix_filter_lists": nil,
	}
	b, _ := json.Marshal(state)
	return b
}

// invokeMCRStateMover decodes the raw V1 state against the mover's
// SourceSchema exactly as the framework server does, then runs the mover.
func invokeMCRStateMover(t *testing.T, providerAddr, typeName string, rawJSON []byte) *resource.MoveStateResponse {
	t.Helper()
	ctx := context.Background()
	r := &mcrResource{}

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

func TestMoveState_MCR_V1ToV2(t *testing.T) {
	ctx := context.Background()

	resp := invokeMCRStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mcr", v1MCRState())
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)

	var model mcrResourceModel
	diags := resp.TargetState.Get(ctx, &model)
	require.False(t, diags.HasError(), "failed to read target state: %v", diags)

	// Verify kept fields are correctly migrated.
	assert.Equal(t, "mcr-uid-123", model.UID.ValueString())
	assert.Equal(t, "My MCR", model.Name.ValueString())
	assert.Equal(t, "CC-001", model.CostCentre.ValueString())
	assert.Equal(t, int64(10000), model.PortSpeed.ValueInt64())
	assert.Equal(t, int64(7), model.LocationID.ValueInt64())
	assert.False(t, model.MarketplaceVisibility.ValueBool())
	assert.Equal(t, "company-xyz", model.CompanyUID.ValueString())
	assert.Equal(t, int64(12), model.ContractTermMonths.ValueInt64())
	assert.Equal(t, int64(64512), model.ASN.ValueInt64())
	assert.Equal(t, "red", model.DiversityZone.ValueString())
	assert.Equal(t, "PROMO1", model.PromoCode.ValueString())

	// Verify attribute and resource tags.
	require.False(t, model.AttributeTags.IsNull())
	require.Len(t, model.AttributeTags.Elements(), 1)
	require.False(t, model.ResourceTags.IsNull())
	require.Len(t, model.ResourceTags.Elements(), 2)
}

func TestMoveState_MCR_V1ToV2_NilOptionals(t *testing.T) {
	ctx := context.Background()

	// V1 state with null or missing optional fields.
	state := map[string]interface{}{
		"product_uid":            "mcr-nil-test",
		"product_name":           "Nil MCR",
		"port_speed":             2500,
		"location_id":            8,
		"marketplace_visibility": false,
		"contract_term_months":   1,
		"asn":                    nil,
		"cost_centre":            nil,
		"diversity_zone":         nil,
		"promo_code":             nil,
		"company_uid":            nil,
		"attribute_tags":         nil,
		"resource_tags":          nil,
		// V1-only fields
		"last_updated":        nil,
		"product_id":          nil,
		"provisioning_status": nil,
		"created_by":          nil,
	}
	rawJSON, err := json.Marshal(state)
	require.NoError(t, err)

	resp := invokeMCRStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mcr", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)

	var model mcrResourceModel
	diags := resp.TargetState.Get(ctx, &model)
	require.False(t, diags.HasError(), "failed to read target state: %v", diags)

	assert.Equal(t, "mcr-nil-test", model.UID.ValueString())
	assert.Equal(t, "Nil MCR", model.Name.ValueString())
	assert.Equal(t, int64(2500), model.PortSpeed.ValueInt64())
	assert.Equal(t, int64(1), model.ContractTermMonths.ValueInt64())
	assert.True(t, model.ASN.IsNull())
	assert.True(t, model.CostCentre.IsNull())
	assert.True(t, model.DiversityZone.IsNull())
	assert.True(t, model.PromoCode.IsNull())
	assert.True(t, model.CompanyUID.IsNull())
	assert.True(t, model.AttributeTags.IsNull())
	assert.True(t, model.ResourceTags.IsNull())
}

func TestMoveState_MCR_WrongProvider(t *testing.T) {
	resp := invokeMCRStateMover(t, "registry.terraform.io/other/other", "megaport_mcr", v1MCRState())

	// Should be a no-op (skipped): no diagnostics, no state set.
	assert.False(t, resp.Diagnostics.HasError())
	assert.True(t, resp.TargetState.Raw.IsNull(), "expected target state to remain null for wrong provider")
}

func TestMoveState_MCR_WrongType(t *testing.T) {
	resp := invokeMCRStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_port", v1MCRState())

	// Should be a no-op (skipped).
	assert.False(t, resp.Diagnostics.HasError())
	assert.True(t, resp.TargetState.Raw.IsNull(), "expected target state to remain null for wrong type")
}

func TestMoveState_MCR_NilSourceState(t *testing.T) {
	// A nil SourceState means the framework could not decode the source raw
	// state against the source schema; the mover must error, not panic.
	resp := invokeMCRStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mcr", nil)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Unable to migrate V1 state")
}
