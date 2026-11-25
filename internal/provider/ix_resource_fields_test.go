package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

func modifyIXPlan(t *testing.T, config, plan, state ixResourceModel) (ixResourceModel, diag.Diagnostics) {
	t.Helper()
	ctx := context.Background()
	r := &ixResource{}

	tfPlan, tfState := ixTestSchema(t, r)
	require.False(t, tfState.Set(ctx, &state).HasError())
	require.False(t, tfPlan.Set(ctx, &config).HasError())
	tfConfig := tfsdk.Config(tfPlan)
	require.False(t, tfPlan.Set(ctx, &plan).HasError())

	resp := &fwresource.ModifyPlanResponse{Plan: tfPlan}
	r.ModifyPlan(ctx, fwresource.ModifyPlanRequest{Config: tfConfig, Plan: tfPlan, State: tfState}, resp)

	var got ixResourceModel
	require.False(t, resp.Plan.Get(ctx, &got).HasError())
	return got, resp.Diagnostics
}

func runIXCreate(t *testing.T, plan ixResourceModel, live, updated *megaport.IX) (ixResourceModel, *MockIXService) {
	t.Helper()
	mock := &MockIXService{GetIXResult: live, UpdatedIX: updated}
	resp := createIX(t, plan, mock)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())

	var got ixResourceModel
	require.False(t, resp.State.Get(context.Background(), &got).HasError())
	return got, mock
}

func createIX(t *testing.T, plan ixResourceModel, mock *MockIXService) *fwresource.CreateResponse {
	t.Helper()
	ctx := context.Background()
	mock.BuyIXResult = &megaport.BuyIXResponse{TechnicalServiceUID: ixReadTestUID}
	r := &ixResource{client: &megaport.Client{IXService: mock}}

	tfPlan, tfState := ixTestSchema(t, r)
	require.False(t, tfPlan.Set(ctx, &plan).HasError())

	resp := &fwresource.CreateResponse{State: tfState}
	r.Create(ctx, fwresource.CreateRequest{Plan: tfPlan}, resp)
	return resp
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

	var empty ixResourceModel
	empty.fromAPI(context.Background(), ixTestAPI())
	assert.Equal(t, types.StringValue(""), empty.ReverseDNS)
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

func TestIXUpdateSendsEachSettingAlone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		set  func(*ixResourceModel)
	}{
		{"cost_centre", func(m *ixResourceModel) { m.CostCentre = types.StringValue("cc-2") }},
		{"public_graph", func(m *ixResourceModel) { m.PublicGraph = types.BoolValue(true) }},
		{"reverse_dns", func(m *ixResourceModel) { m.ReverseDNS = types.StringValue("ix.example.com") }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			plan := ixTestModel()
			tc.set(&plan)

			_, mock := runIXUpdate(t, plan, ixTestModel(), ixTestAPI())

			assert.Len(t, mock.GotUpdateIXReqs, 1)
		})
	}
}

func TestIXUpdateSendsClearedCostCentre(t *testing.T) {
	t.Parallel()
	state := ixTestModel()
	state.CostCentre = types.StringValue("cc-1")

	got, mock := runIXUpdate(t, ixTestModel(), state, ixTestAPI())

	require.Len(t, mock.GotUpdateIXReqs, 1)
	require.NotNil(t, mock.GotUpdateIXReqs[0].CostCentre)
	assert.Equal(t, "", *mock.GotUpdateIXReqs[0].CostCentre)
	assert.Equal(t, types.StringValue(""), got.CostCentre)
}

func TestIXModifyPlanReverseDNSFollowsASN(t *testing.T) {
	t.Parallel()
	const megaportDefault = "as12345.sydney.megaport.com."
	cases := []struct {
		name       string
		stateDNS   string
		configDNS  types.String
		asn        types.Int64
		wantDNS    types.String
		wantErrors int
	}{
		{"new ASN with reverse_dns unset", megaportDefault, types.StringNull(), types.Int64Value(65000), types.StringUnknown(), 0},
		{"new ASN with a custom reverse_dns set", megaportDefault, types.StringValue("ix.example.com"), types.Int64Value(65000), types.StringValue("ix.example.com"), 0},
		{"new ASN with the default reverse_dns set", megaportDefault, types.StringValue("AS12345.SYDNEY.MEGAPORT.COM"), types.Int64Value(65000), types.StringValue("AS12345.SYDNEY.MEGAPORT.COM"), 1},
		{"unknown ASN with reverse_dns unset", megaportDefault, types.StringNull(), types.Int64Unknown(), types.StringUnknown(), 0},
		{"unknown ASN with the default reverse_dns set", megaportDefault, types.StringValue(megaportDefault), types.Int64Unknown(), types.StringValue(megaportDefault), 0},
		{"new ASN with a custom reverse_dns in state", "ix.example.com.", types.StringNull(), types.Int64Value(65000), types.StringValue("ix.example.com."), 0},
		{"new ASN with a custom name ending in megaport.com", "ix.notmegaport.com.", types.StringNull(), types.Int64Value(65000), types.StringValue("ix.notmegaport.com."), 0},
		{"new ASN with megaport.com inside a custom name", "ix.MEGAPORT.COM.example.net.", types.StringNull(), types.Int64Value(65000), types.StringValue("ix.MEGAPORT.COM.example.net."), 0},
		{"same ASN", megaportDefault, types.StringNull(), types.Int64Value(12345), types.StringValue(megaportDefault), 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			state := ixTestModel()
			state.ReverseDNS = types.StringValue(tc.stateDNS)
			plan := state
			plan.ASN = tc.asn
			if !tc.configDNS.IsNull() {
				plan.ReverseDNS = tc.configDNS
			}
			config := plan
			config.ReverseDNS = tc.configDNS

			got, diags := modifyIXPlan(t, config, plan, state)

			require.Len(t, diags.Errors(), tc.wantErrors)
			if tc.wantErrors > 0 {
				withPath, ok := diags.Errors()[0].(diag.DiagnosticWithPath)
				require.True(t, ok)
				assert.Equal(t, path.Root("reverse_dns"), withPath.Path())
			}
			assert.Equal(t, tc.wantDNS, got.ReverseDNS)
		})
	}
}

func TestIXModifyPlanSkipsCreateAndDestroy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &ixResource{}
	model := ixTestModel()

	nullPlan, nullState := ixTestSchema(t, r)
	plan, state := ixTestSchema(t, r)
	require.False(t, plan.Set(ctx, &model).HasError())
	require.False(t, state.Set(ctx, &model).HasError())

	create := &fwresource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, fwresource.ModifyPlanRequest{Config: tfsdk.Config(plan), Plan: plan, State: nullState}, create)
	assert.False(t, create.Diagnostics.HasError(), "create")

	destroy := &fwresource.ModifyPlanResponse{Plan: nullPlan}
	r.ModifyPlan(ctx, fwresource.ModifyPlanRequest{Config: tfsdk.Config(nullPlan), Plan: nullPlan, State: state}, destroy)
	assert.False(t, destroy.Diagnostics.HasError(), "destroy")
}

// State from the previous release holds null for these attributes when the config leaves them unset.
func TestIXSchemaPlansOnlyNonNullState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &ixResource{}
	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	_, tfState := ixTestSchema(t, r)
	state := ixTestModel()
	require.False(t, tfState.Set(ctx, &state).HasError())

	planString := func(name string, stateValue types.String) types.String {
		attr, ok := schemaResp.Schema.Attributes[name].(schema.StringAttribute)
		require.True(t, ok, name)
		req := planmodifier.StringRequest{State: tfState, StateValue: stateValue, PlanValue: types.StringUnknown(), ConfigValue: types.StringNull()}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		for _, m := range attr.PlanModifiers {
			m.PlanModifyString(ctx, req, resp)
		}
		return resp.PlanValue
	}
	for _, name := range []string{"cost_centre", "reverse_dns"} {
		assert.True(t, planString(name, types.StringNull()).IsUnknown(), name)
		assert.Equal(t, types.StringValue("live"), planString(name, types.StringValue("live")), name)
	}

	shutdown, ok := schemaResp.Schema.Attributes["shutdown"].(schema.BoolAttribute)
	require.True(t, ok)
	planShutdown := func(stateValue types.Bool) types.Bool {
		req := planmodifier.BoolRequest{State: tfState, StateValue: stateValue, PlanValue: types.BoolUnknown(), ConfigValue: types.BoolNull()}
		resp := &planmodifier.BoolResponse{PlanValue: req.PlanValue}
		for _, m := range shutdown.PlanModifiers {
			m.PlanModifyBool(ctx, req, resp)
		}
		return resp.PlanValue
	}
	assert.True(t, planShutdown(types.BoolNull()).IsUnknown(), "shutdown")
	assert.Equal(t, types.BoolValue(true), planShutdown(types.BoolValue(true)), "shutdown")
}

// The API sets a default reverse DNS when it gets none, so an empty value can never match it.
func TestIXSchemaRejectsEmptyReverseDNS(t *testing.T) {
	t.Parallel()
	schemaResp := fwresource.SchemaResponse{}
	(&ixResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, &schemaResp)

	resp := &validator.StringResponse{}
	reverseDNS, ok := schemaResp.Schema.Attributes["reverse_dns"].(schema.StringAttribute)
	require.True(t, ok)
	for _, v := range reverseDNS.Validators {
		v.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("reverse_dns"), ConfigValue: types.StringValue("")}, resp)
	}
	assert.True(t, resp.Diagnostics.HasError())
}

// An upgraded state can hold a null shutdown, which plans as unknown.
func TestIXUpdateSkipsUnknownShutdown(t *testing.T) {
	t.Parallel()
	plan := ixTestModel()
	plan.ProductName = types.StringValue("renamed")
	plan.Shutdown = types.BoolUnknown()
	state := ixTestModel()
	state.Shutdown = types.BoolNull()

	_, mock := runIXUpdate(t, plan, state, ixTestAPI())

	require.Len(t, mock.GotUpdateIXReqs, 1)
	assert.Nil(t, mock.GotUpdateIXReqs[0].Shutdown)
}

func TestIXUpdateSendsShutdown(t *testing.T) {
	t.Parallel()
	plan := ixTestModel()
	plan.Shutdown = types.BoolValue(true)

	_, mock := runIXUpdate(t, plan, ixTestModel(), ixTestAPI())

	require.Len(t, mock.GotUpdateIXReqs, 1)
	require.NotNil(t, mock.GotUpdateIXReqs[0].Shutdown)
	assert.True(t, *mock.GotUpdateIXReqs[0].Shutdown)
}

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

func TestIXCreateSendsSettingsInFollowUpUpdate(t *testing.T) {
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
		got, mock := runIXCreate(t, unset, ixTestAPI(), nil)
		assert.Empty(t, mock.GotUpdateIXReqs)
		assert.Equal(t, types.StringValue(""), got.CostCentre)
		assert.Equal(t, types.StringValue(""), got.ReverseDNS)
	})

	t.Run("unset keeps the live settings", func(t *testing.T) {
		t.Parallel()
		live := ixTestAPI()
		live.CostCentre, live.PublicGraph, live.ReverseDNS = "cc-live", true, "live.example.com."
		got, mock := runIXCreate(t, unset, live, nil)
		assert.Empty(t, mock.GotUpdateIXReqs)
		assert.Equal(t, "cc-live", got.CostCentre.ValueString())
		assert.True(t, got.PublicGraph.ValueBool())
		assert.Equal(t, "live.example.com.", got.ReverseDNS.ValueString())
	})

	t.Run("set sends one update", func(t *testing.T) {
		t.Parallel()
		live := ixTestAPI()
		live.MACAddress = "02:ab:cd:ef:00:01"
		updated := ixTestAPI()
		updated.MACAddress = "02:ab:cd:ef:00:01"
		updated.CostCentre, updated.PublicGraph, updated.ReverseDNS = "cc-1", true, "ix.example.com."

		plan := set
		plan.MACAddress = types.StringValue("02:AB:CD:EF:00:01")
		got, mock := runIXCreate(t, plan, live, updated)

		require.Len(t, mock.GotUpdateIXReqs, 1)
		req := mock.GotUpdateIXReqs[0]
		require.NotNil(t, req.CostCentre)
		assert.Equal(t, "cc-1", *req.CostCentre)
		require.NotNil(t, req.PublicGraph)
		assert.True(t, *req.PublicGraph)
		require.NotNil(t, req.ReverseDns)
		assert.Equal(t, "ix.example.com", *req.ReverseDns)
		assert.Equal(t, "cc-1", got.CostCentre.ValueString())
		assert.Equal(t, "02:AB:CD:EF:00:01", got.MACAddress.ValueString())
		assert.Equal(t, "ix.example.com", got.ReverseDNS.ValueString())
	})

	t.Run("set to the live value sends no update", func(t *testing.T) {
		t.Parallel()
		live := ixTestAPI()
		live.CostCentre, live.PublicGraph, live.ReverseDNS = "cc-1", true, "ix.example.com"
		_, mock := runIXCreate(t, set, live, nil)
		assert.Empty(t, mock.GotUpdateIXReqs)
	})

	t.Run("failed update keeps the UID", func(t *testing.T) {
		t.Parallel()
		resp := createIX(t, set, &MockIXService{GetIXResult: ixTestAPI(), UpdateIXErr: errors.New("update failed")})

		require.Len(t, resp.Diagnostics.Errors(), 1)
		assert.Equal(t, "IX ordered but settings not applied", resp.Diagnostics.Errors()[0].Summary())
		var uid types.String
		require.False(t, resp.State.GetAttribute(context.Background(), path.Root("product_uid"), &uid).HasError())
		assert.Equal(t, ixReadTestUID, uid.ValueString())
	})
}

// The SDK refuses a longer cost_centre, which would fail the update after Create orders the IX.
func TestIXSchemaLimitsCostCentre(t *testing.T) {
	t.Parallel()
	schemaResp := fwresource.SchemaResponse{}
	(&ixResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, &schemaResp)
	costCentre, ok := schemaResp.Schema.Attributes["cost_centre"].(schema.StringAttribute)
	require.True(t, ok)

	validate := func(value string) diag.Diagnostics {
		resp := &validator.StringResponse{}
		for _, v := range costCentre.Validators {
			v.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("cost_centre"), ConfigValue: types.StringValue(value)}, resp)
		}
		return resp.Diagnostics
	}
	assert.False(t, validate(strings.Repeat("a", 255)).HasError())
	assert.True(t, validate(strings.Repeat("a", 256)).HasError())
}

// The API lowercases the MAC and adds a trailing dot to reverse DNS.
func TestIXUpdateKeepsPlannedFormOfNormalizedValues(t *testing.T) {
	t.Parallel()

	t.Run("same_values_after_import", func(t *testing.T) {
		t.Parallel()
		plan := ixTestModel()
		plan.MACAddress = types.StringValue("02:AB:CD:EF:00:01")
		plan.ReverseDNS = types.StringValue("IX.example.com")
		state := ixTestModel()
		state.MACAddress = types.StringValue("02:ab:cd:ef:00:01")
		state.ReverseDNS = types.StringValue("ix.example.com.")
		live := ixTestAPI()
		live.MACAddress, live.ReverseDNS = "02:ab:cd:ef:00:01", "ix.example.com."

		got, mock := runIXUpdate(t, plan, state, live)

		assert.Empty(t, mock.GotUpdateIXReqs)
		assert.Equal(t, "02:AB:CD:EF:00:01", got.MACAddress.ValueString())
		assert.Equal(t, "IX.example.com", got.ReverseDNS.ValueString())
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
