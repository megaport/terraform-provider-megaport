package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	megaport "github.com/megaport/megaportgo"
)

const (
	ixTestPortUID = "port-uid-456"
	ixTestMAC     = "02:00:00:00:00:01"
	ixTestMACNew  = "02:00:00:00:00:02"
)

// ixTestModel is an IX as Terraform holds it after a create with no optional settings.
func ixTestModel() ixResourceModel {
	return ixResourceModel{
		RequestedProductUID: types.StringValue(ixTestPortUID),
		ProductUID:          types.StringValue(ixReadTestUID),
		ProductName:         types.StringValue("test-ix"),
		NetworkServiceType:  types.StringValue("Sydney IX"),
		ASN:                 types.Int64Value(12345),
		MACAddress:          types.StringValue(ixTestMAC),
		RateLimit:           types.Int64Value(500),
		VLAN:                types.Int64Value(2001),
		Shutdown:            types.BoolValue(false),
		CostCentre:          types.StringValue(""),
		PublicGraph:         types.BoolValue(false),
		ReverseDNS:          types.StringValue(""),
		AttributeTags:       types.MapNull(types.StringType),
		Resources:           types.ObjectNull(resourcesAttrTypes),
	}
}

// ixTestAPI is the API side of ixTestModel.
func ixTestAPI() *megaport.IX {
	return &megaport.IX{
		ProductUID:         ixReadTestUID,
		ProductName:        "test-ix",
		NetworkServiceType: "Sydney IX",
		ASN:                12345,
		MACAddress:         ixTestMAC,
		RateLimit:          500,
		VLAN:               2001,
		ProvisioningStatus: megaport.SERVICE_LIVE,
	}
}

func ixTestSchema(t *testing.T, r *ixResource) (tfsdk.Plan, tfsdk.State) {
	t.Helper()
	schemaResp := fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	objType, ok := s.Type().TerraformType(context.Background()).(tftypes.Object)
	require.True(t, ok, "schema type is not tftypes.Object")
	return tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(objType, nil)}, tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, nil)}
}

func runIXUpdate(t *testing.T, plan, state ixResourceModel, live *megaport.IX) (ixResourceModel, *MockIXService) {
	t.Helper()
	ctx := context.Background()
	mock := &MockIXService{GetIXResult: live}
	r := &ixResource{client: &megaport.Client{IXService: mock}}

	tfPlan, tfState := ixTestSchema(t, r)
	require.False(t, tfPlan.Set(ctx, &plan).HasError())
	require.False(t, tfState.Set(ctx, &state).HasError())

	resp := &fwresource.UpdateResponse{State: tfState}
	r.Update(ctx, fwresource.UpdateRequest{Plan: tfPlan, State: tfState}, resp)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())

	var got ixResourceModel
	require.False(t, resp.State.Get(ctx, &got).HasError())
	return got, mock
}

func runIXCreate(t *testing.T, plan ixResourceModel, live *megaport.IX) *MockIXService {
	t.Helper()
	ctx := context.Background()
	mock := &MockIXService{GetIXResult: live, BuyIXResult: &megaport.BuyIXResponse{TechnicalServiceUID: ixReadTestUID}}
	r := &ixResource{client: &megaport.Client{IXService: mock}}

	tfPlan, tfState := ixTestSchema(t, r)
	require.False(t, tfPlan.Set(ctx, &plan).HasError())

	resp := &fwresource.CreateResponse{State: tfState}
	r.Create(ctx, fwresource.CreateRequest{Plan: tfPlan}, resp)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())
	return mock
}

func TestIXFromAPIMapsTopLevelFields(t *testing.T) {
	t.Parallel()
	ix := ixTestAPI()
	ix.Shutdown = true
	ix.CostCentre = "cc-1"
	ix.PublicGraph = true
	ix.ReverseDNS = "ix.example.com"

	var got ixResourceModel
	got.fromAPI(context.Background(), ix)

	assert.Equal(t, ixTestMAC, got.MACAddress.ValueString())
	assert.True(t, got.Shutdown.ValueBool())
	assert.Equal(t, "cc-1", got.CostCentre.ValueString())
	assert.True(t, got.PublicGraph.ValueBool())
	assert.Equal(t, "ix.example.com", got.ReverseDNS.ValueString())
}

func TestIXUpdateSendsSettingChanges(t *testing.T) {
	t.Parallel()
	plan := ixTestModel()
	plan.CostCentre = types.StringValue("cc-2")
	plan.PublicGraph = types.BoolValue(true)
	plan.ReverseDNS = types.StringValue("ix.example.com")
	live := ixTestAPI()
	live.CostCentre, live.PublicGraph, live.ReverseDNS = "cc-2", true, "ix.example.com"

	got, mock := runIXUpdate(t, plan, ixTestModel(), live)

	require.Len(t, mock.GotUpdateIXReqs, 1)
	req := mock.GotUpdateIXReqs[0]
	require.NotNil(t, req.CostCentre)
	assert.Equal(t, "cc-2", *req.CostCentre)
	require.NotNil(t, req.PublicGraph)
	assert.True(t, *req.PublicGraph)
	require.NotNil(t, req.ReverseDns)
	assert.Equal(t, "ix.example.com", *req.ReverseDns)
	assert.Nil(t, req.Name)
	assert.Nil(t, req.MACAddress)
	assert.Equal(t, "cc-2", got.CostCentre.ValueString())
}

// A MAC change used to leave the old MAC in state, so the apply failed as inconsistent.
func TestIXUpdateRecordsNewMAC(t *testing.T) {
	t.Parallel()
	plan := ixTestModel()
	plan.MACAddress = types.StringValue(ixTestMACNew)
	live := ixTestAPI()
	live.MACAddress = ixTestMACNew

	got, mock := runIXUpdate(t, plan, ixTestModel(), live)

	require.Len(t, mock.GotUpdateIXReqs, 1)
	require.NotNil(t, mock.GotUpdateIXReqs[0].MACAddress)
	assert.Equal(t, ixTestMACNew, *mock.GotUpdateIXReqs[0].MACAddress)
	assert.Equal(t, ixTestMACNew, got.MACAddress.ValueString())
}

func TestIXUpdateRequestedProductUID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		state types.String
		want  string
	}{
		{"null after import takes the plan", types.StringNull(), ixTestPortUID},
		{"set keeps the state", types.StringValue("other-port-uid"), "other-port-uid"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			state := ixTestModel()
			state.RequestedProductUID = tc.state

			got, mock := runIXUpdate(t, ixTestModel(), state, ixTestAPI())

			assert.Empty(t, mock.GotUpdateIXReqs)
			assert.Equal(t, tc.want, got.RequestedProductUID.ValueString())
		})
	}
}

func TestIXCreateSetsSettingsTheOrderCannotCarry(t *testing.T) {
	t.Parallel()
	unset := ixTestModel()
	unset.Shutdown = types.BoolUnknown()
	unset.CostCentre = types.StringUnknown()
	unset.PublicGraph = types.BoolUnknown()
	unset.ReverseDNS = types.StringUnknown()

	set := unset
	set.CostCentre = types.StringValue("cc-1")
	set.PublicGraph = types.BoolValue(true)
	set.ReverseDNS = types.StringValue("ix.example.com")

	t.Run("unset sends no update", func(t *testing.T) {
		t.Parallel()
		mock := runIXCreate(t, unset, ixTestAPI())
		assert.Empty(t, mock.GotUpdateIXReqs)
	})

	t.Run("set sends one update", func(t *testing.T) {
		t.Parallel()
		mock := runIXCreate(t, set, ixTestAPI())
		require.Len(t, mock.GotUpdateIXReqs, 1)
		req := mock.GotUpdateIXReqs[0]
		require.NotNil(t, req.CostCentre)
		assert.Equal(t, "cc-1", *req.CostCentre)
		require.NotNil(t, req.PublicGraph)
		assert.True(t, *req.PublicGraph)
		require.NotNil(t, req.ReverseDns)
		assert.Equal(t, "ix.example.com", *req.ReverseDns)
	})

	t.Run("set to the live value sends no update", func(t *testing.T) {
		t.Parallel()
		live := ixTestAPI()
		live.CostCentre, live.PublicGraph, live.ReverseDNS = "cc-1", true, "ix.example.com"
		mock := runIXCreate(t, set, live)
		assert.Empty(t, mock.GotUpdateIXReqs)
	})
}

// The API lowercases the MAC and adds a trailing dot to reverse DNS.
func TestIXUpdateKeepsPlannedFormOfNormalizedValues(t *testing.T) {
	t.Parallel()

	t.Run("same_values_after_import", func(t *testing.T) {
		t.Parallel()
		plan := ixTestModel()
		plan.MACAddress = types.StringValue("02:AB:CD:EF:00:01")
		plan.ReverseDNS = types.StringValue("ix.example.com")
		state := ixTestModel()
		state.MACAddress = types.StringValue("02:ab:cd:ef:00:01")
		state.ReverseDNS = types.StringValue("ix.example.com.")
		live := ixTestAPI()
		live.MACAddress, live.ReverseDNS = "02:ab:cd:ef:00:01", "ix.example.com."

		got, mock := runIXUpdate(t, plan, state, live)

		assert.Empty(t, mock.GotUpdateIXReqs)
		assert.Equal(t, "02:AB:CD:EF:00:01", got.MACAddress.ValueString())
		assert.Equal(t, "ix.example.com", got.ReverseDNS.ValueString())
	})

	t.Run("changed_values", func(t *testing.T) {
		t.Parallel()
		plan := ixTestModel()
		plan.MACAddress = types.StringValue("02:AB:CD:EF:00:02")
		plan.ReverseDNS = types.StringValue("ix2.example.com")
		live := ixTestAPI()
		live.MACAddress, live.ReverseDNS = "02:ab:cd:ef:00:02", "ix2.example.com."

		got, mock := runIXUpdate(t, plan, ixTestModel(), live)

		require.Len(t, mock.GotUpdateIXReqs, 1)
		assert.NotNil(t, mock.GotUpdateIXReqs[0].MACAddress)
		assert.NotNil(t, mock.GotUpdateIXReqs[0].ReverseDns)
		assert.Equal(t, "02:AB:CD:EF:00:02", got.MACAddress.ValueString())
		assert.Equal(t, "ix2.example.com", got.ReverseDNS.ValueString())
	})
}
