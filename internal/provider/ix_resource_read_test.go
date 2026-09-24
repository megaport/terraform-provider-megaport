package provider

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	megaport "github.com/megaport/megaportgo"
)

// MockIXService is a mock of the IX service for testing
type MockIXService struct {
	BuyIXResult *megaport.BuyIXResponse
	GetIXResult *megaport.IX
	GetIXErr    error
}

func (m *MockIXService) GetIX(ctx context.Context, id string) (*megaport.IX, error) {
	return m.GetIXResult, m.GetIXErr
}

// Implement other required methods of the IXService interface with minimal stubs
func (m *MockIXService) BuyIX(ctx context.Context, req *megaport.BuyIXRequest) (*megaport.BuyIXResponse, error) {
	return m.BuyIXResult, nil
}

func (m *MockIXService) ValidateIXOrder(ctx context.Context, req *megaport.BuyIXRequest) error {
	return nil
}

func (m *MockIXService) UpdateIX(ctx context.Context, id string, req *megaport.UpdateIXRequest) (*megaport.IX, error) {
	return nil, nil
}

func (m *MockIXService) DeleteIX(ctx context.Context, id string, req *megaport.DeleteIXRequest) error {
	return nil
}

func (m *MockIXService) ListIXs(ctx context.Context, req *megaport.ListIXsRequest) ([]*megaport.IX, error) {
	return nil, nil
}

func (m *MockIXService) ListIXPs(ctx context.Context, req *megaport.ListIXPsRequest) ([]*megaport.IXP, error) {
	return nil, nil
}

const ixReadTestUID = "ix-uid-123"

// ErrorResponse.Error() dereferences Response.Request, so both have to be set.
func ixAPIError(statusCode int, message string) *megaport.ErrorResponse {
	return &megaport.ErrorResponse{
		Response: &http.Response{
			StatusCode: statusCode,
			Request: &http.Request{
				Method: http.MethodGet,
				URL:    &url.URL{Scheme: "https", Host: "api.megaport.com", Path: "/v2/product/" + ixReadTestUID},
			},
		},
		Message: message,
	}
}

// runIXRead drives ixResource.Read against a stubbed GetIX, and returns the
// response alongside the state it started from so callers can assert the state
// was either cleared or left exactly as it was.
func runIXRead(t *testing.T, ix *megaport.IX, getIXErr error) (*fwresource.ReadResponse, tftypes.Value) {
	t.Helper()
	ctx := context.Background()

	r := &ixResource{client: &megaport.Client{IXService: &MockIXService{GetIXResult: ix, GetIXErr: getIXErr}}}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

	schemaObjType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	require.True(t, ok, "schema type is not tftypes.Object")

	stateAttrs := nullValueMap(schemaObjType)
	stateAttrs["product_uid"] = tftypes.NewValue(tftypes.String, ixReadTestUID)
	stateAttrs["product_name"] = tftypes.NewValue(tftypes.String, "test-ix")
	stateVal := tftypes.NewValue(schemaObjType, stateAttrs)

	state := tfsdk.State{Schema: s, Raw: stateVal}
	resp := &fwresource.ReadResponse{State: state}
	r.Read(ctx, fwresource.ReadRequest{State: state}, resp)

	return resp, stateVal
}

// TestIXReadClearsStateOnNotFound covers the IX being deleted outside
// Terraform, in both forms the API reports it.
func TestIXReadClearsStateOnNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{"404", ixAPIError(http.StatusNotFound, "Not Found")},
		{"400_could_not_find_service", ixAPIError(http.StatusBadRequest, "Could not find a service with UID "+ixReadTestUID)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, _ := runIXRead(t, nil, tc.err)

			assert.False(t, resp.Diagnostics.HasError(), "expected no diagnostic, got: %v", resp.Diagnostics.Errors())
			assert.True(t, resp.State.Raw.IsNull(), "expected the IX to be removed from state")
		})
	}
}

// TestIXReadKeepsStateOnOtherErrors covers the reason this branch exists: a
// transient failure must not drop a live, billing IX out of state.
func TestIXReadKeepsStateOnOtherErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{"500", ixAPIError(http.StatusInternalServerError, "Internal Server Error")},
		{"401_expired_token", ixAPIError(http.StatusUnauthorized, "Token has expired")},
		{"400_unrelated_message", ixAPIError(http.StatusBadRequest, "Invalid request")},
		{"transport_failure", errors.New("dial tcp: lookup api.megaport.com: no such host")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, stateVal := runIXRead(t, nil, tc.err)

			require.True(t, resp.Diagnostics.HasError(), "expected a diagnostic")
			require.Len(t, resp.Diagnostics.Errors(), 1)
			assert.Equal(t, "Error Reading IX", resp.Diagnostics.Errors()[0].Summary())
			assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), ixReadTestUID)
			assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), tc.err.Error())
			assert.True(t, resp.State.Raw.Equal(stateVal), "expected state to be left untouched")
		})
	}
}

// TestIXReadClearsStateOnDecommissioned covers the other way the API reports a
// gone IX: a 200 carrying a terminal provisioning status. CANCELLED is not
// terminal, the IX stays live until the end of its term, so clearing state on
// it would propose a duplicate against a service that is still billing.
func TestIXReadClearsStateOnDecommissioned(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     string
		wantAbsent bool
	}{
		{"decommissioned", megaport.STATUS_DECOMMISSIONED, true},
		{"cancelled", megaport.STATUS_CANCELLED, false},
		{"live", megaport.SERVICE_LIVE, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ix := &megaport.IX{ProductUID: ixReadTestUID, ProductName: "test-ix", ProvisioningStatus: tc.status}
			resp, _ := runIXRead(t, ix, nil)

			assert.False(t, resp.Diagnostics.HasError(), "expected no diagnostic, got: %v", resp.Diagnostics.Errors())
			assert.Equal(t, tc.wantAbsent, resp.State.Raw.IsNull(), "unexpected state presence for status %q", tc.status)
		})
	}
}

// readIXResources runs Read and returns the resources object from the new state.
func readIXResources(t *testing.T, ix *megaport.IX) map[string]attr.Value {
	t.Helper()
	resp, _ := runIXRead(t, ix, nil)
	require.False(t, resp.Diagnostics.HasError(), "expected no diagnostic, got: %v", resp.Diagnostics.Errors())

	var got ixResourceModel
	require.False(t, resp.State.Get(context.Background(), &got).HasError())
	require.False(t, got.Resources.IsNull(), "expected resources to be set")
	require.Len(t, got.Resources.Attributes(), len(resourcesAttrTypes))
	return got.Resources.Attributes()
}

func TestIXReadMapsPopulatedResources(t *testing.T) {
	t.Parallel()
	ix := &megaport.IX{
		ProductUID:         ixReadTestUID,
		ProductName:        "test-ix",
		ProvisioningStatus: megaport.SERVICE_LIVE,
		Resources: megaport.IXResources{
			Interface:      megaport.IXInterface{ResourceName: "interface", ResourceType: "interface", PortSpeed: 10000},
			BGPConnections: []megaport.IXBGPConnection{{ASN: 65000, CustomerIPAddress: "192.0.2.2/24", ResourceType: "bgp_connection"}},
			IPAddresses:    []megaport.IXIPAddress{{Address: "192.0.2.2/24", ResourceType: "ip_address", Version: 4}},
			VPLSInterface:  megaport.IXVPLSInterface{ResourceName: "vpls_interface", ResourceType: "vpls_interface", VLAN: 100},
		},
	}

	for name, v := range readIXResources(t, ix) {
		assert.False(t, v.IsNull(), "expected %s to be set", name)
	}
}

func TestIXReadLeavesEmptyResourcesNull(t *testing.T) {
	t.Parallel()
	ix := &megaport.IX{ProductUID: ixReadTestUID, ProductName: "test-ix", ProvisioningStatus: megaport.SERVICE_LIVE}

	for name, v := range readIXResources(t, ix) {
		assert.True(t, v.IsNull(), "expected %s to be null", name)
	}
}

// Not parallel: it breaks a package-level attr type map to force a conversion error.
func TestIXConversionErrorDoesNotSaveState(t *testing.T) {
	saved := interfaceAttrTypes
	t.Cleanup(func() { interfaceAttrTypes = saved })
	interfaceAttrTypes = map[string]attr.Type{"demarcation": types.StringType}

	ctx := context.Background()
	ix := &megaport.IX{
		ProductUID:         ixReadTestUID,
		ProductName:        "test-ix",
		ProvisioningStatus: megaport.SERVICE_LIVE,
		Resources:          megaport.IXResources{Interface: megaport.IXInterface{ResourceType: "interface"}},
	}

	t.Run("read", func(t *testing.T) {
		resp, stateVal := runIXRead(t, ix, nil)
		require.True(t, resp.Diagnostics.HasError(), "expected a conversion diagnostic")
		assert.True(t, resp.State.Raw.Equal(stateVal), "expected state to be left untouched")
	})

	r := &ixResource{client: &megaport.Client{IXService: &MockIXService{
		BuyIXResult: &megaport.BuyIXResponse{TechnicalServiceUID: ixReadTestUID},
		GetIXResult: ix,
	}}}
	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	schemaObjType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	require.True(t, ok, "schema type is not tftypes.Object")

	t.Run("create", func(t *testing.T) {
		planAttrs := nullValueMap(schemaObjType)
		planAttrs["product_name"] = tftypes.NewValue(tftypes.String, "test-ix")
		resp := &fwresource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(schemaObjType, nil)}}
		r.Create(ctx, fwresource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(schemaObjType, planAttrs)}}, resp)

		require.True(t, resp.Diagnostics.HasError(), "expected a conversion diagnostic")
		var got ixResourceModel
		require.False(t, resp.State.Get(ctx, &got).HasError())
		assert.Equal(t, ixReadTestUID, got.ProductUID.ValueString(), "expected the UID to stay in state")
		assert.True(t, got.Resources.IsNull(), "expected resources not to be saved")
	})

	t.Run("update", func(t *testing.T) {
		stateAttrs := nullValueMap(schemaObjType)
		stateAttrs["product_uid"] = tftypes.NewValue(tftypes.String, ixReadTestUID)
		stateVal := tftypes.NewValue(schemaObjType, stateAttrs)

		resp := &fwresource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: stateVal}}
		r.Update(ctx, fwresource.UpdateRequest{
			Plan:  tfsdk.Plan{Schema: s, Raw: stateVal},
			State: tfsdk.State{Schema: s, Raw: stateVal},
		}, resp)

		require.True(t, resp.Diagnostics.HasError(), "expected a conversion diagnostic")
		assert.True(t, resp.State.Raw.Equal(stateVal), "expected state to be left untouched")
	})
}
