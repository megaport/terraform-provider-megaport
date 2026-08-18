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

// The V1 to V2 state upgrade needs no upgrader code: the framework's
// UpgradeResourceState passthrough decodes prior state against the current
// schema with IgnoreUndefinedAttributes, so removed attributes are dropped.
// These tests replicate that passthrough to prove real V1 state, including
// the deeply nested a_end/b_end/partner objects, still decodes against the
// V2 schema (i.e. no kept attribute changed type).

// v1VXCState returns a complete V1 VXC state JSON with all fields that
// existed in V1 (including those removed in V2).
func v1VXCState() []byte {
	state := map[string]interface{}{
		// Fields kept in V2
		"product_uid":          "vxc-uid-123",
		"service_id":           42,
		"product_name":         "My VXC",
		"rate_limit":           1000,
		"distance_band":        "ZONE",
		"promo_code":           "PROMO1",
		"service_key":          "svc-key",
		"contract_term_months": 12,
		"company_uid":          "company-xyz",
		"attribute_tags":       map[string]string{"env": "staging"},
		"cost_centre":          "CC-001",
		"shutdown":             false,
		"a_end": map[string]interface{}{
			"owner_uid":             "owner-a",
			"requested_product_uid": "port-a-requested",
			"current_product_uid":   "port-a-current",
			"product_name":          "Port A",
			"location_id":           7,
			"location":              "Sydney",
			"ordered_vlan":          100,
			"vlan":                  100,
			"inner_vlan":            nil,
			"vnic_index":            0,
			"secondary_name":        "a-secondary",
		},
		"b_end": map[string]interface{}{
			"owner_uid":             "owner-b",
			"requested_product_uid": "port-b-requested",
			"current_product_uid":   "port-b-current",
			"product_name":          "Port B",
			"location_id":           8,
			"location":              "Melbourne",
			"ordered_vlan":          nil,
			"vlan":                  200,
			"inner_vlan":            300,
			"vnic_index":            nil,
			"secondary_name":        nil,
		},
		"a_end_partner_config": nil,
		"b_end_partner_config": map[string]interface{}{
			"partner": "aws",
			"aws_config": map[string]interface{}{
				"connect_type":        "AWS",
				"type":                "private",
				"owner_account":       "123456789012",
				"asn":                 64512,
				"amazon_asn":          64513,
				"auth_key":            "authkey",
				"prefixes":            "10.0.0.0/24",
				"customer_ip_address": "10.0.0.1/30",
				"amazon_ip_address":   "10.0.0.2/30",
				"name":                "aws-vif",
			},
			"azure_config":         nil,
			"google_config":        nil,
			"oracle_config":        nil,
			"vrouter_config":       nil,
			"partner_a_end_config": nil,
			"ibm_config":           nil,
		},
		"csp_connections": []map[string]interface{}{
			{
				"connect_type":  "AWS",
				"resource_name": "b_csp_connection",
				"resource_type": "CSP_CONNECTION",
				"vlan":          200,
				"owner_account": "123456789012",
				"asn":           64512,
				"peer_asn":      64513,
				"type":          "private",
				"vif_id":        "dxvif-abc",
			},
		},
		"resource_tags": map[string]string{
			"env":  "staging",
			"team": "infra",
		},
		// Fields removed in V2
		"last_updated":        "2024-01-01T00:00:00Z",
		"product_id":          99,
		"product_type":        "VXC",
		"provisioning_status": "LIVE",
		"secondary_name":      "vxc-secondary",
		"usage_algorithm":     "some-algo",
		"created_by":          "user@example.com",
		"live_date":           "2024-01-02T00:00:00Z",
		"create_date":         "2024-01-01T00:00:00Z",
		"contract_start_date": "2024-01-01",
		"contract_end_date":   "2025-01-01",
		"company_name":        "My Company",
		"locked":              false,
		"admin_locked":        false,
		"cancelable":          true,
	}
	b, _ := json.Marshal(state)
	return b
}

// decodeV1VXCStateAgainstV2Schema mimics the framework's same-version
// UpgradeResourceState passthrough (fwserver/server_upgraderesourcestate.go).
func decodeV1VXCStateAgainstV2Schema(t *testing.T, rawJSON []byte) tfsdk.State {
	t.Helper()
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	(&vxcResource{}).Schema(ctx, resource.SchemaRequest{}, schemaResp)
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

func TestVXCStateUpgrade_V1ToV2(t *testing.T) {
	ctx := context.Background()

	state := decodeV1VXCStateAgainstV2Schema(t, v1VXCState())

	var model vxcResourceModel
	diags := state.Get(ctx, &model)
	require.False(t, diags.HasError(), "failed to read upgraded state: %v", diags)

	// Verify kept top-level fields survive the upgrade.
	assert.Equal(t, "vxc-uid-123", model.UID.ValueString())
	assert.Equal(t, int64(42), model.ServiceID.ValueInt64())
	assert.Equal(t, "My VXC", model.Name.ValueString())
	assert.Equal(t, int64(1000), model.RateLimit.ValueInt64())
	assert.Equal(t, "ZONE", model.DistanceBand.ValueString())
	assert.Equal(t, "PROMO1", model.PromoCode.ValueString())
	assert.Equal(t, "svc-key", model.ServiceKey.ValueString())
	assert.Equal(t, int64(12), model.ContractTermMonths.ValueInt64())
	assert.Equal(t, "company-xyz", model.CompanyUID.ValueString())
	assert.Equal(t, "CC-001", model.CostCentre.ValueString())
	assert.False(t, model.Shutdown.ValueBool())

	// Verify a_end nested object.
	require.False(t, model.AEndConfiguration.IsNull())
	var aEnd vxcEndConfigurationModel
	diags = model.AEndConfiguration.As(ctx, &aEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError(), "failed to read a_end: %v", diags)
	assert.Equal(t, "port-a-requested", aEnd.RequestedProductUID.ValueString())
	assert.Equal(t, "port-a-current", aEnd.CurrentProductUID.ValueString())
	assert.Equal(t, int64(100), aEnd.OrderedVLAN.ValueInt64())
	assert.Equal(t, int64(100), aEnd.VLAN.ValueInt64())
	assert.True(t, aEnd.InnerVLAN.IsNull())
	assert.Equal(t, int64(0), aEnd.NetworkInterfaceIndex.ValueInt64())
	assert.Equal(t, "a-secondary", aEnd.SecondaryName.ValueString())

	// Verify b_end nested object.
	require.False(t, model.BEndConfiguration.IsNull())
	var bEnd vxcEndConfigurationModel
	diags = model.BEndConfiguration.As(ctx, &bEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError(), "failed to read b_end: %v", diags)
	assert.Equal(t, "port-b-requested", bEnd.RequestedProductUID.ValueString())
	assert.Equal(t, int64(200), bEnd.VLAN.ValueInt64())
	assert.Equal(t, int64(300), bEnd.InnerVLAN.ValueInt64())
	assert.True(t, bEnd.OrderedVLAN.IsNull())

	// Verify partner configs carry over unchanged.
	assert.True(t, model.AEndPartnerConfig.IsNull())
	require.False(t, model.BEndPartnerConfig.IsNull())
	var bEndPartner vxcPartnerConfigurationModel
	diags = model.BEndPartnerConfig.As(ctx, &bEndPartner, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError(), "failed to read b_end_partner_config: %v", diags)
	assert.Equal(t, "aws", bEndPartner.Partner.ValueString())
	require.False(t, bEndPartner.AWSPartnerConfig.IsNull())
	var awsConfig vxcPartnerConfigAWSModel
	diags = bEndPartner.AWSPartnerConfig.As(ctx, &awsConfig, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError(), "failed to read aws_config: %v", diags)
	assert.Equal(t, "AWS", awsConfig.ConnectType.ValueString())
	assert.Equal(t, "private", awsConfig.Type.ValueString())
	assert.Equal(t, "123456789012", awsConfig.OwnerAccount.ValueString())
	assert.Equal(t, int64(64512), awsConfig.ASN.ValueInt64())
	assert.Equal(t, "10.0.0.1/30", awsConfig.CustomerIPAddress.ValueString())

	// Verify csp_connections list.
	require.False(t, model.CSPConnections.IsNull())
	cspElements := model.CSPConnections.Elements()
	require.Len(t, cspElements, 1)
	cspObj, ok := cspElements[0].(types.Object)
	require.True(t, ok)
	var csp cspConnectionModel
	diags = cspObj.As(ctx, &csp, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError(), "failed to read csp_connection: %v", diags)
	assert.Equal(t, "AWS", csp.ConnectType.ValueString())
	assert.Equal(t, int64(200), csp.VLAN.ValueInt64())
	assert.Equal(t, "dxvif-abc", csp.VIFID.ValueString())

	// Verify resource tags.
	require.False(t, model.ResourceTags.IsNull())
	tagElements := model.ResourceTags.Elements()
	require.Len(t, tagElements, 2)
	envTag, ok := tagElements["env"].(types.String)
	require.True(t, ok)
	assert.Equal(t, "staging", envTag.ValueString())

	// Verify attribute tags.
	require.False(t, model.AttributeTags.IsNull())
	require.Len(t, model.AttributeTags.Elements(), 1)
}

func TestVXCStateUpgrade_V1ToV2_NilOptionals(t *testing.T) {
	ctx := context.Background()

	// V1 state with null or missing optional fields.
	state := map[string]interface{}{
		"product_uid":          "vxc-nil-test",
		"product_name":         "Nil VXC",
		"rate_limit":           500,
		"contract_term_months": 1,
		"promo_code":           nil,
		"cost_centre":          nil,
		"a_end":                nil,
		"b_end":                nil,
		"a_end_partner_config": nil,
		"b_end_partner_config": nil,
		"csp_connections":      nil,
		"resource_tags":        nil,
		// V1-only fields
		"last_updated":        nil,
		"product_id":          nil,
		"provisioning_status": nil,
		"created_by":          nil,
	}
	rawJSON, err := json.Marshal(state)
	require.NoError(t, err)

	upgraded := decodeV1VXCStateAgainstV2Schema(t, rawJSON)

	var model vxcResourceModel
	diags := upgraded.Get(ctx, &model)
	require.False(t, diags.HasError(), "failed to read upgraded state: %v", diags)

	assert.Equal(t, "vxc-nil-test", model.UID.ValueString())
	assert.Equal(t, "Nil VXC", model.Name.ValueString())
	assert.Equal(t, int64(500), model.RateLimit.ValueInt64())
	assert.Equal(t, int64(1), model.ContractTermMonths.ValueInt64())
	assert.True(t, model.PromoCode.IsNull())
	assert.True(t, model.CostCentre.IsNull())
	assert.True(t, model.ServiceID.IsNull())
	assert.True(t, model.AEndConfiguration.IsNull())
	assert.True(t, model.BEndConfiguration.IsNull())
	assert.True(t, model.AEndPartnerConfig.IsNull())
	assert.True(t, model.BEndPartnerConfig.IsNull())
	assert.True(t, model.CSPConnections.IsNull())
	assert.True(t, model.ResourceTags.IsNull())
}
