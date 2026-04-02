package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	megaport "github.com/megaport/megaportgo"
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
		wantNull bool
		wantVal  string
	}{
		{
			name:     "zero ID returns null",
			id:       0,
			pflMap:   pflMap,
			wantNull: true,
		},
		{
			name:     "known ID returns name",
			id:       100,
			pflMap:   pflMap,
			wantNull: false,
			wantVal:  "whitelist-filter",
		},
		{
			name:     "another known ID returns name",
			id:       200,
			pflMap:   pflMap,
			wantNull: false,
			wantVal:  "blacklist-filter",
		},
		{
			name:     "unknown non-zero ID returns null",
			id:       999,
			pflMap:   pflMap,
			wantNull: true,
		},
		{
			name:     "empty map with non-zero ID returns null",
			id:       100,
			pflMap:   map[int]string{},
			wantNull: true,
		},
		{
			name:     "nil map with zero ID returns null",
			id:       0,
			pflMap:   nil,
			wantNull: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := prefixFilterIDToName(tt.id, tt.pflMap)
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
		IPAddresses:    types.ListNull(types.StringType),
		IPRoutes:       types.ListNull(types.ObjectType{}.WithAttributeTypes(ipRouteAttrs)),
		NatIPAddresses: types.ListNull(types.StringType),
		Bfd:            types.ObjectNull(bfdConfigAttrs),
		BgpConnections: bgpList,
		VLAN:           types.Int64Null(),
		IpMtu:          types.Int64Null(),
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

func TestExtractExistingBGPPasswords(t *testing.T) {
	ctx := context.Background()

	t.Run("null partner config returns empty map", func(t *testing.T) {
		nullObj := types.ObjectNull(vxcPartnerConfigAttrs)
		passwords := extractExistingBGPPasswords(ctx, nullObj, "vrouter")
		if len(passwords) != 0 {
			t.Errorf("expected empty map, got %d entries", len(passwords))
		}
	})

	t.Run("vrouter with single BGP connection preserves password", func(t *testing.T) {
		bgpModels := []bgpConnectionConfigModel{
			{
				PeerAsn:            types.Int64Value(64512),
				LocalAsn:           types.Int64Null(),
				PeerType:           types.StringValue("NON_CLOUD"),
				LocalIPAddress:     types.StringValue("10.0.0.1"),
				PeerIPAddress:      types.StringValue("10.0.0.2"),
				Password:           types.StringValue("secretPass123"),
				Shutdown:           types.BoolValue(false),
				Description:        types.StringValue("test"),
				MedIn:              types.Int64Value(100),
				MedOut:             types.Int64Value(100),
				BfdEnabled:         types.BoolValue(false),
				ExportPolicy:       types.StringValue("permit"),
				PermitExportTo:     types.ListNull(types.StringType),
				DenyExportTo:       types.ListNull(types.StringType),
				ImportWhitelist:    types.StringNull(),
				ImportBlacklist:    types.StringNull(),
				ExportWhitelist:    types.StringNull(),
				ExportBlacklist:    types.StringNull(),
				AsPathPrependCount: types.Int64Value(0),
			},
		}

		obj := buildVrouterPartnerConfigObject(t, ctx, bgpModels)
		passwords := extractExistingBGPPasswords(ctx, obj, "vrouter")

		key := "0:0"
		pw, ok := passwords[key]
		if !ok {
			t.Fatalf("expected password at key %q, not found", key)
		}
		if pw.ValueString() != "secretPass123" {
			t.Errorf("expected password %q, got %q", "secretPass123", pw.ValueString())
		}
	})

	t.Run("vrouter with multiple BGP connections preserves all passwords", func(t *testing.T) {
		bgpModels := []bgpConnectionConfigModel{
			{
				PeerAsn:            types.Int64Value(64512),
				LocalAsn:           types.Int64Null(),
				PeerType:           types.StringValue("NON_CLOUD"),
				LocalIPAddress:     types.StringValue("10.0.0.1"),
				PeerIPAddress:      types.StringValue("10.0.0.2"),
				Password:           types.StringValue("pass1"),
				Shutdown:           types.BoolValue(false),
				Description:        types.StringValue("bgp1"),
				MedIn:              types.Int64Value(0),
				MedOut:             types.Int64Value(0),
				BfdEnabled:         types.BoolValue(false),
				ExportPolicy:       types.StringValue(""),
				PermitExportTo:     types.ListNull(types.StringType),
				DenyExportTo:       types.ListNull(types.StringType),
				ImportWhitelist:    types.StringNull(),
				ImportBlacklist:    types.StringNull(),
				ExportWhitelist:    types.StringNull(),
				ExportBlacklist:    types.StringNull(),
				AsPathPrependCount: types.Int64Value(0),
			},
			{
				PeerAsn:            types.Int64Value(64513),
				LocalAsn:           types.Int64Null(),
				PeerType:           types.StringValue("NON_CLOUD"),
				LocalIPAddress:     types.StringValue("10.0.1.1"),
				PeerIPAddress:      types.StringValue("10.0.1.2"),
				Password:           types.StringValue("pass2"),
				Shutdown:           types.BoolValue(false),
				Description:        types.StringValue("bgp2"),
				MedIn:              types.Int64Value(0),
				MedOut:             types.Int64Value(0),
				BfdEnabled:         types.BoolValue(false),
				ExportPolicy:       types.StringValue(""),
				PermitExportTo:     types.ListNull(types.StringType),
				DenyExportTo:       types.ListNull(types.StringType),
				ImportWhitelist:    types.StringNull(),
				ImportBlacklist:    types.StringNull(),
				ExportWhitelist:    types.StringNull(),
				ExportBlacklist:    types.StringNull(),
				AsPathPrependCount: types.Int64Value(0),
			},
		}

		obj := buildVrouterPartnerConfigObject(t, ctx, bgpModels)
		passwords := extractExistingBGPPasswords(ctx, obj, "vrouter")

		if len(passwords) != 2 {
			t.Fatalf("expected 2 passwords, got %d", len(passwords))
		}
		if passwords["0:0"].ValueString() != "pass1" {
			t.Errorf("expected pass1, got %q", passwords["0:0"].ValueString())
		}
		if passwords["0:1"].ValueString() != "pass2" {
			t.Errorf("expected pass2, got %q", passwords["0:1"].ValueString())
		}
	})

	t.Run("vrouter with null password preserves null", func(t *testing.T) {
		bgpModels := []bgpConnectionConfigModel{
			{
				PeerAsn:            types.Int64Value(64512),
				LocalAsn:           types.Int64Null(),
				PeerType:           types.StringValue("NON_CLOUD"),
				LocalIPAddress:     types.StringValue("10.0.0.1"),
				PeerIPAddress:      types.StringValue("10.0.0.2"),
				Password:           types.StringNull(),
				Shutdown:           types.BoolValue(false),
				Description:        types.StringValue("test"),
				MedIn:              types.Int64Value(0),
				MedOut:             types.Int64Value(0),
				BfdEnabled:         types.BoolValue(false),
				ExportPolicy:       types.StringValue(""),
				PermitExportTo:     types.ListNull(types.StringType),
				DenyExportTo:       types.ListNull(types.StringType),
				ImportWhitelist:    types.StringNull(),
				ImportBlacklist:    types.StringNull(),
				ExportWhitelist:    types.StringNull(),
				ExportBlacklist:    types.StringNull(),
				AsPathPrependCount: types.Int64Value(0),
			},
		}

		obj := buildVrouterPartnerConfigObject(t, ctx, bgpModels)
		passwords := extractExistingBGPPasswords(ctx, obj, "vrouter")

		pw, ok := passwords["0:0"]
		if !ok {
			t.Fatal("expected password entry at 0:0, not found")
		}
		if !pw.IsNull() {
			t.Errorf("expected null password, got %q", pw.ValueString())
		}
	})

	t.Run("wrong partner type returns empty map", func(t *testing.T) {
		bgpModels := []bgpConnectionConfigModel{
			{
				PeerAsn:            types.Int64Value(64512),
				LocalAsn:           types.Int64Null(),
				PeerType:           types.StringValue("NON_CLOUD"),
				LocalIPAddress:     types.StringValue("10.0.0.1"),
				PeerIPAddress:      types.StringValue("10.0.0.2"),
				Password:           types.StringValue("secret"),
				Shutdown:           types.BoolValue(false),
				Description:        types.StringValue("test"),
				MedIn:              types.Int64Value(0),
				MedOut:             types.Int64Value(0),
				BfdEnabled:         types.BoolValue(false),
				ExportPolicy:       types.StringValue(""),
				PermitExportTo:     types.ListNull(types.StringType),
				DenyExportTo:       types.ListNull(types.StringType),
				ImportWhitelist:    types.StringNull(),
				ImportBlacklist:    types.StringNull(),
				ExportWhitelist:    types.StringNull(),
				ExportBlacklist:    types.StringNull(),
				AsPathPrependCount: types.Int64Value(0),
			},
		}

		// Build a vrouter config but query as "a-end" - the vrouter_config will be populated
		// but a-end config will be null, so it should return empty
		obj := buildVrouterPartnerConfigObject(t, ctx, bgpModels)
		passwords := extractExistingBGPPasswords(ctx, obj, "a-end")

		if len(passwords) != 0 {
			t.Errorf("expected empty map for wrong partner type, got %d entries", len(passwords))
		}
	})
}

func TestReconstructVrouterPartnerConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("empty interfaces returns null", func(t *testing.T) {
		vrConn := megaport.CSPConnectionVirtualRouter{
			Interfaces: []megaport.CSPConnectionVirtualRouterInterface{},
		}
		existingConfig := types.ObjectNull(vxcPartnerConfigAttrs)

		result, diags := reconstructVrouterPartnerConfig(ctx, vrConn, existingConfig, "", nil, "vrouter")
		if diags.HasError() {
			t.Fatalf("unexpected error: %s", diags.Errors())
		}
		if !result.IsNull() {
			t.Error("expected null result for empty interfaces")
		}
	})

	t.Run("single BGP connection maps all fields correctly", func(t *testing.T) {
		localAsn := 64555
		vrConn := megaport.CSPConnectionVirtualRouter{
			Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
				{
					IPAddresses: []string{"10.0.0.1/30"},
					BGPConnections: []megaport.BgpConnectionConfig{
						{
							PeerAsn:            64512,
							LocalAsn:           &localAsn,
							LocalIpAddress:     "10.0.0.1",
							PeerIpAddress:      "10.0.0.2",
							Shutdown:           false,
							Description:        "Test BGP",
							MedIn:              100,
							MedOut:             200,
							BfdEnabled:         true,
							ExportPolicy:       "permit",
							AsPathPrependCount: 3,
							PeerType:           "NON_CLOUD",
						},
					},
				},
			},
		}

		// Create existing config with a password to verify preservation
		bgpModels := []bgpConnectionConfigModel{
			{
				PeerAsn:            types.Int64Value(64512),
				LocalAsn:           types.Int64Value(64555),
				PeerType:           types.StringValue("NON_CLOUD"),
				LocalIPAddress:     types.StringValue("10.0.0.1"),
				PeerIPAddress:      types.StringValue("10.0.0.2"),
				Password:           types.StringValue("mySecret"),
				Shutdown:           types.BoolValue(false),
				Description:        types.StringValue("Test BGP"),
				MedIn:              types.Int64Value(100),
				MedOut:             types.Int64Value(200),
				BfdEnabled:         types.BoolValue(true),
				ExportPolicy:       types.StringValue("permit"),
				PermitExportTo:     types.ListNull(types.StringType),
				DenyExportTo:       types.ListNull(types.StringType),
				ImportWhitelist:    types.StringNull(),
				ImportBlacklist:    types.StringNull(),
				ExportWhitelist:    types.StringNull(),
				ExportBlacklist:    types.StringNull(),
				AsPathPrependCount: types.Int64Value(3),
			},
		}
		existingConfig := buildVrouterPartnerConfigObject(t, ctx, bgpModels)

		result, diags := reconstructVrouterPartnerConfig(ctx, vrConn, existingConfig, "", nil, "vrouter")
		if diags.HasError() {
			t.Fatalf("unexpected error: %s", diags.Errors())
		}
		if result.IsNull() {
			t.Fatal("expected non-null result")
		}

		// Extract and verify the reconstructed partner config
		partnerModel := &vxcPartnerConfigurationModel{}
		pDiags := result.As(ctx, partnerModel, basetypes.ObjectAsOptions{})
		if pDiags.HasError() {
			t.Fatalf("failed to extract partner config: %s", pDiags.Errors())
		}

		if partnerModel.Partner.ValueString() != "vrouter" {
			t.Errorf("expected partner 'vrouter', got %q", partnerModel.Partner.ValueString())
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
		if len(ifaceModels) != 1 {
			t.Fatalf("expected 1 interface, got %d", len(ifaceModels))
		}

		// Verify IP addresses
		ipAddrs := []string{}
		ipDiags := ifaceModels[0].IPAddresses.ElementsAs(ctx, &ipAddrs, false)
		if ipDiags.HasError() {
			t.Fatalf("failed to extract IP addresses: %s", ipDiags.Errors())
		}
		if len(ipAddrs) != 1 || ipAddrs[0] != "10.0.0.1/30" {
			t.Errorf("expected IP address [10.0.0.1/30], got %v", ipAddrs)
		}

		// Verify BGP connections
		bgpConns := []*bgpConnectionConfigModel{}
		bgpDiags := ifaceModels[0].BgpConnections.ElementsAs(ctx, &bgpConns, false)
		if bgpDiags.HasError() {
			t.Fatalf("failed to extract BGP connections: %s", bgpDiags.Errors())
		}
		if len(bgpConns) != 1 {
			t.Fatalf("expected 1 BGP connection, got %d", len(bgpConns))
		}

		bgp := bgpConns[0]
		assertInt64(t, "peer_asn", bgp.PeerAsn, 64512)
		assertInt64(t, "local_asn", bgp.LocalAsn, 64555)
		assertString(t, "local_ip_address", bgp.LocalIPAddress, "10.0.0.1")
		assertString(t, "peer_ip_address", bgp.PeerIPAddress, "10.0.0.2")
		assertString(t, "password", bgp.Password, "mySecret") // preserved from state
		assertBool(t, "shutdown", bgp.Shutdown, false)
		assertString(t, "description", bgp.Description, "Test BGP")
		assertInt64(t, "med_in", bgp.MedIn, 100)
		assertInt64(t, "med_out", bgp.MedOut, 200)
		assertBool(t, "bfd_enabled", bgp.BfdEnabled, true)
		assertString(t, "export_policy", bgp.ExportPolicy, "permit")
		assertInt64(t, "as_path_prepend_count", bgp.AsPathPrependCount, 3)
		assertString(t, "peer_type", bgp.PeerType, "NON_CLOUD")
	})

	t.Run("null LocalAsn maps to null", func(t *testing.T) {
		vrConn := megaport.CSPConnectionVirtualRouter{
			Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
				{
					BGPConnections: []megaport.BgpConnectionConfig{
						{
							PeerAsn:        64512,
							LocalAsn:       nil, // null
							LocalIpAddress: "10.0.0.1",
							PeerIpAddress:  "10.0.0.2",
						},
					},
				},
			},
		}
		existingConfig := types.ObjectNull(vxcPartnerConfigAttrs)

		result, diags := reconstructVrouterPartnerConfig(ctx, vrConn, existingConfig, "", nil, "vrouter")
		if diags.HasError() {
			t.Fatalf("unexpected error: %s", diags.Errors())
		}

		bgp := extractBGPFromResult(t, ctx, result)
		if !bgp.LocalAsn.IsNull() {
			t.Errorf("expected null LocalAsn, got %d", bgp.LocalAsn.ValueInt64())
		}
	})

	t.Run("password defaults to null when no existing state", func(t *testing.T) {
		vrConn := megaport.CSPConnectionVirtualRouter{
			Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
				{
					BGPConnections: []megaport.BgpConnectionConfig{
						{
							PeerAsn:        64512,
							LocalIpAddress: "10.0.0.1",
							PeerIpAddress:  "10.0.0.2",
							Password:       "api-wont-return-this",
						},
					},
				},
			},
		}
		existingConfig := types.ObjectNull(vxcPartnerConfigAttrs)

		result, diags := reconstructVrouterPartnerConfig(ctx, vrConn, existingConfig, "", nil, "vrouter")
		if diags.HasError() {
			t.Fatalf("unexpected error: %s", diags.Errors())
		}

		bgp := extractBGPFromResult(t, ctx, result)
		if !bgp.Password.IsNull() {
			t.Errorf("expected null password when no existing state, got %q", bgp.Password.ValueString())
		}
	})

	t.Run("prefix filter list IDs are resolved to names", func(t *testing.T) {
		vrConn := megaport.CSPConnectionVirtualRouter{
			Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
				{
					BGPConnections: []megaport.BgpConnectionConfig{
						{
							PeerAsn:         64512,
							LocalIpAddress:  "10.0.0.1",
							PeerIpAddress:   "10.0.0.2",
							ImportWhitelist: 100,
							ImportBlacklist: 200,
							ExportWhitelist: 300,
							ExportBlacklist: 0, // unset
						},
					},
				},
			},
		}

		// We can't pass a real client, but we can test the pflMap logic
		// by calling reconstructVrouterPartnerConfig with client=nil (which skips the API call)
		// and then checking that all prefix filter fields are null (since pflMap will be empty).
		existingConfig := types.ObjectNull(vxcPartnerConfigAttrs)

		result, diags := reconstructVrouterPartnerConfig(ctx, vrConn, existingConfig, "", nil, "vrouter")
		if diags.HasError() {
			t.Fatalf("unexpected error: %s", diags.Errors())
		}

		bgp := extractBGPFromResult(t, ctx, result)

		// With no client, all prefix filter IDs resolve to null (pflMap is empty)
		if !bgp.ImportWhitelist.IsNull() {
			t.Errorf("expected null ImportWhitelist (no pflMap), got %q", bgp.ImportWhitelist.ValueString())
		}
		if !bgp.ImportBlacklist.IsNull() {
			t.Errorf("expected null ImportBlacklist (no pflMap), got %q", bgp.ImportBlacklist.ValueString())
		}
		if !bgp.ExportWhitelist.IsNull() {
			t.Errorf("expected null ExportWhitelist (no pflMap), got %q", bgp.ExportWhitelist.ValueString())
		}
		if !bgp.ExportBlacklist.IsNull() {
			t.Errorf("expected null ExportBlacklist (unset), got %q", bgp.ExportBlacklist.ValueString())
		}
	})

	t.Run("BFD configuration is mapped correctly", func(t *testing.T) {
		vrConn := megaport.CSPConnectionVirtualRouter{
			Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
				{
					BFD: megaport.BfdConfig{
						TxInterval: 500,
						RxInterval: 400,
						Multiplier: 5,
					},
					BGPConnections: []megaport.BgpConnectionConfig{
						{
							PeerAsn:        64512,
							LocalIpAddress: "10.0.0.1",
							PeerIpAddress:  "10.0.0.2",
						},
					},
				},
			},
		}
		existingConfig := types.ObjectNull(vxcPartnerConfigAttrs)

		result, diags := reconstructVrouterPartnerConfig(ctx, vrConn, existingConfig, "", nil, "vrouter")
		if diags.HasError() {
			t.Fatalf("unexpected error: %s", diags.Errors())
		}

		iface := extractInterfaceFromResult(t, ctx, result)
		if iface.Bfd.IsNull() {
			t.Fatal("expected non-null BFD config")
		}

		bfd := &bfdConfigModel{}
		bfdDiags := iface.Bfd.As(ctx, bfd, basetypes.ObjectAsOptions{})
		if bfdDiags.HasError() {
			t.Fatalf("failed to extract BFD config: %s", bfdDiags.Errors())
		}
		assertInt64(t, "tx_interval", bfd.TxInterval, 500)
		assertInt64(t, "rx_interval", bfd.RxInterval, 400)
		assertInt64(t, "multiplier", bfd.Multiplier, 5)
	})

	t.Run("zero BFD values produce null BFD object", func(t *testing.T) {
		vrConn := megaport.CSPConnectionVirtualRouter{
			Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
				{
					BFD: megaport.BfdConfig{
						TxInterval: 0,
						RxInterval: 0,
						Multiplier: 0,
					},
					BGPConnections: []megaport.BgpConnectionConfig{
						{
							PeerAsn:        64512,
							LocalIpAddress: "10.0.0.1",
							PeerIpAddress:  "10.0.0.2",
						},
					},
				},
			},
		}
		existingConfig := types.ObjectNull(vxcPartnerConfigAttrs)

		result, diags := reconstructVrouterPartnerConfig(ctx, vrConn, existingConfig, "", nil, "vrouter")
		if diags.HasError() {
			t.Fatalf("unexpected error: %s", diags.Errors())
		}

		iface := extractInterfaceFromResult(t, ctx, result)
		if !iface.Bfd.IsNull() {
			t.Error("expected null BFD config when all values are zero")
		}
	})

	t.Run("IP routes are mapped correctly", func(t *testing.T) {
		vrConn := megaport.CSPConnectionVirtualRouter{
			Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
				{
					IPRoutes: []megaport.IpRoute{
						{
							Prefix:      "192.168.1.0/24",
							Description: "Route to LAN",
							NextHop:     "10.0.0.2",
						},
					},
					BGPConnections: []megaport.BgpConnectionConfig{
						{
							PeerAsn:        64512,
							LocalIpAddress: "10.0.0.1",
							PeerIpAddress:  "10.0.0.2",
						},
					},
				},
			},
		}
		existingConfig := types.ObjectNull(vxcPartnerConfigAttrs)

		result, diags := reconstructVrouterPartnerConfig(ctx, vrConn, existingConfig, "", nil, "vrouter")
		if diags.HasError() {
			t.Fatalf("unexpected error: %s", diags.Errors())
		}

		iface := extractInterfaceFromResult(t, ctx, result)
		if iface.IPRoutes.IsNull() {
			t.Fatal("expected non-null IP routes")
		}

		routes := []*ipRouteModel{}
		routeDiags := iface.IPRoutes.ElementsAs(ctx, &routes, false)
		if routeDiags.HasError() {
			t.Fatalf("failed to extract IP routes: %s", routeDiags.Errors())
		}
		if len(routes) != 1 {
			t.Fatalf("expected 1 route, got %d", len(routes))
		}
		assertString(t, "prefix", routes[0].Prefix, "192.168.1.0/24")
		assertString(t, "description", routes[0].Description, "Route to LAN")
		assertString(t, "next_hop", routes[0].NextHop, "10.0.0.2")
	})

	t.Run("NAT IP addresses are mapped correctly", func(t *testing.T) {
		vrConn := megaport.CSPConnectionVirtualRouter{
			Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
				{
					NatIPAddresses: []string{"203.0.113.1", "203.0.113.2"},
					BGPConnections: []megaport.BgpConnectionConfig{
						{
							PeerAsn:        64512,
							LocalIpAddress: "10.0.0.1",
							PeerIpAddress:  "10.0.0.2",
						},
					},
				},
			},
		}
		existingConfig := types.ObjectNull(vxcPartnerConfigAttrs)

		result, diags := reconstructVrouterPartnerConfig(ctx, vrConn, existingConfig, "", nil, "vrouter")
		if diags.HasError() {
			t.Fatalf("unexpected error: %s", diags.Errors())
		}

		iface := extractInterfaceFromResult(t, ctx, result)
		if iface.NatIPAddresses.IsNull() {
			t.Fatal("expected non-null NAT IP addresses")
		}

		natAddrs := []string{}
		natDiags := iface.NatIPAddresses.ElementsAs(ctx, &natAddrs, false)
		if natDiags.HasError() {
			t.Fatalf("failed to extract NAT addresses: %s", natDiags.Errors())
		}
		if len(natAddrs) != 2 || natAddrs[0] != "203.0.113.1" || natAddrs[1] != "203.0.113.2" {
			t.Errorf("expected [203.0.113.1, 203.0.113.2], got %v", natAddrs)
		}
	})

	t.Run("PermitExportTo and DenyExportTo are mapped", func(t *testing.T) {
		vrConn := megaport.CSPConnectionVirtualRouter{
			Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
				{
					BGPConnections: []megaport.BgpConnectionConfig{
						{
							PeerAsn:        64512,
							LocalIpAddress: "10.0.0.1",
							PeerIpAddress:  "10.0.0.2",
							PermitExportTo: []string{"10.0.1.0/24", "10.0.2.0/24"},
							DenyExportTo:   []string{"192.168.0.0/16"},
						},
					},
				},
			},
		}
		existingConfig := types.ObjectNull(vxcPartnerConfigAttrs)

		result, diags := reconstructVrouterPartnerConfig(ctx, vrConn, existingConfig, "", nil, "vrouter")
		if diags.HasError() {
			t.Fatalf("unexpected error: %s", diags.Errors())
		}

		bgp := extractBGPFromResult(t, ctx, result)

		permitExportTo := []string{}
		pDiags := bgp.PermitExportTo.ElementsAs(ctx, &permitExportTo, false)
		if pDiags.HasError() {
			t.Fatalf("failed to extract PermitExportTo: %s", pDiags.Errors())
		}
		if len(permitExportTo) != 2 {
			t.Errorf("expected 2 PermitExportTo entries, got %d", len(permitExportTo))
		}

		denyExportTo := []string{}
		dDiags := bgp.DenyExportTo.ElementsAs(ctx, &denyExportTo, false)
		if dDiags.HasError() {
			t.Fatalf("failed to extract DenyExportTo: %s", dDiags.Errors())
		}
		if len(denyExportTo) != 1 || denyExportTo[0] != "192.168.0.0/16" {
			t.Errorf("expected [192.168.0.0/16], got %v", denyExportTo)
		}
	})

	t.Run("empty PermitExportTo and DenyExportTo are null", func(t *testing.T) {
		vrConn := megaport.CSPConnectionVirtualRouter{
			Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
				{
					BGPConnections: []megaport.BgpConnectionConfig{
						{
							PeerAsn:        64512,
							LocalIpAddress: "10.0.0.1",
							PeerIpAddress:  "10.0.0.2",
							PermitExportTo: nil,
							DenyExportTo:   nil,
						},
					},
				},
			},
		}
		existingConfig := types.ObjectNull(vxcPartnerConfigAttrs)

		result, diags := reconstructVrouterPartnerConfig(ctx, vrConn, existingConfig, "", nil, "vrouter")
		if diags.HasError() {
			t.Fatalf("unexpected error: %s", diags.Errors())
		}

		bgp := extractBGPFromResult(t, ctx, result)
		if !bgp.PermitExportTo.IsNull() {
			t.Error("expected null PermitExportTo")
		}
		if !bgp.DenyExportTo.IsNull() {
			t.Error("expected null DenyExportTo")
		}
	})

	t.Run("multiple interfaces are mapped", func(t *testing.T) {
		vrConn := megaport.CSPConnectionVirtualRouter{
			Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
				{
					IPAddresses: []string{"10.0.0.1/30"},
					BGPConnections: []megaport.BgpConnectionConfig{
						{
							PeerAsn:        64512,
							LocalIpAddress: "10.0.0.1",
							PeerIpAddress:  "10.0.0.2",
						},
					},
				},
				{
					IPAddresses: []string{"10.0.1.1/30"},
					BGPConnections: []megaport.BgpConnectionConfig{
						{
							PeerAsn:        64513,
							LocalIpAddress: "10.0.1.1",
							PeerIpAddress:  "10.0.1.2",
						},
					},
				},
			},
		}
		existingConfig := types.ObjectNull(vxcPartnerConfigAttrs)

		result, diags := reconstructVrouterPartnerConfig(ctx, vrConn, existingConfig, "", nil, "vrouter")
		if diags.HasError() {
			t.Fatalf("unexpected error: %s", diags.Errors())
		}

		partnerModel := &vxcPartnerConfigurationModel{}
		result.As(ctx, partnerModel, basetypes.ObjectAsOptions{})
		vrouterModel := &vxcPartnerConfigVrouterModel{}
		partnerModel.VrouterPartnerConfig.As(ctx, vrouterModel, basetypes.ObjectAsOptions{})

		ifaceModels := []*vxcPartnerConfigInterfaceModel{}
		vrouterModel.Interfaces.ElementsAs(ctx, &ifaceModels, false)

		if len(ifaceModels) != 2 {
			t.Fatalf("expected 2 interfaces, got %d", len(ifaceModels))
		}

		// Verify first interface
		bgpConns1 := []*bgpConnectionConfigModel{}
		ifaceModels[0].BgpConnections.ElementsAs(ctx, &bgpConns1, false)
		if len(bgpConns1) != 1 {
			t.Fatalf("expected 1 BGP connection on iface 0, got %d", len(bgpConns1))
		}
		assertInt64(t, "iface0 peer_asn", bgpConns1[0].PeerAsn, 64512)

		// Verify second interface
		bgpConns2 := []*bgpConnectionConfigModel{}
		ifaceModels[1].BgpConnections.ElementsAs(ctx, &bgpConns2, false)
		if len(bgpConns2) != 1 {
			t.Fatalf("expected 1 BGP connection on iface 1, got %d", len(bgpConns2))
		}
		assertInt64(t, "iface1 peer_asn", bgpConns2[0].PeerAsn, 64513)
	})

	t.Run("no BGP connections produce null list", func(t *testing.T) {
		vrConn := megaport.CSPConnectionVirtualRouter{
			Interfaces: []megaport.CSPConnectionVirtualRouterInterface{
				{
					IPAddresses:    []string{"10.0.0.1/30"},
					BGPConnections: nil,
				},
			},
		}
		existingConfig := types.ObjectNull(vxcPartnerConfigAttrs)

		result, diags := reconstructVrouterPartnerConfig(ctx, vrConn, existingConfig, "", nil, "vrouter")
		if diags.HasError() {
			t.Fatalf("unexpected error: %s", diags.Errors())
		}

		iface := extractInterfaceFromResult(t, ctx, result)
		if !iface.BgpConnections.IsNull() {
			t.Error("expected null BGP connections when none in API response")
		}
	})
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

func TestResolveUnknownsInObject(t *testing.T) {
	ctx := context.Background()

	t.Run("known values are preserved", func(t *testing.T) {
		attrTypes := map[string]attr.Type{
			"name": types.StringType,
			"age":  types.Int64Type,
		}
		obj, diags := types.ObjectValue(attrTypes, map[string]attr.Value{
			"name": types.StringValue("test"),
			"age":  types.Int64Value(42),
		})
		if diags.HasError() {
			t.Fatal(diags.Errors())
		}

		result := resolveUnknownsInObject(obj)
		attrs := result.Attributes()
		nameVal, ok := attrs["name"].(basetypes.StringValue)
		if !ok || nameVal.ValueString() != "test" {
			t.Errorf("expected name 'test', got %v", attrs["name"])
		}
		ageVal, ok := attrs["age"].(basetypes.Int64Value)
		if !ok || ageVal.ValueInt64() != 42 {
			t.Errorf("expected age 42, got %v", attrs["age"])
		}
	})

	t.Run("unknown values become null", func(t *testing.T) {
		attrTypes := map[string]attr.Type{
			"name":  types.StringType,
			"count": types.Int64Type,
			"flag":  types.BoolType,
		}
		obj, diags := types.ObjectValue(attrTypes, map[string]attr.Value{
			"name":  types.StringValue("known"),
			"count": types.Int64Unknown(),
			"flag":  types.BoolUnknown(),
		})
		if diags.HasError() {
			t.Fatal(diags.Errors())
		}

		result := resolveUnknownsInObject(obj)
		attrs := result.Attributes()

		nameVal, ok := attrs["name"].(basetypes.StringValue)
		if !ok || nameVal.ValueString() != "known" {
			t.Error("expected name to remain 'known'")
		}
		countVal, ok := attrs["count"].(basetypes.Int64Value)
		if !ok || !countVal.IsNull() {
			t.Error("expected count to be null after resolving unknown")
		}
		flagVal, ok := attrs["flag"].(basetypes.BoolValue)
		if !ok || !flagVal.IsNull() {
			t.Error("expected flag to be null after resolving unknown")
		}
	})

	t.Run("nested unknown objects become null", func(t *testing.T) {
		innerType := types.ObjectType{}.WithAttributeTypes(map[string]attr.Type{
			"value": types.StringType,
		})
		attrTypes := map[string]attr.Type{
			"name":  types.StringType,
			"inner": innerType,
		}
		obj, diags := types.ObjectValue(attrTypes, map[string]attr.Value{
			"name":  types.StringValue("test"),
			"inner": types.ObjectUnknown(map[string]attr.Type{"value": types.StringType}),
		})
		if diags.HasError() {
			t.Fatal(diags.Errors())
		}

		result := resolveUnknownsInObject(obj)
		attrs := result.Attributes()

		innerVal, ok := attrs["inner"].(basetypes.ObjectValue)
		if !ok || !innerVal.IsNull() {
			t.Error("expected unknown inner object to become null")
		}
	})

	t.Run("unknown list becomes null", func(t *testing.T) {
		attrTypes := map[string]attr.Type{
			"items": types.ListType{ElemType: types.StringType},
		}
		obj, diags := types.ObjectValue(attrTypes, map[string]attr.Value{
			"items": types.ListUnknown(types.StringType),
		})
		if diags.HasError() {
			t.Fatal(diags.Errors())
		}

		result := resolveUnknownsInObject(obj)
		attrs := result.Attributes()

		itemsVal, ok := attrs["items"].(basetypes.ListValue)
		if !ok || !itemsVal.IsNull() {
			t.Error("expected unknown list to become null")
		}
	})

	t.Run("null object stays null", func(t *testing.T) {
		nullObj := types.ObjectNull(map[string]attr.Type{"name": types.StringType})
		result := resolveUnknownsInObject(nullObj)
		if !result.IsNull() {
			t.Error("expected null object to stay null")
		}
	})

	t.Run("resolves unknowns in real partner config structure", func(t *testing.T) {
		// Build a partner config with some unknowns (simulating plan state)
		bgpModel := bgpConnectionConfigModel{
			PeerAsn:            types.Int64Value(64512),
			LocalAsn:           types.Int64Unknown(),  // unset by user, marked unknown
			PeerType:           types.StringUnknown(), // unset by user, marked unknown
			LocalIPAddress:     types.StringValue("10.0.0.1"),
			PeerIPAddress:      types.StringValue("10.0.0.2"),
			Password:           types.StringValue("secret"),
			Shutdown:           types.BoolValue(false),
			Description:        types.StringValue("test"),
			MedIn:              types.Int64Value(100),
			MedOut:             types.Int64Value(100),
			BfdEnabled:         types.BoolValue(false),
			ExportPolicy:       types.StringValue("permit"),
			PermitExportTo:     types.ListNull(types.StringType),
			DenyExportTo:       types.ListUnknown(types.StringType), // unset, marked unknown
			ImportWhitelist:    types.StringUnknown(),               // unset, marked unknown
			ImportBlacklist:    types.StringNull(),
			ExportWhitelist:    types.StringNull(),
			ExportBlacklist:    types.StringNull(),
			AsPathPrependCount: types.Int64Value(0),
		}

		obj := buildVrouterPartnerConfigObject(t, ctx, []bgpConnectionConfigModel{bgpModel})
		resolved := resolveUnknownsInObject(obj)

		// Verify the resolved object has no unknowns - extract BGP and check
		partnerModel := &vxcPartnerConfigurationModel{}
		resolved.As(ctx, partnerModel, basetypes.ObjectAsOptions{})
		vrouterModel := &vxcPartnerConfigVrouterModel{}
		partnerModel.VrouterPartnerConfig.As(ctx, vrouterModel, basetypes.ObjectAsOptions{})
		ifaceModels := []*vxcPartnerConfigInterfaceModel{}
		vrouterModel.Interfaces.ElementsAs(ctx, &ifaceModels, false)
		bgpConns := []*bgpConnectionConfigModel{}
		ifaceModels[0].BgpConnections.ElementsAs(ctx, &bgpConns, false)

		bgp := bgpConns[0]
		// Known values should be preserved
		assertInt64(t, "peer_asn", bgp.PeerAsn, 64512)
		assertString(t, "local_ip_address", bgp.LocalIPAddress, "10.0.0.1")
		assertString(t, "password", bgp.Password, "secret")
		assertInt64(t, "med_in", bgp.MedIn, 100)

		// Unknown values should now be null
		if !bgp.LocalAsn.IsNull() {
			t.Errorf("expected LocalAsn to be null after resolving unknown, got %v", bgp.LocalAsn)
		}
		if !bgp.PeerType.IsNull() {
			t.Errorf("expected PeerType to be null after resolving unknown, got %v", bgp.PeerType)
		}
		if !bgp.DenyExportTo.IsNull() {
			t.Errorf("expected DenyExportTo to be null after resolving unknown, got %v", bgp.DenyExportTo)
		}
		if !bgp.ImportWhitelist.IsNull() {
			t.Errorf("expected ImportWhitelist to be null after resolving unknown, got %v", bgp.ImportWhitelist)
		}
	})
}

// --- Test helpers ---

// extractBGPFromResult extracts the first BGP connection from a reconstructed partner config result.
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

// extractInterfaceFromResult extracts the first interface from a reconstructed partner config result.
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

func assertBool(t *testing.T, field string, got types.Bool, want bool) {
	t.Helper()
	if got.IsNull() {
		t.Errorf("%s: expected %v, got null", field, want)
		return
	}
	if got.ValueBool() != want {
		t.Errorf("%s: expected %v, got %v", field, want, got.ValueBool())
	}
}

// Verify unused import suppression
var _ = fmt.Sprintf
