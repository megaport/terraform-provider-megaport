// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	megaport "github.com/megaport/megaportgo"
)

// lagUpdatePortService answers the port calls Update makes and keeps every modify it receives.
// The real ModifyPort polls every 30 seconds, which rules out an HTTP stub here.
type lagUpdatePortService struct {
	megaport.PortService

	primary   megaport.Port
	listed    []*megaport.Port
	boughtUID string
	failUID   string
	listErr   error

	mu       sync.Mutex
	modified []megaport.ModifyPortRequest
}

func (s *lagUpdatePortService) GetPort(_ context.Context, _ string) (*megaport.Port, error) {
	port := s.primary
	return &port, nil
}

func (s *lagUpdatePortService) ListPorts(_ context.Context) ([]*megaport.Port, error) {
	return s.listed, s.listErr
}

func (s *lagUpdatePortService) ModifyPort(_ context.Context, req *megaport.ModifyPortRequest) (*megaport.ModifyPortResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.modified = append(s.modified, *req)
	if req.PortID == s.failUID {
		return nil, errors.New("the stub rejects the modify")
	}
	return &megaport.ModifyPortResponse{IsUpdated: true}, nil
}

func (s *lagUpdatePortService) ValidatePortOrder(_ context.Context, _ *megaport.BuyPortRequest) error {
	return nil
}

func (s *lagUpdatePortService) BuyPort(_ context.Context, _ *megaport.BuyPortRequest) (*megaport.BuyPortResponse, error) {
	return &megaport.BuyPortResponse{TechnicalServiceUIDs: []string{s.boughtUID}}, nil
}

func (s *lagUpdatePortService) ListPortResourceTags(_ context.Context, _ string) (map[string]string, error) {
	return nil, nil
}

func lagMember(uid, name string, term int) *megaport.Port {
	return &megaport.Port{UID: uid, Name: name, ContractTermMonths: term, CostCentre: "cc-1", AggregationID: 7}
}

// TestLagPortUpdate_ModifiesEveryMember covers which ports Update modifies. The API changes
// only the port named in the call, and a term sent twice extends that port's contract.
func TestLagPortUpdate_ModifiesEveryMember(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		planName       string
		planTerm       int64
		planCostCentre string
		planVisibility bool
		planLagCount   int64
		stateLagCount  int64
		listed         []*megaport.Port
		listErr        error
		failUID        string
		wantModified   []string
		wantTerm       map[string]int
		wantError      string
	}{
		{
			name:         "a name change reaches every member, the primary last",
			planName:     "lag-new",
			planTerm:     12,
			planLagCount: 2,
			listed:       []*megaport.Port{lagMember("lag-1", "lag-old", 12), lagMember("lag-2", "lag-old", 12)},
			wantModified: []string{"lag-2", "lag-1"},
		},
		{
			name:         "a term change reaches every member",
			planName:     "lag-old",
			planTerm:     24,
			planLagCount: 2,
			listed:       []*megaport.Port{lagMember("lag-1", "lag-old", 12), lagMember("lag-2", "lag-old", 12)},
			wantModified: []string{"lag-2", "lag-1"},
			wantTerm:     map[string]int{"lag-1": 24, "lag-2": 24},
		},
		{
			name:           "a cost centre change reaches every member",
			planName:       "lag-old",
			planTerm:       12,
			planCostCentre: "cc-2",
			planLagCount:   2,
			listed:         []*megaport.Port{lagMember("lag-1", "lag-old", 12), lagMember("lag-2", "lag-old", 12)},
			wantModified:   []string{"lag-2", "lag-1"},
		},
		{
			name:           "a marketplace visibility change reaches every member",
			planName:       "lag-old",
			planTerm:       12,
			planVisibility: true,
			planLagCount:   2,
			listed:         []*megaport.Port{lagMember("lag-1", "lag-old", 12), lagMember("lag-2", "lag-old", 12)},
			wantModified:   []string{"lag-2", "lag-1"},
		},
		{
			name:         "a name change leaves a member on another term at that term",
			planName:     "lag-new",
			planTerm:     12,
			planLagCount: 2,
			listed:       []*megaport.Port{lagMember("lag-1", "lag-old", 12), lagMember("lag-2", "lag-old", 24)},
			wantModified: []string{"lag-2", "lag-1"},
		},
		{
			name:         "cancelled and decommissioned members are skipped",
			planName:     "lag-new",
			planTerm:     12,
			planLagCount: 2,
			listed: []*megaport.Port{
				lagMember("lag-1", "lag-old", 12),
				{UID: "lag-2", Name: "lag-old", ContractTermMonths: 12, AggregationID: 7, ProvisioningStatus: megaport.STATUS_CANCELLED},
				{UID: "lag-3", Name: "lag-old", ContractTermMonths: 12, AggregationID: 7, ProvisioningStatus: megaport.STATUS_DECOMMISSIONED},
			},
			wantModified: []string{"lag-1"},
		},
		{
			// The retry after a failed apply. The member already holds the new term.
			name:         "a term change skips a member that already has the term",
			planName:     "lag-old",
			planTerm:     24,
			planLagCount: 2,
			listed:       []*megaport.Port{lagMember("lag-1", "lag-old", 12), lagMember("lag-2", "lag-old", 24)},
			wantModified: []string{"lag-1"},
			wantTerm:     map[string]int{"lag-1": 24},
		},
		{
			name:         "a lag_count raise alone modifies no member",
			planName:     "lag-old",
			planTerm:     12,
			planLagCount: 3,
			listErr:      errors.New("the stub rejects the list, which Update must not read"),
		},
		{
			name:         "a name change with a grow skips the port the grow ordered",
			planName:     "lag-new",
			planTerm:     12,
			planLagCount: 3,
			listed: []*megaport.Port{
				lagMember("lag-1", "lag-old", 12), lagMember("lag-2", "lag-old", 12), lagMember("lag-3", "lag-new", 12),
			},
			wantModified: []string{"lag-2", "lag-1"},
		},
		{
			name:          "a port that reports no aggregation id is modified alone",
			planName:      "lag-new",
			planTerm:      12,
			planLagCount:  1,
			stateLagCount: 1,
			listed: []*megaport.Port{
				{UID: "lag-1", Name: "lag-old", ContractTermMonths: 12, CostCentre: "cc-1"},
				{UID: "other", Name: "lag-old", ContractTermMonths: 12, CostCentre: "cc-1"},
			},
			wantModified: []string{"lag-1"},
		},
		{
			name:         "a failed member names the port and leaves the primary alone",
			planName:     "lag-new",
			planTerm:     12,
			planLagCount: 2,
			listed:       []*megaport.Port{lagMember("lag-1", "lag-old", 12), lagMember("lag-2", "lag-old", 12)},
			failUID:      "lag-2",
			wantModified: []string{"lag-2"},
			wantError:    "port lag-2 in LAG lag-1 failed or did not finish: the stub rejects the modify. Run the apply again",
		},
		{
			name:         "a short list read modifies no port",
			planName:     "lag-new",
			planTerm:     12,
			planLagCount: 2,
			listed:       []*megaport.Port{lagMember("lag-1", "lag-old", 12)},
			wantError:    "does not hold port lag-2 ",
		},
		{
			name:         "a list read with a grow that misses a state member modifies no port",
			planName:     "lag-new",
			planTerm:     12,
			planLagCount: 3,
			listed:       []*megaport.Port{lagMember("lag-1", "lag-old", 12), lagMember("lag-3", "lag-new", 12)},
			wantError:    "does not hold port lag-2 ",
		},
		{
			name:         "a list read that misses the primary modifies no port",
			planName:     "lag-new",
			planTerm:     12,
			planLagCount: 2,
			listed:       []*megaport.Port{lagMember("lag-2", "lag-old", 12)},
			wantError:    "port lag-1 is not in the product list",
		},
		{
			name:         "a failed list read modifies no port",
			planName:     "lag-new",
			planTerm:     12,
			planLagCount: 2,
			listErr:      errors.New("the stub rejects the list"),
			wantError:    "the stub rejects the list",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			primary := megaport.Port{UID: "lag-1", Name: "lag-old", ContractTermMonths: 12, CostCentre: "cc-1", AggregationID: 7, LagCount: 2}
			svc := &lagUpdatePortService{primary: primary, listed: tc.listed, listErr: tc.listErr, boughtUID: "lag-3", failUID: tc.failUID}
			r := &lagPortResource{client: &megaport.Client{PortService: svc}}

			schemaResp := fwresource.SchemaResponse{}
			r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

			stateLagCount := int64(2)
			if tc.stateLagCount != 0 {
				stateLagCount = tc.stateLagCount
			}
			stateModel := lagUpdateTestModel(t, "lag-old", 12, stateLagCount)
			planModel := lagUpdateTestModel(t, tc.planName, tc.planTerm, tc.planLagCount)
			if tc.planCostCentre != "" {
				planModel.CostCentre = types.StringValue(tc.planCostCentre)
			}
			planModel.MarketplaceVisibility = types.BoolValue(tc.planVisibility)
			state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
			plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
			if diags := state.Set(ctx, &stateModel); diags.HasError() {
				t.Fatalf("building state: %v", diags.Errors())
			}
			if diags := plan.Set(ctx, &planModel); diags.HasError() {
				t.Fatalf("building plan: %v", diags.Errors())
			}

			resp := fwresource.UpdateResponse{State: state}
			r.Update(ctx, fwresource.UpdateRequest{State: state, Plan: plan}, &resp)

			if tc.wantError == "" && resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics.Errors())
			}
			if tc.wantError != "" {
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected an error, got none")
				}
				if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, tc.wantError) {
					t.Errorf("error %q does not contain %q", detail, tc.wantError)
				}
			}

			if got, want := len(svc.modified), len(tc.wantModified); got != want {
				t.Fatalf("modified %d ports (%v), want %d", got, svc.modified, want)
			}
			for i, req := range svc.modified {
				if req.PortID != tc.wantModified[i] {
					t.Errorf("modify %d went to %q, want %q", i, req.PortID, tc.wantModified[i])
				}
				if !req.WaitForUpdate {
					t.Errorf("modify %s does not wait for the update", req.PortID)
				}
				if req.Name != tc.planName {
					t.Errorf("modify %s sent name %q, want %q", req.PortID, req.Name, tc.planName)
				}
				if want := planModel.CostCentre.ValueString(); req.CostCentre != want {
					t.Errorf("modify %s sent cost centre %q, want %q", req.PortID, req.CostCentre, want)
				}
				if req.MarketplaceVisibility == nil || *req.MarketplaceVisibility != tc.planVisibility {
					t.Errorf("modify %s sent visibility %v, want %t", req.PortID, req.MarketplaceVisibility, tc.planVisibility)
				}
				wantTerm, sendsTerm := tc.wantTerm[req.PortID]
				switch {
				case !sendsTerm && req.ContractTermMonths != nil:
					t.Errorf("modify %s sent term %d, want none", req.PortID, *req.ContractTermMonths)
				case sendsTerm && (req.ContractTermMonths == nil || *req.ContractTermMonths != wantTerm):
					t.Errorf("modify %s sent term %v, want %d", req.PortID, req.ContractTermMonths, wantTerm)
				}
			}
		})
	}
}

func lagUpdateTestModel(t *testing.T, name string, term, lagCount int64) lagPortResourceModel {
	t.Helper()

	uids, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"lag-1", "lag-2"})
	if diags.HasError() {
		t.Fatalf("building lag_port_uids: %v", diags.Errors())
	}
	// State holds no UID list until the API reports an aggregation.
	if lagCount == 1 {
		uids = types.ListNull(types.StringType)
	}
	return lagPortResourceModel{
		UID:                   types.StringValue("lag-1"),
		Name:                  types.StringValue(name),
		ContractTermMonths:    types.Int64Value(term),
		CostCentre:            types.StringValue("cc-1"),
		MarketplaceVisibility: types.BoolValue(false),
		PortSpeed:             types.Int64Value(10000),
		LocationID:            types.Int64Value(5),
		LagCount:              types.Int64Value(lagCount),
		LagPortUIDs:           uids,
		Resources:             types.ObjectNull(portResourcesAttrs),
		ResourceTags:          types.MapNull(types.StringType),
	}
}
