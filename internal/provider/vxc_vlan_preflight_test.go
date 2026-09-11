package provider

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
)

// stubPortService embeds the interface so only the one method under test needs
// implementing; any other call would nil-panic and fail the test loudly.
type stubPortService struct {
	megaport.PortService
	available bool
	err       error
	calls     int
	gotPortID string
	gotVLAN   int
}

func (s *stubPortService) CheckPortVLANAvailability(_ context.Context, portID string, vlan int) (bool, error) {
	s.calls++
	s.gotPortID = portID
	s.gotVLAN = vlan
	return s.available, s.err
}

func TestVLANAvailabilityPreflight(t *testing.T) {
	tests := []struct {
		name             string
		productUID       string
		productType      string
		orderedVLAN      types.Int64
		currentVLAN      types.Int64
		hasPartnerConfig bool
		available        bool
		apiErr           error
		wantCalls        int
		wantError        bool
	}{
		{
			name:        "taken vlan errors",
			productUID:  "port-uid-1",
			productType: megaport.PRODUCT_MEGAPORT,
			orderedVLAN: types.Int64Value(730),
			available:   false,
			wantCalls:   1,
			wantError:   true,
		},
		{
			// Pinning the VLAN the API already allocated is a no-op, but the
			// port genuinely holds it, so an unguarded check would reject it.
			name:        "pinning the vlan this end already holds is not checked",
			productUID:  "port-uid-1",
			productType: megaport.PRODUCT_MEGAPORT,
			orderedVLAN: types.Int64Value(730),
			currentVLAN: types.Int64Value(730),
			available:   false,
		},
		{
			name:        "changing away from the current vlan is checked",
			productUID:  "port-uid-1",
			productType: megaport.PRODUCT_MEGAPORT,
			orderedVLAN: types.Int64Value(730),
			currentVLAN: types.Int64Value(600),
			available:   false,
			wantCalls:   1,
			wantError:   true,
		},
		{
			name:        "free vlan passes",
			productUID:  "port-uid-1",
			productType: megaport.PRODUCT_MEGAPORT,
			orderedVLAN: types.Int64Value(730),
			available:   true,
			wantCalls:   1,
		},
		{
			name:        "the requested vlan is the one checked",
			productUID:  "port-uid-1",
			productType: megaport.PRODUCT_MEGAPORT,
			orderedVLAN: types.Int64Value(2049),
			available:   false,
			wantCalls:   1,
			wantError:   true,
		},
		{
			// Megaport may rotate a Partner Port to a sibling, so the requested
			// port's answer can be about a port the order never lands on.
			name:             "partner-configured end is not checked",
			productUID:       "partner-uid-1",
			productType:      megaport.PRODUCT_MEGAPORT,
			orderedVLAN:      types.Int64Value(730),
			hasPartnerConfig: true,
			available:        false,
		},
		{
			// A preflight that cannot answer must never block a valid apply.
			name:        "api error fails open",
			productUID:  "port-uid-1",
			productType: megaport.PRODUCT_MEGAPORT,
			orderedVLAN: types.Int64Value(730),
			apiErr:      errors.New("403 forbidden"),
			wantCalls:   1,
		},
		{
			name:        "product type is matched case-insensitively",
			productUID:  "port-uid-1",
			productType: "MEGAPORT",
			orderedVLAN: types.Int64Value(730),
			available:   false,
			wantCalls:   1,
			wantError:   true,
		},
		{
			name:        "auto-assign vlan is not checked",
			productUID:  "port-uid-1",
			productType: megaport.PRODUCT_MEGAPORT,
			orderedVLAN: types.Int64Value(0),
			available:   false,
		},
		{
			name:        "untagged vlan is not checked",
			productUID:  "port-uid-1",
			productType: megaport.PRODUCT_MEGAPORT,
			orderedVLAN: types.Int64Value(-1),
			available:   false,
		},
		{
			name:        "null vlan is not checked",
			productUID:  "port-uid-1",
			productType: megaport.PRODUCT_MEGAPORT,
			orderedVLAN: types.Int64Null(),
			available:   false,
		},
		{
			name:        "unknown vlan is not checked",
			productUID:  "port-uid-1",
			productType: megaport.PRODUCT_MEGAPORT,
			orderedVLAN: types.Int64Unknown(),
			available:   false,
		},
		{
			// MVE VLANs are scoped per vNIC, so a per-service answer would be wrong.
			name:        "mve end is not checked",
			productUID:  "mve-uid-1",
			productType: megaport.PRODUCT_MVE,
			orderedVLAN: types.Int64Value(730),
			available:   false,
		},
		{
			name:        "mcr end is not checked",
			productUID:  "mcr-uid-1",
			productType: megaport.PRODUCT_MCR,
			orderedVLAN: types.Int64Value(730),
			available:   false,
		},
		{
			// GetProductType failed, or the end is a partner port we do not own.
			name:        "unresolved product type is not checked",
			productUID:  "partner-uid-1",
			productType: "",
			orderedVLAN: types.Int64Value(730),
			available:   false,
		},
		{
			name:        "empty product uid is not checked",
			productUID:  "",
			productType: megaport.PRODUCT_MEGAPORT,
			orderedVLAN: types.Int64Value(730),
			available:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &stubPortService{available: tt.available, err: tt.apiErr}

			// A zero-value types.Int64 is null, which is the create case.
			diags := vlanAvailabilityPreflight(context.Background(), vlanPreflightInput{
				svc:              svc,
				end:              "A-End",
				productUID:       tt.productUID,
				productType:      tt.productType,
				orderedVLAN:      tt.orderedVLAN,
				currentVLAN:      tt.currentVLAN,
				hasPartnerConfig: tt.hasPartnerConfig,
			})

			assert.Equal(t, tt.wantCalls, svc.calls, "unexpected number of availability calls")
			if !tt.wantError {
				assert.False(t, diags.HasError(), "unexpected error: %v", diags)
				return
			}

			vlan := int(tt.orderedVLAN.ValueInt64())
			assert.True(t, diags.HasError(), "expected an error diagnostic")
			summary := diags.Errors()[0].Summary()
			detail := diags.Errors()[0].Detail()
			assert.Contains(t, summary, strconv.Itoa(vlan), "error should name the VLAN")
			assert.Contains(t, summary, "A-End", "error should name the end")
			assert.Contains(t, detail, tt.productUID, "error should name the port")
			assert.Contains(t, detail, "ordered_vlan", "error should name the attribute to change")

			assert.Equal(t, tt.productUID, svc.gotPortID)
			assert.Equal(t, vlan, svc.gotVLAN, "the requested VLAN should be the one checked")
		})
	}
}
