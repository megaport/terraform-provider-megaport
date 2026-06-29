package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrefixFilterIDToName(t *testing.T) {
	pflMap := map[int]string{
		100: "whitelist-filter",
		200: "blacklist-filter",
		300: "export-filter",
	}

	tests := []struct {
		name     string
		id       int
		pflMap   map[int]string
		existing basetypes.StringValue
		wantNull bool
		wantVal  string
	}{
		{
			name:     "zero ID returns null even when existing is set",
			id:       0,
			pflMap:   pflMap,
			existing: types.StringValue("old-filter"),
			wantNull: true,
		},
		{
			name:    "known ID returns name",
			id:      100,
			pflMap:  pflMap,
			wantVal: "whitelist-filter",
		},
		{
			name:    "another known ID returns name",
			id:      200,
			pflMap:  pflMap,
			wantVal: "blacklist-filter",
		},
		{
			name:     "unknown non-zero ID preserves existing value",
			id:       999,
			pflMap:   pflMap,
			existing: types.StringValue("kept-filter"),
			wantVal:  "kept-filter",
		},
		{
			name:     "empty map with non-zero ID preserves existing value",
			id:       100,
			pflMap:   map[int]string{},
			existing: types.StringValue("kept-filter"),
			wantVal:  "kept-filter",
		},
		{
			name:     "unknown non-zero ID with null existing returns null",
			id:       999,
			pflMap:   pflMap,
			existing: types.StringNull(),
			wantNull: true,
		},
		{
			name:     "nil map with zero ID returns null",
			id:       0,
			pflMap:   nil,
			existing: types.StringNull(),
			wantNull: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := prefixFilterIDToName(tt.id, tt.pflMap, tt.existing)
			if tt.wantNull {
				if !result.IsNull() {
					t.Errorf("expected null, got %q", result.ValueString())
				}
			} else {
				if result.IsNull() {
					t.Errorf("expected %q, got null", tt.wantVal)
				} else if result.ValueString() != tt.wantVal {
					t.Errorf("expected %q, got %q", tt.wantVal, result.ValueString())
				}
			}
		})
	}
}

func TestGetPartnerType(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		partner string
		want    string
	}{
		{
			name:    "vrouter partner type",
			partner: "vrouter",
			want:    "vrouter",
		},
		{
			name:    "a-end partner type",
			partner: "a-end",
			want:    "a-end",
		},
		{
			name:    "aws partner type",
			partner: "aws",
			want:    "aws",
		},
		{
			name:    "empty partner type",
			partner: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              types.StringValue(tt.partner),
				AWSPartnerConfig:     types.ObjectNull(vxcPartnerConfigAWSAttrs),
				AzurePartnerConfig:   types.ObjectNull(vxcPartnerConfigAzureAttrs),
				GooglePartnerConfig:  types.ObjectNull(vxcPartnerConfigGoogleAttrs),
				OraclePartnerConfig:  types.ObjectNull(vxcPartnerConfigOracleAttrs),
				IBMPartnerConfig:     types.ObjectNull(vxcPartnerConfigIbmAttrs),
				VrouterPartnerConfig: types.ObjectNull(vxcPartnerConfigVrouterAttrs),
				PartnerAEndConfig:    types.ObjectNull(vxcPartnerConfigAEndAttrs),
			}
			obj, diags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, partnerConfigModel)
			if diags.HasError() {
				t.Fatalf("failed to create partner config object: %s", diags.Errors())
			}
			got := getPartnerType(ctx, obj)
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestGetPartnerType_NullObject(t *testing.T) {
	ctx := context.Background()
	nullObj := types.ObjectNull(vxcPartnerConfigAttrs)
	got := getPartnerType(ctx, nullObj)
	if got != "" {
		t.Errorf("expected empty string for null object, got %q", got)
	}
}

// buildVrouterPartnerConfigObject is a test helper that creates a vrouter partner config
// types.Object with the given BGP connections for testing purposes.
func buildVrouterPartnerConfigObject(t *testing.T, ctx context.Context, bgpModels []bgpConnectionConfigModel) basetypes.ObjectValue {
	t.Helper()

	bgpList, diags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(bgpVrouterConnectionConfig), bgpModels)
	if diags.HasError() {
		t.Fatalf("failed to create bgp list: %s", diags.Errors())
	}

	ifaceModel := vxcPartnerConfigInterfaceModel{
		IPAddresses:        types.ListNull(types.StringType),
		IPRoutes:           types.ListNull(types.ObjectType{}.WithAttributeTypes(ipRouteAttrs)),
		NatIPAddresses:     types.ListNull(types.StringType),
		Bfd:                types.ObjectNull(bfdConfigAttrs),
		BgpConnections:     bgpList,
		VLAN:               types.Int64Null(),
		IpMtu:              types.Int64Null(),
		IpSecTunnelOptions: types.ObjectNull(ipSecTunnelOptionsAttrs),
	}

	ifaceList, diags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(vxcVrouterInterfaceAttrs), []vxcPartnerConfigInterfaceModel{ifaceModel})
	if diags.HasError() {
		t.Fatalf("failed to create interface list: %s", diags.Errors())
	}

	vrouterModel := vxcPartnerConfigVrouterModel{Interfaces: ifaceList}
	vrouterObj, diags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, vrouterModel)
	if diags.HasError() {
		t.Fatalf("failed to create vrouter object: %s", diags.Errors())
	}

	partnerConfigModel := &vxcPartnerConfigurationModel{
		Partner:              types.StringValue("vrouter"),
		AWSPartnerConfig:     types.ObjectNull(vxcPartnerConfigAWSAttrs),
		AzurePartnerConfig:   types.ObjectNull(vxcPartnerConfigAzureAttrs),
		GooglePartnerConfig:  types.ObjectNull(vxcPartnerConfigGoogleAttrs),
		OraclePartnerConfig:  types.ObjectNull(vxcPartnerConfigOracleAttrs),
		IBMPartnerConfig:     types.ObjectNull(vxcPartnerConfigIbmAttrs),
		VrouterPartnerConfig: vrouterObj,
		PartnerAEndConfig:    types.ObjectNull(vxcPartnerConfigAEndAttrs),
	}

	obj, diags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, partnerConfigModel)
	if diags.HasError() {
		t.Fatalf("failed to create partner config object: %s", diags.Errors())
	}
	return obj
}

func TestReconcilePartnerConfigs_PreservesNonVrouterFromPlan(t *testing.T) {
	ctx := context.Background()

	// Build an AWS partner config object for the plan
	awsModel := &vxcPartnerConfigAWSModel{
		ConnectType:       types.StringValue("AWS"),
		Type:              types.StringValue("private"),
		OwnerAccount:      types.StringValue("123456789"),
		ASN:               types.Int64Value(64512),
		AmazonASN:         types.Int64Value(64513),
		AuthKey:           types.StringValue("key"),
		Prefixes:          types.StringValue(""),
		CustomerIPAddress: types.StringValue("10.0.0.1"),
		AmazonIPAddress:   types.StringValue("10.0.0.2"),
		ConnectionName:    types.StringValue("test"),
	}
	awsObj, diags := types.ObjectValueFrom(ctx, vxcPartnerConfigAWSAttrs, awsModel)
	if diags.HasError() {
		t.Fatalf("failed to create AWS config: %s", diags.Errors())
	}

	partnerConfigModel := &vxcPartnerConfigurationModel{
		Partner:              types.StringValue("aws"),
		AWSPartnerConfig:     awsObj,
		AzurePartnerConfig:   types.ObjectNull(vxcPartnerConfigAzureAttrs),
		GooglePartnerConfig:  types.ObjectNull(vxcPartnerConfigGoogleAttrs),
		OraclePartnerConfig:  types.ObjectNull(vxcPartnerConfigOracleAttrs),
		IBMPartnerConfig:     types.ObjectNull(vxcPartnerConfigIbmAttrs),
		VrouterPartnerConfig: types.ObjectNull(vxcPartnerConfigVrouterAttrs),
		PartnerAEndConfig:    types.ObjectNull(vxcPartnerConfigAEndAttrs),
	}
	awsPartnerConfig, diags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, partnerConfigModel)
	if diags.HasError() {
		t.Fatalf("failed to create partner config: %s", diags.Errors())
	}

	plan := &vxcResourceModel{
		BEndPartnerConfig: awsPartnerConfig,
	}

	orm := &vxcResourceModel{}

	vxc := &megaport.VXC{
		AEndConfiguration: megaport.VXCEndConfiguration{},
		BEndConfiguration: megaport.VXCEndConfiguration{},
	}

	reconDiags := orm.reconcilePartnerConfigs(ctx, vxc, plan, nil)
	if reconDiags.HasError() {
		t.Fatalf("unexpected error: %s", reconDiags.Errors())
	}

	// The AWS partner config should be preserved from plan
	if orm.BEndPartnerConfig.IsNull() {
		t.Fatal("expected B-end partner config to be preserved from plan")
	}

	partnerType := getPartnerType(ctx, orm.BEndPartnerConfig)
	if partnerType != "aws" {
		t.Errorf("expected partner type 'aws', got %q", partnerType)
	}
}

// --- Test helpers ---

// extractBGPFromResult extracts the first BGP connection from a merged partner config result.
func extractBGPFromResult(t *testing.T, ctx context.Context, result basetypes.ObjectValue) *bgpConnectionConfigModel {
	t.Helper()
	iface := extractInterfaceFromResult(t, ctx, result)

	bgpConns := []*bgpConnectionConfigModel{}
	bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConns, false)
	if bgpDiags.HasError() {
		t.Fatalf("failed to extract BGP connections: %s", bgpDiags.Errors())
	}
	if len(bgpConns) == 0 {
		t.Fatal("expected at least 1 BGP connection")
	}
	return bgpConns[0]
}

// extractInterfaceFromResult extracts the first interface from a merged partner config result.
func extractInterfaceFromResult(t *testing.T, ctx context.Context, result basetypes.ObjectValue) *vxcPartnerConfigInterfaceModel {
	t.Helper()

	partnerModel := &vxcPartnerConfigurationModel{}
	pDiags := result.As(ctx, partnerModel, basetypes.ObjectAsOptions{})
	if pDiags.HasError() {
		t.Fatalf("failed to extract partner config: %s", pDiags.Errors())
	}

	vrouterModel := &vxcPartnerConfigVrouterModel{}
	vrDiags := partnerModel.VrouterPartnerConfig.As(ctx, vrouterModel, basetypes.ObjectAsOptions{})
	if vrDiags.HasError() {
		t.Fatalf("failed to extract vrouter config: %s", vrDiags.Errors())
	}

	ifaceModels := []*vxcPartnerConfigInterfaceModel{}
	ifDiags := vrouterModel.Interfaces.ElementsAs(ctx, &ifaceModels, false)
	if ifDiags.HasError() {
		t.Fatalf("failed to extract interfaces: %s", ifDiags.Errors())
	}
	if len(ifaceModels) == 0 {
		t.Fatal("expected at least 1 interface")
	}
	return ifaceModels[0]
}

func assertInt64(t *testing.T, field string, got types.Int64, want int64) {
	t.Helper()
	if got.IsNull() {
		t.Errorf("%s: expected %d, got null", field, want)
		return
	}
	if got.ValueInt64() != want {
		t.Errorf("%s: expected %d, got %d", field, want, got.ValueInt64())
	}
}

func assertString(t *testing.T, field string, got types.String, want string) {
	t.Helper()
	if got.IsNull() {
		t.Errorf("%s: expected %q, got null", field, want)
		return
	}
	if got.ValueString() != want {
		t.Errorf("%s: expected %q, got %q", field, want, got.ValueString())
	}
}

// TestCreateVrouterPartnerConfig_IPsecTunnelOptions verifies the model -> SDK
// mapping for ip_sec_tunnel_options. There is one tunnel per ipSecTunnel
// interface, so the multi-tunnel case is two interfaces. It also covers the
// "nil keeps the API default" behavior for the optional pointer fields and that
// a nil tunnel is omitted entirely (never an empty object).
func TestCreateVrouterPartnerConfig_IPsecTunnelOptions(t *testing.T) {
	ctx := context.Background()

	// pre_shared_key is write-only, so it is null in the plan-derived model and
	// supplied separately via the preSharedKeys map (keyed by interface index).
	fullTunnel := ipSecTunnelOptionsModel{
		SourceIPAddress:      types.StringValue("169.254.1.1"),
		DestinationIPAddress: types.StringValue("203.0.113.1"),
		PreSharedKey:         types.StringNull(),
		Passive:              types.BoolValue(false),
		LocalID:              types.StringValue("local-1"),
		RemoteID:             types.StringValue("remote-1"),
		Phase1Lifetime:       types.Int64Value(7200),
		Phase2Lifetime:       types.Int64Value(3600),
	}
	// Minimal tunnel: only the required fields; optionals left null so the API
	// applies its defaults.
	minimalTunnel := ipSecTunnelOptionsModel{
		SourceIPAddress:      types.StringValue("169.254.2.1"),
		DestinationIPAddress: types.StringValue("203.0.113.2"),
		PreSharedKey:         types.StringNull(),
		Passive:              types.BoolNull(),
		LocalID:              types.StringNull(),
		RemoteID:             types.StringNull(),
		Phase1Lifetime:       types.Int64Null(),
		Phase2Lifetime:       types.Int64Null(),
	}

	fullObj, diags := types.ObjectValueFrom(ctx, ipSecTunnelOptionsAttrs, fullTunnel)
	require.False(t, diags.HasError(), "building full tunnel object: %v", diags)
	minimalObj, diags := types.ObjectValueFrom(ctx, ipSecTunnelOptionsAttrs, minimalTunnel)
	require.False(t, diags.HasError(), "building minimal tunnel object: %v", diags)

	newIface := func(tunnel types.Object) vxcPartnerConfigInterfaceModel {
		return vxcPartnerConfigInterfaceModel{
			InterfaceType:      types.StringValue("ipSecTunnel"),
			IPAddresses:        types.ListNull(types.StringType),
			IPRoutes:           types.ListNull(types.ObjectType{}.WithAttributeTypes(ipRouteAttrs)),
			NatIPAddresses:     types.ListNull(types.StringType),
			Bfd:                types.ObjectNull(bfdConfigAttrs),
			BgpConnections:     types.ListNull(types.ObjectType{}.WithAttributeTypes(bgpVrouterConnectionConfig)),
			IpSecTunnelOptions: tunnel,
		}
	}

	// A third interface with no tunnel confirms nil is omitted, not serialized.
	noTunnelIface := newIface(types.ObjectNull(ipSecTunnelOptionsAttrs))

	ifaceList, diags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(vxcVrouterInterfaceAttrs), []vxcPartnerConfigInterfaceModel{
		newIface(fullObj),
		newIface(minimalObj),
		noTunnelIface,
	})
	require.False(t, diags.HasError(), "building interface list: %v", diags)

	model := vxcPartnerConfigVrouterModel{Interfaces: ifaceList}

	preSharedKeys := map[int]string{0: "secret-one", 1: "secret-two"}
	diags, vrouterConfig, _ := createVrouterPartnerConfig(ctx, model, nil, preSharedKeys)
	require.False(t, diags.HasError(), "createVrouterPartnerConfig: %v", diags)
	require.Len(t, vrouterConfig.Interfaces, 3)

	// Fully populated tunnel.
	full := vrouterConfig.Interfaces[0].IpSecTunnelOptions
	require.NotNil(t, full)
	assert.Equal(t, "169.254.1.1", full.SourceIpAddress)
	assert.Equal(t, "203.0.113.1", full.DestinationIpAddress)
	assert.Equal(t, "secret-one", full.PreSharedKey)
	assert.Equal(t, "local-1", full.LocalId)
	assert.Equal(t, "remote-1", full.RemoteId)
	require.NotNil(t, full.Passive)
	assert.False(t, *full.Passive)
	require.NotNil(t, full.Phase1Lifetime)
	assert.Equal(t, 7200, *full.Phase1Lifetime)
	require.NotNil(t, full.Phase2Lifetime)
	assert.Equal(t, 3600, *full.Phase2Lifetime)

	// Minimal tunnel: optional pointers stay nil so the API default applies.
	minimal := vrouterConfig.Interfaces[1].IpSecTunnelOptions
	require.NotNil(t, minimal)
	assert.Equal(t, "169.254.2.1", minimal.SourceIpAddress)
	assert.Equal(t, "203.0.113.2", minimal.DestinationIpAddress)
	assert.Equal(t, "secret-two", minimal.PreSharedKey)
	assert.Empty(t, minimal.LocalId)
	assert.Empty(t, minimal.RemoteId)
	assert.Nil(t, minimal.Passive)
	assert.Nil(t, minimal.Phase1Lifetime)
	assert.Nil(t, minimal.Phase2Lifetime)

	// No tunnel: pointer stays nil so the field is omitted from the payload.
	assert.Nil(t, vrouterConfig.Interfaces[2].IpSecTunnelOptions)
}

// TestIPSecPhaseLifetimeValidator covers the cross-field rule that phase2 must
// be strictly less than phase1, and that the check is skipped when either side
// is unset.
func TestIPSecPhaseLifetimeValidator(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name      string
		phase1    types.Int64
		phase2    types.Int64
		wantError bool
	}{
		{"phase2 less than phase1", types.Int64Value(7200), types.Int64Value(3600), false},
		{"phase2 equal to phase1", types.Int64Value(3600), types.Int64Value(3600), true},
		{"phase2 greater than phase1", types.Int64Value(3600), types.Int64Value(7200), true},
		{"both null", types.Int64Null(), types.Int64Null(), false},
		{"only phase1 set", types.Int64Value(7200), types.Int64Null(), false},
		{"only phase2 set", types.Int64Null(), types.Int64Value(3600), false},
		// Null lifetimes take the API defaults (phase1 28800, phase2 3600).
		{"only phase2 set above default phase1", types.Int64Null(), types.Int64Value(50000), true},
		{"only phase1 set to minimum (default phase2 not less)", types.Int64Value(3600), types.Int64Null(), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tunnel := ipSecTunnelOptionsModel{
				SourceIPAddress:      types.StringValue("169.254.1.1"),
				DestinationIPAddress: types.StringValue("203.0.113.1"),
				PreSharedKey:         types.StringValue("secret"),
				Passive:              types.BoolNull(),
				LocalID:              types.StringNull(),
				RemoteID:             types.StringNull(),
				Phase1Lifetime:       tc.phase1,
				Phase2Lifetime:       tc.phase2,
			}
			obj, diags := types.ObjectValueFrom(ctx, ipSecTunnelOptionsAttrs, tunnel)
			require.False(t, diags.HasError(), "building object: %v", diags)

			req := validator.ObjectRequest{
				Path:        path.Root("ip_sec_tunnel_options"),
				ConfigValue: obj,
			}
			resp := &validator.ObjectResponse{}
			ipSecPhaseLifetimeValidator{}.ValidateObject(ctx, req, resp)

			assert.Equal(t, tc.wantError, resp.Diagnostics.HasError(), "diagnostics: %v", resp.Diagnostics)
		})
	}
}

// nullBGPModel returns a BGP connection model with every field null, the
// baseline for a config where the user set only specific attributes.
func nullBGPModel() bgpConnectionConfigModel {
	return bgpConnectionConfigModel{
		PeerAsn:            types.Int64Null(),
		LocalAsn:           types.Int64Null(),
		PeerType:           types.StringNull(),
		LocalIPAddress:     types.StringNull(),
		PeerIPAddress:      types.StringNull(),
		Password:           types.StringNull(),
		Shutdown:           types.BoolNull(),
		Description:        types.StringNull(),
		MedIn:              types.Int64Null(),
		MedOut:             types.Int64Null(),
		BfdEnabled:         types.BoolNull(),
		ExportPolicy:       types.StringNull(),
		PermitExportTo:     types.ListNull(types.StringType),
		DenyExportTo:       types.ListNull(types.StringType),
		ImportWhitelist:    types.StringNull(),
		ImportBlacklist:    types.StringNull(),
		ExportWhitelist:    types.StringNull(),
		ExportBlacklist:    types.StringNull(),
		AsPathPrependCount: types.Int64Null(),
	}
}

// TestMergeVrouterPartnerConfigFromAPI_PreservesUnconfiguredOptionals guards the
// core drift-detection contract: fields the user set track the API, while
// Optional-only fields the user never set (shutdown, bfd_enabled, med_out,
// description) stay null even though the API returns concrete values for them.
// Writing those API values into null state would manufacture perpetual drift.
func TestMergeVrouterPartnerConfigFromAPI_PreservesUnconfiguredOptionals(t *testing.T) {
	ctx := context.Background()

	stateBGP := nullBGPModel()
	stateBGP.PeerAsn = types.Int64Value(64512)
	stateBGP.LocalIPAddress = types.StringValue("10.0.0.1")
	stateBGP.PeerIPAddress = types.StringValue("10.0.0.2")
	stateBGP.Password = types.StringValue("secret")
	stateBGP.MedIn = types.Int64Value(100)

	existing := buildVrouterPartnerConfigObject(t, ctx, []bgpConnectionConfigModel{stateBGP})

	vrConn := megaport.CSPConnectionVirtualRouter{
		Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
			{
				BGPConnections: []megaport.BgpConnectionConfig{
					{
						PeerAsn:        64512,
						LocalIpAddress: "10.0.0.1",
						PeerIpAddress:  "10.0.0.2",
						Shutdown:       true,
						BfdEnabled:     true,
						MedIn:          150,
						MedOut:         999,
						Description:    "from-api",
					},
				},
			},
		},
	}

	result, diags := mergeVrouterPartnerConfigFromAPI(ctx, vrConn, existing, "", nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags.Errors())
	}

	bgp := extractBGPFromResult(t, ctx, result)

	// Configured fields track the API.
	assertInt64(t, "peer_asn", bgp.PeerAsn, 64512)
	assertInt64(t, "med_in", bgp.MedIn, 150)
	assertString(t, "password", bgp.Password, "secret") // preserved; API never returns it

	// Unconfigured Optional-only fields must remain null.
	if !bgp.Shutdown.IsNull() {
		t.Errorf("shutdown: expected null (unconfigured), got %v", bgp.Shutdown.ValueBool())
	}
	if !bgp.BfdEnabled.IsNull() {
		t.Errorf("bfd_enabled: expected null (unconfigured), got %v", bgp.BfdEnabled.ValueBool())
	}
	if !bgp.MedOut.IsNull() {
		t.Errorf("med_out: expected null (unconfigured), got %d", bgp.MedOut.ValueInt64())
	}
	if !bgp.Description.IsNull() {
		t.Errorf("description: expected null (unconfigured), got %q", bgp.Description.ValueString())
	}
}

// TestMergeVrouterPartnerConfigFromAPI_MatchesByPeerIP verifies that state BGP
// connections are matched to their API counterparts by peer IP, not position.
// The API here returns the two connections in the opposite order from state, so
// positional matching would attach each peer's ASN to the wrong connection.
func TestMergeVrouterPartnerConfigFromAPI_MatchesByPeerIP(t *testing.T) {
	ctx := context.Background()

	bgpA := nullBGPModel()
	bgpA.PeerAsn = types.Int64Value(64512)
	bgpA.PeerIPAddress = types.StringValue("10.0.0.2")

	bgpB := nullBGPModel()
	bgpB.PeerAsn = types.Int64Value(64513)
	bgpB.PeerIPAddress = types.StringValue("10.0.1.2")

	existing := buildVrouterPartnerConfigObject(t, ctx, []bgpConnectionConfigModel{bgpA, bgpB})

	// API returns the connections swapped relative to state order.
	vrConn := megaport.CSPConnectionVirtualRouter{
		Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
			{
				BGPConnections: []megaport.BgpConnectionConfig{
					{PeerAsn: 65013, PeerIpAddress: "10.0.1.2"},
					{PeerAsn: 65012, PeerIpAddress: "10.0.0.2"},
				},
			},
		},
	}

	result, diags := mergeVrouterPartnerConfigFromAPI(ctx, vrConn, existing, "", nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags.Errors())
	}

	iface := extractInterfaceFromResult(t, ctx, result)
	bgps := []*bgpConnectionConfigModel{}
	if d := iface.BgpConnections.ElementsAs(ctx, &bgps, false); d.HasError() {
		t.Fatalf("failed to extract BGP connections: %s", d.Errors())
	}
	if len(bgps) != 2 {
		t.Fatalf("expected 2 BGP connections, got %d", len(bgps))
	}

	// State connection 0 (peer 10.0.0.2) must take the API ASN for 10.0.0.2.
	assertString(t, "bgp0 peer_ip", bgps[0].PeerIPAddress, "10.0.0.2")
	assertInt64(t, "bgp0 peer_asn", bgps[0].PeerAsn, 65012)
	assertString(t, "bgp1 peer_ip", bgps[1].PeerIPAddress, "10.0.1.2")
	assertInt64(t, "bgp1 peer_asn", bgps[1].PeerAsn, 65013)
}

// TestMergeVrouterPartnerConfigFromAPI_PreservesUnresolvedPrefixFilter verifies
// that when the API returns a non-zero prefix filter ID the provider can't
// resolve to a name (no client to look it up), the existing state name is kept
// rather than nulled, avoiding false drift against a still-attached filter.
func TestMergeVrouterPartnerConfigFromAPI_PreservesUnresolvedPrefixFilter(t *testing.T) {
	ctx := context.Background()

	stateBGP := nullBGPModel()
	stateBGP.PeerIPAddress = types.StringValue("10.0.0.2")
	stateBGP.ImportWhitelist = types.StringValue("my-import-filter")

	existing := buildVrouterPartnerConfigObject(t, ctx, []bgpConnectionConfigModel{stateBGP})

	vrConn := megaport.CSPConnectionVirtualRouter{
		Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
			{
				BGPConnections: []megaport.BgpConnectionConfig{
					{PeerIpAddress: "10.0.0.2", ImportWhitelist: 4242},
				},
			},
		},
	}

	// client is nil, so the ID→name map is empty and 4242 can't be resolved.
	result, diags := mergeVrouterPartnerConfigFromAPI(ctx, vrConn, existing, "", nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags.Errors())
	}

	bgp := extractBGPFromResult(t, ctx, result)
	assertString(t, "import_whitelist", bgp.ImportWhitelist, "my-import-filter")
}

// TestMatchVrouterCSPConn covers content-based connection matching: highest BGP
// IP overlap wins, ties/no-overlap fall back to the first unused connection,
// and an all-used input returns -1.
func TestMatchVrouterCSPConn(t *testing.T) {
	conns := []megaport.CSPConnectionVirtualRouter{
		{Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
			{BGPConnections: []megaport.BgpConnectionConfig{{LocalIpAddress: "10.0.0.1", PeerIpAddress: "10.0.0.2"}}},
		}},
		{Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
			{BGPConnections: []megaport.BgpConnectionConfig{{LocalIpAddress: "10.0.1.1", PeerIpAddress: "10.0.1.2"}}},
		}},
	}

	// Matches the second connection by IP overlap even though it is not first.
	used := make([]bool, len(conns))
	if got := matchVrouterCSPConn(conns, used, map[string]bool{"10.0.1.1": true}); got != 1 {
		t.Errorf("expected index 1 by IP overlap, got %d", got)
	}

	// No overlap falls back to the first unused connection.
	used = []bool{true, false}
	if got := matchVrouterCSPConn(conns, used, map[string]bool{}); got != 1 {
		t.Errorf("expected fallback to first unused index 1, got %d", got)
	}

	// Everything used returns -1.
	used = []bool{true, true}
	if got := matchVrouterCSPConn(conns, used, map[string]bool{"10.0.0.1": true}); got != -1 {
		t.Errorf("expected -1 when all connections used, got %d", got)
	}
}

// TestMergeVrouterPartnerConfigFromAPI_PreservesEchoOmittedFields covers the
// omitempty BGP fields (peer_type, description, med_in, med_out, export_policy,
// as_path_prepend_count). When the read endpoint omits them (zero/empty value),
// the configured state must be preserved rather than reset to zero, which would
// otherwise surface false drift on every refresh.
func TestMergeVrouterPartnerConfigFromAPI_PreservesEchoOmittedFields(t *testing.T) {
	ctx := context.Background()

	stateBGP := nullBGPModel()
	stateBGP.PeerIPAddress = types.StringValue("10.0.0.2")
	stateBGP.PeerType = types.StringValue("NON_CLOUD")
	stateBGP.Description = types.StringValue("my-session")
	stateBGP.MedIn = types.Int64Value(100)
	stateBGP.MedOut = types.Int64Value(200)
	stateBGP.ExportPolicy = types.StringValue("permit")
	stateBGP.AsPathPrependCount = types.Int64Value(3)

	existing := buildVrouterPartnerConfigObject(t, ctx, []bgpConnectionConfigModel{stateBGP})

	// API echoes the peer IP but omits every omitempty field (zero/empty).
	vrConn := megaport.CSPConnectionVirtualRouter{
		Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
			{
				BGPConnections: []megaport.BgpConnectionConfig{
					{PeerIpAddress: "10.0.0.2"},
				},
			},
		},
	}

	result, diags := mergeVrouterPartnerConfigFromAPI(ctx, vrConn, existing, "", nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags.Errors())
	}

	bgp := extractBGPFromResult(t, ctx, result)
	assertString(t, "peer_type", bgp.PeerType, "NON_CLOUD")
	assertString(t, "description", bgp.Description, "my-session")
	assertInt64(t, "med_in", bgp.MedIn, 100)
	assertInt64(t, "med_out", bgp.MedOut, 200)
	assertString(t, "export_policy", bgp.ExportPolicy, "permit")
	assertInt64(t, "as_path_prepend_count", bgp.AsPathPrependCount, 3)
}

// buildVrouterConfigWithRoutes builds a vrouter partner config object with a single
// interface carrying the given IP routes (and no BGP connections).
func buildVrouterConfigWithRoutes(t *testing.T, ctx context.Context, routes []ipRouteModel) basetypes.ObjectValue {
	t.Helper()

	routeList, diags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(ipRouteAttrs), routes)
	if diags.HasError() {
		t.Fatalf("failed to create route list: %s", diags.Errors())
	}

	ifaceModel := vxcPartnerConfigInterfaceModel{
		IPAddresses:        types.ListNull(types.StringType),
		IPRoutes:           routeList,
		NatIPAddresses:     types.ListNull(types.StringType),
		Bfd:                types.ObjectNull(bfdConfigAttrs),
		BgpConnections:     types.ListNull(types.ObjectType{}.WithAttributeTypes(bgpVrouterConnectionConfig)),
		VLAN:               types.Int64Null(),
		IpMtu:              types.Int64Null(),
		IpSecTunnelOptions: types.ObjectNull(ipSecTunnelOptionsAttrs),
	}

	ifaceList, diags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(vxcVrouterInterfaceAttrs), []vxcPartnerConfigInterfaceModel{ifaceModel})
	if diags.HasError() {
		t.Fatalf("failed to create interface list: %s", diags.Errors())
	}

	vrouterModel := vxcPartnerConfigVrouterModel{Interfaces: ifaceList}
	vrouterObj, diags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, vrouterModel)
	if diags.HasError() {
		t.Fatalf("failed to create vrouter object: %s", diags.Errors())
	}

	partnerConfigModel := &vxcPartnerConfigurationModel{
		Partner:              types.StringValue("vrouter"),
		AWSPartnerConfig:     types.ObjectNull(vxcPartnerConfigAWSAttrs),
		AzurePartnerConfig:   types.ObjectNull(vxcPartnerConfigAzureAttrs),
		GooglePartnerConfig:  types.ObjectNull(vxcPartnerConfigGoogleAttrs),
		OraclePartnerConfig:  types.ObjectNull(vxcPartnerConfigOracleAttrs),
		IBMPartnerConfig:     types.ObjectNull(vxcPartnerConfigIbmAttrs),
		VrouterPartnerConfig: vrouterObj,
		PartnerAEndConfig:    types.ObjectNull(vxcPartnerConfigAEndAttrs),
	}

	obj, diags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, partnerConfigModel)
	if diags.HasError() {
		t.Fatalf("failed to create partner config object: %s", diags.Errors())
	}
	return obj
}

func TestMergeVrouterPartnerConfigFromAPI_PreservesIPRouteDescription(t *testing.T) {
	ctx := context.Background()

	// User configured one route with no description and one with a description.
	existing := buildVrouterConfigWithRoutes(t, ctx, []ipRouteModel{
		{Prefix: types.StringValue("10.0.0.0/24"), Description: types.StringNull(), NextHop: types.StringValue("10.0.0.1")},
		{Prefix: types.StringValue("10.0.1.0/24"), Description: types.StringValue("kept"), NextHop: types.StringValue("10.0.0.1")},
	})

	// API echoes both routes but omits both descriptions (omitempty -> "").
	vrConn := megaport.CSPConnectionVirtualRouter{
		Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
			{
				IPRoutes: []megaport.IpRoute{
					{Prefix: "10.0.0.0/24", NextHop: "10.0.0.1"},
					{Prefix: "10.0.1.0/24", NextHop: "10.0.0.1"},
				},
			},
		},
	}

	result, diags := mergeVrouterPartnerConfigFromAPI(ctx, vrConn, existing, "", nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags.Errors())
	}

	iface := extractInterfaceFromResult(t, ctx, result)
	var routes []ipRouteModel
	if d := iface.IPRoutes.ElementsAs(ctx, &routes, false); d.HasError() {
		t.Fatalf("failed to extract routes: %s", d.Errors())
	}
	require.Len(t, routes, 2)

	byPrefix := map[string]ipRouteModel{}
	for _, r := range routes {
		byPrefix[r.Prefix.ValueString()] = r
	}
	// Unset description stays null (no false drift); configured one is preserved.
	assert.True(t, byPrefix["10.0.0.0/24"].Description.IsNull(), "expected null description preserved as null")
	assertString(t, "description", byPrefix["10.0.1.0/24"].Description, "kept")
}
