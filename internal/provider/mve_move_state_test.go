package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// v1MVEState returns a complete V1 MVE state JSON with all fields that existed
// in V1, including the removed legacy fields, the old union vendor_config block,
// and the removed vnics.vlan attribute.
func v1MVEState() []byte {
	state := map[string]interface{}{
		// Fields kept in V2
		"product_uid":            "mve-uid-123",
		"product_name":           "My MVE",
		"location_id":            7,
		"marketplace_visibility": false,
		"company_uid":            "company-xyz",
		"contract_term_months":   12,
		"promo_code":             "PROMO1",
		"cost_centre":            "CC-001",
		"diversity_zone":         "red",
		"vendor":                 "CISCO",
		"mve_size":               "SMALL",
		"attribute_tags":         map[string]string{"env": "staging"},
		"resource_tags":          map[string]string{"team": "infra"},
		"vnics": []map[string]interface{}{
			{"description": "Data Plane", "vlan": 100},
			{"description": "Management", "vlan": 200},
		},
		// Old union vendor_config block (removed in V2, replaced by per-vendor blocks)
		"vendor_config": map[string]interface{}{
			"vendor":               "cisco",
			"image_id":             42,
			"product_size":         "SMALL",
			"mve_label":            "cisco-label",
			"admin_ssh_public_key": "ssh-rsa AAAA",
			"ssh_public_key":       "ssh-rsa BBBB",
			"admin_password":       "sup3rs3cret!",
			"manage_locally":       true,
			"cloud_init":           "base64data",
			"fmc_ip_address":       "10.0.0.1",
			"fmc_registration_key": "regkey",
			"fmc_nat_id":           "natid",
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
		"product_type":        "MVE",
		"usage_algorithm":     "some-algo",
		"contract_start_date": "2024-01-01",
		"contract_end_date":   "2025-01-01",
		"vxc_permitted":       true,
		"vxc_auto_approval":   false,
		"secondary_name":      "mve-secondary",
		"company_name":        "My Company",
		"virtual":             true,
		"buyout_port":         false,
		"locked":              false,
		"admin_locked":        false,
		"cancelable":          true,
	}
	b, _ := json.Marshal(state)
	return b
}

// invokeMVEStateMover decodes the raw V1 state against the mover's SourceSchema
// exactly as the framework server does, then runs the mover.
func invokeMVEStateMover(t *testing.T, providerAddr, typeName string, rawJSON []byte) *resource.MoveStateResponse {
	t.Helper()
	ctx := context.Background()
	r := &mveResource{}

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

func TestMoveState_MVE_V1ToV2(t *testing.T) {
	ctx := context.Background()

	resp := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", v1MVEState())
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)

	var model mveResourceModel
	diags := resp.TargetState.Get(ctx, &model)
	require.False(t, diags.HasError(), "failed to read target state: %v", diags)

	// Kept fields migrate unchanged.
	assert.Equal(t, "mve-uid-123", model.UID.ValueString())
	assert.Equal(t, "My MVE", model.Name.ValueString())
	assert.Equal(t, int64(7), model.LocationID.ValueInt64())
	assert.False(t, model.MarketplaceVisibility.ValueBool())
	assert.Equal(t, "company-xyz", model.CompanyUID.ValueString())
	assert.Equal(t, int64(12), model.ContractTermMonths.ValueInt64())
	assert.Equal(t, "PROMO1", model.PromoCode.ValueString())
	assert.Equal(t, "CC-001", model.CostCentre.ValueString())
	assert.Equal(t, "red", model.DiversityZone.ValueString())
	assert.Equal(t, "CISCO", model.Vendor.ValueString())
	assert.Equal(t, "SMALL", model.Size.ValueString())

	// The old union vendor_config is dropped; per-vendor blocks decode to null.
	assert.True(t, allVendorConfigsNull(model), "expected all per-vendor config blocks to be null after migration")

	// vnics carry over, but the removed vlan attribute is dropped.
	require.False(t, model.NetworkInterfaces.IsNull())
	vnicElements := model.NetworkInterfaces.Elements()
	require.Len(t, vnicElements, 2)
	vnicObj, ok := vnicElements[0].(types.Object)
	require.True(t, ok)
	var vnic mveNetworkInterfaceModel
	diags = vnicObj.As(ctx, &vnic, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError(), "failed to read vnic: %v", diags)
	assert.Equal(t, "Data Plane", vnic.Description.ValueString())
	vnicAttrKeys := vnicObj.Attributes()
	_, hasVLAN := vnicAttrKeys["vlan"]
	assert.False(t, hasVLAN, "expected vnics.vlan to be dropped after migration")

	// Resource tags carry over.
	require.False(t, model.ResourceTags.IsNull())
	require.Len(t, model.ResourceTags.Elements(), 1)
}

func TestMoveState_MVE_V1ToV2_NilOptionals(t *testing.T) {
	ctx := context.Background()

	state := map[string]interface{}{
		"product_uid":          "mve-nil-test",
		"product_name":         "Nil MVE",
		"location_id":          8,
		"contract_term_months": 1,
		"vendor":               "ARUBA",
		"mve_size":             "MEDIUM",
		"promo_code":           nil,
		"cost_centre":          nil,
		"diversity_zone":       nil,
		"vnics":                nil,
		"resource_tags":        nil,
		"vendor_config":        nil,
		// V1-only fields
		"last_updated":        nil,
		"product_id":          nil,
		"provisioning_status": nil,
		"created_by":          nil,
	}
	rawJSON, err := json.Marshal(state)
	require.NoError(t, err)

	resp := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)

	var model mveResourceModel
	diags := resp.TargetState.Get(ctx, &model)
	require.False(t, diags.HasError(), "failed to read target state: %v", diags)

	assert.Equal(t, "mve-nil-test", model.UID.ValueString())
	assert.Equal(t, "Nil MVE", model.Name.ValueString())
	assert.Equal(t, int64(8), model.LocationID.ValueInt64())
	assert.Equal(t, "ARUBA", model.Vendor.ValueString())
	assert.True(t, model.PromoCode.IsNull())
	assert.True(t, model.CostCentre.IsNull())
	assert.True(t, model.DiversityZone.IsNull())
	assert.True(t, model.NetworkInterfaces.IsNull())
	assert.True(t, model.ResourceTags.IsNull())
	assert.True(t, allVendorConfigsNull(model))
}

func TestMoveState_MVE_WrongProvider(t *testing.T) {
	resp := invokeMVEStateMover(t, "registry.terraform.io/other/other", "megaport_mve", v1MVEState())

	assert.False(t, resp.Diagnostics.HasError())
	assert.True(t, resp.TargetState.Raw.IsNull(), "expected target state to remain null for wrong provider")
}

func TestMoveState_MVE_WrongType(t *testing.T) {
	resp := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_vxc", v1MVEState())

	assert.False(t, resp.Diagnostics.HasError())
	assert.True(t, resp.TargetState.Raw.IsNull(), "expected target state to remain null for wrong type")
}

func TestMoveState_MVE_NilSourceState(t *testing.T) {
	resp := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", nil)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Unable to migrate V1 state")
}
