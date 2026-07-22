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

// The V1 to V2 state upgrade needs no upgrader code: the framework's
// UpgradeResourceState passthrough decodes prior state against the current
// schema with IgnoreUndefinedAttributes, so removed attributes are dropped.
// These tests replicate that passthrough to prove real V1 state still
// decodes against the V2 schema (i.e. no kept attribute changed type).

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

// decodePortV1StateAgainstV2Schema mimics the framework's same-version
// UpgradeResourceState passthrough (fwserver/server_upgraderesourcestate.go)
// against the given resource's current schema.
func decodePortV1StateAgainstV2Schema(t *testing.T, r resource.Resource, rawJSON []byte) tfsdk.State {
	t.Helper()
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	rawState := &tfprotov6.RawState{JSON: rawJSON}
	rawValue, err := rawState.UnmarshalWithOpts(
		schemaResp.Schema.Type().TerraformType(ctx),
		tfprotov6.UnmarshalOpts{
			ValueFromJSONOpts: tftypes.ValueFromJSONOpts{
				IgnoreUndefinedAttributes: true,
			},
		},
	)
	require.NoError(t, err, "V1 state failed to decode against the V2 schema")

	return tfsdk.State{
		Raw:    rawValue,
		Schema: schemaResp.Schema,
	}
}

func TestPortStateUpgrade_V1ToV2(t *testing.T) {
	ctx := context.Background()

	state := decodePortV1StateAgainstV2Schema(t, &portResource{}, v1PortState())

	var model singlePortResourceModel
	diags := state.Get(ctx, &model)
	require.False(t, diags.HasError(), "failed to read upgraded state: %v", diags)

	// Verify kept fields survive the upgrade.
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

func TestPortStateUpgrade_V1ToV2_NilOptionals(t *testing.T) {
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

	upgraded := decodePortV1StateAgainstV2Schema(t, &portResource{}, rawJSON)

	var model singlePortResourceModel
	diags := upgraded.Get(ctx, &model)
	require.False(t, diags.HasError(), "failed to read upgraded state: %v", diags)

	assert.Equal(t, "uid-nil-test", model.UID.ValueString())
	assert.Equal(t, "Nil Port", model.Name.ValueString())
	assert.Equal(t, int64(1000), model.PortSpeed.ValueInt64())
	assert.True(t, model.CostCentre.IsNull())
	assert.True(t, model.DiversityZone.IsNull())
	assert.True(t, model.PromoCode.IsNull())
	assert.True(t, model.Resources.IsNull())
	assert.True(t, model.ResourceTags.IsNull())
}

func TestLagPortStateUpgrade_V1ToV2(t *testing.T) {
	ctx := context.Background()

	state := decodePortV1StateAgainstV2Schema(t, &lagPortResource{}, v1LagPortState())

	var model lagPortResourceModel
	diags := state.Get(ctx, &model)
	require.False(t, diags.HasError(), "failed to read upgraded state: %v", diags)

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
