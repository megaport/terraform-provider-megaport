package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateVrouterPartnerConfig_IPsecTunnelOptions verifies the model -> SDK
// mapping for ip_sec_tunnel_options, including the multi-tunnel case and the
// "nil keeps the API default" behavior for the optional pointer fields.
func TestCreateVrouterPartnerConfig_IPsecTunnelOptions(t *testing.T) {
	ctx := context.Background()

	tunnels := []ipSecTunnelOptionsModel{
		{
			// Fully populated tunnel.
			SourceIPAddress:      types.StringValue("169.254.1.1"),
			DestinationIPAddress: types.StringValue("203.0.113.1"),
			PreSharedKey:         types.StringValue("secret-one"),
			Passive:              types.BoolValue(false),
			LocalID:              types.StringValue("local-1"),
			RemoteID:             types.StringValue("remote-1"),
			Phase1Lifetime:       types.Int64Value(7200),
			Phase2Lifetime:       types.Int64Value(3600),
		},
		{
			// Minimal tunnel: only the required fields; optionals left null so
			// the API applies its defaults.
			SourceIPAddress:      types.StringValue("169.254.2.1"),
			DestinationIPAddress: types.StringValue("203.0.113.2"),
			PreSharedKey:         types.StringValue("secret-two"),
			Passive:              types.BoolNull(),
			LocalID:              types.StringNull(),
			RemoteID:             types.StringNull(),
			Phase1Lifetime:       types.Int64Null(),
			Phase2Lifetime:       types.Int64Null(),
		},
	}

	tunnelList, diags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(ipSecTunnelOptionsAttrs), tunnels)
	require.False(t, diags.HasError(), "building tunnel list: %v", diags)

	iface := vxcPartnerConfigInterfaceModel{
		InterfaceType:      types.StringValue("ipSecTunnel"),
		IPAddresses:        types.ListNull(types.StringType),
		IPRoutes:           types.ListNull(types.ObjectType{}.WithAttributeTypes(ipRouteAttrs)),
		NatIPAddresses:     types.ListNull(types.StringType),
		Bfd:                types.ObjectNull(bfdConfigAttrs),
		BgpConnections:     types.ListNull(types.ObjectType{}.WithAttributeTypes(bgpVrouterConnectionConfig)),
		IpSecTunnelOptions: tunnelList,
	}
	ifaceList, diags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(vxcVrouterInterfaceAttrs), []vxcPartnerConfigInterfaceModel{iface})
	require.False(t, diags.HasError(), "building interface list: %v", diags)

	model := vxcPartnerConfigVrouterModel{Interfaces: ifaceList}

	diags, vrouterConfig, _ := createVrouterPartnerConfig(ctx, model, nil)
	require.False(t, diags.HasError(), "createVrouterPartnerConfig: %v", diags)
	require.Len(t, vrouterConfig.Interfaces, 1)

	got := vrouterConfig.Interfaces[0].IpSecTunnelOptions
	require.Len(t, got, 2, "expected both tunnels mapped")

	// Fully populated tunnel.
	full := got[0]
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
	minimal := got[1]
	assert.Equal(t, "169.254.2.1", minimal.SourceIpAddress)
	assert.Equal(t, "203.0.113.2", minimal.DestinationIpAddress)
	assert.Equal(t, "secret-two", minimal.PreSharedKey)
	assert.Empty(t, minimal.LocalId)
	assert.Empty(t, minimal.RemoteId)
	assert.Nil(t, minimal.Passive)
	assert.Nil(t, minimal.Phase1Lifetime)
	assert.Nil(t, minimal.Phase2Lifetime)
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
