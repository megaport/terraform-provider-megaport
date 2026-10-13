// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	megaport "github.com/megaport/megaportgo"
)

// readApprovalService answers the approvals list the port reads make.
type readApprovalService struct {
	megaport.OrderApprovalService

	approvals []*megaport.OrderApproval
	err       error
	requests  []megaport.ListOrderApprovalsRequest
}

func (s *readApprovalService) ListOrderApprovals(_ context.Context, req *megaport.ListOrderApprovalsRequest) (*megaport.ListOrderApprovalsResponse, error) {
	s.requests = append(s.requests, *req)
	if s.err != nil {
		return nil, s.err
	}
	return &megaport.ListOrderApprovalsResponse{OrderApprovals: s.approvals}, nil
}

func pendingApproval(serviceID int, approvalType megaport.OrderApprovalType) []*megaport.OrderApproval {
	return []*megaport.OrderApproval{{ServiceID: serviceID, Type: approvalType, Status: megaport.OrderApprovalStatusPending}}
}

// portReadCases cover a refresh after Update kept the requested values for a term change held for approval.
var portReadCases = []struct {
	name      string
	imported  bool
	approved  bool
	approvals []*megaport.OrderApproval
	listErr   error
	wantList  bool
	wantKept  bool
}{
	{
		name:      "a pending term change keeps the requested values and warns",
		approvals: pendingApproval(101, megaport.OrderApprovalTypeTermChange),
		wantList:  true,
		wantKept:  true,
	},
	{
		name:     "an approved term change reads back the new values",
		approved: true,
	},
	{
		name:     "a rejected term change brings the old values back",
		wantList: true,
	},
	{
		name:     "a failed approvals list brings the old values back",
		listErr:  errors.New("the stub rejects the list"),
		wantList: true,
	},
	{
		name:      "a pending approval of another type brings the old values back",
		approvals: pendingApproval(101, megaport.OrderApprovalTypeSpeedChange),
		wantList:  true,
	},
	{
		name:      "a pending term change on another service brings the old values back",
		approvals: pendingApproval(999, megaport.OrderApprovalTypeTermChange),
		wantList:  true,
	},
	{
		name:      "an import takes the API values",
		imported:  true,
		approvals: pendingApproval(101, megaport.OrderApprovalTypeTermChange),
	},
}

func TestSinglePortRead_PendingTermChange(t *testing.T) {
	t.Parallel()

	for _, tc := range portReadCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			apiPort := megaport.Port{ID: 101, UID: "port-1", Name: "port-old", ContractTermMonths: 12, CostCentre: "cc-1"}
			if tc.approved {
				apiPort = megaport.Port{ID: 101, UID: "port-1", Name: "port-new", ContractTermMonths: 24, CostCentre: "cc-2", MarketplaceVisibility: true}
			}
			approvals := &readApprovalService{approvals: tc.approvals, err: tc.listErr}
			r := &portResource{client: &megaport.Client{PortService: &singlePortUpdateService{port: apiPort}, OrderApprovalService: approvals}}

			schemaResp := fwresource.SchemaResponse{}
			r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

			requested := singlePortUpdateTestModel("port-new", 24, "cc-2", true)
			prior := requested
			if tc.imported {
				prior.Name, prior.CostCentre = types.StringNull(), types.StringNull()
				prior.ContractTermMonths, prior.MarketplaceVisibility = types.Int64Null(), types.BoolNull()
			}
			state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
			if diags := state.Set(ctx, &prior); diags.HasError() {
				t.Fatalf("building state: %v", diags.Errors())
			}

			resp := fwresource.ReadResponse{State: state}
			r.Read(ctx, fwresource.ReadRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected errors: %v", resp.Diagnostics.Errors())
			}

			assertApprovalList(t, approvals.requests, tc.wantList)
			assertPendingWarning(t, resp.Diagnostics.Warnings(), tc.wantKept, "term increase on port port-1 needs order approval")

			var got singlePortResourceModel
			if diags := resp.State.Get(ctx, &got); diags.HasError() {
				t.Fatalf("reading state: %v", diags.Errors())
			}
			want := singlePortUpdateTestModel(apiPort.Name, int64(apiPort.ContractTermMonths), apiPort.CostCentre, apiPort.MarketplaceVisibility)
			if tc.wantKept {
				want = requested
			}
			if !got.Name.Equal(want.Name) || !got.ContractTermMonths.Equal(want.ContractTermMonths) ||
				!got.CostCentre.Equal(want.CostCentre) || !got.MarketplaceVisibility.Equal(want.MarketplaceVisibility) {
				t.Errorf("state holds name %s, term %s, cost centre %s, visibility %s; want %s, %s, %s, %s",
					got.Name, got.ContractTermMonths, got.CostCentre, got.MarketplaceVisibility,
					want.Name, want.ContractTermMonths, want.CostCentre, want.MarketplaceVisibility)
			}
		})
	}
}

func TestLagPortRead_PendingTermChange(t *testing.T) {
	t.Parallel()

	for _, tc := range portReadCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			primary := *lagMember("lag-1", "lag-old", 12)
			if tc.approved {
				primary = *lagMember("lag-1", "lag-new", 24)
				primary.CostCentre, primary.MarketplaceVisibility = "cc-2", true
			}
			primary.ID = 101
			primary.LagPortUIDs = []string{"lag-1", "lag-2"}
			approvals := &readApprovalService{approvals: tc.approvals, err: tc.listErr}
			r := &lagPortResource{client: &megaport.Client{PortService: &lagUpdatePortService{primary: primary}, OrderApprovalService: approvals}}

			schemaResp := fwresource.SchemaResponse{}
			r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

			requested := lagUpdateTestModel(t, "lag-new", 24, 2)
			requested.CostCentre, requested.MarketplaceVisibility = types.StringValue("cc-2"), types.BoolValue(true)
			prior := requested
			if tc.imported {
				prior.Name, prior.CostCentre = types.StringNull(), types.StringNull()
				prior.ContractTermMonths, prior.MarketplaceVisibility = types.Int64Null(), types.BoolNull()
			}
			state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
			if diags := state.Set(ctx, &prior); diags.HasError() {
				t.Fatalf("building state: %v", diags.Errors())
			}

			resp := fwresource.ReadResponse{State: state}
			r.Read(ctx, fwresource.ReadRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected errors: %v", resp.Diagnostics.Errors())
			}

			assertApprovalList(t, approvals.requests, tc.wantList)
			assertPendingWarning(t, resp.Diagnostics.Warnings(), tc.wantKept, "term increase on port lag-1 in LAG lag-1 needs order approval")

			var got lagPortResourceModel
			if diags := resp.State.Get(ctx, &got); diags.HasError() {
				t.Fatalf("reading state: %v", diags.Errors())
			}
			wantName, wantTerm, wantCostCentre, wantVisibility := primary.Name, int64(primary.ContractTermMonths), primary.CostCentre, primary.MarketplaceVisibility
			if tc.wantKept {
				wantName, wantTerm, wantCostCentre, wantVisibility = "lag-new", 24, "cc-2", true
			}
			if got.Name.ValueString() != wantName || got.ContractTermMonths.ValueInt64() != wantTerm ||
				got.CostCentre.ValueString() != wantCostCentre || got.MarketplaceVisibility.ValueBool() != wantVisibility {
				t.Errorf("state holds name %s, term %s, cost centre %s, visibility %s; want %s, %d, %s, %t",
					got.Name, got.ContractTermMonths, got.CostCentre, got.MarketplaceVisibility,
					wantName, wantTerm, wantCostCentre, wantVisibility)
			}
		})
	}
}

func assertApprovalList(t *testing.T, requests []megaport.ListOrderApprovalsRequest, want bool) {
	t.Helper()
	if !want {
		if len(requests) != 0 {
			t.Errorf("listed approvals %d times, want none", len(requests))
		}
		return
	}
	if len(requests) != 1 {
		t.Fatalf("listed approvals %d times, want 1", len(requests))
	}
	req := requests[0]
	if req.Status == nil || *req.Status != megaport.OrderApprovalStatusPending || !slices.Equal(req.ServiceIDs, []int{101}) {
		t.Errorf("listed approvals with status %v and service IDs %v, want PENDING and [101]", req.Status, req.ServiceIDs)
	}
}

func assertPendingWarning(t *testing.T, warnings diag.Diagnostics, want bool, detail string) {
	t.Helper()
	switch {
	case !want && len(warnings) > 0:
		t.Errorf("unexpected warnings: %v", warnings)
	case want && len(warnings) != 1:
		t.Errorf("got %d warnings (%v), want 1", len(warnings), warnings)
	case want && !strings.Contains(warnings[0].Detail(), detail):
		t.Errorf("warning %q does not contain %q", warnings[0].Detail(), detail)
	}
}
