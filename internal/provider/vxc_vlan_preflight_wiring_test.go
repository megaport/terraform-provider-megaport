package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	megaport "github.com/megaport/megaportgo"
)

// vlanQuery is one hit on the VLAN availability endpoint.
type vlanQuery struct {
	portUID string
	vlan    string
}

// preflightServer fakes the two endpoints the preflight needs. Every port is
// a MEGAPORT, every VLAN in taken is unavailable, and any other path returns
// 500 so Create and Update fail fast after the preflight instead of ordering.
type preflightServer struct {
	*httptest.Server
	mu           sync.Mutex
	vlanQueries  []vlanQuery
	productTypes []string

	// serviceKeyBEnd is the port UID a service key lookup resolves to.
	serviceKeyBEnd string
}

func newPreflightServer(t *testing.T, taken map[string]int) *preflightServer {
	t.Helper()
	ps := &preflightServer{}
	ps.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ps.mu.Lock()
		defer ps.mu.Unlock()

		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		isGetV2 := r.Method == http.MethodGet && parts[0] == "v2"
		switch {
		case isGetV2 && len(parts) == 5 && parts[1] == "product" && parts[2] == "port" && parts[4] == "vlan":
			portUID := parts[3]
			vlan := r.URL.Query().Get("vlan")
			ps.vlanQueries = append(ps.vlanQueries, vlanQuery{portUID: portUID, vlan: vlan})
			v, _ := strconv.Atoi(vlan)
			data := []int{}
			if taken[portUID] != v {
				data = append(data, v)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
		case isGetV2 && len(parts) == 3 && parts[1] == "service" && strings.HasPrefix(parts[2], "key"):
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]string{"productUid": ps.serviceKeyBEnd}})
		case isGetV2 && len(parts) == 3 && parts[1] == "product":
			ps.productTypes = append(ps.productTypes, parts[2])
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]string{"productType": megaport.PRODUCT_MEGAPORT}})
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"not faked"}`))
		}
	}))
	t.Cleanup(ps.Close)
	return ps
}

func (ps *preflightServer) resource(t *testing.T) *vxcResource {
	t.Helper()
	client, err := megaport.New(nil,
		megaport.WithBaseURL(ps.URL),
		megaport.WithAccessToken("test-token", time.Now().Add(time.Hour)),
	)
	if err != nil {
		t.Fatalf("megaport.New: %v", err)
	}
	return &vxcResource{client: client}
}

// vxcEndSpec is the subset of a_end / b_end the preflight reads.
type vxcEndSpec struct {
	productUID  string
	orderedVLAN *int64
	vlan        *int64
}

type vxcValueBuilder struct {
	schema     fwschema.Schema
	objType    tftypes.Object
	endType    tftypes.Object
	partnerTyp tftypes.Object
}

func newVXCValueBuilder(t *testing.T) *vxcValueBuilder {
	t.Helper()
	ctx := context.Background()
	schemaResp := fwresource.SchemaResponse{}
	(&vxcResource{}).Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}
	endType, ok := objType.AttributeTypes["a_end"].(tftypes.Object)
	if !ok {
		t.Fatal("a_end type is not tftypes.Object")
	}
	partnerType, ok := objType.AttributeTypes["b_end_partner_config"].(tftypes.Object)
	if !ok {
		t.Fatal("b_end_partner_config type is not tftypes.Object")
	}
	return &vxcValueBuilder{
		schema:     schemaResp.Schema,
		objType:    objType,
		endType:    endType,
		partnerTyp: partnerType,
	}
}

func (b *vxcValueBuilder) end(spec vxcEndSpec) tftypes.Value {
	attrs := nullValueMap(b.endType)
	attrs["requested_product_uid"] = tftypes.NewValue(tftypes.String, spec.productUID)
	if spec.orderedVLAN != nil {
		attrs["ordered_vlan"] = tftypes.NewValue(tftypes.Number, *spec.orderedVLAN)
	}
	if spec.vlan != nil {
		attrs["vlan"] = tftypes.NewValue(tftypes.Number, *spec.vlan)
	}
	return tftypes.NewValue(b.endType, attrs)
}

// transitPartner is the simplest non-null partner config: no nested object.
func (b *vxcValueBuilder) transitPartner() tftypes.Value {
	attrs := nullValueMap(b.partnerTyp)
	attrs["partner"] = tftypes.NewValue(tftypes.String, "transit")
	return tftypes.NewValue(b.partnerTyp, attrs)
}

// withServiceKey returns v with service_key set. The builder leaves it null.
func (b *vxcValueBuilder) withServiceKey(t *testing.T, v tftypes.Value, key string) tftypes.Value {
	t.Helper()
	attrs := map[string]tftypes.Value{}
	if err := v.As(&attrs); err != nil {
		t.Fatalf("unpacking vxc value: %v", err)
	}
	attrs["service_key"] = tftypes.NewValue(tftypes.String, key)
	return tftypes.NewValue(b.objType, attrs)
}

func (b *vxcValueBuilder) vxc(aEnd, bEnd tftypes.Value, bEndPartner *tftypes.Value) tftypes.Value {
	attrs := nullValueMap(b.objType)
	attrs["product_name"] = tftypes.NewValue(tftypes.String, "test-vxc")
	attrs["rate_limit"] = tftypes.NewValue(tftypes.Number, 1000)
	attrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, 1)
	attrs["a_end"] = aEnd
	attrs["b_end"] = bEnd
	if bEndPartner != nil {
		attrs["b_end_partner_config"] = *bEndPartner
	}
	return tftypes.NewValue(b.objType, attrs)
}

func int64p(v int64) *int64 { return &v }

func TestVXCCreate_VLANPreflightBlocksTakenAEndVLAN(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ps := newPreflightServer(t, map[string]int{"port-a": 730})
	b := newVXCValueBuilder(t)

	plan := b.vxc(
		b.end(vxcEndSpec{productUID: "port-a", orderedVLAN: int64p(730)}),
		b.end(vxcEndSpec{productUID: "port-b", orderedVLAN: int64p(100)}),
		nil,
	)

	resp := fwresource.CreateResponse{State: tfsdk.State{Schema: b.schema}}
	ps.resource(t).Create(ctx, fwresource.CreateRequest{Plan: tfsdk.Plan{Schema: b.schema, Raw: plan}}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected Create to fail on the taken A-End VLAN")
	}
	summary := resp.Diagnostics.Errors()[0].Summary()
	if summary != "VLAN 730 is not available on the A-End port" {
		t.Fatalf("unexpected error summary: %q", summary)
	}
	if got := ps.vlanQueries; len(got) != 1 || got[0] != (vlanQuery{"port-a", "730"}) {
		t.Fatalf("expected one VLAN query for port-a/730, got %v", got)
	}
}

func TestVXCCreate_VLANPreflightSkipsPartnerConfiguredBEnd(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ps := newPreflightServer(t, map[string]int{"port-b": 100})
	b := newVXCValueBuilder(t)

	partner := b.transitPartner()
	plan := b.vxc(
		b.end(vxcEndSpec{productUID: "port-a", orderedVLAN: int64p(730)}),
		b.end(vxcEndSpec{productUID: "port-b", orderedVLAN: int64p(100)}),
		&partner,
	)

	resp := fwresource.CreateResponse{State: tfsdk.State{Schema: b.schema}}
	ps.resource(t).Create(ctx, fwresource.CreateRequest{Plan: tfsdk.Plan{Schema: b.schema, Raw: plan}}, &resp)

	// The B-End product type lookup sits right before the B-End preflight, so
	// seeing it proves Create reached the preflight rather than exiting early.
	if !slices.Contains(ps.productTypes, "port-b") {
		t.Fatalf("expected a B-End product type lookup, got %v", ps.productTypes)
	}
	if got := ps.vlanQueries; len(got) != 1 || got[0].portUID != "port-a" {
		t.Fatalf("expected only the A-End VLAN query, got %v", got)
	}
}

func TestVXCUpdate_VLANPreflightOnlyOnOrderedVLANChange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)

	// B-End vlan differs from ordered_vlan so only the plan-vs-state gate in
	// Update, not the helper's same-VLAN guard, keeps the check off.
	state := b.vxc(
		b.end(vxcEndSpec{productUID: "port-a", orderedVLAN: int64p(100), vlan: int64p(100)}),
		b.end(vxcEndSpec{productUID: "port-b", orderedVLAN: int64p(200), vlan: int64p(250)}),
		nil,
	)

	t.Run("unchanged ordered_vlan is not checked", func(t *testing.T) {
		t.Parallel()
		ps := newPreflightServer(t, map[string]int{"port-a": 100, "port-b": 200})

		resp := fwresource.UpdateResponse{State: tfsdk.State{Schema: b.schema, Raw: state}}
		ps.resource(t).Update(ctx, fwresource.UpdateRequest{
			Plan:  tfsdk.Plan{Schema: b.schema, Raw: state},
			State: tfsdk.State{Schema: b.schema, Raw: state},
		}, &resp)

		if !slices.Contains(ps.productTypes, "port-b") {
			t.Fatalf("expected a B-End product type lookup, got %v", ps.productTypes)
		}
		if len(ps.vlanQueries) != 0 {
			t.Fatalf("expected no VLAN queries, got %v", ps.vlanQueries)
		}
	})

	t.Run("changed ordered_vlan is checked and blocks when taken", func(t *testing.T) {
		t.Parallel()
		ps := newPreflightServer(t, map[string]int{"port-b": 300})

		plan := b.vxc(
			b.end(vxcEndSpec{productUID: "port-a", orderedVLAN: int64p(100), vlan: int64p(100)}),
			b.end(vxcEndSpec{productUID: "port-b", orderedVLAN: int64p(300), vlan: int64p(250)}),
			nil,
		)

		resp := fwresource.UpdateResponse{State: tfsdk.State{Schema: b.schema, Raw: state}}
		ps.resource(t).Update(ctx, fwresource.UpdateRequest{
			Plan:  tfsdk.Plan{Schema: b.schema, Raw: plan},
			State: tfsdk.State{Schema: b.schema, Raw: state},
		}, &resp)

		if !resp.Diagnostics.HasError() {
			t.Fatal("expected Update to fail on the taken B-End VLAN")
		}
		summary := resp.Diagnostics.Errors()[0].Summary()
		if summary != "VLAN 300 is not available on the B-End port" {
			t.Fatalf("unexpected error summary: %q", summary)
		}
		if got := ps.vlanQueries; len(got) != 1 || got[0] != (vlanQuery{"port-b", "300"}) {
			t.Fatalf("expected one VLAN query for port-b/300, got %v", got)
		}
	})
}

// A port move re-requests the VLAN on the new port without changing
// ordered_vlan, so the plan-vs-state VLAN comparison alone never fires.
func TestVXCUpdate_VLANPreflightOnPortMove(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)

	state := b.vxc(
		b.end(vxcEndSpec{productUID: "port-a", orderedVLAN: int64p(100), vlan: int64p(100)}),
		b.end(vxcEndSpec{productUID: "port-b", orderedVLAN: int64p(200), vlan: int64p(200)}),
		nil,
	)

	t.Run("a-end move is checked against the new port", func(t *testing.T) {
		t.Parallel()
		// VLAN 100 is taken on the destination. The end already holds 100 on
		// its old port, so a current-VLAN of 100 would wrongly skip the check.
		ps := newPreflightServer(t, map[string]int{"port-c": 100})

		plan := b.vxc(
			b.end(vxcEndSpec{productUID: "port-c", orderedVLAN: int64p(100), vlan: int64p(100)}),
			b.end(vxcEndSpec{productUID: "port-b", orderedVLAN: int64p(200), vlan: int64p(200)}),
			nil,
		)

		resp := fwresource.UpdateResponse{State: tfsdk.State{Schema: b.schema, Raw: state}}
		ps.resource(t).Update(ctx, fwresource.UpdateRequest{
			Plan:  tfsdk.Plan{Schema: b.schema, Raw: plan},
			State: tfsdk.State{Schema: b.schema, Raw: state},
		}, &resp)

		if !resp.Diagnostics.HasError() {
			t.Fatal("expected Update to fail on the taken VLAN at the move destination")
		}
		summary := resp.Diagnostics.Errors()[0].Summary()
		if summary != "VLAN 100 is not available on the A-End port" {
			t.Fatalf("unexpected error summary: %q", summary)
		}
		if got := ps.vlanQueries; len(got) != 1 || got[0] != (vlanQuery{"port-c", "100"}) {
			t.Fatalf("expected one VLAN query for port-c/100, got %v", got)
		}
	})

	t.Run("a-end move passes when the vlan is free there", func(t *testing.T) {
		t.Parallel()
		ps := newPreflightServer(t, nil)

		plan := b.vxc(
			b.end(vxcEndSpec{productUID: "port-c", orderedVLAN: int64p(100), vlan: int64p(100)}),
			b.end(vxcEndSpec{productUID: "port-b", orderedVLAN: int64p(200), vlan: int64p(200)}),
			nil,
		)

		resp := fwresource.UpdateResponse{State: tfsdk.State{Schema: b.schema, Raw: state}}
		ps.resource(t).Update(ctx, fwresource.UpdateRequest{
			Plan:  tfsdk.Plan{Schema: b.schema, Raw: plan},
			State: tfsdk.State{Schema: b.schema, Raw: state},
		}, &resp)

		for _, d := range resp.Diagnostics.Errors() {
			if strings.Contains(d.Summary(), "is not available on the") {
				t.Fatalf("preflight blocked a free VLAN: %q", d.Summary())
			}
		}
		if got := ps.vlanQueries; len(got) != 1 || got[0] != (vlanQuery{"port-c", "100"}) {
			t.Fatalf("expected one VLAN query for port-c/100, got %v", got)
		}
	})

	t.Run("b-end move is checked against the new port", func(t *testing.T) {
		t.Parallel()
		ps := newPreflightServer(t, map[string]int{"port-d": 200})

		plan := b.vxc(
			b.end(vxcEndSpec{productUID: "port-a", orderedVLAN: int64p(100), vlan: int64p(100)}),
			b.end(vxcEndSpec{productUID: "port-d", orderedVLAN: int64p(200), vlan: int64p(200)}),
			nil,
		)

		resp := fwresource.UpdateResponse{State: tfsdk.State{Schema: b.schema, Raw: state}}
		ps.resource(t).Update(ctx, fwresource.UpdateRequest{
			Plan:  tfsdk.Plan{Schema: b.schema, Raw: plan},
			State: tfsdk.State{Schema: b.schema, Raw: state},
		}, &resp)

		if !resp.Diagnostics.HasError() {
			t.Fatal("expected Update to fail on the taken VLAN at the move destination")
		}
		summary := resp.Diagnostics.Errors()[0].Summary()
		if summary != "VLAN 200 is not available on the B-End port" {
			t.Fatalf("unexpected error summary: %q", summary)
		}
		if got := ps.vlanQueries; len(got) != 1 || got[0] != (vlanQuery{"port-d", "200"}) {
			t.Fatalf("expected one VLAN query for port-d/200, got %v", got)
		}
	})
}

// With ordered_vlan omitted or set to auto-assign, the move request carries no
// VLAN, and the API then validates the VLAN the VXC already holds.
func TestVXCUpdate_VLANPreflightOnPortMoveWithoutOrderedVLAN(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)

	for _, tc := range []struct {
		name        string
		orderedVLAN *int64
	}{
		{"ordered_vlan omitted", nil},
		{"ordered_vlan auto-assign", int64p(0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ps := newPreflightServer(t, map[string]int{"port-c": 100})

			aEnd := func(uid string) tftypes.Value {
				return b.end(vxcEndSpec{productUID: uid, orderedVLAN: tc.orderedVLAN, vlan: int64p(100)})
			}
			bEnd := b.end(vxcEndSpec{productUID: "port-b", orderedVLAN: int64p(200), vlan: int64p(200)})
			state := b.vxc(aEnd("port-a"), bEnd, nil)

			resp := fwresource.UpdateResponse{State: tfsdk.State{Schema: b.schema, Raw: state}}
			ps.resource(t).Update(ctx, fwresource.UpdateRequest{
				Plan:  tfsdk.Plan{Schema: b.schema, Raw: b.vxc(aEnd("port-c"), bEnd, nil)},
				State: tfsdk.State{Schema: b.schema, Raw: state},
			}, &resp)

			if !resp.Diagnostics.HasError() {
				t.Fatal("expected Update to fail on the VLAN the move carries over")
			}
			summary := resp.Diagnostics.Errors()[0].Summary()
			if summary != "VLAN 100 is not available on the A-End port" {
				t.Fatalf("unexpected error summary: %q", summary)
			}
			if got := ps.vlanQueries; len(got) != 1 || got[0] != (vlanQuery{"port-c", "100"}) {
				t.Fatalf("expected one VLAN query for port-c/100, got %v", got)
			}
		})
	}
}

// A service key redirects the order to its own B-End, so the port named in
// config is not the one the VXC uses and must not be checked.
func TestVXCUpdate_VLANPreflightSkipsServiceKeyBEnd(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)
	// VLAN 300 is taken on the named port, so an unskipped check would block.
	ps := newPreflightServer(t, map[string]int{"port-b": 300})

	withKey := func(v tftypes.Value) tftypes.Value {
		return b.withServiceKey(t, v, "test-service-key")
	}

	aEnd := b.end(vxcEndSpec{productUID: "port-a", orderedVLAN: int64p(100), vlan: int64p(100)})
	state := withKey(b.vxc(aEnd, b.end(vxcEndSpec{productUID: "port-b", orderedVLAN: int64p(200), vlan: int64p(200)}), nil))
	plan := withKey(b.vxc(aEnd, b.end(vxcEndSpec{productUID: "port-b", orderedVLAN: int64p(300), vlan: int64p(200)}), nil))

	resp := fwresource.UpdateResponse{State: tfsdk.State{Schema: b.schema, Raw: state}}
	ps.resource(t).Update(ctx, fwresource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: b.schema, Raw: plan},
		State: tfsdk.State{Schema: b.schema, Raw: state},
	}, &resp)

	for _, d := range resp.Diagnostics.Errors() {
		if strings.Contains(d.Summary(), "is not available on the") {
			t.Fatalf("preflight ran on a service-key B-End: %q", d.Summary())
		}
	}
	if len(ps.vlanQueries) != 0 {
		t.Fatalf("expected no VLAN query, got %v", ps.vlanQueries)
	}
}

func TestVXCCreate_VLANPreflightBlocksTakenBEndVLAN(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)
	// The A-End VLAN is free, so Create reaches the B-End check.
	ps := newPreflightServer(t, map[string]int{"port-b": 200})

	plan := b.vxc(
		b.end(vxcEndSpec{productUID: "port-a", orderedVLAN: int64p(100)}),
		b.end(vxcEndSpec{productUID: "port-b", orderedVLAN: int64p(200)}),
		nil,
	)

	resp := fwresource.CreateResponse{State: tfsdk.State{Schema: b.schema}}
	ps.resource(t).Create(ctx, fwresource.CreateRequest{Plan: tfsdk.Plan{Schema: b.schema, Raw: plan}}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected Create to fail on the taken B-End VLAN")
	}
	summary := resp.Diagnostics.Errors()[0].Summary()
	if summary != "VLAN 200 is not available on the B-End port" {
		t.Fatalf("unexpected error summary: %q", summary)
	}
	want := []vlanQuery{{"port-a", "100"}, {"port-b", "200"}}
	if !slices.Equal(ps.vlanQueries, want) {
		t.Fatalf("expected queries %v, got %v", want, ps.vlanQueries)
	}
}

func TestVXCCreate_VLANPreflightSkipsServiceKeyBEnd(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b := newVXCValueBuilder(t)
	// VLAN 200 is taken on the port named in config, so an unskipped check
	// would block a create the service key redirects elsewhere.
	ps := newPreflightServer(t, map[string]int{"port-b": 200})
	ps.serviceKeyBEnd = "port-key"

	plan := b.withServiceKey(t, b.vxc(
		b.end(vxcEndSpec{productUID: "port-a", orderedVLAN: int64p(100)}),
		b.end(vxcEndSpec{productUID: "port-b", orderedVLAN: int64p(200)}),
		nil,
	), "test-service-key")

	resp := fwresource.CreateResponse{State: tfsdk.State{Schema: b.schema}}
	ps.resource(t).Create(ctx, fwresource.CreateRequest{Plan: tfsdk.Plan{Schema: b.schema, Raw: plan}}, &resp)

	for _, d := range resp.Diagnostics.Errors() {
		if strings.Contains(d.Summary(), "is not available on the") {
			t.Fatalf("preflight ran on a service-key B-End: %q", d.Summary())
		}
	}
	// The B-End product type lookup sits right before the B-End preflight, so
	// seeing it proves Create reached the preflight rather than exiting early.
	if !slices.Contains(ps.productTypes, "port-b") {
		t.Fatalf("expected a B-End product type lookup, got %v", ps.productTypes)
	}
	want := []vlanQuery{{"port-a", "100"}}
	if !slices.Equal(ps.vlanQueries, want) {
		t.Fatalf("expected queries %v, got %v", want, ps.vlanQueries)
	}
}
