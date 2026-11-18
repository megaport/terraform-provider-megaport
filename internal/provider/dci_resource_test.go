package provider

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	megaport "github.com/megaport/megaportgo"
)

const dciStubUID = "8da180e1-61ea-4961-a6e3-d80152ee9b14"

// dciStub answers the calls the DCI resource makes for one DCI.
type dciStub struct {
	// solution overrides fields of the DCI the stub returns.
	solution map[string]any
	// readStatus and readMessage, when set, fail every read of the DCI.
	readStatus  int
	readMessage string

	mu       sync.Mutex
	buyBody  map[string]any
	requests []string
}

func (s *dciStub) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.requests = append(s.requests, r.Method+" "+r.URL.Path)
		s.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v4/networkdesign/buy":
			body := map[string]any{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("could not decode the order: %v", err)
			}
			s.mu.Lock()
			s.buyBody = body
			s.mu.Unlock()
			solutions, _ := body["solutions"].([]any)
			if len(solutions) != 1 {
				t.Errorf("the order declares %d solutions, want 1", len(solutions))
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			declared, _ := solutions[0].(map[string]any)
			writeStubJSON(t, w, http.StatusOK, map[string]any{
				"message":   "ok",
				"data":      []map[string]any{{"technicalServiceUid": "a-end-uid"}, {"technicalServiceUid": "b-end-uid"}},
				"solutions": []map[string]any{{"uid": dciStubUID, "ref": declared["ref"], "type": "DCI", "name": declared["name"]}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/solutions/"+dciStubUID:
			if s.readStatus != 0 {
				writeStubJSON(t, w, s.readStatus, map[string]any{"message": s.readMessage})
				return
			}
			solution := map[string]any{
				"uid":                   dciStubUID,
				"type":                  "DCI",
				"name":                  "test-dci",
				"active":                true,
				"locked":                false,
				"serviceLevelReference": "SL-001",
				"provisioningStatus":    "CONFIGURED",
				"aLocationId":           143,
				"bLocationId":           145,
				"speedMbps":             10000,
				"createDate":            "2026-10-21T14:00:00Z",
				"members": []map[string]any{
					{"position": 0, "role": "A_END_PORT", "locationId": 143, "provisioningStatus": "CONFIGURED", "serviceName": "test-dci", "serviceType": "MEGAPORT", "serviceUid": "a-end-uid"},
					{"position": 1, "role": "B_END_PORT", "locationId": 145, "provisioningStatus": "CONFIGURED", "serviceName": "test-dci", "serviceType": "MEGAPORT", "serviceUid": "b-end-uid"},
					{"position": 2, "role": "VXC", "provisioningStatus": "CONFIGURED", "serviceName": "test-dci", "serviceType": "VXC", "serviceUid": "vxc-uid"},
				},
			}
			maps.Copy(solution, s.solution)
			writeStubJSON(t, w, http.StatusOK, map[string]any{"message": "ok", "data": solution})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/solutions/"+dciStubUID+"/billable":
			writeStubJSON(t, w, http.StatusOK, map[string]any{"message": "ok", "data": map[string]any{
				"autoRenewTerm":      true,
				"canReterm":          false,
				"contractStartDate":  nil,
				"contractEndDate":    nil,
				"contractTermMonths": 24,
				"price":              nil,
			}})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// newDCITestResource returns a DCI resource whose client talks to stub, with the resource's schema.
func newDCITestResource(t *testing.T, stub *dciStub) (*dciResource, fwresource.SchemaResponse, tftypes.Object) {
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
	r := &dciResource{client: client}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(t.Context(), fwresource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(t.Context()).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}
	return r, schemaResp, objType
}

// dciState returns a state that holds only the DCI's UID, as an import or a failed create leaves it.
func dciState(schemaResp fwresource.SchemaResponse, objType tftypes.Object) tfsdk.State {
	attrs := nullValueMap(objType)
	attrs["solution_uid"] = tftypes.NewValue(tftypes.String, dciStubUID)
	return tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, attrs)}
}

func dciModel(t *testing.T, state tfsdk.State) dciResourceModel {
	t.Helper()

	var model dciResourceModel
	if diags := state.Get(t.Context(), &model); diags.HasError() {
		t.Fatalf("reading state: %v", diags.Errors())
	}
	return model
}

func requireOneError(t *testing.T, diags diag.Diagnostics, summary string) {
	t.Helper()

	errs := diags.Errors()
	if len(errs) != 1 || !strings.Contains(errs[0].Summary(), summary) {
		t.Fatalf("errors = %v, want one %q error", errs, summary)
	}
}

// TestDCICreateOrdersOneDCI covers the order Create places, and the UID it saves when the DCI is not ready in time.
// The order decides what the customer pays for, so every setting must reach both Ports and the VXC.
func TestDCICreateOrdersOneDCI(t *testing.T) {
	t.Parallel()

	stub := &dciStub{}
	r, schemaResp, objType := newDCITestResource(t, stub)

	attrs := nullValueMap(objType)
	attrs["name"] = tftypes.NewValue(tftypes.String, "test-dci")
	attrs["a_end_location_id"] = tftypes.NewValue(tftypes.Number, 143)
	attrs["b_end_location_id"] = tftypes.NewValue(tftypes.Number, 145)
	attrs["speed"] = tftypes.NewValue(tftypes.Number, 10000)
	attrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, 24)
	attrs["auto_renew"] = tftypes.NewValue(tftypes.Bool, true)
	attrs["cost_centre"] = tftypes.NewValue(tftypes.String, "SL-001")
	raw := tftypes.NewValue(objType, attrs)

	// The wait checks the status every 30 seconds, so the deadline ends the wait first.
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	resp := fwresource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)}}
	r.Create(ctx, fwresource.CreateRequest{
		Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: raw},
	}, &resp)

	requireOneError(t, resp.Diagnostics, "ordered but not ready")
	if got := dciModel(t, resp.State).UID.ValueString(); got != dciStubUID {
		t.Fatalf("solution_uid in state = %q, want %q", got, dciStubUID)
	}

	stub.mu.Lock()
	defer stub.mu.Unlock()

	solutions, _ := stub.buyBody["solutions"].([]any)
	solution, _ := solutions[0].(map[string]any)
	if solution["type"] != "DCI" || solution["name"] != "test-dci" || solution["serviceLevelReference"] != "SL-001" {
		t.Errorf("the order declares the solution %v", solution)
	}

	design, _ := stub.buyBody["networkDesign"].([]any)
	if len(design) != 2 {
		t.Fatalf("the order holds %d ports, want 2", len(design))
	}
	aEnd, _ := design[0].(map[string]any)
	bEnd, _ := design[1].(map[string]any)
	vxcs, _ := aEnd["associatedVxcs"].([]any)
	if len(vxcs) != 1 {
		t.Fatalf("the A-End port carries %d VXCs, want 1", len(vxcs))
	}
	vxc, _ := vxcs[0].(map[string]any)

	if aEnd["locationId"] != float64(143) || bEnd["locationId"] != float64(145) {
		t.Errorf("the ports are ordered at locations %v and %v, want 143 and 145", aEnd["locationId"], bEnd["locationId"])
	}
	if aEnd["portSpeed"] != float64(10000) || bEnd["portSpeed"] != float64(10000) || vxc["rateLimit"] != float64(10000) {
		t.Errorf("the speeds are %v, %v and %v, want 10000 for each", aEnd["portSpeed"], bEnd["portSpeed"], vxc["rateLimit"])
	}
	for name, member := range map[string]map[string]any{"A-End port": aEnd, "B-End port": bEnd, "VXC": vxc} {
		if member["term"] != float64(24) || member["autoRenewTerm"] != true {
			t.Errorf("the %s is ordered with term %v and autoRenewTerm %v, want 24 and true", name, member["term"], member["autoRenewTerm"])
		}
		if member["solutionRef"] != solution["ref"] {
			t.Errorf("the %s points to the solution %v, want %v", name, member["solutionRef"], solution["ref"])
		}
	}
}

// TestDCIReadFillsState covers a read of a DCI that state knows only by UID, as after an import.
func TestDCIReadFillsState(t *testing.T) {
	t.Parallel()

	r, schemaResp, objType := newDCITestResource(t, &dciStub{})
	state := dciState(schemaResp, objType)

	resp := fwresource.ReadResponse{State: state}
	r.Read(t.Context(), fwresource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read errors: %v", resp.Diagnostics.Errors())
	}

	got := dciModel(t, resp.State)
	want := map[string][2]any{
		"solution_uid":         {got.UID.ValueString(), dciStubUID},
		"name":                 {got.Name.ValueString(), "test-dci"},
		"a_end_location_id":    {got.AEndLocationID.ValueInt64(), int64(143)},
		"b_end_location_id":    {got.BEndLocationID.ValueInt64(), int64(145)},
		"speed":                {got.Speed.ValueInt64(), int64(10000)},
		"contract_term_months": {got.ContractTermMonths.ValueInt64(), int64(24)},
		"auto_renew":           {got.AutoRenew.ValueBool(), true},
		"cost_centre":          {got.CostCentre.ValueString(), "SL-001"},
		"provisioning_status":  {got.ProvisioningStatus.ValueString(), "CONFIGURED"},
		"a_end_port_uid":       {got.AEndPortUID.ValueString(), "a-end-uid"},
		"b_end_port_uid":       {got.BEndPortUID.ValueString(), "b-end-uid"},
		"vxc_uid":              {got.VXCUID.ValueString(), "vxc-uid"},
	}
	for name, v := range want {
		if v[0] != v[1] {
			t.Errorf("%s = %v, want %v", name, v[0], v[1])
		}
	}
}

// TestDCIReadKeepsOrderedValuesBeforeVXC covers a DCI whose VXC is not created yet.
// The API reports no speed for it, and its locations only on the Ports.
func TestDCIReadKeepsOrderedValuesBeforeVXC(t *testing.T) {
	t.Parallel()

	stub := &dciStub{solution: map[string]any{"aLocationId": nil, "bLocationId": nil, "speedMbps": nil}}
	r, schemaResp, objType := newDCITestResource(t, stub)
	attrs := nullValueMap(objType)
	attrs["solution_uid"] = tftypes.NewValue(tftypes.String, dciStubUID)
	attrs["speed"] = tftypes.NewValue(tftypes.Number, 10000)
	state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, attrs)}

	resp := fwresource.ReadResponse{State: state}
	r.Read(t.Context(), fwresource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read errors: %v", resp.Diagnostics.Errors())
	}

	got := dciModel(t, resp.State)
	if got.Speed.ValueInt64() != 10000 {
		t.Errorf("speed = %d, want the ordered 10000", got.Speed.ValueInt64())
	}
	if got.AEndLocationID.ValueInt64() != 143 || got.BEndLocationID.ValueInt64() != 145 {
		t.Errorf("locations = %d and %d, want the ports' 143 and 145", got.AEndLocationID.ValueInt64(), got.BEndLocationID.ValueInt64())
	}
}

// TestDCIReadRemovesMissingDCI covers a DCI that was canceled outside Terraform.
func TestDCIReadRemovesMissingDCI(t *testing.T) {
	t.Parallel()

	cases := map[string]*dciStub{
		"not found": {readStatus: http.StatusNotFound, readMessage: "Solution not found"},
		"canceled":  {solution: map[string]any{"active": false}},
	}
	for name, stub := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r, schemaResp, objType := newDCITestResource(t, stub)
			state := dciState(schemaResp, objType)

			resp := fwresource.ReadResponse{State: state}
			r.Read(t.Context(), fwresource.ReadRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("Read errors: %v", resp.Diagnostics.Errors())
			}
			if !resp.State.Raw.IsNull() {
				t.Fatal("Read kept the DCI in state")
			}
		})
	}
}

// TestDCIReadKeepsStateWhenReadFails covers the reads that must not drop a live DCI from state.
// The API answers 404 on every DCI endpoint while DCI ordering is off for the account.
func TestDCIReadKeepsStateWhenReadFails(t *testing.T) {
	t.Parallel()

	cases := map[string]*dciStub{
		"ordering off": {readStatus: http.StatusNotFound, readMessage: "The requested endpoint is not found."},
		"server error": {readStatus: http.StatusInternalServerError, readMessage: "the stub fails the read"},
		"not a DCI":    {solution: map[string]any{"type": "OTHER"}},
	}
	for name, stub := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r, schemaResp, objType := newDCITestResource(t, stub)
			state := dciState(schemaResp, objType)

			resp := fwresource.ReadResponse{State: state}
			r.Read(t.Context(), fwresource.ReadRequest{State: state}, &resp)
			requireOneError(t, resp.Diagnostics, "Error Reading DCI")
			if resp.State.Raw.IsNull() {
				t.Fatal("Read removed the DCI from state")
			}
		})
	}
}

// TestDCIUpdateAndDeleteAreRejected covers the two actions this version cannot do.
// Neither may call the API or drop the DCI from state, because the DCI stays live and billed.
func TestDCIUpdateAndDeleteAreRejected(t *testing.T) {
	t.Parallel()

	stub := &dciStub{}
	r, schemaResp, objType := newDCITestResource(t, stub)
	state := dciState(schemaResp, objType)

	updateResp := fwresource.UpdateResponse{State: state}
	r.Update(t.Context(), fwresource.UpdateRequest{State: state, Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: state.Raw}}, &updateResp)
	requireOneError(t, updateResp.Diagnostics, "Changing a DCI is not supported")

	deleteResp := fwresource.DeleteResponse{State: state}
	r.Delete(t.Context(), fwresource.DeleteRequest{State: state}, &deleteResp)
	requireOneError(t, deleteResp.Diagnostics, "Canceling a DCI is not supported")
	if deleteResp.State.Raw.IsNull() {
		t.Fatal("Delete removed the DCI from state")
	}

	// A change of location or speed must reach Update, not force a replace: a replace orders a second DCI.
	for name, attr := range schemaResp.Schema.Attributes {
		if a, ok := attr.(schema.Int64Attribute); ok && len(a.PlanModifiers) != 0 {
			t.Errorf("%s has plan modifiers %v, want none", name, a.PlanModifiers)
		}
	}

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.requests) != 0 {
		t.Fatalf("Update and Delete called the API: %v", stub.requests)
	}
}
