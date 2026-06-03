package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper to build a minimal valid types.Object for vxcAEndConfigModel.
func newAEndObject(t *testing.T, m *vxcAEndConfigModel) types.Object {
	t.Helper()
	ctx := context.Background()
	if m.VrouterPartnerConfig.IsNull() || m.VrouterPartnerConfig.IsUnknown() {
		m.VrouterPartnerConfig = types.ObjectNull(vxcPartnerConfigVrouterAttrs)
	}
	obj, diags := types.ObjectValueFrom(ctx, vxcAEndConfigAttrs, m)
	require.False(t, diags.HasError(), "failed to build a_end object: %v", diags)
	return obj
}

// helper to build a minimal valid types.Object for vxcBEndConfigModel.
func newBEndObject(t *testing.T, m *vxcBEndConfigModel) types.Object {
	t.Helper()
	ctx := context.Background()
	nullPartners(m)
	obj, diags := types.ObjectValueFrom(ctx, vxcBEndConfigAttrs, m)
	require.False(t, diags.HasError(), "failed to build b_end object: %v", diags)
	return obj
}

// nullPartners sets all partner config fields to null if not already present.
func nullPartners(m *vxcBEndConfigModel) {
	if !isPresent(m.AWSPartnerConfig) {
		m.AWSPartnerConfig = types.ObjectNull(vxcPartnerConfigAWSAttrs)
	}
	if !isPresent(m.AzurePartnerConfig) {
		m.AzurePartnerConfig = types.ObjectNull(vxcPartnerConfigAzureAttrs)
	}
	if !isPresent(m.GooglePartnerConfig) {
		m.GooglePartnerConfig = types.ObjectNull(vxcPartnerConfigGoogleAttrs)
	}
	if !isPresent(m.OraclePartnerConfig) {
		m.OraclePartnerConfig = types.ObjectNull(vxcPartnerConfigOracleAttrs)
	}
	if !isPresent(m.IBMPartnerConfig) {
		m.IBMPartnerConfig = types.ObjectNull(vxcPartnerConfigIbmAttrs)
	}
	if !isPresent(m.VrouterPartnerConfig) {
		m.VrouterPartnerConfig = types.ObjectNull(vxcPartnerConfigVrouterAttrs)
	}
	if m.Transit.IsUnknown() {
		m.Transit = types.BoolNull()
	}
}

// ---------------------------------------------------------------------------
// TestFromAPIVXC_Basic — fully populated VXC, nil plan (Read path)
// ---------------------------------------------------------------------------
func TestFromAPIVXC_Basic(t *testing.T) {
	ctx := context.Background()
	apiVXC := &megaport.VXC{
		UID:                "vxc-uid-123",
		Name:               "Test VXC",
		ServiceID:          42,
		RateLimit:          1000,
		DistanceBand:       "ZONE",
		CreatedBy:          "user@example.com",
		ContractTermMonths: 12,
		CompanyUID:         "company-uid-456",
		Shutdown:           false,
		CostCentre:         "CC-100",
		AEndConfiguration: megaport.VXCEndConfiguration{
			UID:                   "a-end-uid",
			VLAN:                  100,
			InnerVLAN:             10,
			NetworkInterfaceIndex: 0,
		},
		BEndConfiguration: megaport.VXCEndConfiguration{
			UID:                   "b-end-uid",
			VLAN:                  200,
			InnerVLAN:             20,
			NetworkInterfaceIndex: 1,
		},
		AttributeTags: map[string]string{"account": "prod"},
	}
	tags := map[string]string{"env": "test"}

	model := &vxcResourceModel{}
	diags := model.fromAPIVXC(ctx, apiVXC, tags, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	assert.Equal(t, "vxc-uid-123", model.UID.ValueString())
	assert.Equal(t, "Test VXC", model.Name.ValueString())
	assert.Equal(t, int64(42), model.ServiceID.ValueInt64())
	assert.Equal(t, int64(1000), model.RateLimit.ValueInt64())
	assert.Equal(t, "ZONE", model.DistanceBand.ValueString())
	assert.Equal(t, "user@example.com", model.CreatedBy.ValueString())
	assert.Equal(t, int64(12), model.ContractTermMonths.ValueInt64())
	assert.Equal(t, "company-uid-456", model.CompanyUID.ValueString())
	assert.False(t, model.Shutdown.ValueBool())
	assert.Equal(t, "CC-100", model.CostCentre.ValueString())

	// Verify A-End
	require.False(t, model.AEndConfiguration.IsNull())
	aEnd := &vxcAEndConfigModel{}
	diags = model.AEndConfiguration.As(ctx, aEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, "a-end-uid", aEnd.AssignedProductUID.ValueString())
	assert.Equal(t, int64(100), aEnd.VLAN.ValueInt64())
	assert.Equal(t, int64(10), aEnd.InnerVLAN.ValueInt64())

	// Verify B-End
	require.False(t, model.BEndConfiguration.IsNull())
	bEnd := &vxcBEndConfigModel{}
	diags = model.BEndConfiguration.As(ctx, bEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, "b-end-uid", bEnd.AssignedProductUID.ValueString())
	assert.Equal(t, int64(200), bEnd.VLAN.ValueInt64())
	assert.Equal(t, int64(20), bEnd.InnerVLAN.ValueInt64())
	assert.Equal(t, int64(1), bEnd.NetworkInterfaceIndex.ValueInt64())

	// Tags
	assert.False(t, model.AttributeTags.IsNull())
	assert.False(t, model.ResourceTags.IsNull())
}

// ---------------------------------------------------------------------------
// TestFromAPIVXC_MinimalFields — only required fields, zero-value optional
// ---------------------------------------------------------------------------
func TestFromAPIVXC_MinimalFields(t *testing.T) {
	ctx := context.Background()
	apiVXC := &megaport.VXC{
		UID:  "vxc-min",
		Name: "Minimal",
		AEndConfiguration: megaport.VXCEndConfiguration{
			UID: "a-uid",
		},
		BEndConfiguration: megaport.VXCEndConfiguration{
			UID: "b-uid",
		},
	}

	model := &vxcResourceModel{}
	diags := model.fromAPIVXC(ctx, apiVXC, nil, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	assert.Equal(t, "vxc-min", model.UID.ValueString())
	assert.Equal(t, "Minimal", model.Name.ValueString())
	assert.Equal(t, int64(0), model.RateLimit.ValueInt64())
	assert.Equal(t, int64(0), model.ContractTermMonths.ValueInt64())
	assert.Equal(t, "", model.CostCentre.ValueString())
	assert.False(t, model.Shutdown.ValueBool())

	// VLAN should be null when API returns 0 and there's no state/plan
	aEnd := &vxcAEndConfigModel{}
	diags = model.AEndConfiguration.As(ctx, aEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.True(t, aEnd.VLAN.IsNull(), "VLAN should be null when API returns 0")
	assert.True(t, aEnd.InnerVLAN.IsNull(), "InnerVLAN should be null when API returns 0")
}

// ---------------------------------------------------------------------------
// TestFromAPIVXC_WithTags — test attribute tags and resource tags mapping
// ---------------------------------------------------------------------------
func TestFromAPIVXC_WithTags(t *testing.T) {
	ctx := context.Background()
	apiVXC := &megaport.VXC{
		UID:  "vxc-tags",
		Name: "Tags Test",
		AEndConfiguration: megaport.VXCEndConfiguration{
			UID: "a-uid",
		},
		BEndConfiguration: megaport.VXCEndConfiguration{
			UID: "b-uid",
		},
		AttributeTags: map[string]string{
			"account": "prod",
			"region":  "au",
		},
	}
	tags := map[string]string{
		"env":  "staging",
		"team": "platform",
	}

	model := &vxcResourceModel{}
	diags := model.fromAPIVXC(ctx, apiVXC, tags, nil)
	require.False(t, diags.HasError())

	// Attribute tags
	require.False(t, model.AttributeTags.IsNull())
	attrTags := map[string]string{}
	diags = model.AttributeTags.ElementsAs(ctx, &attrTags, false)
	require.False(t, diags.HasError())
	assert.Equal(t, "prod", attrTags["account"])
	assert.Equal(t, "au", attrTags["region"])

	// Resource tags
	require.False(t, model.ResourceTags.IsNull())
	resTags := map[string]string{}
	diags = model.ResourceTags.ElementsAs(ctx, &resTags, false)
	require.False(t, diags.HasError())
	assert.Equal(t, "staging", resTags["env"])
	assert.Equal(t, "platform", resTags["team"])
}

// ---------------------------------------------------------------------------
// TestFromAPIVXC_EmptyTags — empty tags map to null
// ---------------------------------------------------------------------------
func TestFromAPIVXC_EmptyTags(t *testing.T) {
	ctx := context.Background()
	apiVXC := &megaport.VXC{
		UID:           "vxc-empty-tags",
		Name:          "Empty Tags",
		AttributeTags: nil,
		AEndConfiguration: megaport.VXCEndConfiguration{
			UID: "a-uid",
		},
		BEndConfiguration: megaport.VXCEndConfiguration{
			UID: "b-uid",
		},
	}

	model := &vxcResourceModel{}
	diags := model.fromAPIVXC(ctx, apiVXC, map[string]string{}, nil)
	require.False(t, diags.HasError())

	assert.True(t, model.AttributeTags.IsNull(), "AttributeTags should be null when API returns nil")
	assert.True(t, model.ResourceTags.IsNull(), "ResourceTags should be null when empty map")
}

// ---------------------------------------------------------------------------
// TestFromAPIVXC_WithPlan — non-nil plan (Create/Update path), verify plan VLANs preserved
// ---------------------------------------------------------------------------
func TestFromAPIVXC_WithPlan(t *testing.T) {
	ctx := context.Background()
	apiVXC := &megaport.VXC{
		UID:       "vxc-plan",
		Name:      "Plan Test",
		RateLimit: 500,
		AEndConfiguration: megaport.VXCEndConfiguration{
			UID:  "a-uid",
			VLAN: 999, // API returns different VLAN
		},
		BEndConfiguration: megaport.VXCEndConfiguration{
			UID:  "b-uid",
			VLAN: 888,
		},
	}

	// Plan has VLANs that should take priority over API values
	plan := &vxcResourceModel{
		AEndConfiguration: newAEndObject(t, &vxcAEndConfigModel{
			ProductUID:            types.StringValue("plan-a-uid"),
			AssignedProductUID:    types.StringValue(""),
			VLAN:                  types.Int64Value(100),
			InnerVLAN:             types.Int64Null(),
			NetworkInterfaceIndex: types.Int64Value(2),
			VrouterPartnerConfig:  types.ObjectNull(vxcPartnerConfigVrouterAttrs),
		}),
		BEndConfiguration: newBEndObject(t, &vxcBEndConfigModel{
			ProductUID:            types.StringValue("plan-b-uid"),
			AssignedProductUID:    types.StringValue(""),
			VLAN:                  types.Int64Value(200),
			InnerVLAN:             types.Int64Null(),
			NetworkInterfaceIndex: types.Int64Value(3),
			Transit:               types.BoolNull(),
		}),
	}

	model := &vxcResourceModel{}
	diags := model.fromAPIVXC(ctx, apiVXC, nil, plan)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	// Verify plan VLANs take priority over API values
	aEnd := &vxcAEndConfigModel{}
	diags = model.AEndConfiguration.As(ctx, aEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, int64(100), aEnd.VLAN.ValueInt64(), "A-end VLAN should come from plan, not API")
	assert.Equal(t, "plan-a-uid", aEnd.ProductUID.ValueString(), "A-end ProductUID should come from plan")
	assert.Equal(t, int64(2), aEnd.NetworkInterfaceIndex.ValueInt64())

	bEnd := &vxcBEndConfigModel{}
	diags = model.BEndConfiguration.As(ctx, bEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, int64(200), bEnd.VLAN.ValueInt64(), "B-end VLAN should come from plan, not API")
	assert.Equal(t, "plan-b-uid", bEnd.ProductUID.ValueString(), "B-end ProductUID should come from plan")
	assert.Equal(t, int64(3), bEnd.NetworkInterfaceIndex.ValueInt64())
}

// ---------------------------------------------------------------------------
// TestBuildAEndConfig_Basic — basic A-end mapping with nil plan
// ---------------------------------------------------------------------------
func TestBuildAEndConfig_Basic(t *testing.T) {
	ctx := context.Background()
	apiVXC := &megaport.VXC{
		AEndConfiguration: megaport.VXCEndConfiguration{
			UID:                   "a-end-assigned",
			VLAN:                  150,
			InnerVLAN:             15,
			NetworkInterfaceIndex: 0,
		},
	}

	model := &vxcResourceModel{}
	diags := model.buildAEndConfig(ctx, apiVXC, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	aEnd := &vxcAEndConfigModel{}
	diags = model.AEndConfiguration.As(ctx, aEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())

	assert.Equal(t, "a-end-assigned", aEnd.AssignedProductUID.ValueString())
	assert.Equal(t, "", aEnd.ProductUID.ValueString()) // no plan or state, defaults to empty
	assert.Equal(t, int64(150), aEnd.VLAN.ValueInt64())
	assert.Equal(t, int64(15), aEnd.InnerVLAN.ValueInt64())
	assert.Equal(t, int64(0), aEnd.NetworkInterfaceIndex.ValueInt64())
}

// ---------------------------------------------------------------------------
// TestBuildAEndConfig_WithPlan — A-end with plan values (VLAN from plan takes priority)
// ---------------------------------------------------------------------------
func TestBuildAEndConfig_WithPlan(t *testing.T) {
	ctx := context.Background()
	apiVXC := &megaport.VXC{
		AEndConfiguration: megaport.VXCEndConfiguration{
			UID:                   "a-end-assigned",
			VLAN:                  999, // API returns different value
			InnerVLAN:             50,
			NetworkInterfaceIndex: 0,
		},
	}

	plan := &vxcResourceModel{
		AEndConfiguration: newAEndObject(t, &vxcAEndConfigModel{
			ProductUID:            types.StringValue("plan-product-uid"),
			AssignedProductUID:    types.StringValue(""),
			VLAN:                  types.Int64Value(100), // plan value should win
			InnerVLAN:             types.Int64Value(10),
			NetworkInterfaceIndex: types.Int64Value(2),
			VrouterPartnerConfig:  types.ObjectNull(vxcPartnerConfigVrouterAttrs),
		}),
	}

	model := &vxcResourceModel{}
	diags := model.buildAEndConfig(ctx, apiVXC, plan)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	aEnd := &vxcAEndConfigModel{}
	diags = model.AEndConfiguration.As(ctx, aEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())

	assert.Equal(t, "a-end-assigned", aEnd.AssignedProductUID.ValueString())
	assert.Equal(t, "plan-product-uid", aEnd.ProductUID.ValueString())
	assert.Equal(t, int64(100), aEnd.VLAN.ValueInt64(), "plan VLAN should take priority")
	assert.Equal(t, int64(50), aEnd.InnerVLAN.ValueInt64(), "API InnerVLAN != 0 so API wins")
	assert.Equal(t, int64(2), aEnd.NetworkInterfaceIndex.ValueInt64())
}

// ---------------------------------------------------------------------------
// TestBuildBEndConfig_Basic — basic B-end mapping with nil plan
// ---------------------------------------------------------------------------
func TestBuildBEndConfig_Basic(t *testing.T) {
	ctx := context.Background()
	apiVXC := &megaport.VXC{
		BEndConfiguration: megaport.VXCEndConfiguration{
			UID:                   "b-end-assigned",
			VLAN:                  250,
			InnerVLAN:             25,
			NetworkInterfaceIndex: 1,
		},
	}

	model := &vxcResourceModel{}
	diags := model.buildBEndConfig(ctx, apiVXC, nil)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	bEnd := &vxcBEndConfigModel{}
	diags = model.BEndConfiguration.As(ctx, bEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())

	assert.Equal(t, "b-end-assigned", bEnd.AssignedProductUID.ValueString())
	assert.Equal(t, "", bEnd.ProductUID.ValueString())
	assert.Equal(t, int64(250), bEnd.VLAN.ValueInt64())
	assert.Equal(t, int64(25), bEnd.InnerVLAN.ValueInt64())
	assert.Equal(t, int64(1), bEnd.NetworkInterfaceIndex.ValueInt64())

	// All partner configs should be null
	assert.True(t, bEnd.AWSPartnerConfig.IsNull())
	assert.True(t, bEnd.AzurePartnerConfig.IsNull())
	assert.True(t, bEnd.GooglePartnerConfig.IsNull())
	assert.True(t, bEnd.OraclePartnerConfig.IsNull())
	assert.True(t, bEnd.IBMPartnerConfig.IsNull())
	assert.True(t, bEnd.VrouterPartnerConfig.IsNull())
}

// ---------------------------------------------------------------------------
// TestBuildBEndConfig_WithPlan — B-end with plan values
// ---------------------------------------------------------------------------
func TestBuildBEndConfig_WithPlan(t *testing.T) {
	ctx := context.Background()
	apiVXC := &megaport.VXC{
		BEndConfiguration: megaport.VXCEndConfiguration{
			UID:                   "b-end-assigned",
			VLAN:                  999,
			InnerVLAN:             0,
			NetworkInterfaceIndex: 0,
		},
	}

	plan := &vxcResourceModel{
		BEndConfiguration: newBEndObject(t, &vxcBEndConfigModel{
			ProductUID:            types.StringValue("plan-b-uid"),
			AssignedProductUID:    types.StringValue(""),
			VLAN:                  types.Int64Value(300),
			InnerVLAN:             types.Int64Value(-1), // untagged
			NetworkInterfaceIndex: types.Int64Value(4),
			Transit:               types.BoolNull(),
		}),
	}

	model := &vxcResourceModel{}
	diags := model.buildBEndConfig(ctx, apiVXC, plan)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

	bEnd := &vxcBEndConfigModel{}
	diags = model.BEndConfiguration.As(ctx, bEnd, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())

	assert.Equal(t, "b-end-assigned", bEnd.AssignedProductUID.ValueString())
	assert.Equal(t, "plan-b-uid", bEnd.ProductUID.ValueString())
	assert.Equal(t, int64(300), bEnd.VLAN.ValueInt64(), "plan VLAN should take priority")
	// InnerVLAN: API returned 0, plan has -1 (untagged) — inner_vlan logic: API==0 + state has -1 => -1
	assert.Equal(t, int64(-1), bEnd.InnerVLAN.ValueInt64(), "InnerVLAN -1 from plan should be preserved")
	assert.Equal(t, int64(4), bEnd.NetworkInterfaceIndex.ValueInt64())
}

// ---------------------------------------------------------------------------
// TestMapVXCTags_Full — both attribute and resource tags populated
// ---------------------------------------------------------------------------
func TestMapVXCTags_Full(t *testing.T) {
	ctx := context.Background()
	apiVXC := &megaport.VXC{
		AttributeTags: map[string]string{
			"account": "production",
			"tier":    "gold",
		},
	}
	tags := map[string]string{
		"env":    "prod",
		"region": "ap-southeast-2",
	}

	model := &vxcResourceModel{}
	diags := model.mapVXCTags(ctx, apiVXC, tags)
	require.False(t, diags.HasError())

	// Attribute tags
	require.False(t, model.AttributeTags.IsNull())
	attrMap := map[string]string{}
	diags = model.AttributeTags.ElementsAs(ctx, &attrMap, false)
	require.False(t, diags.HasError())
	assert.Equal(t, "production", attrMap["account"])
	assert.Equal(t, "gold", attrMap["tier"])

	// Resource tags
	require.False(t, model.ResourceTags.IsNull())
	resMap := map[string]string{}
	diags = model.ResourceTags.ElementsAs(ctx, &resMap, false)
	require.False(t, diags.HasError())
	assert.Equal(t, "prod", resMap["env"])
	assert.Equal(t, "ap-southeast-2", resMap["region"])
}

// ---------------------------------------------------------------------------
// TestMapVXCTags_Empty — empty maps result in null values
// ---------------------------------------------------------------------------
func TestMapVXCTags_Empty(t *testing.T) {
	ctx := context.Background()
	apiVXC := &megaport.VXC{
		AttributeTags: map[string]string{}, // empty, but non-nil
	}

	model := &vxcResourceModel{}
	diags := model.mapVXCTags(ctx, apiVXC, map[string]string{})
	require.False(t, diags.HasError())

	// Non-nil empty map for AttributeTags -> still mapped (non-null, but empty)
	assert.False(t, model.AttributeTags.IsNull(), "non-nil empty map should produce non-null AttributeTags")
	assert.Equal(t, 0, len(model.AttributeTags.Elements()))

	// Empty resource tags -> null
	assert.True(t, model.ResourceTags.IsNull(), "empty resource tags map should produce null")
}

// ---------------------------------------------------------------------------
// TestMapVXCTags_NilTags — nil resource tags
// ---------------------------------------------------------------------------
func TestMapVXCTags_NilTags(t *testing.T) {
	ctx := context.Background()
	apiVXC := &megaport.VXC{
		AttributeTags: nil,
	}

	model := &vxcResourceModel{}
	diags := model.mapVXCTags(ctx, apiVXC, nil)
	require.False(t, diags.HasError())

	assert.True(t, model.AttributeTags.IsNull(), "nil AttributeTags should produce null")
	assert.True(t, model.ResourceTags.IsNull(), "nil resource tags should produce null")
}

// ---------------------------------------------------------------------------
// TestIsPresent — table-driven: null->false, unknown->false, known->true
// ---------------------------------------------------------------------------
func TestIsPresent(t *testing.T) {
	tests := []struct {
		name     string
		value    attr.Value
		expected bool
	}{
		{"null string", types.StringNull(), false},
		{"unknown string", types.StringUnknown(), false},
		{"known string", types.StringValue("hello"), true},
		{"empty string", types.StringValue(""), true},
		{"null int64", types.Int64Null(), false},
		{"unknown int64", types.Int64Unknown(), false},
		{"known int64", types.Int64Value(42), true},
		{"zero int64", types.Int64Value(0), true},
		{"null bool", types.BoolNull(), false},
		{"unknown bool", types.BoolUnknown(), false},
		{"known bool true", types.BoolValue(true), true},
		{"known bool false", types.BoolValue(false), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isPresent(tt.value))
		})
	}
}

// ---------------------------------------------------------------------------
// TestSupportVLANUpdates — table-driven for all partner types
// ---------------------------------------------------------------------------
func TestSupportVLANUpdates(t *testing.T) {
	tests := []struct {
		partnerType string
		expected    bool
	}{
		{"aws", false},
		{"transit", false},
		{"azure", true},
		{"google", true},
		{"oracle", true},
		{"ibm", true},
		{"vrouter", true},
		{"", true},
		{"unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.partnerType, func(t *testing.T) {
			assert.Equal(t, tt.expected, supportVLANUpdates(tt.partnerType))
		})
	}
}
