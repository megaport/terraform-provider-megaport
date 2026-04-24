package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromAPIPortInterface_Full(t *testing.T) {
	ctx := context.Background()
	apiInterface := &megaport.PortInterface{
		Demarcation:  "Test Demarcation",
		Description:  "Test Description",
		ID:           42,
		LOATemplate:  "loa-template",
		Media:        "LR4",
		Name:         "interface-name",
		PortSpeed:    10000,
		ResourceName: "resource-name",
		ResourceType: "resource-type",
		Up:           1,
	}

	obj, diags := fromAPIPortInterface(ctx, apiInterface)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	require.False(t, obj.IsNull())
	require.False(t, obj.IsUnknown())

	var m portInterfaceModel
	diags = obj.As(ctx, &m, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError(), "unexpected diagnostics unmarshalling: %v", diags)

	assert.Equal(t, "Test Demarcation", m.Demarcation.ValueString())
	assert.Equal(t, int64(1), m.Up.ValueInt64())
}

func TestFromAPIPortInterface_NilInput(t *testing.T) {
	// fromAPIPortInterface takes a *PortInterface, not nil-safe by contract,
	// but we test with a zero-value struct to verify no panic.
	ctx := context.Background()
	apiInterface := &megaport.PortInterface{}

	obj, diags := fromAPIPortInterface(ctx, apiInterface)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	require.False(t, obj.IsNull())

	var m portInterfaceModel
	diags = obj.As(ctx, &m, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError(), "unexpected diagnostics unmarshalling: %v", diags)

	assert.Equal(t, "", m.Demarcation.ValueString())
	assert.Equal(t, int64(0), m.Up.ValueInt64())
}

func TestLagPortUIDsList_WithUIDs(t *testing.T) {
	uids := []string{"uid-1", "uid-2", "uid-3"}
	result, diags := lagPortUIDsList(uids)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	require.False(t, result.IsNull())
	require.False(t, result.IsUnknown())

	elements := result.Elements()
	require.Len(t, elements, 3)
	assert.Equal(t, "uid-1", elements[0].(types.String).ValueString())
	assert.Equal(t, "uid-2", elements[1].(types.String).ValueString())
	assert.Equal(t, "uid-3", elements[2].(types.String).ValueString())
}

func TestLagPortUIDsList_Empty(t *testing.T) {
	t.Run("nil slice", func(t *testing.T) {
		result, diags := lagPortUIDsList(nil)
		require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
		assert.True(t, result.IsNull(), "expected null list for nil input")
	})

	t.Run("empty slice", func(t *testing.T) {
		result, diags := lagPortUIDsList([]string{})
		require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
		assert.True(t, result.IsNull(), "expected null list for empty input")
	})
}

func TestResolvePortModifyParams(t *testing.T) {
	tests := []struct {
		name                     string
		planName, stateName      types.String
		planVis, stateVis        types.Bool
		planCostCentre           types.String
		planTerm, stateTerm      types.Int64
		wantName                 string
		wantVisibility           bool
		wantCostCentre           string
		wantContractTermMonths   *int
	}{
		{
			name:                   "no changes — all same",
			planName:               types.StringValue("port-1"),
			stateName:              types.StringValue("port-1"),
			planVis:                types.BoolValue(true),
			stateVis:               types.BoolValue(true),
			planCostCentre:         types.StringValue("cc-1"),
			planTerm:               types.Int64Value(12),
			stateTerm:              types.Int64Value(12),
			wantName:               "port-1",
			wantVisibility:         true,
			wantCostCentre:         "cc-1",
			wantContractTermMonths: nil,
		},
		{
			name:                   "name changed",
			planName:               types.StringValue("port-new"),
			stateName:              types.StringValue("port-old"),
			planVis:                types.BoolValue(false),
			stateVis:               types.BoolValue(false),
			planCostCentre:         types.StringValue(""),
			planTerm:               types.Int64Value(12),
			stateTerm:              types.Int64Value(12),
			wantName:               "port-new",
			wantVisibility:         false,
			wantCostCentre:         "",
			wantContractTermMonths: nil,
		},
		{
			name:                   "visibility changed",
			planName:               types.StringValue("port-1"),
			stateName:              types.StringValue("port-1"),
			planVis:                types.BoolValue(false),
			stateVis:               types.BoolValue(true),
			planCostCentre:         types.StringValue("cc"),
			planTerm:               types.Int64Value(12),
			stateTerm:              types.Int64Value(12),
			wantName:               "port-1",
			wantVisibility:         false,
			wantCostCentre:         "cc",
			wantContractTermMonths: nil,
		},
		{
			name:                   "term changed",
			planName:               types.StringValue("port-1"),
			stateName:              types.StringValue("port-1"),
			planVis:                types.BoolValue(true),
			stateVis:               types.BoolValue(true),
			planCostCentre:         types.StringValue("cc"),
			planTerm:               types.Int64Value(24),
			stateTerm:              types.Int64Value(12),
			wantName:               "port-1",
			wantVisibility:         true,
			wantCostCentre:         "cc",
			wantContractTermMonths: intPtr(24),
		},
		{
			name:                   "cost centre cleared",
			planName:               types.StringValue("port-1"),
			stateName:              types.StringValue("port-1"),
			planVis:                types.BoolValue(true),
			stateVis:               types.BoolValue(true),
			planCostCentre:         types.StringValue(""),
			planTerm:               types.Int64Value(12),
			stateTerm:              types.Int64Value(12),
			wantName:               "port-1",
			wantVisibility:         true,
			wantCostCentre:         "",
			wantContractTermMonths: nil,
		},
		{
			name:                   "all fields changed",
			planName:               types.StringValue("new-name"),
			stateName:              types.StringValue("old-name"),
			planVis:                types.BoolValue(true),
			stateVis:               types.BoolValue(false),
			planCostCentre:         types.StringValue("new-cc"),
			planTerm:               types.Int64Value(36),
			stateTerm:              types.Int64Value(12),
			wantName:               "new-name",
			wantVisibility:         true,
			wantCostCentre:         "new-cc",
			wantContractTermMonths: intPtr(36),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolvePortModifyParams(
				tt.planName, tt.stateName,
				tt.planVis, tt.stateVis,
				tt.planCostCentre,
				tt.planTerm, tt.stateTerm,
			)
			assert.Equal(t, tt.wantName, result.name)
			assert.Equal(t, tt.wantVisibility, result.marketplaceVisibility)
			assert.Equal(t, tt.wantCostCentre, result.costCentre)
			if tt.wantContractTermMonths == nil {
				assert.Nil(t, result.contractTermMonths)
			} else {
				require.NotNil(t, result.contractTermMonths)
				assert.Equal(t, *tt.wantContractTermMonths, *result.contractTermMonths)
			}
		})
	}
}

func intPtr(v int) *int { return &v }
