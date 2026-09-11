package provider

import (
	"context"
	"math/big"
	"net/http"
	"net/url"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	megaport "github.com/megaport/megaportgo"
)

// mcrPrefixFilterListReadRequest builds a Read request and response for the
// prefix filter list resource, both seeded with the same populated state. The
// seeded response is what makes the cleared-state assertion meaningful.
func mcrPrefixFilterListReadRequest(t *testing.T, r *mcrPrefixFilterListResource) (resource.ReadRequest, *resource.ReadResponse, tftypes.Value) {
	t.Helper()
	ctx := context.Background()

	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	require.True(t, ok)
	attrValues := nullValueMap(objType)
	attrValues["mcr_id"] = tftypes.NewValue(tftypes.String, "mcr-1")
	attrValues["id"] = tftypes.NewValue(tftypes.Number, big.NewFloat(7))
	startingState := tftypes.NewValue(objType, attrValues)

	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: startingState},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: startingState.Copy()},
	}
	return req, resp, startingState
}

func notFoundResponseErr(status int, message string) *megaport.ErrorResponse {
	return &megaport.ErrorResponse{
		Response: &http.Response{
			StatusCode: status,
			Request:    &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/v2/product/mcr-1"}},
		},
		Message: message,
	}
}

func TestMCRPrefixFilterListRead(t *testing.T) {
	ctx := context.Background()

	livePrefixList := map[int]*megaport.MCRPrefixFilterList{
		7: {
			ID:            7,
			Description:   "live list",
			AddressFamily: "IPv4",
			Entries: []*megaport.MCRPrefixListEntry{
				{Action: "permit", Prefix: "10.0.0.0/24"},
			},
		},
	}

	tests := []struct {
		name string
		mock *MockMCRService
	}{
		{
			name: "decommissioned MCR",
			mock: &MockMCRService{
				GetMCRResult:                 &megaport.MCR{UID: "mcr-1", ProvisioningStatus: megaport.STATUS_DECOMMISSIONED},
				GetMCRPrefixFilterListResult: livePrefixList,
			},
		},
		{
			// A 200 with no product body means the MCR is gone too.
			name: "MCR returns no product",
			mock: &MockMCRService{
				GetMCRPrefixFilterListResult: livePrefixList,
			},
		},
		{
			name: "MCR returns 404",
			mock: &MockMCRService{
				GetMCRErr: notFoundResponseErr(http.StatusNotFound, "Service not found"),
			},
		},
		{
			// The 400 form IsServiceNotFoundError also treats as gone.
			name: "MCR returns 400 could not find a service",
			mock: &MockMCRService{
				GetMCRErr: notFoundResponseErr(http.StatusBadRequest, "Could not find a service with UID mcr-1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" removes the resource", func(t *testing.T) {
			r := &mcrPrefixFilterListResource{client: &megaport.Client{MCRService: tt.mock}}
			req, resp, _ := mcrPrefixFilterListReadRequest(t, r)

			r.Read(ctx, req, resp)

			require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
			require.True(t, resp.State.Raw.IsNull(), "expected the resource to be removed from state")
		})
	}

	t.Run("live MCR keeps the resource and reads the list", func(t *testing.T) {
		mock := &MockMCRService{
			GetMCRResult:                 &megaport.MCR{UID: "mcr-1", ProvisioningStatus: megaport.SERVICE_LIVE},
			GetMCRPrefixFilterListResult: livePrefixList,
		}
		r := &mcrPrefixFilterListResource{client: &megaport.Client{MCRService: mock}}
		req, resp, _ := mcrPrefixFilterListReadRequest(t, r)

		r.Read(ctx, req, resp)

		require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
		require.False(t, resp.State.Raw.IsNull())

		var state mcrPrefixFilterListResourceModel
		require.False(t, resp.State.Get(ctx, &state).HasError())
		require.Equal(t, "live list", state.Description.ValueString())
		require.Equal(t, int64(7), state.ID.ValueInt64())
	})

	t.Run("an unrelated MCR error keeps prior state", func(t *testing.T) {
		mock := &MockMCRService{
			GetMCRErr: notFoundResponseErr(http.StatusInternalServerError, "boom"),
		}
		r := &mcrPrefixFilterListResource{client: &megaport.Client{MCRService: mock}}
		req, resp, startingState := mcrPrefixFilterListReadRequest(t, r)

		r.Read(ctx, req, resp)

		require.True(t, resp.Diagnostics.HasError())
		require.True(t, resp.State.Raw.Equal(startingState), "expected prior state to survive the error")
	})
}
