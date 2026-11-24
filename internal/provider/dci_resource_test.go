package provider

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	megaport "github.com/megaport/megaportgo"
)

const dciStubUID = "8da180e1-61ea-4961-a6e3-d80152ee9b14"

// dciStub answers the calls the DCI resource makes for one DCI.
// Set the fields above mu before the server starts: the handler reads them without the lock.
type dciStub struct {
	// solution overrides fields of the DCI the stub returns.
	solution map[string]any
	// billable overrides fields of the billing the stub returns.
	billable map[string]any
	// readStatus and readMessage, when set, fail every read of the DCI.
	readStatus  int
	readMessage string

	mu         sync.Mutex
	buyBody    map[string]any
	cancelBody map[string]any
	requests   []string
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
				w.WriteHeader(http.StatusBadRequest)
				return
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
		case r.Method == http.MethodPost && r.URL.Path == "/v1/solutions/"+dciStubUID+"/cancel":
			body := map[string]any{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("could not decode the cancel: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			s.mu.Lock()
			s.cancelBody = body
			s.mu.Unlock()
			writeStubJSON(t, w, http.StatusOK, map[string]any{"message": "ok"})
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
			billable := map[string]any{
				"autoRenewTerm":      true,
				"canReterm":          false,
				"contractStartDate":  nil,
				"contractEndDate":    nil,
				"contractTermMonths": 24,
				"price":              nil,
			}
			maps.Copy(billable, s.billable)
			writeStubJSON(t, w, http.StatusOK, map[string]any{"message": "ok", "data": billable})
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

// dciPlan returns the plan of a new DCI.
func dciPlan(objType tftypes.Object) tftypes.Value {
	attrs := nullValueMap(objType)
	attrs["name"] = tftypes.NewValue(tftypes.String, "test-dci")
	attrs["a_end_location_id"] = tftypes.NewValue(tftypes.Number, 143)
	attrs["b_end_location_id"] = tftypes.NewValue(tftypes.Number, 145)
	attrs["speed"] = tftypes.NewValue(tftypes.Number, 10000)
	attrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, 24)
	attrs["auto_renew"] = tftypes.NewValue(tftypes.Bool, true)
	attrs["cost_centre"] = tftypes.NewValue(tftypes.String, "SL-001")
	return tftypes.NewValue(objType, attrs)
}

// newDCIMockResource returns a DCI resource whose client is mock, with the resource's schema.
// The mock answers at once, where the SDK waits 30 seconds between checks of a new DCI.
func newDCIMockResource(t *testing.T, mock *MockSolutionService) (*dciResource, fwresource.SchemaResponse, tftypes.Object) {
	t.Helper()

	r := &dciResource{client: &megaport.Client{SolutionService: mock}}
	schemaResp := fwresource.SchemaResponse{}
	r.Schema(t.Context(), fwresource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(t.Context()).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}
	return r, schemaResp, objType
}

// dciMock returns a mock that orders a DCI and reads it back live.
func dciMock() *MockSolutionService {
	return &MockSolutionService{
		BuyDCIResult: &megaport.BuyDCIResponse{SolutionUID: dciStubUID},
		GetSolutionResult: &megaport.Solution{
			UID:                   dciStubUID,
			Type:                  megaport.SolutionTypeDCI,
			Name:                  "test-dci",
			Active:                true,
			ServiceLevelReference: "SL-001",
			ProvisioningStatus:    "LIVE",
			ALocationID:           143,
			BLocationID:           145,
			SpeedMbps:             10000,
			Members: []megaport.SolutionMember{
				{Role: megaport.SolutionMemberRoleAEndPort, ServiceUID: "a-end-uid", LocationID: 143},
				{Role: megaport.SolutionMemberRoleBEndPort, ServiceUID: "b-end-uid", LocationID: 145},
				{Role: megaport.SolutionMemberRoleVXC, ServiceUID: "vxc-uid"},
			},
		},
		GetSolutionBillableResult: &megaport.SolutionBillable{ContractTermMonths: 24, AutoRenewTerm: true},
	}
}

// dciNotFound returns the error the SDK gives for a 404 from the API.
func dciNotFound(message string) error {
	return dciAPIError(http.StatusNotFound, message)
}

// dciAPIError returns the error the SDK gives for an error status from the API.
func dciAPIError(status int, message string) error {
	return &megaport.ErrorResponse{
		Response: &http.Response{
			StatusCode: status,
			Request:    &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/v1/solutions/" + dciStubUID + "/billable"}},
		},
		Message: message,
	}
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
	raw := dciPlan(objType)

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

// TestDCICreateFillsState covers a create whose DCI is ready in time.
func TestDCICreateFillsState(t *testing.T) {
	t.Parallel()

	mock := dciMock()
	r, schemaResp, objType := newDCIMockResource(t, mock)
	raw := dciPlan(objType)

	resp := fwresource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)}}
	r.Create(t.Context(), fwresource.CreateRequest{
		Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: raw},
	}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create errors: %v", resp.Diagnostics.Errors())
	}

	if req := mock.CapturedBuyDCIRequest; req == nil || !req.WaitForProvision {
		t.Errorf("the order is %+v, want one that waits for the DCI", req)
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
		"provisioning_status":  {got.ProvisioningStatus.ValueString(), "LIVE"},
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

// TestDCICreateFailures covers the creates that fail.
// Once the DCI is ordered its UID must reach state, or Terraform would lose track of a billed DCI.
func TestDCICreateFailures(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		fail    func(*MockSolutionService)
		summary string
		ordered bool
	}{
		"buy fails": {
			fail:    func(m *MockSolutionService) { m.BuyDCIResult, m.BuyDCIErr = nil, errors.New("the mock fails the buy") },
			summary: "Error buying DCI",
		},
		"read fails": {
			fail:    func(m *MockSolutionService) { m.GetSolutionErr = errors.New("the mock fails the read") },
			summary: "ordered but could not be read",
			ordered: true,
		},
		"billable read fails": {
			fail:    func(m *MockSolutionService) { m.GetSolutionBillableErr = dciNotFound("No billable was found") },
			summary: "ordered but could not be read",
			ordered: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mock := dciMock()
			tc.fail(mock)
			r, schemaResp, objType := newDCIMockResource(t, mock)
			raw := dciPlan(objType)

			resp := fwresource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)}}
			r.Create(t.Context(), fwresource.CreateRequest{
				Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
				Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: raw},
			}, &resp)

			requireOneError(t, resp.Diagnostics, tc.summary)
			if !tc.ordered {
				if !resp.State.Raw.IsNull() {
					t.Fatal("Create saved state for a DCI that was not ordered")
				}
				return
			}
			if got := dciModel(t, resp.State).UID.ValueString(); got != dciStubUID {
				t.Fatalf("solution_uid in state = %q, want %q", got, dciStubUID)
			}
			if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, "terraform untaint") {
				t.Errorf("the error does not say how to recover: %s", detail)
			}
		})
	}
}

// TestDCIImportState covers an import, which starts state from the DCI's UID alone.
func TestDCIImportState(t *testing.T) {
	t.Parallel()

	r, schemaResp, objType := newDCIMockResource(t, dciMock())

	resp := fwresource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nullValueMap(objType))}}
	r.ImportState(t.Context(), fwresource.ImportStateRequest{ID: dciStubUID}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState errors: %v", resp.Diagnostics.Errors())
	}
	if got := dciModel(t, resp.State).UID.ValueString(); got != dciStubUID {
		t.Fatalf("solution_uid in state = %q, want %q", got, dciStubUID)
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

// TestDCIReadContractTerm covers the term a read leaves in state.
// The API rolls a DCI to a term of 1 when its term ends without auto-renew, which must not drift from the config.
func TestDCIReadContractTerm(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		ordered int64 // the term in state before the read, 0 for none
		apiTerm int
		want    int64
	}{
		"rolled to month-to-month keeps the ordered term": {ordered: 12, apiTerm: 1, want: 12},
		"re-termed takes the new term":                    {ordered: 12, apiTerm: 24, want: 24},
		"imported after the roll takes month-to-month":    {apiTerm: 1, want: 1},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stub := &dciStub{billable: map[string]any{"autoRenewTerm": false, "contractTermMonths": tc.apiTerm}}
			r, schemaResp, objType := newDCITestResource(t, stub)
			attrs := nullValueMap(objType)
			attrs["solution_uid"] = tftypes.NewValue(tftypes.String, dciStubUID)
			if tc.ordered != 0 {
				attrs["contract_term_months"] = tftypes.NewValue(tftypes.Number, tc.ordered)
			}
			state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, attrs)}

			resp := fwresource.ReadResponse{State: state}
			r.Read(t.Context(), fwresource.ReadRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("Read errors: %v", resp.Diagnostics.Errors())
			}

			if got := dciModel(t, resp.State).ContractTermMonths.ValueInt64(); got != tc.want {
				t.Errorf("contract_term_months = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestDCIReadRemovesCanceledDCI covers a DCI that was canceled outside Terraform.
// The API keeps a canceled DCI and reads it back with active set to false.
func TestDCIReadRemovesCanceledDCI(t *testing.T) {
	t.Parallel()

	r, schemaResp, objType := newDCITestResource(t, &dciStub{solution: map[string]any{"active": false}})
	state := dciState(schemaResp, objType)

	resp := fwresource.ReadResponse{State: state}
	r.Read(t.Context(), fwresource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read errors: %v", resp.Diagnostics.Errors())
	}
	if !resp.State.Raw.IsNull() {
		t.Fatal("Read kept the DCI in state")
	}
}

// TestDCIReadKeepsStateWhenReadFails covers the reads that must not drop a live DCI from state.
// The API never deletes a DCI. It answers 404 while DCI ordering is off for the account,
// or when the account does not own the DCI, with the same body as for a UID that does not exist.
func TestDCIReadKeepsStateWhenReadFails(t *testing.T) {
	t.Parallel()

	cases := map[string]*dciStub{
		"all ordering off": {readStatus: http.StatusNotFound, readMessage: "The requested endpoint is not found."},
		"not found":        {readStatus: http.StatusNotFound, readMessage: "Solution not found: " + dciStubUID},
		"server error":     {readStatus: http.StatusInternalServerError, readMessage: "the stub fails the read"},
		"not a DCI":        {solution: map[string]any{"type": "OTHER"}},
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

// TestDCIReadKeepsStateWhenBillableReadFails covers a DCI that reads back while its billing does not.
func TestDCIReadKeepsStateWhenBillableReadFails(t *testing.T) {
	t.Parallel()

	mock := dciMock()
	mock.GetSolutionBillableErr = dciNotFound("No billable was found")
	r, schemaResp, objType := newDCIMockResource(t, mock)
	state := dciState(schemaResp, objType)

	resp := fwresource.ReadResponse{State: state}
	r.Read(t.Context(), fwresource.ReadRequest{State: state}, &resp)
	requireOneError(t, resp.Diagnostics, "Error Reading DCI")
	if resp.State.Raw.IsNull() {
		t.Fatal("Read removed the DCI from state")
	}
}

// forcesReplace reports whether a change to attr makes Terraform replace the resource.
func forcesReplace(t *testing.T, s schema.Schema, attr schema.Attribute) bool {
	t.Helper()

	objType, ok := s.Type().TerraformType(t.Context()).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}
	raw := tftypes.NewValue(objType, nullValueMap(objType))
	state, plan := tfsdk.State{Schema: s, Raw: raw}, tfsdk.Plan{Schema: s, Raw: raw}

	switch a := attr.(type) {
	case schema.StringAttribute:
		for _, m := range a.PlanModifiers {
			req := planmodifier.StringRequest{State: state, Plan: plan, StateValue: types.StringValue("old"), PlanValue: types.StringValue("new"), ConfigValue: types.StringValue("new")}
			resp := planmodifier.StringResponse{PlanValue: req.PlanValue}
			if m.PlanModifyString(t.Context(), req, &resp); resp.RequiresReplace {
				return true
			}
		}
	case schema.Int64Attribute:
		for _, m := range a.PlanModifiers {
			req := planmodifier.Int64Request{State: state, Plan: plan, StateValue: types.Int64Value(1), PlanValue: types.Int64Value(2), ConfigValue: types.Int64Value(2)}
			resp := planmodifier.Int64Response{PlanValue: req.PlanValue}
			if m.PlanModifyInt64(t.Context(), req, &resp); resp.RequiresReplace {
				return true
			}
		}
	case schema.BoolAttribute:
		for _, m := range a.PlanModifiers {
			req := planmodifier.BoolRequest{State: state, Plan: plan, StateValue: types.BoolValue(false), PlanValue: types.BoolValue(true), ConfigValue: types.BoolValue(true)}
			resp := planmodifier.BoolResponse{PlanValue: req.PlanValue}
			if m.PlanModifyBool(t.Context(), req, &resp); resp.RequiresReplace {
				return true
			}
		}
	default:
		t.Fatalf("no replace check for an attribute of type %T", attr)
	}
	return false
}

// dciLiveValue returns the state of a live DCI, as the mock from dciMock reads it back, with changes applied on top.
func dciLiveValue(objType tftypes.Object, changes map[string]tftypes.Value) tftypes.Value {
	attrs := map[string]tftypes.Value{}
	if err := dciPlan(objType).As(&attrs); err != nil {
		panic(err)
	}
	attrs["solution_uid"] = tftypes.NewValue(tftypes.String, dciStubUID)
	attrs["provisioning_status"] = tftypes.NewValue(tftypes.String, "LIVE")
	attrs["a_end_port_uid"] = tftypes.NewValue(tftypes.String, "a-end-uid")
	attrs["b_end_port_uid"] = tftypes.NewValue(tftypes.String, "b-end-uid")
	attrs["vxc_uid"] = tftypes.NewValue(tftypes.String, "vxc-uid")
	maps.Copy(attrs, changes)
	return tftypes.NewValue(objType, attrs)
}

// runDCIUpdate runs Update on a live DCI with the changes applied to its plan.
func runDCIUpdate(t *testing.T, mock *MockSolutionService, changes map[string]tftypes.Value) fwresource.UpdateResponse {
	t.Helper()

	r, schemaResp, objType := newDCIMockResource(t, mock)
	state := tfsdk.State{Schema: schemaResp.Schema, Raw: dciLiveValue(objType, nil)}
	plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: dciLiveValue(objType, changes)}

	resp := fwresource.UpdateResponse{State: state}
	r.Update(t.Context(), fwresource.UpdateRequest{State: state, Plan: plan}, &resp)
	return resp
}

// TestDCIUpdateChangesNameAndCostCentre covers the two changes Update makes.
// The API sets both values on every update, so the one that does not change must be sent too.
func TestDCIUpdateChangesNameAndCostCentre(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		changes        map[string]tftypes.Value
		wantName       string
		wantCostCentre string
	}{
		"name": {
			changes:        map[string]tftypes.Value{"name": tftypes.NewValue(tftypes.String, "new-name")},
			wantName:       "new-name",
			wantCostCentre: "SL-001",
		},
		"cost centre": {
			changes:        map[string]tftypes.Value{"cost_centre": tftypes.NewValue(tftypes.String, "SL-002")},
			wantName:       "test-dci",
			wantCostCentre: "SL-002",
		},
		"both": {
			changes: map[string]tftypes.Value{
				"name":        tftypes.NewValue(tftypes.String, "new-name"),
				"cost_centre": tftypes.NewValue(tftypes.String, "SL-002"),
			},
			wantName:       "new-name",
			wantCostCentre: "SL-002",
		},
		"cleared cost centre": {
			changes:        map[string]tftypes.Value{"cost_centre": tftypes.NewValue(tftypes.String, "")},
			wantName:       "test-dci",
			wantCostCentre: "",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// The mock reads the DCI back as the API would after the update.
			mock := dciMock()
			mock.GetSolutionResult.Name = tc.wantName
			mock.GetSolutionResult.ServiceLevelReference = tc.wantCostCentre

			resp := runDCIUpdate(t, mock, tc.changes)
			if resp.Diagnostics.HasError() {
				t.Fatalf("Update errors: %v", resp.Diagnostics.Errors())
			}

			if len(mock.CapturedUpdateSolutionRequests) != 1 {
				t.Fatalf("Update sent %d updates, want 1", len(mock.CapturedUpdateSolutionRequests))
			}
			want := megaport.UpdateSolutionRequest{SolutionUID: dciStubUID, Name: tc.wantName, ServiceLevelReference: tc.wantCostCentre}
			if got := *mock.CapturedUpdateSolutionRequests[0]; got != want {
				t.Errorf("update = %+v, want %+v", got, want)
			}

			model := dciModel(t, resp.State)
			if model.Name.ValueString() != tc.wantName || model.CostCentre.ValueString() != tc.wantCostCentre {
				t.Errorf("state name = %q, cost centre = %q, want %q and %q", model.Name.ValueString(), model.CostCentre.ValueString(), tc.wantName, tc.wantCostCentre)
			}
			if model.CostCentre.IsNull() {
				t.Error("state cost centre is null")
			}
			if model.UID.ValueString() != dciStubUID || model.Speed.ValueInt64() != 10000 || model.ContractTermMonths.ValueInt64() != 24 {
				t.Errorf("state lost the DCI's other values: %+v", model)
			}
		})
	}
}

// runDCIModifyPlan plans a change from one DCI state to another.
func runDCIModifyPlan(t *testing.T, stateRaw, planRaw func(tftypes.Object) tftypes.Value) fwresource.ModifyPlanResponse {
	t.Helper()

	r, schemaResp, objType := newDCIMockResource(t, dciMock())
	null := tftypes.NewValue(objType, nil)
	stateValue, planValue := null, null
	if stateRaw != nil {
		stateValue = stateRaw(objType)
	}
	if planRaw != nil {
		planValue = planRaw(objType)
	}

	req := fwresource.ModifyPlanRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := fwresource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(t.Context(), req, &resp)
	return resp
}

func dciLive(changes map[string]tftypes.Value) func(tftypes.Object) tftypes.Value {
	return func(objType tftypes.Object) tftypes.Value { return dciLiveValue(objType, changes) }
}

// TestDCIPlanRejectsOtherChanges covers the changes the API cannot make.
// The plan must fail, so the customer learns of it before an apply, and a rename in the same change is not half done.
func TestDCIPlanRejectsOtherChanges(t *testing.T) {
	t.Parallel()

	cases := map[string]tftypes.Value{
		"a_end_location_id":    tftypes.NewValue(tftypes.Number, 150),
		"b_end_location_id":    tftypes.NewValue(tftypes.Number, 150),
		"speed":                tftypes.NewValue(tftypes.Number, 1000),
		"contract_term_months": tftypes.NewValue(tftypes.Number, 36),
		"auto_renew":           tftypes.NewValue(tftypes.Bool, false),
	}
	for attr, value := range cases {
		t.Run(attr, func(t *testing.T) {
			t.Parallel()

			resp := runDCIModifyPlan(t, dciLive(nil), dciLive(map[string]tftypes.Value{
				attr:   value,
				"name": tftypes.NewValue(tftypes.String, "new-name"),
			}))
			requireOneError(t, resp.Diagnostics, "Changing a DCI is not supported")
			if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, attr) {
				t.Errorf("error detail = %q, want it to name %s", detail, attr)
			}
		})
	}
}

// TestDCIPlanAllowsOtherPlans covers the plans that must pass.
func TestDCIPlanAllowsOtherPlans(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		state, plan func(tftypes.Object) tftypes.Value
	}{
		"create":  {plan: dciPlan},
		"destroy": {state: dciLive(nil)},
		"name and cost centre": {
			state: dciLive(nil),
			plan: dciLive(map[string]tftypes.Value{
				"name":        tftypes.NewValue(tftypes.String, "new-name"),
				"cost_centre": tftypes.NewValue(tftypes.String, "SL-002"),
			}),
		},
		// After a create that timed out, the state holds no speed until the VXC is created.
		"state without a speed": {
			state: dciLive(map[string]tftypes.Value{"speed": tftypes.NewValue(tftypes.Number, nil)}),
			plan:  dciLive(map[string]tftypes.Value{"name": tftypes.NewValue(tftypes.String, "new-name")}),
		},
		"speed not known until apply": {
			state: dciLive(nil),
			plan:  dciLive(map[string]tftypes.Value{"speed": tftypes.NewValue(tftypes.Number, tftypes.UnknownValue)}),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if resp := runDCIModifyPlan(t, tc.state, tc.plan); resp.Diagnostics.HasError() {
				t.Fatalf("ModifyPlan errors: %v", resp.Diagnostics.Errors())
			}
		})
	}
}

// TestDCIUpdateFailures covers an update the API refuses, such as one on a locked DCI,
// and an update that cannot be read back. Both must keep the DCI in state.
func TestDCIUpdateFailures(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		mock    func() *MockSolutionService
		summary string
	}{
		"update refused": {
			mock: func() *MockSolutionService {
				mock := dciMock()
				mock.UpdateSolutionErr = errors.New("the solution is locked")
				return mock
			},
			summary: "Error updating DCI",
		},
		"read fails": {
			mock: func() *MockSolutionService {
				mock := dciMock()
				mock.GetSolutionErr = errors.New("the mock fails the read")
				return mock
			},
			summary: "DCI updated but could not be read",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := runDCIUpdate(t, tc.mock(), map[string]tftypes.Value{"name": tftypes.NewValue(tftypes.String, "new-name")})
			requireOneError(t, resp.Diagnostics, tc.summary)
			if got := dciModel(t, resp.State).Name.ValueString(); got != "test-dci" {
				t.Errorf("state name = %q, want the old name", got)
			}
		})
	}
}

// TestDCIUpdateBillableReadFails covers an update whose billing cannot be read back. It must keep the DCI in state.
func TestDCIUpdateBillableReadFails(t *testing.T) {
	t.Parallel()

	mock := dciMock()
	mock.GetSolutionBillableErr = errors.New("the mock fails the billing read")

	resp := runDCIUpdate(t, mock, map[string]tftypes.Value{"name": tftypes.NewValue(tftypes.String, "new-name")})
	requireOneError(t, resp.Diagnostics, "DCI updated but could not be read")
	if got := dciModel(t, resp.State).Name.ValueString(); got != "test-dci" {
		t.Errorf("state name = %q, want the old name", got)
	}
}

// TestDCIUpdateWithoutChange covers an update that changes neither the name nor the cost centre,
// as when the state has no speed yet. It must not call the update API, which would set both again.
func TestDCIUpdateWithoutChange(t *testing.T) {
	t.Parallel()

	noSpeed := map[string]tftypes.Value{"speed": tftypes.NewValue(tftypes.Number, nil)}
	run := func(mock *MockSolutionService) fwresource.UpdateResponse {
		r, schemaResp, objType := newDCIMockResource(t, mock)
		state := tfsdk.State{Schema: schemaResp.Schema, Raw: dciLiveValue(objType, noSpeed)}
		plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: dciLiveValue(objType, nil)}

		resp := fwresource.UpdateResponse{State: state}
		r.Update(t.Context(), fwresource.UpdateRequest{State: state, Plan: plan}, &resp)
		return resp
	}

	mock := dciMock()
	resp := run(mock)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update errors: %v", resp.Diagnostics.Errors())
	}
	if len(mock.CapturedUpdateSolutionRequests) != 0 {
		t.Errorf("Update sent %d updates, want none", len(mock.CapturedUpdateSolutionRequests))
	}
	if got := dciModel(t, resp.State).Speed.ValueInt64(); got != 10000 {
		t.Errorf("state speed = %d, want 10000", got)
	}

	// A failed read must not say the DCI was updated.
	mock = dciMock()
	mock.GetSolutionErr = errors.New("the mock fails the read")
	requireOneError(t, run(mock).Diagnostics, "Error Reading DCI")
}

// TestDCIUpdateSkipsValuesStateLacks covers a state that holds no name or cost centre yet, as after a create
// that timed out, and a cost centre not known. None is a change: an update would overwrite the DCI's own values.
func TestDCIUpdateSkipsValuesStateLacks(t *testing.T) {
	t.Parallel()

	mock := dciMock()
	r, schemaResp, objType := newDCIMockResource(t, mock)
	state := tfsdk.State{Schema: schemaResp.Schema, Raw: dciLiveValue(objType, map[string]tftypes.Value{
		"name":        tftypes.NewValue(tftypes.String, nil),
		"cost_centre": tftypes.NewValue(tftypes.String, nil),
	})}
	plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: dciLiveValue(objType, map[string]tftypes.Value{
		"name":        tftypes.NewValue(tftypes.String, "new-name"),
		"cost_centre": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})}

	resp := fwresource.UpdateResponse{State: state}
	r.Update(t.Context(), fwresource.UpdateRequest{State: state, Plan: plan}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update errors: %v", resp.Diagnostics.Errors())
	}
	if len(mock.CapturedUpdateSolutionRequests) != 0 {
		t.Errorf("Update sent %d updates, want none", len(mock.CapturedUpdateSolutionRequests))
	}
	if got := dciModel(t, resp.State).CostCentre.ValueString(); got != "SL-001" {
		t.Errorf("state cost centre = %q, want the one read back", got)
	}
}

// TestDCIUpdateFillsUnknownUIDs covers a state that holds no Port or VXC UID yet, as after a create that timed out.
// The plan does not know them, so Update must take them from the read: Terraform rejects an unknown value after apply.
func TestDCIUpdateFillsUnknownUIDs(t *testing.T) {
	t.Parallel()

	r, schemaResp, objType := newDCIMockResource(t, dciMock())
	uids := func(v any) map[string]tftypes.Value {
		return map[string]tftypes.Value{
			"a_end_port_uid": tftypes.NewValue(tftypes.String, v),
			"b_end_port_uid": tftypes.NewValue(tftypes.String, v),
			"vxc_uid":        tftypes.NewValue(tftypes.String, v),
		}
	}
	state := tfsdk.State{Schema: schemaResp.Schema, Raw: dciLiveValue(objType, uids(nil))}
	planned := uids(tftypes.UnknownValue)
	planned["name"] = tftypes.NewValue(tftypes.String, "new-name")
	plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: dciLiveValue(objType, planned)}

	resp := fwresource.UpdateResponse{State: state}
	r.Update(t.Context(), fwresource.UpdateRequest{State: state, Plan: plan}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update errors: %v", resp.Diagnostics.Errors())
	}

	model := dciModel(t, resp.State)
	if model.AEndPortUID.ValueString() != "a-end-uid" || model.BEndPortUID.ValueString() != "b-end-uid" || model.VXCUID.ValueString() != "vxc-uid" {
		t.Errorf("state UIDs = %s, %s and %s, want the ones read back", model.AEndPortUID, model.BEndPortUID, model.VXCUID)
	}
}

// TestDCICostCentreRemovedFromConfigKeepsValue covers a cost centre removed from the configuration.
// The plan must hold the value in state: an unknown one would show a change on every plan.
func TestDCICostCentreRemovedFromConfigKeepsValue(t *testing.T) {
	t.Parallel()

	_, schemaResp, objType := newDCIMockResource(t, dciMock())
	costCentre, ok := schemaResp.Schema.Attributes["cost_centre"].(schema.StringAttribute)
	if !ok {
		t.Fatal("cost_centre is not a string attribute")
	}
	raw := dciLiveValue(objType, nil)
	req := planmodifier.StringRequest{
		Path:        path.Root("cost_centre"),
		State:       tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
		Plan:        tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
		StateValue:  types.StringValue("SL-001"),
		PlanValue:   types.StringUnknown(),
		ConfigValue: types.StringNull(),
	}
	resp := planmodifier.StringResponse{PlanValue: req.PlanValue}
	for _, m := range costCentre.PlanModifiers {
		m.PlanModifyString(t.Context(), req, &resp)
	}
	if !resp.PlanValue.Equal(req.StateValue) {
		t.Errorf("planned cost centre = %s, want the one in state, %s", resp.PlanValue, req.StateValue)
	}
}

// TestDCICostCentreIsNotBlank covers a cost centre of only spaces. The API stores it as none,
// so the state would never match what is read back. An empty one is allowed: it clears the value.
func TestDCICostCentreIsNotBlank(t *testing.T) {
	t.Parallel()

	_, schemaResp, _ := newDCIMockResource(t, dciMock())
	costCentre, ok := schemaResp.Schema.Attributes["cost_centre"].(schema.StringAttribute)
	if !ok {
		t.Fatal("cost_centre is not a string attribute")
	}
	for value, wantErr := range map[string]bool{"SL-001": false, " SL 001 ": false, "": false, " ": true, "\t\n": true} {
		req := validator.StringRequest{Path: path.Root("cost_centre"), ConfigValue: types.StringValue(value)}
		resp := validator.StringResponse{}
		for _, v := range costCentre.Validators {
			v.ValidateString(t.Context(), req, &resp)
		}
		if resp.Diagnostics.HasError() != wantErr {
			t.Errorf("cost centre %q: error = %v, want %v", value, resp.Diagnostics.HasError(), wantErr)
		}
	}
}

// TestDCINameIsRequired covers a blank name, which the API refuses.
func TestDCINameIsRequired(t *testing.T) {
	t.Parallel()

	_, schemaResp, _ := newDCIMockResource(t, dciMock())
	name, ok := schemaResp.Schema.Attributes["name"].(schema.StringAttribute)
	if !ok {
		t.Fatal("name is not a string attribute")
	}
	for value, wantErr := range map[string]bool{"a": false, " a ": false, "": true, " ": true, "\t\n": true} {
		req := validator.StringRequest{Path: path.Root("name"), ConfigValue: types.StringValue(value)}
		resp := validator.StringResponse{}
		for _, v := range name.Validators {
			v.ValidateString(t.Context(), req, &resp)
		}
		if resp.Diagnostics.HasError() != wantErr {
			t.Errorf("name %q: error = %v, want %v", value, resp.Diagnostics.HasError(), wantErr)
		}
	}
}

// TestDCIUpdateKeepsPlannedValues covers a read-back that differs from the plan: a stale name,
// and a VXC created since the refresh. Terraform rejects a result that differs from what it planned,
// so the state must hold the planned values, and take from the read only what the plan did not know.
func TestDCIUpdateKeepsPlannedValues(t *testing.T) {
	t.Parallel()

	// The mock still reads back the old name, and a VXC the state does not hold.
	mock := dciMock()
	r, schemaResp, objType := newDCIMockResource(t, mock)
	noVXC := map[string]tftypes.Value{"vxc_uid": tftypes.NewValue(tftypes.String, "")}
	state := tfsdk.State{Schema: schemaResp.Schema, Raw: dciLiveValue(objType, noVXC)}
	plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: dciLiveValue(objType, map[string]tftypes.Value{
		"vxc_uid":             tftypes.NewValue(tftypes.String, ""),
		"name":                tftypes.NewValue(tftypes.String, "new-name"),
		"provisioning_status": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})}

	resp := fwresource.UpdateResponse{State: state}
	r.Update(t.Context(), fwresource.UpdateRequest{State: state, Plan: plan}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update errors: %v", resp.Diagnostics.Errors())
	}

	model := dciModel(t, resp.State)
	if got := model.Name.ValueString(); got != "new-name" {
		t.Errorf("state name = %q, want the planned name", got)
	}
	if got := model.VXCUID.ValueString(); got != "" {
		t.Errorf("state VXC UID = %q, want the planned empty value", got)
	}
	if got := model.ProvisioningStatus.ValueString(); got != "LIVE" {
		t.Errorf("state provisioning status = %q, want the one read back", got)
	}
}

// TestDCICostCentreLength covers the longest cost centre the API accepts.
func TestDCICostCentreLength(t *testing.T) {
	t.Parallel()

	_, schemaResp, _ := newDCIMockResource(t, dciMock())
	costCentre, ok := schemaResp.Schema.Attributes["cost_centre"].(schema.StringAttribute)
	if !ok {
		t.Fatal("cost_centre is not a string attribute")
	}
	for length, wantErr := range map[int]bool{255: false, 256: true} {
		req := validator.StringRequest{Path: path.Root("cost_centre"), ConfigValue: types.StringValue(strings.Repeat("a", length))}
		resp := validator.StringResponse{}
		for _, v := range costCentre.Validators {
			v.ValidateString(t.Context(), req, &resp)
		}
		if resp.Diagnostics.HasError() != wantErr {
			t.Errorf("a cost centre of %d characters: error = %v, want %v", length, resp.Diagnostics.HasError(), wantErr)
		}
	}
}

// TestDCINoAttributeForcesReplace covers the plan of a change. A change of any attribute must reach Update,
// not force a replace: a replace orders a second DCI.
func TestDCINoAttributeForcesReplace(t *testing.T) {
	t.Parallel()

	_, schemaResp, _ := newDCIMockResource(t, dciMock())
	for name, attr := range schemaResp.Schema.Attributes {
		if forcesReplace(t, schemaResp.Schema, attr) {
			t.Errorf("a change of %s forces a replace", name)
		}
	}
	replacing := schema.StringAttribute{PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}
	if !forcesReplace(t, schemaResp.Schema, replacing) {
		t.Error("the replace check misses an attribute that forces a replace")
	}
}

// TestDCIDeleteCancelsDCI covers a destroy. It cancels the DCI, which cancels both Ports and the VXC,
// and sends no reason, as the other resources do when they cancel a service.
func TestDCIDeleteCancelsDCI(t *testing.T) {
	t.Parallel()

	stub := &dciStub{}
	r, schemaResp, objType := newDCITestResource(t, stub)
	state := dciState(schemaResp, objType)

	deleteResp := fwresource.DeleteResponse{State: state}
	r.Delete(t.Context(), fwresource.DeleteRequest{State: state}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("Delete errors: %v", deleteResp.Diagnostics.Errors())
	}

	stub.mu.Lock()
	defer stub.mu.Unlock()
	want := []string{"POST /v1/solutions/" + dciStubUID + "/cancel"}
	if strings.Join(stub.requests, ",") != strings.Join(want, ",") {
		t.Fatalf("requests = %v, want %v", stub.requests, want)
	}
	if len(stub.cancelBody) != 0 {
		t.Errorf("cancel body = %v, want no reason or comment", stub.cancelBody)
	}
}

// TestDCIDeleteFailures covers a cancel the API refuses. A DCI that is still active must stay in state,
// because it is still live and billed. One that is already canceled is gone, so the destroy succeeds.
func TestDCIDeleteFailures(t *testing.T) {
	t.Parallel()

	locked := errors.New("the solution is locked")
	cases := map[string]struct {
		active     bool
		cancelErr  error
		readErr    error
		wantErr    bool
		wantDetail string
	}{
		"refused while active": {active: true, cancelErr: locked, wantErr: true, wantDetail: "the solution is locked"},
		// The read error says why the check for an already canceled DCI could not run.
		"refused and read fails": {active: true, cancelErr: locked, readErr: errors.New("the mock fails the read"), wantErr: true, wantDetail: "the mock fails the read"},
		// A cancel that stops part-way leaves the DCI active, and a retry finishes it.
		"stopped part-way": {active: true, cancelErr: dciAPIError(http.StatusConflict, "Failed to decommission member services"), wantErr: true, wantDetail: "Run the same Terraform command again"},
		// A DCI the account cannot see may still be live, so it stays in state and the user is told how to drop it.
		"not found": {active: true, cancelErr: dciNotFound("No solution was found"), readErr: dciNotFound("No solution was found"), wantErr: true, wantDetail: "terraform state rm"},
		// The hint comes from the cancel's own 404, not from the read's.
		"not found but read": {active: true, cancelErr: dciNotFound("No solution was found"), wantErr: true, wantDetail: "terraform state rm"},
		"already canceled":   {active: false, cancelErr: dciAPIError(http.StatusBadRequest, "Solution is already cancelled.")},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mock := dciMock()
			mock.CancelSolutionErr = tc.cancelErr
			mock.GetSolutionResult.Active = tc.active
			mock.GetSolutionErr = tc.readErr
			r, schemaResp, objType := newDCIMockResource(t, mock)
			state := dciState(schemaResp, objType)

			deleteResp := fwresource.DeleteResponse{State: state}
			r.Delete(t.Context(), fwresource.DeleteRequest{State: state}, &deleteResp)
			if tc.wantErr {
				requireOneError(t, deleteResp.Diagnostics, "Error canceling DCI")
				if detail := deleteResp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, tc.wantDetail) {
					t.Errorf("error detail = %q, want it to contain %q", detail, tc.wantDetail)
				}
			} else if deleteResp.Diagnostics.HasError() {
				t.Fatalf("Delete errors: %v", deleteResp.Diagnostics.Errors())
			}
			if len(mock.CapturedCancelSolutionRequests) != 1 || mock.CapturedCancelSolutionRequests[0].SolutionUID != dciStubUID {
				t.Errorf("cancel requests = %v, want one for %s", mock.CapturedCancelSolutionRequests, dciStubUID)
			}
		})
	}
}
