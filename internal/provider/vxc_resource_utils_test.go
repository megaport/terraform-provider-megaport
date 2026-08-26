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

// TestCreateVrouterPartnerConfig_BgpAsOverride verifies the model -> SDK mapping
// for the optional as_override BGP field: set true/false round-trips the value,
// and unset (null) leaves the pointer nil so the API default applies.
func TestCreateVrouterPartnerConfig_BgpAsOverride(t *testing.T) {
	ctx := context.Background()

	newBgp := func(asOverride types.Bool) bgpConnectionConfigModel {
		return bgpConnectionConfigModel{
			PeerAsn:        types.Int64Value(65000),
			LocalIPAddress: types.StringValue("169.254.1.1"),
			PeerIPAddress:  types.StringValue("169.254.1.2"),
			AsOverride:     asOverride,
			PermitExportTo: types.ListNull(types.StringType),
			DenyExportTo:   types.ListNull(types.StringType),
		}
	}

	bgpList, diags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(bgpVrouterConnectionConfig), []bgpConnectionConfigModel{
		newBgp(types.BoolValue(true)),
		newBgp(types.BoolValue(false)),
		newBgp(types.BoolNull()),
	})
	require.False(t, diags.HasError(), "building bgp list: %v", diags)

	iface := vxcPartnerConfigInterfaceModel{
		IPAddresses:        types.ListNull(types.StringType),
		IPRoutes:           types.ListNull(types.ObjectType{}.WithAttributeTypes(ipRouteAttrs)),
		NatIPAddresses:     types.ListNull(types.StringType),
		Bfd:                types.ObjectNull(bfdConfigAttrs),
		BgpConnections:     bgpList,
		IpSecTunnelOptions: types.ObjectNull(ipSecTunnelOptionsAttrs),
	}
	ifaceList, diags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(vxcVrouterInterfaceAttrs), []vxcPartnerConfigInterfaceModel{iface})
	require.False(t, diags.HasError(), "building interface list: %v", diags)

	model := vxcPartnerConfigVrouterModel{Interfaces: ifaceList}

	diags, vrouterConfig, _ := createVrouterPartnerConfig(ctx, model, nil, nil)
	require.False(t, diags.HasError(), "createVrouterPartnerConfig: %v", diags)
	require.Len(t, vrouterConfig.Interfaces, 1)
	conns := vrouterConfig.Interfaces[0].BgpConnections
	require.Len(t, conns, 3)

	require.NotNil(t, conns[0].AsOverride)
	assert.True(t, *conns[0].AsOverride)

	require.NotNil(t, conns[1].AsOverride)
	assert.False(t, *conns[1].AsOverride)

	assert.Nil(t, conns[2].AsOverride, "unset as_override must stay nil so the API default applies")
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

// TestVerifyUpdateApplied covers the -1 (requested untagged) / 0 (API
// returned untagged) inner VLAN normalization, a genuine mismatch that must
// still fail, and unrelated fields being unaffected by the normalization.
func TestVerifyUpdateApplied(t *testing.T) {
	cases := []struct {
		name      string
		vxc       *megaport.VXC
		updateReq *megaport.UpdateVXCRequest
		want      bool
	}{
		{
			name: "untagged inner VLAN match: requested -1, API returns 0",
			vxc: &megaport.VXC{
				AEndConfiguration: megaport.VXCEndConfiguration{InnerVLAN: 0},
				BEndConfiguration: megaport.VXCEndConfiguration{InnerVLAN: 0},
			},
			updateReq: &megaport.UpdateVXCRequest{
				AEndInnerVLAN: megaport.PtrTo(-1),
				BEndInnerVLAN: megaport.PtrTo(-1),
			},
			want: true,
		},
		{
			name: "genuine inner VLAN mismatch is not masked",
			vxc: &megaport.VXC{
				AEndConfiguration: megaport.VXCEndConfiguration{InnerVLAN: 200},
			},
			updateReq: &megaport.UpdateVXCRequest{
				AEndInnerVLAN: megaport.PtrTo(100),
			},
			want: false,
		},
		{
			name: "requested -1 but API returns a tagged value is a mismatch",
			vxc: &megaport.VXC{
				AEndConfiguration: megaport.VXCEndConfiguration{InnerVLAN: 100},
			},
			updateReq: &megaport.UpdateVXCRequest{
				AEndInnerVLAN: megaport.PtrTo(-1),
			},
			want: false,
		},
		{
			name: "matching non-zero inner VLAN values still verify",
			vxc: &megaport.VXC{
				AEndConfiguration: megaport.VXCEndConfiguration{InnerVLAN: 100},
				BEndConfiguration: megaport.VXCEndConfiguration{InnerVLAN: 200},
			},
			updateReq: &megaport.UpdateVXCRequest{
				AEndInnerVLAN: megaport.PtrTo(100),
				BEndInnerVLAN: megaport.PtrTo(200),
			},
			want: true,
		},
		{
			name: "unrelated fields (name, rate limit) unaffected by normalization",
			vxc: &megaport.VXC{
				Name:      "updated-name",
				RateLimit: 500,
				AEndConfiguration: megaport.VXCEndConfiguration{
					InnerVLAN: 0,
				},
			},
			updateReq: &megaport.UpdateVXCRequest{
				AEndInnerVLAN: megaport.PtrTo(-1),
				Name:          megaport.PtrTo("updated-name"),
				RateLimit:     megaport.PtrTo(500),
			},
			want: true,
		},
		{
			name: "unrelated field mismatch still fails despite VLAN match",
			vxc: &megaport.VXC{
				Name: "current-name",
				AEndConfiguration: megaport.VXCEndConfiguration{
					InnerVLAN: 0,
				},
			},
			updateReq: &megaport.UpdateVXCRequest{
				AEndInnerVLAN: megaport.PtrTo(-1),
				Name:          megaport.PtrTo("different-name"),
			},
			want: false,
		},
	}

	r := &vxcResource{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, r.verifyUpdateApplied(tc.vxc, tc.updateReq))
		})
	}
}

// vrouterBGPFromObject decodes a partner config object down to its BGP
// connection models so the assertions below can read individual fields.
func vrouterBGPFromObject(t *testing.T, ctx context.Context, obj basetypes.ObjectValue) []bgpConnectionConfigModel {
	t.Helper()
	partner := &vxcPartnerConfigurationModel{}
	require.False(t, obj.As(ctx, partner, basetypes.ObjectAsOptions{}).HasError())
	vrouter := &vxcPartnerConfigVrouterModel{}
	require.False(t, partner.VrouterPartnerConfig.As(ctx, vrouter, basetypes.ObjectAsOptions{}).HasError())
	var ifaces []vxcPartnerConfigInterfaceModel
	require.False(t, vrouter.Interfaces.ElementsAs(ctx, &ifaces, false).HasError())
	require.Len(t, ifaces, 1)
	var bgps []bgpConnectionConfigModel
	require.False(t, ifaces[0].BgpConnections.ElementsAs(ctx, &bgps, false).HasError())
	return bgps
}

// mcrVrouterConn mirrors the shape a real MCR-to-cloud VXC read returns: one
// interface, one BGP session, one prefix filter list attached. resourceName is
// the end label megalith puts on every CSP connection.
func mcrVrouterConn(resourceName string) megaport.CSPConnectionVirtualRouter {
	localAsn := 133937
	asOverride := true
	return megaport.CSPConnectionVirtualRouter{
		ConnectType:  "VROUTER",
		ResourceName: resourceName,
		VLAN:         2125,
		Interfaces: []megaport.CSPConnectionVirtualRouterInterface{{
			IPAddresses: []string{"169.254.145.218/29"},
			IPRoutes: []megaport.IpRoute{{
				Prefix: "10.0.0.0/8", Description: "to-datacenter", NextHop: "169.254.145.217",
			}},
			BGPConnections: []megaport.BgpConnectionConfig{{
				PeerAsn:            16550,
				LocalAsn:           &localAsn,
				LocalIpAddress:     "169.254.145.218",
				PeerIpAddress:      "169.254.145.217",
				PeerType:           "PUB_CLOUD",
				Shutdown:           false,
				Description:        "gcp-ncc-blue",
				MedIn:              100,
				MedOut:             200,
				BfdEnabled:         true,
				ExportPolicy:       "permit",
				AsPathPrependCount: 2,
				AsOverride:         &asOverride,
				ImportWhitelist:    12345,
			}},
		}},
	}
}

// TestBuildVrouterPartnerConfigFromAPI_PopulatesBGP is the regression test for
// the import gap: a VXC read carries the full BGP config, so an imported VXC
// must end up with it in state instead of a null partner config.
func TestBuildVrouterPartnerConfigFromAPI_PopulatesBGP(t *testing.T) {
	ctx := context.Background()

	obj, diags := buildVrouterPartnerConfigFromAPI(ctx, mcrVrouterConn("a_csp_connection"), map[int]string{12345: "allow-in"})
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	require.False(t, obj.IsNull(), "partner config must not be null when the API returned BGP data")

	partner := &vxcPartnerConfigurationModel{}
	require.False(t, obj.As(ctx, partner, basetypes.ObjectAsOptions{}).HasError())
	assert.Equal(t, "vrouter", partner.Partner.ValueString())
	assert.True(t, partner.AWSPartnerConfig.IsNull())
	assert.True(t, partner.AzurePartnerConfig.IsNull())
	assert.True(t, partner.PartnerAEndConfig.IsNull())

	bgps := vrouterBGPFromObject(t, ctx, obj)
	require.Len(t, bgps, 1)
	bgp := bgps[0]
	assert.Equal(t, int64(16550), bgp.PeerAsn.ValueInt64())
	assert.Equal(t, int64(133937), bgp.LocalAsn.ValueInt64())
	assert.Equal(t, "169.254.145.218", bgp.LocalIPAddress.ValueString())
	assert.Equal(t, "169.254.145.217", bgp.PeerIPAddress.ValueString())
	assert.Equal(t, "PUB_CLOUD", bgp.PeerType.ValueString())
	assert.Equal(t, int64(100), bgp.MedIn.ValueInt64())
	assert.Equal(t, int64(200), bgp.MedOut.ValueInt64())
	assert.Equal(t, int64(2), bgp.AsPathPrependCount.ValueInt64())
	assert.Equal(t, "permit", bgp.ExportPolicy.ValueString())
	assert.True(t, bgp.BfdEnabled.ValueBool())
	assert.True(t, bgp.AsOverride.ValueBool(), "as_override is echoed by the read and must survive import")
	assert.Equal(t, "allow-in", bgp.ImportWhitelist.ValueString(), "prefix filter list ID must resolve to its description")
	assert.True(t, bgp.ImportBlacklist.IsNull())
	assert.True(t, bgp.ExportWhitelist.IsNull())
	assert.True(t, bgp.ExportBlacklist.IsNull())

	// The read does echo the password, but writing it would persist a live MD5
	// key in plain text in state. The export lists are not echoed at all.
	assert.True(t, bgp.Password.IsNull(), "the password must be left out of state")
	assert.True(t, bgp.PermitExportTo.IsNull())
	assert.True(t, bgp.DenyExportTo.IsNull())
}

// TestBuildVrouterPartnerConfigFromAPI_OmittedScalarsStayNull covers the common
// real session: the API omits every optional attribute it has no value for.
// Writing the decoded zero values would make the plan differ from a
// configuration that omits them too, on every apply.
func TestBuildVrouterPartnerConfigFromAPI_OmittedScalarsStayNull(t *testing.T) {
	ctx := context.Background()

	conn := megaport.CSPConnectionVirtualRouter{
		ConnectType:  "VROUTER",
		ResourceName: "a_csp_connection",
		Interfaces: []megaport.CSPConnectionVirtualRouterInterface{{
			IPAddresses: []string{"169.254.1.2/30"},
			IPRoutes:    []megaport.IpRoute{{Prefix: "10.0.0.0/8", NextHop: "169.254.1.1"}},
			BGPConnections: []megaport.BgpConnectionConfig{{
				PeerAsn:        64512,
				LocalIpAddress: "169.254.1.2",
				PeerIpAddress:  "169.254.1.1",
			}},
		}},
	}

	obj, diags := buildVrouterPartnerConfigFromAPI(ctx, conn, nil)
	require.False(t, diags.HasError())
	require.False(t, obj.IsNull())

	bgps := vrouterBGPFromObject(t, ctx, obj)
	require.Len(t, bgps, 1)
	bgp := bgps[0]
	assert.Equal(t, int64(64512), bgp.PeerAsn.ValueInt64())
	assert.True(t, bgp.PeerType.IsNull(), `an empty peer_type would fail its own OneOf validator`)
	assert.True(t, bgp.Description.IsNull())
	assert.True(t, bgp.MedIn.IsNull())
	assert.True(t, bgp.MedOut.IsNull())
	assert.True(t, bgp.ExportPolicy.IsNull())
	assert.True(t, bgp.AsPathPrependCount.IsNull())
	assert.True(t, bgp.Shutdown.IsNull())
	assert.True(t, bgp.BfdEnabled.IsNull())

	partner := &vxcPartnerConfigurationModel{}
	require.False(t, obj.As(ctx, partner, basetypes.ObjectAsOptions{}).HasError())
	vrouter := &vxcPartnerConfigVrouterModel{}
	require.False(t, partner.VrouterPartnerConfig.As(ctx, vrouter, basetypes.ObjectAsOptions{}).HasError())
	var ifaces []vxcPartnerConfigInterfaceModel
	require.False(t, vrouter.Interfaces.ElementsAs(ctx, &ifaces, false).HasError())
	var routes []ipRouteModel
	require.False(t, ifaces[0].IPRoutes.ElementsAs(ctx, &routes, false).HasError())
	require.Len(t, routes, 1)
	assert.True(t, routes[0].Description.IsNull(), "an IP route description the API omitted must stay absent")
}

// TestBuildVrouterPartnerConfigFromAPI_PasswordWarning checks the user is told
// about an MD5 password Terraform will not carry into state, because the first
// apply after import would otherwise look complete when it is not.
func TestBuildVrouterPartnerConfigFromAPI_PasswordWarning(t *testing.T) {
	ctx := context.Background()

	conn := mcrVrouterConn("a_csp_connection")
	conn.Interfaces[0].BGPConnections[0].Password = "liveMD5Key"

	obj, diags := buildVrouterPartnerConfigFromAPI(ctx, conn, map[int]string{12345: "allow-in"})
	require.False(t, diags.HasError())
	require.False(t, obj.IsNull())
	require.Equal(t, 1, diags.WarningsCount())
	assert.Contains(t, diags.Warnings()[0].Detail(), "169.254.145.217")
	assert.NotContains(t, diags.Warnings()[0].Detail(), "liveMD5Key", "the warning must not echo the password")

	bgps := vrouterBGPFromObject(t, ctx, obj)
	require.Len(t, bgps, 1)
	assert.True(t, bgps[0].Password.IsNull())
}

// TestBuildVrouterPartnerConfigFromAPI_InterfaceFields covers the interface
// level: what the read supplies, and what does not come back.
func TestBuildVrouterPartnerConfigFromAPI_InterfaceFields(t *testing.T) {
	ctx := context.Background()

	obj, diags := buildVrouterPartnerConfigFromAPI(ctx, mcrVrouterConn("a_csp_connection"), map[int]string{12345: "allow-in"})
	require.False(t, diags.HasError())

	partner := &vxcPartnerConfigurationModel{}
	require.False(t, obj.As(ctx, partner, basetypes.ObjectAsOptions{}).HasError())
	vrouter := &vxcPartnerConfigVrouterModel{}
	require.False(t, partner.VrouterPartnerConfig.As(ctx, vrouter, basetypes.ObjectAsOptions{}).HasError())
	var ifaces []vxcPartnerConfigInterfaceModel
	require.False(t, vrouter.Interfaces.ElementsAs(ctx, &ifaces, false).HasError())
	require.Len(t, ifaces, 1)

	var ips []string
	require.False(t, ifaces[0].IPAddresses.ElementsAs(ctx, &ips, false).HasError())
	assert.Equal(t, []string{"169.254.145.218/29"}, ips)

	var routes []ipRouteModel
	require.False(t, ifaces[0].IPRoutes.ElementsAs(ctx, &routes, false).HasError())
	require.Len(t, routes, 1)
	assert.Equal(t, "10.0.0.0/8", routes[0].Prefix.ValueString())
	assert.Equal(t, "to-datacenter", routes[0].Description.ValueString())
	assert.Equal(t, "169.254.145.217", routes[0].NextHop.ValueString())

	// megalith does not re-serialize the interface bfd block, and megaportgo
	// models none of ip_mtu, vlan, description, interface_type or the packet
	// filters, so no import can recover them.
	assert.True(t, ifaces[0].Bfd.IsNull(), "bfd is not returned by the read")
	assert.True(t, ifaces[0].IpMtu.IsNull())
	assert.True(t, ifaces[0].VLAN.IsNull())
	assert.True(t, ifaces[0].Description.IsNull())
	assert.True(t, ifaces[0].InterfaceType.IsNull())
	assert.True(t, ifaces[0].PacketFilterIn.IsNull())
	assert.True(t, ifaces[0].PacketFilterOut.IsNull())
	assert.True(t, ifaces[0].IpSecTunnelOptions.IsNull(), "the PSK is write-only, so no tunnel options come back")
	assert.True(t, ifaces[0].NatIPAddresses.IsNull())
}

// TestBuildVrouterPartnerConfigFromAPI_UnresolvedPrefixFilter checks the unsafe
// case is refused: writing a null filter over a live one would make the next
// plan propose removing a route filter.
func TestBuildVrouterPartnerConfigFromAPI_UnresolvedPrefixFilter(t *testing.T) {
	ctx := context.Background()

	obj, diags := buildVrouterPartnerConfigFromAPI(ctx, mcrVrouterConn("a_csp_connection"), map[int]string{999: "some-other-list"})
	assert.False(t, diags.HasError())
	assert.True(t, obj.IsNull(), "an unresolvable prefix filter list must abandon the whole config")
	require.Equal(t, 1, diags.WarningsCount())
	assert.Contains(t, diags.Warnings()[0].Detail(), "12345")
}

// TestBuildVrouterPartnerConfigFromAPI_NoInterfaces covers a vrouter connection
// the API returned without interface data.
func TestBuildVrouterPartnerConfigFromAPI_NoInterfaces(t *testing.T) {
	ctx := context.Background()

	obj, diags := buildVrouterPartnerConfigFromAPI(ctx, megaport.CSPConnectionVirtualRouter{ConnectType: "VROUTER"}, nil)
	assert.False(t, diags.HasError())
	assert.True(t, obj.IsNull())
}

// TestPrefixFilterIDToName covers the three ID cases.
func TestPrefixFilterIDToName(t *testing.T) {
	name, ok := prefixFilterIDToName(0, nil)
	assert.True(t, ok, "an unset ID is not a failure")
	assert.True(t, name.IsNull())

	name, ok = prefixFilterIDToName(7, map[int]string{7: "deny-rfc1918"})
	assert.True(t, ok)
	assert.Equal(t, "deny-rfc1918", name.ValueString())

	_, ok = prefixFilterIDToName(7, map[int]string{8: "other"})
	assert.False(t, ok, "a set ID with no matching list must be reported as unresolvable")
}

// importVXC builds the VXC read an MCR-to-MCR import sees, with one CSP
// connection per end.
func importVXC(conns ...megaport.CSPConnectionVirtualRouter) *megaport.VXC {
	configs := make([]megaport.CSPConnectionConfig, 0, len(conns))
	for _, c := range conns {
		configs = append(configs, c)
	}
	return &megaport.VXC{
		AEndConfiguration: megaport.VXCEndConfiguration{UID: "a-end-uid"},
		BEndConfiguration: megaport.VXCEndConfiguration{UID: "b-end-uid"},
		Resources: &megaport.VXCResources{
			CSPConnection: &megaport.CSPConnection{CSPConnection: configs},
		},
	}
}

// TestFillVrouterPartnerConfigsOnImport_MatchesEndsByResourceName is the test
// that matters most: attaching one end's BGP session to the other end would
// rewrite a live peering on the next apply.
func TestFillVrouterPartnerConfigsOnImport_MatchesEndsByResourceName(t *testing.T) {
	ctx := context.Background()
	products := &MockProductService{
		GetProductTypeFunc: func(_ context.Context, uid string) (string, error) {
			return megaport.PRODUCT_MCR, nil
		},
	}
	mcrs := &MockMCRService{
		ListMCRPrefixFilterListsResult: []*megaport.PrefixFilterList{{Id: 12345, Description: "allow-in"}},
	}
	r := &vxcResource{client: &megaport.Client{ProductService: products, MCRService: mcrs}}

	// The b end is listed first and carries a different peer, so a fix that
	// matched on order or on the VLAN would attach it to the a end.
	bConn := mcrVrouterConn("b_csp_connection")
	bConn.Interfaces[0].BGPConnections[0].PeerIpAddress = "169.254.200.1"
	bConn.Interfaces[0].BGPConnections[0].ImportWhitelist = 0
	aConn := mcrVrouterConn("a_csp_connection")

	state := &vxcResourceModel{
		AEndPartnerConfig: types.ObjectNull(vxcPartnerConfigAttrs),
		BEndPartnerConfig: types.ObjectNull(vxcPartnerConfigAttrs),
	}
	diags := r.fillVrouterPartnerConfigsOnImport(ctx, state, importVXC(bConn, aConn))
	require.False(t, diags.HasError())
	assert.Equal(t, 0, diags.WarningsCount(), "unexpected warnings: %v", diags.Warnings())

	require.False(t, state.AEndPartnerConfig.IsNull())
	require.False(t, state.BEndPartnerConfig.IsNull())
	aBgps := vrouterBGPFromObject(t, ctx, state.AEndPartnerConfig)
	bBgps := vrouterBGPFromObject(t, ctx, state.BEndPartnerConfig)
	require.Len(t, aBgps, 1)
	require.Len(t, bBgps, 1)
	assert.Equal(t, "169.254.145.217", aBgps[0].PeerIPAddress.ValueString())
	assert.Equal(t, "169.254.200.1", bBgps[0].PeerIPAddress.ValueString())
	assert.Equal(t, "allow-in", aBgps[0].ImportWhitelist.ValueString())
	assert.Equal(t, "a-end-uid", products.CapturedGetProductTypeUIDs[0], "the prefix filter lookup must use the end that owns the list")
}

// TestFillVrouterPartnerConfigsOnImport_AmbiguousEnd covers two connections
// labelled for the same end: guessing could attach the wrong BGP session.
func TestFillVrouterPartnerConfigsOnImport_AmbiguousEnd(t *testing.T) {
	ctx := context.Background()
	r := &vxcResource{}

	state := &vxcResourceModel{
		AEndPartnerConfig: types.ObjectNull(vxcPartnerConfigAttrs),
		BEndPartnerConfig: types.ObjectNull(vxcPartnerConfigAttrs),
	}
	vxc := importVXC(mcrVrouterConn("a_csp_connection"), mcrVrouterConn("a_csp_connection"))

	diags := r.fillVrouterPartnerConfigsOnImport(ctx, state, vxc)
	assert.False(t, diags.HasError())
	require.Equal(t, 1, diags.WarningsCount())
	assert.Contains(t, diags.Warnings()[0].Detail(), "a_end_partner_config")
	assert.True(t, state.AEndPartnerConfig.IsNull(), "an ambiguous end must be left for the user to fill in")
	assert.True(t, state.BEndPartnerConfig.IsNull())
}

// TestFillVrouterPartnerConfigsOnImport_NoVrouterConnections covers a VXC with
// no vrouter end, such as port to port, where there is nothing to rebuild. A nil
// client would panic on any API call, so reaching the end proves none is made.
func TestFillVrouterPartnerConfigsOnImport_NoVrouterConnections(t *testing.T) {
	ctx := context.Background()
	r := &vxcResource{}

	state := &vxcResourceModel{
		AEndPartnerConfig: types.ObjectNull(vxcPartnerConfigAttrs),
		BEndPartnerConfig: types.ObjectNull(vxcPartnerConfigAttrs),
	}

	diags := r.fillVrouterPartnerConfigsOnImport(ctx, state, &megaport.VXC{})
	assert.False(t, diags.HasError())
	assert.True(t, state.AEndPartnerConfig.IsNull())
	assert.True(t, state.BEndPartnerConfig.IsNull())
}

// MockProductService stubs the Products API. Only GetProductType is exercised;
// the rest satisfy the interface.
type MockProductService struct {
	GetProductTypeFunc         func(ctx context.Context, productUID string) (string, error)
	CapturedGetProductTypeUIDs []string
}

func (m *MockProductService) GetProductType(ctx context.Context, productUID string) (string, error) {
	m.CapturedGetProductTypeUIDs = append(m.CapturedGetProductTypeUIDs, productUID)
	if m.GetProductTypeFunc != nil {
		return m.GetProductTypeFunc(ctx, productUID)
	}
	return "", nil
}

func (m *MockProductService) ExecuteOrder(ctx context.Context, requestBody interface{}) (*[]byte, error) {
	return nil, nil
}

func (m *MockProductService) ListProducts(ctx context.Context) ([]megaport.Product, error) {
	return nil, nil
}

func (m *MockProductService) ModifyProduct(ctx context.Context, req *megaport.ModifyProductRequest) (*megaport.ModifyProductResponse, error) {
	return nil, nil
}

func (m *MockProductService) DeleteProduct(ctx context.Context, req *megaport.DeleteProductRequest) (*megaport.DeleteProductResponse, error) {
	return nil, nil
}

func (m *MockProductService) RestoreProduct(ctx context.Context, productId string) (*megaport.RestoreProductResponse, error) {
	return nil, nil
}

func (m *MockProductService) ManageProductLock(ctx context.Context, req *megaport.ManageProductLockRequest) (*megaport.ManageProductLockResponse, error) {
	return nil, nil
}

func (m *MockProductService) ValidateProductOrder(ctx context.Context, requestBody interface{}) error {
	return nil
}

func (m *MockProductService) ListProductResourceTags(ctx context.Context, productID string) ([]megaport.ResourceTag, error) {
	return nil, nil
}

func (m *MockProductService) UpdateProductResourceTags(ctx context.Context, productUID string, tagsReq *megaport.UpdateProductResourceTagsRequest) error {
	return nil
}

func (m *MockProductService) GetProductPricing(ctx context.Context, req megaport.PriceBookRequest) (*megaport.PriceBookDTO, error) {
	return nil, nil
}

func (m *MockProductService) GetProductPricingForCompany(ctx context.Context, req *megaport.GetProductPricingRequest) (*megaport.PriceBookDTO, error) {
	return nil, nil
}

// TestFillVrouterPartnerConfigsOnImport_UnlabeledConnection covers a read that
// carries router data with no end name. Silence would look identical to a VXC
// with no router configuration at all.
func TestFillVrouterPartnerConfigsOnImport_UnlabeledConnection(t *testing.T) {
	ctx := context.Background()
	r := &vxcResource{}

	state := &vxcResourceModel{
		AEndPartnerConfig: types.ObjectNull(vxcPartnerConfigAttrs),
		BEndPartnerConfig: types.ObjectNull(vxcPartnerConfigAttrs),
	}

	diags := r.fillVrouterPartnerConfigsOnImport(ctx, state, importVXC(mcrVrouterConn("")))
	assert.False(t, diags.HasError())
	require.Equal(t, 1, diags.WarningsCount())
	assert.Contains(t, diags.Warnings()[0].Detail(), "no end name")
	assert.True(t, state.AEndPartnerConfig.IsNull())
	assert.True(t, state.BEndPartnerConfig.IsNull())
}
