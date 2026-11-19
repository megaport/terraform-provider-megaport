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
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	megaport "github.com/megaport/megaportgo"
)

const dciStubUID = "8da180e1-61ea-4961-a6e3-d80152ee9b14"

// dciStub answers the calls the DCI resource makes for one DCI.
type dciStub struct {
	// solution overrides fields of the DCI the stub returns.
	solution map[string]any
	// billable overrides fields of the billing the stub returns.
	billable map[string]any
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
	return &megaport.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusNotFound,
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

	// A change of any attribute must reach Update, not force a replace: a replace orders a second DCI.
	for name, attr := range schemaResp.Schema.Attributes {
		if forcesReplace(t, schemaResp.Schema, attr) {
			t.Errorf("a change of %s forces a replace", name)
		}
	}
	replacing := schema.StringAttribute{PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}
	if !forcesReplace(t, schemaResp.Schema, replacing) {
		t.Error("the replace check misses an attribute that forces a replace")
	}

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.requests) != 0 {
		t.Fatalf("Update and Delete called the API: %v", stub.requests)
	}
}
