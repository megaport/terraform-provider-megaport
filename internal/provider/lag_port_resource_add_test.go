// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

const lagPortStubUID = "lag-uid-1"

// lagPortAPIStub answers the calls addLagPorts makes and keeps the orders it
// receives, so a test can assert that nothing was ordered.
type lagPortAPIStub struct {
	aggregationID int
	memberUIDs    []string

	mu     sync.Mutex
	orders []map[string]any
}

func (s *lagPortAPIStub) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/product/" + lagPortStubUID:
			writeStubJSON(t, w, http.StatusOK, map[string]any{
				"message": "ok",
				"data":    stubPort(lagPortStubUID, s.aggregationID),
			})
		case "/v2/products":
			members := []map[string]any{}
			for _, uid := range s.memberUIDs {
				members = append(members, stubPort(uid, s.aggregationID))
			}
			writeStubJSON(t, w, http.StatusOK, map[string]any{"message": "ok", "data": members})
		case "/v3/networkdesign/validate":
			s.recordOrder(t, r)
			writeStubJSON(t, w, http.StatusOK, map[string]any{"message": "ok"})
		case "/v4/networkdesign/buy":
			s.recordOrder(t, r)
			// Reject the purchase. A successful one sends the SDK into a 30 second
			// provisioning poll, and the order body is already recorded here.
			writeStubJSON(t, w, http.StatusBadRequest, map[string]any{"message": "the stub rejects the purchase"})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

// recordOrder keeps the first port of the order, which is where the LAG fields sit.
func (s *lagPortAPIStub) recordOrder(t *testing.T, r *http.Request) {
	t.Helper()

	var body json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("could not read the order body: %v", err)
		return
	}

	// The buy call wraps the order, the validate call does not.
	var wrapped struct {
		NetworkDesign []map[string]any `json:"networkDesign"`
	}
	ports := wrapped.NetworkDesign
	if err := json.Unmarshal(body, &wrapped); err == nil && len(wrapped.NetworkDesign) > 0 {
		ports = wrapped.NetworkDesign
	} else if err := json.Unmarshal(body, &ports); err != nil {
		t.Errorf("could not decode the order body: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders = append(s.orders, ports[0])
}

func (s *lagPortAPIStub) recorded() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.orders
}

func stubPort(uid string, aggregationID int) map[string]any {
	return map[string]any{
		"productUid":    uid,
		"productType":   "MEGAPORT",
		"aggregationId": aggregationID,
	}
}

func writeStubJSON(t *testing.T, w http.ResponseWriter, status int, body map[string]any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Errorf("could not write the stub response: %v", err)
	}
}

// TestAddLagPorts covers the order addLagPorts places. The count it sends decides
// what the customer pays for, so the case that orders nothing matters as much as
// the case that orders the missing ports.
func TestAddLagPorts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		aggregationID int
		memberUIDs    []string
		target        int
		current       int
		wantOrder     bool
		wantLagCount  float64
		wantUIDs      []string
		wantError     bool
	}{
		{
			// The retry case. An apply can order the ports and still fail waiting for
			// them, and the live read already holds them.
			name:          "orders nothing when the LAG already has the ports",
			aggregationID: 7,
			memberUIDs:    []string{lagPortStubUID, "lag-uid-2", "lag-uid-3"},
			target:        3,
			current:       3,
			wantUIDs:      []string{lagPortStubUID, "lag-uid-2", "lag-uid-3"},
		},
		{
			name:          "orders only the missing ports",
			aggregationID: 7,
			memberUIDs:    []string{lagPortStubUID, "lag-uid-2"},
			target:        4,
			current:       2,
			wantOrder:     true,
			wantLagCount:  2,
			wantError:     true,
		},
		{
			// The product list read drops entries it cannot parse. Ordering against a
			// short count pushes the LAG past lag_count, and the next plan then
			// proposes replacing a LAG the customer is still using.
			name:          "orders nothing when the live read is short",
			aggregationID: 7,
			memberUIDs:    []string{lagPortStubUID, "lag-uid-2"},
			target:        4,
			current:       3,
			wantError:     true,
		},
		{
			name:          "orders nothing when the port reports no aggregation id",
			aggregationID: 0,
			target:        3,
			wantError:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stub := &lagPortAPIStub{aggregationID: tc.aggregationID, memberUIDs: tc.memberUIDs}
			server := httptest.NewServer(stub.handler(t))
			t.Cleanup(server.Close)

			client, err := megaport.New(nil,
				megaport.WithBaseURL(server.URL),
				megaport.WithAccessToken("test-token", time.Now().Add(time.Hour)),
			)
			if err != nil {
				t.Fatalf("megaport.New: %v", err)
			}

			r := &lagPortResource{client: client}
			plan := &lagPortResourceModel{
				UID:                   types.StringValue(lagPortStubUID),
				Name:                  types.StringValue("lag-one"),
				ContractTermMonths:    types.Int64Value(12),
				PortSpeed:             types.Int64Value(10000),
				LocationID:            types.Int64Value(5),
				MarketplaceVisibility: types.BoolValue(false),
				CostCentre:            types.StringValue("cc-1"),
				PromoCode:             types.StringValue("promo-1"),
				ResourceTags:          types.MapNull(types.StringType),
			}

			uids, diags := r.addLagPorts(context.Background(), plan, tc.target, tc.current)

			if diags.HasError() != tc.wantError {
				t.Fatalf("error = %v, want %v (diags: %v)", diags.HasError(), tc.wantError, diags.Errors())
			}

			orders := stub.recorded()
			if !tc.wantOrder {
				if len(orders) > 0 {
					t.Fatalf("expected no order, got %v", orders)
				}
				if got, want := len(uids), len(tc.wantUIDs); got != want {
					t.Fatalf("returned %d UIDs, want %d", got, want)
				}
				for i, uid := range tc.wantUIDs {
					if uids[i] != uid {
						t.Errorf("UID %d = %q, want %q", i, uids[i], uid)
					}
				}
				return
			}

			if len(orders) == 0 {
				t.Fatal("expected an order, got none")
			}
			want := map[string]any{
				"aggregationId":         float64(tc.aggregationID),
				"lagPortCount":          tc.wantLagCount,
				"portSpeed":             float64(10000),
				"locationId":            float64(5),
				"term":                  float64(12),
				"promoCode":             "promo-1",
				"costCentre":            "cc-1",
				"marketplaceVisibility": false,
			}
			for field, wantValue := range want {
				if got := orders[0][field]; got != wantValue {
					t.Errorf("order %s = %v, want %v", field, got, wantValue)
				}
			}
		})
	}
}
