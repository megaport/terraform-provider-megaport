package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	megaport "github.com/megaport/megaportgo"
)

// mcrIPsecReadRequest builds a ReadRequest with mcr_id set and every computed
// attribute null, and a ReadResponse with an empty state.
func mcrIPsecReadRequest(t *testing.T, ds *mcrIPsecDataSource, mcrID string) (datasource.ReadRequest, *datasource.ReadResponse) {
	t.Helper()
	ctx := context.Background()

	schemaResp := datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)

	attrValues := make(map[string]tftypes.Value, len(schemaResp.Schema.Attributes))
	for name, attr := range schemaResp.Schema.Attributes {
		attrValues[name] = tftypes.NewValue(attr.GetType().TerraformType(ctx), nil)
	}
	attrValues["mcr_id"] = tftypes.NewValue(tftypes.String, mcrID)
	configRaw := tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), attrValues)

	req := datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw},
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	return req, resp
}

func TestReadMCRIPsec_WithTunnels(t *testing.T) {
	ctx := context.Background()
	var calledWith string
	mockMCRService := &MockMCRService{
		GetMCRIPsecFunc: func(_ context.Context, mcrID string) (*megaport.MCRIPsecConfiguration, error) {
			calledWith = mcrID
			return &megaport.MCRIPsecConfiguration{
				TotalTunnelCount:    3,
				MaxTunnelCountLimit: 30,
				IPsecConfiguredVXCs: []megaport.IPsecConfiguredVXC{
					{
						Name:       "VXC One",
						ProductUID: "vxc-1",
						Tunnels: []megaport.IPsecTunnel{
							{
								Description:          "primary",
								SourceIpAddress:      "169.254.100.1",
								DestinationIpAddress: "203.0.113.10",
								LocalId:              "local.example",
								RemoteId:             "remote.example",
								VLAN:                 megaport.PtrTo(100),
							},
							{
								SourceIpAddress:      "169.254.100.5",
								DestinationIpAddress: "203.0.113.11",
							},
						},
					},
					{
						Name:       "VXC Two",
						ProductUID: "vxc-2",
						Tunnels: []megaport.IPsecTunnel{
							{
								SourceIpAddress:      "169.254.200.1",
								DestinationIpAddress: "198.51.100.20",
								VLAN:                 megaport.PtrTo(200),
							},
						},
					},
				},
			}, nil
		},
	}
	ds := &mcrIPsecDataSource{client: &megaport.Client{MCRService: mockMCRService}}

	req, resp := mcrIPsecReadRequest(t, ds, "mcr-1")
	ds.Read(ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())
	assert.Equal(t, "mcr-1", calledWith)

	var state mcrIPsecDataSourceModel
	require.False(t, resp.State.Get(ctx, &state).HasError())
	assert.Equal(t, "mcr-1", state.MCRID.ValueString())
	assert.Equal(t, int64(3), state.TotalTunnelCount.ValueInt64())
	assert.Equal(t, int64(30), state.MaxTunnelCountLimit.ValueInt64())

	var vxcs []mcrIPsecVXCModel
	require.False(t, state.VXCs.ElementsAs(ctx, &vxcs, false).HasError())
	require.Len(t, vxcs, 2)
	assert.Equal(t, "VXC One", vxcs[0].Name.ValueString())
	assert.Equal(t, "vxc-1", vxcs[0].ProductUID.ValueString())
	assert.Equal(t, "VXC Two", vxcs[1].Name.ValueString())
	assert.Equal(t, "vxc-2", vxcs[1].ProductUID.ValueString())

	var tunnels []mcrIPsecTunnelModel
	require.False(t, vxcs[0].Tunnels.ElementsAs(ctx, &tunnels, false).HasError())
	require.Len(t, tunnels, 2)
	assert.Equal(t, "primary", tunnels[0].Description.ValueString())
	assert.Equal(t, "169.254.100.1", tunnels[0].SourceIPAddress.ValueString())
	assert.Equal(t, "203.0.113.10", tunnels[0].DestinationIPAddress.ValueString())
	assert.Equal(t, "local.example", tunnels[0].LocalID.ValueString())
	assert.Equal(t, "remote.example", tunnels[0].RemoteID.ValueString())
	assert.Equal(t, int64(100), tunnels[0].VLAN.ValueInt64())
	assert.Equal(t, "169.254.100.5", tunnels[1].SourceIPAddress.ValueString())
	assert.True(t, tunnels[1].VLAN.IsNull(), "an untagged source interface should leave vlan null")

	require.False(t, vxcs[1].Tunnels.ElementsAs(ctx, &tunnels, false).HasError())
	require.Len(t, tunnels, 1)
	assert.Equal(t, "198.51.100.20", tunnels[0].DestinationIPAddress.ValueString())
	assert.Equal(t, int64(200), tunnels[0].VLAN.ValueInt64())
}

func TestReadMCRIPsec_NoTunnels(t *testing.T) {
	ctx := context.Background()
	mockMCRService := &MockMCRService{
		GetMCRIPsecFunc: func(_ context.Context, _ string) (*megaport.MCRIPsecConfiguration, error) {
			return &megaport.MCRIPsecConfiguration{MaxTunnelCountLimit: 30}, nil
		},
	}
	ds := &mcrIPsecDataSource{client: &megaport.Client{MCRService: mockMCRService}}

	req, resp := mcrIPsecReadRequest(t, ds, "mcr-1")
	ds.Read(ctx, req, resp)

	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics.Errors())

	var state mcrIPsecDataSourceModel
	require.False(t, resp.State.Get(ctx, &state).HasError())
	assert.Equal(t, int64(0), state.TotalTunnelCount.ValueInt64())
	assert.Equal(t, int64(30), state.MaxTunnelCountLimit.ValueInt64())
	assert.False(t, state.VXCs.IsNull(), "vxcs should be an empty list, not null")
	assert.Empty(t, state.VXCs.Elements())
}

func TestReadMCRIPsec_APIError(t *testing.T) {
	ctx := context.Background()
	mockMCRService := &MockMCRService{
		GetMCRIPsecFunc: func(_ context.Context, _ string) (*megaport.MCRIPsecConfiguration, error) {
			return nil, errors.New("API error")
		},
	}
	ds := &mcrIPsecDataSource{client: &megaport.Client{MCRService: mockMCRService}}

	req, resp := mcrIPsecReadRequest(t, ds, "mcr-1")
	ds.Read(ctx, req, resp)

	require.Len(t, resp.Diagnostics.Errors(), 1)
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "mcr-1")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "API error")
	assert.True(t, resp.State.Raw.IsNull(), "a failed read should not write state")
}
