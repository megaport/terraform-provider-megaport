// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	megaport "github.com/megaport/megaportgo"
)

// createWaitStub answers the calls that Create, Read, and Delete make for one product.
type createWaitStub struct {
	uid         string
	productType string
	product     map[string]any
	rejectBuy   bool

	mu        sync.Mutex
	cancelled bool
}

func (s *createWaitStub) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v3/networkdesign/validate":
			writeStubJSON(t, w, http.StatusOK, map[string]any{"message": "ok"})
		case r.Method == http.MethodPost && r.URL.Path == "/v4/networkdesign/buy":
			if s.rejectBuy {
				writeStubJSON(t, w, http.StatusBadRequest, map[string]any{"message": "the stub rejects the purchase"})
				return
			}
			writeStubJSON(t, w, http.StatusOK, map[string]any{
				"message": "ok",
				"data":    []map[string]any{{"technicalServiceUid": s.uid}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/product/"+s.uid:
			product := map[string]any{
				"productUid":         s.uid,
				"productType":        s.productType,
				"provisioningStatus": "CONFIGURED",
			}
			for k, v := range s.product {
				product[k] = v
			}
			writeStubJSON(t, w, http.StatusOK, map[string]any{"message": "ok", "data": product})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/product/"+s.uid+"/tags":
			writeStubJSON(t, w, http.StatusOK, map[string]any{"message": "ok", "data": map[string]any{"resourceTags": []any{}}})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/product/mcr2/"+s.uid+"/prefixLists":
			writeStubJSON(t, w, http.StatusOK, map[string]any{"message": "ok", "data": []any{}})
		case r.Method == http.MethodPost && r.URL.Path == "/v3/product/"+s.uid+"/action/CANCEL_NOW":
			s.mu.Lock()
			s.cancelled = true
			s.mu.Unlock()
			writeStubJSON(t, w, http.StatusOK, map[string]any{"message": "ok", "data": map[string]any{}})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

type createWaitCase struct {
	name        string
	productType string
	product     map[string]any
	newResource func(*megaport.Client) fwresource.Resource
	setPlan     func(attrs map[string]tftypes.Value, objType tftypes.Object)
}

func createWaitCases() []createWaitCase {
	return []createWaitCase{
		{
			name:        "port",
			productType: megaport.PRODUCT_MEGAPORT,
			newResource: func(c *megaport.Client) fwresource.Resource { return &portResource{client: c} },
			setPlan: func(attrs map[string]tftypes.Value, _ tftypes.Object) {
				attrs["port_speed"] = tftypes.NewValue(tftypes.Number, 10000)
				attrs["location_id"] = tftypes.NewValue(tftypes.Number, 1)
				attrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, 12)
			},
		},
		{
			name:        "lag_port",
			productType: megaport.PRODUCT_MEGAPORT,
			newResource: func(c *megaport.Client) fwresource.Resource { return &lagPortResource{client: c} },
			setPlan: func(attrs map[string]tftypes.Value, _ tftypes.Object) {
				attrs["port_speed"] = tftypes.NewValue(tftypes.Number, 10000)
				attrs["location_id"] = tftypes.NewValue(tftypes.Number, 1)
				attrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, 12)
				attrs["lag_count"] = tftypes.NewValue(tftypes.Number, 2)
			},
		},
		{
			name:        "mcr",
			productType: megaport.PRODUCT_MCR,
			newResource: func(c *megaport.Client) fwresource.Resource { return &mcrResource{client: c} },
			setPlan: func(attrs map[string]tftypes.Value, _ tftypes.Object) {
				attrs["port_speed"] = tftypes.NewValue(tftypes.Number, 1000)
				attrs["location_id"] = tftypes.NewValue(tftypes.Number, 1)
				attrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, 12)
			},
		},
		{
			name:        "mve",
			productType: megaport.PRODUCT_MVE,
			product:     map[string]any{"vnics": []map[string]any{{"vlan": 1, "description": "Data Plane"}}},
			newResource: func(c *megaport.Client) fwresource.Resource { return &mveResource{client: c} },
			setPlan: func(attrs map[string]tftypes.Value, objType tftypes.Object) {
				attrs["location_id"] = tftypes.NewValue(tftypes.Number, 1)
				attrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, 12)
				vendorType, _ := objType.AttributeTypes["vendor_config"].(tftypes.Object)
				vendorAttrs := nullValueMap(vendorType)
				vendorAttrs["vendor"] = tftypes.NewValue(tftypes.String, "6wind")
				attrs["vendor_config"] = tftypes.NewValue(vendorType, vendorAttrs)
			},
		},
		{
			name:        "ix",
			productType: megaport.PRODUCT_IX,
			newResource: func(c *megaport.Client) fwresource.Resource { return &ixResource{client: c} },
			setPlan: func(attrs map[string]tftypes.Value, _ tftypes.Object) {
				attrs["requested_product_uid"] = tftypes.NewValue(tftypes.String, "port-uid")
				attrs["network_service_type"] = tftypes.NewValue(tftypes.String, "Los Angeles IX")
				attrs["asn"] = tftypes.NewValue(tftypes.Number, 65000)
				attrs["mac_address"] = tftypes.NewValue(tftypes.String, "00:11:22:33:44:55")
				attrs["rate_limit"] = tftypes.NewValue(tftypes.Number, 1000)
				attrs["vlan"] = tftypes.NewValue(tftypes.Number, 100)
			},
		},
	}
}

// runStubCreate builds a plan for tc and runs Create against a stub API.
func runStubCreate(ctx context.Context, t *testing.T, tc createWaitCase, stub *createWaitStub) (fwresource.Resource, fwresource.CreateResponse) {
	t.Helper()

	server := httptest.NewServer(stub.handler(t))
	t.Cleanup(server.Close)
	client, err := megaport.New(nil,
		megaport.WithBaseURL(server.URL),
		megaport.WithAccessToken("test-token", time.Now().Add(time.Hour)),
	)
	if err != nil {
		t.Fatalf("megaport.New: %v", err)
	}
	r := tc.newResource(client)

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	objType, ok := s.Type().TerraformType(context.Background()).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}

	attrs := nullValueMap(objType)
	attrs["product_name"] = tftypes.NewValue(tftypes.String, "test-"+tc.name)
	tc.setPlan(attrs, objType)
	raw := tftypes.NewValue(objType, attrs)

	resp := fwresource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, nil)}}
	r.Create(ctx, fwresource.CreateRequest{
		Plan:   tfsdk.Plan{Schema: s, Raw: raw},
		Config: tfsdk.Config{Schema: s, Raw: raw},
	}, &resp)
	return r, resp
}

// TestCreateSavesUIDWhenWaitFails covers an order that goes through but never
// reaches a ready state. The UID in state lets the next apply refresh and
// replace the product instead of ordering a second one.
func TestCreateSavesUIDWhenWaitFails(t *testing.T) {
	t.Parallel()

	for _, tc := range createWaitCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stub := &createWaitStub{uid: tc.name + "-uid", productType: tc.productType, product: tc.product}

			// The SDK checks the status every 30 seconds, so the deadline ends the wait first.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			r, createResp := runStubCreate(ctx, t, tc, stub)

			errs := createResp.Diagnostics.Errors()
			if len(errs) != 1 || !strings.Contains(errs[0].Summary(), "ordered but not ready") {
				t.Fatalf("Create errors = %v, want one \"ordered but not ready\" error", errs)
			}
			var uid types.String
			if diags := createResp.State.GetAttribute(context.Background(), path.Root("product_uid"), &uid); diags.HasError() {
				t.Fatalf("reading product_uid: %v", diags.Errors())
			}
			if uid.ValueString() != stub.uid {
				t.Fatalf("product_uid in state = %q, want %q", uid.ValueString(), stub.uid)
			}

			readResp := fwresource.ReadResponse{State: createResp.State}
			r.Read(context.Background(), fwresource.ReadRequest{State: createResp.State}, &readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatalf("Read errors: %v", readResp.Diagnostics.Errors())
			}
			if readResp.State.Raw.IsNull() {
				t.Fatal("Read removed the resource from state")
			}

			deleteResp := fwresource.DeleteResponse{State: readResp.State}
			r.Delete(context.Background(), fwresource.DeleteRequest{State: readResp.State}, &deleteResp)
			if deleteResp.Diagnostics.HasError() {
				t.Fatalf("Delete errors: %v", deleteResp.Diagnostics.Errors())
			}
			stub.mu.Lock()
			defer stub.mu.Unlock()
			if !stub.cancelled {
				t.Fatal("Delete did not cancel the product")
			}
		})
	}
}

// TestCreateWritesNoStateWhenOrderRejected covers an order the API rejects:
// nothing was ordered, so nothing goes to state.
func TestCreateWritesNoStateWhenOrderRejected(t *testing.T) {
	t.Parallel()

	for _, tc := range createWaitCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stub := &createWaitStub{uid: tc.name + "-uid", productType: tc.productType, product: tc.product, rejectBuy: true}
			_, createResp := runStubCreate(context.Background(), t, tc, stub)

			errs := createResp.Diagnostics.Errors()
			if len(errs) != 1 || strings.Contains(errs[0].Summary(), "ordered but not ready") {
				t.Fatalf("Create errors = %v, want one order error", errs)
			}
			if !createResp.State.Raw.IsNull() {
				t.Fatalf("state = %v, want null", createResp.State.Raw)
			}
		})
	}
}
